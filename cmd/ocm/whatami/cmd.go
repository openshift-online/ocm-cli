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

package whatami

import (
	"encoding/json"
	"fmt"
	"github.com/openshift-online/ocm-cli/pkg/config"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "whatami",
	Short: "Prints user roles information",
	Long:  "Prints user roles information.",
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
	listResponse, err := connection.AccountsMgmt().V1().CurrentAccess().List().Send()
	if err != nil {
		return fmt.Errorf("Can't send request: %v", err)
	}

	type Role struct {
		Kind string
		Name string
	}

	whatAmI := struct {
		Kind  string
		Page  int
		Size  int
		Total int
		Names []Role
	}{
		Kind:  "Roles",
		Page:  listResponse.Page(),
		Size:  listResponse.Size(),
		Total: listResponse.Total(),
		Names: make([]Role, 0, listResponse.Size()),
	}

	listResponse.Items().Each(func(role *amsv1.Role) bool {
		r := Role{
			Kind: "Role",
			Name: role.Name(),
		}
		whatAmI.Names = append(whatAmI.Names, r)

		return true
	})

	bytes, _ := json.MarshalIndent(whatAmI, "", "  ")
	fmt.Printf("%s\n", bytes)

	return nil
}
