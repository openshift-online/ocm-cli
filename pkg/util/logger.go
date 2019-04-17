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

package util

import (
	"github.com/golang/glog"

	"gitlab.cee.redhat.com/service/uhc-sdk/pkg/client"
)

// NewLogger creates a new logger with the debug level enabled according to the given value.
func NewLogger(debug bool) (logger client.Logger, err error) {
	debugV := glog.Level(1)
	if debug {
		debugV = glog.Level(0)
	}
	logger, err = client.NewGlogLoggerBuilder().
		DebugV(debugV).
		Build()
	return
}
