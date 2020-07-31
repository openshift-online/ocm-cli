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
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{4,5}$`)

var args struct {
	clusterKey string
	private    bool
	labelMatch string
}

var Cmd = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"route", "routes", "ingresses"},
	Short:   "Edit a cluster Ingress",
	Long:    "Edit an Ingress endpoint to determine access to the cluster.",
	Example: `  #  Update the router selectors for the additional ingress with ID 'a1b2'
  ocm edit ingress --label-match=foo=bar --cluster=mycluster a1b2
  #  Update the default ingress using the sub-domain identifier
  ocm edit ingress --private=false --cluster=mycluster apps"`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the ingress to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict application route to direct, private connectivity.",
	)

	flags.StringVar(
		&args.labelMatch,
		"label-match",
		"",
		"Label match for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers.",
	)
}

func run(cmd *cobra.Command, argv []string) error {

	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameter containing the id of the ingress")
	}

	ingressID := argv[0]
	if !ingressKeyRE.MatchString(ingressID) {
		return fmt.Errorf(
			"Ingress  identifier '%s' isn't valid: it must contain only letters or digits",
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

	if cluster.CloudProvider().ID() != "aws" {
		return fmt.Errorf(
			"Editing ingresses is not supported for cloud provider '%s'", cluster.CloudProvider().ID())
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	ingresses, err := c.GetIngresses(clusterCollection, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
	}

	var ingress *cmv1.Ingress
	for _, item := range ingresses {
		if ingressID == "apps" && item.Default() {
			ingress = item
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

	ingressBuilder := cmv1.NewIngress().ID(ingress.ID())
	if cmd.Flags().Changed("private") {
		if args.private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}

	routeSelectors := make(map[string]string)
	if args.labelMatch != "" {
		for _, labelMatch := range strings.Split(args.labelMatch, ",") {
			if !strings.Contains(labelMatch, "=") {
				return fmt.Errorf("Expected key=value format for label-match")
			}
			tokens := strings.Split(labelMatch, "=")
			routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}

	// Add route selectors
	if cmd.Flags().Changed("label-match") || len(routeSelectors) > 0 {
		if ingress.Default() {
			return fmt.Errorf("Adding route selectors to the default ingress is not allowed")
		}
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}

	ingress, err = ingressBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to edit ingress for cluster '%s': %v", clusterKey, err)
	}

	_, err = clusterCollection.
		Cluster(cluster.ID()).
		Ingresses().
		Ingress(ingress.ID()).
		Update().
		Body(ingress).
		Send()
	if err != nil {
		return fmt.Errorf("Failed to edit ingress for cluster '%s': %v", clusterKey, err)
	}
	return nil
}
