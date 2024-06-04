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

package users

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	acc_util "github.com/openshift-online/ocm-cli/pkg/account"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var args struct {
	debug bool
	org   string
	roles []string
}

// Cmd configures a new Cobra Command
var Cmd = &cobra.Command{
	Use:   "users",
	Short: "Retrieve users and their roles",
	Long:  "Retrieve information of all users/roles in the same organization",
	Args:  cobra.NoArgs,
	RunE:  run,
}

type userModel struct {
	userName string
	userID   string
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
	flags.StringVar(
		&args.org,
		"org",
		"", // Default value gets assigned later as connection is needed.
		"Organization identifier. Defaults to the organization of the current user.",
	)
	flags.StringSliceVar(
		&args.roles,
		"roles",
		[]string{},
		`Role identifiers. Returns users with one or more of the specified roles.
		Multiple roles can be specified like: --roles="role1,role2,role2".`,
	)
}

func run(cmd *cobra.Command, argv []string) error {

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return err
	}
	defer connection.Close()

	// needed variables:
	pageSize := 100
	pageIndex := 1
	namePad := 40
	searchQuery := ""

	if args.org != "" {
		searchQuery = fmt.Sprintf("organization_id='%s'", args.org)
	}
	// Organization to search in case one was not provided:
	if args.org == "" && len(args.roles) == 0 {
		// Get organization of current user:
		userConn, err := connection.AccountsMgmt().V1().CurrentAccount().Get().
			Send()
		if err != nil {
			return fmt.Errorf("Can't retrieve current user information: %v", err)
		}
		userOrg, ok := userConn.Body().GetOrganization()
		if !ok {
			return fmt.Errorf("Failed to get current user organization")
		}
		args.org = userOrg.ID()

		// Format search request:
		searchQuery = fmt.Sprintf("organization_id='%s'", args.org)
	}

	// Print top.
	fmt.Println(stringPad("USER", namePad), stringPad("USER ID", namePad), "ROLES")

	// Display a list of all users in our organization and their roles:
	for {
		// Get all users within organization
		usersResponse, err := connection.AccountsMgmt().V1().Accounts().List().
			Size(pageSize).
			Page(pageIndex).
			Parameter("search", searchQuery).
			Send()
		if err != nil {
			return fmt.Errorf("Can't retrieve accounts: %v", err)
		}

		accountList := []*amv1.Account{}
		accountMap := make(map[*amv1.Account]*userModel)

		// Go through users found in page and display info:
		usersResponse.Items().Each(func(account *amv1.Account) bool {
			username := stringPad(account.Username(), namePad)
			userID := stringPad(account.ID(), namePad)

			accountList = append(accountList, account)
			accountMap[account] = &userModel{
				userName: username,
				userID:   userID,
			}
			return true
		})

		accountRoleMap, err := acc_util.GetRolesFromUsers(accountList, connection)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get roles for user: %s\n", err)
			os.Exit(1)
		}

		for k, v := range accountRoleMap {
			if len(args.roles) > 0 {
				if checkRoles(v, args.roles) {
					fmt.Println(accountMap[k].userName, accountMap[k].userID, printArray(v))
				}
			} else {
				fmt.Println(accountMap[k].userName, accountMap[k].userID, printArray(v))
			}
		}
		// Resume loop:
		if usersResponse.Size() < pageSize {
			break
		}
		pageIndex++
	}

	return nil
}

func checkRoles(roles, roleArgs []string) bool {
	for _, role := range roles {
		for _, roleArg := range roleArgs {
			if role == roleArg {
				return true
			}
		}
	}
	return false
}

// stringPad will add whitespace or clip a string
// depending on the strings size in comparison to padd variable.
func stringPad(str string, padd int) string {
	// Add padding
	if len(str) < padd {
		str = str + strings.Repeat(" ", padd-len(str))
		// Clip
	} else {
		str = str[:padd-2] + "  "
	}
	return str
}

// printArray turns an array into a string
// sepparated by `space`.
func printArray(arrStr []string) string {
	var finalString string
	for item := range arrStr {
		finalString = fmt.Sprint(arrStr[item], " ", finalString)
	}
	return finalString
}
