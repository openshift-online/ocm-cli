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

package upgradepolicy

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "upgradepolicies --cluster={NAME|ID|EXTERNAL_ID}",
	Aliases: []string{"upgrade-policy", "upgrade-policies", "upgradepolicy"},
	Short:   "List cluster upgrade policies",
	Long:    "List upgrade policies for a cluster.",
	Example: `  # List all upgrade policies on a cluster named "mycluster"
  ocm list upgradepolicies --cluster=mycluster`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to list the upgrade policies of (required).",
	)
	arguments.Must(Cmd.MarkFlagRequired("cluster"))
}

func run(cmd *cobra.Command, argv []string) error {

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the cluster management api
	clusterCollection := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(clusterCollection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	upgradePolicies, err := c.GetUpgradePolicies(clusterCollection, cluster.ID())
	if err != nil {
		return err
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tSCHEDULE TYPE\t\t\tUPGRADE VERSION\t\tNEXT RUN\n")
	for _, upgradePolicy := range upgradePolicies {
		fmt.Fprintf(writer, "%s\t%s\t\t\t%s\t\t%v\n",
			upgradePolicy.ID(),
			upgradePolicy.ScheduleType(),
			upgradePolicy.Version(),
			upgradePolicy.NextRun())
	}

	//nolint:gosec
	writer.Flush()

	return nil
}
