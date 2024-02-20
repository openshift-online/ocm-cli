/*
Copyright (c) 2024 Red Hat, Inc.

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

package reset

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-sdk-go/authentication/securestore"
)

var args struct {
	debug bool
}

// start: variables
const KEYRING = "keyring"

// end: variables

func buildVarHelpText() string {
	ret := []string{}
	var variables = map[string]string{
		KEYRING: "Removes OCM configuration from the current RH_KEYRING keyring",
	}

	for k, v := range variables {
		ret = append(ret, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(ret, "\n")
}

func longHelp() (ret string) {
	ret = fmt.Sprintf(`Resets/removes the requested option from configuration.

The following variables are supported:

%s
`, buildVarHelpText())
	return
}

var Cmd = &cobra.Command{
	Use:   "reset [flags] VARIABLE",
	Short: "Resets/removes the requested option from configuration",
	Long:  longHelp(),
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	switch argv[0] {
	case KEYRING:
		keyring := os.Getenv("RH_KEYRING")
		if keyring == "" {
			return fmt.Errorf("RH_KEYRING is required to reset config")
		}
		err := securestore.RemoveConfigFromKeyring(keyring)
		if err != nil {
			return fmt.Errorf("can't reset keyring: %v", err)
		}
	default:
		return fmt.Errorf("unknown setting")
	}

	return nil
}
