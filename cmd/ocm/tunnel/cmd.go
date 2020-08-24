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

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:   "tunnel --cluster=<CLUSTERID|CLUSTER_NAME|CLUSTER_NAME_SEARCH> -- [sshuttle arguments]",
	Short: "tunnel to a cluster",
	Long: "Use sshuttle to create a ssh tunnel to a cluster by ID or Name or" +
		"cluster name search string according to the api: " +
		"https://api.openshift.com/#/clusters/get_api_clusters_mgmt_v1_clusters",
	Example: " ocm cluster tunnel <id>\n ocm cluster tunnel %test%",
	RunE:    run,
	Hidden:  true,
	Args:    cobra.ArbitraryArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to create a tunnel to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {
	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	path, err := exec.LookPath("sshuttle")
	if err != nil {
		return fmt.Errorf("to run this, you need install the sshuttle tool first")
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

	fmt.Printf("Will create tunnel to cluster:\n Name: %s\n ID: %s\n", cluster.Name(), cluster.ID())

	sshURL, err := generateSSHURI(cluster)
	if err != nil {
		return err
	}

	sshuttleArgs := []string{
		"--remote", sshURL,
		cluster.Network().MachineCIDR(),
		cluster.Network().ServiceCIDR(),
		cluster.Network().PodCIDR(),
	}
	sshuttleArgs = append(sshuttleArgs, argv[0:]...)

	// #nosec G204
	sshuttleCmd := exec.Command(path, sshuttleArgs...)
	sshuttleCmd.Stderr = os.Stderr
	sshuttleCmd.Stdin = os.Stdin
	sshuttleCmd.Stdout = os.Stdout
	err = sshuttleCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to login to cluster: %s", err)
	}

	return nil
}

func generateSSHURI(cluster *clustersmgmtv1.Cluster) (string, error) {
	r := regexp.MustCompile(`(?mi)^https:\/\/api\.(.*):6443`)
	apiURL := cluster.API().URL()
	if len(apiURL) == 0 {
		return "", fmt.Errorf("cannot find the api URL for cluster: %s", cluster.Name())
	}
	base := r.FindStringSubmatch(apiURL)[1]
	if len(base) == 0 {
		return "", fmt.Errorf("unable to match api URL for cluster: %s", cluster.Name())
	}

	return "sre-user@rh-ssh." + base, nil
}
