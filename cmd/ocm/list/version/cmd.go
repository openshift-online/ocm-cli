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

package version

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/spf13/cobra"
)

var args struct {
	defaultVersion bool
	channelGroup   string
	marketplaceGcp string
}

var Cmd = &cobra.Command{
	Use:     "versions",
	Aliases: []string{"version"},
	Short:   "List available versions",
	Long:    "List the versions available for provisioning a cluster",
	Example: `  # List all supported cluster versions
  ocm list versions`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	fs := Cmd.Flags()
	fs.BoolVarP(
		&args.defaultVersion,
		"default",
		"d",
		false,
		"Show only the default version",
	)
	fs.StringVar(
		&args.channelGroup,
		"channel-group",
		"stable",
		"List only versions from the specified channel group",
	)

	fs.StringVar(
		&args.marketplaceGcp,
		"marketplace-gcp",
		"",
		"List only versions that support 'marketplace-gcp' subscription type",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	client := connection.ClustersMgmt().V1()
	versions, defaultVersion, err := cluster.GetEnabledVersions(client, args.channelGroup, args.marketplaceGcp, "")
	if err != nil {
		return fmt.Errorf("Can't retrieve versions: %v", err)
	}

	if args.defaultVersion {
		fmt.Println(defaultVersion)
	} else {
		for _, v := range versions {
			fmt.Println(v)
		}
	}

	return nil
}
