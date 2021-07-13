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
	"context"
	"fmt"
	"os"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/output"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
	columns    string
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
	fs := Cmd.Flags()
	fs.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to list the IdP of (required).",
	)
	fs.StringVar(
		&args.columns,
		"columns",
		"name, type, auth_url",
		"Comma separated list of columns to display.",
	)

	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {
	// Create a context:
	ctx := context.Background()

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return err
	}
	defer connection.Close()

	// Create the output printer:
	printer, err := output.NewPrinter().
		Writer(os.Stdout).
		Pager(cfg.Pager).
		Build(ctx)
	if err != nil {
		return err
	}
	defer printer.Close()

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

	// Get the client for the resource that manages the collection of clusters:
	ocmClient := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(connection, clusterKey)
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

	// Create the output table:
	table, err := printer.NewTable().
		Name("idps").
		Columns(args.columns).
		Value("type", getType).
		Value("auth_url", func(idp *cmv1.IdentityProvider) string {
			return getAuthURL(cluster, idp.Name())
		}).
		Build(ctx)
	if err != nil {
		return err
	}
	defer table.Close()

	// Write the column headers:
	err = table.WriteHeaders()
	if err != nil {
		return err
	}

	// Write the rows:
	for _, idp := range idps {
		err = table.WriteObject(idp)
		if err != nil {
			break
		}
	}
	if err != nil {
		return err
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
