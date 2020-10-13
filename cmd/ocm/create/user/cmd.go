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
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey string
	group      string
}

var Cmd = &cobra.Command{
	Use:     "user --cluster={NAME|ID|EXTERNAL_ID} --group=GROUP_ID [flags] USERS",
	Aliases: []string{"users"},
	Short:   "Configure user access for cluster",
	Long:    "Add users (comma-separated) to a priviledged group on a cluster.",
	Example: `  # Add users to the dedicated-admins group
  ocm create user user1,user2 --cluster=mycluster --group=dedicated-admins`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to add the user to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.group,
		"group",
		"",
		"Group name to add the users to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("group")
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

	if len(argv) != 1 || argv[0] == "" {
		return fmt.Errorf("At least one user must be specified")
	}
	users := argv[0]
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

	_, err = clusterCollection.Cluster(cluster.ID()).
		Groups().
		Group(args.group).
		Get().
		Send()
	if err != nil {
		return fmt.Errorf("Group '%s' in cluster '%s' doesn't exist", args.group, clusterKey)
	}

	for _, username := range strings.Split(users, ",") {
		user, err := cmv1.NewUser().ID(username).Build()
		if err != nil {
			return fmt.Errorf("Failed to create '%s' user '%s' for cluster '%s'", args.group, username, clusterKey)
		}
		_, err = clusterCollection.Cluster(cluster.ID()).
			Groups().
			Group(args.group).
			Users().
			Add().
			Body(user).
			Send()
		if err != nil {
			fmt.Printf("Failed to add '%s' user '%s' to cluster '%s': %v\n", args.group, username, clusterKey, err)
			continue
		}
		fmt.Printf("Added '%s' user '%s' to cluster '%s'\n", args.group, username, clusterKey)
	}

	return nil
}
