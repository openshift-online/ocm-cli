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

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/dump"
	amv1 "github.com/openshift-online/uhc-sdk-go/pkg/client/accountsmgmt/v1"
)

var args struct {
	json bool
	org  string
}

var Cmd = &cobra.Command{
	Use:   "quota",
	Short: "Retrieve cluster quota information.",
	Long:  "Retrieve cluster quota information of a specific organization.",
	Run:   run,
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

	orgID := args.org

	// Organization to search in case one was not provided:
	if args.org == "" {
		// Get organization of current user:
		userConn, err := connection.AccountsMgmt().V1().CurrentAccount().Get().
			Send()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't retrieve current user information: %v\n", err)
			os.Exit(1)
		}
		userOrg, _ := userConn.Body().GetOrganization()
		orgID = userOrg.ID()
	}

	// Get connection
	orgCollection := connection.AccountsMgmt().V1().Organizations().Organization(orgID)
	orgResponse, err := orgCollection.Get().Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't retrieve organization information: %v\n", err)
		os.Exit(1)
	}
	quotaClient := orgCollection.ResourceQuota()

	// Simple output:
	if !args.json {

		// Request
		quotasListResponse, err := quotaClient.List().
			Send()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve quota: %v\n", err)
			os.Exit(1)
		}

		// Display quota information:
		fmt.Printf("Cluster quota for organization '%s' ID: '%s'\n",
			orgResponse.Body().Name(), orgResponse.Body().ID())
		quotasListResponse.Items().Each(func(quota *amv1.ResourceQuota) bool {
			fmt.Printf("%s-AZ: %d/%d\n", strings.ToUpper(quota.AvailabilityZoneType()),
				quota.Reserved(), quota.Allowed())
			return true
		})

		return

	}

	// TODO: Do this without hard-code; could not find any marshall method
	jsonDisplay, err := connection.Get().Path(
		fmt.Sprintf("/api/accounts_mgmt/v1/organizations/%s/resource_quota", orgID)).
		Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get resource quota: %v\n", err)
		os.Exit(1)
	}
	jsonDisplay.Bytes()
	err = dump.Pretty(os.Stdout, jsonDisplay.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to display quota JSON: %v\n", err)
		os.Exit(1)
	}
}
