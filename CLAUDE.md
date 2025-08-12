# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

OCM CLI is a command-line tool for interacting with the OpenShift Cluster Manager (OCM) API at api.openshift.com. The codebase is a Go application built around Cobra for command structure and the OCM SDK for API interactions.

## Development Commands

### Building
- `make` or `make cmds` - Build all command binaries
- `make install` - Install the `ocm` binary to Go's bin directory
- `go build "./cmd/ocm"` - Build just the main ocm binary

### Testing
- `make test` or `make tests` - Run all tests using Ginkgo
- `make ginkgo-install` - Install Ginkgo test framework locally
- Tests are located in the `tests/` directory and use Ginkgo/Gomega

### Code Quality
- `make lint` - Run golangci-lint using podman/docker container (version defined in `.golangciversion`)
- `make fmt` - Format Go code using gofmt
- `make clean` - Clean build artifacts

### Container Support
- Use `podman` instead of `docker` (configurable via `container_runner` variable)
- `make build_release_images` - Build release container images

## Architecture

### Command Structure
The CLI uses Cobra for command hierarchy with commands organized in `cmd/ocm/` by functionality:
- **Core API operations**: `get`, `post`, `patch`, `delete` for direct API interaction
- **Resource management**: `create`, `edit`, `describe`, `list` for higher-level operations
- **Authentication**: `login`, `logout`, `token`, `whoami`
- **Cluster operations**: `cluster/` subcommands for cluster lifecycle
- **Account management**: `account/` subcommands for organizations, users, roles
- **Utilities**: `config`, `completion`, `version`, `tunnel`

### Key Packages
- `pkg/ocm/` - Core OCM SDK connection and authentication handling
- `pkg/arguments/` - Command-line argument parsing and interactive prompts
- `pkg/config/` - Configuration file management (~/.config/ocm/ocm.json)
- `pkg/output/` - Output formatting (JSON, table, etc.) with YAML table definitions
- `pkg/cluster/` - Cluster-specific operations and utilities
- `pkg/properties/` - Environment variable and property handling

### Plugin System
- Supports plugin extensions with `ocm-` prefix binaries in PATH
- Plugin discovery and execution handled in `pkg/plugin/`
- `ocm plugin list` command to discover available plugins

### Connection Management
- OCM SDK connections built through `pkg/ocm/connection-builder/`
- Supports multiple authentication methods and environments
- Environment variable `OCM_CONFIG` for alternate config files
- Keyring support via `OCM_KEYRING` for secure credential storage

### Testing Strategy
- Integration tests in `tests/` package using Ginkgo/Gomega
- Platform-specific tests with build tags (darwin, windows, linux)
- Mock API responses for testing without live API calls
- Unit tests embedded within packages using standard testing patterns

## Development Patterns

### Error Handling
- Uses pkg/errors for error wrapping and context
- User-friendly error messages with specific handling for common scenarios (e.g., expired tokens)
- Debug output controlled via `--debug` flag

### Configuration
- Configuration stored in `~/.config/ocm/ocm.json` (or platform equivalent)
- Environment variable overrides supported (OCM_CONFIG, OCM_KEYRING, etc.)
- URL override via environment for development/testing

### Output Formatting
- Consistent table output using YAML definitions in `pkg/output/tables/`
- JSON output for programmatic consumption
- Color support via jsoncolor for enhanced readability

### API Integration
- Built on ocm-sdk-go for type-safe API interactions
- Connection builder pattern for flexible authentication
- Automatic token refresh and session management