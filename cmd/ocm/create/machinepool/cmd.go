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

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey   string
	instanceType string
	replicas     int
	labels       string
}

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Add machine pool to cluster",
	Long:    "Add a machine pool to the cluster.",
	Example: `  # Add a machine pool to a cluster named "mycluster"
  ocm create machinepool --cluster=mycluster mp-1
  # Add a machine pool mp-1 with 3 replicas to a cluster
  ocm create machinepool --cluster=mycluster --replicas=3 mp-1
  # Add a machine pool mp-1 with labels to a cluster
  ocm create machinepool --cluster=mycluster --labels="foo=bar,bar=baz" mp-1`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the machine pool to (required).",
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

	//nolint:gosec
	Cmd.MarkFlagRequired("replicas")

	flags.StringVar(
		&args.labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
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

	_, err = c.ValidateMachineType(
		connection.ClustersMgmt().V1(),
		cluster.CloudProvider().ID(),
		args.instanceType)
	if err != nil {
		return fmt.Errorf("Expected a valid machine type: %s", err)
	}

	machinePool, err := cmv1.NewMachinePool().ID(machinePoolID).
		InstanceType(args.instanceType).
		Replicas(args.replicas).
		Labels(labels).
		Build()

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
