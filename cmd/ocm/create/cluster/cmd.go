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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/openshift-online/ocm-cli/cmd/ocm/edit/ingress"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/billing"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/provider"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultIngressRouteSelectorFlag            = "default-ingress-route-selector"
	defaultIngressExcludedNamespacesFlag       = "default-ingress-excluded-namespaces"
	defaultIngressWildcardPolicyFlag           = "default-ingress-wildcard-policy"
	defaultIngressNamespaceOwnershipPolicyFlag = "default-ingress-namespace-ownership-policy"
	gcpTermsAgreementsHyperlink                = "https://console.cloud.google.com" +
		"/marketplace/agreements/redhat-marketplace/red-hat-openshift-dedicated"
	gcpTermsAgreementInteractiveError    = "Please accept Google Terms and Agreements in order to proceed"
	gcpTermsAgreementNonInteractiveError = "Review and accept Google Terms and Agreements on " +
		gcpTermsAgreementsHyperlink + ". Set the flag --marketplace-gcp-terms to true " +
		"once agreed in order to proceed further."

	privateFlag            = "private"
	vpcNameFlag            = "vpc-name"
	controlPlaneSubnetFlag = "control-plane-subnet"
	computePlaneSubnetFlag = "compute-subnet"
	pscSubnetFlag          = "psc-subnet"
	//Gcp Custom Encryption
	KmsKeyLocationFlag   = "kms-key-location"
	kmsKeyRingFlag       = "kms-key-ring"
	kmsKeyNameFlag       = "kms-key-name"
	kmsKeySvcAccountFlag = "kms-key-service-account"
)

var args struct {
	// positional args
	clusterName  string
	domainPrefix string

	// flags
	interactive bool
	dryRun      bool

	region                string
	version               string
	channelGroup          string
	flavour               string
	provider              string
	expirationTime        string
	expirationSeconds     time.Duration
	private               bool
	multiAZ               bool
	ccs                   c.CCS
	existingVPC           c.ExistingVPC
	clusterWideProxy      c.ClusterWideProxy
	gcpServiceAccountFile arguments.FilePath
	gcpSecureBoot         c.GcpSecurity
	gcpAuthentication     c.GcpAuthentication
	gcpPrivateSvcConnect  c.GcpPrivateSvcConnect
	gcpEncryption         c.GcpEncryption
	gcpWifConfig          string
	fips                  bool
	etcdEncryption        bool
	subscriptionType      string
	marketplaceGcpTerms   bool

	// Scaling options
	computeMachineType string
	computeNodes       int
	autoscaling        c.Autoscaling

	// Networking options
	networkType string
	hostPrefix  int
	machineCIDR net.IPNet
	serviceCIDR net.IPNet
	podCIDR     net.IPNet

	// Default Ingress Attributes
	defaultIngressRouteSelectors           string
	defaultIngressExcludedNamespaces       string
	defaultIngressWildcardPolicy           string
	defaultIngressNamespaceOwnershipPolicy string
}

const clusterNameHelp = "The name can be used as the identifier of the cluster." +
	" The maximum length is 54 characters. Once set, the cluster name cannot be changed."

const subnetTemplate = "%s (%s)"

const subscriptionTypeTemplate = "%s (%s)"

// Creates a subnet options using a predefined template.
func setSubnetOption(subnet, zone string) string {
	return fmt.Sprintf(subnetTemplate, subnet, zone)
}

// Parses the subnet from the option chosen by the user.
func parseSubnet(subnetOption string) string {
	return strings.Split(subnetOption, " ")[0]
}

func setSubscriptionTypeOption(id, description string) string {
	return fmt.Sprintf(subscriptionTypeTemplate, id, description)
}

func parseSubscriptionType(subscriptionTypeOption string) string {
	return strings.Split(subscriptionTypeOption, " ")[0]
}

func setWifConfigOption(id, name string) string {
	return fmt.Sprintf("%s (%s)", name, id)
}

// Returns the name of the WIF config from the option
func parseWifConfigOption(wifConfigOption string) string {
	return strings.Split(wifConfigOption, " ")[0]
}

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "cluster [flags] NAME",
	Short: "Create managed clusters",
	Long: fmt.Sprintf("Create managed OpenShift Dedicated v4 clusters via OCM.\n"+
		"\n"+
		"NAME %s", clusterNameHelp),
	PreRunE: preRun,
	RunE:    run,
}

func init() {
	fs := Cmd.Flags()
	arguments.AddInteractiveFlag(fs, &args.interactive)
	fs.BoolVar(
		&args.dryRun,
		"dry-run",
		false,
		"Simulate creating the cluster.",
	)

	arguments.AddProviderFlag(fs, &args.provider)
	Cmd.RegisterFlagCompletionFunc("provider", arguments.MakeCompleteFunc(osdProviderOptions))

	arguments.AddCCSFlags(fs, &args.ccs)
	arguments.AddExistingVPCFlags(fs, &args.existingVPC)
	arguments.AddClusterWideProxyFlags(fs, &args.clusterWideProxy)

	fs.Var(
		&args.gcpServiceAccountFile,
		"service-account-file",
		"GCP service account JSON file path.",
	)
	fs.StringVar(
		&args.region,
		"region",
		"",
		"The cloud provider region to create the cluster in. See `ocm list regions`.",
	)
	Cmd.MarkFlagRequired("region")
	Cmd.RegisterFlagCompletionFunc("region", arguments.MakeCompleteFunc(getRegionOptions))

	fs.StringVar(
		&args.version,
		"version",
		"",
		"The OpenShift version to create the cluster at (for example, \"4.1.16\")",
	)
	arguments.SetQuestion(fs, "version", "OpenShift version:")
	Cmd.RegisterFlagCompletionFunc("version", arguments.MakeCompleteFunc(getVersionOptions))

	fs.StringVar(
		&args.domainPrefix,
		"domain-prefix",
		"",
		"An optional unique domain prefix of the cluster. If not provided, the cluster name will be "+
			"used if it contains at most 15 characters, otherwise a generated value will be used. This "+
			"will be used when generating a sub-domain for your cluster. It must be unique and consist "+
			"of lowercase alphanumeric,characters or '-', start with an alphabetic character, and end with "+
			"an alphanumeric character. The maximum length is 15 characters. Once set, the cluster domain "+
			"prefix cannot be changed.",
	)
	arguments.SetQuestion(fs, "domain-prefix", "Domain Prefix:")

	fs.StringVar(
		&args.channelGroup,
		"channel-group",
		"",
		"The channel group to create the cluster at (for example, \"stable\")",
	)

	fs.StringVar(
		&args.flavour,
		"flavour",
		"osd-4",
		"The OCM flavour to create the cluster with",
	)
	Cmd.RegisterFlagCompletionFunc("flavour", arguments.MakeCompleteFunc(getFlavourOptions))

	fs.StringVar(
		&args.expirationTime,
		"expiration-time",
		args.expirationTime,
		"Specified time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	//nolint:gosec
	fs.MarkHidden("expiration-time")
	fs.DurationVar(
		&args.expirationSeconds,
		"expiration",
		args.expirationSeconds,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	//nolint:gosec
	fs.MarkHidden("expiration")
	fs.BoolVar(
		&args.private,
		privateFlag,
		false,
		"Restrict master API endpoint and application routes to direct, private connectivity.",
	)
	arguments.SetQuestion(fs, "private", "Private cluster (optional):")
	fs.BoolVar(
		&args.multiAZ,
		"multi-az",
		false,
		"Deploy to multiple data centers.",
	)
	arguments.SetQuestion(fs, "multi-az", "Multiple AZ:")

	fs.BoolVar(
		&args.fips,
		"fips",
		false,
		"Install a cluster that uses FIPS Validated / Modules in Process cryptographic libraries on the x86_64 architecture.",
	)
	arguments.SetQuestion(fs, "fips", "Enable FIPS cryptography:")

	fs.BoolVar(
		&args.etcdEncryption,
		"etcd-encryption",
		false,
		"Add more encryption for OpenShift and Kubernetes API resources.",
	)
	arguments.SetQuestion(fs, "etcd-encryption", "Enable additional etcd encryption:")

	// Scaling options
	fs.StringVar(
		&args.computeMachineType,
		"compute-machine-type",
		"",
		"Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.",
	)
	Cmd.RegisterFlagCompletionFunc("compute-machine-type", arguments.MakeCompleteFunc(getMachineTypeOptions))

	fs.IntVar(
		&args.computeNodes,
		"compute-nodes",
		0,
		fmt.Sprintf("Number of worker nodes to provision. "+
			"Single zone clusters need at least %d nodes on Red Hat infra, "+
			"%d on CCS. "+
			"Multi-AZ at least %d nodes on Red Hat infra, "+
			"%d on CCS, and must be a multiple of 3. "+
			"If omitted, uses minimum.",
			minComputeNodes(false, false), minComputeNodes(true, false),
			minComputeNodes(false, true), minComputeNodes(true, true),
		),
	)
	arguments.AddAutoscalingFlags(fs, &args.autoscaling)

	fs.StringVar(
		&args.networkType,
		"network-type",
		"",
		fmt.Sprintf("The main controller responsible for rendering the core networking components. "+
			"Allowed values are %s and %s", c.NetworkTypeSDN, c.NetworkTypeOVN),
	)
	fs.MarkHidden("network-type")
	Cmd.RegisterFlagCompletionFunc("network-type", networkTypeCompletion)

	fs.IPNetVar(
		&args.machineCIDR,
		"machine-cidr",
		net.IPNet{},
		"Block of IP addresses used by OpenShift while installing the cluster, for example \"10.0.0.0/16\".",
	)
	arguments.SetQuestion(fs, "machine-cidr", "Machine CIDR:")
	fs.IPNetVar(
		&args.serviceCIDR,
		"service-cidr",
		net.IPNet{},
		"Block of IP addresses for services, for example \"172.30.0.0/16\".",
	)
	arguments.SetQuestion(fs, "service-cidr", "Service CIDR:")
	fs.IPNetVar(
		&args.podCIDR,
		"pod-cidr",
		net.IPNet{},
		"Block of IP addresses from which Pod IP addresses are allocated, for example \"10.128.0.0/14\".",
	)
	arguments.SetQuestion(fs, "pod-cidr", "Pod CIDR:")
	fs.IntVar(
		&args.hostPrefix,
		"host-prefix",
		0,
		"Subnet prefix length to assign to each individual node. For example, if host prefix is set "+
			"to \"23\", then each node is assigned a /23 subnet out of the given CIDR.",
	)

	fs.StringVar(
		&args.defaultIngressRouteSelectors,
		defaultIngressRouteSelectorFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers."+
			" For legacy ingress support these are inclusion labels, otherwise they are treated as exclusion label.",
	)

	fs.StringVar(
		&args.defaultIngressExcludedNamespaces,
		defaultIngressExcludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	fs.StringVar(
		&args.defaultIngressWildcardPolicy,
		defaultIngressWildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are { %s }.", strings.Join(ingress.ValidWildcardPolicies, ", ")),
	)

	fs.StringVar(
		&args.defaultIngressNamespaceOwnershipPolicy,
		defaultIngressNamespaceOwnershipPolicyFlag,
		"",
		fmt.Sprintf("Namespace Ownership Policy for ingress. Options are { %s }.",
			strings.Join(ingress.ValidNamespaceOwnershipPolicies, ", ")),
	)

	fs.StringVar(
		&args.subscriptionType,
		"subscription-type",
		billing.StandardSubscriptionType,
		fmt.Sprintf("The subscription billing model for the cluster. Options are { %s }.",
			strings.Join(billing.ValidSubscriptionTypes, ", ")),
	)
	arguments.SetQuestion(fs, "subscription-type", "Subscription type:")
	Cmd.RegisterFlagCompletionFunc("subscription-type", arguments.MakeCompleteFunc(getSubscriptionTypeOptions))

	fs.BoolVar(
		&args.marketplaceGcpTerms,
		"marketplace-gcp-terms",
		false,
		fmt.Sprintf("Review and accept Google Terms and Agreements on %s. "+
			"Set the flag to true once agreed in order to proceed further.", gcpTermsAgreementsHyperlink),
	)
	arguments.SetQuestion(fs, "marketplace-gcp-terms", "I have accepted Google Terms and Agreements:")

	fs.BoolVar(
		&args.gcpSecureBoot.SecureBoot,
		"secure-boot-for-shielded-vms",
		false,
		"Secure Boot enables the use of Shielded VMs in the Google Cloud Platform.",
	)
	arguments.SetQuestion(fs, "secure-boot-for-shielded-vms", "Secure boot support for Shielded VMs:")

	fs.StringVar(
		&args.gcpPrivateSvcConnect.SvcAttachmentSubnet,
		pscSubnetFlag,
		"",
		"Specifies the ServiceAttachment Subnet for Private Service Connect in GCP",
	)
	arguments.SetQuestion(fs, pscSubnetFlag, "PrivateServiceConnect ServiceAttachment Subnet:")

	fs.StringVar(
		&args.gcpWifConfig,
		"wif-config",
		"",
		"Specifies the GCP Workload Identity Federation config used for cloud authentication.",
	)
	arguments.SetQuestion(fs, "wif-config", "WIF configuration:")
	Cmd.RegisterFlagCompletionFunc("wif-config", arguments.MakeCompleteFunc(getWifConfigNameOptions))

	addGcpEncryptionFlags(fs, &args.gcpEncryption)

}

func addGcpEncryptionFlags(fs *pflag.FlagSet, encryptionArgs *c.GcpEncryption) {

	fs.StringVar(
		&encryptionArgs.KmsKeyLocation,
		KmsKeyLocationFlag,
		"",
		"The location of KMS keyring in GCP for custom encryption. "+
			"This should match the data center where the cluster's compute pool will be located.",
	)
	arguments.SetQuestion(fs, KmsKeyLocationFlag, "KMS key location:")
	Cmd.RegisterFlagCompletionFunc(kmsKeyRingFlag, arguments.MakeCompleteFunc(getKmsKeyLocationOptions))

	fs.StringVar(
		&encryptionArgs.KmsKeyRing,
		kmsKeyRingFlag,
		"",
		"The name of the KMS key ring in GCP to use for custom encryption. "+
			"The key ring should belong to the KMS location specified.",
	)
	arguments.SetQuestion(fs, kmsKeyRingFlag, "KMS key ring:")
	Cmd.RegisterFlagCompletionFunc(kmsKeyRingFlag, arguments.MakeCompleteFunc(getKmsKeyRingOptions))

	fs.StringVar(
		&encryptionArgs.KmsKeyName,
		kmsKeyNameFlag,
		"",
		"The name of the KMS key in GCP to use for custom encryption. "+
			"The key should belong to the KMS key ring specified.",
	)
	arguments.SetQuestion(fs, kmsKeyNameFlag, "KMS key name:")
	Cmd.RegisterFlagCompletionFunc(kmsKeyRingFlag, arguments.MakeCompleteFunc(getKmsKeyOptions))

	fs.StringVar(
		&encryptionArgs.KmsKeySvcAccount,
		kmsKeySvcAccountFlag,
		"",
		"The name of the service account in GCP with access to KMS keyring and key specified for custom encryption",
	)
	arguments.SetQuestion(fs, kmsKeySvcAccountFlag, "KMS key service-account name:")
}

func osdProviderOptions(_ *sdk.Connection) ([]arguments.Option, error) {
	return []arguments.Option{
		{Value: c.ProviderAWS, Description: ""},
		{Value: c.ProviderGCP, Description: ""},
	}, nil
}

func getRegionOptions(connection *sdk.Connection) ([]arguments.Option, error) {
	regions, err := provider.GetRegions(connection.ClustersMgmt().V1(), args.provider, args.ccs)
	if err != nil {
		return nil, err
	}
	options := []arguments.Option{}
	for _, region := range regions {
		if !args.ccs.Enabled && region.CCSOnly() {
			continue
		}
		if args.multiAZ && !region.SupportsMultiAZ() {
			continue
		}
		// `enabled` flag only affects Red Hat infra. All regions enabled on CCS.
		if args.ccs.Enabled || region.Enabled() {
			options = append(options, arguments.Option{
				Value:       region.ID(),
				Description: region.DisplayName(),
			})
		}
	}
	return options, nil
}

func getFlavourOptions(connection *sdk.Connection) ([]arguments.Option, error) {
	flavours, err := fetchFlavours(connection.ClustersMgmt().V1())
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve flavours: %s", err)
	}
	options := []arguments.Option{}
	for _, flavour := range flavours {
		options = append(options, arguments.Option{
			Value:       flavour.ID(),
			Description: "",
		})
	}
	return options, nil
}

func GetDefaultClusterFlavors(connection *sdk.Connection, flavour string) (dMachinecidr *net.IPNet, dPodcidr *net.IPNet,
	dServicecidr *net.IPNet, dhostPrefix int) {
	flavourGetResponse, err := connection.ClustersMgmt().V1().Flavours().Flavour(flavour).Get().Send()
	if err != nil {
		flavourGetResponse, _ = connection.ClustersMgmt().V1().Flavours().Flavour("osd-4").Get().Send()
	}

	network, ok := flavourGetResponse.Body().GetNetwork()
	if !ok {
		return nil, nil, nil, 0
	}
	_, dMachinecidr, err = net.ParseCIDR(network.MachineCIDR())
	if err != nil {
		dMachinecidr = nil
	}
	_, dPodcidr, err = net.ParseCIDR(network.PodCIDR())
	if err != nil {
		dPodcidr = nil
	}
	_, dServicecidr, err = net.ParseCIDR(network.ServiceCIDR())
	if err != nil {
		dServicecidr = nil
	}
	dhostPrefix, _ = network.GetHostPrefix()

	return dMachinecidr, dPodcidr, dServicecidr, dhostPrefix
}

func getVersionOptions(connection *sdk.Connection) ([]arguments.Option, error) {
	options, _, err := getVersionOptionsWithDefault(connection, "", "", "")
	return options, err
}

func getVersionOptionsWithDefault(
	connection *sdk.Connection,
	channelGroup string,
	gcpMarketplaceEnabled string,
	additionalFilters string,
) (
	options []arguments.Option, defaultVersion string, err error,
) {
	// Check and set the cluster version
	versionList, defaultVersion, err := c.GetEnabledVersions(
		connection.ClustersMgmt().V1(), channelGroup, gcpMarketplaceEnabled, additionalFilters)
	if err != nil {
		return
	}
	options = []arguments.Option{}
	for _, version := range versionList {
		description := ""
		if version == defaultVersion {
			description = "default"
		}
		options = append(options, arguments.Option{
			Value:       version,
			Description: description,
		})
	}
	return
}

func getSubscriptionTypeOptions(connection *sdk.Connection) ([]arguments.Option, error) {
	options := []arguments.Option{}
	billingModels, err := billing.GetBillingModels(connection)
	if err != nil {
		return options, err
	}
	for _, billingModel := range billingModels {
		option := subscriptionTypeOption(billingModel.ID(), billingModel.Description())
		//Standard billing model should always be the first option
		if billingModel.ID() == billing.StandardSubscriptionType {
			options = append([]arguments.Option{option}, options...)
		} else {
			options = append(options, option)
		}
	}
	return options, nil
}

func subscriptionTypeOption(id string, description string) arguments.Option {
	option := arguments.Option{
		Value: setSubscriptionTypeOption(id, description),
	}
	return option
}

func getMachineTypeOptions(connection *sdk.Connection) ([]arguments.Option, error) {
	return provider.GetMachineTypeOptions(
		connection.ClustersMgmt().V1(),
		args.provider, args.ccs.Enabled)
}

func getWifConfigOptions(wifConfigs []*cmv1.WifConfig) ([]arguments.Option, error) {
	options := []arguments.Option{}
	for _, wc := range wifConfigs {
		option := wifConfigOption(wc.ID(), wc.DisplayName())
		options = append(options, option)
	}
	return options, nil
}

func wifConfigOption(id string, name string) arguments.Option {
	return arguments.Option{
		Value: setWifConfigOption(id, name),
	}
}

// getWifConfigNameOptions returns the wif config options for the cluster
// with display name as the value
func getWifConfigNameOptions(connection *sdk.Connection) ([]arguments.Option, error) {
	return provider.GetWifConfigNameOptions(connection.ClustersMgmt().V1())
}

func getKmsKeyLocationOptions(connection *sdk.Connection) ([]arguments.Option, error) {

	keyLocations, err := provider.GetGcpKmsKeyLocations(connection.ClustersMgmt().V1(),
		args.ccs, args.gcpAuthentication, args.region)
	if err != nil {
		return nil, err
	}

	options := []arguments.Option{}
	for _, keyLoc := range keyLocations {
		options = append(options, arguments.Option{
			Value:       keyLoc,
			Description: keyLoc,
		})
	}
	return options, nil
}

func getKmsKeyRingOptions(connection *sdk.Connection) ([]arguments.Option, error) {

	keyRingList, err := provider.GetGcpKmsKeyRings(connection.ClustersMgmt().V1(),
		args.ccs, args.gcpAuthentication, args.gcpEncryption.KmsKeyLocation)
	if err != nil {
		return nil, err
	}

	options := []arguments.Option{}
	for _, kr := range keyRingList {
		options = append(options, arguments.Option{
			Value:       kr.Name(),
			Description: kr.Name(),
		})
	}

	return options, nil
}

func getKmsKeyOptions(connection *sdk.Connection) ([]arguments.Option, error) {

	keysList, err := provider.GetGcpKmsKeys(connection.ClustersMgmt().V1(),
		args.ccs, args.gcpAuthentication, args.gcpEncryption.KmsKeyLocation, args.gcpEncryption.KmsKeyRing)
	if err != nil {
		return nil, err
	}

	options := []arguments.Option{}
	for _, key := range keysList {
		options = append(options, arguments.Option{
			Value:       key.Name(),
			Description: key.Name(),
		})
	}

	return options, nil
}

func networkTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{c.NetworkTypeSDN, c.NetworkTypeOVN}, cobra.ShellCompDirectiveDefault
}

func minComputeNodes(ccs bool, multiAZ bool) (min int) {
	if ccs {
		if multiAZ {
			min = 3
		} else {
			min = 2
		}
	} else {
		if multiAZ {
			min = 9
		} else {
			min = 4
		}
	}
	return
}

func preRun(cmd *cobra.Command, argv []string) error {
	var err error
	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	err = promptName(argv)
	if err != nil {
		return err
	}

	// Validate flags / ask for missing data.
	fs := cmd.Flags()

	err = promptSubscriptionType(fs, connection)
	if err != nil {
		return err
	}

	// If marketplace-gcp subscription type is used, provider can only be GCP
	gcpBillingModel, _ := billing.GetBillingModel(connection, billing.MarketplaceGcpSubscriptionType)
	gcpSubscriptionTypeTemplate := subscriptionTypeOption(gcpBillingModel.ID(), gcpBillingModel.Description())
	isGcpMarketplace :=
		args.subscriptionType == parseSubscriptionType(gcpSubscriptionTypeTemplate.Value)

	if isGcpMarketplace {
		if args.provider != c.ProviderGCP && args.provider != "" {
			return fmt.Errorf("Provider must be set to %s when using %s subscription type",
				c.ProviderGCP, billing.MarketplaceGcpSubscriptionType)
		}
		fmt.Println("setting provider to", c.ProviderGCP)
		args.provider = c.ProviderGCP
		fmt.Println("setting ccs to 'true'")
		args.ccs.Enabled = true
		if args.interactive {
			fmt.Println("Review and accept Google Terms and Agreements on", gcpTermsAgreementsHyperlink)
			err = arguments.PromptBool(fs, "marketplace-gcp-terms")
			if err != nil {
				return err
			}
		}
		if !args.marketplaceGcpTerms {
			if args.interactive {
				return errors.New(gcpTermsAgreementInteractiveError)
			}
			return errors.New(gcpTermsAgreementNonInteractiveError)
		}
	} else {
		err = promptProvider(fs, connection)
		if err != nil {
			return err
		}
	}

	err = arguments.CheckIgnoredProviderFlags(fs, args.provider)
	if err != nil {
		return err
	}

	if wasClusterWideProxyReceived() {
		args.ccs.Enabled = true
		args.existingVPC.Enabled = true
		args.clusterWideProxy.Enabled = true
	}

	err = promptCCS(fs, args.ccs.Enabled)
	if err != nil {
		return err
	}

	err = promptAuthentication(fs, connection)
	if err != nil {
		return err
	}

	regions, err := getRegionOptions(connection)
	if err != nil {
		return err
	}
	err = arguments.PromptOrCheckOneOf(fs, "region", regions)
	if err != nil {
		return err
	}

	err = arguments.PromptBool(fs, "multi-az")
	if err != nil {
		return err
	}

	err = promptSecureBoot(fs)
	if err != nil {
		return err
	}

	err = promptGcpCustomEncryption(fs, connection)
	if err != nil {
		return err
	}

	err = arguments.PromptBool(fs, "fips")
	if err != nil {
		return err
	}

	err = promptEtcdEncryption(fs)
	if err != nil {
		return err
	}

	var gcpMarketplaceEnabled string
	if isGcpMarketplace {
		gcpMarketplaceEnabled = strconv.FormatBool(isGcpMarketplace)
	}
	additionalFilters := getVersionFilters()
	versions, defaultVersion, err := getVersionOptionsWithDefault(connection, args.channelGroup,
		gcpMarketplaceEnabled, additionalFilters)
	if err != nil {
		return err
	}
	if args.version == "" {
		args.version = defaultVersion
	}
	args.version = c.DropOpenshiftVPrefix(args.version)
	err = arguments.PromptOrCheckOneOf(fs, "version", versions)
	if err != nil {
		return err
	}

	if cmd.Flags().Changed("channel-group") && !cmd.Flags().Changed("version") {
		return fmt.Errorf("Version is required for channel group '%s'", args.channelGroup)
	}

	// Retrieve valid flavours
	flavours, err := getFlavourOptions(connection)
	if err != nil {
		return err
	}
	err = arguments.CheckOneOf(fs, "flavour", flavours)
	if err != nil {
		return err
	}

	// Compute node instance type:
	machineTypes, err := getMachineTypeOptions(connection)
	if err != nil {
		return err
	}
	err = arguments.PromptOrCheckOneOf(fs, "compute-machine-type", machineTypes)
	if err != nil {
		return err
	}

	err = promptAutoscaling(fs)
	if err != nil {
		return err
	}

	err = arguments.CheckAutoscalingFlags(args.autoscaling, args.computeNodes)
	if err != nil {
		return err
	}

	if !args.autoscaling.Enabled {
		// Default compute nodes:
		if args.computeNodes == 0 {
			args.computeNodes = minComputeNodes(args.ccs.Enabled, args.multiAZ)
		}
		err = arguments.PromptInt(fs, "compute-nodes", validateComputeNodes)
		if err != nil {
			return err
		}
	}

	err = promptClusterPrivacy(fs)
	if err != nil {
		return err
	}

	if args.existingVPC.SubnetIDs != "" {
		args.existingVPC.Enabled = true
	}

	err = promptExistingVPC(fs, connection)
	if err != nil {
		return err
	}

	err = promptPrivateServiceConnect(fs)
	if err != nil {
		return err
	}

	err = promptClusterWideProxy()
	if err != nil {
		return err
	}

	if args.interactive {
		machineCIDR, podCIDR, serviceCIDR, hostPrefix := GetDefaultClusterFlavors(connection, args.flavour)
		args.machineCIDR, args.podCIDR, args.serviceCIDR, args.hostPrefix = *machineCIDR, *podCIDR, *serviceCIDR, hostPrefix
	}

	err = promptNetwork(fs)
	if err != nil {
		return err
	}

	err = arguments.PromptString(fs, "domain-prefix")
	if err != nil {
		return err
	}

	return nil
}

func run(cmd *cobra.Command, argv []string) error {
	// TODO: can we reuse the connection from preRun()?
	// TODO: call config.Save (https://github.com/openshift-online/ocm-cli/issues/153).
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	clusterVersion := c.EnsureOpenshiftVPrefix(args.version)

	expiration, err := c.ValidateClusterExpiration(args.expirationTime, args.expirationSeconds)
	if err != nil {
		return err
	}

	defaultIngress, err := buildDefaultIngressSpec()
	if err != nil {
		return err
	}

	clusterConfig := c.Spec{
		Name:                 args.clusterName,
		DomainPrefix:         args.domainPrefix,
		Region:               args.region,
		Provider:             args.provider,
		CCS:                  args.ccs,
		ExistingVPC:          args.existingVPC,
		ClusterWideProxy:     args.clusterWideProxy,
		Flavour:              args.flavour,
		MultiAZ:              args.multiAZ,
		Version:              clusterVersion,
		ChannelGroup:         args.channelGroup,
		Expiration:           expiration,
		ComputeMachineType:   args.computeMachineType,
		ComputeNodes:         args.computeNodes,
		Autoscaling:          args.autoscaling,
		NetworkType:          args.networkType,
		MachineCIDR:          args.machineCIDR,
		ServiceCIDR:          args.serviceCIDR,
		PodCIDR:              args.podCIDR,
		HostPrefix:           args.hostPrefix,
		Private:              &args.private,
		Fips:                 args.fips,
		EtcdEncryption:       args.etcdEncryption,
		DefaultIngress:       defaultIngress,
		SubscriptionType:     args.subscriptionType,
		GcpSecurity:          args.gcpSecureBoot,
		GcpAuthentication:    args.gcpAuthentication,
		GcpPrivateSvcConnect: args.gcpPrivateSvcConnect,
		GcpEncryption:        args.gcpEncryption,
	}

	cluster, err := c.CreateCluster(connection.ClustersMgmt().V1(), clusterConfig, args.dryRun)
	if err != nil {
		return fmt.Errorf("Failed to create cluster: %v", err)
	}

	// Print the result:
	if cluster == nil {
		if args.dryRun {
			fmt.Println("dry run: Would be successful.")
		}
	} else {
		err = c.PrintClusterDescription(connection, cluster)
		if err != nil {
			return err
		}
		err = c.PrintClusterWarnings(connection, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildDefaultIngressSpec() (c.DefaultIngressSpec, error) {
	defaultIngress := c.NewDefaultIngressSpec()
	if args.defaultIngressRouteSelectors != "" {
		routeSelectors, err := ingress.GetRouteSelector(args.defaultIngressRouteSelectors)
		if err != nil {
			return defaultIngress, err
		}
		defaultIngress.RouteSelectors = routeSelectors
	}

	if args.defaultIngressExcludedNamespaces != "" {
		defaultIngress.ExcludedNamespaces = ingress.GetExcludedNamespaces(args.defaultIngressExcludedNamespaces)
	}

	if args.defaultIngressWildcardPolicy != "" {
		defaultIngress.WildcardPolicy = args.defaultIngressWildcardPolicy
	}

	if args.defaultIngressNamespaceOwnershipPolicy != "" {
		defaultIngress.NamespaceOwnershipPolicy = args.defaultIngressNamespaceOwnershipPolicy
	}
	return defaultIngress, nil
}

func wasClusterWideProxyReceived() bool {
	return (args.clusterWideProxy.HTTPProxy != nil && *args.clusterWideProxy.HTTPProxy != "") ||
		(args.clusterWideProxy.HTTPSProxy != nil && *args.clusterWideProxy.HTTPSProxy != "") ||
		(args.clusterWideProxy.AdditionalTrustBundleFile != nil && *args.clusterWideProxy.AdditionalTrustBundleFile != "")
}

// promptName checks and/or reads the cluster name
func promptName(argv []string) error {
	if len(argv) == 1 && argv[0] != "" {
		args.clusterName = argv[0]
		return nil
	}

	if args.interactive {
		prompt := &survey.Input{
			Message: "Cluster name:",
			Help:    clusterNameHelp,
		}
		return survey.AskOne(prompt, &args.clusterName, survey.WithValidator(survey.Required))
	}

	return fmt.Errorf("A cluster name must be specified")
}

func promptSubscriptionType(fs *pflag.FlagSet, connection *sdk.Connection) error {

	subscriptionTypeOptions, _ := getSubscriptionTypeOptions(connection)
	err := arguments.PromptOneOf(fs, "subscription-type", subscriptionTypeOptions)
	if err != nil {
		return err
	}

	if args.interactive {
		args.subscriptionType = parseSubscriptionType(args.subscriptionType)
	}

	if !utils.Contains(billing.ValidSubscriptionTypes, args.subscriptionType) {
		return fmt.Errorf("'%s' is not a valid  subscription type.\nValid options are : %+v",
			args.subscriptionType, subscriptionTypeOptions)
	}

	return nil
}

// promptProvider reads or prompts for the provider
func promptProvider(fs *pflag.FlagSet, connection *sdk.Connection) error {
	// Only offer the 2 providers known to support OSD now;
	// but don't validate if set, to not block `ocm` CLI from creating clusters on future providers.
	providers, _ := osdProviderOptions(connection)

	err := arguments.PromptOneOf(fs, "provider", providers)
	if err != nil {
		return err
	}
	if args.provider == "" {
		return fmt.Errorf("A provider must be specified")
	}
	return nil
}

func promptClusterWideProxy() error {
	var err error
	if args.existingVPC.Enabled && !wasClusterWideProxyReceived() && args.interactive {
		args.clusterWideProxy.Enabled, err = interactive.GetBool(interactive.Input{
			Question: "Use cluster-wide proxy",
			Help: "To install cluster-wide proxy, you need to set one of the following attributes: 'http-proxy', " +
				"'https-proxy', additional-trust-bundle",
			Default: args.clusterWideProxy.Enabled,
		})
		if err != nil {
			return err
		}
	}

	if args.interactive {
		if args.clusterWideProxy.Enabled {
			if args.clusterWideProxy.HTTPProxy == nil {
				args.clusterWideProxy.HTTPProxy = new(string)
			}
			*args.clusterWideProxy.HTTPProxy, err = interactive.GetString(interactive.Input{
				Question: "HTTP Proxy",
				Required: false,
				Default:  *args.clusterWideProxy.HTTPProxy,
			})
			if err != nil {
				return err
			}
			if len(*args.clusterWideProxy.HTTPProxy) == 0 {
				args.clusterWideProxy.HTTPProxy = nil
			} else {
				err := utils.ValidateHTTPProxy(*args.clusterWideProxy.HTTPProxy)
				if err != nil {
					return err
				}
				args.existingVPC.Enabled = true
			}

			if args.clusterWideProxy.HTTPSProxy == nil {
				args.clusterWideProxy.HTTPSProxy = new(string)
			}
			*args.clusterWideProxy.HTTPSProxy, err = interactive.GetString(interactive.Input{
				Question: "HTTPS Proxy",
				Required: false,
				Default:  *args.clusterWideProxy.HTTPSProxy,
			})
			if err != nil {
				return err
			}
			if len(*args.clusterWideProxy.HTTPSProxy) == 0 {
				args.clusterWideProxy.HTTPSProxy = nil
			} else {
				err := utils.IsURL(*args.clusterWideProxy.HTTPSProxy)
				if err != nil {
					return fmt.Errorf("Invalid https-proxy value '%s'", *args.clusterWideProxy.HTTPSProxy)
				}
				args.existingVPC.Enabled = true
			}

			if args.clusterWideProxy.NoProxy == nil {
				args.clusterWideProxy.NoProxy = new(string)
			}
			*args.clusterWideProxy.NoProxy, err = interactive.GetString(interactive.Input{
				Question: "No Proxy",
				Required: false,
				Default:  *args.clusterWideProxy.NoProxy,
			})
			if err != nil {
				return err
			}
			if len(*args.clusterWideProxy.NoProxy) == 0 {
				args.clusterWideProxy.NoProxy = nil
			} else {
				if *args.clusterWideProxy.NoProxy != "" {
					noProxyValues := strings.Split(*args.clusterWideProxy.NoProxy, ",")
					err := utils.MatchNoPorxyRE(noProxyValues)
					if err != nil {
						return err
					}

					duplicate, found := utils.HasDuplicates(noProxyValues)
					if found {
						return fmt.Errorf("no-proxy values must be unique, duplicate key '%s' found", duplicate)
					}
					if args.clusterWideProxy.HTTPProxy == nil && args.clusterWideProxy.HTTPSProxy == nil &&
						len(noProxyValues) > 0 {
						return fmt.Errorf("Expected at least one of the following: http-proxy, https-proxy")
					}
				}
				args.existingVPC.Enabled = true
			}

			if args.clusterWideProxy.AdditionalTrustBundleFile == nil {
				args.clusterWideProxy.AdditionalTrustBundleFile = new(string)
			}
			*args.clusterWideProxy.AdditionalTrustBundleFile, err = interactive.GetString(interactive.Input{
				Question: "Additional trust bundle file path",
				Required: false,
				Default:  *args.clusterWideProxy.AdditionalTrustBundleFile,
			})
			if err != nil {
				return err
			}
			if len(*args.clusterWideProxy.AdditionalTrustBundleFile) == 0 {
				args.clusterWideProxy.AdditionalTrustBundleFile = nil
			} else {
				err := utils.ValidateAdditionalTrustBundle(*args.clusterWideProxy.AdditionalTrustBundleFile)
				if err != nil {
					return err
				}
				args.existingVPC.Enabled = true
			}
		}
	}
	if (args.clusterWideProxy.HTTPProxy != nil && *args.clusterWideProxy.HTTPProxy == "") &&
		(args.clusterWideProxy.HTTPSProxy != nil && *args.clusterWideProxy.HTTPSProxy == "") &&
		(args.clusterWideProxy.NoProxy != nil && *args.clusterWideProxy.NoProxy != "") {
		return fmt.Errorf("Expected at least one of the following: http-proxy, https-proxy")
	}

	// Get certificate contents
	if args.clusterWideProxy.AdditionalTrustBundleFile != nil &&
		*args.clusterWideProxy.AdditionalTrustBundleFile != "" {
		cert, err := os.ReadFile(*args.clusterWideProxy.AdditionalTrustBundleFile)
		if err != nil {
			return err
		}
		args.clusterWideProxy.AdditionalTrustBundle = new(string)
		*args.clusterWideProxy.AdditionalTrustBundle = string(cert)
	}
	if args.clusterWideProxy.AdditionalTrustBundleFile == nil {
		args.clusterWideProxy.AdditionalTrustBundle = nil
	}

	if args.existingVPC.Enabled && args.clusterWideProxy.Enabled && !isAtLeastOneProxyValueSet() {
		return fmt.Errorf("Expected at least one of the following: http-proxy, https-proxy, " +
			"additional-trust-bundle-file")
	}

	return nil
}

func isAtLeastOneProxyValueSet() bool {
	return (args.clusterWideProxy.HTTPProxy != nil && *args.clusterWideProxy.HTTPProxy != "") ||
		(args.clusterWideProxy.HTTPSProxy != nil && *args.clusterWideProxy.HTTPSProxy != "") ||
		(args.clusterWideProxy.AdditionalTrustBundleFile != nil && *args.clusterWideProxy.AdditionalTrustBundleFile != "")
}

func promptExistingAWSVPC(fs *pflag.FlagSet, connection *sdk.Connection) error {
	var err error
	if !args.existingVPC.Enabled && args.existingVPC.SubnetIDs == "" && args.interactive {
		args.existingVPC.Enabled, err = interactive.GetBool(interactive.Input{
			Question: "Install into an existing VPC",
			Help: "To install into an existing VPC you need to ensure that your VPC is configured " +
				"with two subnets for each availability zone that you want the cluster installed into. ",
			Default: args.existingVPC.Enabled,
		})
		if err != nil {
			return err
		}
	}

	if args.existingVPC.Enabled || args.existingVPC.SubnetIDs != "" {
		//subnets provided in the command
		providedSubnetIDs := strings.Split(args.existingVPC.SubnetIDs, ",")
		areSubnetsProvided := len(args.existingVPC.SubnetIDs) > 0

		var availabilityZones []string
		if args.existingVPC.Enabled || areSubnetsProvided {
			//get subnetworks from the provider
			subnetworks, err := provider.GetAWSSubnetworks(connection.ClustersMgmt().V1(),
				args.ccs, args.region)
			if err != nil {
				return err
			}
			var subnetIDs []string
			for _, subnetwork := range subnetworks {
				subnetIDs = append(subnetIDs, subnetwork.SubnetID())
			}

			// Verify subnets provided in the command exist.
			if areSubnetsProvided {
				for _, providedSubnetID := range providedSubnetIDs {
					verifiedSubnet := false
					for _, subnetID := range subnetIDs {
						if subnetID == providedSubnetID {
							verifiedSubnet = true
						}
					}
					if !verifiedSubnet {
						return fmt.Errorf("Could not find the following subnet provided: %s", providedSubnetID)
					}
				}
			}

			mapSubnetToAZ := make(map[string]string)
			mapAZCreated := make(map[string]bool)
			//a map for all provider subnets to be shown in the user prompt
			options := make([]string, len(subnetIDs))
			//a map for all user's provided subnets to be shown in the user prompt
			var defaultOptions []string
			//slice of subnets to send out in the request
			var result []string
			providedSubnetIDMap := make(map[string]bool)
			for _, sub := range providedSubnetIDs {
				providedSubnetIDMap[sub] = true
			}
			for i, subnet := range subnetworks {
				subnetID := subnet.SubnetID()
				availabilityZone := subnet.AvailabilityZone()
				// Create the options to prompt the user.
				options[i] = setSubnetOption(subnetID, availabilityZone)
				if areSubnetsProvided {
					if providedSubnetIDMap[subnetID] {
						//subnetIDs that were provided by the user, so they could be checked while
						//showing up in the prompt. i.s '[X] subnet-xxxxx (us-east-1)'
						defaultOptions = append(defaultOptions, setSubnetOption(subnetID, availabilityZone))
						result = append(result, subnetID)
					}
				}
				mapSubnetToAZ[subnetID] = availabilityZone
				mapAZCreated[availabilityZone] = false
			}

			flag := fs.Lookup("subnet-ids")
			if (!areSubnetsProvided && args.interactive) && !flag.Changed &&
				len(options) > 0 && (!args.multiAZ || len(mapAZCreated) >= 3) {
				result, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Subnet IDs",
					Required: false,
					Options:  options,
					Default:  defaultOptions,
				})
				if err != nil {
					return err
				}
				//remove the az as want to send only the subnet itself.
				//i.e 'subnet-xxxxx' instead 'subnet-xxxxx (us-east-1)'
				for i, subnet := range result {
					result[i] = parseSubnet(subnet)
				}
			}

			//create slice of availability zones to be sent int the request
			for _, subnet := range result {
				az := mapSubnetToAZ[subnet]
				if !mapAZCreated[az] && az != "" {
					availabilityZones = append(availabilityZones, az)
					mapAZCreated[az] = true
				}
			}
			if len(result) > 0 {
				fs.Set("use-existing-vpc", "true")
				fs.Set("subnet-ids", strings.Join(result, ","))
				flag := fs.Lookup("availability-zones")
				if !flag.Changed && len(availabilityZones) > 0 {
					fs.Set("availability-zones", strings.Join(availabilityZones, ","))
				}
			}
		}
		cleanSecurityGroups(&args.existingVPC.AdditionalComputeSecurityGroupIds)
		cleanSecurityGroups(&args.existingVPC.AdditionalInfraSecurityGroupIds)
		cleanSecurityGroups(&args.existingVPC.AdditionalControlPlaneSecurityGroupIds)
	}
	return nil
}

func cleanSecurityGroups(securityGroups *[]string) {
	for i, sg := range *securityGroups {
		(*securityGroups)[i] = strings.TrimSpace(sg)
	}
}

func wasGCPNetworkReceived() bool {
	// 'required' network info for considering it byo-vpc
	// xpn/psc are optional and not required to be set for network info to be considered complete
	return args.existingVPC.VPCName != "" && args.existingVPC.ControlPlaneSubnet != "" &&
		args.existingVPC.ComputeSubnet != ""
}

func promptExistingGCPVPC(fs *pflag.FlagSet, connection *sdk.Connection) error {
	var err error

	//autoenable byo-vpc for wif+private or if GCPNetwork Info has been input
	isWifPrivate := (args.gcpAuthentication.Type == c.AuthenticationWif) && args.private
	if isWifPrivate || wasGCPNetworkReceived() {
		args.existingVPC.Enabled = true
	}

	if !args.existingVPC.Enabled && args.interactive {
		args.existingVPC.Enabled, err = interactive.GetBool(interactive.Input{
			Question: "Install into an existing VPC",
			Help: "To install into an existing VPC you need to ensure that your VPC is configured " +
				"with two subnets for each availability zone that you want the cluster installed into. ",
			Default: args.existingVPC.Enabled,
		})
		if err != nil {
			return err
		}
	}

	//return if user did not enable vpc and only some or none of the gcp network is provided
	//continue if user enabled vpc or all network received
	if !args.existingVPC.Enabled {
		return nil
	}

	err = arguments.PromptString(fs, vpcNameFlag)
	if err != nil {
		return err
	}
	if args.existingVPC.VPCName == "" {
		errMsg := fmt.Sprintf("flag '%s' is required for an existing VPC.", vpcNameFlag)
		if isWifPrivate {
			errMsg += fmt.Sprintf("Installing into an existing VPC with private service connect is required when "+
				"cluster is '%s' and GCP authentication type is '%s' \n", privateFlag, c.AuthenticationWif)
		}
		return errors.New(errMsg)
	}

	err = arguments.PromptString(fs, controlPlaneSubnetFlag)
	if err != nil {
		return err
	}
	if args.existingVPC.ControlPlaneSubnet == "" {
		return fmt.Errorf(
			"flag '%s' is required for an existing VPC", controlPlaneSubnetFlag)
	}

	err = arguments.PromptString(fs, computePlaneSubnetFlag)
	if err != nil {
		return err
	}
	if args.existingVPC.ComputeSubnet == "" {
		return fmt.Errorf(
			"flag '%s' is required for an existing VPC", computePlaneSubnetFlag)
	}

	useSharedVpc := (args.existingVPC.VPCProjectID != "")
	if !useSharedVpc && args.interactive {
		useSharedVpc, err = interactive.GetBool(interactive.Input{
			Question: "Install into a shared VPC",
			Help: "To install with shared VPC you need to have a shared VPC network configured in a separate " +
				"project in the same organization as the project that you want the cluster installed into. ",
			Default: false,
		})
		if err != nil {
			return err
		}
	}

	if useSharedVpc {
		err = arguments.PromptString(fs, "vpc-project-id")
		if err != nil {
			return err
		}
	}

	// skip validation if shared vpc is used
	if !useSharedVpc {
		//get vpc's from the provider
		vpcList, err := provider.GetGCPVPCs(connection.ClustersMgmt().V1(),
			args.ccs, args.gcpAuthentication, args.region)
		if err != nil {
			return err
		}

		verifiedVPCName := false
		for _, vpc := range vpcList {
			if vpc.Name() == args.existingVPC.VPCName {
				verifiedVPCName = true
				break
			}
		}
		if !verifiedVPCName {
			if wasClusterWideProxyReceived() && args.existingVPC.VPCName == "" {
				return fmt.Errorf("Please provide vpc name")
			}
			return fmt.Errorf("Could not find the following vpc name provided: '%s'", args.existingVPC.VPCName)
		}

		//get subnets from the provider
		subnetList, err := provider.GetGCPSubnetList(connection.ClustersMgmt().V1(), args.provider,
			args.ccs, args.gcpAuthentication, args.region)
		if err != nil {
			return err
		}

		// Verify that the control-plane-subnet provided in the command, does exist.
		verifiedControlPlaneSubnet := false
		for _, subnetID := range subnetList {
			if subnetID == args.existingVPC.ControlPlaneSubnet {
				verifiedControlPlaneSubnet = true
				break
			}
		}
		if !verifiedControlPlaneSubnet {
			return fmt.Errorf("Could not find the following control-plane-subnet provided: '%s'",
				args.existingVPC.ControlPlaneSubnet)
		}

		// Verify that compute-subnet provided in the command, does exist.
		verifiedComputeSubnet := false
		for _, subnetID := range subnetList {
			if subnetID == args.existingVPC.ComputeSubnet {
				verifiedComputeSubnet = true
				break
			}
		}
		if !verifiedComputeSubnet {
			return fmt.Errorf("Could not find the following compute-subnet provided: '%s'",
				args.existingVPC.ComputeSubnet)
		}
	}

	fs.Set("use-existing-vpc", "true")
	flag := fs.Lookup(vpcNameFlag)
	if !flag.Changed {
		fs.Set("vpc-name", args.existingVPC.VPCName)
	}
	flag = fs.Lookup(controlPlaneSubnetFlag)
	if !flag.Changed {
		fs.Set("control-plabe-subnet", args.existingVPC.ControlPlaneSubnet)
	}
	flag = fs.Lookup(computePlaneSubnetFlag)
	if !flag.Changed {
		fs.Set("compute-subnet", args.existingVPC.ComputeSubnet)
	}
	flag = fs.Lookup("vpc-project-id")
	if !flag.Changed {
		fs.Set("vpc-project-id", args.existingVPC.VPCProjectID)
	}

	return nil

}

func promptExistingVPC(fs *pflag.FlagSet, connection *sdk.Connection) error {
	var err error
	if args.provider == "aws" {
		err = promptExistingAWSVPC(fs, connection)
	} else if args.provider == "gcp" {
		err = promptExistingGCPVPC(fs, connection)
	}
	return err
}

func promptClusterPrivacy(fs *pflag.FlagSet) error {
	return arguments.PromptBool(fs, privateFlag)
}

func promptCCS(fs *pflag.FlagSet, presetCCS bool) error {
	var err error
	if !presetCCS {
		err = arguments.PromptBool(fs, "ccs")
	}
	if err != nil {
		return err
	}

	err = arguments.CheckIgnoredCCSFlags(args.ccs, fs)
	if err != nil {
		return err
	}
	return nil
}

func promptAuthentication(fs *pflag.FlagSet, connection *sdk.Connection) error {
	var err error
	if !args.ccs.Enabled {
		return nil
	}
	switch args.provider {
	case c.ProviderAWS:
		err = arguments.PromptString(fs, "aws-account-id")
		if err != nil {
			return err
		}

		err = arguments.PromptString(fs, "aws-access-key-id")
		if err != nil {
			return err
		}

		err = arguments.PromptPassword(fs, "aws-secret-access-key")
		if err != nil {
			return err
		}
	case c.ProviderGCP:
		err = promptGcpAuth(fs, connection)
		if err != nil {
			return err
		}
	}
	return nil
}

func promptGcpAuth(fs *pflag.FlagSet, connection *sdk.Connection) error {
	var err error
	isWif := fs.Changed("wif-config")
	isNonWif := fs.Changed("service-account-file")

	if isWif && isNonWif {
		return fmt.Errorf("can't use both wif-config and GCP service account file at the same time")
	}
	if !isWif && !isNonWif {
		if !args.interactive {
			return fmt.Errorf("either wif-config or GCP service account file must be specified")
		}
		// if the user has not specified the authentication method, we need to ask
		args.gcpAuthentication.Type, err = interactive.GetOption(interactive.Input{
			Question: "Authentication type",
			Help:     "Select the authentication method to use for the GCP cluster",
			Required: true,
			Options:  []string{c.AuthenticationWif, c.AuthenticationKey},
		})
		if err != nil {
			return err
		}
	}

	if args.gcpAuthentication.Type == "" {
		// if the user has not specified the authentication method, we can determine it based on the flags
		if isWif {
			args.gcpAuthentication.Type = c.AuthenticationWif
		} else if isNonWif {
			args.gcpAuthentication.Type = c.AuthenticationKey
		}
	}

	switch args.gcpAuthentication.Type {
	case c.AuthenticationWif:
		err = promptWifConfig(fs, connection)
		if err != nil {
			return err
		}
	case c.AuthenticationKey:
		// TODO: re-prompt when selected file is not readable / invalid JSON
		err = arguments.PromptFilePath(fs, "service-account-file", true)
		if err != nil {
			return err
		}

		if args.gcpServiceAccountFile == "" {
			return fmt.Errorf("a valid GCP service account file must be specified for CCS clusters")
		}
		err = constructGCPCredentials(args.gcpServiceAccountFile, &args.ccs)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unexpected GCP authentication method %q", args.gcpAuthentication.Type)
	}
	return nil
}

func promptWifConfig(fs *pflag.FlagSet, connection *sdk.Connection) error {
	flag := fs.Lookup("wif-config")

	// if the flag was set, validate the value
	if flag.Changed {
		wifKey := flag.Value.String()
		wifConfig, err := provider.GetWifConfig(connection.ClustersMgmt().V1(), wifKey)
		if err != nil {
			return err
		}
		args.gcpAuthentication.Id = wifConfig.ID()
		return nil
	}

	// if the flag was not set, prompt the user
	wifConfigs, err := provider.GetWifConfigs(connection.ClustersMgmt().V1())
	if err != nil {
		return err
	}
	options, err := getWifConfigOptions(wifConfigs)
	if err != nil {
		return err
	}
	err = arguments.PromptOneOf(fs, "wif-config", options)
	if err != nil {
		return err
	}
	args.gcpWifConfig = parseWifConfigOption(args.gcpWifConfig)

	// map wif name to wif id
	wifMapping := map[string]string{}
	for _, wc := range wifConfigs {
		wifMapping[wc.DisplayName()] = wc.ID()
	}
	args.gcpAuthentication.Id = wifMapping[args.gcpWifConfig]
	return nil
}

func promptNetwork(fs *pflag.FlagSet) error {
	for _, flagName := range []string{"machine-cidr", "service-cidr", "pod-cidr"} {
		err := arguments.PromptIPNet(fs, flagName)
		if err != nil {
			return err
		}
	}
	err := arguments.PromptInt(fs, "host-prefix", nil)
	if err != nil {
		return err
	}

	return nil
}

func promptSecureBoot(fs *pflag.FlagSet) error {
	// this is a GCP setting
	if args.provider != c.ProviderGCP {
		return nil
	}
	err := arguments.PromptBool(fs, "secure-boot-for-shielded-vms")
	if err != nil {
		return err
	}

	return nil
}

func promptGcpCustomEncryption(fs *pflag.FlagSet, connection *sdk.Connection) error {

	if !args.ccs.Enabled || args.provider != c.ProviderGCP {
		return nil
	}

	isCustomKeys := args.gcpEncryption.KmsKeyLocation != "" ||
		args.gcpEncryption.KmsKeyRing != "" ||
		args.gcpEncryption.KmsKeyName != "" ||
		args.gcpEncryption.KmsKeySvcAccount != ""

	if !isCustomKeys && args.interactive {
		var err error
		isCustomKeys, err = interactive.GetBool(interactive.Input{
			Question: "Use Custom KMS Keys",
			Help:     "To use custom encryption keys managed via GCP Key management service.",
			Default:  false,
		})
		if err != nil {
			return err
		}
	}

	if !isCustomKeys {
		return nil
	}

	KmsKeyLocationOptions, err := getKmsKeyLocationOptions(connection)
	if err != nil {
		return err
	}

	err = arguments.PromptOrCheckOneOf(fs, KmsKeyLocationFlag, KmsKeyLocationOptions)
	if err != nil {
		return err
	}

	keyringOptions, err := getKmsKeyRingOptions(connection)
	if err != nil {
		return err
	}

	err = arguments.PromptOrCheckOneOf(fs, kmsKeyRingFlag, keyringOptions)
	if err != nil {
		return err
	}

	keysOptions, err := getKmsKeyOptions(connection)
	if err != nil {
		return err
	}

	err = arguments.PromptOrCheckOneOf(fs, kmsKeyNameFlag, keysOptions)
	if err != nil {
		return err
	}

	err = arguments.PromptString(fs, kmsKeySvcAccountFlag)
	if err != nil {
		return err
	}

	return nil
}

func promptEtcdEncryption(fs *pflag.FlagSet) error {
	if !args.fips {
		return arguments.PromptBool(fs, "etcd-encryption")
	}

	if fs.Changed("etcd-encryption") && !args.etcdEncryption {
		return fmt.Errorf("When FIPS mode is enabled, etcd encryption cannot be disabled")
	}

	//if FIPS encrytion is enabled, etcd encryption should be enabled
	args.etcdEncryption = true
	return nil
}

func promptPrivateServiceConnect(fs *pflag.FlagSet) error {
	if args.provider != c.ProviderGCP ||
		!args.existingVPC.Enabled || !args.private {
		return nil
	}

	//if Wif cluster and private is enabled then has to be PSC
	isWif := (args.gcpAuthentication.Type == c.AuthenticationWif)
	isPSC := (args.gcpPrivateSvcConnect.SvcAttachmentSubnet != "") || isWif

	if !isPSC && args.interactive {
		var err error
		isPSC, err = interactive.GetBool(interactive.Input{
			Question: "Enable Private Service Connect",
			Help: "To enable private service connect, you must have a subnet of purpose " +
				"'Private Service Connect' configured in the VPC you want your cluster installed into. ",
			Default: false,
		})
		if err != nil {
			return err
		}
	}
	if isPSC {
		err := arguments.PromptString(fs, pscSubnetFlag)
		if err != nil {
			return err
		}
	}
	if isWif && args.gcpPrivateSvcConnect.SvcAttachmentSubnet == "" {
		return fmt.Errorf(
			"flag '%s' is required when cluster is '%s' and GCP authentication type is %s",
			pscSubnetFlag, privateFlag, c.AuthenticationWif)
	}
	return nil
}

func validateComputeNodes() error {
	min := minComputeNodes(args.ccs.Enabled, args.multiAZ)
	if args.computeNodes < min {
		return fmt.Errorf("Minimum is %d nodes", min)
	}
	if args.multiAZ && args.computeNodes%3 != 0 {
		return fmt.Errorf("Multi-zone clusters require nodes to be multiple of 3")
	}
	return nil
}

func validateAutoscalingMin() error {
	min := minComputeNodes(args.ccs.Enabled, args.multiAZ)

	if args.autoscaling.MinReplicas < min {
		return fmt.Errorf("Minimum is %d nodes", min)
	}

	if args.multiAZ && args.autoscaling.MinReplicas%3 != 0 {
		return fmt.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
	}
	return nil
}

func validateAutoscalingMax() error {
	if args.autoscaling.MinReplicas > args.autoscaling.MaxReplicas {
		return fmt.Errorf("max-replicas must be greater or equal to min-replicas")
	}

	if args.multiAZ && args.autoscaling.MaxReplicas%3 != 0 {
		return fmt.Errorf("Multi AZ clusters require that the number of compute nodes be a multiple of 3")
	}
	return nil
}

func fetchFlavours(client *cmv1.Client) (flavours []*cmv1.Flavour, err error) {
	collection := client.Flavours()
	page := 1
	size := 100
	for {
		var response *cmv1.FlavoursListResponse
		response, err = collection.List().
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return
		}
		flavours = append(flavours, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
}

func constructGCPCredentials(filePath arguments.FilePath, value *c.CCS) error {
	// Open our jsonFile
	jsonFile, err := os.Open(filePath.String())
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &value.GCP)
	if err != nil {
		return err
	}
	return nil
}

func promptAutoscaling(fs *pflag.FlagSet) error {
	err := arguments.PromptBool(fs, "enable-autoscaling")
	if err != nil {
		return err
	}
	if args.autoscaling.Enabled {
		// set default for interactive mode
		if args.interactive && args.autoscaling.MinReplicas == 0 {
			args.autoscaling.MinReplicas = minComputeNodes(args.ccs.Enabled, args.multiAZ)
		}
		err = arguments.PromptInt(fs, "min-replicas", validateAutoscalingMin)
		if err != nil {
			return err
		}

		// set default for interactive mode
		if args.interactive && args.autoscaling.MaxReplicas == 0 {
			args.autoscaling.MaxReplicas = args.autoscaling.MinReplicas
		}
		err = arguments.PromptInt(fs, "max-replicas", validateAutoscalingMax)
		if err != nil {
			return err
		}

	}
	return nil
}

// getVersionFilters returns a version filter based on the current args
func getVersionFilters() string {
	filter := ""
	// WIF filter
	if args.gcpAuthentication.Type == c.AuthenticationWif {
		filter = fmt.Sprintf("%s AND wif_enabled = 'true'", filter)
	}
	return filter
}
