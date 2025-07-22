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

package machinepool

import (
	"fmt"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

type Args struct {
	Argv                       []string
	ClusterKey                 string
	InstanceType               string
	Replicas                   int
	Autoscaling                c.Autoscaling
	Labels                     string
	Taints                     string
	AdditionalSecurityGroupIds []string
	AvailabilityZone           string
	SecureBoot                 bool
}

var args Args

const (
	additionalSecurityGroupIdsFlag = "additional-security-group-ids"
	secureBootForShieldedVmsFlag   = "secure-boot-for-shielded-vms"
)

var Cmd = &cobra.Command{
	Use:     "machinepool --cluster={NAME|ID|EXTERNAL_ID} --instance-type=TYPE --replicas=N [flags] MACHINE_POOL_ID",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Add machine pool to cluster",
	Long:    "Add a machine pool to the cluster.",
	Example: `  # Add a machine pool mp-1 with 3 replicas and m5.xlarge instance type to a cluster
  ocm create machinepool --cluster mycluster --instance-type m5.xlarge --replicas 3 mp-1
  # Add a machine pool mp-1 with autoscaling enabled and 3 to 6 replicas of m5.xlarge to a cluster
  ocm create machinepool --cluster=mycluster --enable-autoscaling \
  --min-replicas=3 --max-replicas=6 --instance-type=m5.xlarge mp-1 
  # Add a machine pool mp-1 with labels and m5.xlarge instance type to a cluster
  ocm create machinepool --cluster mycluster --instance-type m5.xlarge --replicas 3 --labels "foo=bar,bar=baz" mp-1
  # Add a machine pool mp-1 with taints and m5.xlarge instance type to a cluster
  ocm create machinepool --cluster mycluster --instance-type m5.xlarge --replicas 3 --taints "foo=bar:NoSchedule" mp-1`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.ClusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to add the machine pool to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.InstanceType,
		"instance-type",
		"",
		"Instance type that should be used.",
	)

	//nolint:gosec
	Cmd.MarkFlagRequired("instance-type")

	flags.IntVar(
		&args.Replicas,
		"replicas",
		0,
		"Count of machines for this machine pool.",
	)

	arguments.AddAutoscalingFlags(flags, &args.Autoscaling)

	flags.StringVar(
		&args.Labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.Taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:scheduleType'. "+
			"This list will overwrite any modifications made to Node taints on an ongoing basis.",
	)

	flags.StringSliceVar(&args.AdditionalSecurityGroupIds,
		additionalSecurityGroupIdsFlag,
		nil,
		"The additional Security Group IDs to be added to the machine pool. "+
			"Format should be a comma-separated list.",
	)

	flags.StringVar(
		&args.AvailabilityZone,
		"availability-zone",
		"",
		"Select availability zone to create a single AZ machine pool for a multi-AZ cluster",
	)

	flags.BoolVar(
		&args.SecureBoot,
		secureBootForShieldedVmsFlag,
		false,
		"Secure Boot enables the use of Shielded VMs in the Google Cloud Platform for the instances in this machine pool. "+
			"This will override the cluster level configuration of secure boot.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	args.Argv = argv
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	cluster, err := getCluster(connection, args.ClusterKey)
	if err != nil {
		return err
	}

	if err := VerifyCluster(cluster); err != nil {
		return err
	}

	if err := VerifyArguments(
		args,
		&flagSet{cmd.Flags()},
		&machineTypeListGetter{connection},
		cluster,
	); err != nil {
		return err
	}

	machinePoolId := argv[0]

	machinePool, err := buildMachinePool(machinePoolId, &flagSet{cmd.Flags()})
	if err != nil {
		return err
	}

	if err := addMachinePoolToCluster(connection, cluster.Id(), machinePool); err != nil {
		return err
	}

	fmt.Printf("Machine pool '%s' created on cluster '%s'\n", machinePoolId, args.ClusterKey)
	return nil
}

func getCluster(
	connection *sdk.Connection,
	clusterKey string,
) (ocm.Cluster, error) {
	clusterData, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}
	cluster := ocm.NewCluster(clusterData)
	return cluster, nil
}

func VerifyCluster(cluster ocm.Cluster) error {
	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", args.ClusterKey)
	}
	return nil
}

func VerifyArguments(
	args Args,
	flags FlagSet,
	machineTypeListGetter MachineTypeListGetter,
	cluster ocm.Cluster,
) error {
	if len(args.Argv) < 1 || args.Argv[0] == "" {
		return fmt.Errorf("Missing machine pool ID")
	}

	if args.Labels != "" {
		for _, label := range strings.Split(args.Labels, ",") {
			if !strings.Contains(label, "=") {
				return fmt.Errorf("Expected key=value format for label-match")
			}
		}
	}

	if args.Taints != "" {
		for _, taint := range strings.Split(args.Taints, ",") {
			if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
				return fmt.Errorf("Expected key=value:scheduleType format for taints")
			}
		}
	}

	isMinReplicasSet := flags.Changed("min-replicas")
	isMaxReplicasSet := flags.Changed("max-replicas")
	isReplicasSet := flags.Changed("replicas")

	if args.Autoscaling.Enabled {
		if isReplicasSet {
			return fmt.Errorf("--replicas is only allowed when --enable-autoscaling=false")
		}

		if !isMaxReplicasSet || !isMinReplicasSet {
			return fmt.Errorf("Both --min-replicas and --max-replicas are required when --enable-autoscaling=true")
		}
	} else {
		if !isReplicasSet {
			return fmt.Errorf("--replicas is required when --enable-autoscaling=false")
		}

		if isMaxReplicasSet || isMinReplicasSet {
			return fmt.Errorf("--min-replicas and --max-replicas are not allowed when --enable-autoscaling=false")
		}
	}

	machineTypeList, err := machineTypeListGetter.GetMachineTypeOptions(cluster)
	if err != nil {
		return err
	}
	if err := flags.CheckOneOf("instance-type", machineTypeList); err != nil {
		return err
	}

	if len(args.AdditionalSecurityGroupIds) != 0 && cluster.CloudProviderId() != c.ProviderAWS {
		return fmt.Errorf("'%s' may only be set for clusters using the '%s' cloud provider.",
			additionalSecurityGroupIdsFlag, c.ProviderAWS)
	}

	if args.AvailabilityZone != "" {
		if cluster.CloudProviderId() != c.ProviderGCP {
			return fmt.Errorf(
				"At this time, OCM CLI does not support setting 'availability-zone'"+
					" for clusters using the cloud provider '%s'",
				cluster.CloudProviderId())
		}
	}

	isSecureBootSet := flags.Changed(secureBootForShieldedVmsFlag)
	if isSecureBootSet && cluster.CloudProviderId() != c.ProviderGCP {
		return fmt.Errorf(
			"--secure-boot-for-shielded-vms is only supported for clusters using the '%s' cloud provider",
			c.ProviderGCP)
	}

	return nil
}

func buildMachinePool(
	machinePoolId string,
	flags FlagSet,
) (*cmv1.MachinePool, error) {
	labels := make(map[string]string)
	if args.Labels != "" {
		for _, label := range strings.Split(args.Labels, ",") {
			tokens := strings.Split(label, "=")
			labels[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}

	taintBuilders := []*cmv1.TaintBuilder{}
	if args.Taints != "" {
		for _, taint := range strings.Split(args.Taints, ",") {
			tokens := strings.FieldsFunc(taint, arguments.Split)
			taintBuilders = append(taintBuilders, cmv1.NewTaint().Key(tokens[0]).Value(tokens[1]).Effect(tokens[2]))
		}
	}
	mpBuilder := cmv1.NewMachinePool().
		ID(machinePoolId).
		InstanceType(args.InstanceType).
		Labels(labels).
		Taints(taintBuilders...)

	if len(args.AdditionalSecurityGroupIds) != 0 {
		for i, sg := range args.AdditionalSecurityGroupIds {
			args.AdditionalSecurityGroupIds[i] = strings.TrimSpace(sg)
		}
		mpBuilder.AWS(
			cmv1.NewAWSMachinePool().
				AdditionalSecurityGroupIds(args.AdditionalSecurityGroupIds...))
	}

	if flags.Changed(secureBootForShieldedVmsFlag) {
		mpBuilder.GCP(
			cmv1.NewGCPMachinePool().
				SecureBoot(args.SecureBoot),
		)
	}

	if args.Autoscaling.Enabled {
		mpBuilder = mpBuilder.Autoscaling(
			cmv1.NewMachinePoolAutoscaling().
				MinReplicas(args.Autoscaling.MinReplicas).
				MaxReplicas(args.Autoscaling.MaxReplicas))
	} else {
		mpBuilder = mpBuilder.Replicas(args.Replicas)
	}

	if args.AvailabilityZone != "" {
		mpBuilder = mpBuilder.AvailabilityZones(args.AvailabilityZone)
	}

	machinePool, err := mpBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create machine pool for cluster '%s': %v", args.ClusterKey, err)
	}
	return machinePool, nil
}

func addMachinePoolToCluster(
	connection *sdk.Connection,
	clusterId string,
	machinePool *cmv1.MachinePool,
) error {
	if _, err := connection.ClustersMgmt().V1().Clusters().Cluster(clusterId).
		MachinePools().
		Add().
		Body(machinePool).
		Send(); err != nil {
		return fmt.Errorf("Failed to add machine pool to cluster '%s': %v", args.ClusterKey, err)
	}
	return nil
}
