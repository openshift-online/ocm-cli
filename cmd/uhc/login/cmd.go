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
	"net/url"
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/util"
)

// Preferred OpenID details:
const (
	// #nosec G101
	preferredTokenURL = "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token"
	preferredClientID = "cloud-services"
)

// Deprecated OpenID details used only when trying to authenticate with a user name and a password
// or with a token issued by the deprecated OpenID server:
const (
	// #nosec G101
	deprecatedTokenURL = "https://developers.redhat.com/auth/realms/rhd/protocol/openid-connect/token"
	deprecatedClientID = "uhc"
	deprecatedIssuer   = "sso.redhat.com"
)

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
	debug        bool
	persistent   bool
}

var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in",
	Long:  "Log in, saving the credentials to the configuration file.",
	Run:   run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.tokenURL,
		"token-url",
		"",
		fmt.Sprintf(
			"OpenID token URL. The default value is '%s'. Except when authenticating "+
				"with a user name and password or with a token issued by '%s'. "+
				"In that case the default is '%s'.",
			preferredTokenURL, deprecatedIssuer, deprecatedTokenURL,
		),
	)
	flags.StringVar(
		&args.clientID,
		"client-id",
		"",
		fmt.Sprintf(
			"OpenID client identifier. The default value is '%s'. Except when "+
				"authenticating with a user name and password or with a token "+
				"issued by '%s'. In that case the default is '%s'.",
			preferredClientID, deprecatedIssuer, deprecatedClientID,
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
		client.DefaultScopes,
		"OpenID scope. If this option is used it will replace completely the default "+
			"scopes. Can be repeated multiple times to specify multiple scopes.",
	)
	flags.StringVar(
		&args.url,
		"url",
		client.DefaultURL,
		"URL of the API gateway.",
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
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
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

func run(cmd *cobra.Command, argv []string) {
	// Check mandatory options:
	ok := true
	if args.url == "" {
		fmt.Fprintf(os.Stderr, "Option '--url' is mandatory\n")
		ok = false
	}
	if !ok {
		os.Exit(1)
	}

	// Check that we have some kind of credentials:
	havePassword := args.user != "" && args.password != ""
	haveSecret := args.clientID != "" && args.clientSecret != ""
	haveToken := args.token != ""
	if !havePassword && !haveSecret && !haveToken {
		fmt.Fprintf(
			os.Stderr,
			"In order to log in it is mandatory to use '--token', '--user' and "+
				"'--password', or '--client-id' and '--client-secret'.\n",
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

	// Initially the default OpenID details will be the preferred ones:
	defaultTokenURL := preferredTokenURL
	defaultClientID := preferredClientID

	// If authentication is performed with a user name and password then select the deprecated
	// OpenID details. Otherwise select them according to the issuer of the token.
	if havePassword {
		defaultTokenURL = deprecatedTokenURL
		defaultClientID = deprecatedClientID
	} else if haveToken {
		issuerURL, err := tokenIssuer(args.token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't get token issuer: %v\n", err)
			os.Exit(1)
		}
		if issuerURL != nil && strings.EqualFold(issuerURL.Hostname(), deprecatedIssuer) {
			defaultTokenURL = deprecatedTokenURL
			defaultClientID = deprecatedClientID
		}
	}

	// Apply the default OpenID details if not explicitly provided by the user:
	tokenURL := defaultTokenURL
	if args.tokenURL != "" {
		tokenURL = args.tokenURL
	}
	clientID := defaultClientID
	if args.clientID != "" {
		clientID = args.clientID
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		cfg = new(config.Config)
	}
	cfg.TokenURL = tokenURL
	cfg.ClientID = clientID
	cfg.ClientSecret = args.clientSecret
	cfg.Scopes = args.scopes
	cfg.URL = args.url
	cfg.User = args.user
	cfg.Token = args.token
	cfg.Password = args.password
	cfg.Insecure = args.insecure

	// Create a connection and get the token to verify that the crendentials are correct:
	logger, err := util.NewLogger(args.debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create logger: %v\n", err)
		os.Exit(1)
	}
	builder := client.NewConnectionBuilder().
		Logger(logger).
		TokenURL(cfg.TokenURL).
		Client(cfg.ClientID, cfg.ClientSecret).
		Scopes(cfg.Scopes...).
		URL(cfg.URL).
		User(cfg.User, cfg.Password).
		Insecure(cfg.Insecure)
	if cfg.Token != "" {
		builder.Tokens(cfg.Token)
	}
	connection, err := builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}
	accessToken, refreshToken, err := connection.Tokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't get token: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Can't save config file: %v\n", err)
		os.Exit(1)
	}

	// Bye:
	os.Exit(0)
}

// tokenIssuer parses the given token and tries to extract the value of the `iss` claim. It then
// returns tha value as a URL, or nil if there is no such claim.
func tokenIssuer(token string) (issuer *url.URL, err error) {
	// Parse the token:
	parser := new(jwt.Parser)
	parsed, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return
	}

	// Try to get the `iss` claim:
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("expected map claims but got %T", claims)
		return
	}
	claim, ok := claims["iss"]
	if !ok {
		return
	}
	value, ok := claim.(string)
	if !ok {
		err = fmt.Errorf("expected string 'iss' but got %T", claim)
		return
	}
	issuer, err = url.Parse(value)
	return
}
