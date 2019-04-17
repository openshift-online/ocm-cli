/*
Copyright (c) 2018 Red Hat, Inc.

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

package version

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/uhc-cli/pkg/info"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version",
	Long:  "Prints the version number of the client.",
	Run:   run,
}

func run(cmd *cobra.Command, argv []string) {
	// Print the version:
	fmt.Fprintf(os.Stdout, "%s\n", info.Version)

	// Bye:
	os.Exit(0)
}
