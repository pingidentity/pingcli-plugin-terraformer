# Architecture

**Repository**: `github.com/pingidentity/pingcli-plugin-terraformer`

---

## Overview

This project is a **schema-driven, multi-format Terraform configuration export engine** for PingOne services. It reads resource data from PingOne APIs, processes it through a generic engine driven by YAML definitions, and outputs Terraform HCL or JSON — including dependency resolution, import blocks, variable extraction, and module scaffolding.

### Core Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                     RESOURCE DEFINITIONS                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ variable.yaml│  │ flow.yaml    │  │ connector.yaml│  ...        │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                     │
│  definitions/                                                       │
│  └── pingone/                                                       │
│      └── davinci/                                                   │
│          ├── application.yaml                                       │
│          ├── connector_instance.yaml                                │
│          ├── environment.yaml                                       │
│          ├── flow.yaml                                              │
│          ├── flow_deploy.yaml                                       │
│          ├── flow_enable.yaml                                       │
│          ├── flow_policy.yaml                                       │
│          └── variable.yaml                                          │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    CORE PROCESSING ENGINE                           │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                   ExportOrchestrator                           │  │
│  │    - Discovers resource types from registry                   │  │
│  │    - Orders by dependency (topological sort)                  │  │
│  │    - Fetches data via API client                              │  │
│  │    - Processes via Processor                                  │  │
│  │    - Resolves cross-resource references                       │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐       │
│  │ Processor      │  │ DependencyGraph│  │ CustomHandlers │       │
│  └────────────────┘  └────────────────┘  └────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    OUTPUT LAYER                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ HCL Formatter│  │ JSON Format  │  │ Module Gen   │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                     │
│  ┌──────────────┐  ┌──────────────┐                                │
│  │ Import Gen   │  │ Variable Ext │                                │
│  └──────────────┘  └──────────────┘                                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Design Principles

### 1. Schema-Driven Development

All resource behavior is derived from declarative YAML schemas. No resource-specific Go code is needed for standard resources. The YAML schema handles:

- **Type-discriminated blocks**: Attributes whose HCL block key depends on the runtime type of the API value
- **Conditional attribute overrides**: Post-processing rules that set attribute defaults based on the state of other processed attributes
- **Skip conditions**: Suppression of attributes based on API response field values
- **Value mapping**: Declarative API-to-Terraform value normalization
- **Override values**: Fixed constant values regardless of API response

Custom Go handlers are reserved for genuinely complex transformations — specifically, cases requiring deep JSON tree traversal or multi-field correlation. Currently only connector instance property mapping uses a custom handler.

### 2. Separation of Concerns

```
WHAT to convert    → Resource Definitions (YAML)
HOW to convert     → Core Processing Engine (Go)
OUTPUT format      → Output Formatters (Go)
API interaction    → API Clients (Go)
```

### 3. SDK-Aligned Data Model

The processor operates on **Go SDK structs** — the same types used by the Terraform provider's read functions.

- `source_path` values in YAML definitions use **Go struct field names**, not JSON tags. Example: `Id` (not `ID`), `Environment.Id` (not `Environment.ID`), `ApiKey` (not `APIKey`).
- The processor uses Go reflection (`reflect` package) to traverse struct fields by exact name.
- SDK-specific types are coerced to Go primitives during extraction:
  - `uuid.UUID` → `string` via `.String()`
  - SDK enum types implementing `fmt.Stringer` → `string` via `.String()`
  - Pointer types → dereferenced to their underlying value
  - SDK union/choice wrapper structs → unwrapped to the single non-nil field

The SDK version is pinned once in `go.mod`. When the SDK version is bumped, all definitions must be audited for struct field renames or type changes.

---

## Package Layout

```
cmd/
    export.go              # Cobra command: export pipeline entry point
    tf.go                  # Root command setup
    tf_test.go

definitions/
    embed.go               # go:embed for YAML definitions
    pingone/davinci/       # YAML resource definitions

internal/
    api/                   # Legacy API client (being replaced)
    clients/
        interface.go       # APIClient interface
    compare/               # Content comparison utilities
    config/
        flags.go           # CLI flag resolution
    core/
        apidata.go         # Reflection-based struct field readers
        custom_handlers.go # CustomHandlerRegistry and CustomTransformFunc
        handler_queue.go   # Init-time handler queuing
        orchestrator.go    # ExportOrchestrator (full pipeline)
        processor.go       # Processor (single-resource processing)
        transforms.go      # Standard transform implementations
    formatters/
        formatter.go       # OutputFormatter interface + factory
        hcl/               # HCL formatter (hclwrite-based)
        tfjson/            # Terraform JSON formatter
    graph/
        graph.go           # DependencyGraph (cycle detection, topo sort)
    imports/
        generator.go       # Terraform import block generation
    module/
        generator.go       # Module structure generation
        types.go           # Module types (Variable, Output, etc.)
    platform/
        pingone/davinci/   # DaVinci API client + resource handlers
    schema/
        keys.go            # CanonicalAttributeKey
        loader.go          # YAML file loading
        registry.go        # Thread-safe definition registry
        types.go           # All schema type definitions
        validator.go       # Definition validation
    utils/
        hclsort.go         # HCL block sorting
        sanitize.go        # Canonicalized name sanitization functions:
                           #   - SanitizeResourceName() — hex-encode specials, add pingcli__ prefix
                           #   - SanitizeMultiKeyResourceName() — same for composite keys
                           #   - SanitizeVariableName() — replace non-[a-zA-Z0-9_] with _
    variables/
        extractor.go       # Schema-driven variable extraction
```

---

## Layer 1: Schema System

### Resource Definition Structure

Resource definitions are YAML files in `definitions/{platform}/{service}/`. They are embedded at compile time via `go:embed` and loaded into a `Registry`.

```go
// internal/schema/types.go

type ResourceDefinition struct {
    Metadata            ResourceMetadata         `yaml:"metadata"`
    API                 APIDefinition            `yaml:"api"`
    Attributes          []AttributeDefinition    `yaml:"attributes"`
    Dependencies        DependencyDefinition     `yaml:"dependencies"`
    Variables           VariableDefinition       `yaml:"variables"`
    CustomHandlers      *CustomHandlerDefinition `yaml:"custom_handlers,omitempty"`
    ConditionalDefaults []ConditionalDefault     `yaml:"conditional_defaults,omitempty"`
}
```

### ResourceMetadata

```go
type ResourceMetadata struct {
    Platform     string `yaml:"platform"`       // "pingone"
    Service      string `yaml:"service"`        // "davinci"
    ResourceType string `yaml:"resource_type"`  // "pingone_davinci_variable"
    APIType      string `yaml:"api_type"`       // "Variable"
    Name         string `yaml:"name"`           // "DaVinci Variable"
    ShortName    string `yaml:"short_name"`     // "variable"
    Version      string `yaml:"version"`        // "1.0"
}
```

### APIDefinition

```go
type APIDefinition struct {
    SDKPackage           string   `yaml:"sdk_package"`
    SDKType              string   `yaml:"sdk_type"`
    ListMethod           string   `yaml:"list_method"`
    GetMethod            string   `yaml:"get_method,omitempty"`
    IDField              string   `yaml:"id_field"`
    NameField            string   `yaml:"name_field"`
    LabelFields          []string `yaml:"label_fields,omitempty"`
    PaginationType       string   `yaml:"pagination_type"`
    AdditionalIDFields   []string `yaml:"additional_id_fields,omitempty"`
    AllowDuplicateLabels bool     `yaml:"allow_duplicate_labels,omitempty"`
}
```

- `LabelFields`: Ordered list of terraform attribute names combined to produce the HCL resource label. The combined label is passed through `utils.SanitizeResourceName()` to generate a valid Terraform resource name. Falls back to the `name_field` value when empty.
- `AllowDuplicateLabels`: Skips unique label validation for resource types where the upstream system permits duplicate names (e.g., flow policies).

### AttributeDefinition

```go
type AttributeDefinition struct {
    Name                   string                        `yaml:"name"`
    TerraformName          string                        `yaml:"terraform_name"`
    Type                   string                        `yaml:"type"`
    SourcePath             string                        `yaml:"source_path,omitempty"`
    Required               bool                          `yaml:"required"`
    Computed               bool                          `yaml:"computed"`
    Sensitive              bool                          `yaml:"sensitive"`
    VariableEligible       bool                          `yaml:"variable_eligible"`
    VariableDefault        interface{}                   `yaml:"variable_default,omitempty"`
    ReferencesType         string                        `yaml:"references_type,omitempty"`
    ReferenceField         string                        `yaml:"reference_field,omitempty"`
    Transform              string                        `yaml:"transform,omitempty"`
    CustomTransform        string                        `yaml:"custom_transform,omitempty"`
    NestedAttributes       []AttributeDefinition         `yaml:"nested_attributes,omitempty"`
    Lifecycle              *LifecycleConfig              `yaml:"lifecycle,omitempty"`
    MapKeyPath             string                        `yaml:"map_key_path,omitempty"`
    ValueMap               map[string]string             `yaml:"value_map,omitempty"`
    ValueMapDefault        string                        `yaml:"value_map_default,omitempty"`
    MaskedSecret           *MaskedSecretConfig           `yaml:"masked_secret,omitempty"`
    TypeDiscriminatedBlock *TypeDiscriminatedBlockConfig `yaml:"type_discriminated_block,omitempty"`
    OverrideValue          interface{}                   `yaml:"override_value,omitempty"`
}
```

**Attribute types**: `string`, `number`, `bool`, `object`, `list`, `map`, `set`, `type_discriminated_block`

**Available transforms**: `passthrough`, `base64_encode`, `base64_decode`, `json_encode`, `json_decode`, `jsonencode_raw`, `to_string`, `value_map`, `custom`

**Key fields**:
- `SourcePath`: Dot-notation path to the Go struct field (uses Go field names, not JSON tags)
- `MapKeyPath`: Sub-field path used to key slice elements when converting an API slice to a Terraform map
- `ValueMap` / `ValueMapDefault`: Declarative API-to-Terraform value normalization
- `OverrideValue`: Forces a constant value regardless of API response
- `MaskedSecret`: Configures sentinel detection and variable substitution for masked secret API values

### Declarative Attribute Patterns

#### Type-Discriminated Blocks

Some Terraform provider attributes expect a block with exactly one type-specific key. The YAML definition declares the mapping:

```yaml
- name: Value
  type: type_discriminated_block
  source_path: Value
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
```

The processor reads the runtime type of the extracted value, looks up the block key in `type_key_map`, and emits the appropriate single-key map. Values for keys listed in `json_encode_keys` are `json.Marshal`'d and emitted as `RawHCLValue`.

#### Conditional Defaults

Post-processing attribute overrides based on the state of other processed attributes:

```yaml
conditional_defaults:
  - target_attribute: mutable
    set_value: true
    when_all:
      - attribute_empty: value
      - attribute_equals:
          name: mutable
          value: false
```

Evaluated after all attributes are processed and before formatting.

### Schema Registry

```go
// internal/schema/registry.go

type Registry struct { ... }

func NewRegistry() *Registry
func (r *Registry) Register(def *ResourceDefinition) error
func (r *Registry) Get(resourceType string) (*ResourceDefinition, error)
func (r *Registry) ListAll() []*ResourceDefinition
func (r *Registry) ListByPlatform(platform string) []*ResourceDefinition
func (r *Registry) ListByService(platform, service string) []*ResourceDefinition
func (r *Registry) LoadFromFS(fsys fs.FS, dir string) error
func (r *Registry) LoadFromDirectory(dir string) error
```

Thread-safe. Definitions are loaded from embedded YAML files at startup via `LoadFromFS`.

---

## Layer 2: Core Processing Engine

### Processor

The `Processor` converts API responses (Go SDK structs) into `ResourceData` — the format-agnostic intermediate representation.

```go
// internal/core/processor.go

type Processor struct {
    registry       *schema.Registry
    customHandlers *CustomHandlerRegistry
}

func NewProcessor(registry *schema.Registry, opts ...ProcessorOption) *Processor

func (p *Processor) ProcessResource(resourceType string, apiData interface{}) (*ResourceData, error)
func (p *Processor) ProcessResourceList(resourceType string, apiDataList interface{}) ([]*ResourceData, error)
```

**Processing order per resource**:
1. Extract raw values from API data per `source_path` (reflection-based)
2. Handle `type_discriminated_block` attributes
3. Handle `override_value` attributes
4. Extract nested attributes for `object`/`list`/`map` types
5. Apply transforms (standard or custom)
6. Populate dependencies from schema
7. Run resource-level custom handlers (if any)
8. Evaluate `conditional_defaults`

### ResourceData (Intermediate Representation)

```go
type ResourceData struct {
    ResourceType       string
    ID                 string
    Name               string
    Label              string                    // Sanitized, unique Terraform label
    Attributes         map[string]interface{}    // Processed attribute values
    Dependencies       []Dependency
    ExtractedVariables []ExtractedVariable
}
```

### ResolvedReference

Cross-resource references are resolved by the orchestrator and stored as `ResolvedReference` values in the attribute map. Formatters detect these via type assertion:

```go
type ResolvedReference struct {
    ResourceType  string  // e.g. "pingone_davinci_flow"
    ResourceName  string  // e.g. "pingcli__my_flow"
    Field         string  // e.g. "id"
    IsVariable    bool    // true → render as var.{VariableName}
    VariableName  string  // e.g. "pingone_environment_id"
    OriginalValue string  // raw UUID, preserved for import blocks
}

func (r ResolvedReference) Expression() string  // "var.name" or "type.label.field"
```

### RawHCLValue

```go
type RawHCLValue string
```

Wraps a string that should be written to HCL without quoting. Custom transforms return this for values like `jsonencode(...)` expressions.

### ExportOrchestrator

Coordinates the full export pipeline:

```go
// internal/core/orchestrator.go

type ExportOrchestrator struct {
    registry   *schema.Registry
    processor  *Processor
    client     clients.APIClient
    progressFn ProgressFunc
}

func NewExportOrchestrator(registry, processor, client, ...opts) *ExportOrchestrator

func (o *ExportOrchestrator) Export(ctx context.Context, opts ExportOptions) (*ExportResult, error)
```

**Pipeline steps**:
1. Discover resource types from registry for the client's platform/service
2. Order by declared dependencies (topological sort via Kahn's algorithm)
3. For each type: fetch from API → process via Processor → assign unique labels → populate graph
4. Resolve cross-resource references (replace UUID strings with `ResolvedReference` values)
5. Validate dependency graph
6. Return `ExportResult`

```go
type ExportOptions struct {
    SkipDependencies bool    // Raw UUIDs instead of Terraform references
    GenerateImports  bool
    EnvironmentID    string
}

type ExportResult struct {
    ResourcesByType []*ExportedResourceData
    Graph           *graph.DependencyGraph
    EnvironmentID   string
}
```

The caller (cmd/export.go) is responsible for formatting, variable extraction, import block generation, and module assembly using the returned `ExportResult`.

### Transforms

Standard transforms are registered in `internal/core/transforms.go`:

| Transform | Description |
|-----------|-------------|
| `passthrough` | Returns value unchanged |
| `base64_encode` | Encodes string to base64 |
| `base64_decode` | Decodes base64 string |
| `json_encode` | Marshals value to JSON string |
| `json_decode` | Parses JSON string to Go value |
| `jsonencode_raw` | Marshals to `jsonencode(...)` HCL expression (returns `RawHCLValue`) |
| `to_string` | Converts any value to string |
| `value_map` | Looks up value in `ValueMap`, with `ValueMapDefault` fallback |
| `custom` | Dispatches to named function in `CustomHandlerRegistry` |

### Custom Handlers

For complex resources that cannot be expressed declaratively:

```go
// internal/core/custom_handlers.go

type CustomHandlerFunc func(data interface{}, def *schema.ResourceDefinition) (map[string]interface{}, error)

type CustomTransformFunc func(value interface{}, apiData interface{}, attr *schema.AttributeDefinition, def *schema.ResourceDefinition) (interface{}, error)

type CustomHandlerRegistry struct { ... }
func NewCustomHandlerRegistry() *CustomHandlerRegistry
func (r *CustomHandlerRegistry) RegisterHandler(name string, fn CustomHandlerFunc)
func (r *CustomHandlerRegistry) RegisterTransform(name string, fn CustomTransformFunc)
```

Custom handlers are registered during `init()` in resource files and queued via `CustomHandlerQueue` for bulk loading at startup.

### API Data Helpers

Reflection-based field readers in `internal/core/apidata.go`:

```go
func ReadStringField(data interface{}, fieldName string) string
func ReadBoolField(data interface{}, fieldName string) bool
func ReadInterfaceField(data interface{}, fieldName string) interface{}
```

Used by the processor's `type_discriminated_block` implementation (to evaluate `skip_conditions`) and by custom handlers.

### Name Sanitization (Canonical Utilities)

**Always use the shared utilities in `internal/utils/sanitize.go`**. Do not inline sanitization logic.

- `utils.SanitizeResourceName(name)` — Produces valid Terraform resource names: hex-encodes special characters and prepends `pingcli__`. Used for HCL resource labels.
- `utils.SanitizeMultiKeyResourceName(keys...)` — Combines multiple key components (e.g., variable name + context) with hex-encoding and `pingcli__` prefix. For resources with multi-field labels.
- `utils.SanitizeVariableName(name)` — Produces valid Terraform variable identifiers: replaces any non-alphanumeric and non-underscore character with `_`. Used for Terraform variable names in modules and tfvars files.

---

## Layer 3: Output Formatters

### OutputFormatter Interface

```go
// internal/formatters/formatter.go

type OutputFormatter interface {
    Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
    FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
    FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error)
    FileExtension() string
}

const (
    FormatHCL    = "hcl"    // .tf files
    FormatTFJSON = "tfjson" // .tf.json files
)

func NewFormatter(format string) (OutputFormatter, error)
```

### Implemented Formatters

| Format | Package | Extension | Description |
|--------|---------|-----------|-------------|
| HCL | `internal/formatters/hcl/` | `.tf` | hclwrite-based, produces Terraform HCL |
| tfjson | `internal/formatters/tfjson/` | `.tf.json` | json.MarshalIndent, produces Terraform JSON |

### Adding a New Format

1. Create `internal/formatters/{format}/formatter.go`
2. Add an adapter struct in `internal/formatters/formatter.go`
3. Register in `NewFormatter()` switch
4. Add format value to `--output-format` flag validation in `cmd/export.go`

---

## Layer 4: API Client Abstraction

### APIClient Interface

```go
// internal/clients/interface.go

type APIClient interface {
    ListResources(ctx context.Context, resourceType string, envID string) ([]interface{}, error)
    GetResource(ctx context.Context, resourceType string, envID string, resourceID string) (interface{}, error)
    Platform() string
    Service() string
}
```

The orchestrator calls **only** `ListResources`. It never calls `GetResource` directly. `ListResources` returns fully-populated SDK structs. When the underlying list API returns summary data, the implementation internally calls `GetResource` for each item (list-then-get pattern).

### Platform Package Structure

Each service package follows a unified pattern under `internal/platform/{platform}/{service}/`:

```
internal/platform/pingone/davinci/
├── client.go              # Client struct, dispatches through handler table
├── dispatch.go            # Handler dispatch table + custom handler queuing
├── resource_application.go
├── resource_connection.go
├── resource_flow.go
├── resource_flow_deploy.go
├── resource_flow_enable.go
├── resource_flow_policy.go
└── resource_variable.go
```

- `dispatch.go` defines the `resourceHandler` dispatch table and `registerResource()` function
- Each `resource_*.go` file self-registers via `init()` — no edits to other files needed
- Custom handlers and transforms are also registered in `init()`

```go
// resource_variable.go
func init() {
    registerResource("pingone_davinci_variable", resourceHandler{
        list: listVariables,
        get:  getVariable,
    })
}
```

### Adding a New Resource

1. Create `resource_{short_name}.go` in the service package
2. Implement `list` and `get` functions
3. Register via `init()` calling `registerResource()`
4. Optionally register custom handlers/transforms
5. Create the YAML definition in `definitions/{platform}/{service}/`
6. No edits to `client.go` or `dispatch.go` required

### Currently Supported Resources

| Resource Type | Custom Handler |
|--------------|----------------|
| `pingone_davinci_variable` | None (fully declarative) |
| `pingone_davinci_flow` | None (fully declarative) |
| `pingone_davinci_flow_deploy` | None (projection struct) |
| `pingone_davinci_flow_enable` | None |
| `pingone_davinci_connector_instance` | `handleConnectorProperties` (custom transform) |
| `pingone_davinci_application` | None |
| `pingone_davinci_application_flow_policy` | None |

---

## Dependency Management

### Dependency Graph

```go
// internal/graph/graph.go

type DependencyGraph struct { ... }

func New() *DependencyGraph
func (g *DependencyGraph) AddResource(resourceType, id, name string)
func (g *DependencyGraph) AddDependency(from, to ResourceNode, field, location string)
func (g *DependencyGraph) GetReferenceName(resourceType, id string) (string, error)
func (g *DependencyGraph) GenerateTerraformReference(resourceType, resourceID, attribute string) (string, error)
func (g *DependencyGraph) TopologicalSort() ([]ResourceNode, error)
func (g *DependencyGraph) DetectCycles() [][]ResourceNode
func (g *DependencyGraph) Validate() error
```

Thread-safe. Supports cycle detection via DFS, topological sorting via Kahn's algorithm, and dangling edge validation.

### Reference Resolution

The orchestrator handles reference resolution in two passes after all resources are processed:

1. **Direct references**: Attributes with `references_type` in the schema are resolved via graph lookup. Environment references always become `var.pingone_environment_id`. Unknown references fall back to variable references.

2. **Correlated references**: Non-string reference attributes (e.g., numeric version numbers) are resolved by correlating with sibling attributes that already resolved to the same `ReferencesType`.

---

## Module Generation

The module generator (`internal/module/generator.go`) creates a complete Terraform module structure:

```
output-dir/
├── ping-export-module/          # Child module
│   ├── versions.tf
│   ├── variables.tf
│   ├── outputs.tf
│   ├── pingone_davinci_flow.tf
│   ├── pingone_davinci_variable.tf
│   └── ...
├── ping-export-module.tf        # Root module block
├── ping-export-variables.tf     # Root variable declarations
├── ping-export-imports.tf       # Import blocks (optional)
└── ping-export-terraform.auto.tfvars  # Variable values
```

---

## Variable Extraction

The `VariableExtractor` (`internal/variables/extractor.go`) evaluates schema-driven rules to determine which resource attributes should become module variables:

- Checks the `variable_eligible` flag on attribute definitions
- Applies `variables.eligible_attributes` rules for naming and descriptions
- Handles `type_discriminated_block` value unwrapping for tfvars output
- Custom transforms can also produce variables via `TransformResultWithVariables`
- **Variable names are sanitized via `utils.SanitizeVariableName()` to ensure valid Terraform identifiers**

---

## Import Block Generation

The import generator (`internal/imports/generator.go`) creates Terraform 1.5+ import blocks by expanding placeholders in `import_id_format`:

- `{env_id}` → environment ID
- `{resource_id}` → resource ID
- `{<attr_name>}` → resource attribute value

---

## Export Pipeline (cmd/export.go)

The CLI export command orchestrates the full pipeline:

1. Parse flags and environment variables → `config.ResolveFlags()`
2. Build PingOne SDK client with OAuth2 authentication
3. Load YAML definitions from embedded FS into `schema.Registry`
4. Create `Processor` with custom handler registry
5. Create `ExportOrchestrator` with registry, processor, and API client
6. Run `orchestrator.Export()` → `ExportResult`
7. For each resource type in result:
   - Format resources via `OutputFormatter`
   - Generate import blocks via `imports.Generator`
   - Extract variables via `variables.VariableExtractor`
8. Assemble module structure via `module.Generator`
9. Write files to output directory

---

## Example Resource Definition

```yaml
# definitions/pingone/davinci/variable.yaml
metadata:
  platform: pingone
  service: davinci
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

conditional_defaults:
  - target_attribute: mutable
    set_value: true
    when_all:
      - attribute_empty: value
      - attribute_equals:
          name: mutable
          value: false

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

---

## Testing

Tests are organized by scope:

- **Unit tests** (`*_test.go` alongside source): Individual function testing — schema loading, attribute transformation, reference generation, name sanitization
- **Integration tests** (`tests/integration/`): Multi-component tests exercising the full pipeline
- **Acceptance tests** (`tests/acceptance/`): Live environment tests

Run all internal tests:
```bash
go test ./internal/... -v -count=1
```

Run acceptance tests (requires live PingOne environment):
```bash
go test ./tests/acceptance/... -v -count=1
```
