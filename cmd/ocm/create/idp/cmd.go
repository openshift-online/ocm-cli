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
	"strconv"
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string

	idpType string
	idpName string

	clientID      string
	clientSecret  string
	mappingMethod string

	// GitHub
	githubHostname      string
	githubOrganizations string
	githubTeams         string

	// Google
	googleHostedDomain string

	// LDAP
	ldapURL          string
	ldapBindDN       string
	ldapBindPassword string
	ldapIDs          string
	ldapUsernames    string
	ldapDisplayNames string
	ldapEmails       string

	// OpenID
	openidIssuerURL   string
	openidEmail       string
	openidName        string
	openidUsername    string
	openidExtraScopes string

	// HTPasswd
	htpasswdUsername string
	htpasswdPassword string
}

var validIdps = []string{"github", "google", "ldap", "openid", "htpasswd"}

var Cmd = &cobra.Command{
	Use:   "idp --cluster={NAME|ID|EXTERNAL_ID}",
	Short: "Add IDP for cluster",
	Long:  "Add an Identity providers to determine how users log into the cluster.",
	Example: `  # Add a GitHub identity provider to a cluster named "mycluster"
  ocm create idp --type=github --cluster=mycluster
  # Add an identity provider following interactive prompts
  ocm create idp --cluster=mycluster`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to add the IdP to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.StringVarP(
		&args.idpType,
		"type",
		"t",
		"",
		fmt.Sprintf("Type of identity provider. Options are %s\n", validIdps),
	)

	flags.StringVarP(
		&args.idpName,
		"name",
		"n",
		"",
		"Name of the identity provider.",
	)

	flags.StringVar(
		&args.mappingMethod,
		"mapping-method",
		"claim",
		"Specifies how new identities are mapped to users when they log in",
	)
	flags.StringVar(
		&args.clientID,
		"client-id",
		"",
		"Client ID from the registered application.",
	)
	flags.StringVar(
		&args.clientSecret,
		"client-secret",
		"",
		"Client Secret from the registered application.\n",
	)

	// GitHub
	flags.StringVar(
		&args.githubHostname,
		"hostname",
		"",
		"GitHub: Optional domain to use with a hosted instance of GitHub Enterprise.",
	)
	flags.StringVar(
		&args.githubOrganizations,
		"organizations",
		"",
		"GitHub: Only users that are members of at least one of the listed organizations will be allowed to log in.",
	)
	flags.StringVar(
		&args.githubTeams,
		"teams",
		"",
		"GitHub: Only users that are members of at least one of the listed teams will be allowed to log in. "+
			"The format is <org>/<team>.\n",
	)

	// Google
	flags.StringVar(
		&args.googleHostedDomain,
		"hosted-domain",
		"",
		"Google: Restrict users to a Google Apps domain. Example: http://redhat.com (scheme required)\n",
	)

	// LDAP
	flags.StringVar(
		&args.ldapURL,
		"url",
		"",
		"LDAP: An RFC 2255 URL which specifies the LDAP search parameters to use.",
	)
	flags.StringVar(
		&args.ldapBindDN,
		"bind-dn",
		"",
		"LDAP: DN to bind with during the search phase.",
	)
	flags.StringVar(
		&args.ldapBindPassword,
		"bind-password",
		"",
		"LDAP: Password to bind with during the search phase.",
	)
	flags.StringVar(
		&args.ldapIDs,
		"id-attributes",
		"dn",
		"LDAP: The list of attributes whose values should be used as the user ID.",
	)
	flags.StringVar(
		&args.ldapUsernames,
		"username-attributes",
		"uid",
		"LDAP: The list of attributes whose values should be used as the preferred username.",
	)
	flags.StringVar(
		&args.ldapDisplayNames,
		"name-attributes",
		"cn",
		"LDAP: The list of attributes whose values should be used as the display name.",
	)
	flags.StringVar(
		&args.ldapEmails,
		"email-attributes",
		"",
		"LDAP: The list of attributes whose values should be used as the email address.\n",
	)

	// OpenID
	flags.StringVar(
		&args.openidIssuerURL,
		"issuer-url",
		"",
		"OpenID: The URL that the OpenID Provider asserts as the Issuer Identifier. "+
			"It must use the https scheme with no URL query parameters or fragment.",
	)
	flags.StringVar(
		&args.openidEmail,
		"email-claims",
		"",
		"OpenID: List of claims to use as the email address.",
	)
	flags.StringVar(
		&args.openidName,
		"name-claims",
		"",
		"OpenID: List of claims to use as the display name.",
	)
	flags.StringVar(
		&args.openidUsername,
		"username-claims",
		"",
		"OpenID: List of claims to use as the preferred username when provisioning a user.\n",
	)
	flags.StringVar(
		&args.openidExtraScopes,
		"extra-scopes",
		"",
		"OpenID: List of extra scopes to request when provisioning a user.\n",
	)

	// HTPasswd
	flags.StringVar(
		&args.htpasswdUsername,
		"username",
		"",
		"HTPasswd: Username.\n",
	)

	flags.StringVar(
		&args.htpasswdPassword,
		"password",
		"",
		"HTPasswd: Password.\n",
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

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	idps, err := c.GetIdentityProviders(clusterCollection, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get identity providers for cluster '%s': %v", clusterKey, err)
	}

	// Grab all the IDP information interactively if necessary
	idpType := args.idpType

	if idpType == "" {
		prompt := &survey.Select{
			Message: "Type of identity provider:",
			Options: validIdps,
		}
		err = survey.AskOne(prompt, &idpType)
		if err != nil {
			return fmt.Errorf("Failed to get a valid IDP type")
		}
	}

	idpName := args.idpName

	if idpName == "" {
		prompt := &survey.Input{
			Message: "Name of the identity provider:",
		}
		err = survey.AskOne(prompt, &idpName)
		if err != nil {
			return fmt.Errorf("Failed to get a valid IDP name")
		}
	}

	var idpBuilder cmv1.IdentityProviderBuilder
	if idpName == "" {
		idpName = getNextName(idpType, idps)
	}

	message := ""
	switch idpType {
	case "github":
		idpBuilder, err = buildGithubIdp(cluster, idpName)
	case "google":
		idpBuilder, err = buildGoogleIdp(cluster, idpName)
	case "ldap":
		idpBuilder, err = buildLdapIdp(cluster, idpName)
	case "openid":
		idpBuilder, err = buildOpenidIdp(cluster, idpName)
	case "htpasswd":
		idpBuilder, message, err = buildHtpasswdIdp(cluster, idpName)
	default:
		err = fmt.Errorf("Invalid IDP type '%s'", idpType)
	}
	if err != nil {
		return fmt.Errorf("Failed to create IDP for cluster '%s': %v", clusterKey, err)
	}

	fmt.Printf("Configuring IDP for cluster '%s'\n", clusterKey)

	idp, err := idpBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to create IDP for cluster '%s': %v", clusterKey, err)
	}

	_, err = ocm.SendTypedAndHandleDeprecation(clusterCollection.Cluster(cluster.ID()).
		IdentityProviders().
		Add().
		Body(idp))
	if err != nil {
		return fmt.Errorf("Failed to add IDP to cluster '%s': %v", clusterKey, err)
	}

	fmt.Printf(
		"Identity Provider '%s' has been created.\nYou need to ensure that there is a list "+
			"of cluster administrators defined.\nSee 'ocm create user --help' for more "+
			"information.\nTo login into the console, open %s and click on %s.\n%s",
		idpName, cluster.Console().URL(), idpName, message,
	)
	return nil
}

func getNextName(idpType string, idps []*cmv1.IdentityProvider) string {
	nextSuffix := 0
	for _, idp := range idps {
		if strings.Contains(idp.Name(), idpType) {
			lastSuffix, err := strconv.Atoi(strings.Split(idp.Name(), "-")[1])
			if err != nil {
				continue
			}
			if lastSuffix >= nextSuffix {
				nextSuffix = lastSuffix
			}
		}
	}
	return fmt.Sprintf("%s-%d", idpType, nextSuffix+1)
}
