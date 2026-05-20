---
name: architecture
description: Schema-driven architecture rules, package placement, YAML definition structure, processing pipeline, and anti-patterns for this repo. Use when planning, implementing, or reviewing any change.
---

## Full reference

For complete type definitions, the embedded reference resolution system, all transform descriptions, and the full orchestrator pipeline, read `contributing/ARCHITECTURE.md`.

## Core principle: schema-driven

Standard resources require **zero Go code** — only a YAML definition. The core engine uses Go reflection to traverse SDK structs by the field paths in the YAML. Custom Go handlers exist only when the transformation genuinely cannot be expressed declaratively (e.g., deep JSON tree traversal, multi-field correlation).

## Separation of concerns

| What | Where |
|------|-------|
| WHAT to convert (resource definitions) | `definitions/{platform}/{category}/*.yaml` |
| HOW to convert (processing logic) | `internal/core/` |
| OUTPUT format (rendering) | `internal/formatters/{format}/` |
| API interaction + resource fetching | `internal/platform/{platform}/` |
| Schema types, loader, registry, validator | `internal/schema/` |
| Dependency graph (topo sort, cycle detection) | `internal/graph/` |
| Variable extraction | `internal/variables/` |
| Import block generation | `internal/imports/` |
| Module file generation | `internal/module/` |
| String utilities (sanitization) | `internal/utils/` |

## Processing pipeline

```
API Response (Go struct)
  → ResourceProcessor.ProcessResource()     [internal/core/processor.go]
    → ResourceData{ID, Name, Attributes}    map[string]interface{}
  → ExportOrchestrator.Export()             [internal/core/orchestrator.go]
    → resolveReferences()                   UUID → ResolvedReference
    → DependencyGraph population            [internal/graph/]
  → Formatter.Format(data, def, opts)       [internal/formatters/hcl/ or tfjson/]
    → Terraform HCL or JSON output
```

References are resolved in the orchestrator. Formatters receive `ResolvedReference` values and detect them via type assertion — they never do UUID lookups themselves.

## Platform package layout

`internal/platform/pingone/` is a **single flat package** — no sub-packages. It contains:
- `client.go` — Client struct implementing `clients.APIClient`
- `dispatch.go` — handler dispatch table and custom handler queue
- `resource_*.go` — one file per resource type with list/get functions and `init()` registrations

Adding a new resource: create one YAML file in `definitions/` and one `resource_*.go` file in `internal/platform/pingone/`. Zero edits to existing files.

## YAML definition structure

```yaml
metadata:
  platform: pingone                          # always "pingone", not "pingone-davinci"
  resource_type: pingone_davinci_variable    # Terraform resource type
  api_type: Variable                         # Go SDK type name
  name: DaVinci Variable
  short_name: variable
  version: "1.0"

api:
  sdk_package: github.com/pingidentity/pingone-go-sdk-v2/davinci
  sdk_type: Variable
  list_method: EnvironmentVariables.GetAll
  id_field: id
  name_field: name

attributes:
  - name: EnvironmentID
    terraform_name: environment_id
    type: string
    source_path: Environment.Id        # Go struct field names, NOT JSON tags
    references_type: pingone_environment
    reference_field: id

dependencies:
  import_id_format: "{env_id}/{resource_id}"
```

## Critical `source_path` rule

`source_path` values must use **Go struct exported field names**, not JSON tags.

| Wrong (JSON tag) | Correct (Go field name) |
|---|---|
| `environment.id` | `Environment.Id` |
| `ID` | `Id` |
| `apiKey` | `ApiKey` |

The processor uses `reflect` to traverse by exact field name. A JSON tag path silently produces nil.

## Sanitization

Always use the utilities in `internal/utils/` — never inline:
- `utils.SanitizeResourceName()` — hex-encodes special chars, adds `pingcli__` prefix
- `utils.SanitizeMultiKeyResourceName()` — same for composite keys
- `utils.SanitizeVariableName()` — replaces non-`[a-zA-Z0-9_]` with `_`

## Custom handlers

The `__depends_on` sentinel key in a `ResourceData.Attributes` map causes the orchestrator to emit a `depends_on` block in the output. Custom handlers use this to express runtime dependencies.

`RawHCLValue` — a string type that the HCL formatter emits unquoted (no `"` wrapping). Use for references like `var.pingone_environment_id`.

## Anti-patterns

| Anti-pattern | Why wrong | Correct approach |
|---|---|---|
| `if resourceType == "X"` in processor/orchestrator | Breaks schema-driven design | Use YAML definition or custom handler |
| Formatter imports `internal/graph/` | Formatter renders resolved data only | Resolve in orchestrator |
| `source_path: environment.id` | JSON tag, not Go field name | `source_path: Environment.Id` |
| Processing logic in `internal/platform/` | Duplicated when adding a second platform | Extract to `internal/core/` |
| Format-specific code in `internal/core/` | Couples core to output format | Move to `internal/formatters/` |
| `metadata.platform: pingone-davinci` | Platform subsystem in platform field | `platform: pingone` |
| Inline name sanitization | Misses edge cases, inconsistent output | Use `internal/utils/` functions |

## Review checklist

- [ ] `source_path` values use Go struct field names, not JSON tags
- [ ] No `if resourceType == "X"` branching in generic processing code
- [ ] Formatter does not import `internal/graph/`
- [ ] New code in correct package (platform-specific in `internal/platform/`, shared in `internal/core/`)
- [ ] No new packages without justification
- [ ] Name sanitization goes through `internal/utils/`
- [ ] Required YAML sections present: `metadata`, `api`, `attributes`, `dependencies`
- [ ] `metadata.platform` is `pingone`, not a subsystem name
