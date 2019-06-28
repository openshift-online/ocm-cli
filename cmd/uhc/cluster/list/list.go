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
	"strconv"
	"strings"

	v1 "github.com/openshift-online/uhc-sdk-go/pkg/client/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
)

var args struct {
	parameter []string
	debug     bool
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

// printTop prints top of list.
func printTop(columns []string, padding []int) {
	fmt.Println()
	prettyPrint(updateRowPad(padding, columns))
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

// prettyPrint prints a string array without brackets.
func prettyPrint(arrStr ...[]string) {
	for fi := range arrStr {
		finalString := ""
		for item := range arrStr[fi] {
			finalString = fmt.Sprint(finalString, arrStr[fi][item])
		}
		fmt.Println(finalString)
	}
}

// updateRowPad updates the length of all strings in a given list
// to match wanted length.
func updateRowPad(columnPad []int, columnList []string) []string {
	// Make sure padding list is at least as long as row list
	st := columnList
	fixLen := len(columnPad) - len(st)
	if fixLen < 0 {
		// Get last value of column pad and simply re-use it to fill columnPad up
		valueToUse := columnPad[len(columnPad)-1]
		for i := 0; i < fixLen*(-1); i++ {
			columnPad = append(columnPad, valueToUse)
		}
	}
	for i := range st {
		// Add padding
		if len(st[i]) < columnPad[i] {
			st[i] = st[i] + strings.Repeat(" ", columnPad[i]-len(st[i]))
			// Clip
		} else {
			st[i] = st[i][:columnPad[i]-2] + "  "
		}
	}
	return st
}

// findMapValue will find a key and retrieve its value from the given map. The key has to be
// a string and can be multilayered, for example `foo.bar`. Returns the value and a boolean
// indicating if the value was found.
func findMapValue(data map[string]interface{}, key string) (string, bool) {

	// Split key into array
	keys := strings.Split(key, ".")

	// loop though elements in sliced string:
	for _, element := range keys {

		// if key is found, continue:
		if val, ok := data[element]; ok {

			switch typed := val.(type) {

			// If key points to string:
			case string:
				return typed, true

			// If key points to interface insance:
			case map[string]interface{}:
				data = typed

			// If key points to an integer:
			case int:
				return strconv.Itoa(typed), true

			// If key points to a float:
			case float32:
				return fmt.Sprintf("%f", typed), true

			case float64:
				return fmt.Sprintf("%f", typed), true

			// If key points to bool:
			case bool:
				return strconv.FormatBool(typed), true

			// Not dealing with other possible datatypes:
			default:
				return "", false

			}

			// Key not in map
		} else {
			return "", false
		}
	}

	return "", false
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
	flags.StringArrayVar(
		&args.parameter,
		"parameter",
		nil,
		"Query parameters to add to the request. The value must be the name of the "+
			"parameter, followed by an optional equals sign and then the value "+
			"of the parameter. Can be used multiple times to specifprintTopy multiple "+
			"parameters or multiple values for the same parameter.",
	)
	flags.BoolVar(
		&args.managed,
		"managed",
		false,
		"Filter managed/unmanaged clusters",
	)
	flags.BoolVar(
		&args.step,
		"step",
		false,
		"Load pages one step at a time",
	)
	flags.StringVar(
		&args.columns,
		"columns",
		"id, name, api.url, version.id, region.id",
		"Specify which columns to display separated by commas, path is based on Cluster struct i.e. "+
			"'id, name, api.url, version.id, region.id' "+
			"will output the default values.",
	)
	flags.IntVar(
		&args.padding,
		"padding",
		45,
		"Takes padding for custom columns, default to 35.",
	)
}

func run(cmd *cobra.Command, argv []string) {

	pageSize := 100
	pageIndex := 1

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
	connection, err := cfg.Connection(args.debug)
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

	// Update our column name displaying variable:
	args.columns = strings.Replace(args.columns, " ", "", -1)
	colUpper := strings.ToUpper(args.columns)
	colUpper = strings.Replace(colUpper, ".", " ", -1)
	columnNames := strings.Split(colUpper, ",")
	paddingByColumn := []int{args.padding}

	// Print Header Row:
	printTop(columnNames, paddingByColumn)

	for {
		// Fetch the next page:
		response := getResponse(collection, managed, args.parameter, pageSize, pageIndex)

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

			// Get JSON from Cluster bytes
			err = json.Unmarshal(boddyBuffer.Bytes(), &jsonBody)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to turn cluster bytes into JSON map: %s\n", err)
				os.Exit(1)
			}

			// Loop through wanted columns and populate a cluster instance
			iter := strings.Split(args.columns, ",")
			thisCluster := []string{}
			for _, element := range iter {
				value, status := findMapValue(jsonBody, element)
				if !status {
					value = "NONE"
				}
				thisCluster = append(thisCluster, value)
				thisCluster = updateRowPad([]int{args.padding}, thisCluster)
			}

			// Print current cluster:
			prettyPrint(thisCluster)
			return true

		})

		// if step was flagged, load only one page at a time:
		if args.step {
			if response.Size() < pageSize {
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
			printTop(columnNames, paddingByColumn)
		}

		// If the number of fetched results is less than requested, then
		// this was the last page, otherwise process the next one:
		if response.Size() < pageSize {
			break
		}
		pageIndex++
	}

	fmt.Println()
}

func getResponse(collection *v1.ClustersClient,
	managed bool,
	parameter []string,
	pageSize int,
	pageIndex int) *v1.ClustersListResponse {

	listRequest := collection.List().
		Size(pageSize).
		Page(pageIndex)

	if managed {
		listRequest.Search("managed='true'")
	} else if len(parameter) > 0 {
		for _, parameter := range args.parameter {
			var name string
			var value string
			position := strings.Index(parameter, "=")
			if position != -1 {
				name = parameter[:position]
				value = parameter[position+1:]
			} else {
				name = parameter
				value = ""
			}
			listRequest.Parameter(name, value)
		}
	}

	response, err := listRequest.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't retrieve clusters: %s\n", err)
		os.Exit(1)
	}

	return response
}
