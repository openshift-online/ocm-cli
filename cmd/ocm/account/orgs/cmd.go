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

package orgs

import (
	"context"
	"fmt"
	"os"

	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/output"
)

var args struct {
	columns   string
	parameter []string
	header    []string
}

var Cmd = &cobra.Command{
	Use:   "orgs",
	Short: "List organizations.",
	Long:  "Display a list of organizations.",
	Args:  cobra.NoArgs,
	RunE:  run,
}

func init() {
	// Add flags to rootCmd:
	fs := Cmd.Flags()
	arguments.AddParameterFlag(fs, &args.parameter)
	arguments.AddHeaderFlag(fs, &args.header)
	fs.StringVar(
		&args.columns,
		"columns",
		"id,name",
		"Comma separated list of columns to display.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	// Create a context:
	ctx := context.Background()

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return err
	}
	defer connection.Close()

	// Create the output printer:
	printer, err := output.NewPrinter().
		Writer(os.Stdout).
		Pager(cfg.Pager).
		Build(ctx)
	if err != nil {
		return err
	}
	defer printer.Close()

	// Create the output table:
	table, err := printer.NewTable().
		Name("orgs").
		Columns(args.columns).
		Build(ctx)
	if err != nil {
		return err
	}
	defer table.Close()

	// Write the header row:
	err = table.WriteHeaders()
	if err != nil {
		return err
	}

	// Create the request. Note that this request can be created outside of the loop and used
	// for all iterations just changing the values of the `size` and `page` parameters.
	request := connection.AccountsMgmt().V1().Organizations().List()
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
			return fmt.Errorf("can't retrieve organizations: %w", err)
		}

		// Display the items of the fetched page:
		response.Items().Each(func(org *amv1.Organization) bool {
			err = table.WriteObject(org)
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
