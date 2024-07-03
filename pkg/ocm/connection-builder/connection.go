/*
Copyright (c) 2020 Red Hat, Inc.
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

package connection

import (
	"fmt"

	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/debug"
	"github.com/openshift-online/ocm-cli/pkg/info"
)

// ConnectionBuilder contains the information and logic needed to build a connection to OCM. Don't
// create instances of this type directly; use the NewConnection function instead.
type ConnectionBuilder struct {
	// cfg is the ocm config file loaded from disk or keychain
	cfg *config.Config

	// logger is a logging instance used by the ocm sdk
	// defaults to a basic logger instance
	logger logging.Logger

	// api url override is provided to override the configuration file API url
	// defaults to whatever is in the ocm config file
	apiUrlOverride string

	// agent is the UserAgent for a given CLI.
	// defaults to OCM_CLI+version
	agent string
}

// NewConnection creates a builder that can then be used to configure and build an OCM connection.
// Don't create instances of this type directly; use the NewConnection function instead.
func NewConnection() *ConnectionBuilder {
	return &ConnectionBuilder{}
}

// Config sets the configuration that the connection will use to authenticate the user
func (b *ConnectionBuilder) Config(value *config.Config) *ConnectionBuilder {
	b.cfg = value
	return b
}

// Override the default logging implementation
func (b *ConnectionBuilder) WithLogger(logger logging.Logger) *ConnectionBuilder {
	b.logger = logger
	return b
}

// Override the default API URL
func (b *ConnectionBuilder) WithApiUrl(url string) *ConnectionBuilder {
	b.apiUrlOverride = url
	return b
}

// Override the default UserAgent String
func (b *ConnectionBuilder) AsAgent(agent string) *ConnectionBuilder {
	b.agent = agent
	return b
}

// Build uses the information stored in the builder to create a new OCM connection.
func (b *ConnectionBuilder) Build() (result *sdk.Connection, err error) {
	if b.cfg == nil {
		// Load the configuration file:
		b.cfg, err = config.Load()
		if err != nil {
			return
		}
		if b.cfg == nil {
			err = fmt.Errorf("Not logged in, run the 'login' command")
			return
		}
	}

	// Check that the configuration has credentials or tokens that haven't have expired:
	armed, reason, err := b.cfg.Armed()
	if err != nil {
		return
	}
	if !armed {
		err = fmt.Errorf("Not logged in, %s, run the 'login' command", reason)
		return
	}

	builder := b.initConnectionBuilderFromConfig()

	logger, err := b.getLogger()
	if err != nil {
		return
	}
	builder.Logger(logger)

	agent := b.getAgent()
	builder.Agent(agent)

	if b.apiUrlOverride != "" {
		builder.URL(b.apiUrlOverride)
	}

	// Create the connection:
	return builder.Build()
}

func (b *ConnectionBuilder) initConnectionBuilderFromConfig() *sdk.ConnectionBuilder {
	builder := sdk.NewConnectionBuilder()

	// Prepare the builder for the connection adding only the properties that have explicit
	// values in the configuration, so that default values won't be overridden:
	if b.cfg.TokenURL != "" {
		builder.TokenURL(b.cfg.TokenURL)
	}
	if b.cfg.ClientID != "" || b.cfg.ClientSecret != "" {
		builder.Client(b.cfg.ClientID, b.cfg.ClientSecret)
	}
	if b.cfg.Scopes != nil {
		builder.Scopes(b.cfg.Scopes...)
	}
	if b.cfg.User != "" || b.cfg.Password != "" {
		builder.User(b.cfg.User, b.cfg.Password)
	}
	if b.cfg.URL != "" {
		builder.URL(b.cfg.URL)
	}
	tokens := make([]string, 0, 2)
	if b.cfg.AccessToken != "" {
		tokens = append(tokens, b.cfg.AccessToken)
	}
	if b.cfg.RefreshToken != "" {
		tokens = append(tokens, b.cfg.RefreshToken)
	}
	if len(tokens) > 0 {
		builder.Tokens(tokens...)
	}
	builder.Insecure(b.cfg.Insecure)

	return builder
}

// Returns the configured logger or a default if there is none configured
func (b *ConnectionBuilder) getLogger() (logging.Logger, error) {
	if b.logger != nil {
		return b.logger, nil
	}

	// Create a default logger:
	level := glog.Level(1)
	if debug.Enabled() {
		level = glog.Level(0)
	}

	return sdk.NewGlogLoggerBuilder().
		DebugV(level).
		InfoV(level).
		WarnV(level).
		Build()
}

// Returns the configured agent or a default value if there is none configured
func (b *ConnectionBuilder) getAgent() string {
	if b.agent != "" {
		return b.agent
	}
	return "OCM-CLI/" + info.Version
}

// Returns the configured API URL or the one defined in the config file, otherwise
// return an empty string and let the sdk handle it
func (b *ConnectionBuilder) getApiUrl() string {
	if b.apiUrlOverride != "" {
		return b.apiUrlOverride
	}
	return ""
}
