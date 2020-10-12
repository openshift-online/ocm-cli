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

package provider

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/ocm-cli/pkg/ocm"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "providers",
	Aliases: []string{"provider"},
	Short:   "List known cloud providers",
	Long:    "List known cloud providers.",
	RunE:    run,
}

func run(cmd *cobra.Command, argv []string) error {
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	collection := connection.ClustersMgmt().V1().CloudProviders().List()
	// There are just a few providers, can get them all in one page.
	response, err := collection.Page(1).Size(-1).Send()
	if err != nil {
		return err
	}
	providers := response.Items().Slice()

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintf(writer, "NAME\t\tDISPLAY NAME\n")
	for _, provider := range providers {
		fmt.Fprintf(writer, "%s\t\t%s\n", provider.Name(), provider.DisplayName())
	}
	err = writer.Flush()
	return err
}
