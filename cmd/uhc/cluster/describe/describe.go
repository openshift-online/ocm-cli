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

package describe

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/util"
	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	v1 "github.com/openshift-online/uhc-sdk-go/pkg/client/clustersmgmt/v1"

	"github.com/spf13/cobra"
)

var args struct {
	debug  bool
	json   bool
	output bool
}

var Cmd = &cobra.Command{
	Use:   "describe CLUSTERID [--output] [--short]",
	Short: "Decribe a cluster",
	Long:  "Get info about a cluster identified by its cluster ID",
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
	flags.BoolVar(
		&args.output,
		"output",
		false,
		"Output result into JSON file.",
	)
	flags.BoolVar(
		&args.json,
		"json",
		false,
		"Output the entire JSON structure",
	)
}

func run(cmd *cobra.Command, argv []string) {

	if len(argv) != 1 {
		fmt.Fprintf(os.Stderr, "Expected exactly one cluster\n")
		os.Exit(1)
	}

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
	resource := connection.ClustersMgmt().V1().Clusters()

	// Get the resource that manages the cluster that we want to display:
	clusterResource := resource.Cluster(argv[0])

	// Retrieve the collection of clusters:
	response, err := clusterResource.Get().Send()

	// Get cluster body:
	cluster := response.Body()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't retrieve clusters: %s", err)
		os.Exit(1)
	}

	if args.output {
		// Create a filename based on cluster name:
		filename := fmt.Sprintf("cluster-%s.json", cluster.ID())

		// Attempt to create file:
		myFile, err := os.Create(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create file: %v\n", err)
			os.Exit(1)
		}

		// Reasign encoder io.Writer to file writer:
		encoder := json.NewEncoder(myFile)
		encoder.SetIndent("", " ")

		// Dump encoder content into file:
		err = v1.MarshalCluster(cluster, encoder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to Marshal cluster into file: %v\n", err)
			os.Exit(1)
		}
	}

	// Get full API response (JSON):
	if args.json {
		// Get JSON encoder:
		fmt.Println()
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", " ")

		// Convert cluster to JSON and dump to encoder:
		err = v1.MarshalCluster(cluster, encoder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to Marshal cluster into JSON encoder: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Get creation date info:
		clusterTimetamp := cluster.CreationTimestamp()
		year, month, day := clusterTimetamp.Date()

		// Get API URL:
		api := cluster.API()
		apiURL, _ := api.GetURL()

		// Print short cluster description:
		fmt.Printf("\nID:       %s\n"+
			"Name:     %s.%s\n"+
			"API URL:  %s\n"+
			"Masters:  %d\n"+
			"Computes: %d\n"+
			"Region:   %s\n"+
			"Multi-az: %t\n"+
			"Creator:  %s\n"+
			"Created:  %s %d %d\n",
			cluster.ID(),
			cluster.Name(),
			cluster.DNS().BaseDomain(),
			apiURL,
			cluster.Nodes().Master(),
			cluster.Nodes().Compute(),
			cluster.Region().ID(),
			cluster.MultiAZ(),
			cluster.Creator(),
			month.String(), day, year,
		)
		fmt.Println()
	}
}
