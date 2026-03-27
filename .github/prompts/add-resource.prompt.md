---
description: 'Add a new PingOne resource to the Terraform exporter. Provide the SDK type reference and Terraform provider schema reference.'
mode: 'Coordinator'
---

# Add New Resource to Terraform Exporter

You are orchestrating the addition of a new resource to the PingOne Terraform exporter.

## Required Inputs

The user must provide:
1. **SDK Type Reference** — Link or path to the Go SDK struct definition in `pingidentity/pingone-go-client`
2. **Terraform Provider Schema Reference** — Link to the Terraform registry docs or provider source for the resource in `pingidentity/terraform-provider-pingone`

If either is missing, ask for it before proceeding.

## Pre-Analysis (Coordinator performs directly)

Before delegating, gather raw facts:

1. **Read the SDK type** from the Go module cache at `~/go/pkg/mod/github.com/pingidentity/pingone-go-client@*/pingone/`.
   - Document every exported field: name, Go type, pointer vs value, JSON tag
   - For nested structs, recursively document their fields
   - For enum types (string aliases), list all constant values
   - Note which fields are `uuid.UUID` (coerced to string by processor)

2. **Read the Terraform provider schema** from the link provided.
   - Document every attribute: `terraform_name`, required/optional/computed, type
   - For nested attributes (blocks), recursively document
   - Note any `value_map` needs (enum conversions between API and Terraform values)

3. **Build the attribute mapping table**:

   | Terraform Name | TF Type | Required/Optional/Computed | SDK Source Path | SDK Go Type | Notes |
   |---|---|---|---|---|---|
   | id | string | computed | Id | uuid.UUID | |
   | name | string | required | Name | string | |
   | ... | ... | ... | ... | ... | |

   Rules for `source_path`:
   - Use **Go struct field names**, not JSON tags (e.g., `Id` not `id`, `ApiKey` not `APIKey`)
   - Dot notation for nested fields: `Environment.Id`, `BillOfMaterials.Products`
   - For deeply nested, trace the full path through structs

4. **Identify**:
   - Which attributes reference other resources (`references_type`)
   - Which attributes need `value_map` (enum code → Terraform string)
   - Which attributes are `variable_eligible`
   - Whether a custom handler or custom transform is needed
   - What the `label_fields` should be (usually `[name]`)
   - What the `import_id_format` is
   - Whether a projection struct is needed (when API data must be reshaped)
   - What the API list/get methods are on the SDK client

5. **Determine placement**:
   - Handler goes in `internal/platform/pingone/` (all PingOne handlers share one package)
   - YAML definition goes in `definitions/pingone/{category}/` (e.g., `base/`, `davinci/`) — subdirectories are organizational only

## Phase 1: Plan

Delegate to **Planner** with:
- The complete attribute mapping table
- The SDK struct definition
- The Terraform schema
- The identified complexities (custom handlers, value maps, projections)
- The target definition subdirectory (e.g., `base/`, `davinci/`)

Ask the Planner to produce an ordered task list with deliverables.

## Phase 2: Architecture Validation

Delegate to **Architect** with:
- The Planner's task list
- The attribute mapping table
- The target package location

Ask the Architect to validate:
- All PingOne handlers go in `internal/platform/pingone/` (single flat package)
- YAML definition follows existing patterns in `definitions/`
- SDK type coercion rules are respected (uuid.UUID → string, enum string aliases, pointers)
- Reference resolution patterns match existing definitions
- Whether any changes to core engine code are needed (reference resolution, processor)

If the Architect identifies issues, relay back to Planner for revision.

## Phase 3: Test First (TDD)

Delegate to **Tester** with:
- The finalized task list
- The YAML definition (or its specification)
- The handler function signatures

Ask the Tester to write:
- Unit tests for the resource handler (list/get functions)
- Unit tests for any custom transforms
- Processor integration tests verifying YAML definition produces correct `ResourceData`
- Any projection struct tests (field name ↔ YAML source_path alignment)

## Phase 4: Implementation

Delegate to **Implementer** with:
- The finalized task list
- The test files to make pass
- The attribute mapping table

Implementation artifacts:
1. **YAML definition** — `definitions/pingone/{category}/{short_name}.yaml`
2. **Resource handler** — `internal/platform/pingone/resource_{short_name}.go`
3. **Optional: Custom transform** — registered via `registerTransform()` in the handler file
4. **Optional: Projection struct** — for reshaping API data, defined in the handler file
5. **Optional: Core engine changes** — only when existing patterns don't support the resource

### YAML Definition Template

```yaml
metadata:
  platform: pingone
  resource_type: pingone_{resource_name}
  api_type: {SDKTypeName}
  name: {Human Readable Name}
  short_name: {short_name}
  version: "1.0"

api:
  sdk_package: github.com/pingidentity/pingone-go-client/pingone
  sdk_type: {SDKTypeName}
  list_method: {ApiService}.{ListMethod}
  get_method: {ApiService}.{GetMethod}
  id_field: id
  name_field: name
  pagination_type: cursor

attributes:
  # ID - always computed
  - name: ID
    terraform_name: id
    type: string
    source_path: Id
    computed: true

  # ... map each attribute from the mapping table ...

dependencies:
  depends_on: []
  import_id_format: "{resource_id}"

variables:
  eligible_attributes: []
```

### Resource Handler Template

```go
package pingone

import (
    "context"
    "fmt"
    "github.com/google/uuid"
)

func init() {
    registerResource("pingone_{resource_name}", resourceHandler{
        list: list{Resources},
        get:  get{Resource},
    })
}

func list{Resources}(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
    // Use paginated iterator or single-call API
    // Return []interface{} where each element is a pointer to the SDK struct
}

func get{Resource}(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
    // Parse resourceID as UUID if needed
    // Call SDK Get method
    // Return pointer to SDK struct
}
```

## Phase 5: Review

Delegate to **Reviewer** with all changed files. Ask to verify:
- YAML definition matches Terraform schema exactly
- Source paths use Go struct field names
- All required/optional/computed marks are correct
- Reference resolution is consistent with existing definitions
- Handler follows existing patterns (error wrapping, nil checks)
- Tests are comprehensive and pass

## Phase 6: Verify

Run verification commands:
```bash
go run ./tools/validate-definitions definitions/
go build ./internal/platform/pingone/...
go test ./internal/... -v -count=1
go vet ./...
```

## Phase 7: Documentation

Delegate to **Docs** to update:
- Resource list in README.md if applicable
- Architecture docs if patterns were extended

## Convergence Criteria

The resource is complete when:
1. `go run ./tools/validate-definitions` passes
2. `go build ./...` succeeds
3. `go test ./internal/... -count=1` passes
4. `go vet ./...` reports no issues
5. Reviewer agent has approved
6. Documentation is current

## Common Patterns Reference

### SDK Type Coercion (handled automatically by processor)
- `uuid.UUID` → `string` via `.String()`
- SDK enum types (string aliases like `EnvironmentTypeValue`) → `string`
- `*T` (pointer) → dereferenced to `T`

### Value Maps (for enum code ↔ Terraform string conversion)
```yaml
- name: Type
  terraform_name: type
  type: string
  source_path: Type
  transform: value_map
  value_map:
    API_VALUE_ONE: "terraform_value_one"
    API_VALUE_TWO: "terraform_value_two"
  value_map_default: ""
```

### Nested Objects
```yaml
- name: License
  terraform_name: license_id
  type: string
  source_path: License.Id
```

### Set of Nested Objects
```yaml
- name: Services
  terraform_name: services
  type: set
  source_path: BillOfMaterials.Products
  nested_attributes:
    - name: Type
      terraform_name: type
      type: string
      source_path: Type
```

### Reference Attributes
```yaml
- name: EnvironmentID
  terraform_name: environment_id
  type: string
  source_path: Environment.Id
  required: true
  references_type: pingone_environment
  reference_field: id
```

### Dependencies
```yaml
dependencies:
  depends_on:
    - resource_type: pingone_environment
      field_path: Environment.ID
      lookup_field: id
      reference_format: "var.pingone_environment_id"
  import_id_format: "{env_id}/{resource_id}"
```
