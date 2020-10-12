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

package ingress

import (
	"fmt"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{4,5}$`)
var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "ingress --cluster={NAME|ID|EXTERNAL_ID} [flags] INGRESS_ID",
	Aliases: []string{"ingresses", "route", "routes"},
	Short:   "Delete cluster ingress",
	Long:    "Delete the additional non-default application router for a cluster.",
	Example: `  # Delete ingress with ID a1b2 from a cluster named 'mycluster'
  ocm delete ingress --cluster=mycluster a1b2
  # Delete secondary ingress using the sub-domain name
  ocm delete ingress --cluster=mycluster apps2`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to delete the ingress from (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {

	// Check command line arguments:
	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameters containing the ID " +
				"of the ingress.",
		)
	}

	ingressID := argv[0]
	if !ingressKeyRE.MatchString(ingressID) {
		return fmt.Errorf(
			"Ingress  identifier '%s' isn't valid: it must contain only four letters or digits",
			ingressID,
		)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}
	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the cluster management api
	clusterCollection := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(clusterCollection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	ingresses, err := c.GetIngresses(clusterCollection, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
	}

	var ingress *cmv1.Ingress
	for _, item := range ingresses {
		if ingressID == "apps" && item.Default() {
			return fmt.Errorf("Default ingress '%s' on cluster '%s' cannot be deleted", ingressID, clusterKey)
		}
		if ingressID == "apps2" && !item.Default() {
			ingress = item
		}
		if item.ID() == ingressID {
			ingress = item
		}
	}
	if ingress == nil {
		return fmt.Errorf("Failed to get ingress '%s' for cluster '%s'", ingressID, clusterKey)
	}

	_, err = clusterCollection.
		Cluster(cluster.ID()).
		Ingresses().
		Ingress(ingress.ID()).
		Delete().
		Send()
	if err != nil {
		return fmt.Errorf("Failed to delete ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
	}

	fmt.Printf("Deleted ingress '%s' on cluster '%s'\n", ingressID, clusterKey)
	return nil
}
