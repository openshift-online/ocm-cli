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
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega/types"
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
	env    map[string]string
	args   []string
	config string
	in     []byte
}

// CommandResult contains the result of executing a CLI command.
type CommandResult struct {
	configFile string
	configData []byte
	out        []byte
	err        []byte
	exitCode   int
}

// NewCommand creates a new CLI command runner.
func NewCommand() *CommandRunner {
	return &CommandRunner{
		env: map[string]string{},
	}
}

// ConfigString sets the content of the CLI configuration file.
func (r *CommandRunner) ConfigString(template string, vars ...interface{}) *CommandRunner {
	r.config = sdktesting.EvaluateTemplate(template, vars...)
	return r
}

// Env sets an environment variable to the CLI command.
func (r *CommandRunner) Env(name, value string) *CommandRunner {
	r.env[name] = value
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

// In sets the standard input for the CLI command.
func (r *CommandRunner) InString(value string) *CommandRunner {
	r.in = []byte(value)
	return r
}

// Run runs the command.
func (r *CommandRunner) Run(ctx context.Context) *CommandResult {
	var err error

	// Create a temporary directory for the configuration file, so that we don't interfere with
	// the configuration that may already exist for the user running the tests.
	tmpDir, err := ioutil.TempDir("", "ocm-test-*.d")
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	defer func() {
		err = os.RemoveAll(tmpDir)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}()

	// Create the configuration file:
	configFile := filepath.Join(tmpDir, ".ocm.json")
	if r.config != "" {
		err = ioutil.WriteFile(configFile, []byte(r.config), 0600)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	// Parse the current environment into a map so that it is easy to update it:
	envMap := map[string]string{}
	for _, text := range os.Environ() {
		index := strings.Index(text, "=")
		var name string
		var value string
		if index > 0 {
			name = text[0:index]
			value = text[index+1:]
		} else {
			name = text
			value = ""
		}
		envMap[name] = value
	}

	// Add the environment variables:
	for name, value := range r.env {
		envMap[name] = value
	}

	// Add to the environment the variable that points to a configuration file:
	envMap["OCM_CONFIG"] = configFile

	// Reconstruct the environment list:
	envList := make([]string, 0, len(envMap))
	for name, value := range envMap {
		envList = append(envList, name+"="+value)
	}

	// Create the buffers:
	inBuf := &bytes.Buffer{}
	if r.in != nil {
		inBuf.Write(r.in)
	}
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	// Create the command:
	cmd := exec.Command(binary, r.args...)
	cmd.Env = envList
	cmd.Stdin = inBuf
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf

	// Run the command:
	err = cmd.Run()
	switch err.(type) {
	case *exec.ExitError:
		// Nothing, this is a normal situation and the caller is expected to check it using
		// the `ExitCode` method.
	default:
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	// Read the potentially created or modified configuration file:
	_, err = os.Stat(configFile)
	if errors.Is(err, os.ErrNotExist) {
		configFile = ""
	} else if err != nil {
		Expect(err).ToNot(HaveOccurred())
	}
	var configData []byte
	if configFile != "" {
		configData, err = ioutil.ReadFile(configFile)
		Expect(err).ToNot(HaveOccurred())
	}

	// Create the result:
	result := &CommandResult{
		configFile: configFile,
		configData: configData,
		out:        outBuf.Bytes(),
		err:        errBuf.Bytes(),
		exitCode:   cmd.ProcessState.ExitCode(),
	}

	return result
}

// ConfigFile returns the name of the configuration file. It will be an empty string if the config
// file doesn't exist. Note that at the time this is called the actual file has already been
// deleted. If you need to check the contents of the file use the ConfigString method.
func (r *CommandResult) ConfigFile() string {
	return r.configFile
}

// ConfigString returns the content of the configuration file.
func (r *CommandResult) ConfigString() string {
	return string(r.configData)
}

// OutString returns the standard output of the CLI command.
func (r *CommandResult) OutString() string {
	return string(r.out)
}

// OutLines returns the standard output of the CLI command as an array of strings.
func (r *CommandResult) OutLines() []string {
	// Split the output into lines:
	lines := strings.Split(string(r.out), "\n")

	// If there is a blank line at the end remove it:
	count := len(lines)
	if count > 0 && lines[count-1] == "" {
		lines = lines[0 : count-1]
	}

	// Return the lines:
	return lines
}

// Err returns the standard errour output of the CLI command.
func (r *CommandResult) ErrString() string {
	return string(r.err)
}

// ExitCode returns the exit code of the CLI command.
func (r *CommandResult) ExitCode() int {
	return r.exitCode
}

// MatchJSONTemplate succeeds if actual is a string or stringer of JSON that matches the result of
// evaluating the given template with the given arguments.
func MatchJSONTemplate(template string, args ...interface{}) types.GomegaMatcher {
	return MatchJSON(sdktesting.EvaluateTemplate(template, args...))
}
