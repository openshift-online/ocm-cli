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

package cluster

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gopkg.in/AlecAivazis/survey.v1"
)

type AddOnItem struct {
	ID        string
	Name      string
	State     string
	Available bool
}

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

const ClustersPageSize = 50

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func GetCluster(client *cmv1.ClustersClient, clusterKey string) (*cmv1.Cluster, error) {
	query := fmt.Sprintf(
		"(id = '%s' or name = '%s')",
		clusterKey, clusterKey,
	)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to locate cluster '%s': %v", clusterKey, err)
	}

	switch response.Total() {
	case 0:
		return nil, fmt.Errorf("There is no cluster with identifier or name '%s'", clusterKey)
	case 1:
		return response.Items().Slice()[0], nil
	default:
		return nil, fmt.Errorf("There are %d clusters with identifier or name '%s'", response.Total(), clusterKey)
	}
}

func GetClusterOauthURL(cluster *cmv1.Cluster) string {
	var oauthURL string
	consoleURL := cluster.Console().URL()
	if cluster.Product().ID() == "rhmi" {
		oauthURL = strings.Replace(consoleURL, "solution-explorer", "oauth-openshift", 1)
	} else {
		oauthURL = strings.Replace(consoleURL, "console-openshift-console", "oauth-openshift", 1)
	}
	return oauthURL
}

func GetIdentityProviders(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.IdentityProvider, error) {
	idpClient := client.Cluster(clusterID).IdentityProviders()
	response, err := idpClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get identity providers for cluster '%s': %v", clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetIngresses(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.Ingress, error) {
	ingressClient := client.Cluster(clusterID).Ingresses()
	response, err := ingressClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get ingresses for cluster '%s': %v", clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetGroups(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.Group, error) {
	groupClient := client.Cluster(clusterID).Groups()
	response, err := groupClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get groups for cluster '%s': %v", clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetClusterAddOns(connection *sdk.Connection, clusterID string) ([]*AddOnItem, error) {
	// Get organization ID (used to get add-on quotas)
	acctResponse, err := connection.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get current account: %s", err)
	}
	organization := acctResponse.Body().Organization().ID()

	// Get a list of add-on quotas for the current organization
	resourceQuotasResponse, err := connection.AccountsMgmt().V1().Organizations().
		Organization(organization).
		ResourceQuota().
		List().
		Search("resource_type='addon'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-ons: %v", err)
	}
	resourceQuotas := resourceQuotasResponse.Items()

	// Get complete list of enabled add-ons
	addOnsResponse, err := connection.ClustersMgmt().V1().Addons().
		List().
		Search("enabled='t'").
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-ons: %v", err)
	}
	addOns := addOnsResponse.Items()

	// Get add-ons already installed on cluster
	addOnInstallationsResponse, err := connection.ClustersMgmt().V1().Clusters().
		Cluster(clusterID).
		Addons().
		List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get add-on installations for cluster '%s': %v", clusterID, err)
	}
	addOnInstallations := addOnInstallationsResponse.Items()

	var clusterAddOns []*AddOnItem

	// Populate add-on installations with all add-on metadata
	addOns.Each(func(addOn *cmv1.AddOn) bool {
		if addOn.ID() != "rhmi" {
			clusterAddOn := AddOnItem{
				ID:        addOn.ID(),
				Name:      addOn.Name(),
				State:     "not installed",
				Available: addOn.ResourceCost() == 0,
			}

			// Only display add-ons for which the org has quota
			resourceQuotas.Each(func(resourceQuota *amsv1.ResourceQuota) bool {
				if addOn.ResourceName() == resourceQuota.ResourceName() {
					clusterAddOn.Available = float64(resourceQuota.Allowed()) >= addOn.ResourceCost()
				}
				return true
			})

			// Get the state of add-on installations on the cluster
			addOnInstallations.Each(func(addOnInstallation *cmv1.AddOnInstallation) bool {
				if addOn.ID() == addOnInstallation.Addon().ID() {
					clusterAddOn.State = string(addOnInstallation.State())
					if clusterAddOn.State == "" {
						clusterAddOn.State = string(cmv1.AddOnInstallationStateInstalling)
					}
				}
				return true
			})

			// Only display add-ons that meet the above criteria
			if clusterAddOn.Available {
				clusterAddOns = append(clusterAddOns, &clusterAddOn)
			}
		}
		return true

	})

	return clusterAddOns, nil
}

// DoSurvey will ask user to choose one if there are more than one clusters match the query
func DoSurvey(clusters []*clustersmgmtv1.Cluster) (cluster *clustersmgmtv1.Cluster, err error) {
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

// FindClusters finds the clusters that match the given key. A cluster matches the key if its
// identifier is that key, or if its name starts with that key. For example, the key `prd-2305`
// doesn't match a cluster directly because it isn't a valid identifier, but it matches all clusters
// whose names start with `prd-2305`.
func FindClusters(collection *clustersmgmtv1.ClustersClient, key string,
	size int) (clusters []*clustersmgmtv1.Cluster, total int, err error) {

	// Get the resource that manages the cluster that we want to display:
	clusterResource := collection.Cluster(key)
	response, err := clusterResource.Get().Send()

	if err == nil && response != nil {
		cluster := response.Body()
		clusters = []*clustersmgmtv1.Cluster{cluster}
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
	listResponse.Items().Each(func(cluster *clustersmgmtv1.Cluster) bool {
		clusters = append(clusters, cluster)
		return true
	})
	return
}
