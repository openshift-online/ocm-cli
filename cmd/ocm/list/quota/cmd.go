/*
Copyright (c) 2021 Red Hat, Inc.

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
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
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
	Args:  cobra.NoArgs,
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
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()
	orgID := args.org

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

	orgCollection := connection.AccountsMgmt().V1().Organizations().Organization(orgID)
	if err != nil {
		return fmt.Errorf("Can't retrieve organization information: %v", err)
	}

	quotaClient := orgCollection.QuotaCost()

	if !args.json {
		quotasListResponse, err := quotaClient.List().
			Parameter("fetchRelatedResources", true).
			Send()
		if err != nil {
			return fmt.Errorf("Failed to retrieve quota: %v", err)
		}
		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(
			writer,
			"CONSUMED\t\tALLOWED\t\tQUOTA ID\n")

		quotasListResponse.Items().Each(func(quota *amv1.QuotaCost) bool {
			fmt.Fprintf(writer, "%d\t\t%d\t\t%s\n", quota.Consumed(), quota.Allowed(), quota.QuotaID())
			return true
		})

		err = writer.Flush()
		if err != nil {
			return nil
		}

		return nil
	}

	// TODO: Do this without hard-code; could not find any marshall method
	jsonDisplay, err := connection.Get().Path(
		fmt.Sprintf("/api/accounts_mgmt/v1/organizations/%s/resource_quota", orgID)).
		Parameter("fetchRelatedResources", true).
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
