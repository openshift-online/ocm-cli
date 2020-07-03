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
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
	private    bool
	labelMatch string
}

var Cmd = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"route", "routes", "ingresses"},
	Short:   "Add Ingress to cluster",
	Long:    "Add an Ingress endpoint to determine API access to the cluster.",
	Example: `  # Add an internal ingress to a cluster named "mycluster"
  ocm-cli create ingress --private --cluster=mycluster
  # Add a public ingress to a cluster
  ocm-cli create ingress --cluster=mycluster
  # Add an ingress with route selector label match
  ocm-cli create ingress -c mycluster --label-match="foo=bar,bar=baz"`,
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
			"Creating ingresses is not supported for cloud provider '%s'", cluster.CloudProvider().ID())
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	ingressBuilder := cmv1.NewIngress()
	if cmd.Flags().Changed("private") {
		if args.private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}
	if len(routeSelectors) > 0 {
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}
	ingress, err := ingressBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to create ingress for cluster '%s': %v", clusterKey, err)
	}

	_, err = clusterCollection.Cluster(cluster.ID()).
		Ingresses().
		Add().
		Body(ingress).
		Send()
	if err != nil {
		return fmt.Errorf("Failed to add ingress to cluster '%s': %v", clusterKey, err)
	}
	return nil
}
