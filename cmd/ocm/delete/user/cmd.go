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

package user

import (
	"fmt"

	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey string
	group      string
}

var Cmd = &cobra.Command{
	Use:     "user --cluster={NAME|ID|EXTERNAL_ID} --group=GROUP_ID [flags] USER1",
	Aliases: []string{"users"},
	Short:   "Remove user access from cluster",
	Long:    "Remove a user from a priviledged group on a cluster.",
	Example: `# Delete users from the dedicated-admins group
  ocm delete user user1 --cluster=mycluster --group=dedicated-admins`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to delete the user from (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.group,
		"group",
		"",
		"Group name to delete the user from.",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("group")
}

func run(cmd *cobra.Command, argv []string) error {

	// Check command line arguments:
	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameters containing the user name")
	}
	username := argv[0]

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

	_, err = ocm.SendTypedAndHandleDeprecation(clusterCollection.Cluster(cluster.ID()).
		Groups().
		Group(args.group).
		Get())
	if err != nil {
		return fmt.Errorf("Group '%s' in cluster '%s' doesn't exist", args.group, clusterKey)
	}

	_, err = ocm.SendTypedAndHandleDeprecation(clusterCollection.
		Cluster(cluster.ID()).
		Groups().
		Group(args.group).
		Users().
		User(username).
		Delete())
	if err != nil {
		return fmt.Errorf("Failed to delete '%s' user '%s' on cluster '%s'", args.group, username, clusterKey)
	}

	fmt.Printf("Deleted '%s' user '%s' on cluster '%s'\n", args.group, username, clusterKey)
	return nil
}
