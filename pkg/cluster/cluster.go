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
	"reflect"
	"regexp"
	"strings"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	ProviderAWS = "aws"
	ProviderGCP = "gcp"

	NetworkTypeSDN = "OpenShiftSDN"
	NetworkTypeOVN = "OVNKubernetes"

	AuthenticationWif = "Workload Identity Federation (WIF)"
	AuthenticationKey = "Service account"
)

type DefaultIngressSpec struct {
	RouteSelectors           map[string]string
	ExcludedNamespaces       []string
	WildcardPolicy           string
	NamespaceOwnershipPolicy string
}

func NewDefaultIngressSpec() DefaultIngressSpec {
	defaultIngressSpec := DefaultIngressSpec{}
	defaultIngressSpec.RouteSelectors = map[string]string{}
	defaultIngressSpec.ExcludedNamespaces = []string{}
	return defaultIngressSpec
}

// Spec is the configuration for a cluster spec.
type Spec struct {
	// Basic configs
	Name             string
	DomainPrefix     string
	Region           string
	Provider         string
	CCS              CCS
	ExistingVPC      ExistingVPC
	ClusterWideProxy ClusterWideProxy
	Flavour          string
	MultiAZ          bool
	Version          string
	ChannelGroup     string
	Expiration       time.Time
	Fips             bool
	EtcdEncryption   bool
	SubscriptionType string

	// Scaling config
	ComputeMachineType string
	ComputeNodes       int
	Autoscaling        Autoscaling

	// Network config
	NetworkType string
	MachineCIDR net.IPNet
	ServiceCIDR net.IPNet
	PodCIDR     net.IPNet
	HostPrefix  int
	Private     *bool

	// Properties
	CustomProperties map[string]string

	// Default Ingress Attributes
	DefaultIngress DefaultIngressSpec

	// Gcp-specific settings
	GcpSecurity GcpSecurity

	// GCP Authentication settings
	GcpAuthentication GcpAuthentication

	// GCP PrivateServiceConnect settings
	GcpPrivateSvcConnect GcpPrivateSvcConnect

	//Includes Custom KMS encryption key settings
	GcpEncryption GcpEncryption
}

type Autoscaling struct {
	Enabled     bool
	MinReplicas int
	MaxReplicas int
}

type CCS struct {
	Enabled bool
	AWS     AWSCredentials
	GCP     GCPCredentials
}

type ExistingVPC struct {
	Enabled                                bool
	SubnetIDs                              string
	AvailabilityZones                      []string
	VPCName                                string
	VPCProjectID                           string
	ControlPlaneSubnet                     string
	ComputeSubnet                          string
	AdditionalComputeSecurityGroupIds      []string
	AdditionalInfraSecurityGroupIds        []string
	AdditionalControlPlaneSecurityGroupIds []string
}

type ClusterWideProxy struct {
	Enabled                   bool
	HTTPProxy                 *string
	HTTPSProxy                *string
	NoProxy                   *string
	AdditionalTrustBundleFile *string
	AdditionalTrustBundle     *string
}

type AWSCredentials struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
}

type GCPCredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

type GcpSecurity struct {
	SecureBoot bool `json:"secure_boot,omitempty"`
}

type GcpAuthentication struct {
	Type string
	Id   string
}

type GcpPrivateSvcConnect struct {
	SvcAttachmentSubnet string
}

type GcpEncryption struct {
	KmsKeySvcAccount string
	KmsKeyLocation   string
	KmsKeyRing       string
	KmsKeyName       string
}

type AddOnItem struct {
	ID        string
	Name      string
	State     string
	Available bool
}

type lmtSprReasonItem struct {
	ID      string
	Summary string
	Details string
}

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func GetCluster(connection *sdk.Connection, key string) (cluster *cmv1.Cluster, err error) {
	if !IsValidClusterKey(key) {
		return nil, fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			key,
		)
	}

	// Prepare the resources that we will be using:
	subsResource := connection.AccountsMgmt().V1().Subscriptions()
	clustersResource := connection.ClustersMgmt().V1().Clusters()

	// Try to find a matching subscription:
	subsSearch := fmt.Sprintf(
		"(display_name = '%s' or cluster_id = '%s' or external_cluster_id = '%s')",
		key, key, key,
	)
	subsListResponse, err := subsResource.List().
		Search(subsSearch).
		Size(1).
		Send()
	if err != nil {
		err = fmt.Errorf("Can't retrieve subscription for key '%s': %v", key, err)
		return
	}

	// If there is exactly one matching subscription then return the corresponding cluster:
	subsTotal := subsListResponse.Total()
	if subsTotal == 1 {
		sub := subsListResponse.Items().Slice()[0]
		status, ok := sub.GetStatus()
		subID, _ := sub.GetID()
		if !ok || (status != "Reserved" && status != "Active") {
			err = fmt.Errorf("Cluster was %s, see `ocm get subscription %s` for details", status, subID)
			return
		}
		id, ok := sub.GetClusterID()
		if ok {
			var clusterGetResponse *cmv1.ClusterGetResponse
			clusterGetResponse, err = clustersResource.Cluster(id).Get().
				Send()
			if err != nil {
				err = fmt.Errorf(
					"Can't retrieve cluster for key '%s': %v",
					key, err,
				)
				return
			}
			cluster = clusterGetResponse.Body()
			return
		}
	}

	// If there are multiple subscriptions that match the cluster then we should report it as
	// an error:
	if subsTotal > 1 {
		err = fmt.Errorf(
			"There are %d subscriptions with cluster identifier or name '%s'",
			subsTotal, key,
		)
		return
	}

	// If we are here then no subscription matches the passed key. It may still be possible that
	// the cluster exists but it is not reporting metrics, so it will not have the external
	// identifier in the accounts management service. To find those clusters we need to check
	// directly in the clusters management service.
	clustersSearch := fmt.Sprintf(
		"id = '%s' or name = '%s' or external_id = '%s'",
		key, key, key,
	)
	clustersListResponse, err := clustersResource.List().
		Search(clustersSearch).
		Size(1).
		Send()
	if err != nil {
		err = fmt.Errorf("Can't retrieve clusters for key '%s': %v", key, err)
		return
	}

	// If there is exactly one cluster matching then return it:
	clustersTotal := clustersListResponse.Total()
	if clustersTotal == 1 {
		cluster = clustersListResponse.Items().Slice()[0]
		return
	}

	// If there are multiple matching clusters then we should report it as an error:
	if clustersTotal > 1 {
		err = fmt.Errorf(
			"There are %d clusters with identifier or name '%s'",
			clustersTotal, key,
		)
		return
	}

	// If we are here then there are no subscriptions or clusters matching the passed key:
	err = fmt.Errorf(
		"There are no subscriptions or clusters with identifier or name '%s'",
		key,
	)
	return
}

func GetClusterLimitedSupportReasons(connection *sdk.Connection, clusterID string) ([]*lmtSprReasonItem, error) {

	limitedSupportReasons, err := connection.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		LimitedSupportReasons().
		List().
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get limited Support Reasons: %s", err)
	}

	lmtReason := limitedSupportReasons.Items()

	var clusterLmtSprReasons []*lmtSprReasonItem

	lmtReason.Each(func(lmtSprReason *cmv1.LimitedSupportReason) bool {
		clusterLmtSprReason := lmtSprReasonItem{
			ID:      lmtSprReason.ID(),
			Summary: lmtSprReason.Summary(),
			Details: lmtSprReason.Details(),
		}
		clusterLmtSprReasons = append(clusterLmtSprReasons, &clusterLmtSprReason)

		return true
	})

	return clusterLmtSprReasons, nil
}

func CreateCluster(cmv1Client *cmv1.Client, config Spec, dryRun bool) (*cmv1.Cluster, error) {
	clusterProperties := map[string]string{}

	if config.CustomProperties != nil {
		for key, value := range config.CustomProperties {
			clusterProperties[key] = value
		}
	}

	// Create the cluster:
	clusterBuilder := cmv1.NewCluster().
		Name(config.Name).
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
		FIPS(config.Fips).
		EtcdEncryption(config.EtcdEncryption).
		BillingModel(cmv1.BillingModel(config.SubscriptionType)).
		Properties(clusterProperties)

	if config.DomainPrefix != "" {
		clusterBuilder = clusterBuilder.DomainPrefix(config.DomainPrefix)
	}

	clusterBuilder = clusterBuilder.Version(
		cmv1.NewVersion().
			ID(config.Version).ChannelGroup(config.ChannelGroup))

	if !config.Expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(config.Expiration)
	}

	if config.NetworkType != "" ||
		!cidrIsEmpty(config.MachineCIDR) ||
		!cidrIsEmpty(config.ServiceCIDR) ||
		!cidrIsEmpty(config.PodCIDR) ||
		config.HostPrefix != 0 {
		networkBuilder := cmv1.NewNetwork()
		if config.NetworkType != "" {
			networkBuilder = networkBuilder.Type(config.NetworkType)
		}
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

	gcpBuilder := cmv1.NewGCP()

	if config.CCS.Enabled {
		clusterBuilder = clusterBuilder.CCS(cmv1.NewCCS().Enabled(true))
		switch config.Provider {
		case ProviderAWS:
			var subnets []string
			if config.ExistingVPC.SubnetIDs != "" {
				subnets = strings.Split(config.ExistingVPC.SubnetIDs, ",")
			}
			awsBuilder := cmv1.NewAWS().
				AccountID(config.CCS.AWS.AccountID).
				AccessKeyID(config.CCS.AWS.AccessKeyID).
				SecretAccessKey(config.CCS.AWS.SecretAccessKey).
				SubnetIDs(subnets...)
			if len(config.ExistingVPC.AdditionalComputeSecurityGroupIds) != 0 {
				awsBuilder.AdditionalComputeSecurityGroupIds(config.ExistingVPC.AdditionalComputeSecurityGroupIds...)
			}
			if len(config.ExistingVPC.AdditionalInfraSecurityGroupIds) != 0 {
				awsBuilder.AdditionalInfraSecurityGroupIds(config.ExistingVPC.AdditionalInfraSecurityGroupIds...)
			}
			if len(config.ExistingVPC.AdditionalControlPlaneSecurityGroupIds) != 0 {
				awsBuilder.AdditionalControlPlaneSecurityGroupIds(config.ExistingVPC.AdditionalControlPlaneSecurityGroupIds...)
			}
			clusterBuilder = clusterBuilder.AWS(awsBuilder)
		case ProviderGCP:
			switch config.GcpAuthentication.Type {
			case AuthenticationWif:
				if config.GcpAuthentication.Id == "" {
					return nil, fmt.Errorf("missing WIF config ID")
				}
				gcpAuth := cmv1.NewGcpAuthentication().
					Kind(cmv1.WifConfigKind).
					Id(config.GcpAuthentication.Id)
				gcpBuilder.Authentication(gcpAuth)
			case AuthenticationKey:
				if config.CCS.GCP.Type == "" || config.CCS.GCP.ClientEmail == "" ||
					config.CCS.GCP.ProjectID == "" {
					return nil, fmt.Errorf("missing credentials for GCP CCS cluster")
				}
				gcpBuilder.
					Type(config.CCS.GCP.Type).
					ProjectID(config.CCS.GCP.ProjectID).
					PrivateKeyID(config.CCS.GCP.PrivateKeyID).
					PrivateKey(config.CCS.GCP.PrivateKey).
					ClientEmail(config.CCS.GCP.ClientEmail).
					ClientID(config.CCS.GCP.ClientID).
					AuthURI(config.CCS.GCP.AuthURI).
					TokenURI(config.CCS.GCP.TokenURI).
					AuthProviderX509CertURL(config.CCS.GCP.AuthProviderX509CertURL).
					ClientX509CertURL(config.CCS.GCP.ClientX509CertURL)
			default:
				return nil, fmt.Errorf("unexpected GCP authentication method %q", config.GcpAuthentication.Type)
			}

			if isGCPNetworkExists(config.ExistingVPC) {
				gcpNetwork := cmv1.NewGCPNetwork().VPCName(config.ExistingVPC.VPCName).
					ControlPlaneSubnet(config.ExistingVPC.ControlPlaneSubnet).ComputeSubnet(config.ExistingVPC.ComputeSubnet)
				if isGCPSharedVPC(config.ExistingVPC) {
					gcpNetwork = gcpNetwork.VPCProjectID(config.ExistingVPC.VPCProjectID)
				}
				clusterBuilder = clusterBuilder.GCPNetwork(gcpNetwork)
			}
		default:
			return nil, fmt.Errorf("Unexpected CCS provider %q", config.Provider)
		}

		//cluster-wide proxy
		if config.ClusterWideProxy.Enabled {
			proxyBuilder := cmv1.NewProxy()
			if config.ClusterWideProxy.HTTPProxy != nil && len(*config.ClusterWideProxy.HTTPProxy) != 0 {
				proxyBuilder.HTTPProxy(*config.ClusterWideProxy.HTTPProxy)
			}
			if config.ClusterWideProxy.HTTPSProxy != nil && len(*config.ClusterWideProxy.HTTPSProxy) != 0 {
				proxyBuilder.HTTPSProxy(*config.ClusterWideProxy.HTTPSProxy)
			}
			if config.ClusterWideProxy.NoProxy != nil && len(*config.ClusterWideProxy.NoProxy) != 0 {
				proxyBuilder.NoProxy(*config.ClusterWideProxy.NoProxy)
			}
			clusterBuilder = clusterBuilder.Proxy(proxyBuilder)
		}

		if config.ClusterWideProxy.AdditionalTrustBundle != nil {
			clusterBuilder = clusterBuilder.AdditionalTrustBundle(*config.ClusterWideProxy.AdditionalTrustBundle)
		}
	}

	if config.GcpSecurity.SecureBoot {
		gcpSecurity := cmv1.NewGcpSecurity().SecureBoot(config.GcpSecurity.SecureBoot)
		gcpBuilder.Security(gcpSecurity)
	}

	if useGcpCustomEncryption(config.GcpEncryption) {
		gcpEncryption := cmv1.NewGCPEncryptionKey().
			KeyLocation(config.GcpEncryption.KmsKeyLocation).
			KeyRing(config.GcpEncryption.KmsKeyRing).
			KeyName(config.GcpEncryption.KmsKeyName).
			KMSKeyServiceAccount(config.GcpEncryption.KmsKeySvcAccount)
		clusterBuilder.GCPEncryptionKey(gcpEncryption)
	}

	if isGcpPsc(config.GcpPrivateSvcConnect) {
		gcpPsc := cmv1.NewGcpPrivateServiceConnect().ServiceAttachmentSubnet(config.GcpPrivateSvcConnect.SvcAttachmentSubnet)
		gcpBuilder.PrivateServiceConnect(gcpPsc)
	}

	if config.ComputeMachineType != "" || config.ComputeNodes > 0 || len(config.ExistingVPC.AvailabilityZones) > 0 ||
		config.Autoscaling.Enabled {
		clusterNodesBuilder := cmv1.NewClusterNodes()
		if config.ComputeMachineType != "" {
			clusterNodesBuilder = clusterNodesBuilder.ComputeMachineType(
				cmv1.NewMachineType().ID(config.ComputeMachineType),
			)
		}
		clusterNodesBuilder = buildCompute(config, clusterNodesBuilder)

		if len(config.ExistingVPC.AvailabilityZones) > 0 {
			availabilityZones := strings.Join(config.ExistingVPC.AvailabilityZones, ",")
			clusterNodesBuilder = clusterNodesBuilder.AvailabilityZones(strings.Split(availabilityZones, ",")...)
		}
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	}

	if !reflect.DeepEqual(config.DefaultIngress, NewDefaultIngressSpec()) {
		defaultIngress := cmv1.NewIngress().Default(true)
		if len(config.DefaultIngress.RouteSelectors) != 0 {
			defaultIngress.RouteSelectors(config.DefaultIngress.RouteSelectors)
		}
		if len(config.DefaultIngress.ExcludedNamespaces) != 0 {
			defaultIngress.ExcludedNamespaces(config.DefaultIngress.ExcludedNamespaces...)
		}
		if config.DefaultIngress.WildcardPolicy != "" {
			defaultIngress.RouteWildcardPolicy(cmv1.WildcardPolicy(config.DefaultIngress.WildcardPolicy))
		}
		if config.DefaultIngress.NamespaceOwnershipPolicy != "" {
			defaultIngress.RouteNamespaceOwnershipPolicy(
				cmv1.NamespaceOwnershipPolicy(config.DefaultIngress.NamespaceOwnershipPolicy))
		}
		clusterBuilder.Ingresses(cmv1.NewIngressList().Items(defaultIngress))
	}

	if config.Provider == ProviderGCP {
		clusterBuilder = clusterBuilder.GCP(gcpBuilder)
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create description of cluster: %v", err)
	}

	// Send a request to create the cluster:
	request := cmv1Client.Clusters().Add().
		Body(clusterSpec)
	if dryRun {
		request = request.Parameter("dryRun", "true")
	}
	response, err := request.Send()
	if err != nil {
		if dryRun {
			return nil, fmt.Errorf("dry run: unable to create cluster: %v", err)
		}
		return nil, fmt.Errorf("unable to create cluster: %v", err)
	}

	if response.Status() == http.StatusNoContent {
		return nil, nil
	}
	return response.Body(), nil
}

func isGCPNetworkExists(existingVPC ExistingVPC) bool {
	return existingVPC.VPCName != "" || existingVPC.ControlPlaneSubnet != "" ||
		existingVPC.ComputeSubnet != "" || existingVPC.VPCProjectID != ""
}

func useGcpCustomEncryption(gcpEncData GcpEncryption) bool {
	return gcpEncData.KmsKeyLocation != "" || gcpEncData.KmsKeyRing != "" ||
		gcpEncData.KmsKeyName != "" || gcpEncData.KmsKeySvcAccount != ""
}

func isGCPSharedVPC(existingVPC ExistingVPC) bool {
	return existingVPC.VPCProjectID != ""
}

func isGcpPsc(gcpPsc GcpPrivateSvcConnect) bool {
	return gcpPsc.SvcAttachmentSubnet != ""
}
func UpdateCluster(client *cmv1.ClustersClient, clusterID string, config Spec) error {
	clusterBuilder := cmv1.NewCluster()

	// Update expiration timestamp
	if !config.Expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(config.Expiration)
	}

	// Scale cluster
	if config.ComputeNodes > 0 || config.Autoscaling.Enabled {
		clusterBuilder = clusterBuilder.Nodes(buildCompute(config, cmv1.NewClusterNodes()))
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

	if config.ChannelGroup != "" {
		clusterBuilder = clusterBuilder.Version(cmv1.NewVersion().ChannelGroup(config.ChannelGroup))
	}

	clusterProxyBuilder := cmv1.NewProxy()
	if config.ClusterWideProxy.HTTPProxy != nil || config.ClusterWideProxy.HTTPSProxy != nil {
		if config.ClusterWideProxy.HTTPProxy != nil {
			clusterProxyBuilder = clusterProxyBuilder.HTTPProxy(*config.ClusterWideProxy.HTTPProxy)
		}
		if config.ClusterWideProxy.HTTPSProxy != nil {
			clusterProxyBuilder = clusterProxyBuilder.HTTPSProxy(*config.ClusterWideProxy.HTTPSProxy)
		}
		clusterBuilder = clusterBuilder.Proxy(clusterProxyBuilder)
	}
	if config.ClusterWideProxy.NoProxy != nil {
		clusterProxyBuilder = clusterProxyBuilder.NoProxy(*config.ClusterWideProxy.NoProxy)
		clusterBuilder = clusterBuilder.Proxy(clusterProxyBuilder)
	}

	if config.ClusterWideProxy.AdditionalTrustBundle != nil {
		clusterBuilder = clusterBuilder.AdditionalTrustBundle(*config.ClusterWideProxy.AdditionalTrustBundle)
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

func UpdateDeleteProtection(client *cmv1.ClustersClient, clusterID string, enable bool) error {
	deleteProtection, _ := cmv1.NewDeleteProtection().Enabled(enable).Build()
	_, err := client.Cluster(clusterID).DeleteProtection().Update().Body(deleteProtection).Send()
		client.Cluster(clusterID).DeleteProtection().Update().Body(deleteProtection))
	if err != nil {
		return err
	}
	return nil
}

func buildCompute(config Spec, clusterNodesBuilder *cmv1.ClusterNodesBuilder) *cmv1.ClusterNodesBuilder {
	if config.Autoscaling.Enabled {
		autoscalingBuilder := cmv1.NewMachinePoolAutoscaling()
		if config.Autoscaling.MinReplicas != 0 {
			autoscalingBuilder = autoscalingBuilder.MinReplicas(config.Autoscaling.MinReplicas)
		}
		if config.Autoscaling.MaxReplicas != 0 {
			autoscalingBuilder = autoscalingBuilder.MaxReplicas(config.Autoscaling.MaxReplicas)
		}
		clusterNodesBuilder = clusterNodesBuilder.AutoscaleCompute(autoscalingBuilder)
	} else if config.ComputeNodes > 0 {
		clusterNodesBuilder = clusterNodesBuilder.Compute(config.ComputeNodes)
	}
	return clusterNodesBuilder
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

	// Get a list of quota-cost for the current organization
	quotaCostResponse, err := connection.AccountsMgmt().V1().Organizations().
		Organization(organization).QuotaCost().
		List().
		Parameter("fetchRelatedResources", true).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to get quota-cost: %v", err)
	}
	quotaCosts := quotaCostResponse.Items()

	// Get complete list of enabled add-ons
	addOnsResponse, err := connection.AddonsMgmt().V1().Addons().
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
	addOnInstallationsResponse, err := connection.AddonsMgmt().V1().Clusters().
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
	addOns.Each(func(addOn *asv1.Addon) bool {
		if addOn.ID() != "rhmi" {
			clusterAddOn := AddOnItem{
				ID:        addOn.ID(),
				Name:      addOn.Name(),
				State:     "not installed",
				Available: addOn.ResourceCost() == 0,
			}

			// Only display add-ons for which the org has quota
			quotaCosts.Each(func(quotaCost *amv1.QuotaCost) bool {
				relatedResources := quotaCost.RelatedResources()
				for _, relatedResource := range relatedResources {
					if relatedResource.ResourceType() == "add-on" &&
						addOn.ResourceName() == relatedResource.ResourceName() {
						clusterAddOn.Available = true
						break
					}
				}
				return true
			})

			// Get the state of add-on installations on the cluster
			addOnInstallations.Each(func(addOnInstallation *asv1.AddonInstallation) bool {
				if addOn.ID() == addOnInstallation.Addon().ID() {
					clusterAddOn.State = string(addOnInstallation.State())
					if clusterAddOn.State == "" {
						clusterAddOn.State = string(asv1.AddonInstallationStateInstalling)
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
		return createVersionID(cluster.OpenshiftVersion(), cluster.Version().ChannelGroup())
	}
	return cluster.Version().ID()
}

func GetAvailableUpgrades(
	client *cmv1.Client, versionID string, productID string) ([]string, error) {
	response, err := client.Versions().Version(versionID).Get().Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to find version ID %s", versionID)
	}
	version := response.Body()
	availableUpgrades := version.AvailableUpgrades()
	if productID == "ROSA" {
		availableUpgrades, err = filterROSAVersions(
			client, availableUpgrades, version.ChannelGroup())
		if err != nil {
			return nil, err
		}
	}

	return availableUpgrades, nil
}

func createVersionID(version string, channelGroup string) string {
	versionID := fmt.Sprintf("openshift-v%s", version)
	if channelGroup != "stable" {
		versionID = fmt.Sprintf("%s-%s", versionID, channelGroup)
	}
	return versionID

}

func filterROSAVersions(
	client *cmv1.Client, versions []string, channelGroup string) ([]string, error) {
	enabledVersions := []string{}
	for _, version := range versions {
		versionID := createVersionID(version, channelGroup)
		response, err := client.Versions().Version(versionID).Get().Send()
		if err != nil {
			return nil, fmt.Errorf("Failed to find version ID %s", versionID)
		}
		rosaEnabled := response.Body().ROSAEnabled()
		if rosaEnabled {
			enabledVersions = append(enabledVersions, version)
		}
	}
	return enabledVersions, nil
}

func cidrIsEmpty(cidr net.IPNet) bool {
	return cidr.String() == "<nil>"
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
