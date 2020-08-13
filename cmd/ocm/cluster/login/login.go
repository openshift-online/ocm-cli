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

package login

import (
	"fmt"
	"os"
	"os/exec"

	clusterpkg "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var args struct {
	user    string
	console bool
}

const ClustersPageSize = 50

var Cmd = &cobra.Command{
	Use:   "login [CLUSTERID|CLUSTER_NAME|CLUSTER_NAME_SEARCH]",
	Short: "login to a cluster",
	Long: "login to a cluster by ID or Name or cluster name search string according to the api: " +
		"https://api.openshift.com/#/clusters/get_api_clusters_mgmt_v1_clusters",
	Example: " ocm cluster login <id>\n ocm cluster login %test%",
	RunE:    run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVarP(
		&args.user,
		"username",
		"u",
		"",
		"Username, will prompt if not provided",
	)
	flags.BoolVarP(
		&args.console,
		"console",
		"",
		false,
		"Open the OpenShift console for the cluster in the default browser",
	)

}
func run(cmd *cobra.Command, argv []string) error {

	if len(argv) != 1 {
		return fmt.Errorf("Expected exactly one cluster")
	}
	path, err := exec.LookPath("oc")
	if err != nil {
		return fmt.Errorf("To run this, you need install the OpenShift CLI (oc) first")
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	collection := connection.ClustersMgmt().V1().Clusters()
	clusters, total, err := clusterpkg.FindClusters(collection, argv[0], clusterpkg.ClustersPageSize)
	if err != nil || len(clusters) == 0 {
		return fmt.Errorf("Can't find clusters: %v", err)
	}

	// If there are more clusters than `ClustersPageSize`, print a msg out
	if total > clusterpkg.ClustersPageSize {
		fmt.Printf(
			"There are %d clusters that match key '%s', but only the first %d will "+
				"be shown; consider using a more specific key.\n",
			total, argv[0], len(clusters),
		)
	}
	var cluster *clustersmgmtv1.Cluster
	if len(clusters) == 1 {
		cluster = clusters[0]
	} else {
		cluster, err = clusterpkg.DoSurvey(clusters)
		if err != nil {
			return fmt.Errorf("Can't find clusters: %v", err)
		}
	}
	fmt.Printf("Will login to cluster:\n Name: %s\n ID: %s\n", cluster.Name(), cluster.ID())

	if args.console {
		if len(cluster.Console().URL()) == 0 {
			return fmt.Errorf("Cannot find the console URL for cluster: %s", cluster.Name())
		}

		fmt.Printf(" Console URL: %s\n", cluster.Console().URL())

		// Open the console url in the broswer, return any errors
		return browser.OpenURL(cluster.Console().URL())
	}

	if len(cluster.API().URL()) == 0 {
		return fmt.Errorf("Cannot find the api URL for cluster: %s", cluster.Name())
	}
	ocArgs := []string{}
	ocArgs = append(ocArgs, "login", cluster.API().URL())
	if args.user != "" {
		ocArgs = append(ocArgs, "--username="+args.user)
	}

	// #nosec G204
	ocCmd := exec.Command(path, ocArgs...)
	ocCmd.Stderr = os.Stderr
	ocCmd.Stdin = os.Stdin
	ocCmd.Stdout = os.Stdout
	err = ocCmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to login to cluster: %s", err)
	}

	return nil
}
