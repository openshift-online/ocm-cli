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

package output

import (
	"bytes"
	"context"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	. "github.com/onsi/ginkgo/v2" // nolint
	. "github.com/onsi/gomega"    // nolint
)

var _ = Describe("Table", func() {
	var ctx context.Context
	var buffer *bytes.Buffer
	var printer *Printer

	BeforeEach(func() {
		var err error

		// Create a context:
		ctx = context.Background()

		// Create the buffer:
		buffer = &bytes.Buffer{}

		// Create a printer that writes to a memory buffer so that we can check the results:
		printer, err = NewPrinter().
			Writer(buffer).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		var err error

		// Close the printer:
		if printer != nil {
			err = printer.Close()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	It("Writes headers", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("clusters").
			Columns(
				"id",
				"external_id",
				"name",
			).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Write the headers:
		err = table.WriteHeaders()
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		Expect(buffer.String()).To(MatchRegexp(
			`^ID\s+EXTERNAL ID\s+NAME\s*$`,
		))
	})

	It("Doesn't trim `external_id` column", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("clusters").
			Columns(
				"id",
				"external_id",
				"name",
			).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the object that will be written to the table:
		object, err := cmv1.NewCluster().
			ID("123").
			ExternalID("e30bac0b-b337-47d7-a378-2c302b4c868a").
			Name("mycluster").
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteObject(object)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		Expect(buffer.String()).To(MatchRegexp(
			`^123\s+e30bac0b-b337-47d7-a378-2c302b4c868a\s+mycluster\s*$`,
		))
	})

	It("Honors explicit column values", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("clusters").
			Columns("id", "my_column", "your_column").
			Value("my_column", "my_value").
			Value("your_column", "your_value").
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the object that will be written to the table:
		object, err := cmv1.NewCluster().
			ID("123").
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteObject(object)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		Expect(buffer.String()).To(MatchRegexp(
			`^123\s+my_value\s+your_value\s*$`,
		))
	})

	It("Honors calculated column", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("clusters").
			Columns("id", "my_column", "your_column").
			Value("my_column", func(object *cmv1.Cluster) string {
				return "my_" + object.ID()
			}).
			Value("your_column", func(object *cmv1.Cluster) string {
				return "your_" + object.ID()
			}).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the object that will be written to the table:
		object, err := cmv1.NewCluster().
			ID("123").
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteObject(object)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		Expect(buffer.String()).To(MatchRegexp(
			`^123\s+my_123\s+your_123\s*$`,
		))
	})

	It("Learns column widths from values", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("idps").
			Columns("name", "type").
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the object that will be written to the table:
		object, err := cmv1.NewIdentityProvider().
			Name("my_github").
			Type(cmv1.IdentityProviderTypeGithub).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteHeaders()
		Expect(err).ToNot(HaveOccurred())
		err = table.WriteObject(object)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		lines := strings.Split(buffer.String(), "\n")
		Expect(lines).To(HaveLen(3))
		Expect(lines[0]).To(Equal(`NAME       TYPE                  `))
		Expect(lines[1]).To(Equal(`my_github  GithubIdentityProvider`))
	})

	It("Learns column widths from headers", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("idps").
			Columns("name", "type").
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the object that will be written to the table:
		object, err := cmv1.NewIdentityProvider().
			Name("1").
			Type("my").
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteHeaders()
		Expect(err).ToNot(HaveOccurred())
		err = table.WriteObject(object)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		lines := strings.Split(buffer.String(), "\n")
		Expect(lines).To(HaveLen(3))
		Expect(lines[0]).To(Equal(`NAME  TYPE`))
		Expect(lines[1]).To(Equal(`1     my  `))
	})

	It("Honours disabled learning", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("clusters").
			Columns("id", "name").
			Learning(false).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the object that will be written to the table:
		object, err := cmv1.NewCluster().
			ID("123").
			Name("mycluster").
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteHeaders()
		Expect(err).ToNot(HaveOccurred())
		err = table.WriteObject(object)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		lines := strings.Split(buffer.String(), "\n")
		Expect(lines).To(HaveLen(3))
		Expect(lines[0]).To(Equal(`ID                                NAME                        `))
		Expect(lines[1]).To(Equal(`123                               mycluster                   `))
	})

	It("Honours learning limit", func() {
		// Create the table:
		table, err := printer.NewTable().
			Name("idps").
			Columns("name", "type").
			LearningLimit(2).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Create the objects that will be written to the table:
		first, err := cmv1.NewIdentityProvider().
			Name("123").
			Type("my_github").
			Build()
		Expect(err).ToNot(HaveOccurred())
		second, err := cmv1.NewIdentityProvider().
			Name("456").
			Type("your_github").
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Write the object to the table:
		err = table.WriteHeaders()
		Expect(err).ToNot(HaveOccurred())
		err = table.WriteObject(first)
		Expect(err).ToNot(HaveOccurred())
		err = table.WriteObject(second)
		Expect(err).ToNot(HaveOccurred())
		err = table.Close()
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		lines := strings.Split(buffer.String(), "\n")
		Expect(lines).To(HaveLen(4))
		Expect(lines[0]).To(Equal(`NAME  TYPE     `))
		Expect(lines[1]).To(Equal(`123   my_github`))
		Expect(lines[2]).To(Equal(`456   your_gith`))
	})
})
