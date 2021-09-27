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

	// Get details of service logs for this cluster
	serviceLogsSRE, _ := connection.ServiceLogs().V1().Clusters().
		Cluster(cluster.ExternalID()).
		ClusterLogs().
		List().
		Search("service_name = 'SREManualAction'").
		Send()
	serviceLogsSRELastWeek, _ := connection.ServiceLogs().V1().Clusters().
		Cluster(cluster.ExternalID()).
		ClusterLogs().
		List().
		Search("service_name = 'SREManualAction' and created_at >= '" + time.Now().AddDate(0, 0, -7).Format("2006-01-02") + "'").
		Send()

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
				label.Value() == "true" {
				clusterAdminEnabled = true
			}
		}
	}

	privateLinkEnabled := false
	if cluster.CloudProvider().ID() == ProviderAWS && cluster.AWS() != nil {
		privateLinkEnabled = cluster.AWS().PrivateLink()
	}

	// Print short cluster description:
	fmt.Printf("\n"+
		"ID:                   %s\n"+
		"External ID:          %s\n"+
		"Name:                 %s\n"+
		"State:                %s\n",
		cluster.ID(),
		cluster.ExternalID(),
		cluster.Name(),
		cluster.State(),
	)
	if cluster.Status().State() == cmv1.ClusterStateError {
		fmt.Printf("Details:       %s - %s\n",
			cluster.Status().ProvisionErrorCode(),
			cluster.Status().ProvisionErrorMessage(),
		)
	}
	fmt.Printf("API URL:              %s\n"+
		"API Listening:        %s\n"+
		"Console URL:          %s\n"+
		"Masters:              %d\n"+
		"Infra:                %d\n"+
		"Computes:             %s\n"+
		"Product:              %s\n"+
		"Provider:             %s\n"+
		"Version:              %s\n"+
		"Region:               %s\n"+
		"Multi-az:             %t\n"+
		"CCS:                  %t\n"+
		"PrivateLink:          %t\n"+
		"Channel Group:        %v\n"+
		"Cluster Admin:        %t\n"+
		"Organization:         %s\n"+
		"Creator:              %s\n"+
		"Email:                %s\n"+
		"Created:              %v\n"+
		"Expiration:           %v\n",
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
		privateLinkEnabled,
		cluster.Version().ChannelGroup(),
		clusterAdminEnabled,
		organization,
		creator,
		email,
		cluster.CreationTimestamp().Round(time.Second).Format(time.RFC3339Nano),
		cluster.ExpirationTimestamp().Round(time.Second).Format(time.RFC3339Nano),
	)
	if shard != "" {
		fmt.Printf("Shard:                %v\n", shard)
	}
	if serviceLogsSRE, ok := serviceLogsSRE.GetTotal(); ok {
		fmt.Printf("Service Log Count:    %v (%v in last week)\n", serviceLogsSRE, serviceLogsSRELastWeek.Total())
	}
	fmt.Println()

	return nil
}
