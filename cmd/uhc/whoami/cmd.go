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
	"bytes"
	"fmt"
	"os"

	amsv1 "github.com/openshift-online/uhc-sdk-go/pkg/client/accountsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/dump"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Prints user information",
	Long:  "Prints user information.",
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {
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

	// Send the request:
	response, err := connection.AccountsMgmt().V1().CurrentAccount().Get().
		Send()
	if err != nil {
		return fmt.Errorf("Can't send request: %v", err)
	}

	// Buffer for pretty output:
	buf := new(bytes.Buffer)

	// Output account info.
	err = amsv1.MarshalAccount(response.Body(), buf)
	if err != nil {
		return fmt.Errorf("Failed to marshal account into JSON encoder: %v", err)
	}

	if response.Status() < 400 {
		err = dump.Pretty(os.Stdout, buf.Bytes())
	} else {
		err = dump.Pretty(os.Stderr, buf.Bytes())
	}
	if err != nil {
		return fmt.Errorf("Can't print body: %v", err)
	}

	return nil
}
