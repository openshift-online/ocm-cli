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

package status

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "status [flags] CLUSTER_ID",
	Short: "Status of a cluster",
	Long:  "Get the status of a cluster identified by its cluster ID",
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {

	if len(argv) != 1 {
		return fmt.Errorf("Expected exactly one cluster id")
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	resource := connection.ClustersMgmt().V1().Clusters()

	// Get the resource that manages the cluster that we want to display:
	clusterResource := resource.Cluster(argv[0])

	// Retrieve the collection of clusters:
	response, err := clusterResource.Get().
		Send()
	if err != nil {
		return fmt.Errorf("Can't retrieve clusters: %s", err)
	}

	cluster := response.Body()

	//Get data out of the response
	clusterMemory := cluster.Metrics().Memory()
	clusterCPU := cluster.Metrics().CPU()
	memUsed := clusterMemory.Used().Value() / 1000000000
	memTotal := clusterMemory.Total().Value() / 1000000000

	fmt.Printf("State:   %s\n"+
		"Memory:  %.2f/%.2f used\n"+
		"CPU:     %.2f/%.2f used\n",
		cluster.State(),
		memUsed, memTotal,
		clusterCPU.Used().Value(), clusterCPU.Total().Value(),
	)

	return nil
}
