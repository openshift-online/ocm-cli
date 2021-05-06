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

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-sdk-go/jobqueue/v1"
)

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "job QUEUE_NAME",
	Short: "Pop (i.e. fetch) a Job",
	Long:  "Fetch a job from the specified Job Queue.",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func run(_ *cobra.Command, argv []string) error {
	var (
		pop *v1.QueuePopResponse
		err error
	)

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the Job Queue management api
	client := connection.JobQueue().V1()

	// Send a request to create the Job:
	pop, err = client.Queues().Queue(argv[0]).Pop().Send()
	if err != nil {
		if pop != nil && pop.Status() == 204 {
			// No Content
			fmt.Printf("No job found\n")
			return nil
		}
		return fmt.Errorf("unable to fetch a Job: %v", err)
	}
	fmt.Printf("{\n"+
		"  \"id\": \"%s\",\n"+
		"  \"kind\": \"Job\",\n"+
		"  \"href\": \"%s\",\n"+
		"  \"attempts\": \"%d\",\n"+
		"  \"abandoned_at\": \"%s\",\n"+
		"  \"created_at\": \"%s\",\n"+
		"  \"updated_at\": \"%s\",\n"+
		"  \"receipt_id\": \"%s\",\n"+
		"}",
		pop.ID(),
		pop.HREF(),
		pop.Attempts(),
		pop.AbandonedAt(),
		pop.CreatedAt(),
		pop.UpdatedAt(),
		pop.ReceiptId(),
	)

	return nil
}
