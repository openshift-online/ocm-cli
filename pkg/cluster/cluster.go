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
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	sdk "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const AWS = "aws"

// Spec is the configuration for a cluster spec.
type Spec struct {
	// Basic configs
	Name           string
	Region         string
	Provider       string
	Flavour        string
	MultiAZ        bool
	CCS            bool
	AWSCredentials AWSCredentials
	Version        string
	Expiration     time.Time

	// Scaling config
	ComputeMachineType string
	ComputeNodes       int

	// Network config
	MachineCIDR net.IPNet
	ServiceCIDR net.IPNet
	PodCIDR     net.IPNet
	HostPrefix  int
	Private     *bool

	// Properties
	CustomProperties map[string]string
}

type AWSCredentials struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
}

type AddOnItem struct {
	ID        string
	Name      string
	State     string
	Available bool
}

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func GetCluster(client *cmv1.ClustersClient, clusterKey string) (*cmv1.Cluster, error) {
	query := fmt.Sprintf(
		"(id = '%s' or name = '%s' or external_id = '%s')",
		clusterKey, clusterKey, clusterKey,
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

func CreateCluster(cmv1Client *cmv1.Client, config Spec, parameters []string, headers []string) (*cmv1.Cluster, error) {
	clusterProperties := map[string]string{}

	if config.CustomProperties != nil {
		for key, value := range config.CustomProperties {
			clusterProperties[key] = value
		}
	}

	// Create the cluster:
	clusterBuilder := cmv1.NewCluster().
		Name(config.Name).
		DisplayName(config.Name).
		MultiAZ(config.MultiAZ).
		CloudProvider(
			cmv1.NewCloudProvider().
				ID(config.Provider),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(config.Region),
		).
		Flavour(
			cmv1.NewFlavour().
				ID(config.Flavour),
		).
		Properties(clusterProperties)

	if config.Version != "" {
		clusterBuilder = clusterBuilder.Version(
			cmv1.NewVersion().
				ID(config.Version),
		)

	}

	if config.ComputeMachineType != "" || config.ComputeNodes != 0 {
		clusterNodesBuilder := cmv1.NewClusterNodes()
		if config.ComputeMachineType != "" {
			clusterNodesBuilder = clusterNodesBuilder.ComputeMachineType(
				cmv1.NewMachineType().ID(config.ComputeMachineType),
			)
		}
		if config.ComputeNodes != 0 {
			clusterNodesBuilder = clusterNodesBuilder.Compute(config.ComputeNodes)
		}
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	}

	if !config.Expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(config.Expiration)
	}

	if !cidrIsEmpty(config.MachineCIDR) ||
		!cidrIsEmpty(config.ServiceCIDR) ||
		!cidrIsEmpty(config.PodCIDR) ||
		config.HostPrefix != 0 {
		networkBuilder := cmv1.NewNetwork()
		if !cidrIsEmpty(config.MachineCIDR) {
			networkBuilder = networkBuilder.MachineCIDR(config.MachineCIDR.String())
		}
		if !cidrIsEmpty(config.ServiceCIDR) {
			networkBuilder = networkBuilder.ServiceCIDR(config.ServiceCIDR.String())
		}
		if !cidrIsEmpty(config.PodCIDR) {
			networkBuilder = networkBuilder.PodCIDR(config.PodCIDR.String())
		}
		if config.HostPrefix != 0 {
			networkBuilder = networkBuilder.HostPrefix(config.HostPrefix)
		}
		clusterBuilder = clusterBuilder.Network(networkBuilder)
	}

	if config.Private != nil {
		if *config.Private {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodInternal),
			)
		} else {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodExternal),
			)
		}
	}

	if config.CCS {
		clusterBuilder = clusterBuilder.CCS(cmv1.NewCCS().Enabled(true))
		clusterBuilder = clusterBuilder.AWS(
			cmv1.NewAWS().
				AccountID(config.AWSCredentials.AccountID).
				AccessKeyID(config.AWSCredentials.AccessKeyID).
				SecretAccessKey(config.AWSCredentials.SecretAccessKey),
		)
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create description of cluster: %v", err)
	}

	// Send a request to create the cluster:
	request := cmv1Client.Clusters().Add().
		Body(clusterSpec)
	arguments.ApplyParameterFlag(request, parameters)
	arguments.ApplyHeaderFlag(request, headers)
	response, err := request.Send()
	if err != nil {
		return nil, fmt.Errorf("unable to create cluster: %v", err)
	}

	// Happens in dryRun mode when there were no errors.
	if response.Status() == http.StatusNoContent {
		return nil, nil
	}
	return response.Body(), nil
}

func UpdateCluster(client *cmv1.ClustersClient, clusterID string, config Spec) error {

	clusterBuilder := cmv1.NewCluster()

	// Update expiration timestamp
	if !config.Expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(config.Expiration)
	}

	// Scale cluster
	if config.ComputeNodes != 0 {
		clusterBuilder = clusterBuilder.Nodes(
			cmv1.NewClusterNodes().
				Compute(config.ComputeNodes),
		)
	}

	// Toggle private mode
	if config.Private != nil {
		if *config.Private {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodInternal),
			)
		} else {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodExternal),
			)
		}
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return err
	}

	_, err = client.Cluster(clusterID).Update().Body(clusterSpec).Send()
	if err != nil {
		return err
	}

	return nil
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

func GetMachinePools(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.MachinePool, error) {
	response, err := client.Cluster(clusterID).MachinePools().
		List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get machine pools for cluster '%s': %v", clusterID, err)
	}

	return response.Items().Slice(), nil
}

func GetUpgradePolicies(client *cmv1.ClustersClient, clusterID string) ([]*cmv1.UpgradePolicy, error) {
	response, err := client.Cluster(clusterID).UpgradePolicies().
		List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get upgrade policies for cluster '%s': %v", clusterID, err)
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

func GetVersionID(cluster *cmv1.Cluster) string {
	if cluster.OpenshiftVersion() != "" {

		if cluster.Version().ChannelGroup() != "stable" {
			return fmt.Sprintf("openshift-v%s-%s", cluster.OpenshiftVersion(), cluster.Version().ChannelGroup())
		}
		return fmt.Sprintf("openshift-v%s", cluster.OpenshiftVersion())
	}
	return cluster.Version().ID()

}

func GetAvailableUpgrades(client *cmv1.Client, versionID string) ([]string, error) {
	response, err := client.Versions().Version(versionID).Get().Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to find version ID %s", versionID)
	}
	availableUpgrades := response.Body().AvailableUpgrades()

	return availableUpgrades, nil
}

func cidrIsEmpty(cidr net.IPNet) bool {
	return cidr.String() == "<nil>"
}

func getMachineTypes(client *cmv1.Client, provider string) (machineTypes []*cmv1.MachineType, err error) {
	collection := client.MachineTypes()
	page := 1
	size := 100
	for {
		var response *cmv1.MachineTypesListResponse
		response, err = collection.List().
			Search(fmt.Sprintf("cloud_provider.id = '%s'", provider)).
			Order("size desc").
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return
		}
		machineTypes = append(machineTypes, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
}

func ValidateMachineType(client *cmv1.Client, provider string, machineType string) (string, error) {
	machineTypeList, err := getMachineTypeList(client, provider)
	if err != nil {
		return "", err
	}
	if machineType != "" {
		// Check and set the cluster machineType
		hasMachineType := false
		for _, v := range machineTypeList {
			if v == machineType {
				hasMachineType = true
			}
		}
		if !hasMachineType {
			allMachineTypes := strings.Join(machineTypeList, " ")
			err := fmt.Errorf("A valid machine type number must be specified\nValid machine types: %s", allMachineTypes)
			return machineType, err
		}
	}

	return machineType, nil
}

func getMachineTypeList(client *cmv1.Client, provider string) (machineTypeList []string, err error) {
	machineTypes, err := getMachineTypes(client, provider)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve machine types: %s", err)
		return
	}

	for _, v := range machineTypes {
		machineTypeList = append(machineTypeList, v.ID())
	}

	return
}

func ValidateClusterExpiration(
	expirationTime string,
	expirationDuration time.Duration,
) (expiration time.Time, err error) {
	// Validate options
	if len(expirationTime) > 0 && expirationDuration != 0 {
		err = errors.New("At most one of 'expiration-time' or 'expiration' may be specified")
		return
	}

	// Parse the expiration options
	if len(expirationTime) > 0 {
		t, err := parseRFC3339(expirationTime)
		if err != nil {
			err = fmt.Errorf("Failed to parse expiration-time: %s", err)
			return expiration, err
		}

		expiration = t
	}
	if expirationDuration != 0 {
		// round up to the nearest second
		expiration = time.Now().Add(expirationDuration).Round(time.Second)
	}

	return
}

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
