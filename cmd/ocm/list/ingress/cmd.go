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
	"context"
	"fmt"
	"os"
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/output"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "ingresses --cluster={NAME|ID|EXTERNAL_ID}",
	Aliases: []string{"route", "routes", "ingress"},
	Short:   "List cluster Ingresses",
	Long:    "List API and ingress endpoints for a cluster.",
	Example: `  # List all routes on a cluster named "mycluster"
  ocm list ingresses --cluster=mycluster`,
	Args: cobra.NoArgs,
	RunE: run,
}

func init() {
	fs := Cmd.Flags()

	fs.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to list the routes of (required).",
	)

	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {
	// Create a context:
	ctx := context.Background()

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return err
	}
	defer connection.Close()

	// Create the output printer:
	printer, err := output.NewPrinter().
		Writer(os.Stdout).
		Pager(cfg.Pager).
		Build(ctx)
	if err != nil {
		return err
	}
	defer printer.Close()

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

	// Get the client for the cluster management api
	clusterCollection := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(clusterCollection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	ingresses, err := c.GetIngresses(clusterCollection, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
	}

	// Write the endpoints:
	endpointsTable, err := printer.NewTable().
		Name("endpoints").
		Columns("id", "api.url", "api.listening").
		Value("id", "api").
		Build(ctx)
	if err != nil {
		return err
	}
	err = endpointsTable.WriteHeaders()
	if err != nil {
		return err
	}
	err = endpointsTable.WriteObject(cluster)
	if err != nil {
		return err
	}
	err = endpointsTable.Close()
	if err != nil {
		return err
	}
	fmt.Fprintf(printer, "\n")

	// Write the ingresses:
	ingressesTable, err := printer.NewTable().
		Name("ingresses").
		Columns("id", "application_router", "listening", "default", "route_selectors").
		Value("application_router", applicationRouter).
		Value("route_selectors", routeSelectors).
		Build(ctx)
	if err != nil {
		return err
	}
	err = ingressesTable.WriteHeaders()
	if err != nil {
		return err
	}
	for _, ingress := range ingresses {
		err = ingressesTable.WriteObject(ingress)
		if err != nil {
			break
		}
	}
	if err != nil {
		return err
	}
	err = ingressesTable.Close()
	if err != nil {
		return err
	}

	return nil
}

func routeSelectors(ingress *cmv1.Ingress) string {
	routeSelectors := ingress.RouteSelectors()
	if len(routeSelectors) == 0 {
		return ""
	}
	output := []string{}
	for k, v := range routeSelectors {
		output = append(output, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(output, ", ")
}

func applicationRouter(ingress *cmv1.Ingress) string {
	return "https://" + ingress.DNSName()
}
