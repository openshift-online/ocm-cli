//go:build !windows
// +build !windows

/*
Copyright (c) 2024 Red Hat, Inc.

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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	"github.com/openshift-online/ocm-cli/cmd/ocm/login"
	"github.com/openshift-online/ocm-cli/pkg/properties"
	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

// This test requires `pass` to be installed.
// macOS: `brew install pass`
// linux: `sudo apt-get install pass` or `sudo yum install pass`

const keyring_dir = "keyring-pass-test-*"

func runCmd(cmds ...string) {
	cmd := exec.Command(cmds[0], cmds[1:]...) //nolint:gosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd)
		fmt.Println(string(out))
		Fail(err.Error())
	}
}

var _ = Describe("Pass Keyring", Ordered, func() {
	BeforeAll(func() {
		// Check if 'pass' is available in PATH
		_, err := exec.LookPath("pass")
		if err != nil {
			Skip("Skipping Pass keyring tests: 'pass' command not found in PATH. Install with: sudo dnf install pass")
		}

		pwd, err := os.Getwd()
		if err != nil {
			Fail(err.Error())
		}
		pwdParent := filepath.Dir(pwd)

		// the default temp directory can't be used because gpg-agent complains with "socket name too long"
		tmpdir, err := os.MkdirTemp("/tmp", keyring_dir)
		if err != nil {
			Fail(err.Error())

		}
		tmpdirPass, err := os.MkdirTemp("/tmp", ".password-store-*")
		if err != nil {
			Fail(err.Error())
		}

		// Initialise a blank GPG homedir; import & trust the test key
		gnupghome := filepath.Join(tmpdir, ".gnupg")
		err = os.Mkdir(gnupghome, os.FileMode(int(0700)))
		if err != nil {
			Fail(err.Error())
		}
		os.Setenv("GNUPGHOME", gnupghome)
		os.Setenv("PASSWORD_STORE_DIR", tmpdirPass)
		os.Unsetenv("GPG_AGENT_INFO")
		os.Unsetenv("GPG_TTY")

		runCmd("gpg", "--batch", "--import", filepath.Join(pwdParent, "testdata", "test-gpg.key"))
		runCmd("gpg", "--batch", "--import-ownertrust", filepath.Join(pwdParent, "testdata", "test-ownertrust-gpg.txt"))
		runCmd("pass", "init", "ocm-devel@redhat.com")

		DeferCleanup(func() {
			os.Unsetenv("GNUPGHOME")
			os.Unsetenv("PASSWORD_STORE_DIR")
			os.RemoveAll(filepath.Join("/tmp", keyring_dir))
		})
	})

	var ctx context.Context
	var ssoServer *Server

	BeforeEach(func() {
		// Create the context
		ctx = context.Background()

		// Create the server
		ssoServer = MakeTCPServer()
	})

	AfterEach(func() {
		// Close the server
		ssoServer.Close()
	})

	When("Using OCM_KEYRING", func() {
		AfterEach(func() {
			// reset keyring
			os.Setenv(properties.KeyringEnvKey, "")
		})

		It("Stores/Removes configuration in Pass", func() {
			// Create the token
			accessToken := MakeTokenString("Bearer", 15*time.Minute)

			// Prepare the server
			ssoServer.AppendHandlers(
				RespondWithAccessToken(accessToken),
			)

			os.Setenv(properties.KeyringEnvKey, "pass")

			// Run login
			result := NewCommand().
				Args(
					"login",
					"--client-id", "my-client",
					"--client-secret", "my-secret",
					"--token-url", ssoServer.URL(),
				).
				Run(ctx)

			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ExitCode()).To(BeZero())
			// Verify no config file data exists
			Expect(result.ConfigFile()).To(BeEmpty())
			Expect(result.ConfigString()).To(BeEmpty())

			// Check the content of the keyring
			result = NewCommand().
				Args(
					"config",
					"get",
					"access_token",
				).
				Run(ctx)
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutLines()[0]).To(ContainSubstring(accessToken))

			// Remove the configuration from the keyring
			result = NewCommand().
				Args(
					"logout",
				).
				Run(ctx)
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutLines()).To(BeEmpty())

			// Ensure the keyring is empty
			result = NewCommand().
				Args(
					"config",
					"get",
					"access_token",
				).
				Run(ctx)
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutLines()[0]).To(BeEmpty())
		})

		Context("Using auth code", func() {
			It("Stores/Removes configuration in Keychain", func() {
				os.Setenv(properties.KeyringEnvKey, "pass")

				login.InitiateAuthCode = InitiateAuthCodeMock

				cmd := login.Cmd
				cmd.SetArgs([]string{"--use-auth-code"})
				err := cmd.Execute()
				Expect(err).NotTo(HaveOccurred())

				// Check the content of the keyring
				result := NewCommand().
					Args(
						"config",
						"get",
						"access_token",
					).
					Run(ctx)
				Expect(result.ExitCode()).To(BeZero())
				Expect(result.ErrString()).To(BeEmpty())
				Expect(result.OutLines()[0]).NotTo(BeEmpty())

				// Remove the configuration from the keyring
				result = NewCommand().
					Args(
						"logout",
					).
					Run(ctx)
				Expect(result.ExitCode()).To(BeZero())
				Expect(result.ErrString()).To(BeEmpty())
				Expect(result.OutLines()).To(BeEmpty())

				// Ensure the keyring is empty
				result = NewCommand().
					Args(
						"config",
						"get",
						"access_token",
					).
					Run(ctx)
				Expect(result.ErrString()).To(BeEmpty())
				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutLines()[0]).To(BeEmpty())
			})
		})
	})
})
