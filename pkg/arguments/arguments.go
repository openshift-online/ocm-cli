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

// This file contains functions that add common arguments to the command line.

package arguments

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"strings"

	isatty "github.com/onsi/ginkgo/reporters/stenographer/support/go-isatty"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/debug"
)

type FilePath string

func (f *FilePath) String() string {
	return string(*f)
}

func (f *FilePath) Set(v string) error {
	_, err := os.Stat(v)
	if err != nil {
		return err
	}
	*f = FilePath(v)
	return nil
}

func (f *FilePath) Type() string {
	return "filepath"
}

// AddDebugFlag adds the '--debug' flag to the given set of command line flags.
func AddDebugFlag(fs *pflag.FlagSet) {
	debug.AddFlag(fs)
}

// AddParameterFlag adds the '--parameter' flag to the given set of command line flags.
func AddParameterFlag(fs *pflag.FlagSet, values *[]string) {
	fs.StringArrayVar(
		values,
		"parameter",
		nil,
		"Query parameters to add to the request. The value must be the name of the "+
			"parameter, followed by an optional equals sign and then the value "+
			"of the parameter. Can be used multiple times to specify multiple "+
			"parameters or multiple values for the same parameter. Example: "+
			"--parameter search=\"username like 'myname%'\"",
	)
}

// AddHeaderFlag adds the '--header' flag to the given set of command line flags.
func AddHeaderFlag(fs *pflag.FlagSet, values *[]string) {
	fs.StringArrayVar(
		values,
		"header",
		nil,
		"Headers to add to the request. The value must be the name of the header "+
			"followed by an optional equals sign and then the value of the "+
			"header. Can be used multiple times to specify multiple headers "+
			"or multiple values for the same header.",
	)
}

// AddBodyFlag adds the '--body' flag to the given set of command line flags.
func AddBodyFlag(fs *pflag.FlagSet, value *string) {
	fs.StringVar(
		value,
		"body",
		"",
		"Name fo the file containing the request body. If this isn't given then "+
			"the body will be taken from the standard input.",
	)
}

// AddCCSFlagsWithoutAccountID is sufficient for list regions command.
func AddCCSFlagsWithoutAccountID(fs *pflag.FlagSet, value *cluster.CCS) {
	fs.BoolVar(
		&value.Enabled,
		"ccs",
		false,
		"Leverage your own cloud account (Customer Cloud Subscription).",
	)
	fs.StringVar(
		&value.AWS.AccessKeyID,
		"aws-access-key-id",
		"",
		"AWS access key ID.",
	)
	fs.StringVar(
		&value.AWS.SecretAccessKey,
		"aws-secret-access-key",
		"",
		"AWS secret access key.",
	)
}

// AddCCSFlags adds all the flags needed for creating a cluster.
func AddCCSFlags(fs *pflag.FlagSet, value *cluster.CCS) {
	AddCCSFlagsWithoutAccountID(fs, value)
	fs.StringVar(
		&value.AWS.AccountID,
		"aws-account-id",
		"",
		"AWS account ID.",
	)
}

// CheckIgnoredCCSFlags errors if --aws-... were used without --ccs.
func CheckIgnoredCCSFlags(ccs cluster.CCS) error {
	if !ccs.Enabled {
		bad := []string{}
		if ccs.AWS.AccountID != "" {
			bad = append(bad, "--aws-account-id")
		}
		if ccs.AWS.AccessKeyID != "" {
			bad = append(bad, "--aws-access-key-id")
		}
		if ccs.AWS.SecretAccessKey != "" {
			bad = append(bad, "--aws-secret-access-key")
		}
		if len(bad) == 1 {
			return fmt.Errorf("%s flag is meaningless without --ccs", bad[0])
		} else if len(bad) > 1 {
			return fmt.Errorf("%s flags are meaningless without --ccs",
				strings.Join(bad, ", "))
		}
	}
	return nil
}

func AddProviderFlag(fs *pflag.FlagSet, value *string) {
	fs.StringVar(
		value,
		"provider",
		"aws",
		"The cloud provider to create the cluster on",
	)
}

// ApplyParameterFlag applies the value of the '--parameter' command line flag to the given
// request.
func ApplyParameterFlag(request interface{}, values []string) {
	applyNVFlag(request, "Parameter", values)
}

// ApplyHeaderFlag applies the value of the '--header' command line flag to the given request.
func ApplyHeaderFlag(request interface{}, values []string) {
	applyNVFlag(request, "Header", values)
}

// applyNVFlag finds the method with the given name in a request and calls it to set a collection of
// name value pairs.
func applyNVFlag(request interface{}, method string, values []string) {
	// Find the method:
	callable := reflect.ValueOf(request).MethodByName(method)
	if !callable.IsValid() {
		return
	}

	// Split the values into name value pairs and call the method for each one:
	for _, value := range values {
		var name string
		position := strings.Index(value, "=")
		if position != -1 {
			name = value[:position]
			value = value[position+1:]
		} else {
			name = value
			value = ""
		}
		args := []reflect.Value{
			reflect.ValueOf(name),
			reflect.ValueOf(value),
		}
		callable.Call(args)
	}
}

// ApplyBodyFlag applies the value of the '--body' command line flag to the given request.
func ApplyBodyFlag(request *sdk.Request, value string) error {
	var body []byte
	var err error
	if value != "" {
		// #nosec G304
		body, err = ioutil.ReadFile(value)
	} else {
		if isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stderr.Fd()) {
			fmt.Fprintln(os.Stderr, "No --body file specified, reading request body from stdin:")
		}
		body, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil {
		return err
	}
	request.Bytes(body)
	return nil
}

// ApplyPathArg applies the value of the path given in the command line to the given request.
func ApplyPathArg(request *sdk.Request, value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	request.Path(parsed.Path)
	query := parsed.Query()
	for name, values := range query {
		for _, value := range values {
			request.Parameter(name, value)
		}
	}
	return nil
}

// CheckOneOf returns error if flag has been set and is not one of given options.
// It's appropriate for both optional flags (no error not given)
// and required flags (Cobra validated they're given before command .Run).
func CheckOneOf(fs *pflag.FlagSet, flagName string, options []string) error {
	if fs.Changed(flagName) {
		return requireOneOf(fs, flagName, options)
	}
	return nil
}

// requireOneOf returns error if flag is not one of given options.
func requireOneOf(fs *pflag.FlagSet, flagName string, options []string) error {
	flag := fs.Lookup(flagName)
	if flag == nil {
		return fmt.Errorf("no such flag %q", flagName)
	}

	if !sets.NewString(options...).Has(flag.Value.String()) {
		return fmt.Errorf("A valid --%s must be specified.\nValid options: %+v", flagName, options)
	}
	return nil
}

func Split(r rune) bool {
	return r == '=' || r == ':'
}
