---
name: new-resource
description: Generic workflow for adding a new resource to the exporter. Covers pre-analysis, YAML definition, Go handler, tests, and live validation. Read the platform-specific supplement alongside this (e.g., new-resource-pingone) for provider URLs, SDK paths, and platform-specific dependency analysis patterns.
---

## Overview

Adding a new resource requires exactly two files and zero edits to existing code:
1. `definitions/{platform}/{category}/{short_name}.yaml`
2. `internal/platform/{platform}/resource_{short_name}.go`

Do not proceed to implementation until Phase 1 pre-analysis is complete. The pre-analysis determines correctness of every field in the YAML definition.

See `contributing/DEVELOPER_HANDBOOK.md` for full YAML schema reference and the `contributing/ARCHITECTURE.md` for the processing pipeline.

---

## Phase 1: Pre-Analysis

Complete all analyses before writing any code. Document findings as you go — they feed directly into the YAML definition.

### 1a. SDK struct inspection

Use `go doc` to inspect the Go SDK struct (ppds `go-package-documentation-browsing` skill):

```bash
go doc <sdk_package> <TypeName>
go doc -src <sdk_package> <TypeName>
```

For each field, record:
- **Go field name** — this is what goes in `source_path`, NOT the JSON tag
- **Go type** — pointer, slice, map, nested struct, UUID, SDK enum, or JSON-encoded string
- **Whether it's a UUID reference** to another exported resource type
- **Whether it contains JSON-encoded blobs** — indicates potential `jsonencode_raw` + `EmbeddedReferenceRule`

Inspect the list and get API methods:
```bash
go doc <sdk_package> <APIName>
```

Determine:
- Does the list API return fully-populated structs or summaries? If summaries, the handler must call get for each item internally (list-then-get pattern — see `resource_flow.go`).
- Is there pagination, and what type?

**Critical**: `source_path` must use Go field names, not JSON tags. `Id` not `ID`, `Environment.Id` not `environment.id`. Verify every path against the actual struct.

### 1b. Terraform provider schema review

Fetch the provider documentation for the resource. The platform-specific supplement has the correct provider URL.

From the schema, identify:
- Every writable attribute and its type
- Which attributes are `Required`, `Optional`, `Computed`
- Which attributes are sensitive
- The import ID format (shown in the Import section)
- Attributes that behave differently from the SDK struct — non-standard handling, computed-only fields, type mismatches

**Attributes in the provider schema but absent from the SDK struct** are the most common source of bugs — flag these for investigation before proceeding.

### 1c. Variable and dependency identification

**Variable-eligible attributes** (will become Terraform module variables):
- Environment-specific IDs (`environment_id` is always a variable)
- Secrets and sensitive values
- Endpoint URLs, region-specific values
- Any attribute the provider marks as sensitive

**Cross-resource references** (fields containing UUIDs pointing to other exported resource types):
- Explicit: fields named `*_id`, `flow_id`, etc.
- Embedded: UUIDs buried inside JSON-encoded blob attributes — requires an `EmbeddedReferenceRule` alongside the YAML definition

Use the platform-specific supplement for platform patterns that help identify dependencies (e.g., HAL links in PingOne responses).

### 1d. Non-standard behavior identification

Compare the SDK struct to the provider schema to find:
- Provider-computed fields the API doesn't return → `computed: true`, no `source_path`
- Type mismatches between API and provider → `to_string` or `value_map` transform
- Masked secret fields (API returns a sentinel like `"************"`) → `masked_secret` config
- Attributes requiring `conditional_defaults` (attribute B must be `true` when attribute A is empty)
- Fixed-value attributes regardless of API response → `override_value`

---

## Phase 2: YAML Definition

Create `definitions/{platform}/{category}/{short_name}.yaml`. Use an existing definition as a template.

Key decisions from Phase 1:
- `label_fields`: attributes that form a human-readable unique Terraform label. Labels are automatically sanitized — no manual sanitization needed.
- `variable_eligible: true`: all attributes from Phase 1c variable list
- `references_type`: all cross-resource UUID fields from Phase 1c
- `transform: jsonencode_raw`: JSON-encoded blob attributes
- `sensitive: true` + `masked_secret`: masked API values
- `computed: true` (no `source_path`): provider-computed fields the API doesn't return

Validate after writing:
```bash
go run ./tools/validate-definitions definitions/
```

Fix all validation errors before proceeding.

---

## Phase 3: Go Handler

Create `internal/platform/{platform}/resource_{short_name}.go`. See `contributing/DEVELOPER_HANDBOOK.md` for the full handler template and registration pattern.

Key points:
- Register via `init()` calling `registerResource()` — no edits to other files needed
- If the list API returns summaries, call get for each item internally
- Register `EmbeddedReferenceRule` entries in `init()` if embedded UUIDs were identified in Phase 1c
- Never inline name sanitization — use `internal/utils/` functions

Build and vet:
```bash
go build ./internal/platform/{platform}/...
go vet ./internal/platform/{platform}/...
```

---

## Phase 4: Unit Tests

Create `internal/platform/{platform}/resource_{short_name}_test.go`.

Cover handler behavior, error cases, and any custom transforms. Use table-driven tests with `testify/require`. Always include nil, empty, and edge-case inputs.

```bash
go test ./internal/... -v -count=1
```

All unit tests must pass before proceeding.

---

## Phase 5: Live Environment Validation

**Requires live credentials.** See the platform-specific supplement for required env vars and credential setup.

**Prerequisite**: the target environment must contain at least one instance of the new resource type.

### 5a. Build and export

```bash
make build
./pingcli-terraformer export --out /tmp/export-test --output-format hcl --include-imports
```

### 5b. Validate output

```bash
ls /tmp/export-test/ping-export-module/{resource_type}.tf
```

Verify:
- All expected attributes are present with correct values
- No raw UUIDs remain for attributes that should be resolved references
- `depends_on` blocks are present where expected
- Variable-eligible attributes appear in `variables.tf` and `.auto.tfvars`
- Import blocks are correct

Check for unresolved UUIDs:
```bash
grep -E '"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"' \
  /tmp/export-test/ping-export-module/{resource_type}.tf
```

Any unresolved UUID that should be a reference is a bug — fix `references_type` config or add an `EmbeddedReferenceRule`.

### 5c. Acceptance tests

```bash
make testacc
```

If no acceptance test exists for the new resource, add one to `tests/acceptance/` following existing patterns.

### 5d. Regression check

```bash
make regression-local
```

Additions are expected. Deletions or replacements in previously-exported resources are regressions — fix before opening a PR.

---

## Phase 6: Pre-PR Verification

```bash
make build
make vet
go fmt ./...
make test
go run ./tools/validate-definitions definitions/
make regression-local
```

Create a changelog entry:
```bash
./shared-configs/release-notes/scripts/create-changelog-entry.sh <PR-NUMBER> new-resource \
  "\`resource/{resource_type}\`: Added export support"
```

---

## Escalate (do not proceed autonomously) when:

- Phase 1 reveals an attribute requiring a new YAML feature or transform type not already in the schema
- The SDK struct differs significantly from the provider schema in a way that can't be explained by `computed` or `override_value`
- Live export produces unexpected output that can't be fixed by adjusting the YAML definition
- The resource has circular dependencies with an existing resource type
- Implementing this resource requires changes to `internal/core/` or `internal/graph/`
