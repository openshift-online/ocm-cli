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
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "tunnel [flags] {CLUSTERID|CLUSTER_NAME|CLUSTER_NAME_SEARCH} -- [sshuttle arguments]",
	Short: "tunnel to a cluster",
	Long: "Use sshuttle to create a ssh tunnel to a cluster by ID or Name or" +
		"cluster name search string according to the api: " +
		"https://api.openshift.com/#/clusters/get_api_clusters_mgmt_v1_clusters",
	Example: " ocm tunnel <cluster_id>\n ocm tunnel %test%",
	RunE:    run,
	Hidden:  true,
	Args:    cobra.ArbitraryArgs,
}

func init() {
}

func run(cmd *cobra.Command, argv []string) error {
	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if len(argv) < 1 {
		fmt.Fprintf(
			os.Stderr,
			"Expected exactly one cluster name, identifier or external identifier "+
				"is required\n",
		)
		os.Exit(1)
	}

	clusterKey := argv[0]

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

	err = validateSSHConfig()
	if err != nil {
		return err
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
	sshuttleArgs = append(sshuttleArgs, argv[1:]...)

	// Output sshuttle command execution string for review
	fmt.Printf("\n# %s %s\n\n", path, strings.Join(sshuttleArgs, " "))

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

func validateSSHConfig() error {

	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to get users home directory\n")
	}

	sshConfigFilePath := fmt.Sprintf("%s/.ssh/config", user.HomeDir)

	sshConfigFile, err := os.Open(sshConfigFilePath)
	if err != nil {
		return fmt.Errorf("unable to read SSH config file: %s\n", sshConfigFilePath)
	}

	defer func() {
		if err = sshConfigFile.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	foundSSHConfig, err := parseSSHConfig(sshConfigFile)
	if err != nil {
		return fmt.Errorf("unable to parse SSH config file: %s", err)
	}

	if !foundSSHConfig {
		return fmt.Errorf("ProxyJump configuration not found for \"*.devshift.org\" and \"*.openshiftapps.com\"")
	}

	return nil
}

func parseSSHConfig(configFile *os.File) (bool, error) {
	var found bool

	s := bufio.NewScanner(configFile)

	// Find line containing devshift.org and openshiftapps.com in any order
	// assume this is the ProxyJump configuration
	re := regexp.MustCompile(`^Host\ (\*\.devshift\.org\ \*\.openshiftapps\.com|\*\.openshiftapps\.com\ \*\.devshift\.org)`)

	for s.Scan() {
		found = re.MatchString(s.Text())
		if found {
			break
		}
	}

	err := s.Err()
	if err != nil {
		return false, err
	}

	return found, nil
}
