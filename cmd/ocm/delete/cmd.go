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

package delete

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/idp"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/ingress"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/machinepool"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/upgradepolicy"
	"github.com/openshift-online/ocm-cli/cmd/ocm/delete/user"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/urls"
)

var args struct {
	parameter []string
	header    []string
}

var Cmd = &cobra.Command{
	Use:       "delete [flags] (PATH | RESOURCE_ALIAS RESOURCE_ID)",
	Short:     "Send a DELETE request",
	Long:      "Send a DELETE request to the given path.",
	RunE:      run,
	ValidArgs: urls.Resources(),
}

// for template format refer: https://pkg.go.dev/text/template
var usageTemplate = `
Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if .HasExample}}

Resource Alias:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}
{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}
{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

func init() {
	Cmd.SetUsageTemplate(usageTemplate)
	Cmd.SetHelpTemplate(helpTemplate)
	Cmd.Example = "  account\n  addon\n  cluster\n  role_binding\n  sku_rule\n  subscription"
	fs := Cmd.Flags()
	arguments.AddParameterFlag(fs, &args.parameter)
	arguments.AddHeaderFlag(fs, &args.header)
	Cmd.AddCommand(idp.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(machinepool.Cmd)
	Cmd.AddCommand(upgradepolicy.Cmd)
	Cmd.AddCommand(user.Cmd)
}

func run(cmd *cobra.Command, argv []string) error {
	path, err := urls.Expand(argv)
	if err != nil {
		return fmt.Errorf("could not create URI: %w", err)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("failed to create OCM connection: %w", err)
	}
	defer connection.Close()

	// Create and populate the request:
	request := connection.Delete()
	err = arguments.ApplyPathArg(request, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't parse path '%s': %v\n", path, err)
		os.Exit(1)
	}
	arguments.ApplyParameterFlag(request, args.parameter)
	arguments.ApplyHeaderFlag(request, args.header)

	// Send the request:
	response, err := ocm.SendAndHandleDeprecation(request)
	if err != nil {
		return fmt.Errorf("can't send request: %w", err)
	}

	status := response.Status()
	body := response.Bytes()
	if status < 400 {
		err = dump.Pretty(os.Stdout, body)
	} else {
		err = dump.Pretty(os.Stderr, body)
	}
	if err != nil {
		return fmt.Errorf("can't print body: %w", err)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("can't load config file: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("not logged in, run the 'login' command")
	}

	// Save the configuration:
	cfg.AccessToken, cfg.RefreshToken, err = connection.Tokens()
	if err != nil {
		return fmt.Errorf("can't get tokens: %w", err)
	}
	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("can't save config file: %w", err)
	}

	// Bye:
	if status >= 400 {
		os.Exit(1)
	}

	return nil
}
