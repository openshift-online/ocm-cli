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
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var args struct {
	user    string
	console bool
	token   bool
}

var Cmd = &cobra.Command{
	Use:   "login [CLUSTERID|CLUSTER_NAME|CLUSTER_NAME_SEARCH]",
	Short: "login to a cluster",
	Long: "login to a cluster by ID or Name or cluster name search string according to the api: " +
		"https://api.openshift.com/#/clusters/get_api_clusters_mgmt_v1_clusters",
	Example: " ocm cluster login <id>\n ocm cluster login %test%",
	RunE:    run,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("cluster name expected")
		}

		return nil
	},
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
	flags.BoolVarP(
		&args.token,
		"token",
		"t",
		false,
		"Display the cluster API login token using the default browser",
	)

}
func run(cmd *cobra.Command, argv []string) error {
	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := argv[0]
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"cluster name, identifier '%s' isn't valid: it must contain only"+
				"letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	path, err := exec.LookPath("oc")
	if err != nil {
		return fmt.Errorf("to run this, you need install the OpenShift CLI (oc) first")
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	clusterCollection := connection.ClustersMgmt().V1().Clusters()
	cluster, err := c.GetCluster(clusterCollection, clusterKey)
	if err != nil {
		return fmt.Errorf("failed to get cluster '%s': %v", clusterKey, err)
	}

	fmt.Printf("Will login to cluster:\n Name: %s\n ID: %s\n", cluster.Name(), cluster.ID())

	if args.console {
		if len(cluster.Console().URL()) == 0 {
			return fmt.Errorf("cannot find the console URL for cluster: %s", cluster.Name())
		}

		fmt.Printf(" Console URL: %s\n", cluster.Console().URL())

		// Open the console url in the broswer, return any errors
		return browser.OpenURL(cluster.Console().URL())
	}

	if args.token {
		if len(cluster.Console().URL()) == 0 {
			return fmt.Errorf("cannot find the console URL for cluster: %s", cluster.Name())
		}

		fmt.Printf(" Console URL: %s\n", cluster.Console().URL())

		// Create token url from console URL and open browser
		loginURL := strings.Replace(cluster.Console().URL(), "console-openshift-console", "oauth-openshift", 1)
		loginURL += "/oauth/token/request"
		return browser.OpenURL(loginURL)
	}

	if len(cluster.API().URL()) == 0 {
		return fmt.Errorf("cannot find the api URL for cluster: %s", cluster.Name())
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
		return fmt.Errorf("failed to login to cluster: %s", err)
	}

	return nil
}
