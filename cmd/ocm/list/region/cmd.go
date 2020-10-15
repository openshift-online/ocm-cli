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

package region

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/provider"
	"github.com/spf13/cobra"
)

var args struct {
	provider string
	ccs      cluster.CCS
}

var Cmd = &cobra.Command{
	Use:     "regions --provider=CLOUD_PROVIDER [--ccs CCS FLAGS]",
	Aliases: []string{"regions"},
	Short:   "List known/available cloud provider regions",
	Long: "List regions of a cloud provider.\n\n" +
		"In --ccs mode, fetch regions that would be available to *your* cloud account\n" +
		"(currently only supported with --provider=aws).",
	RunE: run,
}

func init() {
	fs := Cmd.Flags()
	arguments.AddProviderFlag(fs, &args.provider)
	//nolint:gosec
	Cmd.MarkFlagRequired("provider")
	arguments.AddCCSFlagsWithoutAccountID(fs, &args.ccs)
}

func run(cmd *cobra.Command, argv []string) error {
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	regions, err := provider.GetRegions(connection.ClustersMgmt().V1(), args.provider, args.ccs)
	if err != nil {
		return err
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintf(writer, "IT\t\tDISPLAY NAME\tENABLED\tSUPPORTS MULTI-AZ\n")
	for _, region := range regions {
		fmt.Fprintf(writer, "%s\t\t%s\t%v\t%v\n",
			region.ID(), region.DisplayName(), region.Enabled(), region.SupportsMultiAZ())
	}
	err = writer.Flush()
	return err
}
