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

	v1 "github.com/openshift-online/uhc-sdk-go/pkg/client/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/flags"
	table "github.com/openshift-online/uhc-cli/pkg/table"
)

var args struct {
	parameter []string
	header    []string
	managed   bool
	step      bool
	columns   string
	padding   int
}

var managed bool

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "list [flags] ",
	Short: "List clusters",
	Long:  "List clusters by ID and Name",
	Run:   run,
}

func init() {
	fs := Cmd.Flags()
	flags.AddParameterFlag(fs, &args.parameter)
	flags.AddHeaderFlag(fs, &args.header)
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
		"id, name, api.url, version.id, region.id",
		"Specify which columns to display separated by commas, path is based on Cluster struct i.e. "+
			"id,name,api.url,version.id,region.id"+
			"will output the default values.",
	)
	fs.IntVar(
		&args.padding,
		"padding",
		45,
		"Takes padding for custom columns, default to 35.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "Not logged in, run the 'login' command\n")
		os.Exit(1)
	}

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection, and remember to close it:
	connection, err := cfg.Connection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	collection := connection.ClustersMgmt().V1().Clusters()

	if cmd.Flags().Changed("managed") {
		managed = args.managed
	} else {
		managed = false
	}

	// Update our column name and padding variables:
	args.columns = strings.Replace(args.columns, " ", "", -1)
	colUpper := strings.ToUpper(args.columns)
	colUpper = strings.Replace(colUpper, ".", " ", -1)
	columnNames := strings.Split(colUpper, ",")
	paddingByColumn := []int{35, 25, 70, 25, 15}
	if args.columns != "id,name,api.url,version.id,region.id" {
		paddingByColumn = []int{args.padding}
	}

	// Print Header Row:
	table.PrintPadded(os.Stdout, columnNames, paddingByColumn)
	fmt.Println()

	size := 100
	index := 1
	for {
		// Fetch the next page:
		request := collection.List().Size(size).Page(index)
		flags.ApplyParameterFlag(request, args.parameter)
		flags.ApplyHeaderFlag(request, args.header)
		if managed {
			request.Search("managed = 't'")
		}
		response, err := request.Send()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't retrieve clusters: %s\n", err)
			os.Exit(1)
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
				fmt.Fprintf(os.Stderr, "Failed to retrieve input: %s\n", err)
				os.Exit(1)
			}
			clearPage()
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

	fmt.Println()
}

// clearPage clears the page.
func clearPage() {
	// #nosec 204
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to clear page: %s\n", err)
		os.Exit(1)
	}
}
