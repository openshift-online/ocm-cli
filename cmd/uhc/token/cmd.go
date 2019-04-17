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

package token

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/uhc-sdk/pkg/client"

	"gitlab.cee.redhat.com/service/uhc-cli/pkg/config"
	"gitlab.cee.redhat.com/service/uhc-cli/pkg/dump"
	"gitlab.cee.redhat.com/service/uhc-cli/pkg/util"
)

var args struct {
	debug     bool
	header    bool
	payload   bool
	signature bool
	refresh   bool
}

var Cmd = &cobra.Command{
	Use:   "token",
	Short: "Generates a token",
	Long:  "Uses the stored credentials to generate a token.",
	Run:   run,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.payload,
		"payload",
		false,
		"Print the JSON payload.",
	)
	flags.BoolVar(
		&args.header,
		"header",
		false,
		"Print the JSON header.",
	)
	flags.BoolVar(
		&args.signature,
		"signature",
		false,
		"Print the signature.",
	)
	flags.BoolVar(
		&args.refresh,
		"refresh",
		false,
		"Print the refresh token instead of the access token.",
	)
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	// Check that there there are no command line arguments:
	if len(argv) != 0 {
		fmt.Fprintf(os.Stderr, "Expected zero argument\n")
		os.Exit(1)
	}

	// Check the options:
	count := 0
	if args.header {
		count++
	}
	if args.payload {
		count++
	}
	if args.signature {
		count++
	}
	if count > 1 {
		fmt.Fprintf(
			os.Stderr,
			"Options '--payload', '--header' and '--signature' are mutually exclusive\n",
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
		fmt.Fprintf(os.Stderr, "Not logged in, run the 'login' command\n")
		os.Exit(1)
	}

	// Check that the configuration has credentials or tokens that don't have expired:
	armed, err := config.Armed(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection:
	logger, err := util.NewLogger(args.debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create logger: %v\n", err)
		os.Exit(1)
	}
	connection, err := client.NewConnectionBuilder().
		Logger(logger).
		TokenURL(cfg.TokenURL).
		Client(cfg.ClientID, cfg.ClientSecret).
		Scopes(cfg.Scopes...).
		URL(cfg.URL).
		User(cfg.User, cfg.Password).
		Tokens(cfg.AccessToken, cfg.RefreshToken).
		Insecure(cfg.Insecure).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}

	// Get the tokens:
	accessToken, refreshToken, err := connection.Tokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't get token: %v\n", err)
		os.Exit(1)
	}

	// Select the token according to the options:
	selectedToken := accessToken
	if args.refresh {
		selectedToken = refreshToken
	}

	// Parse the token:
	parser := new(jwt.Parser)
	_, parts, err := parser.ParseUnverified(selectedToken, jwt.MapClaims{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't parse token: %v\n", err)
		os.Exit(1)
	}
	encoding := base64.RawURLEncoding
	header, err := encoding.DecodeString(parts[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't decode header: %v\n", err)
		os.Exit(1)
	}
	payload, err := encoding.DecodeString(parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't decode payload: %v\n", err)
		os.Exit(1)
	}
	signature, err := encoding.DecodeString(parts[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't decode signature: %v\n", err)
		os.Exit(1)
	}

	// Print the data:
	if args.header {
		err = dump.Pretty(os.Stdout, header)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't dump header: %v\n", err)
			os.Exit(1)
		}
	} else if args.payload {
		err = dump.Pretty(os.Stdout, payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't dump payload: %v\n", err)
			os.Exit(1)
		}
	} else if args.signature {
		err = dump.Pretty(os.Stdout, signature)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't dump signature: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s\n", selectedToken)
	}

	// Save the configuration:
	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken
	err = config.Save(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't save config file: %v\n", err)
		os.Exit(1)
	}

	// Bye:
	os.Exit(0)
}
