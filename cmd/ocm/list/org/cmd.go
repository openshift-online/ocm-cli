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

package org

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	table "github.com/openshift-online/ocm-cli/pkg/table"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var args struct {
	columns   string
	parameter []string
	padding   int
}

// Cmd ...
var Cmd = &cobra.Command{
	Use:     "organization",
	Aliases: []string{"organizations", "org", "orgs"},
	Short:   "List organizations.",
	Long:    "Display a list of organizations.",
	Args:    cobra.NoArgs,
	RunE:    run,
}

func init() {
	// Add flags to rootCmd:
	fs := Cmd.Flags()
	arguments.AddParameterFlag(fs, &args.parameter)
	fs.StringVar(
		&args.columns,
		"columns",
		"id,name", // Default value gets assigned later as connection is needed.
		"columns to show.",
	)
	fs.IntVar(
		&args.padding,
		"padding",
		45,
		"Takes padding for custom columns, default to 45.",
	)
}

func run(cmd *cobra.Command, argv []string) error {

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Indices
	pageIndex := 1
	pageSize := 100

	// Setting column names and padding size
	// Update our column name displaying variable:
	args.columns = strings.Replace(args.columns, " ", "", -1)
	colUpper := strings.ToUpper(args.columns)
	colUpper = strings.Replace(colUpper, ".", " ", -1)
	columnNames := strings.Split(colUpper, ",")
	paddingByColumn := []int{29, 65, 70}
	if args.columns != "id,name" {
		paddingByColumn = []int{args.padding}
	}

	// Print Header Row:
	table.PrintPadded(os.Stdout, columnNames, paddingByColumn)

	for {
		// Next page request:
		request := connection.AccountsMgmt().V1().Organizations().
			List().
			Page(pageIndex).
			Size(pageSize)

		// Apply parameters
		arguments.ApplyParameterFlag(request, args.parameter)

		// Fetch next page
		orgList, err := request.Send()
		if err != nil {
			return fmt.Errorf("Failed to retrieve organization list: %v", err)
		}

		// Display organization information
		orgList.Items().Each(func(org *amv1.Organization) bool {
			// String to output marshal -
			// Map used to parse Organization data -
			// Writer to body variable:
			var body string
			var jsonBody map[string]interface{}
			bodyBuffer := bytes.NewBufferString(body)

			// Write Organization data to body variable:
			err := amv1.MarshalOrganization(org, bodyBuffer)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to marshal organization into byte buffer: %s\n", err)
				os.Exit(1)
			}

			// Get JSON from Organization bytes
			err = json.Unmarshal(bodyBuffer.Bytes(), &jsonBody)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to turn organization bytes into JSON map: %s\n", err)
				os.Exit(1)
			}

			// Loop through wanted columns and populate an organization instance
			iter := strings.Split(args.columns, ",")
			thisOrg := []string{}
			for _, element := range iter {
				value, status := table.FindMapValue(jsonBody, element)
				if !status {
					value = "NONE"
				}
				thisOrg = append(thisOrg, value)
			}
			table.PrintPadded(os.Stdout, thisOrg, paddingByColumn)
			return true
		})

		// Break if we reach last page
		if orgList.Size() < pageSize {
			break
		}

		pageIndex++
	}

	return nil
}
