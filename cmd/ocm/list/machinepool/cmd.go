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
	Use:     "machinepools --cluster={NAME|ID|EXTERNAL_ID}",
	Aliases: []string{"machine-pool", "machine-pools", "machinepool"},
	Short:   "List cluster machine pools",
	Long:    "List machine pools for a cluster.",
	Example: `  # List all machine pools on a cluster named "mycluster"
  ocm list machine-pools --cluster=mycluster`,
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
		"Name or ID or external_id of the cluster to list the machine pools of (required).",
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

	fmt.Fprintf(writer, "ID\tAUTOSCALING\tREPLICAS\tINSTANCE TYPE\tLABELS\t\tTAINTS\t\tAVAILABILITY ZONES\n")
	fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\n",
		"default",
		printAutoscaling(cluster.Nodes().AutoscaleCompute()),
		printReplicas(cluster.Nodes().AutoscaleCompute(), cluster.Nodes().Compute()),
		cluster.Nodes().ComputeMachineType().ID(),
		printLabels(cluster.Nodes().ComputeLabels()),
		"",
		printAZ(cluster.Nodes().AvailabilityZones()),
	)
	for _, machinePool := range machinePools {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t\t%s\t\t%s\n",
			machinePool.ID(),
			printAutoscaling(machinePool.Autoscaling()),
			printReplicas(machinePool.Autoscaling(), machinePool.Replicas()),
			machinePool.InstanceType(),
			printLabels(machinePool.Labels()),
			printTaints(machinePool.Taints()),
			printAZ(machinePool.AvailabilityZones()),
		)
	}
	writer.Flush()

	return nil
}

func printAutoscaling(autoscaling *cmv1.MachinePoolAutoscaling) string {
	if autoscaling != nil {
		return "Yes"
	}
	return "No"
}

func printReplicas(autoscaling *cmv1.MachinePoolAutoscaling, replicas int) string {
	if autoscaling != nil {
		return fmt.Sprintf("%d-%d",
			autoscaling.MinReplicas(),
			autoscaling.MaxReplicas())
	}
	return fmt.Sprintf("%d", replicas)
}

func printAZ(az []string) string {
	if len(az) == 0 {
		return ""
	}
	return strings.Join(az, ", ")
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

func printTaints(taints []*cmv1.Taint) string {
	if len(taints) == 0 {
		return ""
	}
	output := []string{}
	for _, taint := range taints {
		output = append(output, fmt.Sprintf("%s=%s:%s", taint.Key(), taint.Value(), taint.Effect()))
	}

	return strings.Join(output, ", ")
}
