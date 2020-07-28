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

package delete

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/idp"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/urls"
)

var args struct {
	parameter []string
	header    []string
}

var Cmd = &cobra.Command{
	Use:       "delete PATH",
	Short:     "Send a DELETE request",
	Long:      "Send a DELETE request to the given path.",
	RunE:      run,
	ValidArgs: urls.Resources(),
}

func init() {
	fs := Cmd.Flags()
	arguments.AddParameterFlag(fs, &args.parameter)
	arguments.AddHeaderFlag(fs, &args.header)
	Cmd.AddCommand(idp.Cmd)
}

func run(cmd *cobra.Command, argv []string) error {
	path, err := urls.Expand(argv)
	if err != nil {
		return fmt.Errorf("Could not create URI: %v", err)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}
	if cfg == nil {
		return fmt.Errorf("Not logged in, run the 'login' command")
	}

	// Check that the configuration has credentials or tokens that don't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		return fmt.Errorf("Can't check if tokens have expired: %v", err)
	}
	if !armed {
		return fmt.Errorf("Tokens have expired, run the 'login' command")
	}

	// Create the connection:
	connection, err := cfg.Connection()
	if err != nil {
		return fmt.Errorf("Can't create connection: %v", err)
	}

	// Create and populate the request:
	request := connection.Delete()
	err = arguments.ApplyPathArg(request, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't parse path '%s': %v\n", path, err)
		os.Exit(1)
	}
	arguments.ApplyParameterFlag(request, args.parameter)
	arguments.ApplyHeaderFlag(request, args.header)

	// Send the request:
	response, err := request.Send()
	if err != nil {
		return fmt.Errorf("Can't send request: %v", err)
	}
	status := response.Status()
	body := response.Bytes()
	if status < 400 {
		err = dump.Pretty(os.Stdout, body)
	} else {
		err = dump.Pretty(os.Stderr, body)
	}
	if err != nil {
		return fmt.Errorf("Can't print body: %v", err)
	}

	// Save the configuration:
	cfg.AccessToken, cfg.RefreshToken, err = connection.Tokens()
	if err != nil {
		return fmt.Errorf("Can't get tokens: %v", err)
	}
	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("Can't save config file: %v", err)
	}

	// Bye:
	if status >= 400 {
		os.Exit(1)
	}

	return nil
}
