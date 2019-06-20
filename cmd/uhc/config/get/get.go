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

	"github.com/openshift-online/uhc-cli/pkg/config"
)

var args struct {
	debug bool
}

var Cmd = &cobra.Command{
	Use:   "get VARIABLE",
	Short: "Prints the config variable",
	Run:   run,
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

func run(cmd *cobra.Command, argv []string) {
	if len(argv) < 1 {
		fmt.Fprintf(os.Stderr, "Expected at least one argument\n")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "Not logged in, run the 'login' command\n")
		os.Exit(1)
	}

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
	case "token":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.Token)
	case "token_url":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.TokenURL)
	case "url":
		fmt.Fprintf(os.Stdout, "%s\n", cfg.URL)
	default:
		fmt.Fprintf(os.Stderr, "Uknown setting\n")
	}

	os.Exit(0)
}
