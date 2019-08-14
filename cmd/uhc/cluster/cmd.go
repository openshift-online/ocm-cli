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

package cluster

import (
	"github.com/openshift-online/uhc-cli/cmd/uhc/cluster/describe"
	"github.com/openshift-online/uhc-cli/cmd/uhc/cluster/list"
	"github.com/openshift-online/uhc-cli/cmd/uhc/cluster/login"
	"github.com/openshift-online/uhc-cli/cmd/uhc/cluster/status"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cluster COMMAND",
	Short: "Get information about clusters",
	Long:  "Get status or information about a single cluster, or a list of clusters",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	Cmd.AddCommand(list.Cmd)
	Cmd.AddCommand(status.Cmd)
	Cmd.AddCommand(describe.Cmd)
	Cmd.AddCommand(login.Cmd)
}
