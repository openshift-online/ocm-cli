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

package list

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/util"
	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	"github.com/openshift-online/uhc-sdk-go/pkg/client/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	parameter []string
	debug     bool
	managed   bool
}

var managed bool

var Cmd = &cobra.Command{
	Use:   "list [flags] ",
	Short: "List clusters",
	Long:  "List clusters by ID and Name",
	Run:   run,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
	flags.StringArrayVar(
		&args.parameter,
		"parameter",
		nil,
		"Query parameters to add to the request. The value must be the name of the "+
			"parameter, followed by an optional equals sign and then the value "+
			"of the parameter. Can be used multiple times to specify multiple "+
			"parameters or multiple values for the same parameter.",
	)
	flags.BoolVar(
		&args.managed,
		"managed",
		false,
		"Filter managed/unmanaged clusters",
	)
}

func run(cmd *cobra.Command, argv []string) {

	pageSize := 100
	pageIndex := 1

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
	armed, err := config.Armed(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection:
	logger, err := util.NewLogger(args.debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create logger: %v\n", err)
		os.Exit(1)
	}

	// Create the connection, and remember to close it:
	connection, err := client.NewConnectionBuilder().
		Logger(logger).
		TokenURL(cfg.TokenURL).
		Client(cfg.ClientID, cfg.ClientSecret).
		Scopes(cfg.Scopes...).
		URL(cfg.URL).
		User(cfg.User, cfg.Password).
		Tokens(cfg.AccessToken, cfg.RefreshToken).
		Insecure(cfg.Insecure).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	collection := connection.ClustersMgmt().V1().Clusters()

	if cmd.Flags().Changed("managed") {
		managed = args.managed
	} else {
		managed = false
	}

	for {
		// Fetch the next page:
		response := getResponse(collection, managed, args.parameter, pageSize, pageIndex)

		// Display the fetched page:
		response.Items().Each(func(cluster *v1.Cluster) bool {
			fmt.Printf("ID: %s - Name: %s.%s\n", cluster.ID(), cluster.Name(), cluster.DNS().BaseDomain())
			return true
		})
		// If the number of fetched results is less than requested, then
		// this was the last page, otherwise process the next one:
		if response.Size() < pageSize {
			break
		}
		pageIndex++
	}
}

func getResponse(collection *v1.ClustersClient,
	managed bool,
	parameter []string,
	pageSize int,
	pageIndex int) *v1.ClustersListResponse {

	listRequest := collection.List().
		Size(pageSize).
		Page(pageIndex)

	if managed {
		listRequest.Search("managed='true'")
	} else if len(parameter) > 0 {
		for _, parameter := range args.parameter {
			var name string
			var value string
			position := strings.Index(parameter, "=")
			if position != -1 {
				name = parameter[:position]
				value = parameter[position+1:]
			} else {
				name = parameter
				value = ""
			}
			listRequest.Parameter(name, value)
		}
	}

	response, err := listRequest.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't retrieve clusters: %s", err)
		os.Exit(1)
	}

	return response
}
