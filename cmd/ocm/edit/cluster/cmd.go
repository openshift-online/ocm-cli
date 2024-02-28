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
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var args struct {
	// Basic options
	expirationTime     string
	expirationDuration time.Duration

	// Networking options
	private bool

	channelGroup string

	clusterWideProxy c.ClusterWideProxy
}

var Cmd = &cobra.Command{
	Use:   "cluster [flags] {NAME|ID|EXTERNAL_ID}",
	Short: "Edit cluster",
	Long:  "Edit cluster.",
	Example: `  # Edit a cluster named "mycluster" to make it private
  ocm edit cluster mycluster --private`,
	RunE: run,
	Args: cobra.MinimumNArgs(1),
}

func init() {
	flags := Cmd.Flags()

	// Basic options
	flags.StringVar(
		&args.expirationTime,
		"expiration-time",
		"",
		"Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	flags.DurationVar(
		&args.expirationDuration,
		"expiration",
		0,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	// Cluster expiration is not supported in production
	//nolint:gosec
	flags.MarkHidden("expiration-time")
	//nolint:gosec
	flags.MarkHidden("expiration")

	//Networking options
	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint to direct, private connectivity.",
	)

	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		"",
		"The channel group which the cluster version belongs to.",
	)

	args.clusterWideProxy.HTTPProxy = new(string)
	flags.StringVar(
		args.clusterWideProxy.HTTPProxy,
		"http-proxy",
		"",
		"A proxy URL to use for creating HTTP connections outside the cluster.",
	)

	args.clusterWideProxy.HTTPSProxy = new(string)
	flags.StringVar(
		args.clusterWideProxy.HTTPSProxy,
		"https-proxy",
		"",
		"A proxy URL to use for creating HTTPS connections outside the cluster.",
	)

	args.clusterWideProxy.NoProxy = new(string)
	flags.StringVar(
		args.clusterWideProxy.NoProxy,
		"no-proxy",
		"",
		"A comma-separated list of destination domain names, domains, IP addresses or other network CIDRs to "+
			"exclude proxying.",
	)

	args.clusterWideProxy.AdditionalTrustBundleFile = new(string)
	flags.StringVar(
		args.clusterWideProxy.AdditionalTrustBundleFile,
		"additional-trust-bundle-file",
		"",
		"A file contains a PEM-encoded X.509 certificate bundle that will be "+
			"added to the nodes' trusted certificate store.")

}

func isGCPNetworkEmpty(network *cmv1.GCPNetwork) bool {
	return network != nil && network.VPCName() == "" &&
		network.ControlPlaneSubnet() == "" && network.ComputeSubnet() == ""
}

func wasClusterWideProxyReceived(httpProxy, httpsProxy, noProxy, additionalTrustBundleFile *string) bool {
	return ((httpProxy != nil && *httpProxy != "") || (httpsProxy != nil && *httpsProxy != "") ||
		(noProxy != nil && *noProxy != "") || (additionalTrustBundleFile != nil && *additionalTrustBundleFile != ""))
}

func run(cmd *cobra.Command, argv []string) error {
	// Check that there is exactly one cluster name, identifier or external identifier in the
	// command line arguments:
	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one cluster name, identifier or external identifier " +
				"is required",
		)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := argv[0]
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the cluster management api
	clusterCollection := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	// Validate flags:
	expiration, err := c.ValidateClusterExpiration(args.expirationTime, args.expirationDuration)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s", err))
	}

	var private *bool
	if cmd.Flags().Changed("private") {
		private = &args.private
	}

	var channelGroup string
	if cmd.Flags().Changed("channel-group") {
		channelGroup = args.channelGroup
	}

	var httpProxy *string
	if cmd.Flags().Changed("http-proxy") {
		if *args.clusterWideProxy.HTTPProxy != "" {
			err := utils.ValidateHTTPProxy(*args.clusterWideProxy.HTTPProxy)
			if err != nil {
				return err
			}
		}
		httpProxy = args.clusterWideProxy.HTTPProxy
	}

	var httpsProxy *string
	if cmd.Flags().Changed("https-proxy") {
		if *args.clusterWideProxy.HTTPSProxy != "" {
			err := utils.IsURL(*args.clusterWideProxy.HTTPSProxy)
			if err != nil {
				return fmt.Errorf("Invalid 'proxy.https_proxy' attribute '%s'", *args.clusterWideProxy.HTTPSProxy)
			}
		}
		httpsProxy = args.clusterWideProxy.HTTPSProxy
	}

	var noProxy *string
	if cmd.Flags().Changed("no-proxy") {
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
		}
		noProxy = args.clusterWideProxy.NoProxy
	}

	var additionalTrustBundleFile *string
	var additionalTrustBundleFileValue string
	if cmd.Flags().Changed("additional-trust-bundle-file") {
		additionalTrustBundleFileValue = *args.clusterWideProxy.AdditionalTrustBundleFile
		if additionalTrustBundleFileValue != "" {
			err := utils.ValidateAdditionalTrustBundle(additionalTrustBundleFileValue)
			if err != nil {
				return err
			}
		}
		additionalTrustBundleFile = &additionalTrustBundleFileValue
	}

	if len(cluster.AWS().SubnetIDs()) == 0 && isGCPNetworkEmpty(cluster.GCPNetwork()) &&
		wasClusterWideProxyReceived(httpProxy, httpsProxy, noProxy, additionalTrustBundleFile) {
		return fmt.Errorf("Cluster-wide proxy is not supported on clusters using the default VPC")
	}

	clusterConfig := c.Spec{
		Expiration:   expiration,
		Private:      private,
		ChannelGroup: channelGroup,
	}

	clusterWideProxy := c.ClusterWideProxy{
		HTTPProxy:                 httpProxy,
		HTTPSProxy:                httpsProxy,
		NoProxy:                   noProxy,
		AdditionalTrustBundleFile: additionalTrustBundleFile,
	}

	if clusterWideProxy.AdditionalTrustBundleFile != nil {
		if len(*additionalTrustBundleFile) > 0 {
			cert, err := os.ReadFile(*additionalTrustBundleFile)
			if err != nil {
				return fmt.Errorf("Failed to read additional trust bundle file: %s", err)
			}
			clusterWideProxy.AdditionalTrustBundle = new(string)
			*clusterWideProxy.AdditionalTrustBundle = string(cert)
		} else {
			clusterWideProxy.AdditionalTrustBundle = new(string)
			*clusterWideProxy.AdditionalTrustBundle = ""
		}
	}
	clusterConfig.ClusterWideProxy = clusterWideProxy

	err = c.UpdateCluster(clusterCollection, cluster.ID(), clusterConfig)
	if err != nil {
		return fmt.Errorf("Failed to update cluster: %v", err)
	}

	return nil

}
