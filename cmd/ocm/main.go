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
	_ "github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"os/exec"

	"github.com/openshift-online/ocm-cli/cmd/ocm/account"
	"github.com/openshift-online/ocm-cli/cmd/ocm/cluster"
	"github.com/openshift-online/ocm-cli/cmd/ocm/completion"
	"github.com/openshift-online/ocm-cli/cmd/ocm/config"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete"
	"github.com/openshift-online/ocm-cli/cmd/ocm/describe"
	"github.com/openshift-online/ocm-cli/cmd/ocm/edit"
	"github.com/openshift-online/ocm-cli/cmd/ocm/get"
	"github.com/openshift-online/ocm-cli/cmd/ocm/list"
	"github.com/openshift-online/ocm-cli/cmd/ocm/login"
	"github.com/openshift-online/ocm-cli/cmd/ocm/logout"
	"github.com/openshift-online/ocm-cli/cmd/ocm/patch"
	"github.com/openshift-online/ocm-cli/cmd/ocm/post"
	"github.com/openshift-online/ocm-cli/cmd/ocm/token"
	"github.com/openshift-online/ocm-cli/cmd/ocm/tunnel"
	"github.com/openshift-online/ocm-cli/cmd/ocm/version"
	"github.com/openshift-online/ocm-cli/cmd/ocm/whoami"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/plugin"
)

var root = &cobra.Command{
	Use:          "ocm",
	Long:         "Command line tool for api.openshift.com.",
	SilenceUsage: true,
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

	// Add the command line flags:
	fs := root.PersistentFlags()
	arguments.AddDebugFlag(fs)

	// Register the subcommands:
	root.AddCommand(account.Cmd)
	root.AddCommand(create.Cmd)
	root.AddCommand(delete.Cmd)
	root.AddCommand(describe.Cmd)
	root.AddCommand(edit.Cmd)
	root.AddCommand(get.Cmd)
	root.AddCommand(list.Cmd)
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(patch.Cmd)
	root.AddCommand(post.Cmd)
	root.AddCommand(token.Cmd)
	root.AddCommand(tunnel.Cmd)
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
	args := os.Args
	pluginHandler := plugin.NewDefaultPluginHandler([]string{"ocm"})
	if len(args) > 1 {
		cmdPathPieces := args[1:]

		// only look for suitable extension executables if
		// the specified command does not already exist
		if _, _, err := root.Find(cmdPathPieces); err != nil {
			found, err := plugin.HandlePluginCommand(pluginHandler, cmdPathPieces)
			if err != nil {
				err, ok := err.(*exec.ExitError)
				if ok {
					os.Exit(err.ExitCode())
				}
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			if found {
				os.Exit(0)
			}
		}
	}
	// Execute the root command:
	root.SetArgs(os.Args[1:])
	if err = root.Execute(); err != nil {
		os.Exit(1)
	}
}
