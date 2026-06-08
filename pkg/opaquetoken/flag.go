/*
Copyright (c) 2025 Red Hat, Inc.

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

package opaquetoken

import (
	"os"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/openshift-online/ocm-cli/pkg/properties"
)

// AddFlag adds the --opaque-token flag to the given set of command line flags.
func AddFlag(flags *pflag.FlagSet) {
	flags.BoolVar(
		&enabled,
		"opaque-token",
		false,
		"Treat the access token as an opaque (non-JWT) token. Can also be "+
			"enabled by setting the OCM_OPAQUE_TOKEN environment variable.",
	)
}

// Enabled returns true if opaque token mode is enabled via the flag or environment variable.
func Enabled() bool {
	if enabled {
		return true
	}
	val, err := strconv.ParseBool(os.Getenv(properties.OpaqueTokenEnvKey))
	return err == nil && val
}

var enabled bool
