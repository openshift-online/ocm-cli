---
name: go-toolchain-bump
description: Update this repository's Go toolchain version across module metadata, CI, and container build configuration.
---

# Go Toolchain Bump

Use this skill when bumping the Go toolchain version for `ocm-cli`.

## Version Surfaces

Update every pinned Go toolchain surface that should move with the bump:

- `go.mod`: update the `go` directive to the requested version, for example `go 1.26.10`.
- `.github/workflows/check-pull-request.yaml`: update both `actions/setup-go` `go-version` values.
- `.github/workflows/publish-release.yaml`: update the release job `actions/setup-go` `go-version`.
- `docker/Dockerfile`: update the builder image tag in `registry.access.redhat.com/ubi9/go-toolset:<version>`.

Also scan for additional pins before finishing:

```bash
rg -n "go-version|go 1\\.|GO_VERSION|GOLANG|go-toolset|setup-go|go1\\." go.mod .github .tekton docker -S
```

The `.tekton` pipeline usually delegates to `docker/Dockerfile` and gomod prefetching instead of pinning Go directly, but verify that assumption each time.

## Container Tag Verification

Red Hat UBI Go toolset tags may not always match upstream Go patch versions. Before pinning the Dockerfile, verify the requested tag exists:

```bash
podman manifest inspect registry.access.redhat.com/ubi9/go-toolset:<version>
```

After pinning, verify the base image's Go version:

```bash
podman run --rm registry.access.redhat.com/ubi9/go-toolset:<version> go version
```

The reported version should match the requested toolchain exactly. If the repo intentionally uses a minimum version rather than an exact container version, document that decision in the PR.

## Module Tidy

Run `go mod tidy` after changing `go.mod`. Keep Go caches outside the workspace when useful:

```bash
GOCACHE=/tmp/ocm-cli-go-build GOMODCACHE=/tmp/ocm-cli-go-mod go mod tidy
```

Do not include unrelated dependency upgrades unless `go mod tidy` requires them for the requested toolchain bump.

## Verification

For local verification, rebuild the ignored CLI binary before running integration tests because `tests` executes `../ocm`:

```bash
GOCACHE=/tmp/ocm-cli-go-build GOMODCACHE=/tmp/ocm-cli-go-mod go build -o ocm ./cmd/ocm
GOCACHE=/tmp/ocm-cli-go-build GOMODCACHE=/tmp/ocm-cli-go-mod go test ./tests -run TestCLI
```

If running inside a sandbox, tests that use `httptest` may fail with `socket: operation not permitted`; rerun those integration tests in an environment where local TCP listeners are allowed.

Finish by checking the tracked diff and leaving generated or ignored binaries unstaged:

```bash
git status --short
git diff --stat
```
