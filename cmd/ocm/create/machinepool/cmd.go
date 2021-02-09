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
	"github.com/openshift-online/ocm-cli/pkg/provider"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey   string
	instanceType string
	replicas     int
	autoscaling  c.Autoscaling
	labels       string
	taints       string
}

var Cmd = &cobra.Command{
	Use:     "machinepool --cluster={NAME|ID|EXTERNAL_ID} --instance-type=TYPE --replicas=N [flags] MACHINE_POOL_ID",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Add machine pool to cluster",
	Long:    "Add a machine pool to the cluster.",
	Example: `  # Add a machine pool mp-1 with 3 replicas and m5.xlarge instance type to a cluster
  ocm create machinepool --cluster mycluster --instance-type m5.xlarge --replicas 3 mp-1
  # Add a machine pool mp-1 with autoscaling enabled and 3 to 6 replicas of m5.xlarge to a cluster
  ocm create machinepool --cluster=mycluster --enable-autoscaling \
  --min-replicas=3 --max-replicas=6 --instance-type=m5.xlarge mp-1 
  # Add a machine pool mp-1 with labels and m5.xlarge instance type to a cluster
  ocm create machinepool --cluster mycluster --instance-type m5.xlarge --replicas 3 --labels "foo=bar,bar=baz" mp-1
  # Add a machine pool mp-1 with taints and m5.xlarge instance type to a cluster
  ocm create machinepool --cluster mycluster --instance-type m5.xlarge --replicas 3 --taints "foo=bar:NoSchedule" mp-1`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to add the machine pool to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.instanceType,
		"instance-type",
		"",
		"Instance type that should be used.",
	)

	//nolint:gosec
	Cmd.MarkFlagRequired("instance-type")

	flags.IntVar(
		&args.replicas,
		"replicas",
		0,
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

	if len(argv) < 1 || argv[0] == "" {
		return fmt.Errorf("Missing machine pool ID")
	}
	machinePoolID := argv[0]

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

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isReplicasSet := cmd.Flags().Changed("replicas")

	if args.autoscaling.Enabled {
		if isReplicasSet {
			return fmt.Errorf("--replicas is only allowed when --enable-autoscaling=false")
		}

		if !isMaxReplicasSet || !isMinReplicasSet {
			return fmt.Errorf("Both --min-replicas and --max-replicas are required when --enable-autoscaling=true")
		}
	} else {
		if !isReplicasSet {
			return fmt.Errorf("--replicas is required when --enable-autoscaling=false")
		}

		if isMaxReplicasSet || isMinReplicasSet {
			return fmt.Errorf("--min-replicas and --max-replicas are not allowed when --enable-autoscaling=false")
		}
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

	machineTypeList, err := provider.GetMachineTypeOptions(connection.ClustersMgmt().V1(),
		cluster.CloudProvider().ID())
	if err != nil {
		return err
	}
	err = arguments.CheckOneOf(cmd.Flags(), "instance-type", machineTypeList)
	if err != nil {
		return err
	}

	mpBuilder := cmv1.NewMachinePool().
		ID(machinePoolID).
		InstanceType(args.instanceType).
		Labels(labels).
		Taints(taintBuilders...)

	if args.autoscaling.Enabled {
		mpBuilder = mpBuilder.Autoscaling(
			cmv1.NewMachinePoolAutoscaling().
				MinReplicas(args.autoscaling.MinReplicas).
				MaxReplicas(args.autoscaling.MaxReplicas))
	} else {
		mpBuilder = mpBuilder.Replicas(args.replicas)
	}

	machinePool, err := mpBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
	}

	_, err = clusterCollection.Cluster(cluster.ID()).
		MachinePools().
		Add().
		Body(machinePool).
		Send()
	if err != nil {
		return fmt.Errorf("Failed to add machine pool to cluster '%s': %v", clusterKey, err)
	}
	return nil
}
