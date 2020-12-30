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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/provider"
)

var args struct {
	// positional args
	clusterName string

	// flags
	interactive bool
	dryRun      bool

	region                string
	version               string
	flavour               string
	provider              string
	expirationTime        string
	expirationSeconds     time.Duration
	private               bool
	multiAZ               bool
	ccs                   c.CCS
	gcpServiceAccountFile arguments.FilePath

	// Scaling options
	computeMachineType string
	computeNodes       int

	// Networking options
	hostPrefix  int
	machineCIDR net.IPNet
	serviceCIDR net.IPNet
	podCIDR     net.IPNet
}

const clusterNameHelp = "will be used when generating a sub-domain for your cluster on openshiftapps.com."

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "cluster [flags] NAME",
	Short: "Create managed clusters",
	Long: fmt.Sprintf("Create managed OpenShift Dedicated v4 clusters via OCM.\n"+
		"\n"+
		"NAME %s", clusterNameHelp),
	RunE: run,
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
	arguments.AddCCSFlags(fs, &args.ccs)

	fs.Var(
		&args.gcpServiceAccountFile,
		"service-account-file",
		"GCP service account JSON file path.",
	)
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
	arguments.SetQuestion(fs, "version", "OpenShift version:")
	fs.StringVar(
		&args.flavour,
		"flavour",
		"osd-4",
		"The OCM flavour to create the cluster with",
	)
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
	arguments.SetQuestion(fs, "multi-az", "Multiple AZ:")
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
}

func osdProviders() []string {
	return []string{c.ProviderAWS, c.ProviderGCP}
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

func run(cmd *cobra.Command, argv []string) error {
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

	// Get the client for the cluster management api
	cmv1Client := connection.ClustersMgmt().V1()

	// Validate flags / ask for missing data.
	fs := cmd.Flags()

	// Only offer the 2 providers known to support OSD now;
	// but don't validate if set, to not block `ocm` CLI from creating clusters on future providers.
	err = arguments.PromptOneOf(fs, "provider", osdProviders())
	if err != nil {
		return err
	}

	err = promptCCS(fs)
	if err != nil {
		return err
	}

	err = arguments.PromptBool(fs, "multi-az")
	if err != nil {
		return err
	}

	regionSet := sets.NewString()
	regions, err := provider.GetRegions(connection.ClustersMgmt().V1(), args.provider, args.ccs)
	if err != nil {
		return err
	}
	for _, region := range regions {
		// `enabled` flag only affects Red Hat infra. All regions enabled on CCS.
		if args.ccs.Enabled || region.Enabled() {
			regionSet.Insert(region.ID())
		}
	}
	err = arguments.PromptOrCheckOneOf(fs, "region", regionSet.List())
	if err != nil {
		return err
	}

	// TODO: with --interactive GCP if you simply press Enter without pressing Down/Up,
	// the value stays "us-east-1" and errors out.
	if args.region == "us-east-1" && args.provider != c.ProviderAWS {
		return fmt.Errorf("if specifying a non-aws cloud provider, region must be set to a valid region")
	}

	expiration, err := c.ValidateClusterExpiration(args.expirationTime, args.expirationSeconds)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("%s", err))
	}

	// Check and set the cluster version
	versionList, defaultVersion, err := c.GetEnabledVersions(cmv1Client)
	if err != nil {
		return fmt.Errorf("unable to retrieve versions: %s", err)
	}

	if args.version == "" {
		args.version = defaultVersion
	}
	args.version = c.DropOpenshiftVPrefix(args.version)
	err = arguments.PromptOrCheckOneOf(fs, "version", versionList)
	if err != nil {
		return err
	}
	clusterVersion := c.EnsureOpenshiftVPrefix(args.version)

	// Retrieve valid flavours
	flavourList, err := getFlavourIDs(cmv1Client)
	if err != nil {
		return err
	}
	err = arguments.CheckOneOf(fs, "flavour", flavourList)
	if err != nil {
		return err
	}

	if args.private && args.provider != c.ProviderAWS {
		return fmt.Errorf("Setting cluster as private is not supported for cloud provider '%s'", args.provider)
	}

	// Compute node instance type:
	machineTypeList, err := provider.GetMachineTypeIDs(cmv1Client, args.provider)
	if err != nil {
		return err
	}
	err = arguments.PromptOrCheckOneOf(fs, "compute-machine-type", machineTypeList)
	if err != nil {
		return err
	}

	// Default compute nodes:
	if args.computeNodes == 0 {
		args.computeNodes = minComputeNodes(args.ccs.Enabled, args.multiAZ)
	}
	err = arguments.PromptInt(fs, "compute-nodes", validateComputeNodes)
	if err != nil {
		return err
	}

	err = promptNetwork(fs)
	if err != nil {
		return err
	}

	clusterConfig := c.Spec{
		Name:               args.clusterName,
		Region:             args.region,
		Provider:           args.provider,
		CCS:                args.ccs,
		Flavour:            args.flavour,
		MultiAZ:            args.multiAZ,
		Version:            clusterVersion,
		Expiration:         expiration,
		ComputeMachineType: args.computeMachineType,
		ComputeNodes:       args.computeNodes,
		MachineCIDR:        args.machineCIDR,
		ServiceCIDR:        args.serviceCIDR,
		PodCIDR:            args.podCIDR,
		HostPrefix:         args.hostPrefix,
		Private:            &args.private,
	}

	cluster, err := c.CreateCluster(cmv1Client, clusterConfig, args.dryRun)
	if err != nil {
		return fmt.Errorf("Failed to create cluster: %v", err)
	}

	// Print the result:
	if cluster == nil {
		if args.dryRun {
			fmt.Println("dry run: Would be successful.")
		}
	} else {
		err = c.PrintClusterDesctipion(connection, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

// promptName checks and/or reads the cluster name
func promptName(argv []string) error {
	if len(argv) == 1 && argv[0] != "" {
		args.clusterName = argv[0]
		return nil
	}

	if args.interactive {
		prompt := &survey.Input{
			Message: "cluster name",
			Help:    clusterNameHelp,
		}
		return survey.AskOne(prompt, &args.clusterName, survey.WithValidator(survey.Required))
	}

	return fmt.Errorf("A cluster name must be specified")
}

func promptCCS(fs *pflag.FlagSet) error {
	err := arguments.PromptBool(fs, "ccs")
	if err != nil {
		return err
	}
	if args.ccs.Enabled {
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
			// TODO: re-prompt when selected file is not readable / invalid JSON
			err = arguments.PromptFilePath(fs, "service-account-file")
			if err != nil {
				return err
			}

			if args.gcpServiceAccountFile != "" {
				err = constructGCPCredentials(args.gcpServiceAccountFile, &args.ccs)
				if err != nil {
					return err
				}
			}
		}
	}
	err = arguments.CheckIgnoredCCSFlags(args.ccs)
	if err != nil {
		return err
	}
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

func getFlavourIDs(client *cmv1.Client) (names []string, err error) {
	flavourSet := sets.NewString()
	flavours, err := fetchFlavours(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve flavours: %s", err)
	}
	for _, flavour := range flavours {
		flavourSet.Insert(flavour.ID())
	}
	return flavourSet.List(), nil
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
	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &value.GCP)
	if err != nil {
		return err
	}
	return nil
}
