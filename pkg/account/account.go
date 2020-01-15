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
	"bytes"
	"fmt"

	"github.com/openshift-online/ocm-sdk-go"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

// GetRolesFromUsers gets all roles a specific user possesses.
func GetRolesFromUsers(accounts []*amv1.Account, conn *sdk.Connection) (results map[*amv1.Account][]string, error error) {
	// Prepare the results:
	results = map[*amv1.Account][]string{}

	// Prepare a map of accounts indexed by identifier:
	accountsMap := map[string]*amv1.Account{}
	for _, account := range accounts {
		accountsMap[account.ID()] = account
	}

	// Prepare a query to retrieve all the role bindings that correspond to any of the
	// accounts:
	ids := &bytes.Buffer{}
	for i, account := range accounts {
		if i > 0 {
			ids.WriteString(", ")
		}
		fmt.Fprintf(ids, "'%s'", account.ID())
	}
	query := fmt.Sprintf("account_id in (%s)", ids)

	index := 1
	size := 100

	for {
		// Prepare the request:
		response, err := conn.AccountsMgmt().V1().RoleBindings().List().
			Size(size).
			Page(index).
			Parameter("search", query).
			Send()

		if err != nil {
			return nil, fmt.Errorf("Can't retrieve roles: %v", err)
		}
		// Loop through the results and save them:
		response.Items().Each(func(item *amv1.RoleBinding) bool {
			account := accountsMap[item.Account().ID()]

			itemID := item.Role().ID()

			if _, ok := results[account]; ok {
				if !stringInList(results[account], itemID) {
					results[account] = append(results[account], itemID)
				}
			} else {
				results[account] = append(results[account], itemID)
			}

			return true
		})

		// Break the loop if the page size is smaller than requested, as that indicates
		// that this is the last page:
		if response.Size() < size {
			break
		}
		index++
	}

	return
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
