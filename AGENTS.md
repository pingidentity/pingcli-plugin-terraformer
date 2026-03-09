# General Guidelines

Consult the `README.md` file in the root directory for general project context and guidelines.

## Project Context

This is the official Ping Identity CLI Terraformer plugin. It exports PingOne resources to Terraform configuration using a schema-driven architecture.

- **Repository**: `github.com/pingidentity/pingcli-plugin-terraformer`
- **Architecture**: YAML-based schema definitions drive a generic processing engine that produces HCL or Terraform JSON output
- **Key directories**: `definitions/` (YAML schemas), `internal/core/` (processing engine), `internal/formatters/` (output formatters), `internal/platform/` (platform-specific clients)

## File Creation — Mandatory Workaround

The `create_file` tool corrupts Go files. It duplicates the `package` line and garbles remaining content.

**Required pattern for creating new `.go` files:**

1. Write a Python generator script that produces the Go file content via `print()` or `open().write()`.
2. Save the `.py` script using `create_file`.
3. Run the script via `run_in_terminal` to generate the `.go` file: `python3 gen.py && rm gen.py`.

**Never use `create_file` directly for `.go` files.**

## Architecture Documentation

See `contributing/ARCHITECTURE.md` for system design and `contributing/DEVELOPER_GUIDE.md` for development workflows.
