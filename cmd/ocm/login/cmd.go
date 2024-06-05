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
	"strings"
	"time"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/properties"
	"github.com/openshift-online/ocm-cli/pkg/urls"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/openshift-online/ocm-sdk-go/authentication/securestore"
	"github.com/spf13/cobra"
)

const (
	oauthClientID = "ocm-cli"
)

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
		"",
		"URL of the OCM API gateway. If not provided, will reuse the URL from the configuration "+
			"file or "+sdk.DefaultURL+" as a last resort. The value should be a complete URL "+
			"or a valid URL alias: "+strings.Join(urls.ValidOCMUrlAliases(), ", "),
	)
	flags.StringVar(
		&args.rhRegion,
		"rh-region",
		"",
		"OCM data sovereignty region identifier. --url will be used to initiate a service discovery "+
			"request to find the region URL matching the provided identifier. Use `ocm list rh-regions` "+
			"to see available regions.",
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
			"browser is available. See --use-device-code for remote hosts and containers.",
	)
	flags.BoolVar(
		&args.useDeviceCode,
		"use-device-code",
		false,
		"Login using OAuth Device Code. "+
			"This should only be used for remote hosts and containers where browsers are "+
			"not available. See --use-auth-code for all other scenarios.",
	)
}

var (
	InitiateAuthCode = authentication.InitiateAuthCode
)

func run(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	var err error

	// Fail fast if OCM_KEYRING is provided and invalid
	if keyring, ok := config.IsKeyringManaged(); ok {
		err := securestore.ValidateBackend(keyring)
		if err != nil {
			return err
		}
	}

	if args.useAuthCode {
		fmt.Println("You will now be redirected to Red Hat SSO login")
		// Short wait for a less jarring experience
		time.Sleep(2 * time.Second)
		token, err := InitiateAuthCode(oauthClientID)
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
		if err != nil {
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
	haveClientCreds := args.clientID != "" && args.clientSecret != ""
	haveToken := args.token != ""
	if !havePassword && !haveClientCreds && !haveToken {
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

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config: %v", err)
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

	gatewayURL, err := urls.ResolveGatewayURL(args.url, cfg)
	if err != nil {
		return err
	}

	// If an --rh-region is provided, --url is resolved as above and then used to initiate
	// service discovery for the environment --url is a part of, but the gatewayURL (and
	// ultimately the cfg.URL) is then updated to the URL of the matching --rh-region:
	//   1. resolve the gatewayURL as above
	//   2. fetch a well-known file from sdk.GetRhRegion
	//   3. update the gatewayURL to the region URL matching args.rhRegion
	//
	// So `--url=https://api.stage.openshift.com --rh-region=singapore` might result in
	// gatewayURL/cfg.URL being mutated to "https://api.singapore.stage.openshift.com"
	//
	// See ocm-sdk-go/rh_region.go for full details on how service discovery works.
	if args.rhRegion != "" {
		regValue, err := sdk.GetRhRegion(gatewayURL, args.rhRegion)
		if err != nil {
			return fmt.Errorf("Can't find region: %w", err)
		}
		gatewayURL = fmt.Sprintf("https://%s", regValue.URL)
	}

	if overrideUrl := os.Getenv(properties.URLEnvKey); overrideUrl != "" {
		fmt.Fprintf(os.Stderr, "WARNING: the `%s` environment variable is set, but is not used for the login command. The `ocm login` command will only use the explicitly set flag's url, which is set as %s\n", properties.URLEnvKey, gatewayURL)
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

	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("can't save config: %v", err)
	}

	if args.useAuthCode || args.useDeviceCode {
		ssoURL, err := url.Parse(cfg.TokenURL)
		if err != nil {
			return fmt.Errorf("can't parse token url '%s': %v", args.tokenURL, err)
		}
		ssoHost := ssoURL.Scheme + "://" + ssoURL.Hostname()

		fmt.Println("Login successful")
		fmt.Printf("To switch accounts, logout from %s and run `ocm logout` "+
			"before attempting to login again\n", ssoHost)
	}

	return nil
}
