/*
Copyright (c) 2019 Red Hat, Inc.

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

package list

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	table "github.com/openshift-online/ocm-cli/pkg/table"
)

var args struct {
	parameter []string
	header    []string
	managed   bool
	step      bool
	columns   string
	padding   int
}

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:        "list [flags] [partial cluster ID or name]",
	Short:      "List clusters",
	Long:       "List clusters by ID and Name",
	Deprecated: "please use `ocm list clusters` command",
	Args:       cobra.RangeArgs(0, 1),
	RunE:       run,
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
		false,
		"Load pages one step at a time",
	)
	fs.StringVar(
		&args.columns,
		"columns",
		"id, name, api.url, openshift_version, cloud_provider.id, region.id, state",
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
	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	collection := connection.ClustersMgmt().V1().Clusters()

	// If there is a parameter specified, assume its a filter:
	var argFilter string
	if len(argv) == 1 && argv[0] != "" {
		argFilter = fmt.Sprintf("(name like '%%%s%%' or id like '%%%s%%')", argv[0], argv[0])
	}

	// Update our column name and padding variables:
	args.columns = strings.Replace(args.columns, " ", "", -1)
	colUpper := strings.ToUpper(args.columns)
	colUpper = strings.Replace(colUpper, ".", " ", -1)
	columnNames := strings.Split(colUpper, ",")
	paddingByColumn := []int{34, 40, 60, 20}
	if args.padding != -1 {
		if args.padding < 2 {
			return fmt.Errorf("Padding flag needs to be an integer greater than 2")
		}
		paddingByColumn = []int{args.padding}
	}

	// Print Header Row:
	table.PrintPadded(os.Stdout, columnNames, paddingByColumn)

	size := 100
	index := 1
	for {
		// Fetch the next page:
		request := collection.List().Size(size).Page(index)
		arguments.ApplyParameterFlag(request, args.parameter)
		arguments.ApplyHeaderFlag(request, args.header)
		var search strings.Builder
		if cmd.Flags().Changed("managed") {
			if search.Len() > 0 {
				_, err = search.WriteString(" and ")
				if err != nil {
					return fmt.Errorf("Can't write to string: %v", err)
				}
			}
			if args.managed {
				_, err = search.WriteString("managed = 't'")
			} else {
				_, err = search.WriteString("managed = 'f'")
			}
			if err != nil {
				return fmt.Errorf("Can't write to string: %v", err)
			}
		}
		if argFilter != "" {
			if search.Len() > 0 {
				_, err = search.WriteString(" and ")
				if err != nil {
					return fmt.Errorf("Can't write to string: %v", err)
				}
			}
			_, err = search.WriteString(argFilter)
			if err != nil {
				return fmt.Errorf("Can't write to string: %v", err)
			}
		}
		request.Search(strings.TrimSpace(search.String()))
		response, err := request.Send()
		if err != nil {
			return fmt.Errorf("Can't retrieve clusters: %v", err)
		}

		// Display the fetched page:
		response.Items().Each(func(cluster *v1.Cluster) bool {

			// String to output marshal -
			// Map used to parse Cluster data -
			// Writer to body variable:
			var body string
			var jsonBody map[string]interface{}
			boddyBuffer := bytes.NewBufferString(body)

			// Write Cluster data to body variable:
			err := v1.MarshalCluster(cluster, boddyBuffer)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to marshal cluster into byte buffer: %s\n", err)
				os.Exit(1)
			}

			// Get JSON from Cluster bytes:
			err = json.Unmarshal(boddyBuffer.Bytes(), &jsonBody)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to turn cluster bytes into JSON map: %s\n", err)
				os.Exit(1)
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
			table.PrintPadded(os.Stdout, thisCluster, paddingByColumn)
			return true

		})

		// if step was flagged, load only one page at a time:
		if args.step {
			if response.Size() < size {
				break
			}
			fmt.Println()
			fmt.Println("Press the 'Enter' to load more:")
			_, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
			// var input string
			if err != nil {
				return fmt.Errorf("Failed to retrieve input: %v", err)
			}
			err = clearPage()
			if err != nil {
				return fmt.Errorf("Failed to clear page: %v", err)
			}
			table.PrintPadded(os.Stdout, columnNames, paddingByColumn)
			fmt.Println()
		}

		// If the number of fetched results is less than requested, then
		// this was the last page, otherwise process the next one:
		if response.Size() < size {
			break
		}
		index++
	}

	return nil
}

// clearPage clears the page.
func clearPage() error {
	// #nosec 204
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
