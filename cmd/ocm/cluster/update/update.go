/*
Copyright (c) 2026 Red Hat, Inc.

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

package update

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

const (
	defaultScheduleDelay = 6 * time.Minute
	minScheduleDelay     = 5 * time.Minute
	policySubmitTimeout  = 30 * time.Second
)

var args struct {
	interactive bool
	channel     string
	version     string
	nextRun     string
}

var Cmd = &cobra.Command{
	Use:     "update [flags] CLUSTER",
	Aliases: []string{"update-version"},
	Short:   "Update a cluster to an available version",
	Long:    "Update a cluster by creating a manual update policy for one of its available versions.",
	Example: `  # Update a cluster to a specific version
  ocm cluster update my-cluster --version 4.22.18

  # Change channels before choosing an update version
  ocm cluster update my-cluster --channel candidate-5.0 --version 5.0.0-rc.6

  # Choose from the cluster's available versions interactively
  ocm cluster update my-cluster --interactive`,
	Args: cobra.ExactArgs(1),
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	arguments.AddInteractiveFlag(flags, &args.interactive)

	flags.StringVar(
		&args.channel,
		"channel",
		"",
		"Update channel to use before selecting the update version.",
	)
	arguments.SetQuestion(flags, "channel", "Update channel:")

	flags.StringVar(
		&args.version,
		"version",
		"",
		"OpenShift version to update the cluster to.",
	)
	arguments.SetQuestion(flags, "version", "OpenShift version:")

	flags.StringVar(
		&args.nextRun,
		"next-run",
		"",
		"UTC time to run the update in RFC3339 format. Defaults to six minutes from now.",
	)
}

func run(cmd *cobra.Command, argv []string) (err error) {
	clusterKey := argv[0]
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer func() {
		closeErr := connection.Close()
		if closeErr == nil {
			return
		}
		if err != nil {
			err = fmt.Errorf("%v; failed to close OCM connection: %v", err, closeErr)
			return
		}
		fmt.Fprintf(os.Stderr, "Warning: Failed to close OCM connection: %v\n", closeErr)
	}()

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	cluster, err = patchChannel(cmd, connection, cluster)
	if err != nil {
		return err
	}

	version := cluster.Version()
	if version == nil {
		return fmt.Errorf("There are no available updates for cluster '%s'", clusterKey)
	}
	availableUpdates := c.GetAvailableUpgrades(version)
	if len(availableUpdates) == 0 {
		return fmt.Errorf("There are no available updates for cluster '%s'", clusterKey)
	}

	options := make([]arguments.Option, 0, len(availableUpdates))
	for _, update := range availableUpdates {
		options = append(options, arguments.Option{Value: update})
	}

	err = arguments.PromptOrCheckOneOf(cmd.Flags(), "version", options)
	if err != nil {
		return err
	}
	if args.version == "" {
		return fmt.Errorf("A --version must be specified.\nValid options: %v", availableUpdates)
	}

	nextRun, err := parseNextRun(args.nextRun, time.Now().UTC())
	if err != nil {
		return err
	}

	upgradePolicy, err := cmv1.NewUpgradePolicy().
		ScheduleType("manual").
		Version(args.version).
		NextRun(nextRun).
		Build()
	if err != nil {
		return fmt.Errorf("Failed to create update policy for cluster '%s': %v", clusterKey, err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), policySubmitTimeout)
	defer cancel()
	_, err = connection.ClustersMgmt().V1().
		Clusters().
		Cluster(cluster.ID()).
		UpgradePolicies().
		Add().
		Body(upgradePolicy).
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("Failed to schedule update for cluster '%s': %v", clusterKey, err)
	}

	fmt.Printf("Scheduled update for cluster '%s' to version '%s' at '%s'\n",
		clusterKey, args.version, nextRun.Format(time.RFC3339))
	return nil
}

func patchChannel(cmd *cobra.Command, connection *sdk.Connection, cluster *cmv1.Cluster) (*cmv1.Cluster, error) {
	version := cluster.Version()
	if version == nil {
		if args.channel == "" {
			return cluster, nil
		}
		return nil, fmt.Errorf("There are no available channels for cluster '%s'", cluster.ID())
	}
	availableChannels := version.AvailableChannels()
	if len(availableChannels) == 0 {
		if args.channel == "" {
			return cluster, nil
		}
		return nil, fmt.Errorf("There are no available channels for cluster '%s'", cluster.ID())
	}

	options := make([]arguments.Option, 0, len(availableChannels))
	for _, channel := range availableChannels {
		option := arguments.Option{Value: channel}
		if channel == cluster.Channel() {
			option.Description = "current"
		}
		options = append(options, option)
	}

	channelFlag := cmd.Flags().Lookup("channel")
	if args.channel == "" && !channelFlag.Changed {
		args.channel = cluster.Channel()
		err := channelFlag.Value.Set(cluster.Channel())
		if err != nil {
			return nil, err
		}
	}
	err := arguments.PromptOrCheckOneOf(cmd.Flags(), "channel", options)
	if err != nil {
		return nil, err
	}
	if args.channel == "" || args.channel == cluster.Channel() {
		return cluster, nil
	}

	body, err := cmv1.NewCluster().
		Channel(args.channel).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create channel patch for cluster '%s': %v", cluster.ID(), err)
	}

	response, err := connection.ClustersMgmt().V1().
		Clusters().
		Cluster(cluster.ID()).
		Update().
		Body(body).
		Send()
	if err != nil {
		return nil, fmt.Errorf("Failed to update channel for cluster '%s': %v", cluster.ID(), err)
	}

	return response.Body(), nil
}

func parseNextRun(value string, now time.Time) (time.Time, error) {
	if value == "" {
		return now.Add(defaultScheduleDelay).Truncate(time.Second), nil
	}

	nextRun, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse --next-run: %v", err)
	}
	nextRun = nextRun.UTC()

	if !nextRun.After(now.Add(minScheduleDelay)) {
		return time.Time{}, fmt.Errorf("--next-run must be more than five minutes from now")
	}

	return nextRun, nil
}
