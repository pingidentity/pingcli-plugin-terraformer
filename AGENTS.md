# General Guidelines

Consult the `README.md` file in the root directory for general project context and guidelines.

## Project Context

This is the official Ping Identity CLI Terraformer plugin. It exports PingOne resources to Terraform configuration using a schema-driven architecture.

- **Repository**: `github.com/pingidentity/pingcli-plugin-terraformer`
- **Architecture**: YAML-based schema definitions drive a generic processing engine that produces HCL or Terraform JSON output
- **Key directories**: `definitions/` (YAML schemas), `internal/core/` (processing engine), `internal/formatters/` (output formatters), `internal/platform/` (platform-specific clients)

## Architecture Documentation

See `contributing/ARCHITECTURE.md` for system design and `contributing/DEVELOPER_GUIDE.md` for development workflows.
