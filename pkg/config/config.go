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

// This file contains the types and functions used to manage the configuration of the command line
// client.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/openshift-online/ocm-sdk-go/authentication/securestore"

	"github.com/openshift-online/ocm-cli/pkg/properties"
)

// Config is the type used to store the configuration of the client.
// There's no way to line-split or predefine tags, so...
// nolint:lll
type Config struct {
	// TODO(efried): Better docs for things like AccessToken
	// TODO(efried): Dedup with flag docs in cmd/ocm/login/cmd.go:init where possible
	AccessToken  string   `json:"access_token,omitempty" doc:"Bearer access token."`
	ClientID     string   `json:"client_id,omitempty" doc:"OpenID client identifier."`
	ClientSecret string   `json:"client_secret,omitempty" doc:"OpenID client secret."`
	Insecure     bool     `json:"insecure,omitempty" doc:"Enables insecure communication with the server. This disables verification of TLS certificates and host names."`
	Password     string   `json:"password,omitempty" doc:"User password."`
	RefreshToken string   `json:"refresh_token,omitempty" doc:"Offline or refresh token."`
	Scopes       []string `json:"scopes,omitempty" doc:"OpenID scope. If this option is used it will replace completely the default scopes. Can be repeated multiple times to specify multiple scopes."`
	TokenURL     string   `json:"token_url,omitempty" doc:"OpenID token URL."`
	URL          string   `json:"url,omitempty" doc:"URL of the API gateway. The value can be the complete URL or an alias. The valid aliases are 'production', 'staging' and 'integration'."`
	User         string   `json:"user,omitempty" doc:"User name."`
	Pager        string   `json:"pager,omitempty" doc:"Pager command, for example 'less'. If empty no pager will be used."`
}

// Load loads the configuration from the OS keyring first if available, load from the configuration file if not
func Load() (cfg *Config, err error) {
	if keyring, ok := IsKeyringManaged(); ok {
		return loadFromOS(keyring)
	}

	return loadFromFile()
}

// loadFromOS loads the configuration from the OS keyring. If the configuration doesn't exist
// it will return an empty configuration object.
func loadFromOS(keyring string) (cfg *Config, err error) {
	cfg = &Config{}

	data, err := securestore.GetConfigFromKeyring(keyring)
	if err != nil {
		return nil, fmt.Errorf("can't load config from OS keyring [%s]: %v", keyring, err)
	}
	// No config found, return
	if len(data) == 0 {
		return nil, nil
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		// Treat the config as empty if it can't be unmarshaled, it is invalid
		return nil, nil
	}
	return cfg, nil
}

// loadFromFile loads the configuration from the configuration file. If the configuration file doesn't exist
// it will return an empty configuration object.
func loadFromFile() (cfg *Config, err error) {
	file, err := Location()
	if err != nil {
		return
	}
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		cfg = &Config{}
		err = nil
		return
	}
	if err != nil {
		err = fmt.Errorf("can't check if config file '%s' exists: %v", file, err)
		return
	}
	// #nosec G304
	data, err := os.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("can't read config file '%s': %v", file, err)
		return
	}
	cfg = &Config{}
	if len(data) == 0 {
		return
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		err = fmt.Errorf("can't parse config file '%s': %v", file, err)
		return
	}
	return
}

// Save saves the given configuration to the configuration file.
func Save(cfg *Config) error {
	file, err := Location()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("can't marshal config: %v", err)
	}

	if keyring, ok := IsKeyringManaged(); ok {
		// Use the OS keyring if the OCM_CONFIG env var is set to a valid keyring backend
		err := securestore.UpsertConfigToKeyring(keyring, data)
		if err != nil {
			return fmt.Errorf("can't save config to OS keyring [%s]: %v", keyring, err)
		}
		return nil
	}

	dir := filepath.Dir(file)
	err = os.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("can't create directory %s: %v", dir, err)
	}
	err = os.WriteFile(file, data, 0600)
	if err != nil {
		return fmt.Errorf("can't write file '%s': %v", file, err)
	}
	return nil
}

// Location returns the location of the configuration file. If a configuration file
// already exists in the HOME directory, it uses that, otherwise it prefers to
// use the XDG config directory.
func Location() (path string, err error) {
	if ocmconfig := os.Getenv("OCM_CONFIG"); ocmconfig != "" {
		return ocmconfig, nil
	}

	// Determine home directory to use for the legacy file path
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	path = filepath.Join(home, ".ocm.json")

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// Determine standard config directory
		configDir, err := os.UserConfigDir()
		if err != nil {
			return path, err
		}

		// Use standard config directory
		path = filepath.Join(configDir, "/ocm/ocm.json")
	}

	return path, nil
}

// Armed checks if the configuration contains either credentials or tokens that haven't expired, so
// that it can be used to perform authenticated requests.
func (c *Config) Armed() (armed bool, reason string, err error) {
	// Check URLs:
	haveURL := c.URL != ""
	haveTokenURL := c.TokenURL != ""
	haveURLs := haveURL && haveTokenURL

	// Check credentials:
	havePassword := c.User != "" && c.Password != ""
	haveSecret := c.ClientID != "" && c.ClientSecret != ""
	haveCredentials := havePassword || haveSecret

	// Check tokens:
	haveAccess := c.AccessToken != ""
	accessUsable := false
	if haveAccess {
		accessUsable, err = tokenUsable(c.AccessToken, 5*time.Second)
		if err != nil {
			return
		}
	}
	haveRefresh := c.RefreshToken != ""
	refreshUsable := false
	if haveRefresh {
		if IsEncryptedToken(c.RefreshToken) {
			// We have no way of knowing an encrypted token expiration, so
			// we assume it's valid and let the access token request fail.
			refreshUsable = true
		} else {
			refreshUsable, err = tokenUsable(c.RefreshToken, 10*time.Second)
			if err != nil {
				return
			}
		}
	}

	// Calculate the result:
	armed = haveURLs && (haveCredentials || accessUsable || refreshUsable)
	if armed {
		return
	}

	// If it isn't armed then we should return a human readable reason. We should try to
	// generate a message that describes the more relevant reason. For example, missing
	// credentials is more important than missing URLs, so that condition should be checked
	// first.
	switch {
	case haveAccess && !haveRefresh && !accessUsable:
		reason = "access token is expired"
	case !haveAccess && haveRefresh && !refreshUsable:
		reason = "refresh token is expired"
	case haveAccess && !accessUsable && haveRefresh && !refreshUsable:
		reason = "access and refresh tokens are expired"
	case !haveCredentials:
		reason = "credentials aren't set"
	case !haveURL && haveTokenURL:
		reason = "server URL isn't set"
	case haveURL && !haveTokenURL:
		reason = "token URL isn't set"
	case !haveURL && !haveTokenURL:
		reason = "server and token URLs aren't set"
	}

	return
}

// Disarm removes from the configuration all the settings that are needed for authentication.
func (c *Config) Disarm() {
	c.AccessToken = ""
	c.ClientID = ""
	c.ClientSecret = ""
	c.Insecure = false
	c.Password = ""
	c.RefreshToken = ""
	c.Scopes = nil
	c.TokenURL = ""
	c.URL = ""
	c.User = ""
}

// IsKeyringManaged returns the keyring name and a boolean indicating if the config is managed by the keyring.
func IsKeyringManaged() (keyring string, ok bool) {
	keyring = os.Getenv(properties.KeyringEnvKey)
	return keyring, keyring != ""
}

// GetKeyrings returns the available keyrings on the current host
func GetKeyrings() []string {
	backends := securestore.AvailableBackends()
	if len(backends) == 0 {
		return []string{"no available backends"}
	}
	return backends
}
