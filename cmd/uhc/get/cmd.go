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

package get

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/dump"
	"github.com/openshift-online/uhc-cli/pkg/flags"
	"github.com/openshift-online/uhc-cli/pkg/urls"
)

var args struct {
	parameter []string
	header    []string
	single    bool
}

var Cmd = &cobra.Command{
	Use:   "get RESOURCE {ID}",
	Short: "Send a GET request",
	Long:  "Send a GET request to the given path.",
	Run:   run,
}

func init() {
	fs := Cmd.Flags()
	flags.AddParameterFlag(fs, &args.parameter)
	flags.AddHeaderFlag(fs, &args.header)
	fs.BoolVar(
		&args.single,
		"single",
		false,
		"Return the output as a single line.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	path, err := urls.Expand(argv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create URI: %v\n", err)
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
	armed, err := cfg.Armed()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection:
	connection, err := cfg.Connection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}

	// Create and populate the request:
	request := connection.Get().Path(path)
	flags.ApplyParameterFlag(request, args.parameter)
	flags.ApplyHeaderFlag(request, args.header)

	// Send the request:
	response, err := request.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't send request: %v\n", err)
		os.Exit(1)
	}
	status := response.Status()
	body := response.Bytes()
	if status < 400 {
		if args.single {
			err = dump.Simple(os.Stdout, body)
		} else {
			err = dump.Pretty(os.Stdout, body)
		}
	} else {
		if args.single {
			err = dump.Simple(os.Stderr, body)
		} else {
			err = dump.Pretty(os.Stderr, body)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't print body: %v\n", err)
		os.Exit(1)
	}

	// Save the configuration:
	cfg.AccessToken, cfg.RefreshToken, err = connection.Tokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't get tokens: %v\n", err)
		os.Exit(1)
	}
	err = config.Save(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't save config file: %v\n", err)
		os.Exit(1)
	}

	// Bye:
	if status < 400 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
