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
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	header    bool
	payload   bool
	signature bool
	refresh   bool
	generate  bool
}

var Cmd = &cobra.Command{
	Use:   "token",
	Short: "Generates a token",
	Long:  "Uses the stored credentials to generate a token.",
	Args:  cobra.NoArgs,
	RunE:  run,
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
		&args.generate,
		"generate",
		false,
		"Generate a new token.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
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
	if args.generate {
		count++
	}

	if count > 1 {
		return fmt.Errorf("Options '--payload', '--header', '--signature', and '--generate' are mutually exclusive")
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}

	var accessToken string
	var refreshToken string

	// For opaque tokens, the SDK's unauthenticated connection can't manage
	// tokens via connection.Tokens(), so read them from config directly.
	haveOpaqueToken := cfg != nil && config.IsOpaqueToken(cfg.AccessToken)

	if haveOpaqueToken {
		if args.generate {
			return fmt.Errorf("The '--generate' option is not supported with opaque tokens")
		}
		accessToken = cfg.AccessToken
		refreshToken = cfg.RefreshToken
	} else {
		// Create the client for the OCM API:
		connection, err := ocm.NewConnection().Build()
		if err != nil {
			return fmt.Errorf("Failed to create OCM connection: %v", err)
		}
		defer connection.Close()

		if args.generate {
			// Get new tokens:
			accessToken, refreshToken, err = connection.Tokens(15 * time.Minute)
			if err != nil {
				return fmt.Errorf("Can't get new tokens: %v", err)
			}
		} else {
			// Get the tokens:
			accessToken, refreshToken, err = connection.Tokens()
			if err != nil {
				return fmt.Errorf("Can't get token: %v", err)
			}
		}

		// Save the potentially refreshed tokens:
		cfg.AccessToken = accessToken
		cfg.RefreshToken = refreshToken
		err = config.Save(cfg)
		if err != nil {
			return fmt.Errorf("Can't save config file: %v", err)
		}
	}

	// Select the token according to the options:
	selectedToken := accessToken
	if args.refresh {
		selectedToken = refreshToken
	}

	if !config.IsJWTToken(selectedToken) {
		if args.header || args.payload || args.signature {
			return fmt.Errorf(
				"The token is not a JWT, so '--header', '--payload', and '--signature' options are not available",
			)
		}
		fmt.Fprintf(os.Stdout, "%s\n", selectedToken)
	} else {
		// Parse the token:
		parser := new(jwt.Parser)
		_, parts, err := parser.ParseUnverified(selectedToken, jwt.MapClaims{})
		if err != nil {
			return fmt.Errorf("Can't parse token: %v", err)
		}
		encoding := base64.RawURLEncoding
		header, err := encoding.DecodeString(parts[0])
		if err != nil {
			return fmt.Errorf("Can't decode header: %v", err)
		}
		payload, err := encoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("Can't decode payload: %v", err)
		}
		signature, err := encoding.DecodeString(parts[2])
		if err != nil {
			return fmt.Errorf("Can't decode signature: %v", err)
		}

		// Print the data:
		if args.header {
			err = dump.Pretty(os.Stdout, header)
			if err != nil {
				return fmt.Errorf("Can't dump header: %v", err)
			}
		} else if args.payload {
			err = dump.Pretty(os.Stdout, payload)
			if err != nil {
				return fmt.Errorf("Can't dump payload: %v", err)
			}
		} else if args.signature {
			err = dump.Pretty(os.Stdout, signature)
			if err != nil {
				return fmt.Errorf("Can't dump signature: %v", err)
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", selectedToken)
		}
	}

	// Bye:
	return nil
}
