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

package quota

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var args struct {
	json bool
	org  string
}

var Cmd = &cobra.Command{
	Use:   "quota",
	Short: "Retrieve cluster quota information.",
	Long:  "Retrieve cluster quota information of a specific organization.",
	RunE:  run,
}

func init() {
	// Add flags to rootCmd:
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.json,
		"json",
		false,
		"Returns a list of resource quota objects in JSON.",
	)
	flags.StringVar(
		&args.org,
		"org",
		"",
		"Specify which organization to query information from. Default to local users organization.",
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

	orgID := args.org

	// Organization to search in case one was not provided:
	if args.org == "" {
		// Get organization of current user:
		userConn, err := connection.AccountsMgmt().V1().CurrentAccount().Get().
			Send()
		if err != nil {
			return fmt.Errorf("Can't retrieve current user information: %v", err)
		}
		userOrg, _ := userConn.Body().GetOrganization()
		orgID = userOrg.ID()
	}

	// Get connection
	orgCollection := connection.AccountsMgmt().V1().Organizations().Organization(orgID)
	orgResponse, err := orgCollection.Get().Send()
	if err != nil {
		return fmt.Errorf("Can't retrieve organization information: %v", err)
	}
	quotaClient := orgCollection.QuotaSummary()

	// Simple output:
	if !args.json {

		// Request
		quotasListResponse, err := quotaClient.List().
			Send()
		if err != nil {
			return fmt.Errorf("Failed to retrieve quota: %v", err)
		}

		// Display quota information:
		fmt.Printf("Cluster quota for organization '%s' ID: '%s'\n",
			orgResponse.Body().Name(), orgResponse.Body().ID())
		quotasListResponse.Items().Each(func(quota *amv1.QuotaSummary) bool {
			var byoc string
			if quota.BYOC() {
				byoc = " BYOC"
			}
			fmt.Printf("%s-AZ%s: %d/%d\n", strings.ToUpper(quota.AvailabilityZoneType()),
				byoc, quota.Reserved(), quota.Allowed())
			return true
		})

		return nil

	}

	// TODO: Do this without hard-code; could not find any marshall method
	jsonDisplay, err := connection.Get().Path(
		fmt.Sprintf("/api/accounts_mgmt/v1/organizations/%s/resource_quota", orgID)).
		Send()
	if err != nil {
		return fmt.Errorf("Failed to get resource quota: %v", err)
	}
	jsonDisplay.Bytes()
	err = dump.Pretty(os.Stdout, jsonDisplay.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to display quota JSON: %v", err)
	}

	return nil
}
