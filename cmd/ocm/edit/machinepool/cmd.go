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

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
	replicas   int
}

var Cmd = &cobra.Command{
	Use:     "machinepool --cluster={NAME|ID|EXTERNAL_ID} [flags] MACHINE_POOL_ID",
	Aliases: []string{"machine-pool"},
	Short:   "Edit a cluster machine pool",
	Long:    "Edit a machine pool size.",
	Example: `  #  Update the number of replicas for machine pool with ID 'a1b2'
  ocm edit machinepool --replicas=3 --cluster=mycluster a1b2`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to edit the machine pool (required).",
	)
	arguments.Must(Cmd.MarkFlagRequired("cluster"))

	flags.IntVar(
		&args.replicas,
		"replicas",
		-1,
		"Restrict application route to direct, private connectivity.",
	)
}

func run(cmd *cobra.Command, argv []string) error {

	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameter containing the machine pool ID")
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
	machinePool, err := cmv1.NewMachinePool().ID(machinePoolID).
		Replicas(args.replicas).
		Build()

	if err != nil {
		return fmt.Errorf("Failed to create machine pool body for cluster '%s': %v", clusterKey, err)
	}

	_, err = clusterCollection.
		Cluster(cluster.ID()).
		MachinePools().
		MachinePool(machinePoolID).
		Update().
		Body(machinePool).
		Send()
	if err != nil {
		return fmt.Errorf("Failed to edit machine pool for cluster '%s': %v", clusterKey, err)
	}
	return nil
}
