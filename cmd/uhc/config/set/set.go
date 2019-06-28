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

package set

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
)

var args struct {
	debug bool
}

var Cmd = &cobra.Command{
	Use:   "set VARIABLE VALUE",
	Short: "Sets the variable's value",
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
	if len(argv) < 2 {
		fmt.Fprintf(os.Stderr, "Expected at least two arguments\n")
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
	value := argv[1]

	switch argv[0] {
	case "access_token":
		cfg.AccessToken = value
	case "client_id":
		cfg.ClientID = value
	case "client_secret":
		cfg.ClientSecret = value
	case "insecure":
		cfg.Insecure, err = strconv.ParseBool(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set insecure: %v", value)
			os.Exit(1)
		}
	case "password":
		cfg.Password = value
	case "refresh_token":
		cfg.RefreshToken = value
	case "scopes":
		fmt.Fprintf(os.Stderr, "Setting scopes is unsupported")
		os.Exit(1)
	case "token_url":
		cfg.TokenURL = value
	case "url":
		cfg.URL = value
	default:
		fmt.Fprintf(os.Stderr, "Unknown setting")
		os.Exit(1)
	}

	err = config.Save(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't save config file: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
