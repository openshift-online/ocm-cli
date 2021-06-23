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

package logout

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/pkg/config"
)

var Cmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out",
	Long:  "Log out, removing connection related variables from the config file.",
	Args:  cobra.NoArgs,
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("can't load configuration file: %w", err)
	}

	// Remove all the login related settings from the configuration file:
	cfg.AccessToken = ""
	cfg.ClientID = ""
	cfg.ClientSecret = ""
	cfg.Insecure = false
	cfg.Password = ""
	cfg.RefreshToken = ""
	cfg.Scopes = nil
	cfg.TokenURL = ""
	cfg.URL = ""
	cfg.User = ""

	// Save the configuration file:
	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("can't save configuration file: %w", err)
	}

	return nil
}
