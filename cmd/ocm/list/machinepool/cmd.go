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
	"os"
	"strings"
	"text/tabwriter"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "machinepools",
	Aliases: []string{"machine-pool", "machine-pools", "machinepool"},
	Short:   "List cluster machine pools",
	Long:    "List machine pools for a cluster.",
	Example: `  # List all machine pools on a cluster named "mycluster"
  ocm list machine-pools --cluster=mycluster`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to list the machine pools of (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
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

	machinePools, err := c.GetMachinePools(clusterCollection, cluster.ID())
	if err != nil {
		return err
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tAVAILABILITY ZONES\t\t\tINSTANCE TYPE\t\tLABELS\t\tREPLICAS\n")
	fmt.Fprintf(writer, "default\t%s\t\t\t%s\t\t%v\t\t%d\n",
		strings.Join(cluster.Nodes().AvailabilityZones(), ", "),
		cluster.Nodes().ComputeMachineType().ID(),
		printLabels(cluster.Nodes().ComputeLabels()),
		cluster.Nodes().Compute(),
	)
	for _, machinePool := range machinePools {
		fmt.Fprintf(writer, "%s\t%s\t\t\t%s\t\t%v\t\t%d\n",
			machinePool.ID(),
			strings.Join(machinePool.AvailabilityZones(), ", "),
			machinePool.InstanceType(),
			printLabels(machinePool.Labels()),
			machinePool.Replicas(),
		)
	}
	//nolint:gosec
	writer.Flush()

	return nil
}

func printLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	output := []string{}
	for k, v := range labels {
		output = append(output, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(output, ", ")
}
