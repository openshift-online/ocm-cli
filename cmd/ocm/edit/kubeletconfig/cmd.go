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
	"fmt"
	"net/http"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	kc "github.com/openshift-online/ocm-cli/pkg/kubeletconfig"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var args struct {
	clusterKey   string
	podPidsLimit int
	yes          bool
}

var Cmd = &cobra.Command{
	Use:     "kubeletconfig --cluster={NAME|ID|EXTERNAL_ID} [flags]",
	Aliases: []string{"kubelet-config"},
	Short:   "Edit a kubeletconfig for a cluster",
	Long:    "Edit the kubeletconfig for a cluster.",
	Example: `  # Edit the kubeletconfig to have a pod-pids-limit of 10000 for cluster 'mycluster'
  ocm edit kubeletconfig --cluster=mycluster --pod-pids-limit=10000`,
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

	flags.IntVar(
		&args.podPidsLimit,
		"pod-pids-limit",
		0,
		fmt.Sprintf(
			"Sets the podPidsLimit to be applied in the KubeletConfig. "+
				"Minimum: %d, Maximum: %d (or up to %d with org capability).",
			kc.MinPodPidsLimit, kc.MaxPodPidsLimit, kc.MaxUnsafePodPidsLimit,
		),
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("pod-pids-limit")

	flags.BoolVarP(
		&args.yes,
		"yes",
		"y",
		false,
		"Skip the interactive confirmation prompt.",
	)
}

// run is the Cobra RunE handler for "ocm edit kubeletconfig".
func run(cmd *cobra.Command, argv []string) error {
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	if err = kc.ValidatePodPidsLimit(connection, args.podPidsLimit); err != nil {
		return err
	}

	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Can't retrieve cluster for key '%s': %v", clusterKey, err)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf(
			"Cluster '%s' is not yet ready. Current state is '%s'",
			clusterKey, cluster.State(),
		)
	}

	existingResponse, err := connection.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).KubeletConfig().Get().Send()
	if err != nil {
		if existingResponse != nil && existingResponse.Status() == http.StatusNotFound {
			return fmt.Errorf(
				"No KubeletConfig exists for cluster '%s'. "+
					"You should first create one via 'ocm create kubeletconfig'",
				clusterKey,
			)
		}
		return fmt.Errorf("Failed to get KubeletConfig for cluster '%s': %v", clusterKey, err)
	}

	var confirmed bool
	if args.yes {
		confirmed = true
	} else {
		confirmed, err = kc.ConfirmWorkerNodeReboot("Editing")
		if err != nil {
			return err
		}
	}
	if !confirmed {
		return nil
	}

	kubeletConfig, err := cmv1.NewKubeletConfig().PodPidsLimit(args.podPidsLimit).Build()
	if err != nil {
		return fmt.Errorf("Failed to build KubeletConfig: %v", err)
	}

	_, err = connection.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).KubeletConfig().Update().Body(kubeletConfig).Send()
	if err != nil {
		return fmt.Errorf("Failed to update KubeletConfig for cluster '%s': %v", clusterKey, err)
	}

	fmt.Printf("Successfully updated KubeletConfig for cluster '%s'\n", clusterKey)
	return nil
}
