# Ping CLI Terraformer Plugin

Export PingOne resources to Terraform configuration with automatic dependency resolution and import block generation.

## Features

- **Complete DaVinci Export**: Export PingOne environments, DaVinci flows, variables, connector instances, applications, and flow policies
- **Multiple Output Formats**: Supports Terraform HCL (`.tf`) or Terraform JSON (`.tf.json`) output
- **Automatic Dependency Resolution**: Generates Terraform references between resources
- **Import Block Generation**: Terraform import blocks to manage existing resources (Terraform 1.5+)
- **Module Structure**: Generates reusable Terraform modules with proper variable scaffolding
- **Dual Mode Operation**: Works as standalone CLI or Ping CLI plugin
- **Two-Environment Authentication**: Isolate credentials from exported resources

## Guides

- [Manage an Existing Environment](./guides/manage-existing-environment.md)
- [Migrate from Legacy DaVinci Provider](./guides/migrate-from-legacy-provider.md)

## Installation

### Pre-built Binaries (Recommended)

Download from [GitHub Releases](https://github.com/pingidentity/pingcli-plugin-terraformer/releases).

### Homebrew (macOS/Linux)

```bash
brew install pingidentity/tap/pingcli-plugin-terraformer
```

### Linux Package Managers

```bash
# Debian/Ubuntu
sudo dpkg -i pingcli-terraformer_*.deb

# RHEL/CentOS/Fedora
sudo rpm -i pingcli-terraformer_*.rpm

# Alpine
sudo apk add pingcli-terraformer_*.apk
```

### Docker

```bash
docker run --rm \
  -e PINGCLI_PINGONE_ENVIRONMENT_ID="..." \
  -e PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="..." \
  -e PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="..." \
  -v $(pwd)/output:/output \
  ghcr.io/pingidentity/pingcli-plugin-terraformer:latest \
  export --out /output
```

### From Source

```bash
git clone https://github.com/pingidentity/pingcli-plugin-terraformer.git
cd pingcli-plugin-terraformer
make build
```

## Prerequisites

- PingOne environment with DaVinci
- PingOne worker application with DaVinci API Read access
- Terraform 1.5+ (for import blocks)

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PINGCLI_PINGONE_ENVIRONMENT_ID` | Worker environment ID (authentication) |
| `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID` | Target environment to export (optional, defaults to worker env) |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID` | OAuth2 client ID |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET` | OAuth2 client secret |
| `PINGCLI_PINGONE_REGION_CODE` | Region code: NA, EU, AP, CA, AU |

### Two-Environment Model

Export resources from a different environment than where the worker app is configured:

```bash
pingcli-terraformer export \
  --pingone-worker-environment-id <auth-env-uuid> \
  --pingone-export-environment-id <target-env-uuid> \
  --pingone-worker-client-id <client-id> \
  --pingone-worker-client-secret <secret> \
  --pingone-region-code NA \
  --out ./output
```

## Usage

### Basic Export (HCL)

```bash
pingcli-terraformer export \
  --pingone-worker-environment-id <uuid> \
  --pingone-worker-client-id <client-id> \
  --pingone-worker-client-secret <secret> \
  --pingone-region-code NA \
  --out ./output
```

### Export as Terraform JSON

```bash
pingcli-terraformer export \
  --output-format tfjson \
  --out ./output
```

### Using Environment Variables

```bash
export PINGCLI_PINGONE_ENVIRONMENT_ID="..."
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="..."
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="..."
export PINGCLI_PINGONE_REGION_CODE="NA"

pingcli-terraformer export --out ./output
```

## Command Reference

### Export Command

| Flag | Default | Description |
|------|---------|-------------|
| `--pingone-worker-environment-id` | - | Worker environment ID |
| `--pingone-export-environment-id` | Worker env | Target environment ID |
| `--pingone-worker-client-id` | - | OAuth2 client ID |
| `--pingone-worker-client-secret` | - | OAuth2 client secret |
| `--pingone-region-code` | `NA` | Region: NA, EU, AP, CA, AU |
| `--out` / `-o` | stdout | Output directory path |
| `--output-format` | `hcl` | Output format: `hcl` or `tfjson` |
| `--module-name` | `ping-export` | Module name prefix |
| `--module-dir` | `ping-export-module` | Child module directory name |
| `--include-values` | `false` | Populate variable values from API |
| `--include-imports` | `false` | Generate import blocks in root module |
| `--skip-imports` | `false` | Skip generating import blocks |
| `--skip-dependencies` | `false` | Use hardcoded UUIDs instead of references |

### Output Formats

| Format | Flag Value | File Extension | Description |
|--------|-----------|----------------|-------------|
| HCL | `hcl` | `.tf` | Standard Terraform HCL syntax (default) |
| Terraform JSON | `tfjson` | `.tf.json` | Terraform JSON configuration syntax |

### Supported Resources

| Resource | Terraform Type |
|----------|---------------|
| PingOne Environment | `pingone_environment` |
| DaVinci Flow | `pingone_davinci_flow` |
| DaVinci Variable | `pingone_davinci_variable` |
| DaVinci Connector Instance | `pingone_davinci_connector_instance` |
| DaVinci Application | `pingone_davinci_application` |
| DaVinci Flow Policy | `pingone_davinci_application_flow_policy` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guides, architecture documentation, and how to add new resources.

## License

Apache License 2.0 - see [LICENSE](LICENSE).
