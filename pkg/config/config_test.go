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

package config

import (
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint
	. "github.com/onsi/gomega"    // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Armed", func() {
	It("Is armed if contains user name and password", func() {
		config := &Config{
			User:     "my-user",
			Password: "my-password",
			URL:      "http://my-server.example.com",
			TokenURL: "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeTrue())
		Expect(reason).To(BeEmpty())
	})

	It("Is armed if contains client identifier and secret", func() {
		config := &Config{
			ClientID:     "my-client",
			ClientSecret: "my-secret",
			URL:          "http://my-server.example.com",
			TokenURL:     "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeTrue())
		Expect(reason).To(BeEmpty())
	})

	It("Is armed if contains valid access token", func() {
		config := &Config{
			AccessToken: MakeTokenString("Bearer", 15*time.Minute),
			URL:         "http://my-server.example.com",
			TokenURL:    "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeTrue())
		Expect(reason).To(BeEmpty())
	})

	It("Is armed if contains valid refresh token", func() {
		config := &Config{
			AccessToken: MakeTokenString("Refresh", 10*time.Hour),
			URL:         "http://my-server.example.com",
			TokenURL:    "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeTrue())
		Expect(reason).To(BeEmpty())
	})

	It("Is armed if contains expired access token but valid refresh token", func() {
		config := &Config{
			AccessToken:  MakeTokenString("Access", -5*time.Minute),
			RefreshToken: MakeTokenString("Refresh", 10*time.Hour),
			URL:          "http://my-server.example.com",
			TokenURL:     "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeTrue())
		Expect(reason).To(BeEmpty())
	})

	It("Is armed if contains valid access token but expired refresh token", func() {
		config := &Config{
			AccessToken:  MakeTokenString("Access", 15*time.Minute),
			RefreshToken: MakeTokenString("Refresh", -10*time.Hour),
			URL:          "http://my-server.example.com",
			TokenURL:     "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeTrue())
		Expect(reason).To(BeEmpty())
	})

	It("Isn't armed if contains expired access token only", func() {
		config := &Config{
			AccessToken: MakeTokenString("Bearer", -5*time.Minute),
			URL:         "http://my-server.example.com",
			TokenURL:    "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeFalse())
		Expect(reason).To(Equal("access token is expired"))
	})

	It("Isn't armed if contains expired refresh token only", func() {
		config := &Config{
			RefreshToken: MakeTokenString("Refresh", -5*time.Minute),
			URL:          "http://my-server.example.com",
			TokenURL:     "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeFalse())
		Expect(reason).To(Equal("refresh token is expired"))
	})

	It("Isn't armed if contains expired access and refresh tokens", func() {
		config := &Config{
			AccessToken:  MakeTokenString("Access", -5*time.Minute),
			RefreshToken: MakeTokenString("Refresh", -5*time.Minute),
			URL:          "http://my-server.example.com",
			TokenURL:     "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeFalse())
		Expect(reason).To(Equal("access and refresh tokens are expired"))
	})

	It("Isn't armed if it contains user name but no password", func() {
		config := &Config{
			User:     "my-user",
			URL:      "http://my-server.example.com",
			TokenURL: "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeFalse())
		Expect(reason).To(Equal("credentials aren't set"))
	})

	It("Isn't armed if it contains client identifier but no secret", func() {
		config := &Config{
			ClientID: "my-client",
			URL:      "http://my-server.example.com",
			TokenURL: "http://my-sso.example.com",
		}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeFalse())
		Expect(reason).To(Equal("credentials aren't set"))
	})

	It("Isn't armed if empty", func() {
		config := &Config{}
		armed, reason, err := config.Armed()
		Expect(err).ToNot(HaveOccurred())
		Expect(armed).To(BeFalse())
		Expect(reason).To(Equal("credentials aren't set"))
	})
})
