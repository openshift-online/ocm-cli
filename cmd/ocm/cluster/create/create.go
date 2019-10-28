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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	clusterpkg "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/flags"
)

var args struct {
	parameter []string
	header    []string
	region    string
	version   string
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
	flags.AddParameterFlag(fs, &args.parameter)
	flags.AddHeaderFlag(fs, &args.header)
	fs.StringVar(
		&args.region,
		"region",
		"us-east-1",
		"The AWS region to create the cluster in",
	)
	fs.StringVar(
		&args.version,
		"version",
		"",
		"The OpenShift version to create the cluster at (for example, \"4.1.16\")",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	var err error

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

	cluster, err := cmv1.NewCluster().
		Name(clusterName).
		Flavour(
			cmv1.NewFlavour().
				ID("osd-4"),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(args.region),
		).
		Version(
			cmv1.NewVersion().
				ID(clusterVersion),
		).
		Build()
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
