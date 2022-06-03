/*
Copyright (c) 2018 Red Hat, Inc.

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

// Package dump contains functions used to dump JSON documents to the output of the tool.
package dump

import (
	"encoding/json"
	"io"
	"runtime"

	"github.com/nwidger/jsoncolor"
	"github.com/openshift-online/ocm-cli/pkg/output"
)

// Pretty dumps the given data to the given stream so that it looks pretty. If the data is a valid
// JSON document then it will be indented before printing it. If the stream is a terminal then the
// output will also use colors.
func Pretty(stream io.Writer, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var data interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return dumpBytes(stream, body)
	}
	if output.IsTerminal(stream) && !isWindows() {
		return dumpColor(stream, data)
	}
	return dumpMonochrome(stream, data)
}

func dumpColor(stream io.Writer, data interface{}) error {
	encoder := jsoncolor.NewEncoder(stream)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func dumpMonochrome(stream io.Writer, data interface{}) error {
	encoder := json.NewEncoder(stream)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Single functions exactly the same as Pretty except it generates a single line without indentation
// or any other white space.
func Single(stream io.Writer, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var data interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return dumpBytes(stream, body)
	}
	if output.IsTerminal(stream) && !isWindows() {
		return dumpColorSingleLine(stream, data)
	}
	return dumpMonochromeSingleLine(stream, data)
}

func dumpColorSingleLine(stream io.Writer, data interface{}) error {
	encoder := jsoncolor.NewEncoder(stream)
	err := encoder.Encode(data)
	if err != nil {
		return err
	}
	_, err = stream.Write([]byte("\n"))
	return err
}

func dumpMonochromeSingleLine(stream io.Writer, data interface{}) error {
	encoder := json.NewEncoder(stream)
	return encoder.Encode(data)
}

func dumpBytes(stream io.Writer, data []byte) error {
	_, err := stream.Write(data)
	if err != nil {
		return err
	}
	_, err = stream.Write([]byte("\n"))
	return err
}

// isWindows checks if the operating system is Windows.
func isWindows() bool {
	return runtime.GOOS == "windows"
}
