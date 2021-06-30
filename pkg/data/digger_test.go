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

package data

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Digger", func() {
	var digger *Digger

	BeforeEach(func() {
		var err error

		// Create a context:
		ctx := context.Background()

		// Create the digger:
		digger, err = NewDigger().
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	DescribeTable(
		"Dig",
		func(object interface{}, path string, expected interface{}) {
			actual := digger.Dig(object, path)
			if expected == nil {
				Expect(actual).To(BeNil())
			} else {
				Expect(actual).To(Equal(expected))
			}
		},
		Entry(
			"Empty path on nil",
			nil,
			"",
			nil,
		),
		Entry(
			"Path with one segment on nil",
			nil,
			"a",
			nil,
		),
		Entry(
			"Path with two segments on nil",
			nil,
			"a.b",
			nil,
		),
		Entry(
			"Empty path on pointer",
			&subject{},
			"",
			&subject{},
		),
		Entry(
			"Path to method returning value",
			&subject{},
			"a",
			"a_value",
		),
		Entry(
			"Path to method returning value and true",
			&subject{},
			"b",
			"b_value",
		),
		Entry(
			"Path to method returning value and false",
			&subject{},
			"c",
			nil,
		),
		Entry(
			"Path with one underscore",
			&subject{},
			"my_d",
			"my_d_value",
		),
		Entry(
			"Path with two segments",
			&subject{},
			"s.a",
			"s/a_value",
		),
		Entry(
			"Path with three segments",
			&subject{},
			"s.s.a",
			"s/s/a_value",
		),
		Entry(
			"Integer method",
			&subject{},
			"e",
			123,
		),
		Entry(
			"Float method",
			&subject{},
			"f",
			1.23,
		),
		Entry(
			"Time method",
			&subject{},
			"g",
			time.Unix(1, 23),
		),
		Entry(
			"Duration method",
			&subject{},
			"h",
			time.Duration(123),
		),
		Entry(
			"Prefers `Get...` and returns nil for false",
			&subject{},
			"i",
			nil,
		),
		Entry(
			"Doesn't remove `Get...` prefix if not followed by word",
			&subject{},
			"getaway",
			"getaway_value",
		),
		Entry(
			"Value receiver",
			subject{},
			"j",
			"j_value",
		),
		Entry(
			"String field",
			subject{
				K: "k_value",
			},
			"k",
			"k_value",
		),
		Entry(
			"Integer field",
			subject{
				L: 123,
			},
			"l",
			123,
		),
		Entry(
			"Float field",
			subject{
				M: 1.23,
			},
			"m",
			1.23,
		),
		Entry(
			"Time field",
			subject{
				O: time.Unix(1, 23),
			},
			"o",
			time.Unix(1, 23),
		),
		Entry(
			"Duration field",
			subject{
				P: time.Duration(123),
			},
			"p",
			time.Duration(123),
		),
		Entry(
			"Nil field",
			subject{
				Q: nil,
			},
			"q",
			nil,
		),
		Entry(
			"Field that doesn't exist",
			subject{},
			"does_not_exist",
			nil,
		),
	)
})

// subject is used as a subject of the digger tests.
type subject struct {
	prefix string
	K      string
	L      int
	M      float64
	O      time.Time
	P      time.Duration
	Q      *int
}

func (s *subject) S() *subject {
	return &subject{
		prefix: s.prefix + "s/",
	}
}

func (s *subject) A() string {
	return s.prefix + "a_value"
}

func (s *subject) GetB() (value string, ok bool) {
	value = s.prefix + "b_value"
	ok = true
	return
}

func (s *subject) GetC() (value string, ok bool) {
	value = s.prefix + "c_value"
	ok = false
	return
}

func (s *subject) MyD() string {
	return s.prefix + "my_d_value"
}

func (s *subject) E() int {
	return 123
}

func (s *subject) F() float64 {
	return 1.23
}

func (s *subject) G() time.Time {
	return time.Unix(1, 23)
}

func (s *subject) H() time.Duration {
	return time.Duration(123)
}

func (s *subject) I() string {
	return ""
}

func (s *subject) GetI() (value string, ok bool) {
	value = ""
	ok = false
	return
}

func (s *subject) Getaway() string {
	return "getaway_value"
}

func (s subject) J() string {
	return "j_value"
}
