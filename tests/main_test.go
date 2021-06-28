/*
Copyright (c) 2021 Red Hat, Inc.

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

package tests

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sdktesting "github.com/openshift-online/ocm-sdk-go/testing"

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI")
}

// binary is the path of the `ocm` binary that will be used in the tests.
var binary string

var _ = BeforeSuite(func() {
	// Check that the binary exists:
	binary = filepath.Join("..", "ocm")
	_, err := os.Stat(binary)
	Expect(err).ToNot(
		HaveOccurred(),
		"The '%s' binary doesn't exist, make sure to run 'make cmd' before running "+
			"these tests",
		binary,
	)
})

// CommandRunner contains the data and logic needed to run a CLI command.
type CommandRunner struct {
	args   []string
	config string
	tmp    string
	in     *bytes.Buffer
	out    *bytes.Buffer
	err    *bytes.Buffer
	cmd    *exec.Cmd
}

// NewCommand creates a new CLI command runner.
func NewCommand() *CommandRunner {
	return &CommandRunner{}
}

// Config sets the content of the CLI configuration file.
func (r *CommandRunner) Config(template string, vars ...interface{}) *CommandRunner {
	r.config = sdktesting.EvaluateTemplate(template, vars...)
	return r
}

// Arg adds a command line argument to the CLI command.
func (r *CommandRunner) Arg(value string) *CommandRunner {
	r.args = append(r.args, value)
	return r
}

// Args adds a set of command line arguments for the CLI command.
func (r *CommandRunner) Args(values ...string) *CommandRunner {
	r.args = append(r.args, values...)
	return r
}

// Run runs the command.
func (r *CommandRunner) Run(ctx context.Context) {
	var err error

	// Create a temporary directory for the configuration file, so that we don't interfere with
	// the configuration that may already exist for the user running the tests:
	r.tmp, err = ioutil.TempDir("", "ocm-test-*.d")
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Create the configuration file:
	config := filepath.Join(r.tmp, ".ocm.json")
	if r.config != "" {
		err = ioutil.WriteFile(config, []byte(r.config), 0600)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	// Remove from the environment the variable that points to the configuration file, if it exists:
	var env []string
	for _, text := range os.Environ() {
		index := strings.Index(text, "=")
		var name string
		if index > 0 {
			name = text[0:index]
		} else {
			name = text
		}
		if name != "OCM_CONFIG" {
			env = append(env, text)
		}
	}

	// Add to the environment the variable that points to a configuration file:
	env = append(env, "OCM_CONFIG="+config)

	// Create the buffers:
	r.in = &bytes.Buffer{}
	r.out = &bytes.Buffer{}
	r.err = &bytes.Buffer{}

	// Create the command:
	r.cmd = exec.Command(binary, r.args...)
	r.cmd.Env = env
	r.cmd.Stdin = r.in
	r.cmd.Stdout = r.out
	r.cmd.Stderr = r.err

	// Run the command:
	err = r.cmd.Run()
	switch err.(type) {
	case *exec.ExitError:
		// Nothing, this is a normal situation and the caller is expected to check it using
		// the `ExitCode` method.
	default:
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}
}

// OutString returns the standard output of the CLI command.
func (r *CommandRunner) OutString() string {
	return r.out.String()
}

// Err returns the standard errour output of the CLI command.
func (r *CommandRunner) ErrString() string {
	return r.err.String()
}

// ExitCode returns the exit code of the CLI command.
func (r *CommandRunner) ExitCode() int {
	return r.cmd.ProcessState.ExitCode()
}

// Close releases all the resources used to run the CLI command, like temporary files.
func (r *CommandRunner) Close() {
	var err error

	// Remove the temporary directory:
	if r.tmp != "" {
		err = os.RemoveAll(r.tmp)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}
}
