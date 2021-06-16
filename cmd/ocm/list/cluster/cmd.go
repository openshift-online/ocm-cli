/*
Copyright (c) 2020 Red Hat, Inc.

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

package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/output"
	table "github.com/openshift-online/ocm-cli/pkg/table"
)

var args struct {
	parameter []string
	header    []string
	managed   bool
	step      bool
	noHeaders bool
	columns   string
	padding   int
}

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:     "clusters [flags] [PARTIAL_CLUSTER_ID_OR_NAME]",
	Aliases: []string{"cluster"},
	Short:   "List clusters",
	Long:    "List clusters, optionally filtering by substring of ID or Name",
	Args:    cobra.RangeArgs(0, 1),
	RunE:    run,
}

func init() {
	fs := Cmd.Flags()
	arguments.AddParameterFlag(fs, &args.parameter)
	arguments.AddHeaderFlag(fs, &args.header)
	fs.BoolVar(
		&args.managed,
		"managed",
		false,
		"Filter managed/unmanaged clusters",
	)
	fs.BoolVar(
		&args.step,
		"step",
		true,
		"Load pages one step at a time",
	)
	fs.BoolVar(
		&args.noHeaders,
		"no-headers",
		false,
		"Don't print header row",
	)
	fs.StringVar(
		&args.columns,
		"columns",
		"id, name, api.url, openshift_version, product.id, cloud_provider.id, region.id, state",
		"Specify which columns to display separated by commas, path is based on Cluster struct",
	)
	fs.IntVar(
		&args.padding,
		"padding",
		-1,
		"Change all column sizes.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	// Create a context:
	ctx := context.Background()

	// Create the output printer:
	printer, err := output.NewPrinter().
		Writer(os.Stdout).
		Paging(args.step).
		Build(ctx)
	if err != nil {
		return err
	}
	defer printer.Close()

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// This will contain the terms used to construct the search query:
	var searchTerms []string

	// If there is a parameter specified, assume its a filter:
	if len(argv) == 1 && argv[0] != "" {
		term := fmt.Sprintf("name like '%%%s%%' or id like '%%%s%%'", argv[0], argv[0])
		searchTerms = append(searchTerms, term)
	}

	// Add the search term for the `--managed` flag:
	if cmd.Flags().Changed("managed") {
		var value string
		if args.managed {
			value = "t"
		} else {
			value = "f"
		}
		term := fmt.Sprintf("managed = '%s'", value)
		searchTerms = append(searchTerms, term)
	}

	// If the `search` parameter has been specified with the `--parameter` flag then we have to
	// remove it and add the values to the list of search terms, otherwise we will be sending
	// multiple `search` query parameters and the server will ignore all but one of them. Note
	// that this modification of the `search` parameter isn't applicable in general, as other
	// endpoints may assign a different meaning to the `search` parameter, so be careful if you
	// try to apply this in other places.
	var cleanParameters []string
	for _, parameter := range args.parameter {
		name, value := arguments.ParseNameValuePair(parameter)
		if name == "search" {
			searchTerms = append(searchTerms, value)
		} else {
			cleanParameters = append(cleanParameters, parameter)
		}
	}
	args.parameter = cleanParameters

	// If there are more than one search term then we need to soround each of them with
	// parenthesis before joining them with the `and` connective.
	if len(searchTerms) > 1 {
		for i, term := range searchTerms {
			searchTerms[i] = fmt.Sprintf("(%s)", term)
		}
	}

	// Join all the search terms using the `and` connective:
	searchQuery := strings.Join(searchTerms, " and ")

	// Update our column name and padding variables:
	args.columns = strings.Replace(args.columns, " ", "", -1)
	colUpper := strings.ToUpper(args.columns)
	colUpper = strings.Replace(colUpper, ".", " ", -1)
	columnNames := strings.Split(colUpper, ",")
	paddingByColumn := []int{34, 30, 60, 20, 16}
	if args.padding != -1 {
		if args.padding < 2 {
			return fmt.Errorf("Padding flag needs to be an integer greater than 2")
		}
		paddingByColumn = []int{args.padding}
	}

	// Unless noHeaders set, print header row:
	if !args.noHeaders {
		table.PrintPadded(printer, columnNames, paddingByColumn)
	}

	// Create the request. Note that this request can be created outside of the loop and used
	// for all the iterations just changing the values of the `size` and `page` parameters.
	request := connection.ClustersMgmt().V1().Clusters().List().Search(searchQuery)
	arguments.ApplyParameterFlag(request, args.parameter)
	arguments.ApplyHeaderFlag(request, args.header)

	// Send the request till we receive a page with less items than requested:
	size := 100
	index := 1
	for {
		// Fetch the next page:
		request.Size(size)
		request.Page(index)
		response, err := request.Send()
		if err != nil {
			return fmt.Errorf("Can't retrieve clusters: %v", err)
		}

		// Display the items of the fetched page:
		response.Items().Each(func(cluster *v1.Cluster) bool {
			err = printItem(printer, cluster, paddingByColumn)
			return err == nil
		})
		if err != nil {
			break
		}

		// If the number of fetched items is less than requested, then this was the last
		// page, otherwise process the next one:
		if response.Size() < size {
			break
		}
		index++
	}

	return nil
}

func printItem(writer io.Writer, item *v1.Cluster, padding []int) error {
	// String to output marshal -
	// Map used to parse Cluster data -
	// Writer to body variable:
	var body string
	var jsonBody map[string]interface{}
	boddyBuffer := bytes.NewBufferString(body)

	// Write Cluster data to body variable:
	err := v1.MarshalCluster(item, boddyBuffer)
	if err != nil {
		return err
	}

	// Get JSON from Cluster bytes:
	err = json.Unmarshal(boddyBuffer.Bytes(), &jsonBody)
	if err != nil {
		return err
	}

	// Loop through wanted columns and populate a cluster instance:
	iter := strings.Split(args.columns, ",")
	thisCluster := []string{}
	for _, element := range iter {
		value, status := table.FindMapValue(jsonBody, element)
		if !status {
			value = "NONE"
		}
		thisCluster = append(thisCluster, value)
	}
	err = table.PrintPadded(writer, thisCluster, padding)
	if err != nil {
		return err
	}

	return nil
}
