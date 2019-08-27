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

package roles

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var args struct {
	debug bool
}

var Cmd = &cobra.Command{
	Use:   "roles [role-name]",
	Short: "Retrieve information of the different roles",
	Long:  "Get description of a role or list of all roles ",
	RunE:  run,
}

func init() {
	// Add flags to rootCmd:
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
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

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		return fmt.Errorf("Can't check if tokens have expired: %v", err)
	}
	if !armed {
		return fmt.Errorf("Tokens have expired, run the 'login' command")
	}

	// Create the connection, and remember to close it:
	connection, err := cfg.Connection()
	if err != nil {
		return fmt.Errorf("Can't create connection: %v", err)
	}
	defer connection.Close()

	// No role name was provided; Print all roles.
	var rolesList []string
	if len(argv) < 1 {
		pageIndex := 1
		for {
			rolesListRequest := connection.AccountsMgmt().V1().Roles().List().Page(pageIndex)
			response, err := rolesListRequest.Send()
			if err != nil {
				return fmt.Errorf("Can't send request: %v", err)
			}
			response.Items().Each(func(item *amv1.Role) bool {
				rolesList = append(rolesList, item.ID())
				return true
			})
			pageIndex++

			// Break on last page
			if response.Size() < 100 {
				break
			}

		}

		// Print each role:
		for _, element := range rolesList {
			fmt.Println(element)
		}

	} else {

		// Get role with provided id response:
		roleResponse, err := connection.AccountsMgmt().V1().Roles().Role(argv[0]).Get().
			Send()
		if err != nil {
			return fmt.Errorf("Can't send request: %v", err)
		}
		role := roleResponse.Body()

		// Use role in new get request
		byteRole, err := connection.Get().Path(role.HREF()).
			Send()
		if err != nil {
			return fmt.Errorf("Can't send request: %v", err)
		}

		// Dump pretty:
		err = dump.Pretty(os.Stdout, byteRole.Bytes())
		if err != nil {
			return fmt.Errorf("Failed to display role JSON: %v", err)
		}

	}

	return nil
}
