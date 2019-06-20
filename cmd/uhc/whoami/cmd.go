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

package whoami

import (
	"fmt"
	"os"

	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/dump"
	"github.com/openshift-online/uhc-cli/pkg/util"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Prints user information",
	Long:  "Prints user information.",
	Run:   run,
}

var debug bool

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&debug,
		"debug",
		false,
		"Enable debug mode.",
	)
}

func run(cmd *cobra.Command, argv []string) {
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
	logger, err := util.NewLogger(debug)
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

	request := connection.Get().Path("/api/accounts_mgmt/v1/current_account")

	// Send the request:
	response, err := request.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't send request: %v\n", err)
		os.Exit(1)
	}
	status := response.Status()
	body := response.Bytes()
	if status < 400 {
		err = dump.Pretty(os.Stdout, body)
	} else {
		err = dump.Pretty(os.Stderr, body)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't print body: %v\n", err)
		os.Exit(1)
	}
}
