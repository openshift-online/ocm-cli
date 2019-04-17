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

	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/uhc-sdk/pkg/client"

	"gitlab.cee.redhat.com/service/uhc-cli/pkg/config"
	"gitlab.cee.redhat.com/service/uhc-cli/pkg/util"
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
		client.DefaultTokenURL,
		"OpenID token URL.",
	)
	flags.StringVar(
		&args.clientID,
		"client-id",
		client.DefaultClientID,
		"OpenID client identifier.",
	)
	flags.StringVar(
		&args.clientSecret,
		"client-secret",
		client.DefaultClientSecret,
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

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		cfg = new(config.Config)
	}
	cfg.TokenURL = args.tokenURL
	cfg.ClientID = args.clientID
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
