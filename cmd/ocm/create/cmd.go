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

package create

import (
	"github.com/openshift-online/ocm-cli/cmd/ocm/create/cluster"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create/idp"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create/ingress"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create/machinepool"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create/upgradepolicy"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create/user"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "create [flags] RESOURCE",
	Aliases: []string{"add"},
	Short:   "Create a resource from stdin",
	Long:    "Create a resource from stdin",
}

func init() {
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(idp.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(machinepool.Cmd)
	Cmd.AddCommand(upgradepolicy.Cmd)
	Cmd.AddCommand(user.Cmd)
}
