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

//go:generate go-bindata -pkg output tables

package output

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/data"
	"gopkg.in/yaml.v3"
)

// TableBuilder contains the data and logic needed to create a new output table.
type TableBuilder struct {
	printer *Printer
	name    string
	specs   []string
	digger  *data.Digger
	values  map[string]reflect.Value
}

// Table contains the data and logic needed to write tabular output.
type Table struct {
	printer *Printer
	name    string
	columns []*Column
	digger  *data.Digger
}

// tableYAML is used to load a table description from a YAML document.
type tableYAML struct {
	Columns []*columnYAML `yaml:"columns"`
}

// Column contains the data and logic needed to write columns.
type Column struct {
	table  *Table
	name   string
	header string
	width  int
	value  reflect.Value
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
		printer: p,
		values:  map[string]reflect.Value{},
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
		printer: b.printer,
		name:    b.name,
		columns: []*Column{},
	}

	// Check if there is an asset corresponding to the table. If there is no asset then return
	// the empty table description:
	assetFile := fmt.Sprintf("tables/%s.yaml", b.name)
	assetData, err := Asset(assetFile)
	if err != nil {
		result = table
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
		column.width = *columnData.Width
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

// defaultColumnWidth returns the default width for the given column name.
func (b *TableBuilder) defaultColumnWidth(columnName string) int {
	return len(columnName)
}

// defaultColumnValue returns the default value for the given column.
func (b *TableBuilder) defaultColumnValue(columnName string) reflect.Value {
	return b.values[columnName]
}

// WriteColumns writes a row of a table using the given values.
func (t *Table) WriteColumns(columnValues []interface{}) error {
	// Check that the number of values matches the number of columns:
	valueCount := len(columnValues)
	columnCount := len(t.columns)
	if valueCount != columnCount {
		return fmt.Errorf(
			"table '%s' has %d columns, but %d values have been given",
			t.name, columnCount, valueCount,
		)
	}

	// Prepare a buffer to write the columns (sum of the widths of the columns plus two
	// characters to separate columns, and the new line):
	tableWidth := 2 * columnCount
	for _, column := range t.columns {
		tableWidth += column.width
	}
	var rowBuffer bytes.Buffer
	rowBuffer.Grow(tableWidth)

	// Write the values while trimming or padding to adjust to the desired sizes:
	for i, columnValue := range columnValues {
		if i > 0 {
			rowBuffer.WriteString("  ")
		}
		var columnText string
		if columnValue != nil {
			columnText = fmt.Sprintf("%v", columnValue)
		} else {
			columnText = "NONE"
		}
		actualWidth := len(columnText)
		desiredWidth := t.columns[i].width
		switch {
		case actualWidth > desiredWidth:
			rowBuffer.WriteString(columnText[0:desiredWidth])
		case actualWidth < desiredWidth:
			rowBuffer.WriteString(columnText)
			for j := 0; j < desiredWidth-actualWidth; j++ {
				rowBuffer.WriteString(" ")
			}
		default:
			rowBuffer.WriteString(columnText)
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
		headers[i] = column.header
	}
	return t.WriteColumns(headers)
}

// WriteRow writes a row of a table extracting the values of the columns from the given object.
func (t *Table) WriteRow(object interface{}) error {
	values := make([]interface{}, len(t.columns))
	for i, column := range t.columns {
		values[i] = column.Value(object)
	}
	return t.WriteColumns(values)
}

// Close releases all the resources used by the table.
func (t *Table) Close() error {
	return nil
}

// Value extract the value of this column from the given object.
func (c *Column) Value(object interface{}) interface{} {
	// If the column doesn't have a explicit value then get it ussing the digger:
	if !c.value.IsValid() {
		return c.table.digger.Dig(object, c.name)
	}

	// If there is a explicit value then use reflection to get the value, calling it if it is a
	// function or getting it directly othersise:
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
