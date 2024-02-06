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
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/urls"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/spf13/cobra"
)

const (
	productionURL  = "https://api.openshift.com"
	stagingURL     = "https://api.stage.openshift.com"
	integrationURL = "https://api.integration.openshift.com"
	oauthClientID  = "ocm-cli"
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

var args struct {
	tokenURL      string
	clientID      string
	clientSecret  string
	scopes        []string
	url           string
	token         string
	user          string
	password      string
	rhRegion      string
	insecure      bool
	persistent    bool
	useAuthCode   bool
	useDeviceCode bool
}

var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in",
	Long: "Log in, saving the credentials to the configuration file.\n" +
		"The recommend way is using '--token', which you can obtain at: " +
		urls.OfflineTokenPage,
	Args: cobra.NoArgs,
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
		&args.rhRegion,
		"rh-region",
		"",
		"OCM region identifier. Takes precedence over the --url flag",
	)
	flags.MarkHidden("rh-region")
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
	flags.BoolVar(
		&args.useAuthCode,
		"use-auth-code",
		false,
		"Login using OAuth Authorization Code. This should be used for most cases where a "+
			"browser is available.",
	)
	flags.MarkHidden("use-auth-code")
	flags.BoolVar(
		&args.useDeviceCode,
		"use-device-code",
		false,
		"Login using OAuth Device Code. "+
			"This should only be used for remote hosts and containers where browsers are "+
			"not available. Use auth code for all other scenarios.",
	)
	flags.MarkHidden("use-device-code")
}

func run(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	var err error

	// Check mandatory options:
	if args.url == "" {
		return fmt.Errorf("Option '--url' is mandatory")
	}

	if args.useAuthCode {
		fmt.Println("You will now be redirected to Red Hat SSO login")
		// Short wait for a less jarring experience
		time.Sleep(2 * time.Second)
		token, err := authentication.InitiateAuthCode(oauthClientID)
		if err != nil {
			return fmt.Errorf("an error occurred while retrieving the token : %v", err)
		}
		args.token = token
		args.clientID = oauthClientID
	}

	if args.useDeviceCode {
		deviceAuthConfig := &authentication.DeviceAuthConfig{
			ClientID: oauthClientID,
		}
		_, err = deviceAuthConfig.InitiateDeviceAuth(ctx)
		if err != nil || deviceAuthConfig == nil {
			return fmt.Errorf("an error occurred while initiating device auth: %v", err)
		}
		deviceAuthResp := deviceAuthConfig.DeviceAuthResponse
		fmt.Printf("To login, navigate to %v on another device and enter code %v\n",
			deviceAuthResp.VerificationURI, deviceAuthResp.UserCode)
		fmt.Printf("Checking status every %v seconds...\n", deviceAuthResp.Interval)
		token, err := deviceAuthConfig.PollForTokenExchange(ctx)
		if err != nil {
			return fmt.Errorf("an error occurred while polling for token exchange: %v", err)
		}
		args.token = token
		args.clientID = oauthClientID
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
			urls.OfflineTokenPage,
		)
		os.Exit(1)
	}

	// Inform the user that it isn't recommended to authenticate with user name and password:
	if havePassword {
		fmt.Fprintf(
			os.Stderr,
			"Authenticating with a user name and password is deprecated. To avoid "+
				"this warning go to '%s' to obtain your offline access token "+
				"and then login using the '--token' option.\n",
			urls.OfflineTokenPage,
		)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}
	if cfg == nil {
		cfg = new(config.Config)
	}

	if haveToken {
		// Encrypted tokens are assumed to be refresh tokens:
		if config.IsEncryptedToken(args.token) {
			cfg.AccessToken = ""
			cfg.RefreshToken = args.token
		} else {
			// If a token has been provided parse it:
			token, err := config.ParseToken(args.token)
			if err != nil {
				return fmt.Errorf("Can't parse token '%s': %v", args.token, err)
			}
			// Put the token in the place of the configuration that corresponds to its type:
			typ, err := config.TokenType(token)
			if err != nil {
				return fmt.Errorf("Can't extract type from 'typ' claim of token '%s': %v", args.token, err)
			}
			switch typ {
			case "Bearer", "":
				cfg.AccessToken = args.token
				cfg.RefreshToken = ""
			case "Refresh", "Offline":
				cfg.AccessToken = ""
				cfg.RefreshToken = args.token
			default:
				return fmt.Errorf("Don't know how to handle token type '%s' in token '%s'", typ, args.token)
			}
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

	// Update the configuration with the values given in the command line:
	cfg.TokenURL = tokenURL
	cfg.ClientID = clientID
	cfg.ClientSecret = args.clientSecret
	cfg.Scopes = args.scopes
	cfg.URL = gatewayURL
	cfg.User = args.user
	cfg.Password = args.password
	cfg.Insecure = args.insecure

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

	// If an OCM region is provided, update the config URL with the SDK generated URL
	if args.rhRegion != "" {
		regValue, err := sdk.GetRhRegion(args.url, args.rhRegion)
		if err != nil {
			return fmt.Errorf("Can't find region: %w", err)
		}
		cfg.URL = fmt.Sprintf("https://%s", regValue.URL)
	}

	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("Can't save config file: %v", err)
	}

	if args.useAuthCode || args.useDeviceCode {
		ssoURL, err := url.Parse(cfg.TokenURL)
		if err != nil {
			return fmt.Errorf("can't parse token url '%s': %v", args.tokenURL, err)
		}
		ssoHost := ssoURL.Scheme + "://" + ssoURL.Hostname()

		fmt.Println("Login successful")
		fmt.Printf("To switch accounts, logout from %s and run `ocm logout` "+
			"before attempting to login again", ssoHost)
	}

	return nil
}
