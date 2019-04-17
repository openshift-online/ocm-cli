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

package post

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	"github.com/spf13/cobra"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/dump"
	"github.com/openshift-online/uhc-cli/pkg/util"
)

var args struct {
	debug     bool
	parameter []string
	header    []string
	body      string
}

var Cmd = &cobra.Command{
	Use:   "post PATH",
	Short: "Send a POST request",
	Long:  "Send a POST request to the given path.",
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
	flags.StringSliceVar(
		&args.parameter,
		"parameter",
		nil,
		"Query parameters to add to the request. The value must be the name of the "+
			"parameter, followed by an optional equals sign and then the value "+
			"of the parameter. Can be used multiple times to specify multiple "+
			"parameters or multiple values for the same parameter.",
	)
	flags.StringSliceVar(
		&args.header,
		"header",
		nil,
		"Headers to add to the request. The value must be the name of the header "+
			"followed by an optional equals sign and then the value of the "+
			"header. Can be used multiple times to specify multiple headers "+
			"or multiple values for the same header.",
	)
	flags.StringVar(
		&args.body,
		"body",
		"",
		"Name fo the file containing the request body. If this isn't given then "+
			"the body will be taken from the standard input.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	// Check that there is exactly one command line parameter:
	if len(argv) != 1 {
		fmt.Fprintf(os.Stderr, "Expected exactly one argument\n")
		os.Exit(1)
	}
	path := argv[0]

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
	armed, err := config.Armed(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprintf(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the connection:
	logger, err := util.NewLogger(args.debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create logger: %v\n", err)
		os.Exit(1)
	}
	connection, err := client.NewConnectionBuilder().
		Logger(logger).
		TokenURL(cfg.TokenURL).
		Client(cfg.ClientID, cfg.ClientSecret).
		Scopes(cfg.Scopes...).
		URL(cfg.URL).
		User(cfg.User, cfg.Password).
		Tokens(cfg.AccessToken, cfg.RefreshToken).
		Insecure(cfg.Insecure).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create connection: %v\n", err)
		os.Exit(1)
	}

	// Create and populate the request:
	request := connection.Post().Path(path)
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
	var body []byte
	if args.body != "" {
		body, err = ioutil.ReadFile(args.body)
	} else {
		body, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't read body: %v\n", err)
		os.Exit(1)
	}
	request.Bytes(body)

	// Send the request:
	response, err := request.Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't send request: %v\n", err)
		os.Exit(1)
	}
	status := response.Status()
	body = response.Bytes()
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
