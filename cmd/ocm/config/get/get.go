/*
Copyright (c) 2019 Red Hat, Inc.

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

package get

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-sdk-go/authentication/securestore"
)

var args struct {
	debug bool
}

var Cmd = &cobra.Command{
	Use:   "get [flags] VARIABLE",
	Short: "Prints the value of a config variable",
	Long:  "Prints the value of a config variable. See 'ocm config --help' for supported config variables.",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}

	// If the configuration file doesn't exist yet assume that all the configuration settings
	// are empty:
	if cfg == nil {
		fmt.Printf("\n")
		return nil
	}

	// Print the value of the requested configuration setting:
	switch argv[0] {
	case "access_token":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.AccessToken)
	case "client_id":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.ClientID)
	case "client_secret":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.ClientSecret)
	case "insecure":
		fmt.Fprintf(os.Stdout, "%v\n", cfg.Insecure)
	case "password":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.Password)
	case "refresh_token":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.RefreshToken)
	case "scopes":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.Scopes)
	case "token_url":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.TokenURL)
	case "url":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.URL)
	case "pager":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.Pager)
	case "keyrings":
		fmt.Fprintf(os.Stdout, "%s\n", getKeyrings())
	default:
		return fmt.Errorf("Unknown setting")
	}

	return nil
}

func getKeyrings() []string {
	backends := securestore.AvailableBackends()
	if len(backends) == 0 {
		fmt.Printf("No keyrings available: %s\n", securestore.ErrNoBackendsAvailable)
	}
	return backends
}
