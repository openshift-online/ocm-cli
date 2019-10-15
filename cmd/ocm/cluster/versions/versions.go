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

package versions

import (
	"fmt"
	"os"
	"strings"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
)

var Cmd = &cobra.Command{
	Use:   "versions",
	Short: "List available versions",
	Long:  "List the versions available for provisioning a cluster",
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {

	if len(argv) != 0 {
		return fmt.Errorf("Expected no arguments")
	}

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

	// Get the client for the resource that manages the versions:
	resource := connection.ClustersMgmt().V1().Versions()

	size := 100
	index := 1
	for {
		// Fetch the next page:
		request := resource.List().Size(size).Page(index)
		//flags.ApplyHeaderFlag(request, args.header)
		var search strings.Builder
		request.Search(strings.TrimSpace(search.String()))
		response, err := request.Send()
		if err != nil {
			return fmt.Errorf("Can't retrieve versions: %v", err)
		}

		response.Items().Each(func(version *v1.Version) bool {
			// strip leading "openshift-v" string
			v := strings.Replace(version.ID(), "openshift-v", "", 1)
			fmt.Fprintf(os.Stdout, "%s\n", v)
			return true
		})

		// If the number of fetched results is less than requested, then
		// this was the last page, otherwise process the next one:
		if response.Size() < size {
			break
		}

		index++
	}

	return nil
}
