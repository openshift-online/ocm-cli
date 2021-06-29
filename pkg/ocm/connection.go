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

package ocm

import (
	"fmt"

	sdk "github.com/openshift-online/ocm-sdk-go"

	"github.com/openshift-online/ocm-cli/pkg/config"
)

// ConnectionBuilder contains the information and logic needed to build a connection to OCM. Don't
// create instances of this type directly; use the NewConnection function instead.
type ConnectionBuilder struct {
	cfg *config.Config
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
		err = fmt.Errorf("not logged in, %s, run the 'login' command", reason)
		return
	}

	result, err = b.cfg.Connection()
	if err != nil {
		return
	}

	return
}
