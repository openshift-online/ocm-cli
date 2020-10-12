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

package machinepool

import (
	"fmt"

	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machine-pool", "machinepools", "machine-pools"},
	Short:   "Delete cluster machine pool",
	Long:    "Delete the additional machine pool of a cluster.",
	Example: `  # Delete machine pool with ID mp-1 from a cluster named 'mycluster'
  ocm delete machinepool --cluster=mycluster mp-1`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to delete the machine pool from (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {

	// Check command line arguments:
	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameters containing the ID " +
				"of the machine pool.",
		)
	}

	machinePoolID := argv[0]

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

	_, err = clusterCollection.
		Cluster(cluster.ID()).
		MachinePools().
		MachinePool(machinePoolID).
		Delete().
		Send()
	if err != nil {
		return fmt.Errorf("Failed to delete machine pool '%s' on cluster '%s'", machinePoolID, clusterKey)
	}

	fmt.Printf("Deleted machine pool '%s' on cluster '%s'\n", machinePoolID, clusterKey)
	return nil
}
