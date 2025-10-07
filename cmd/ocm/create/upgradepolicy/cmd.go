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

package upgradepolicy

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "upgrade-policy",
	Aliases: []string{"upgradepolicy", "upgrade-policies", "upgradepolicys"},
	Short:   "set an upgrade policy for the cluster",
	Long:    "set a manual or automatic upgrade policy for the cluster",
	Example: " ocm create upgrade-policy --cluster mycluster\n",
	RunE:    run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the machine pool to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")
}

func run(cmd *cobra.Command, argv []string) error {
	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
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
		return fmt.Errorf("failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	clusterCollection := connection.ClustersMgmt().V1().Clusters()
	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("failed to get cluster '%s': %v", clusterKey, err)
	}

	var scheduleType string
	var version string
	var upgradePreference string
	var timestamp time.Time

	prompt := &survey.Select{
		Message: "Select policy type",
		Options: []string{"manual", "automatic"},
	}
	err = survey.AskOne(prompt, &scheduleType)
	if err != nil {
		return fmt.Errorf("Failed to get a policy type")
	}

	var upgradeBuilder *cmv1.UpgradePolicyBuilder

	if scheduleType == "automatic" {
		prompt = &survey.Select{
			Message: "Select day",
			Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		}
		var day string
		err = survey.AskOne(prompt, &day)
		if err != nil {
			return fmt.Errorf("Failed to get a valid day")
		}

		var daysOfWeek = map[string]time.Weekday{
			"Sunday":    time.Sunday,
			"Monday":    time.Monday,
			"Tuesday":   time.Tuesday,
			"Wednesday": time.Wednesday,
			"Thursday":  time.Thursday,
			"Friday":    time.Friday,
			"Saturday":  time.Saturday,
		}

		dayInt, ok := daysOfWeek[day]
		if !ok {
			return fmt.Errorf("Failed to get a valid day")
		}

		hours := make([]string, 24)
		var start time.Time
		for i := range hours {
			t := start.Add(time.Hour * time.Duration(i))
			hours[i] = t.Format("15:04")
		}

		prompt = &survey.Select{
			Message: "Select hour (UTC)",
			Options: hours,
		}
		var hour string
		err = survey.AskOne(prompt, &hour)
		if err != nil {
			return fmt.Errorf("Failed to get a valid hour")
		}

		hourInt, err := strconv.Atoi(strings.Split(hour, ":")[0])
		if err != nil {
			return nil
		}

		cronExpression := fmt.Sprintf("0 %d * * %d", hourInt, dayInt)

		upgradeBuilder = cmv1.NewUpgradePolicy().
			ScheduleType("automatic").
			Schedule(cronExpression)

	} else {

		availableUpgrades := c.GetAvailableUpgrades(cluster.Version())
		if len(availableUpgrades) == 0 {
			fmt.Println("There are no available upgrades")
			return nil
		}

		prompt := &survey.Select{
			Message: "Select version",
			Options: availableUpgrades,
		}
		err = survey.AskOne(prompt, &version)
		if err != nil {
			return fmt.Errorf("Failed to get a valid version to upgrade to")
		}
		prompt = &survey.Select{
			Message: "Schedule Upgrade",
			Options: []string{"Upgrade now", "Schedule a different time"},
		}
		err = survey.AskOne(prompt, &upgradePreference)
		if err != nil {
			return fmt.Errorf("Failed to get an upgrade time preference")
		}
		if upgradePreference == "Upgrade now" {
			timestamp = time.Now().UTC().Add(time.Minute * 10)

		} else {

			var validationQs = []*survey.Question{
				{
					Name:   "date",
					Prompt: &survey.Input{Message: "Please input desired date in format yyyy-mm-dd"},
					Validate: func(val interface{}) error {
						str, _ := val.(string)
						_, err := time.Parse("2006-01-02", str)
						if err != nil {
							return fmt.Errorf("date format invalid")
						}
						return nil
					},
				},
				{
					Name:   "desiredTime",
					Prompt: &survey.Input{Message: "Please input desired UTC time in format HH:mm"},
					Validate: func(val interface{}) error {
						str, _ := val.(string)
						_, err := time.Parse("15:04", str)
						if err != nil {
							return fmt.Errorf("time format invalid")
						}
						return nil
					},
				},
			}
			answers := struct {
				Date        string
				DesiredTime string
			}{}
			err = survey.Ask(validationQs, &answers)
			if err != nil {
				return err
			}

			desiredTime := fmt.Sprintf("%sT%s:00.000Z", answers.Date, answers.DesiredTime)
			timestamp, _ = time.Parse(time.RFC3339, desiredTime)
			fmt.Println(timestamp)
		}

		upgradeBuilder = cmv1.NewUpgradePolicy().
			ScheduleType("manual").
			NextRun(timestamp).
			Version(version)
	}

	upgradePolicy, err := upgradeBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to set an upgrade policy for cluster '%s': %v", clusterKey, err)
	}

	_, err = clusterCollection.Cluster(cluster.ID()).
		UpgradePolicies().
		Add().
		Body(upgradePolicy).
		Send()
	if err != nil {
		return fmt.Errorf("Failed to create upgrade policy for cluster: %v", err)
	}
	fmt.Println("upgrade policy successfully created")

	return nil
}
