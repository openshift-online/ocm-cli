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

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/provider"
	"github.com/spf13/cobra"
)

var args struct {
	provider           string
	ccs                bool
	awsAccessKeyID     string
	awsSecretAccessKey string
}

var Cmd = &cobra.Command{
	Use:     "region --provider=CLOUD_PROVIDER [--ccs --aws-access-key-id --aws-secret-access-key] ",
	Aliases: []string{"regions"},
	Short:   "List known/available cloud provider regions",
	Long: "List regions of a cloud provider.\n\n" +
		"In --ccs mode, fetch regions that would be available to *your* cloud account\n" +
		"(currently only supported with --provider=aws).",
	RunE: run,
}

func init() {
	fs := Cmd.Flags()
	fs.StringVar(
		&args.provider,
		"provider",
		"",
		"Lists the regions for the specific cloud provider",
	)

	//nolint:gosec
	Cmd.MarkFlagRequired("provider")
	fs.BoolVar(
		&args.ccs,
		"ccs",
		false,
		"Lists  the regions specific to your cloud account",
	)
	fs.StringVar(
		&args.awsAccessKeyID,
		"aws-access-key-id",
		"",
		"AWS access key",
	)

	fs.StringVar(
		&args.awsSecretAccessKey,
		"aws-secret-access-key",
		"",
		"AWS Secret Access",
	)

}

func run(cmd *cobra.Command, argv []string) error {
	ccs := cluster.CCS{}
	if args.ccs {
		if args.provider == "gcp" {
			return fmt.Errorf("--ccs flag is not yet supported for GCP clusters")
		}
		if args.awsAccessKeyID == "" {
			return fmt.Errorf("--aws-access-key-id flag is mandatory for --ccs=true")
		}
		if args.awsSecretAccessKey == "" {
			return fmt.Errorf("--aws-secret-access-key flag is mandatory for --ccs=true")
		}
		ccs = cluster.CCS{
			Enabled: args.ccs,
			AWS: cluster.AWSCredentials{
				AccessKeyID:     args.awsAccessKeyID,
				SecretAccessKey: args.awsSecretAccessKey,
			},
		}
	}
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	regions, err := provider.GetRegions(connection.ClustersMgmt().V1(), args.provider, ccs)
	if err != nil {
		return err
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)

	//We display only the enabled region for both ccs and non ccs regions
	if args.provider == "aws" && args.ccs {
		fmt.Fprintf(writer, "ID\t\tSUPPORTS MULTI-AZ\n")
		for _, region := range regions {
			if !region.Enabled() {
				continue
			}
			fmt.Fprintf(writer, "%s\t\t%v\n",
				region.ID(), region.SupportsMultiAZ())
		}
	} else {
		fmt.Fprintf(writer, "ID\t\tON RED HAT INFRA\t\tCCS ONLY\t\tSUPPORTS MULTI-AZ\n")
		for _, region := range regions {
			if !region.Enabled() {
				continue
			}
			fmt.Fprintf(writer, "%s\t\t%v\t\t%v\t\t%v\n",
				region.ID(), !region.CCSOnly(), region.CCSOnly(), region.SupportsMultiAZ())
		}

	}

	err = writer.Flush()
	return err
}
