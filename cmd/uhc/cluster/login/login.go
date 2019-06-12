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

package login

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/openshift-online/uhc-cli/pkg/config"
	"github.com/openshift-online/uhc-cli/pkg/util"
	"github.com/openshift-online/uhc-sdk-go/pkg/client"
	"github.com/spf13/cobra"
)

var args struct {
	debug bool
	user  string
}

var Cmd = &cobra.Command{
	Use:   "login CLUSTERID",
	Short: "login to a cluster",
	Long:  "login to a cluster by CLUSTERID using openshift oc",
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
	flags.StringVarP(
		&args.user,
		"username",
		"u",
		"",
		"Username, will prompt if not provided",
	)

}
func run(cmd *cobra.Command, argv []string) {

	if len(argv) != 1 {
		fmt.Fprint(os.Stderr, "Expected exactly one cluster\n")
		os.Exit(1)
	}
	path, err := exec.LookPath("oc")
	if err != nil {
		fmt.Fprint(os.Stderr, "To run this, you need install openshfit oc first.\n")
		os.Exit(1)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load config file: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprint(os.Stderr, "Not logged in, run the 'login' command\n")
		os.Exit(1)
	}

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, err := config.Armed(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't check if tokens have expired: %v\n", err)
		os.Exit(1)
	}
	if !armed {
		fmt.Fprint(os.Stderr, "Tokens have expired, run the 'login' command\n")
		os.Exit(1)
	}

	// Create the logger
	logger, err := util.NewLogger(args.debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create logger: %v\n", err)
		os.Exit(1)
	}

	// Create the connection, and remember to close it:
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
	defer connection.Close()

	// Get the client for the resource that manages the collection of clusters:
	resource := connection.ClustersMgmt().V1().Clusters()

	// Get the resource that manages the cluster that we want to display:
	clusterResource := resource.Cluster(argv[0])

	// Retrieve the cluster info:
	response, err := clusterResource.Get().
		Send()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't retrieve clusters: %s", err)
		os.Exit(1)
	}
	cluster := response.Body()

	// Get the cluster api for login:
	url := cluster.API().URL()
	if len(url) == 0 {
		fmt.Fprintf(os.Stderr, "Cannot find the api url for cluster: %s", argv[0])
		os.Exit(1)
	}
	ocArgs := []string{}
	ocArgs = append(ocArgs, "login", url)
	if args.user != "" {
		ocArgs = append(ocArgs, "--username="+args.user)
	}

	// #nosec G204
	ocCmd := exec.Command(path, ocArgs...)
	ocCmd.Stderr = os.Stderr
	ocCmd.Stdin = os.Stdin
	ocCmd.Stdout = os.Stdout
	err = ocCmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to login to cluster: %s", err)
		os.Exit(1)
	}
}
