/*
Copyright (c) 2021 Red Hat, Inc.

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

package fail

import (
	"fmt"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/spf13/cobra"
)

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:     "failure QUEUE_NAME JOB_ID RECEIPT_ID FAILURE_REASON",
	Aliases: []string{"fail"},
	Short:   "Mark the new Job as a failure",
	Long:    "Mark the new Job as a failure with specific reason on the specified Job Queue.",
	Args:    cobra.ExactArgs(4),
	RunE:    run,
}

func run(_ *cobra.Command, argv []string) error {
	var err error

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the Job Queue management api
	client := connection.JobQueue().V1()

	// Send a request to Success a Job:
	_, err = client.Queues().Queue(argv[0]).Jobs().Job(argv[1]).Failure().ReceiptId(argv[2]).FailureReason(argv[3]).Send()
	if err != nil {
		return fmt.Errorf("unable to fail a Job: %v", err)
	}

	return nil
}
