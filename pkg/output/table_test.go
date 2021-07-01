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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint
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
		err = table.WriteRow(object)
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
		err = table.WriteRow(object)
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
		err = table.WriteRow(object)
		Expect(err).ToNot(HaveOccurred())

		// Check the generated text:
		Expect(buffer.String()).To(MatchRegexp(
			`^123\s+my_123\s+your_123\s*$`,
		))
	})
})
