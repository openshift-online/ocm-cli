#
# Copyright (c) 2018 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Ensure go modules are enabled:
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

# Disable CGO so that we always generate static binaries:
export CGO_ENABLED=0

# Allow overriding: `make lint container_runner=docker`.
container_runner:=podman

.PHONY: cmds
cmds:
	for cmd in $$(ls cmd); do \
		go build -o "$${cmd}" "./cmd/$${cmd}" || exit 1; \
	done

.PHONY: install
install:
	go install ./cmd/ocm

.PHONY: test tests
test tests: cmds
	ginkgo -r cmd pkg tests

.PHONY: fmt
fmt:
	gofmt -s -l -w cmd pkg tests

.PHONY: lint
lint:
	$(container_runner) run --rm --security-opt label=disable --volume="$(PWD):/app" --workdir=/app \
		golangci/golangci-lint:v$(shell cat .golangciversion) \
		golangci-lint run

.PHONY: clean
clean:
	rm -rf \
		$$(ls cmd) \
		*-darwin-amd64 \
		*-linux-amd64 \
		*-windows-amd64 \
		*.sha256 \
		$(NULL)
