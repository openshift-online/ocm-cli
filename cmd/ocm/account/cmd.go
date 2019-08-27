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

package account

import (
	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/cmd/ocm/account/orgs"
	"github.com/openshift-online/ocm-cli/cmd/ocm/account/quota"
	"github.com/openshift-online/ocm-cli/cmd/ocm/account/roles"
	"github.com/openshift-online/ocm-cli/cmd/ocm/account/status"
	"github.com/openshift-online/ocm-cli/cmd/ocm/account/users"
)

// Cmd ...
var Cmd = &cobra.Command{
	Use:   "account COMMAND",
	Short: "Get information about users.",
	Long:  "Get status or information about a single or list of users.",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	Cmd.AddCommand(quota.Cmd)
	Cmd.AddCommand(orgs.Cmd)
	Cmd.AddCommand(status.Cmd)
	Cmd.AddCommand(roles.Cmd)
	Cmd.AddCommand(users.Cmd)
}
