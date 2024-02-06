/*
Copyright (c) 2024 Red Hat, Inc.

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

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-sdk-go/authentication/securestore"
)

var args struct {
	debug bool
}

var Cmd = &cobra.Command{
	Use:    "delete",
	Short:  "Deletes the existing configuration from the OS keyring",
	Long:   "Deletes the existing configuration from the OS keyring",
	Args:   cobra.ExactArgs(0),
	RunE:   run,
	Hidden: true,
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
	err := securestore.RemoveConfigFromKeyring()
	if err != nil {
		return fmt.Errorf("can't delete config from keyring: %v", err)
	}
	return nil
}
