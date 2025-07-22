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
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey  string
	replicas    int
	autoscaling c.Autoscaling
	labels      string
	taints      string
}

var Cmd = &cobra.Command{
	Use:     "machinepool --cluster={NAME|ID|EXTERNAL_ID} [flags] MACHINE_POOL_ID",
	Aliases: []string{"machine-pool"},
	Short:   "Edit a cluster machine pool",
	Long:    "Edit a machine pool size.",
	Example: `  #  Update the number of replicas for machine pool with ID 'a1b2'
  ocm edit machinepool --replicas=3 --cluster=mycluster a1b2
  # Enable autoscaling and Set 3-5 replicas on machine pool 'mp1' on cluster 'mycluster'
  ocm edit machinepool --enable-autoscaling --min-replicas=3 max-replicas=5 --cluster=mycluster mp1`,
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
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.IntVar(
		&args.replicas,
		"replicas",
		-1,
		"Count of machines for this machine pool.",
	)

	arguments.AddAutoscalingFlags(flags, &args.autoscaling)

	flags.StringVar(
		&args.labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:scheduleType'. "+
			"This list will overwrite any modifications made to Node taints on an ongoing basis.",
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

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	labels := make(map[string]string)
	if args.labels != "" {
		for _, label := range strings.Split(args.labels, ",") {
			if !strings.Contains(label, "=") {
				return fmt.Errorf("Expected key=value format for label-match")
			}
			tokens := strings.Split(label, "=")
			labels[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}

	taintBuilders := []*cmv1.TaintBuilder{}

	if args.taints != "" {
		for _, taint := range strings.Split(args.taints, ",") {
			if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
				return fmt.Errorf("Expected key=value:scheduleType format for taints")
			}
			tokens := strings.FieldsFunc(taint, arguments.Split)
			taintBuilders = append(taintBuilders, cmv1.NewTaint().Key(tokens[0]).Value(tokens[1]).Effect(tokens[2]))
		}
	}

	machinePoolBuilder := cmv1.NewMachinePool().ID(machinePoolID)

	if cmd.Flags().Changed("labels") {
		machinePoolBuilder = machinePoolBuilder.Labels(labels)
	}

	if cmd.Flags().Changed("taints") {
		machinePoolBuilder = machinePoolBuilder.Taints(taintBuilders...)
	}

	err = validateAutoscalingReplicasFlags(cmd)
	if err != nil {
		return err
	}

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")

	// Editing the default machine pool is a different process
	if machinePoolID == "default" {
		if isReplicasSet {
			err = validateComputeNodes(args.replicas, cluster.CCS().Enabled(), cluster.MultiAZ())
			if err != nil {
				return err
			}
		}
		if isMinReplicasSet {
			err = validateComputeNodes(args.autoscaling.MinReplicas, cluster.CCS().Enabled(), cluster.MultiAZ())
			if err != nil {
				return err
			}
		}

		clusterConfig := c.Spec{
			Autoscaling: c.Autoscaling{
				Enabled:     args.autoscaling.Enabled,
				MinReplicas: args.autoscaling.MinReplicas,
				MaxReplicas: args.autoscaling.MaxReplicas,
			},
			ComputeNodes: args.replicas,
		}

		err = c.UpdateCluster(clusterCollection, cluster.ID(), clusterConfig)
		if err != nil {
			return fmt.Errorf("Failed to update machine pool '%s' on cluster '%s': %s",
				machinePoolID, clusterKey, err)
		}

		return nil
	}

	if args.autoscaling.Enabled {
		asBuilder := cmv1.NewMachinePoolAutoscaling()

		if isMinReplicasSet {
			asBuilder = asBuilder.MinReplicas(args.autoscaling.MinReplicas)
		}
		if isMaxReplicasSet {
			asBuilder = asBuilder.MaxReplicas(args.autoscaling.MaxReplicas)
		}

		machinePoolBuilder = machinePoolBuilder.Autoscaling(asBuilder)
	} else if isReplicasSet {
		machinePoolBuilder = machinePoolBuilder.Replicas(args.replicas)
	}

	machinePool, err := machinePoolBuilder.Build()

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

func validateComputeNodes(nodes int, ccs bool, multiAZ bool) error {
	var min int
	if ccs {
		if multiAZ {
			min = 3
		} else {
			min = 2
		}
	} else {
		if multiAZ {
			min = 9
		} else {
			min = 4
		}
	}

	if nodes < min {
		return fmt.Errorf("Minimum is %d nodes", min)
	}

	if multiAZ && nodes%3 != 0 {
		return fmt.Errorf("Multi-zone clusters require nodes to be multiple of 3")
	}
	return nil
}

func validateAutoscalingReplicasFlags(cmd *cobra.Command) error {
	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")

	// Inferr autoscaling from flags - to avoid getting the existing machine pool
	// defering to OCM validators
	if !isAutoscalingSet && (isMaxReplicasSet || isMinReplicasSet) {
		args.autoscaling.Enabled = true
	}

	if args.autoscaling.Enabled {
		if isReplicasSet {
			return fmt.Errorf("--replicas can't be set with autoscaling paramteres")
		}
		if !isMaxReplicasSet && !isMinReplicasSet {
			return fmt.Errorf(
				"at least one of '--min-replicas' and '--max-replicas' is required when enabling autoscaling")
		}
	}

	if isAutoscalingSet && !args.autoscaling.Enabled {
		if isMinReplicasSet {
			return fmt.Errorf("--min-replicas can't be set when setting --enable-autoscaling=false")
		}
		if isMaxReplicasSet {
			return fmt.Errorf("--max-replicas can't be set when setting --enable-autoscaling=false")
		}
		if !isReplicasSet {
			return fmt.Errorf("--replicas is required when setting --enable-autoscaling=false")
		}
	}
	return nil
}
