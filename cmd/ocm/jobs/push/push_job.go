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

package push

import (
	"fmt"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	v1 "github.com/openshift-online/ocm-sdk-go/jobqueue/v1"
	"github.com/spf13/cobra"
)

var args struct {
	parameter []string
}

// Cmd Constant:
var Cmd = &cobra.Command{
	Use:   "push QUEUE_NAME",
	Short: "Push (i.e. create) a new Job",
	Long:  "Create a new Job on the specified Job Queue.",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	// Add flags to rootCmd:
	flags := Cmd.Flags()
	arguments.AddParameterFlag(flags, &args.parameter)
}

func run(_ *cobra.Command, argv []string) error {
	var (
		push *v1.QueuePushResponse
		err  error
	)

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the Job Queue management api
	client := connection.JobQueue().V1()

	// Send a request to create a Job:
	request := client.Queues().Queue(argv[0]).Push()
	for _, arg := range args.parameter {
		if strings.HasPrefix(arg, "Arguments") {
			params := strings.Split(arg, "=")
			// Apply parameters
			request.Arguments(params[1])
		}
	}
	push, err = request.Send()
	if err != nil {
		return fmt.Errorf("unable to create Job: %v", err)
	}
	fmt.Printf("{\n"+
		"  \"id\": \"%s\",\n"+
		"  \"kind\": \"Job\",\n"+
		"  \"href\": \"%s\",\n"+
		"  \"arguments\": \"%s\",\n"+
		"  \"created_at\": \"%s\",\n"+
		"}",
		push.ID(),
		push.HREF(),
		push.Arguments(),
		push.CreatedAt(),
	)

	return nil
}
