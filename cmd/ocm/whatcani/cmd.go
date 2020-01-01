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

package whatcani

import (
	"fmt"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/spf13/cobra"
	"os"

	"github.com/openshift-online/ocm-cli/pkg/config"
)

var Cmd = &cobra.Command{
	Use:   "whatcani",
	Short: "Prints user roles/permissions information",
	Long:  "Prints user roles/permissions information.",
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("can't load config file: %v", err)
	}
	if cfg == nil {
		return fmt.Errorf("not logged in, run the 'login' command")
	}

	// Check that the configuration has credentials or tokens that don't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		return fmt.Errorf("can't check if tokens have expired: %v", err)
	}
	if !armed {
		return fmt.Errorf("tokens have expired, run the 'login' command")
	}

	// Create the connection:
	connection, err := cfg.Connection()
	if err != nil {
		return fmt.Errorf("can't create connection: %v", err)
	}

	jsonDisplay, err := connection.Get().Path("/api/accounts_mgmt/v1/current_access").Send()
	if err != nil {
		return fmt.Errorf("can't send request: %v", err)
	}

	if jsonDisplay.Status() < 400 {
		err = dump.Pretty(os.Stdout, jsonDisplay.Bytes())
	} else {
		err = dump.Pretty(os.Stderr, jsonDisplay.Bytes())
	}
	if err != nil {
		return fmt.Errorf("can't print body: %v", err)
	}

	return nil
}
