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

package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift-online/uhc-cli/cmd/uhc/cluster"
	"github.com/openshift-online/uhc-cli/cmd/uhc/completion"
	"github.com/openshift-online/uhc-cli/cmd/uhc/config"
	"github.com/openshift-online/uhc-cli/cmd/uhc/delete"
	"github.com/openshift-online/uhc-cli/cmd/uhc/get"
	"github.com/openshift-online/uhc-cli/cmd/uhc/login"
	"github.com/openshift-online/uhc-cli/cmd/uhc/logout"
	"github.com/openshift-online/uhc-cli/cmd/uhc/patch"
	"github.com/openshift-online/uhc-cli/cmd/uhc/post"
	"github.com/openshift-online/uhc-cli/cmd/uhc/token"
	"github.com/openshift-online/uhc-cli/cmd/uhc/version"
	"github.com/openshift-online/uhc-cli/cmd/uhc/whoami"
)

var root = &cobra.Command{
	Use:  "uhc",
	Long: "Command line tool for api.openshift.com.",
}

func init() {
	// Send logs to the standard error stream by default:
	err := flag.Set("logtostderr", "true")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't set default error stream: %v\n", err)
		os.Exit(1)
	}

	// Register the options that are managed by the 'flag' package, so that they will also be parsed
	// by the 'pflag' package:
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Register the subcommands:
	root.AddCommand(delete.Cmd)
	root.AddCommand(get.Cmd)
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(patch.Cmd)
	root.AddCommand(post.Cmd)
	root.AddCommand(token.Cmd)
	root.AddCommand(version.Cmd)
	root.AddCommand(cluster.Cmd)
	root.AddCommand(completion.Cmd)
	root.AddCommand(whoami.Cmd)
	root.AddCommand(config.Cmd)
}

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	err := flag.CommandLine.Parse([]string{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't parse empty command line to satisfy 'glog': %v\n", err)
		os.Exit(1)
	}

	// Execute the root command:
	root.SetArgs(os.Args[1:])
	err = root.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute root command: %v\n", err)
		os.Exit(1)
	}
}
