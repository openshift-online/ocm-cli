/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains the code that writes generates tabular output.

package output

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/openshift-online/ocm-sdk-go/data"
	yaml "gopkg.in/yaml.v3"
)

//go:embed tables
var assetFS embed.FS

// TableBuilder contains the data and logic needed to create a new output table.
type TableBuilder struct {
	printer       *Printer
	name          string
	specs         []string
	digger        *data.Digger
	values        map[string]reflect.Value
	learning      bool
	learningLimit int
}

// Table contains the data and logic needed to write tabular output.
type Table struct {
	printer *Printer
	name    string
	columns []*Column
	digger  *data.Digger

	// We will accumulate the first rows of data, and then will use it to learn how to display
	// it without wasting space.
	learning      bool
	learningLimit int
	learningRows  [][]string
}

// tableYAML is used to load a table description from a YAML document.
type tableYAML struct {
	Columns []*columnYAML `yaml:"columns"`
}

// Column contains the data and logic needed to write columns.
type Column struct {
	// Reference to the table that this column belongs to.
	table *Table

	// Name of the column, for example `cloud_provider.id`.
	name string

	// Header of the column, for example `CLOUD PROVIDER`.
	header string

	// Flag indicating if the width of this column should be automatically learned from the data
	// of the table.
	learn bool

	// Width of the column. This will be initially loaded from the table metadata, and adjusted
	// according to the data of the table if the learn flag is true.
	width int

	// This can be the actual value of the column or, more generally, a function that will be
	// called to obtain the actual value.
	value reflect.Value
}

// columnYAML is used to load a column description from a YAML document.
type columnYAML struct {
	Name   *string `yaml:"name"`
	Header *string `yaml:"header"`
	Width  *int    `yaml:"width"`
}

// NewTable creates a new builder that can then be used to configure and create a table.
func (p *Printer) NewTable() *TableBuilder {
	return &TableBuilder{
		printer:       p,
		values:        map[string]reflect.Value{},
		learning:      true,
		learningLimit: 100,
	}
}

// Name sets the name of the table. This is mandatory.
func (b *TableBuilder) Name(value string) *TableBuilder {
	b.name = value
	return b
}

// Column adds a column to the table. The spec can be a single column identifier or a set of comma
// separated column identifiers.
func (b *TableBuilder) Column(spec string) *TableBuilder {
	b.specs = append(b.specs, spec)
	return b
}

// Columns adds a collection of columns to the table. Each spec can be a single column identifier or
// a set of comman separated column identifiers.
func (b *TableBuilder) Columns(specs ...string) *TableBuilder {
	b.specs = append(b.specs, specs...)
	return b
}

// Value sets the value for the given column name. The value can be an object or a function. If it
// is a function then it will be called passing as parameter the row object and it is expected to
// return as first output parameter the value for the column.
func (b *TableBuilder) Value(name string, value interface{}) *TableBuilder {
	b.values[name] = reflect.ValueOf(value)
	return b
}

// Digger sets the digger that will be used to extract fields from row objects. If not specified the
// digger of the printer will be used.
func (b *TableBuilder) Digger(value *data.Digger) *TableBuilder {
	b.digger = value
	return b
}

// Learning enables or disables the mechanism that the table uses to learn how to display columns.
// When learning is enabled the table will use the first rows of the table to adjust the widths of
// the columns in order to not waste space. The default value is that learning is enabled.
//
// Note that the results of learning are usually good, but there may be cases where they aren't. For
// example, if the first one hundred rows of one row have a column empty then the width of that
// column will be reduced to the width of the header even if there are other later rows that have
// very long values.
func (b *TableBuilder) Learning(value bool) *TableBuilder {
	b.learning = value
	return b
}

// LearningLimit sets the number of rows (including the headers) that the table will use for learning
// is enabled. The default value is to use the first one hundred rows.
func (b *TableBuilder) LearningLimit(value int) *TableBuilder {
	b.learningLimit = value
	return b
}

// Build uses the configuration stored in the builder to create a table.
func (b *TableBuilder) Build(ctx context.Context) (result *Table, err error) {
	// Check parameters:
	if b.printer == nil {
		err = fmt.Errorf("printer is mandatory")
		return
	}
	if b.name == "" {
		err = fmt.Errorf("name is mandatory")
		return
	}
	if len(b.specs) == 0 {
		err = fmt.Errorf("at least one column is required")
		return
	}

	// Split the column specifications into individual column names:
	columnNames := make([]string, 0, len(b.specs))
	for _, specs := range b.specs {
		specChunks := strings.Split(specs, ",")
		for _, specChunk := range specChunks {
			columnName := strings.TrimSpace(specChunk)
			if columnName != "" {
				columnNames = append(columnNames, columnName)
			}
		}
	}

	// Load the table description:
	table, err := b.loadTable(columnNames)
	if err != nil {
		return
	}

	// Create the digger if needed:
	table.digger = b.digger
	if b.digger == nil {
		table.digger = b.printer.digger
	}

	// Return the result:
	result = table
	return
}

func (b *TableBuilder) loadTable(columnNames []string) (result *Table, err error) {
	// Create an initially empty table:
	table := &Table{
		printer:       b.printer,
		name:          b.name,
		columns:       []*Column{},
		learning:      b.learning,
		learningLimit: b.learningLimit,
	}

	// Check if there is an asset corresponding to the table. If there is no asset then return
	// the empty table description:
	assetPath := fmt.Sprintf("tables/%s.yaml", b.name)
	assetFile, err := assetFS.Open(assetPath)
	if err != nil {
		result = table
		return
	}
	defer assetFile.Close()
	assetData, err := io.ReadAll(assetFile)
	if err != nil {
		return
	}

	// Parse the YAML document from the asset:
	var tableData tableYAML
	err = yaml.Unmarshal(assetData, &tableData)
	if err != nil {
		return
	}

	// Load the descriptions of the columns from the asset:
	columnsFromAsset := make([]*Column, len(tableData.Columns))
	for i, columnData := range tableData.Columns {
		columnsFromAsset[i], err = b.loadColumn(i, columnData)
		if err != nil {
			return
		}
	}

	// Create the list of columns using the descriptions loaded from the asset, or else default
	// descriptions for the columns that aren't described in the asset:
	table.columns = make([]*Column, len(columnNames))
	for i, columnName := range columnNames {
		var column *Column
		for _, columnFromAsset := range columnsFromAsset {
			if columnFromAsset.name == columnName {
				column = columnFromAsset
				break
			}
		}
		if column == nil {
			column = b.defaultColumn(columnName)
		}
		column.table = table
		table.columns[i] = column
	}

	// Return the table:
	result = table
	return
}

// loadColumnYAML copies the column data from the YAML document to the object.
func (b *TableBuilder) loadColumn(i int, columnData *columnYAML) (result *Column, err error) {
	// Check that the name of the column has been specified:
	if columnData.Name == nil || *columnData.Name == "" {
		err = fmt.Errorf(
			"column %d of table '%s' doesn't have a name",
			i, b.name,
		)
		return
	}
	columnName := *columnData.Name

	// Create an initially empty column:
	column := b.defaultColumn(columnName)

	// Copy the data from the YAML document:
	if columnData.Header != nil {
		column.header = *columnData.Header
	}
	if columnData.Width != nil {
		column.learn = false
		column.width = *columnData.Width
	} else {
		column.learn = true
	}

	// Return the column:
	result = column
	return
}

// defaultColumn creates a default column description for the given column name.
func (b *TableBuilder) defaultColumn(columnName string) *Column {
	return &Column{
		name:   columnName,
		header: b.defaultColumnHeader(columnName),
		learn:  b.defaultColumnLearn(columnName),
		width:  b.defaultColumnWidth(columnName),
		value:  b.defaultColumnValue(columnName),
	}
}

// defaultColumnHeader returns the default value for the header of the given column name.
func (b *TableBuilder) defaultColumnHeader(columnName string) string {
	columnHeader := columnName
	columnHeader = strings.ReplaceAll(columnHeader, ".", " ")
	columnHeader = strings.ReplaceAll(columnHeader, "_", " ")
	columnHeader = strings.ToUpper(columnHeader)
	return columnHeader
}

// defaultColumnLearn returns the default value for the flag that indicates if the width of the
// column should be learned from the data of the table.
func (b *TableBuilder) defaultColumnLearn(columnName string) bool {
	return true
}

// defaultColumnWidth returns the default width for the given column name.
func (b *TableBuilder) defaultColumnWidth(columnName string) int {
	return len(columnName)
}

// defaultColumnValue returns the default value for the given column.
func (b *TableBuilder) defaultColumnValue(columnName string) reflect.Value {
	return b.values[columnName]
}

// WriteRow writes a row of a table using the given values.
func (t *Table) WriteRow(rowValues []interface{}) error {
	// Check that the number of values matches the number of columns:
	valueCount := len(rowValues)
	columnCount := len(t.columns)
	if valueCount != columnCount {
		return fmt.Errorf(
			"table '%s' has %d columns, but %d values have been given",
			t.name, columnCount, valueCount,
		)
	}

	// Convert the row values to strings:
	rowData := make([]string, columnCount)
	for i, columnValue := range rowValues {
		var columnData string
		if columnValue != nil {
			columnData = fmt.Sprintf("%v", columnValue)
		} else {
			columnData = "NONE"
		}
		rowData[i] = columnData
	}

	// Try to accumulate the row for learning:
	accumulated, err := t.accumulateRow(rowData)
	if err != nil {
		return err
	}
	if accumulated {
		return nil
	}

	// Write the column data:
	err = t.writeRow(rowData)
	if err != nil {
		return err
	}

	return nil
}

// accumulateRow checks if it is necessary to accumulate the given row in order to learn how to
// display columns. If the row is accumulated it returns true. The caller should not display that
// row yet. If the row isn't accumulated it returns false and the client should display that row.
//
// When enough rows have been accumulated it will perform the learning process and display the
// accumulated rows.
func (t *Table) accumulateRow(rowData []string) (accumulated bool, err error) {
	// Do nothing if we aren't learning:
	if !t.learning {
		accumulated = false
		return
	}

	// If we are learning and we didn't accumulate enough data yet, then we need to save the
	// passed row and continue. Actual learning will happen latter, when we have accumulated
	// enough rows.
	if len(t.learningRows) < t.learningLimit {
		t.learningRows = append(t.learningRows, rowData)
		accumulated = true
		return
	}

	// If wa are here it means that we have already accumulated enough rows, so we can complete
	// the learning process:
	err = t.completeLearning()
	if err != nil {
		return
	}
	accumulated = false
	return
}

// complelteLearning completes the learning process and displays the rows that were accumulated for
// that. This is intended to be called from the method that accumulates rows and from the method
// that closes the table, to make sure that we complete the learning process even when there are
// less rows than usually required to complete it.
func (t *Table) completeLearning() error {
	var err error
	t.learnColumnWidths()
	for _, rowData := range t.learningRows {
		err = t.writeRow(rowData)
		if err != nil {
			return err
		}
	}
	t.learning = false
	t.learningRows = nil
	return nil
}

// learnColumnWidths uses the accumulated rows to adjust the column widths so that space isn't
// wasted.
func (t *Table) learnColumnWidths() {
	for i, column := range t.columns {
		if !column.Learn() {
			continue
		}
		learnedWidth := len(column.Header())
		for j := range t.learningRows {
			actualWidth := len(t.learningRows[j][i])
			if actualWidth > learnedWidth {
				learnedWidth = actualWidth
			}
		}
		column.Adjust(learnedWidth)
	}
}

func (t *Table) writeRow(rowData []string) error {
	// Prepare a buffer to write the columns (sum of the widths of the columns plus two
	// characters to separate columns, and the new line):
	rowWidth := 2 * len(rowData)
	for _, column := range t.columns {
		rowWidth += column.Width()
	}
	var rowBuffer bytes.Buffer
	rowBuffer.Grow(rowWidth)

	// Write the values while trimming or padding to adjust to the desired sizes:
	for i, columnValue := range rowData {
		if i > 0 {
			rowBuffer.WriteString("  ")
		}
		actualWidth := len(columnValue)
		desiredWidth := t.columns[i].Width()
		switch {
		case actualWidth > desiredWidth:
			rowBuffer.WriteString(columnValue[0:desiredWidth])
		case actualWidth < desiredWidth:
			rowBuffer.WriteString(columnValue)
			for j := 0; j < desiredWidth-actualWidth; j++ {
				rowBuffer.WriteString(" ")
			}
		default:
			rowBuffer.WriteString(columnValue)
		}
	}
	rowBuffer.WriteString("\n")

	// Write the content of the buffer:
	_, err := rowBuffer.WriteTo(t.printer)
	return err
}

// WriteHeaders writes the headers of the columns of the table.
func (t *Table) WriteHeaders() error {
	headers := make([]interface{}, len(t.columns))
	for i, column := range t.columns {
		headers[i] = column.Header()
	}
	return t.WriteRow(headers)
}

// WriteObject writes a row of a table extracting the values of the columns from the given object.
func (t *Table) WriteObject(object interface{}) error {
	values := make([]interface{}, len(t.columns))
	for i, column := range t.columns {
		values[i] = column.Value(object)
	}
	return t.WriteRow(values)
}

// Flush makes sure that all the potentially pending data in interna buffers is written out.
func (t *Table) Flush() error {
	// Make sure to complete the learning process:
	if t.learning {
		err := t.completeLearning()
		if err != nil {
			return err
		}
	}
	return nil
}

// Close releases all the resources used by the table.
func (t *Table) Close() error {
	return t.Flush()
}

// Learn returns a flag indicating if the width of this column should be learned from the data of
// the table.
func (c *Column) Learn() bool {
	return c.learn
}

// Width returns the width for this column.
func (c *Column) Width() int {
	return c.width
}

// Header returns the header for this column.
func (c *Column) Header() string {
	return c.header
}

// Value extract the value of this column from the given object.
func (c *Column) Value(object interface{}) interface{} {
	// If the column doesn't have a explicit value then get it using the digger:
	if !c.value.IsValid() {
		return c.table.digger.Dig(object, c.name)
	}

	// If there is a explicit value then use reflection to get the value, calling it if it is a
	// function or getting it directly otherwise:
	var result reflect.Value
	switch c.value.Kind() {
	case reflect.Func:
		args := []reflect.Value{
			reflect.ValueOf(object),
		}
		results := c.value.Call(args)
		result = results[0]
	default:
		result = c.value
	}
	if result.IsValid() {
		return result.Interface()
	}
	return nil
}

// Adjust adjust the width of the column to the given value.
func (c *Column) Adjust(value int) {
	c.width = value
}
