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

package create

import (
	"fmt"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	clusterpkg "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/config"
)

var args struct {
	parameter         []string
	header            []string
	region            string
	version           string
	flavour           string
	provider          string
	expirationTime    string
	expirationSeconds time.Duration
	private           bool
}

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "create [flags] <cluster name>",
	Short: "Create managed clusters",
	Long:  "Create managed OpenShift Dedicated v4 clusters via OCM",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	fs := Cmd.Flags()
	arguments.AddParameterFlag(fs, &args.parameter)
	arguments.AddHeaderFlag(fs, &args.header)
	fs.StringVar(
		&args.region,
		"region",
		"us-east-1",
		"The cloud provider region to create the cluster in",
	)
	fs.StringVar(
		&args.version,
		"version",
		"",
		"The OpenShift version to create the cluster at (for example, \"4.1.16\")",
	)
	fs.StringVar(
		&args.flavour,
		"flavour",
		"osd-4",
		"The OCM flavour to create the cluster with",
	)
	fs.StringVar(
		&args.provider,
		"provider",
		"aws",
		"The cloud provider to create the cluster on",
	)
	fs.StringVar(
		&args.expirationTime,
		"expiration-time",
		args.expirationTime,
		"Specified time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	fs.DurationVar(
		&args.expirationSeconds,
		"expiration",
		args.expirationSeconds,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	fs.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint and application routes to direct, private connectivity.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	var err error
	var expiration time.Time

	// Validate options
	if len(args.expirationTime) > 0 && args.expirationSeconds != 0 {
		return fmt.Errorf("at most one of `expiration-time` or `expiration` may be specified")
	}
	if args.region == "us-east-1" && args.provider != "aws" {
		return fmt.Errorf("if specifying a non-aws cloud provider, region must be set to a valid region")
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}
	if cfg == nil {
		return fmt.Errorf("Not logged in, run the 'login' command")
	}

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		return fmt.Errorf("Can't check if tokens have expired: %v", err)
	}
	if !armed {
		return fmt.Errorf("Tokens have expired, run the 'login' command")
	}

	// Create the connection, and remember to close it:
	connection, err := cfg.Connection()
	if err != nil {
		return fmt.Errorf("Can't create connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the cluster management api
	cmv1Client := connection.ClustersMgmt().V1()

	// Check and set the cluster name
	if len(argv) != 1 || argv[0] == "" {
		return fmt.Errorf("A cluster name must be specified")
	}
	clusterName := argv[0]

	// Parse the expiration options
	if len(args.expirationTime) > 0 {
		t, err := parseRFC3339(args.expirationTime)
		if err != nil {
			return fmt.Errorf("unable to parse expiration time: %s", err)
		}

		expiration = t
	}
	if args.expirationSeconds != 0 {
		// round up to the nearest second
		expiration = time.Now().Add(args.expirationSeconds).Round(time.Second)
	}

	// Retrieve valid/default versions
	versionList := sets.NewString()
	var defaultVersion string
	versions, err := fetchEnabledVersions(cmv1Client)
	if err != nil {
		return fmt.Errorf("unable to retrieve versions: %s", err)
	}
	for _, version := range versions {
		versionList.Insert(version.ID())
		if version.Default() {
			defaultVersion = version.ID()
		}
	}

	// Check and set the cluster version
	var clusterVersion string
	if args.version != "" {
		if !versionList.Has("openshift-v" + args.version) {
			return fmt.Errorf("A valid version number must be specified\nValid versions: %+v", versionList.List())
		}
		clusterVersion = "openshift-v" + args.version
	} else {
		clusterVersion = defaultVersion
	}

	// Retrieve valid/default flavours
	flavourList := sets.NewString()
	flavours, err := fetchFlavours(cmv1Client)
	if err != nil {
		return fmt.Errorf("unable to retrieve flavours: %s", err)
	}
	for _, flavour := range flavours {
		flavourList.Insert(flavour.ID())
	}

	// Check and set the cluster flavour
	var clusterFlavour string
	if args.flavour != "" {

		if !flavourList.Has(args.flavour) {
			return fmt.Errorf("A valid flavour number must be specified\nValid flavours: %+v", flavourList.List())
		}
		clusterFlavour = args.flavour
	}

	clusterBuild := cmv1.NewCluster().
		Name(clusterName).
		Flavour(
			cmv1.NewFlavour().
				ID(clusterFlavour),
		).
		CloudProvider(
			cmv1.NewCloudProvider().
				ID(args.provider),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(args.region),
		).
		Version(
			cmv1.NewVersion().
				ID(clusterVersion),
		)
	if !expiration.IsZero() {
		clusterBuild = clusterBuild.ExpirationTimestamp(
			expiration,
		)
	}
	if args.private {
		clusterBuild = clusterBuild.API(
			cmv1.NewClusterAPI().
				Listening(cmv1.ListeningMethodInternal),
		)
	} else {
		clusterBuild = clusterBuild.API(
			cmv1.NewClusterAPI().
				Listening(cmv1.ListeningMethodExternal),
		)
	}
	cluster, err := clusterBuild.Build()
	if err != nil {
		return fmt.Errorf("unable to build cluster object: %v", err)
	}

	// Send a request to create the cluster:
	response, err := cmv1Client.Clusters().Add().
		Body(cluster).
		Send()
	if err != nil {
		return fmt.Errorf("unable to create cluster: %v", err)
	}

	// Print the result:
	cluster = response.Body()

	err = clusterpkg.PrintClusterDesctipion(connection, cluster)
	if err != nil {
		return err
	}

	return nil
}

func fetchEnabledVersions(client *cmv1.Client) (versions []*cmv1.Version, err error) {
	collection := client.Versions()
	page := 1
	size := 100
	for {
		var response *cmv1.VersionsListResponse
		response, err = collection.List().
			Search("enabled = 'true'").
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return
		}
		versions = append(versions, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}
	return
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

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
