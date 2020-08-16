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
	"net"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
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
	multiAZ           bool

	// Scaling options
	computeMachineType string
	computeNodes       int

	// Networking options
	hostPrefix  int
	machineCIDR net.IPNet
	serviceCIDR net.IPNet
	podCIDR     net.IPNet
}

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "cluster",
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
	fs.BoolVar(
		&args.multiAZ,
		"multi-az",
		false,
		"Deploy to multiple data centers.",
	)
	// Scaling options
	fs.StringVar(
		&args.computeMachineType,
		"compute-machine-type",
		"",
		"Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.",
	)
	fs.IntVar(
		&args.computeNodes,
		"compute-nodes",
		4,
		"Number of worker nodes to provision per zone. Single zone clusters need at least 4 nodes, "+
			"multizone clusters need at least 9 nodes.",
	)

	fs.IPNetVar(
		&args.machineCIDR,
		"machine-cidr",
		net.IPNet{},
		"Block of IP addresses used by OpenShift while installing the cluster, for example \"10.0.0.0/16\".",
	)
	fs.IPNetVar(
		&args.serviceCIDR,
		"service-cidr",
		net.IPNet{},
		"Block of IP addresses for services, for example \"172.30.0.0/16\".",
	)
	fs.IPNetVar(
		&args.podCIDR,
		"pod-cidr",
		net.IPNet{},
		"Block of IP addresses from which Pod IP addresses are allocated, for example \"10.128.0.0/14\".",
	)
	fs.IntVar(
		&args.hostPrefix,
		"host-prefix",
		0,
		"Subnet prefix length to assign to each individual node. For example, if host prefix is set "+
			"to \"23\", then each node is assigned a /23 subnet out of the given CIDR.",
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

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
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

	if args.private && args.provider != "aws" {
		return fmt.Errorf("Setting cluster as private is not supported for cloud provider '%s'", args.provider)
	}

	// Compute node instance type:
	computeMachineType := args.computeMachineType

	computeMachineType, err = validateMachineType(cmv1Client, args.provider, computeMachineType)
	if err != nil {
		return fmt.Errorf("Expected a valid machine type: %s", err)
	}

	// Compute nodes:
	computeNodes := args.computeNodes
	// Compute node requirements for multi-AZ clusters are higher
	if args.multiAZ && !cmd.Flags().Changed("compute-nodes") {
		computeNodes = 9
	}

	clusterConfig := c.Spec{
		Name:               clusterName,
		Region:             args.region,
		Provider:           args.provider,
		Flavour:            clusterFlavour,
		MultiAZ:            args.multiAZ,
		Version:            clusterVersion,
		Expiration:         expiration,
		ComputeMachineType: computeMachineType,
		ComputeNodes:       computeNodes,
		MachineCIDR:        args.machineCIDR,
		ServiceCIDR:        args.serviceCIDR,
		PodCIDR:            args.podCIDR,
		HostPrefix:         args.hostPrefix,
		Private:            &args.private,
	}

	cluster, err := c.CreateCluster(cmv1Client, clusterConfig)
	if err != nil {
		return fmt.Errorf("Failed to create cluster: %v", err)
	}

	err = c.PrintClusterDesctipion(connection, cluster)
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

func validateMachineType(client *cmv1.Client, provider string, machineType string) (string, error) {
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
	machineTypes, err := c.GetMachineTypes(client, provider)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve machine types: %s", err)
		return
	}

	for _, v := range machineTypes {
		machineTypeList = append(machineTypeList, v.ID())
	}

	return
}
