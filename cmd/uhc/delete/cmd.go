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
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/dump"
	"github.com/openshift-online/uhc-cli/pkg/urls"
)

var args struct {
	debug     bool
	parameter []string
	header    []string
}

var Cmd = &cobra.Command{
	Use:   "delete PATH",
	Short: "Send a DELETE request",
	Long:  "Send a DELETE request to the given path.",
	Run:   run,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug mode.",
	)
	flags.StringArrayVar(
		&args.parameter,
		"parameter",
		nil,
		"Query parameters to add to the request. The value must be the name of the "+
			"parameter, followed by an optional equals sign and then the value "+
			"of the parameter. Can be used multiple times to specify multiple "+
			"parameters or multiple values for the same parameter.",
	)
	flags.StringArrayVar(
		&args.header,
		"header",
		nil,
		"Headers to add to the request. The value must be the name of the header "+
			"followed by an optional equals sign and then the value of the "+
			"header. Can be used multiple times to specify multiple headers "+
			"or multiple values for the same header.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	path, err := urls.Expand(argv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create URI: %v\n", err)
		os.Exit(1)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "Not logged in, run the 'login' command\n")
		os.Exit(1)
	}

	// Check that the configuration has credentials or tokens that don't have expired:
	armed, err := cfg.Armed()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection:
	connection, err := cfg.Connection(args.debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}

	// Create and populate the request:
	request := connection.Delete().Path(path)
	for _, parameter := range args.parameter {
		var name string
		var value string
		position := strings.Index(parameter, "=")
		if position != -1 {
			name = parameter[:position]
			value = parameter[position+1:]
		} else {
			name = parameter
			value = ""
		}
		request.Parameter(name, value)
	}
	for _, header := range args.header {
		var name string
		var value string
		position := strings.Index(header, "=")
		if position != -1 {
			name = header[:position]
			value = header[position+1:]
		} else {
			name = header
			value = ""
		}
		request.Header(name, value)
	}

	// Send the request:
	response, err := request.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't send request: %v\n", err)
		os.Exit(1)
	}
	status := response.Status()
	body := response.Bytes()
	if status < 400 {
		err = dump.Pretty(os.Stdout, body)
	} else {
		err = dump.Pretty(os.Stderr, body)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't print body: %v\n", err)
		os.Exit(1)
	}

	// Save the configuration:
	cfg.AccessToken, cfg.RefreshToken, err = connection.Tokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't get tokens: %v\n", err)
		os.Exit(1)
	}
	err = config.Save(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't save config file: %v\n", err)
		os.Exit(1)
	}

	// Bye:
	if status < 400 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
