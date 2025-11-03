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

PROJECT_PATH := $(PWD)
LOCAL_BIN_PATH := $(PROJECT_PATH)/bin
GINKGO := $(LOCAL_BIN_PATH)/ginkgo

# Allow overriding: `make lint container_runner=docker`.
container_runner:=podman

.PHONY: all
all: cmds

.PHONY: cmds
cmds:
	for cmd in $$(ls cmd); do \
		go build "./cmd/$${cmd}" || exit 1; \
	done

.PHONY: install
install:
	go install ./cmd/ocm

.PHONY: ginkgo-install
ginkgo-install:
	@GOBIN=$(LOCAL_BIN_PATH) go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4

.PHONY: tools
tools: ginkgo-install

.PHONY: test tests
test tests: cmds tools
	$(GINKGO) run -r

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
		*-linux-arm64 \
		*-linux-ppc64le \
		*-linux-s390x \
		*-windows-amd64 \
		*.sha256 \
		$(NULL)

.PHONY: build_release_images
build_release_images:
	bash ./hack/build_release_images.sh


# NOTE: This requires the tool podman to be installed in the calling environment
.PHONY: image
image:
	bash ./hack/build_image.sh "${IMAGE_REPOSITORY}" "${IMAGE_TAG}" "${IMAGE_NAME}"

# NOTE: This requires the tool hermeto v0.41.0+ to be installed in the calling environment.
.PHONY: hermetic_image
hermetic_image:
	bash ./hack/build_hermetic_image.sh "${IMAGE_REPOSITORY}" "${IMAGE_TAG}" "${IMAGE_NAME}"
