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

package cluster

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey string

	// Basic options
	expirationTime     string
	expirationDuration time.Duration

	// Scaling options
	computeNodes int

	// Networking options
	private bool
}

var Cmd = &cobra.Command{
	Use:   "cluster --cluster={NAME|ID|EXTERNAL_ID}",
	Short: "Edit cluster",
	Long:  "Edit cluster.",
	Example: `  # Edit a cluster named "mycluster" to make it private
  ocm edit cluster mycluster --private`,
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
		"Name or ID or external_id of the cluster.",
	)
	arguments.Must(Cmd.MarkFlagRequired("cluster"))

	// Basic options
	flags.StringVar(
		&args.expirationTime,
		"expiration-time",
		"",
		"Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	flags.DurationVar(
		&args.expirationDuration,
		"expiration",
		0,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	// Cluster expiration is not supported in production
	arguments.Must(flags.MarkHidden("expiration-time"))
	arguments.Must(flags.MarkHidden("expiration"))

	// Scaling options
	flags.IntVar(
		&args.computeNodes,
		"compute-nodes",
		0,
		"Number of worker nodes to provision per zone. Single zone clusters need at least 4 nodes, "+
			"while multizone clusters need at least 9 nodes (3 per zone) for resiliency.",
	)

	// Networking options
	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint to direct, private connectivity.",
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

	// Validate flags:
	expiration, err := c.ValidateClusterExpiration(args.expirationTime, args.expirationDuration)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s", err))
	}

	var private *bool
	if cmd.Flags().Changed("private") {
		private = &args.private
	}

	var computeNodes int
	if cmd.Flags().Changed("compute-nodes") {
		computeNodes = args.computeNodes
	}

	clusterConfig := c.Spec{
		Expiration:   expiration,
		ComputeNodes: computeNodes,
		Private:      private,
	}
	err = c.UpdateCluster(clusterCollection, cluster.ID(), clusterConfig)
	if err != nil {
		return fmt.Errorf("Failed to update cluster: %v", err)
	}

	return nil

}
