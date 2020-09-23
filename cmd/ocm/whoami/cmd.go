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

package whoami

import (
	"bytes"
	"fmt"
	"os"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Prints user information",
	Long:  "Prints user information.",
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Send the request:
	response, err := connection.AccountsMgmt().V1().CurrentAccount().Get().
		Send()
	if err != nil {
		return fmt.Errorf("Can't send request: %v", err)
	}

	// Buffer for pretty output:
	buf := new(bytes.Buffer)

	// Output account info.
	err = amsv1.MarshalAccount(response.Body(), buf)
	if err != nil {
		return fmt.Errorf("Failed to marshal account into JSON encoder: %v", err)
	}

	if response.Status() < 400 {
		err = dump.Pretty(os.Stdout, buf.Bytes())
	} else {
		err = dump.Pretty(os.Stderr, buf.Bytes())
	}
	if err != nil {
		return fmt.Errorf("Can't print body: %v", err)
	}

	return nil
}
