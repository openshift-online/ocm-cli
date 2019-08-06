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

package account

import (
	"fmt"
	"os"

	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	amv1 "github.com/openshift-online/uhc-sdk-go/pkg/client/accountsmgmt/v1"
)

// GetRolesFromUser gets all roles a specific user possesses.
func GetRolesFromUser(account *amv1.Account, conn *client.Connection) []string {

	pageIndex := 1
	var roles []string

	// Get all roles in each role page:
	for {
		rolesList := conn.AccountsMgmt().V1().RoleBindings().List().Page(pageIndex)
		// Format search request:
		searchRequest := ""
		searchRequest = fmt.Sprintf("account_id='%s'", account.ID())
		// Add parameter to search for role with matching user id:
		rolesList.Parameter("search", searchRequest)
		// Get response:
		response, err := rolesList.Send()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't retrieve roles: %v\n", err)
			os.Exit(1)
		}
		// Loop through roles and save their ids
		// iff it is not in the list yet:
		response.Items().Each(func(item *amv1.RoleBinding) bool {
			if !stringInList(roles, item.Role().ID()) {
				roles = append(roles, item.Role().ID())
			}
			return true
		})

		// Break
		if response.Size() < 100 {
			break
		}

		pageIndex++
	}
	return roles
}

// stringInList returns a bool signifying whether
// a string is in a string array.
func stringInList(strArr []string, key string) bool {
	for _, str := range strArr {
		if str == key {
			return true
		}
	}
	return false
}
