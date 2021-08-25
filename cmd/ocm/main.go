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
	"os/exec"
	"strings"

	_ "github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift-online/ocm-cli/cmd/ocm/account"
	"github.com/openshift-online/ocm-cli/cmd/ocm/cluster"
	"github.com/openshift-online/ocm-cli/cmd/ocm/completion"
	"github.com/openshift-online/ocm-cli/cmd/ocm/config"
	"github.com/openshift-online/ocm-cli/cmd/ocm/create"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete"
	"github.com/openshift-online/ocm-cli/cmd/ocm/describe"
	"github.com/openshift-online/ocm-cli/cmd/ocm/edit"
	"github.com/openshift-online/ocm-cli/cmd/ocm/fail"
	"github.com/openshift-online/ocm-cli/cmd/ocm/get"
	"github.com/openshift-online/ocm-cli/cmd/ocm/hibernate"
	"github.com/openshift-online/ocm-cli/cmd/ocm/list"
	"github.com/openshift-online/ocm-cli/cmd/ocm/login"
	"github.com/openshift-online/ocm-cli/cmd/ocm/logout"
	"github.com/openshift-online/ocm-cli/cmd/ocm/patch"
	plugincmd "github.com/openshift-online/ocm-cli/cmd/ocm/plugin"
	"github.com/openshift-online/ocm-cli/cmd/ocm/pop"
	"github.com/openshift-online/ocm-cli/cmd/ocm/post"
	"github.com/openshift-online/ocm-cli/cmd/ocm/push"
	"github.com/openshift-online/ocm-cli/cmd/ocm/resume"
	"github.com/openshift-online/ocm-cli/cmd/ocm/success"
	"github.com/openshift-online/ocm-cli/cmd/ocm/token"
	"github.com/openshift-online/ocm-cli/cmd/ocm/tunnel"
	"github.com/openshift-online/ocm-cli/cmd/ocm/version"
	"github.com/openshift-online/ocm-cli/cmd/ocm/whoami"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	plugin "github.com/openshift-online/ocm-cli/pkg/plugin"
	"github.com/openshift-online/ocm-cli/pkg/urls"
)

var root = &cobra.Command{
	Use:           "ocm",
	Long:          "Command line tool for api.openshift.com.",
	SilenceUsage:  true,
	SilenceErrors: true,
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
	root.AddCommand(cluster.Cmd)
	root.AddCommand(completion.Cmd)
	root.AddCommand(config.Cmd)
	root.AddCommand(create.Cmd)
	root.AddCommand(delete.Cmd)
	root.AddCommand(describe.Cmd)
	root.AddCommand(edit.Cmd)
	root.AddCommand(fail.Cmd)
	root.AddCommand(get.Cmd)
	root.AddCommand(hibernate.Cmd)
	root.AddCommand(list.Cmd)
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(patch.Cmd)
	root.AddCommand(plugincmd.Cmd)
	root.AddCommand(post.Cmd)
	root.AddCommand(pop.Cmd)
	root.AddCommand(push.Cmd)
	root.AddCommand(resume.Cmd)
	root.AddCommand(success.Cmd)
	root.AddCommand(token.Cmd)
	root.AddCommand(tunnel.Cmd)
	root.AddCommand(version.Cmd)
	root.AddCommand(whoami.Cmd)
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

	// Execute the root command and exit inmediately if there was no error:
	root.SetArgs(os.Args[1:])
	err = root.Execute()
	if err == nil {
		os.Exit(0)
	}

	// Replace well known errors with user friendly messages:
	message := err.Error()
	switch {
	case strings.Contains(message, "Offline user session not found"):
		message = fmt.Sprintf(
			"Offline access token is no longer valid. Go to %s to get a new one and "+
				"then use the 'ocm login --token=...' command to log in with "+
				"that new token.",
			urls.OfflineTokenPage,
		)
	default:
		message = fmt.Sprintf("Error: %s", message)
	}
	fmt.Fprintf(os.Stderr, "%s\n", message)

	// Exit signaling an error:
	os.Exit(1)
}
