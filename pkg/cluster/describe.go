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

package cluster

import (
	"fmt"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	notAvailable string = "N/A"
)

func PrintClusterDescription(connection *sdk.Connection, cluster *cmv1.Cluster) error {
	// Get API URL:
	api := cluster.API()
	apiURL, _ := api.GetURL()
	apiListening := api.Listening()

	// Retrieve the details of the subscription:
	var sub *amv1.Subscription
	subID := cluster.Subscription().ID()
	if subID != "" {
		subResponse, err := connection.AccountsMgmt().V1().
			Subscriptions().
			Subscription(subID).
			//nolint
			Get().Parameter("fetchLabels", "true").
			Send()
		if err != nil {
			if subResponse == nil || subResponse.Status() != 404 {
				return fmt.Errorf(
					"can't get subscription '%s': %v",
					subID, err,
				)
			}
		}
		sub = subResponse.Body()
	}

	// Retrieve the details of the account:
	var account *amv1.Account
	accountID := sub.Creator().ID()
	if accountID != "" {
		accountResponse, err := connection.AccountsMgmt().V1().
			Accounts().
			Account(accountID).
			Get().
			Send()
		if err != nil {
			if accountResponse == nil || (accountResponse.Status() != 404 &&
				accountResponse.Status() != 403) {
				return fmt.Errorf(
					"can't get account '%s': %v",
					accountID, err,
				)
			}
		}
		account = accountResponse.Body()
	}

	// Find the details of the creator:
	organization := notAvailable
	if account.Organization() != nil && account.Organization().Name() != "" {
		organization = account.Organization().Name()
	}

	creator := account.Username()
	if creator == "" {
		creator = notAvailable
	}

	email := account.Email()
	if email == "" {
		email = notAvailable
	}

	accountNumber := account.Organization().EbsAccountID()
	if accountNumber == "" {
		accountNumber = notAvailable
	}

	// Find the details of the shard
	shardPath, err := connection.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).
		ProvisionShard().
		Get().
		Send()
	var shard string
	if shardPath != nil && err == nil {
		shard = shardPath.Body().HiveConfig().Server()
	}

	var computesStr string
	if cluster.Nodes().AutoscaleCompute() != nil {
		computesStr = fmt.Sprintf("%d-%d (Autoscaled)",
			cluster.Nodes().AutoscaleCompute().MinReplicas(),
			cluster.Nodes().AutoscaleCompute().MaxReplicas(),
		)
	} else {
		computesStr = fmt.Sprintf("%d", cluster.Nodes().Compute())
	}

	clusterAdminEnabled := false
	if cluster.CCS().Enabled() {
		clusterAdminEnabled = true
	} else {
		for _, label := range sub.Labels() {
			if label.Key() == "capability.cluster.manage_cluster_admin" &&
				//nolint
				label.Value() == "true" {
				clusterAdminEnabled = true
			}
		}
	}

	privateLinkEnabled := false
	stsEnabled := false
	// Setting isExistingVPC to unsupported to avoid confusion
	// when looking at clusters on other providers than AWS
	isExistingVPC := "unsupported"
	if cluster.CloudProvider().ID() == ProviderAWS && cluster.AWS() != nil {
		privateLinkEnabled = cluster.AWS().PrivateLink()
		if cluster.AWS().STS().RoleARN() != "" {
			stsEnabled = true
		}

		isExistingVPC = "false"
		if cluster.AWS().SubnetIDs() != nil && len(cluster.AWS().SubnetIDs()) > 0 {
			//nolint
			isExistingVPC = "true"
		}
	}

	if cluster.CloudProvider().ID() == ProviderGCP &&
		cluster.GCPNetwork().VPCName() != "" && cluster.GCPNetwork().ControlPlaneSubnet() != "" &&
		cluster.GCPNetwork().ComputeSubnet() != "" {
		//nolint
		isExistingVPC = "true"
	}

	// Parse Hypershift-related values
	mgmtClusterName, svcClusterName := findHyperShiftMgmtSvcClusters(connection, cluster)

	// Print short cluster description:
	fmt.Printf("\n"+
		"ID:			%s\n"+
		"External ID:		%s\n"+
		"Name:			%s\n"+
		"Display Name:		%s\n"+
		"State:			%s\n",
		cluster.ID(),
		cluster.ExternalID(),
		cluster.Name(),
		sub.DisplayName(),
		cluster.State(),
	)
	if cluster.Status().State() == cmv1.ClusterStateError {
		fmt.Printf("Details:		%s - %s\n",
			cluster.Status().ProvisionErrorCode(),
			cluster.Status().ProvisionErrorMessage(),
		)
	}
	fmt.Printf("API URL:		%s\n"+
		"API Listening:		%s\n"+
		"Console URL:		%s\n"+
		"Masters:		%d\n"+
		"Infra:			%d\n"+
		"Computes:		%s\n"+
		"Product:		%s\n"+
		"Provider:		%s\n"+
		"Version:		%s\n"+
		"Region:			%s\n"+
		"Multi-az:		%t\n"+
		"CCS:			%t\n"+
		"Subnet IDs:		%s\n"+
		"PrivateLink:		%t\n"+
		"STS:			%t\n"+
		"Existing VPC:		%s\n"+
		"Channel Group:		%v\n"+
		"Cluster Admin:		%t\n"+
		"Organization:		%s\n"+
		"Creator:		%s\n"+
		"Email:			%s\n"+
		"AccountNumber:          %s\n"+
		"Created:		%v\n"+
		"Expiration:		%v\n",
		apiURL,
		apiListening,
		cluster.Console().URL(),
		cluster.Nodes().Master(),
		cluster.Nodes().Infra(),
		computesStr,
		cluster.Product().ID(),
		cluster.CloudProvider().ID(),
		cluster.OpenshiftVersion(),
		cluster.Region().ID(),
		cluster.MultiAZ(),
		cluster.CCS().Enabled(),
		cluster.AWS().SubnetIDs(),
		privateLinkEnabled,
		stsEnabled,
		isExistingVPC,
		cluster.Version().ChannelGroup(),
		clusterAdminEnabled,
		organization,
		creator,
		email,
		accountNumber,
		cluster.CreationTimestamp().Round(time.Second).Format(time.RFC3339Nano),
		cluster.ExpirationTimestamp().Round(time.Second).Format(time.RFC3339Nano),
	)

	// Hive
	if shard != "" {
		fmt.Printf("Shard:			%v\n", shard)
	}

	// HyperShift (should be mutually exclusive with Hive)
	if mgmtClusterName != "" {
		fmt.Printf("Management Cluster:     %s\n", mgmtClusterName)
	}
	if svcClusterName != "" {
		fmt.Printf("Service Cluster:        %s\n", svcClusterName)
	}

	// Cluster-wide-proxy
	if cluster.Proxy().HTTPProxy() != "" {
		fmt.Printf("HTTPProxy:	        %s\n", cluster.Proxy().HTTPProxy())
	}
	if cluster.Proxy().HTTPSProxy() != "" {
		fmt.Printf("HTTPSProxy:	        %s\n", cluster.Proxy().HTTPSProxy())
	}
	if cluster.Proxy().NoProxy() != "" {
		fmt.Printf("NoProxy:	        %s\n", cluster.Proxy().NoProxy())
	}
	if cluster.AdditionalTrustBundle() != "" {
		fmt.Printf("AdditionalTrustBundle:  %s\n", cluster.AdditionalTrustBundle())
	}

	// GCP
	if cluster.GCPNetwork().VPCName() != "" {
		fmt.Printf("VPC-Name:	        %s\n", cluster.GCPNetwork().VPCName())
	}
	if cluster.GCPNetwork().ControlPlaneSubnet() != "" {
		fmt.Printf("Control-Plane-Subnet:   %s\n", cluster.GCPNetwork().ControlPlaneSubnet())
	}
	if cluster.GCPNetwork().ComputeSubnet() != "" {
		fmt.Printf("Compute-Subnet:	        %s\n", cluster.GCPNetwork().ComputeSubnet())
	}

	// Limited Support Status
	if cluster.Status().LimitedSupportReasonCount() > 0 {
		fmt.Printf("Limited Support:	%t\n", cluster.Status().LimitedSupportReasonCount() > 0)
	}

	fmt.Println()

	return nil
}

// findHyperShiftMgmtSvcClusters returns the name of a HyperShift cluster's management and service clusters.
// It essentially ignores error as these endpoint is behind specific permissions by returning empty strings when any
// errors are encountered, which results in them not being printed in the output.
func findHyperShiftMgmtSvcClusters(conn *sdk.Connection, cluster *cmv1.Cluster) (string, string) {
	if !cluster.Hypershift().Enabled() {
		return "", ""
	}

	hypershiftResp, err := conn.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).
		Hypershift().
		Get().
		Send()
	if err != nil {
		return "", ""
	}

	mgmtClusterName := hypershiftResp.Body().ManagementCluster()
	fmMgmtResp, err := conn.OSDFleetMgmt().V1().ManagementClusters().
		List().
		Parameter("search", fmt.Sprintf("name='%s'", mgmtClusterName)).
		Send()
	if err != nil {
		return mgmtClusterName, ""
	}

	if kind := fmMgmtResp.Items().Get(0).Parent().Kind(); kind == "ServiceCluster" {
		return mgmtClusterName, fmMgmtResp.Items().Get(0).Parent().Name()
	}

	// Shouldn't normally happen as every management cluster should have a service cluster
	return mgmtClusterName, ""
}
