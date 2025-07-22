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

package idp

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "idp --cluster={NAME|ID|EXTERNAL_ID} [flags] IDP_NAME",
	Aliases: []string{"idps"},
	Short:   "Delete cluster IDPs",
	Long:    "Delete a specific identity provider for a cluster.",
	Example: `  # Delete an identity provider named github-1
  ocm delete idp github-1 --cluster=mycluster`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to delete the IdP from (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {

	// Check command line arguments:
	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameters containing the name " +
				"of the Identity provider.",
		)
	}

	idpName := argv[0]
	if idpName == "" {
		return fmt.Errorf("Identity provider name is required")
	}

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

	idps, err := c.GetIdentityProviders(clusterCollection, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
	}

	var idp *cmv1.IdentityProvider
	for _, item := range idps {
		if item.Name() == idpName {
			idp = item
		}
	}
	if idp == nil {
		return fmt.Errorf("Failed to get identity provider '%s' for cluster '%s'", idpName, clusterKey)
	}

	_, err = clusterCollection.
		Cluster(cluster.ID()).
		IdentityProviders().
		IdentityProvider(idp.ID()).
		Delete().
		Send()
	if err != nil {
		return fmt.Errorf("Failed to delete identity provider '%s' on cluster '%s'", idpName, clusterKey)
	}
	fmt.Printf("Deleted identity provider '%s' on cluster '%s'\n", idpName, clusterKey)
	return nil
}
