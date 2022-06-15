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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	homedir "github.com/mitchellh/go-homedir"
	sdk "github.com/openshift-online/ocm-sdk-go"

	"github.com/openshift-online/ocm-cli/pkg/debug"
	"github.com/openshift-online/ocm-cli/pkg/info"
)

// Config is the type used to store the configuration of the client.
// There's no way to line-split or predefine tags, so...
//nolint:lll
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

// Load loads the configuration from the configuration file. If the configuration file doesn't exist
// it will return an empty configuration object.
func Load() (cfg *Config, err error) {
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
	data, err := ioutil.ReadFile(file)
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
	dir := filepath.Dir(file)
	err = os.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("can't create directory %s: %v", dir, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("can't marshal config: %v", err)
	}
	err = ioutil.WriteFile(file, data, 0600)
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

// Connection creates a connection using this configuration.
func (c *Config) Connection() (connection *sdk.Connection, err error) {
	// Create the logger:
	level := glog.Level(1)
	if debug.Enabled() {
		level = glog.Level(0)
	}
	logger, err := sdk.NewGlogLoggerBuilder().
		DebugV(level).
		InfoV(level).
		WarnV(level).
		Build()
	if err != nil {
		return
	}

	// Prepare the builder for the connection adding only the properties that have explicit
	// values in the configuration, so that default values won't be overridden:
	builder := sdk.NewConnectionBuilder()
	builder.Logger(logger)
	builder.Agent("OCM-CLI/" + info.Version)
	if c.TokenURL != "" {
		builder.TokenURL(c.TokenURL)
	}
	if c.ClientID != "" || c.ClientSecret != "" {
		builder.Client(c.ClientID, c.ClientSecret)
	}
	if c.Scopes != nil {
		builder.Scopes(c.Scopes...)
	}
	if c.URL != "" {
		builder.URL(c.URL)
	}
	if c.User != "" || c.Password != "" {
		builder.User(c.User, c.Password)
	}
	tokens := make([]string, 0, 2)
	if c.AccessToken != "" {
		tokens = append(tokens, c.AccessToken)
	}
	if c.RefreshToken != "" {
		tokens = append(tokens, c.RefreshToken)
	}
	if len(tokens) > 0 {
		builder.Tokens(tokens...)
	}
	builder.Insecure(c.Insecure)

	// Create the connection:
	connection, err = builder.Build()
	if err != nil {
		return
	}

	return
}
