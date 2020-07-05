/*
Copyright (c) 2020 Red Hat, Inc.

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

package cluster

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	clusterpkg "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	json   bool
	output bool
}

var Cmd = &cobra.Command{
	Use:   "cluster NAME|ID|EXTERNAL_ID",
	Short: "Show details of a cluster",
	Long:  "Show details of a cluster identified by name, identifier or external identifier",
	RunE:  run,
}

func init() {
	// Add flags to rootCmd:
	flags := Cmd.Flags()
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

func run(cmd *cobra.Command, argv []string) error {
	// Check that there is exactly one cluster name, identifir or external identifier in the
	// command line arguments:
	if len(argv) != 1 {
		fmt.Fprintf(
			os.Stderr,
			"Expected exactly one cluster name, identifier or external identifier "+
				"is required\n",
		)
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	key := argv[0]
	if !keyRE.MatchString(key) {
		fmt.Fprintf(
			os.Stderr,
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores\n",
			key,
		)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Try to find the cluster that has the name, identifier or external identifier matching the
	// value given by the user:
	search := fmt.Sprintf("name = '%s' or id = '%s' or external_id = '%s'", key, key, key)
	response, err := connection.ClustersMgmt().V1().Clusters().List().
		Search(search).
		Size(1).
		Send()
	if err != nil {
		return fmt.Errorf("Can't retrieve cluster for key '%s': %v", key, err)
	}
	clusters := response.Items().Slice()
	if len(clusters) == 0 {
		fmt.Fprintf(
			os.Stderr,
			"There is no cluster with name, identifier or external identifier '%s'\n",
			key,
		)
		os.Exit(1)
	}
	cluster := clusters[0]

	if args.output {
		// Create a filename based on cluster name:
		filename := fmt.Sprintf("cluster-%s.json", cluster.ID())

		// Attempt to create file:
		myFile, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("Failed to create file: %v", err)
		}

		// Dump encoder content into file:
		err = cmv1.MarshalCluster(cluster, myFile)
		if err != nil {
			return fmt.Errorf("Failed to Marshal cluster into file: %v", err)
		}
	}

	// Get full API response (JSON):
	if args.json {
		// Buffer for pretty output:
		buf := new(bytes.Buffer)
		fmt.Println()

		// Convert cluster to JSON and dump to encoder:
		err = cmv1.MarshalCluster(cluster, buf)
		if err != nil {
			return fmt.Errorf("Failed to Marshal cluster into JSON encoder: %v", err)
		}

		if response.Status() < 400 {
			err = dump.Pretty(os.Stdout, buf.Bytes())
		} else {
			err = dump.Pretty(os.Stderr, buf.Bytes())
		}
		if err != nil {
			return fmt.Errorf("Can't print body: %v", err)
		}

	} else {
		err = clusterpkg.PrintClusterDesctipion(connection, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

// Regular expression to check that the cluster key (name, identifier or external identifier) given
// by the user is reasonably safe and that there is no risk of SQL injection.
var keyRE = regexp.MustCompile(`^(\w|-)+$`)
