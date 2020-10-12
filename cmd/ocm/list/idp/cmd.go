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
	"os"
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
	Use:     "idps --cluster={NAME|ID|EXTERNAL_ID}",
	Aliases: []string{"idp"},
	Short:   "List cluster IDPs",
	Long:    "List identity providers for a cluster.",
	Example: `  # List all identity providers on a cluster named "mycluster"
  ocm list idps --cluster=mycluster`,
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
		"Name or ID of the cluster to list the IdP of (required).",
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

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	ocmClient := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(ocmClient, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	idps, err := c.GetIdentityProviders(ocmClient, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "NAME\t\tTYPE\t\tAUTH URL\n")
	for _, idp := range idps {
		fmt.Fprintf(writer, "%s\t\t%s\t\t%s\n", idp.Name(), getType(idp), getAuthURL(cluster, idp.Name()))
	}
	err = writer.Flush()
	if err != nil {
		return nil
	}
	return nil
}

func getType(idp *cmv1.IdentityProvider) string {
	switch idp.Type() {
	case "GithubIdentityProvider":
		return "GitHub"
	case "GoogleIdentityProvider":
		return "Google"
	case "LDAPIdentityProvider":
		return "LDAP"
	case "OpenIDIdentityProvider":
		return "OpenID"
	}

	return ""
}

func getAuthURL(cluster *cmv1.Cluster, idpName string) string {
	oauthURL := c.GetClusterOauthURL(cluster)
	return fmt.Sprintf("%s/oauth2callback/%s", oauthURL, idpName)
}
