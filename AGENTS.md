# AGENTS.md

This file provides guidance to AI coding assistants when working with this repository.

## Project Overview

OCM CLI (`ocm`) is a command-line tool for interacting with the OpenShift Cluster Manager (OCM) API. Built in Go with Cobra for command structure and the OCM SDK (`ocm-sdk-go`) for type-safe API interactions.

## Build & Test Commands

```bash
make                  # Build all command binaries
make install          # Install ocm binary to $GOPATH/bin
make test             # Run all tests (Ginkgo)
make lint             # Run golangci-lint via container
make fmt              # Format Go source code
make clean            # Remove build artifacts
```

## Architecture

### Command Structure (`cmd/ocm/`)
- **Core API**: `get`, `post`, `patch`, `delete` — direct API interaction
- **Resource management**: `create`, `edit`, `describe`, `list` — higher-level operations
- **Authentication**: `login`, `logout`, `token`, `whoami`
- **Cluster operations**: `cluster/` subcommands for cluster lifecycle
- **Account management**: `account/` subcommands for organizations, users, roles
- **Utilities**: `config`, `completion`, `version`, `tunnel`

### Key Packages
- **pkg/ocm/** — Core OCM SDK connection and authentication handling
- **pkg/arguments/** — Command-line argument parsing and interactive prompts
- **pkg/config/** — Configuration file management (`~/.config/ocm/ocm.json`)
- **pkg/output/** — Output formatting (JSON, table) with YAML table definitions
- **pkg/cluster/** — Cluster-specific operations and utilities
- **pkg/plugin/** — Plugin discovery and execution (`ocm-` prefix binaries in PATH)

### Connection & Auth
- OCM SDK connections built through `pkg/ocm/connection-builder/`
- Multiple auth methods supported; environment variable `OCM_CONFIG` for alternate config
- Keyring support via `OCM_KEYRING` for secure credential storage
- Automatic token refresh and session management

## Key Conventions

- Module path: `github.com/openshift-online/ocm-cli`
- Ginkgo/Gomega for testing (tests in `tests/` directory)
- Use `podman` over `docker` (configurable via `container_runner` Make variable)
- Static binaries (CGO disabled)
- Plugin extensions use `ocm-` prefix naming convention
