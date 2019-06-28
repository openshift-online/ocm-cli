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
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/openshift-online/uhc-sdk-go/pkg/client/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/openshift-online/uhc-cli/pkg/config"
)

var args struct {
	user string
}

const ClustersPageSize = 50

var Cmd = &cobra.Command{
	Use:   "login [CLUSTERID|CLUSTER_NAME|CLUSTER_NAME_SEARCH]",
	Short: "login to a cluster",
	Long: "login to a cluster by ID or Name or cluster name search string according to the api: " +
		"https://api.openshift.com/#/clusters/get_api_clusters_mgmt_v1_clusters",
	Example: " uhc cluster login <id>\n uhc cluster login %test%",
	Run:     run,
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

}
func run(cmd *cobra.Command, argv []string) {

	if len(argv) != 1 {
		fmt.Fprint(os.Stderr, "Expected exactly one cluster\n")
		os.Exit(1)
	}
	path, err := exec.LookPath("oc")
	if err != nil {
		fmt.Fprint(os.Stderr, "To run this, you need install openshfit oc first.\n")
		os.Exit(1)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprint(os.Stderr, "Not logged in, run the 'login' command\n")
		os.Exit(1)
	}

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprint(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection, and remember to close it:
	connection, err := cfg.Connection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	collection := connection.ClustersMgmt().V1().Clusters()
	clusters, total, err := findClusters(collection, argv[0], ClustersPageSize)
	if err != nil || len(clusters) == 0 {
		fmt.Fprintf(os.Stderr, "Can't find clusters: %v\n", err)
		os.Exit(1)
	}

	// If there are more clusters than `ClustersPageSize`, print a msg out
	if total > ClustersPageSize {
		fmt.Printf(
			"There are %d clusters that match key '%s', but only the first %d will "+
				"be shown; consider using a more specific key.\n",
			total, argv[0], len(clusters),
		)
	}
	var clusterid, clusterName, url string
	if len(clusters) == 1 {
		for _, v := range clusters {
			clusterid = v.ID()
			clusterName = v.Name()
		}
		url = clusters[0].API().URL()
		fmt.Printf("Only one cluster match the args, will login to cluster:\n Name: %s\n ID: %s\n", clusterName, clusterid)
	} else {
		cluster, err := doSurvey(clusters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't find clusters: %v\n", err)
			os.Exit(1)
		}
		url = cluster.API().URL()
		clusterid = cluster.ID()
		clusterName = cluster.Name()
	}

	if len(url) == 0 {
		fmt.Fprintf(os.Stderr, "Cannot find the api url for cluster: %s\n", clusterName)
		os.Exit(1)
	}
	fmt.Printf("Will login to cluster:\n Name: %s\n ID: %s\n", clusterName, clusterid)
	ocArgs := []string{}
	ocArgs = append(ocArgs, "login", url)
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
		fmt.Fprintf(os.Stderr, "Failed to login to cluster: %s\n", err)
		os.Exit(1)
	}
}

// doSurvey will ask user to choose one if there are more than one clusters match the query
func doSurvey(clusters []*v1.Cluster) (cluster *v1.Cluster, err error) {
	clusterList := []string{}
	for _, v := range clusters {
		clusterList = append(clusterList, fmt.Sprintf("Name: %s, ID: %s", v.Name(), v.ID()))
	}
	choice := ""
	prompt := &survey.Select{
		Message: "Please choose a cluster:",
		Options: clusterList,
		Default: clusterList[0],
	}
	survey.PageSize = ClustersPageSize
	err = survey.AskOne(prompt, &choice, func(ans interface{}) error {
		choice := ans.(string)
		found := false
		for _, v := range clusters {
			if strings.Contains(choice, v.ID()) {
				found = true
				cluster = v
			}
		}
		if !found {
			return fmt.Errorf("the cluster you choose is not valid: %s", choice)
		}
		return nil
	})
	return cluster, err
}

// findClusters finds the clusters that match the given key. A cluster matches the key if its
// identifier is that key, or if its name starts with that key. For example, the key `prd-2305`
// doesn't match a cluster directly because it isn't a valid identifier, but it matches all clusters
// whose names start with `prd-2305`.
func findClusters(collection *v1.ClustersClient, key string, size int) (clusters []*v1.Cluster, total int, err error) {

	// Get the resource that manages the cluster that we want to display:
	clusterResource := collection.Cluster(key)
	response, err := clusterResource.Get().Send()

	if err == nil && response != nil {
		cluster := response.Body()
		clusters = []*v1.Cluster{cluster}
		total = 1
		return
	}
	if response == nil || response.Status() != http.StatusNotFound {
		return
	}
	// If it's not an cluster id, try to query clusters using search param, we only list the
	// the `size` number of clusters.
	pageIndex := 1
	listRequest := collection.List().
		Size(size).
		Page(pageIndex)
	listRequest.Search("name like '" + key + "'")
	listResponse, err := listRequest.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't retrieve clusters: %s\n", err)
		return
	}
	total = listResponse.Total()
	listResponse.Items().Each(func(cluster *v1.Cluster) bool {
		clusters = append(clusters, cluster)
		return true
	})
	return
}
