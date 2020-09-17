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
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/ingress"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/user"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
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
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(user.Cmd)
}

func run(cmd *cobra.Command, argv []string) error {
	path, err := urls.Expand(argv)
	if err != nil {
		return fmt.Errorf("Could not create URI: %v", err)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

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

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}
	if cfg == nil {
		return fmt.Errorf("Not logged in, run the 'login' command")
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
