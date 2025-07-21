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

package job

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/spf13/cobra"
)

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "job QUEUE_NAME JOB_ID RECEIPT_ID",
	Short: "Mark the Job as a success",
	Long:  "Mark the Job as a success on the specified Job Queue.",
	Args:  cobra.ExactArgs(3),
	RunE:  run,
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
	_, err = ocm.SendTypedAndHandleDeprecation(
		client.Queues().Queue(argv[0]).Jobs().Job(argv[1]).Success().ReceiptId(argv[2]))
	if err != nil {
		return fmt.Errorf("unable to success a job: %v", err)
	}

	return nil
}
