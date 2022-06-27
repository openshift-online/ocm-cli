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
	"errors"
	"fmt"
	"net/url"
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/AlecAivazis/survey/v2"
)

func buildOpenidIdp(cluster *cmv1.Cluster, idpName string) (idpBuilder cmv1.IdentityProviderBuilder, err error) {
	clientID := args.clientID
	clientSecret := args.clientSecret
	issuerURL := args.openidIssuerURL
	email := args.openidEmail
	name := args.openidName
	username := args.openidUsername
	extraScopes := args.openidExtraScopes

	isInteractive := clientID == "" || clientSecret == "" || issuerURL == "" ||
		(email == "" && name == "" && username == "")

	if isInteractive {
		fmt.Println("To use OpenID as an identity provider, you must first register the application:")
		instructionsURL := "https://docs.openshift.com/dedicated/osd_install_access_delete_cluster/" +
			"config-identity-providers.html#config-openid-idp_config-identity-providers"
		fmt.Println("* Open the following URL:", instructionsURL)
		fmt.Println("* Follow the instructions to register your application")

		oauthURL := c.GetClusterOauthURL(cluster)

		fmt.Println("* When creating the OpenID, use the following URL for the Authorized redirect URI: ",
			oauthURL+"/oauth2callback/"+idpName)

		if clientID == "" {
			prompt := &survey.Input{
				Message: "Copy the Client ID provided by the OpenID Provider:",
			}
			err = survey.AskOne(prompt, &clientID)
			if err != nil {
				return idpBuilder, errors.New("Expected a valid application Client ID")
			}
		}

		if clientSecret == "" {
			prompt := &survey.Input{
				Message: "Copy the Client Secret provided by the OpenID Provider:",
			}
			err = survey.AskOne(prompt, &clientSecret)
			if err != nil {
				return idpBuilder, errors.New("Expected a valid application Client Secret")
			}
		}

		if issuerURL == "" {
			prompt := &survey.Input{
				Message: "URL that the OpenID Provider asserts as the Issuer Identifier:",
			}
			err = survey.AskOne(prompt, &issuerURL)
			if err != nil {
				return idpBuilder, errors.New("Expected a valid OpenID Issuer URL")
			}
		}

		if email == "" {
			prompt := &survey.Input{
				Message: "Claim mappings to use as the email address:",
			}
			err = survey.AskOne(prompt, &email)
			if err != nil {
				return idpBuilder, errors.New("Expected a list of claims to use as the email address")
			}
		}

		if name == "" {
			prompt := &survey.Input{
				Message: "Claim mappings to use as the display name:",
			}
			err = survey.AskOne(prompt, &name)
			if err != nil {
				return idpBuilder, errors.New("Expected a list of claims to use as the display name")
			}
		}

		if username == "" {
			prompt := &survey.Input{
				Message: "Claim mappings to use as the preferred username:",
			}
			err = survey.AskOne(prompt, &username)
			if err != nil {
				return idpBuilder, errors.New("Expected a list of claims to use as the preferred username")
			}
		}

		if extraScopes == "" {
			prompt := &survey.Input{
				Message: "Extra scopes to request:",
			}
			err = survey.AskOne(prompt, &extraScopes)
			if err != nil {
				return idpBuilder, errors.New("Expected a list of extra scopes to request")
			}
		}
	}

	if email == "" && name == "" && username == "" {
		return idpBuilder, errors.New("At least one claim is required: [email-claims name-claims username-claims]")
	}

	parsedIssuerURL, err := url.ParseRequestURI(issuerURL)
	if err != nil {
		return idpBuilder, fmt.Errorf("Expected a valid OpenID issuer URL: %v", err)
	}
	if parsedIssuerURL.Scheme != "https" {
		return idpBuilder, errors.New("Expected OpenID issuer URL to use an https:// scheme")
	}
	if parsedIssuerURL.RawQuery != "" {
		return idpBuilder, errors.New("OpenID issuer URL must not have query parameters")
	}
	if parsedIssuerURL.Fragment != "" {
		return idpBuilder, errors.New("OpenID issuer URL must not have a fragment")
	}

	// Build OpenID Claims
	openIDClaims := cmv1.NewOpenIDClaims()
	if email != "" {
		openIDClaims = openIDClaims.Email(strings.Split(email, ",")...)
	}
	if name != "" {
		openIDClaims = openIDClaims.Name(strings.Split(name, ",")...)
	}
	if username != "" {
		openIDClaims = openIDClaims.PreferredUsername(strings.Split(username, ",")...)
	}

	// Create OpenID IDP
	openIDIDP := cmv1.NewOpenIDIdentityProvider().
		ClientID(clientID).
		ClientSecret(clientSecret).
		Issuer(issuerURL).
		Claims(openIDClaims).
		ExtraScopes(extraScopes)

	// Create new IDP with OpenID provider
	idpBuilder.
		Type("OpenIDIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(args.mappingMethod)).
		OpenID(openIDIDP)

	return
}
