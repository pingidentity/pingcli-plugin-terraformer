# Ping CLI Terraformer Plugin

Export PingOne resources to Terraform configuration with automatic dependency resolution and import block generation.

## Features

- **Complete DaVinci Export**: Export PingOne environments, DaVinci flows, variables, connector instances, applications, and flow policies
- **Multiple Output Formats**: Supports Terraform HCL (`.tf`) or Terraform JSON (`.tf.json`) output
- **Automatic Dependency Resolution**: Generates Terraform references between resources
- **Import Block Generation**: Terraform import blocks to manage existing resources (Terraform 1.5+)
- **Module Structure**: Generates reusable Terraform modules with proper variable scaffolding
- **Output Generation**: Populate `outputs.tf` so parent modules can reference child module resources
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
- Terraform 1.5+ (for import blocks)
- PingOne worker application with `DaVinci Admin Read Only` role access

> NOTE: Generating DaVinci Variable resource references on DaVinci Flow resources use an API POST call that requires the PingOne worker application to have the `DaVinci Admin` role. DaVinci Flows use references dependent variables to properly order the deletion of resources. The tool will produce a warning if this API call cannot be completed.

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PINGCLI_PINGONE_ENVIRONMENT_ID` | Worker environment ID (authentication) |
| `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID` | Target environment to export (optional, defaults to worker env) |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID` | OAuth2 client ID |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET` | OAuth2 client secret |
| `PINGCLI_PINGONE_REGION_CODE` | Region code: NA, EU, AP, CA, AU, SG |

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
| `--pingone-region-code` | `NA` | Region: NA, EU, AP, CA, AU, SG |
| `--out` / `-o` | stdout | Output directory path |
| `--output-format` | `hcl` | Output format: `hcl` or `tfjson` |
| `--module-name` | `ping-export` | Module name prefix |
| `--module-dir` | `ping-export-module` | Child module directory name |
| `--include-values` | `false` | Populate variable values from API |
| `--include-imports` | `false` | Generate import blocks in root module |
| `--skip-imports` | `false` | Skip generating import blocks |
| `--skip-dependencies` | `false` | Use hardcoded UUIDs instead of references |
| `--include-resources` | - | Include resources matching glob/regex pattern (repeatable) |
| `--exclude-resources` | - | Exclude resources matching glob/regex pattern (repeatable) |
| `--include-upstream` | `false` | Include upstream dependencies of filtered resources |
| `--list-resources` | `false` | List resource addresses and exit |
| `--output-attribute` | - | Add an output block for `resource_type.label.attr` (repeatable; glob `*` supported in label position) |
| `--output-attribute-file` | - | File with one `resource_type.label.attr` path per line (same format as `--output-attribute`) |

### List Outputs Command

Enumerates all possible output attribute paths for exported resources without writing any files. Output is newline-delimited on stdout, suitable for piping or redirecting to a file for use with `--output-attribute-file`.

| Flag | Default | Description |
|------|---------|-------------|
| `--pingone-worker-environment-id` | - | Worker environment ID |
| `--pingone-export-environment-id` | Worker env | Target environment ID |
| `--pingone-worker-client-id` | - | OAuth2 client ID |
| `--pingone-worker-client-secret` | - | OAuth2 client secret |
| `--pingone-region-code` | `NA` | Region: NA, EU, AP, CA, AU, SG |
| `--depth` | `1` | Attribute enumeration depth (1 = top-level only; 2 = one level of nesting) |
| `--include-resources` | - | Include resources matching glob/regex pattern (repeatable) |
| `--exclude-resources` | - | Exclude resources matching glob/regex pattern (repeatable) |
| `--include-upstream` | `false` | Include upstream dependencies of filtered resources |

### Output Formats

| Format | Flag Value | File Extension | Description |
|--------|-----------|----------------|-------------|
| HCL | `hcl` | `.tf` | Standard Terraform HCL syntax (default) |
| Terraform JSON | `tfjson` | `.tf.json` | Terraform JSON configuration syntax |

### Supported Resources

| Resource | Terraform Type |
|----------|---------------|
| DaVinci Flow | `pingone_davinci_flow` |
| DaVinci Variable | `pingone_davinci_variable` |
| DaVinci Connector Instance | `pingone_davinci_connector_instance` |
| DaVinci Application | `pingone_davinci_application` |
| DaVinci Flow Policy | `pingone_davinci_application_flow_policy` |

## Resource Filtering

Export only specific resources using glob or regex patterns:

### Flags

| Flag | Description |
|------|-------------|
| `--include-resources <pattern>` | Include resources matching pattern(s). Repeatable. Patterns match `resource_type.terraform_label` (case-insensitive). Use `regex:` prefix for regex. |
| `--exclude-resources <pattern>` | Exclude resources matching pattern(s). Repeatable. Same matching rules as include. |
| `--include-upstream` | Automatically include upstream dependencies of matched resources. Transitive — dependencies of dependencies are also included. |
| `--list-resources` | List available resource addresses (`resource_type.terraform_label`) and exit. Useful for discovering exact addresses to filter. |

### Pattern Syntax

- **Glob** (default): `*` matches any characters, `?` matches single character
  - `pingone_davinci_flow.*` — all flows
  - `pingone_davinci_variable.dev*` — variables starting with "dev"
  
- **Regex**: Prefix pattern with `regex:`, uses Go regexp syntax
  - `regex:pingone_davinci_(flow|variable)\..*` — flows or variables
  - `regex:^pingone_davinci_flow\.(login|logout)` — specific flow names

- Multiple patterns combine via OR (union)
- No filters = export all resources (backwards compatible)

### Upstream Dependencies

Use `--include-upstream` with `--include-resources` to automatically pull in resources that matched resources depend on. This follows the dependency graph transitively:

- A **flow** depends on its **connector instances**
- A **flow_deploy** depends on its **flow**
- A **flow_policy** depends on its **application** and **flow**
- A **variable** depends on its referenced **flow**

For example, exporting a single flow with `--include-upstream` also exports all connector instances used by that flow. Exporting a flow_deploy with `--include-upstream` exports the deploy, its flow, and all connector instances used by the flow.

Explicit `--exclude-resources` patterns always take priority over upstream expansion. If an upstream dependency is explicitly excluded, it will not be included and a Terraform variable will be generated as a placeholder.

### Examples

List all available resource addresses:
```bash
pingcli-terraformer export --list-resources
```

Export only DaVinci flows and variables:
```bash
pingcli-terraformer export \
  --include-resources "pingone_davinci_flow.*" \
  --include-resources "pingone_davinci_variable.*" \
  --out ./output
```

Export everything except flow policies:
```bash
pingcli-terraformer export \
  --exclude-resources "pingone_davinci_application_flow_policy.*" \
  --out ./output
```

Export flows with specific name patterns using regex:
```bash
pingcli-terraformer export \
  --include-resources "regex:pingone_davinci_flow\.(login|mfa|consent)" \
  --out ./output
```

Combine include and exclude (exclude takes precedence for overlaps):
```bash
pingcli-terraformer export \
  --include-resources "pingone_davinci*" \
  --exclude-resources "pingone_davinci_application_flow_policy.*" \
  --out ./output
```

Export a specific flow and all its upstream dependencies:
```bash
pingcli-terraformer export \
  --include-resources "pingone_davinci_flow.pingcli__My-0020-Flow" \
  --include-upstream \
  --out ./output
```

Export flow policies with full dependency chain (policies → apps → flows → connectors):
```bash
pingcli-terraformer export \
  --include-resources "pingone_davinci_application_flow_policy.*" \
  --include-upstream \
  --out ./output
```

Export a flow with upstream, but exclude specific connectors:
```bash
pingcli-terraformer export \
  --include-resources "pingone_davinci_flow.*" \
  --include-upstream \
  --exclude-resources "pingone_davinci_connector_instance.pingcli__Http" \
  --out ./output
```

## Generating Module Outputs

When the exported module is used as a child module in a root Terraform configuration, the root module may need to reference resource attributes (e.g. a flow ID) via output blocks. Use `list-outputs` to discover available paths and `--output-attribute` / `--output-attribute-file` to populate `outputs.tf`.

### Discover available output paths

```bash
pingcli-terraformer list-outputs \
  --include-resources "pingone_davinci_flow.*" \
  --pingone-worker-environment-id <uuid> \
  --pingone-worker-client-id <client-id> \
  --pingone-worker-client-secret <secret>
```

Use `--depth 2` to include nested attributes (e.g. `api_key.value`):

```bash
pingcli-terraformer list-outputs --depth 2 ...
```

### Export with specific output attributes

Using `--output-attribute` directly, with glob support in the label position:

```bash
pingcli-terraformer export \
  --output-attribute "pingone_davinci_flow.*.id" \
  --output-attribute "pingone_davinci_flow.*.name" \
  --out ./output ...
```

### Pipe list-outputs into export

```bash
# Capture all flow output paths, filter to just IDs
pingcli-terraformer list-outputs \
  --include-resources "pingone_davinci_flow.*" ... \
  | grep '\.id$' > flow-outputs.txt

# Edit flow-outputs.txt as needed, then export
pingcli-terraformer export \
  --output-attribute-file flow-outputs.txt \
  --out ./output ...
```

The generated `outputs.tf` in the child module will contain one `output` block per path:

```hcl
output "pingone_davinci_flow__pingcli__My-0020-Flow__id" {
  description = "The id of pingone_davinci_flow pingcli__My-0020-Flow"
  value       = pingone_davinci_flow.pingcli__My-0020-Flow.id
}
```

The root module can then reference it as `module.<module_name>.pingone_davinci_flow__pingcli__My-0020-Flow__id`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guides, architecture documentation, and how to add new resources.

## License

Apache License 2.0 - see [LICENSE](LICENSE).
