/*
Copyright (c) 2026 Red Hat

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

package kubeletconfig

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey string
	json       bool
}

var Cmd = &cobra.Command{
	Use:     "kubeletconfig --cluster={NAME|ID|EXTERNAL_ID}",
	Aliases: []string{"kubelet-config"},
	Short:   "Show details of a kubeletconfig for a cluster",
	Long:    "Show details of the kubeletconfig for a cluster.",
	Example: `  # Describe the kubeletconfig for cluster 'mycluster'
  ocm describe kubeletconfig --cluster=mycluster`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.BoolVar(
		&args.json,
		"json",
		false,
		"Output the entire JSON structure.",
	)
}

// run is the Cobra RunE handler for "ocm describe kubeletconfig".
func run(cmd *cobra.Command, argv []string) error {
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	clusterKey := args.clusterKey

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Can't retrieve cluster for key '%s': %v", clusterKey, err)
	}

	response, err := connection.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).KubeletConfig().Get().Send()
	if err != nil {
		if response != nil && response.Status() == http.StatusNotFound {
			return fmt.Errorf("No KubeletConfig exists for cluster '%s'. "+
				"You can create one via 'ocm create kubeletconfig'", clusterKey)
		}
		return fmt.Errorf("Failed to get KubeletConfig for cluster '%s': %v", clusterKey, err)
	}

	kubeletConfig := response.Body()

	if args.json {
		buf := new(bytes.Buffer)
		err = cmv1.MarshalKubeletConfig(kubeletConfig, buf)
		if err != nil {
			return fmt.Errorf("Failed to marshal KubeletConfig to JSON: %v", err)
		}
		err = dump.Pretty(os.Stdout, buf.Bytes())
		if err != nil {
			return fmt.Errorf("Failed to print KubeletConfig JSON: %v", err)
		}
		return nil
	}

	printKubeletConfig(kubeletConfig)
	return nil
}

// printKubeletConfig writes a human-readable summary of kc to stdout.
func printKubeletConfig(kc *cmv1.KubeletConfig) {
	fmt.Printf("%-20s %s\n", "ID:", kc.ID())
	if kc.Name() != "" {
		fmt.Printf("%-20s %s\n", "Name:", kc.Name())
	}
	fmt.Printf("%-20s %d\n", "Pod PIDs Limit:", kc.PodPidsLimit())
}
