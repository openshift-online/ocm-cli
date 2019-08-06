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

	acc_util "github.com/openshift-online/uhc-cli/pkg/account"
	"github.com/openshift-online/uhc-cli/pkg/config"
	amv1 "github.com/openshift-online/uhc-sdk-go/pkg/client/accountsmgmt/v1"
)

var args struct {
	debug bool
	org   string
}

var Cmd = &cobra.Command{
	Use:   "users",
	Short: "Retrieve users and their roles",
	Long:  "Retrieve information of all users/roles in the same organization",
	Run:   run,
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

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection, and remember to close it:
	connection, err := cfg.Connection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}
	defer connection.Close()

	// needed variables:
	pageSize := 100
	pageIndex := 1
	namePad := 40

	// Organization to search in case one was not provided:
	if args.org == "" {
		// Get organization of current user:
		userConn, err := connection.AccountsMgmt().V1().CurrentAccount().Get().
			Send()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't retrieve current user information: %v\n", err)
			os.Exit(1)
		}
		userOrg, ok := userConn.Body().GetOrganization()
		if !ok {
			fmt.Println("Failed to get current user organization")
			os.Exit(1)
		}
		args.org = userOrg.ID()
	}

	// Print top.
	fmt.Println(stringPad("USER", namePad), stringPad("USER ID", namePad), "ROLES")
	fmt.Println()

	// Display a list of all users in our organization and their roles:
	for {

		// Format search request:
		searchQuery := fmt.Sprintf("organization_id='%s'", args.org)

		// Get all users within organization
		usersResponse, err := connection.AccountsMgmt().V1().Accounts().List().
			Size(pageSize).
			Page(pageIndex).
			Parameter("search", searchQuery).
			Send()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't retrieve accounts: %v\n", err)
			os.Exit(1)
		}
		// Go through users found in page and display info:
		usersResponse.Items().Each(func(account *amv1.Account) bool {
			if args.org == account.Organization().ID() {
				username := stringPad(account.Username(), namePad)
				userID := stringPad(account.ID(), namePad)
				accountRoleList := acc_util.GetRolesFromUser(account, connection)
				fmt.Println(username, userID, printArray(accountRoleList))
			}
			return true
		})

		// Resume loop:
		if usersResponse.Size() < pageSize {
			break
		}
		pageIndex++
	}

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
