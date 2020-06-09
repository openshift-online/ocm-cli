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

package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift-online/ocm-cli/cmd/ocm/config/get"
	"github.com/openshift-online/ocm-cli/cmd/ocm/config/set"
	"github.com/openshift-online/ocm-cli/pkg/config"
)

func configVarDocs() (ret string) {
	// TODO(efried): Figure out how to get the Type without instantiating.
	configType := reflect.ValueOf(config.Config{}).Type()
	fieldHelps := make([]string, configType.NumField())
	for i := 0; i < len(fieldHelps); i++ {
		tag := configType.Field(i).Tag
		// TODO(efried): Use JSON parser instead
		name := strings.Split(tag.Get("json"), ",")[0]
		doc := tag.Get("doc")
		fieldHelps[i] = fmt.Sprintf("\t%-15s%s", name, doc)
	}
	ret = strings.Join(fieldHelps, "\n")
	return
}

func longHelp() (ret string) {
	loc, err := config.Location()
	if err != nil {
		// I think this only happens if homedir.Dir() fails, which is unlikely.
		loc = fmt.Sprintf("UNKNOWN (%s)", err)
	}
	ret = fmt.Sprintf(`Get or set variables from a configuration file.

The location of the configuration file is gleaned from the 'OCM_CONFIG' environment variable,
or ~/.ocm.json if that variable is unset. Currently using: %s

The following variables are supported:

%s`, loc, configVarDocs())
	return
}

var Cmd = &cobra.Command{
	Use:   "config COMMAND VARIABLE",
	Short: "get or set configuration variables",
	Long:  longHelp(),
}

func init() {
	Cmd.AddCommand(get.Cmd)
	Cmd.AddCommand(set.Cmd)
}
