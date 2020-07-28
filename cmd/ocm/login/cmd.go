/*
Copyright (c) 2018 Red Hat, Inc.

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

package login

import (
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
)

const (
	productionURL  = "https://api.openshift.com"
	stagingURL     = "https://api.stage.openshift.com"
	integrationURL = "https://api-integration.6943.hive-integration.openshiftapps.com"
)

// When the value of the `--url` option is one of the keys of this map it will be replaced by the
// corresponding value.
var urlAliases = map[string]string{
	"production":  productionURL,
	"prod":        productionURL,
	"prd":         productionURL,
	"staging":     stagingURL,
	"stage":       stagingURL,
	"stg":         stagingURL,
	"integration": integrationURL,
	"int":         integrationURL,
}

// #nosec G101
const uiTokenPage = "https://cloud.redhat.com/openshift/token"

var args struct {
	tokenURL     string
	clientID     string
	clientSecret string
	scopes       []string
	url          string
	token        string
	user         string
	password     string
	insecure     bool
	persistent   bool
}

var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in",
	Long: "Log in, saving the credentials to the configuration file.\n" +
		"The recommend way is using '--token', which you can obtain at: " + uiTokenPage,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.tokenURL,
		"token-url",
		"",
		fmt.Sprintf(
			"OpenID token URL. The default value is '%s'.",
			sdk.DefaultTokenURL,
		),
	)
	flags.StringVar(
		&args.clientID,
		"client-id",
		"",
		fmt.Sprintf(
			"OpenID client identifier. The default value is '%s'.",
			sdk.DefaultClientID,
		),
	)
	flags.StringVar(
		&args.clientSecret,
		"client-secret",
		"",
		"OpenID client secret.",
	)
	flags.StringSliceVar(
		&args.scopes,
		"scope",
		sdk.DefaultScopes,
		"OpenID scope. If this option is used it will replace completely the default "+
			"scopes. Can be repeated multiple times to specify multiple scopes.",
	)
	flags.StringVar(
		&args.url,
		"url",
		sdk.DefaultURL,
		"URL of the API gateway. The value can be the complete URL or an alias. The "+
			"valid aliases are 'production', 'staging', 'integration' and their shorthands.",
	)
	flags.StringVar(
		&args.token,
		"token",
		"",
		"Access or refresh token.",
	)
	flags.StringVar(
		&args.user,
		"user",
		"",
		"User name.",
	)
	flags.StringVar(
		&args.password,
		"password",
		"",
		"User password.",
	)
	flags.BoolVar(
		&args.insecure,
		"insecure",
		false,
		"Enables insecure communication with the server. This disables verification of TLS "+
			"certificates and host names.",
	)
	flags.BoolVar(
		&args.persistent,
		"persistent",
		false,
		"By default the tool doesn't persistently store the user name and password, so "+
			"when the refresh token expires the user will have to log in again. If "+
			"this option is provided then the user name and password will be stored "+
			"persistently, in clear text, which is potentially unsafe.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	var err error

	// Check mandatory options:
	if args.url == "" {
		return fmt.Errorf("Option '--url' is mandatory")
	}

	// Check that we have some kind of credentials:
	havePassword := args.user != "" && args.password != ""
	haveSecret := args.clientID != "" && args.clientSecret != ""
	haveToken := args.token != ""
	if !havePassword && !haveSecret && !haveToken {
		// Allow bare `ocm login` to suggest the token page without noise of full help.
		fmt.Fprintf(
			os.Stderr,
			"In order to log in it is mandatory to use '--token', '--user' and "+
				"'--password', or '--client-id' and '--client-secret'.\n"+
				"You can obtain a token at: %s .\n"+
				"See 'ocm login --help' for full help.\n",
			uiTokenPage,
		)
		os.Exit(1)
	}

	// Inform the user that it isn't recommended to authenticate with user name and password:
	if havePassword {
		fmt.Fprintf(
			os.Stderr,
			"Authenticating with a user name and password is deprecated. To avoid "+
				"this warning go to 'https://cloud.redhat.com/openshift/token' "+
				"to obtain your offline access token and then login using the "+
				"'--token' option.\n",
		)
	}

	// If a token has been provided parse it:
	var token *jwt.Token
	if haveToken {
		parser := new(jwt.Parser)
		token, _, err = parser.ParseUnverified(args.token, jwt.MapClaims{})
		if err != nil {
			return fmt.Errorf("Can't parse token '%s': %v", args.token, err)
		}
	}

	// Apply the default OpenID details if not explicitly provided by the user:
	tokenURL := sdk.DefaultTokenURL
	if args.tokenURL != "" {
		tokenURL = args.tokenURL
	}
	clientID := sdk.DefaultClientID
	if args.clientID != "" {
		clientID = args.clientID
	}

	// If the value of the `--url` is any of the aliases then replace it with the corresponding
	// real URL:
	gatewayURL, ok := urlAliases[args.url]
	if !ok {
		gatewayURL = args.url
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}
	if cfg == nil {
		cfg = new(config.Config)
	}

	// Update the configuration with the values given in the command line:
	cfg.TokenURL = tokenURL
	cfg.ClientID = clientID
	cfg.ClientSecret = args.clientSecret
	cfg.Scopes = args.scopes
	cfg.URL = gatewayURL
	cfg.User = args.user
	cfg.Password = args.password
	cfg.Insecure = args.insecure
	cfg.AccessToken = ""
	cfg.RefreshToken = ""

	// Put the token in the place of the configuration that corresponds to its type:
	if haveToken {
		typ, err := tokenType(token)
		if err != nil {
			return fmt.Errorf("Can't extract type from 'typ' claim of token '%s': %v", args.token, err)
		}
		switch typ {
		case "Bearer":
			cfg.AccessToken = args.token
		case "Refresh", "Offline":
			cfg.RefreshToken = args.token
		case "":
			return fmt.Errorf("Don't know how to handle empty type in token '%s'", args.token)
		default:
			return fmt.Errorf("Don't know how to handle token type '%s' in token '%s'", typ, args.token)
		}
	}

	// Create a connection and get the token to verify that the crendentials are correct:
	connection, err := cfg.Connection()
	if err != nil {
		return fmt.Errorf("Can't create connection: %v", err)
	}
	accessToken, refreshToken, err := connection.Tokens()
	if err != nil {
		return fmt.Errorf("Can't get token: %v", err)
	}

	// Save the configuration, but clear the user name and password before unless we have
	// explicitly been asked to store them persistently:
	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken
	if !args.persistent {
		cfg.User = ""
		cfg.Password = ""
	}
	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("Can't save config file: %v", err)
	}

	return nil
}

// tokenType extracts the value of the `typ` claim. It returns the value as a string, or the empty
// string if there is no such claim.
func tokenType(token *jwt.Token) (typ string, err error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("expected map claims but got %T", claims)
		return
	}
	claim, ok := claims["typ"]
	if !ok {
		return
	}
	value, ok := claim.(string)
	if !ok {
		err = fmt.Errorf("expected string 'typ' but got %T", claim)
		return
	}
	typ = value
	return
}
