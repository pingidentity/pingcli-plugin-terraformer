# Developer Handbook

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Quick Start: Add a Resource](#quick-start-add-a-resource)
3. [Adding Custom Transforms](#adding-custom-transforms)
4. [Adding a New Platform or Service](#adding-a-new-platform-or-service)
5. [Adding Output Formats](#adding-output-formats)
6. [Testing Guide](#testing-guide)
7. [Debugging Guide](#debugging-guide)
8. [CLI Flags Reference](#cli-flags-reference)
9. [YAML Definition Reference](#yaml-definition-reference)
10. [FAQ](#faq)

---

## Getting Started

### Prerequisites

- Go 1.25+
- Make
- Access to a PingOne environment (for acceptance tests)

### Setup

```bash
git clone https://github.com/samir-gandhi/pingcli-terraformer.git
cd pingcli-terraformer

go mod download

make test
```

### Key Concepts

| Concept | Description |
|---------|-------------|
| Resource Definition | YAML file declaring how to extract and convert an API resource |
| Processor | Generic engine that walks SDK structs via reflection, guided by definitions |
| Formatter | Converts processed `ResourceData` to HCL or Terraform JSON |
| API Client | Platform-specific SDK wrapper implementing `clients.APIClient` |
| Dependency Graph | Tracks cross-resource relationships for reference resolution and ordering |

### Project Structure

```
cmd/
    export.go              # Cobra export command — full pipeline entry point
    tf.go                  # Root command setup

definitions/
    embed.go               # go:embed for YAML definitions
    pingone/
        base/              # PingOne base resource definitions (1 file)
        davinci/           # DaVinci resource definitions (7 files)

internal/
    clients/interface.go   # APIClient interface
    config/flags.go        # CLI flag resolution from flags + env vars
    core/
        orchestrator.go    # ExportOrchestrator — pipeline coordinator
        processor.go       # Processor — single-resource processing
        apidata.go         # Reflection-based struct field readers
        transforms.go      # Standard transform implementations
        custom_handlers.go # CustomHandlerRegistry, func types
        handler_queue.go   # Init-time handler queuing
    formatters/
        formatter.go       # OutputFormatter interface + factory
        hcl/               # HCL formatter (hclwrite-based)
        tfjson/            # Terraform JSON formatter
    filter/filter.go       # Resource filtering (include/exclude patterns)
    graph/graph.go         # DependencyGraph (cycle detection, topo sort)
    imports/generator.go   # Terraform import block generation
    module/generator.go    # Module structure generation
    platform/
        pingone/           # PingOne API client + resource handlers (all services)
    schema/
        types.go           # All YAML-mapped type definitions
        registry.go        # Thread-safe definition registry
        loader.go          # YAML file loading
        validator.go       # Definition validation
        keys.go            # CanonicalAttributeKey
    utils/                 # Sanitization utilities (use these, don't duplicate):
                           #   - SanitizeResourceName() — valid Terraform resource labels
                           #   - SanitizeMultiKeyResourceName() — composite key resources
                           #   - SanitizeVariableName() — valid Terraform variable names
    variables/extractor.go # Schema-driven variable extraction

tools/
    validate-definitions/  # CLI tool to validate YAML definitions
```

---

## Quick Start: Add a Resource

Adding a new PingOne resource requires **two files** and **zero edits** to existing code.

### Step 1: Analyze the SDK Type

Find the Go SDK struct for the resource. The fields you see here are what `source_path` values reference.

```go
// github.com/pingidentity/pingone-go-client/davinci
type Variable struct {
    Id          *string `json:"id,omitempty"`
    Name        *string `json:"name,omitempty"`
    Context     *string `json:"context,omitempty"`
    Value       interface{} `json:"value,omitempty"`
    Mutable     *bool   `json:"mutable,omitempty"`
    Environment *struct {
        Id *string `json:"id,omitempty"`
    } `json:"environment,omitempty"`
}
```

Document:
- All fields, their Go types, and whether they're pointers
- Which fields reference other resources (UUIDs pointing to flows, environments, etc.)
- Which fields should become Terraform module variables

**Critical**: `source_path` uses **Go struct field names** (`Id`, `Environment.Id`), not JSON tags (`id`, `environment.id`). Always verify against the actual SDK type.

### Step 2: Create the YAML Definition

Create `definitions/pingone/{category}/{short_name}.yaml` (e.g., `base/` for PingOne base resources, `davinci/` for DaVinci resources):

**Naming Strategy**: The YAML `label_fields` list combines attributes to form a unique resource label. This label is automatically sanitized via `utils.SanitizeResourceName()` (or `utils.SanitizeMultiKeyResourceName()` for composites) to produce a valid Terraform resource name. You do not specify sanitization in the YAML; the processing engine handles it.

```yaml
metadata:
  platform: pingone
  resource_type: pingone_davinci_variable
  api_type: Variable
  name: DaVinci Variable
  short_name: variable
  version: "1.0"

api:
  sdk_package: github.com/pingidentity/pingone-go-client/davinci
  sdk_type: Variable
  list_method: EnvironmentVariables.GetAll
  get_method: EnvironmentVariables.Get
  id_field: id
  name_field: name
  label_fields: [name, context]
  pagination_type: cursor

attributes:
  - name: ID
    terraform_name: id
    type: string
    source_path: Id
    computed: true

  - name: EnvironmentID
    terraform_name: environment_id
    type: string
    source_path: Environment.Id
    required: true
    references_type: pingone_environment
    reference_field: id

  - name: Name
    terraform_name: name
    type: string
    source_path: Name
    required: true

  - name: Context
    terraform_name: context
    type: string
    source_path: Context
    required: true

  - name: Value
    terraform_name: value
    type: type_discriminated_block
    source_path: Value
    variable_eligible: true
    type_discriminated_block:
      type_key_map:
        string: string
        bool: bool
        float64: float32
        int: float32
        map: json_object
        slice: json_object
      json_encode_keys: [json_object]
      skip_conditions:
        - source_field: DataType
          equals: secret

dependencies:
  depends_on:
    - resource_type: pingone_davinci_flow
      field_path: Flow.ID
      lookup_field: id
      reference_format: "pingone_davinci_flow.{resource_name}.id"
      optional: true
  import_id_format: "{env_id}/{resource_id}"

variables:
  eligible_attributes:
    - attribute_path: value
      variable_prefix: davinci_variable_
      is_secret: false
      description: "Value for DaVinci variable {name}"
```

### Step 3: Create the Resource Handler

Create `internal/platform/pingone/resource_{short_name}.go`:

```go
package pingone

import (
    "context"
    "fmt"
)

func init() {
    registerResource("pingone_davinci_variable", resourceHandler{
        list: listVariables,
        get:  getVariable,
    })
}

func listVariables(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
    resp, err := c.api.EnvironmentVariablesApi.GetAll(ctx, c.envID)
    if err != nil {
        return nil, fmt.Errorf("failed to list variables: %w", err)
    }

    results := make([]interface{}, 0, len(resp))
    for i := range resp {
        results = append(results, &resp[i])
    }
    return results, nil
}

func getVariable(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
    resp, err := c.api.EnvironmentVariablesApi.Get(ctx, c.envID, resourceID)
    if err != nil {
        return nil, fmt.Errorf("failed to get variable %s: %w", resourceID, err)
    }
    return resp, nil
}
```

The `init()` call to `registerResource` self-registers the handler into the dispatch table. No other files need editing.

**List-only vs list-then-get**: The `list` function must return fully-populated SDK structs. If the list API returns complete data, return it directly. If the list API returns only summaries (e.g., flows return only ID + Name + Description), the handler must call `get` for each item internally. The orchestrator always expects complete data from `ListResources`.

### Step 4: Validate

```bash
# Validate definition syntax and schema rules
go run ./tools/validate-definitions definitions/pingone

# Verify the resource handler compiles
go build ./internal/platform/pingone/...

# Run tests
go test ./internal/... -v -count=1
```

---

## Adding Custom Transforms

Most resources need only declarative YAML with standard transforms (`passthrough`, `base64_encode`, `json_encode`, `value_map`, etc.). Custom transforms are for cases requiring complex Go logic — deep JSON traversal, multi-field correlation, or property masking.

**Important**: Do not write inline sanitization logic in custom transforms or handlers. **Always use the canonicalized utilities**:
- `utils.SanitizeResourceName(name)` for Terraform resource labels
- `utils.SanitizeMultiKeyResourceName(keys...)` for composite keys
- `utils.SanitizeVariableName(name)` for Terraform variable identifiers

These are shared utilities maintained in `internal/utils/sanitize.go`. Duplicating sanitization logic across packages creates inconsistency and maintenance debt.

### When a Custom Transform Is Needed

The only custom transform currently in the codebase is `handleConnectorProperties` in [internal/platform/pingone/resource_connection.go](internal/platform/pingone/resource_connection.go). It handles connector instance property mapping, which requires:
- Iterating a dynamic property map
- Detecting masked secrets and replacing with variable references
- Handling nested complex properties
- Generating extracted variables for each property

Standard transforms and declarative YAML features cover everything else.

### Custom Transform Function Signature

```go
// internal/core/custom_handlers.go

type CustomTransformFunc func(
    value     interface{},                     // Raw extracted value
    apiData   interface{},                     // Full SDK struct
    attr      *schema.AttributeDefinition,     // Attribute definition
    def       *schema.ResourceDefinition,      // Resource definition
) (interface{}, error)
```

### Registration Pattern

Custom transforms are registered during `init()` in the resource handler file and queued for bulk loading at startup:

```go
// internal/platform/pingone/resource_connection.go

func init() {
    // Register API dispatch
    registerResource("pingone_davinci_connector_instance", resourceHandler{
        list: listConnectorInstances,
        get:  getConnectorInstance,
    })

    // Register custom transform — queued into CustomHandlerQueue
    registerTransform("handleConnectorProperties", handleConnectorProperties)
}

func handleConnectorProperties(
    value interface{},
    apiData interface{},
    _ *schema.AttributeDefinition,
    def *schema.ResourceDefinition,
) (interface{}, error) {
    // ... complex property mapping logic ...
    return processedValue, nil
}
```

The YAML definition references the transform by name:

```yaml
attributes:
  - name: Properties
    terraform_name: property
    type: map
    source_path: Properties
    transform: custom
    custom_transform: handleConnectorProperties
```

### How the Queue Works

1. Resource files call `registerTransform()` in `init()` — this adds to a package-level `CustomHandlerQueue`
2. At startup, `cmd/export.go` creates a `CustomHandlerRegistry` and calls `pingoneplatform.RegisterCustomHandlers(customReg)`
3. `RegisterCustomHandlers` calls `queue.LoadInto(registry)`, which bulk-registers all queued transforms
4. The `Processor` uses the registry to look up transforms by name at processing time

### Custom Handlers (Resource-Level)

In addition to attribute-level custom transforms, there is a `CustomHandlerFunc` for resource-level custom processing. This runs after all attribute processing and can modify the full attributes map:

```go
type CustomHandlerFunc func(
    data interface{},                      // Full SDK struct
    def  *schema.ResourceDefinition,       // Resource definition
) (map[string]interface{}, error)
```

Register via `registerHandler()` in `init()`. Referenced in YAML as:

```yaml
custom_handlers:
  transformer: handleSomeResource
```

No resource currently uses a resource-level custom handler. All current customization is done via attribute-level custom transforms.

---

## Adding a New Platform

All PingOne resources (base and DaVinci) share a single flat package at `internal/platform/pingone/`. To add a new PingOne resource, use the Quick Start steps above — no new package is needed.

This section covers adding support for an entirely new product platform (not PingOne). Follow the same flat-package pattern:

### Required Files

Create a platform package at `internal/platform/{platform}/`:

```
internal/platform/{platform}/
├── client.go              # Client struct, ListResources/GetResource dispatch
├── dispatch.go            # Handler dispatch table + custom handler queuing
└── resource_{short_name}.go  # One per resource, self-registers via init()
```

### client.go

```go
package newplatform

import (
    "context"
    "fmt"

    "github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
)

var _ clients.APIClient = (*Client)(nil)

type Client struct {
    api   *sdk.APIClient
    envID string
}

func New(apiClient *sdk.APIClient, envID string) *Client {
    return &Client{api: apiClient, envID: envID}
}

func (c *Client) Platform() string { return "newplatform" }

func (c *Client) ListResources(ctx context.Context, resourceType string, envID string) ([]interface{}, error) {
    h, ok := resourceHandlers[resourceType]
    if !ok {
        return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
    }
    return h.list(ctx, c, envID)
}

func (c *Client) GetResource(ctx context.Context, resourceType string, envID string, resourceID string) (interface{}, error) {
    h, ok := resourceHandlers[resourceType]
    if !ok {
        return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
    }
    return h.get(ctx, c, envID, resourceID)
}
```

### dispatch.go

```go
package newplatform

import "github.com/pingidentity/pingcli-plugin-terraformer/internal/core"

type resourceHandler struct {
    list func(ctx context.Context, c *Client, envID string) ([]interface{}, error)
    get  func(ctx context.Context, c *Client, envID, resourceID string) (interface{}, error)
}

var resourceHandlers = map[string]resourceHandler{}

var customHandlerQueue = core.NewCustomHandlerQueue()

func registerResource(resourceType string, h resourceHandler) {
    if _, exists := resourceHandlers[resourceType]; exists {
        panic(fmt.Sprintf("duplicate resource handler registration: %s", resourceType))
    }
    resourceHandlers[resourceType] = h
}

func registerTransform(name string, fn core.CustomTransformFunc) {
    customHandlerQueue.AddTransform(name, fn)
}

func RegisterCustomHandlers(reg *core.CustomHandlerRegistry) {
    customHandlerQueue.LoadInto(reg)
}
```

### Integration

After creating the platform package:
1. Create YAML definitions in `definitions/{platform}/{category}/`
2. Update `definitions/embed.go` to embed the new definition directory
3. Update `cmd/export.go` to instantiate the new client and call `RegisterCustomHandlers`

---

## Adding Output Formats

Output formats convert `ResourceData` to a target syntax.

### OutputFormatter Interface

```go
// internal/formatters/formatter.go

type OutputFormatter interface {
    Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
    FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
    FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error)
    FileExtension() string
}

type FormatOptions struct {
    SkipDependencies bool
    EnvironmentID    string
}
```

### Steps

1. Create `internal/formatters/{format}/formatter.go` — implement the format-specific logic
2. Add an adapter struct in `internal/formatters/formatter.go` that wraps your formatter and satisfies `OutputFormatter`
3. Register in the `NewFormatter()` switch statement
4. Add to `--output-format` flag validation in `cmd/export.go`

Current formats: `hcl` (`.tf` files), `tfjson` (`.tf.json` files).

### Key Implementation Notes

Your formatter must handle these value types in `ResourceData.Attributes`:

| Attribute Value Type | Description |
|---------------------|-------------|
| `string`, `bool`, `float64`, `int` | Primitives |
| `map[string]interface{}` | Nested objects and maps |
| `[]interface{}` | Lists and sets |
| `core.ResolvedReference` | Cross-resource reference — render as expression, not string |
| `core.RawHCLValue` | Pre-formatted expression — render without quoting |

---

## Testing Guide

### Test Organization

| Location | Scope | When to Run |
|----------|-------|-------------|
| `*_test.go` alongside source | Unit tests | Always |
| `tests/integration/` | Multi-component pipeline tests | Before committing |
| `tests/acceptance/` | Live PingOne environment | Before release |

### Running Tests

```bash
# All unit tests
make test

# Or directly
go test ./internal/... -v -count=1

# Acceptance tests (requires live environment)
make testacc

# With coverage
make testcoverage
```

### Writing Unit Tests

Use table-driven tests:

```go
func TestSanitizeName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "spaces replaced",
            input:    "my resource",
            expected: "my_resource",
        },
        {
            name:     "special chars removed",
            input:    "my-resource!@#",
            expected: "my_resource___",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := SanitizeName(tt.input)
            if result != tt.expected {
                t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

### Validating Definitions

```bash
# Validate all YAML definitions
go run ./tools/validate-definitions definitions/

# Validate a specific category
go run ./tools/validate-definitions definitions/pingone
```

---

## Debugging Guide

### Common Issues

#### "unsupported resource type"

The resource type is not in the dispatch table.

**Check**:
1. The `init()` function in `resource_{short_name}.go` calls `registerResource` with the correct resource type string
2. The resource type in the YAML `metadata.resource_type` matches the string passed to `registerResource`
3. The package is imported (Go only runs `init()` for imported packages)

#### Attribute Value Is nil When Expected

The `source_path` does not match the Go struct field name.

**Check**:
- `source_path` uses Go field names (`Id`, not `ID`; `ApiKey`, not `APIKey`)
- Nested paths use dots: `Environment.Id`
- The field exists on the SDK struct at the pinned version in `go.mod`

#### Reference Not Resolved (Raw UUID in Output)

The orchestrator could not find the target resource in the dependency graph.

**Check**:
1. The `references_type` in the attribute definition matches a `metadata.resource_type` of another definition
2. The referenced resource type is in the `depends_on` list in the `dependencies` section
3. The referenced resource was actually fetched from the API (check API client logs)

#### Import Block Has Wrong Format

The `import_id_format` template uses placeholders that don't match available data.

**Available placeholders**: `{env_id}`, `{resource_id}`, `{<terraform_attr_name>}`

### Profiling

```bash
go test -cpuprofile=cpu.prof ./internal/core
go tool pprof cpu.prof

go test -memprofile=mem.prof ./internal/core
go tool pprof mem.prof
```

---

## CLI Flags Reference

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pingone-worker-environment-id` | `string` | | Environment ID containing worker app |
| `--pingone-export-environment-id` | `string` | | Target environment to export |
| `--pingone-region-code` | `string` | | Region: `NA`, `EU`, `AP`, `CA`, `AU` |
| `--pingone-worker-client-id` | `string` | | OAuth worker app client ID |
| `--pingone-worker-client-secret` | `string` | | OAuth worker app client secret |
| `--out`, `-o` | `string` | | Output directory path |
| `--output-format` | `string` | `"hcl"` | Output format: `hcl` or `tfjson` |
| `--skip-dependencies` | `bool` | `false` | Skip dependency resolution |
| `--skip-imports` | `bool` | `false` | Skip generating import blocks |
| `--include-imports` | `bool` | `false` | Generate import blocks in root module |
| `--include-resources` | `string` | | Include resources matching glob/regex pattern(s). Repeatable. Patterns match `resource_type.terraform_label` (case-insensitive). Use `regex:` prefix for regex patterns. Multiple patterns combine via OR. |
| `--exclude-resources` | `string` | | Exclude resources matching glob/regex pattern(s). Repeatable. Same matching rules as `include-resources`. Takes precedence over includes. |
| `--list-resources` | `bool` | `false` | List all resource addresses (`resource_type.terraform_label`) and exit. Useful for discovering exact addresses to use with include/exclude patterns. |
| `--include-values` | `bool` | `false` | Populate variable values from export |
| `--module-dir` | `string` | `"ping-export-module"` | Child module directory name |
| `--module-name` | `string` | `"ping-export"` | Module name / prefix |

Flags can also be sourced from environment variables. See `internal/config/flags.go` for the resolution order.

### Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build binary |
| `make install` | Build and install to `$GOBIN` |
| `make test` | Run unit tests |
| `make testacc` | Run acceptance tests |
| `make testcoverage` | Tests with coverage report |
| `make vet` | Run `go vet` |
| `make lint` | Run `golangci-lint` |
| `make fmt` | Format Go code |
| `make clean` | Remove build artifacts |
| `make devcheck` | Full pre-commit check (build + vet + fmt + lint + test + testacc) |

---

## YAML Definition Reference

### Full Schema

```yaml
metadata:
  platform: <platform>                # e.g., pingone
  resource_type: <terraform_type>     # e.g., pingone_davinci_variable
  api_type: <sdk_struct_name>         # e.g., Variable
  name: <human_readable_name>
  short_name: <short>                 # Used in filenames
  version: "1.0"

api:
  sdk_package: <go_import_path>
  sdk_type: <struct_name>
  list_method: <method>
  get_method: <method>                # Optional
  id_field: <field>
  name_field: <field>
  labels: [<terraform_attr>, ...]  # Combined for HCL label (sanitized automatically via utils.SanitizeResourceName or utils.SanitizeMultiKeyResourceName)
  pagination_type: <cursor|...>
  additional_id_fields: [<field>, ...]   # Optional
  allow_duplicate_labels: true|false     # Optional; default false

attributes:
  - name: <GoFieldName>
    terraform_name: <snake_case_name>    # Required — no implicit default
    type: <type>                         # string|number|bool|object|list|map|set|type_discriminated_block
    source_path: <Go.Struct.Path>        # Dot-notation, Go field names
    required: true|false
    computed: true|false
    sensitive: true|false
    variable_eligible: true|false
    variable_default: <value>            # Optional default for extracted variable
    references_type: <other_resource>    # Target resource type for reference resolution
    reference_field: <field>             # Field on target resource
    transform: <transform_name>         # Standard transform
    custom_transform: <handler_name>     # Custom transform (requires transform: custom)
    override_value: <constant>           # Force this value regardless of API data
    value_map:                           # Declarative value normalization
      <api_value>: <terraform_value>
    value_map_default: <fallback>
    map_key_path: <sub_field>            # Key field when converting slice → map
    lifecycle:
      ignore_changes: [<attr>, ...]
    masked_secret:
      sentinel: <masked_value>           # e.g., "************"
      variable_prefix: <prefix>
    nested_attributes: [...]             # Recursive for object/list/map types
    type_discriminated_block:
      type_key_map:
        <go_runtime_type>: <terraform_block_key>
      json_encode_keys: [<keys>]
      skip_conditions:
        - source_field: <ApiFieldName>
          equals: <value>

conditional_defaults:                    # Post-processing attribute overrides
  - target_attribute: <terraform_name>
    set_value: <value>
    when_all:
      - attribute_empty: <terraform_name>
      - attribute_equals:
          name: <terraform_name>
          value: <value>

dependencies:
  depends_on:
    - resource_type: <type>
      field_path: <Go.Struct.Path>
      lookup_field: <field>
      reference_format: "<type>.{resource_name}.<field>"
      optional: true|false
  import_id_format: "{env_id}/{resource_id}"

variables:
  eligible_attributes:
    - attribute_path: <terraform_attr>
      variable_prefix: <prefix>
      is_secret: true|false
      description: "Template with {name} placeholder"

custom_handlers:
  transformer: <handler_name>            # Resource-level custom handler
```

### Attribute Types

| Type | Go Value | HCL Output |
|------|----------|------------|
| `string` | `string` | `"value"` |
| `number` | `int`, `float64` | `42` |
| `bool` | `bool` | `true` |
| `object` | `map[string]interface{}` | `{ key = "value" }` |
| `list` | `[]interface{}` | `["a", "b"]` |
| `map` | `map[string]interface{}` | `{ a = 1, b = 2 }` |
| `set` | `[]interface{}` | `["a", "b"]` (deduplicated) |
| `type_discriminated_block` | varies | `{ string = "value" }` |

### Standard Transforms and Sanitization

| Transform | Description |
|-----------|----------|
| `passthrough` | Returns value unchanged |
| `base64_encode` | Base64 encodes a string |
| `base64_decode` | Base64 decodes a string |
| `json_encode` | Marshals value to JSON string |
| `json_decode` | Parses JSON string to Go value |
| `jsonencode_raw` | Emits `jsonencode(...)` HCL expression |
| `to_string` | Converts any value to string representation |
| `value_map` | Looks up value in `value_map`, with `value_map_default` fallback |
| `custom` | Dispatches to named function in `CustomHandlerRegistry` via `custom_transform` |

**Sanitization is built-in**: Resource labels from `label_fields` are automatically sanitized via `utils.SanitizeResourceName()` or `utils.SanitizeMultiKeyResourceName()`. Variable names are sanitized via `utils.SanitizeVariableName()`. You do not need to apply sanitization transforms manually — it happens automatically during processing.

---

## FAQ

### When do I need a custom handler?

Almost never. The YAML schema supports declarative patterns for:

- **Type-discriminated blocks**: `type: type_discriminated_block` with `type_key_map` — the HCL block key depends on the runtime Go type of the API value
- **Conditional defaults**: `conditional_defaults` — override attribute values based on other processed attributes
- **Skip conditions**: `skip_conditions` within `type_discriminated_block` — suppress values based on API fields
- **Value mapping**: `value_map` — declarative API-to-Terraform value translation
- **Override values**: `override_value` — force a constant regardless of API response
- **Masked secrets**: `masked_secret` — detect sentinel values and replace with variable references

Custom transforms are reserved for transformations that cannot be expressed declaratively. Currently only `handleConnectorProperties` (connector instance property mapping) uses one.

**Rule of thumb**: If a second resource would need the same Go logic, it should become a declarative YAML feature instead.

### Why does source_path use `Id` instead of `ID`?

`source_path` must match the **Go struct field name** exactly. The processor uses `reflect` to traverse struct fields by exported Go name, not JSON tags. Common differences:

| Incorrect (JSON/convention) | Correct (Go struct field) |
|---|---|
| `ID` | `Id` |
| `Environment.ID` | `Environment.Id` |
| `APIKey` | `ApiKey` |

Verify against the actual SDK type at `github.com/pingidentity/pingone-go-client`.

### How does the processor handle SDK-specific types?

The processor coerces SDK types to Go primitives before processing:
- `uuid.UUID` → `string` via `.String()`
- SDK enum types implementing `fmt.Stringer` → `string` via `.String()`
- Pointer types → dereferenced to underlying value
- SDK union/choice wrapper structs → unwrapped to the single non-nil field

### How do I add an attribute to an existing resource?

1. Add the attribute entry to the resource's YAML definition
2. Verify the `source_path` against the SDK struct
3. Run `go run ./tools/validate-definitions definitions/pingone`
4. Run `go test ./internal/... -v -count=1`

### How do I handle sensitive values?

Use `sensitive: true` and optionally `masked_secret` for API-masked values:

```yaml
- name: ClientSecret
  terraform_name: client_secret
  type: string
  source_path: ClientSecret
  sensitive: true
  variable_eligible: true
  masked_secret:
    sentinel: "************"
    variable_prefix: secret_
```

### When should I use the sanitization utilities?

**Always**. Never write inline sanitization logic.

The `internal/utils/sanitize.go` module provides three canonicalized functions:

- **`utils.SanitizeResourceName(name)`** — Produces valid Terraform resource labels. Hex-encodes non-alphanumeric and non-hyphen characters, prepends `pingcli__`. Use this when generating Terraform resource names.
- **`utils.SanitizeMultiKeyResourceName(keys...)`** — Combines multiple key components (e.g., variable name + context). Same hex-encoding and `pingcli__` prefix, but joins keys with underscores. Use this for resources with composite/multi-field labels.
- **`utils.SanitizeVariableName(name)`** — Produces valid Terraform variable identifiers. Replaces any character not in `[a-zA-Z0-9_]` with underscore. Use this for Terraform variable names.

**In YAML definitions**: You do not need to invoke sanitization. The processor automatically applies `SanitizeResourceName()` or `SanitizeMultiKeyResourceName()` to labels based on the `label_fields` list. Similarly, `SanitizeVariableName()` is applied automatically when extracting variable names.

**In custom Go code** (handlers, transforms): Always import and call `utils.SanitizeResourceName()`, `utils.SanitizeMultiKeyResourceName()`, or `utils.SanitizeVariableName()` rather than writing your own logic. This ensures consistency across the codebase and reduces maintenance debt.

Example:
```go
import "github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"

resourceLabel := utils.SanitizeResourceName(data.Name)
variableName := utils.SanitizeVariableName(attr.TerraformName)
compositeLabel := utils.SanitizeMultiKeyResourceName(data.Name, data.Context)
```

### How do I test against a live environment?

```bash
export PINGCLI_PINGONE_WORKER_ENVIRONMENT_ID="..."
export PINGCLI_PINGONE_WORKER_CLIENT_ID="..."
export PINGCLI_PINGONE_WORKER_CLIENT_SECRET="..."
export PINGCLI_PINGONE_REGION_CODE="NA"

make testacc
```
