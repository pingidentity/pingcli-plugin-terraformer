# pingcli-plugin-terraformer

A Go CLI plugin for `pingcli` that exports PingOne environment resources into Terraform HCL or Terraform JSON configuration files.

## What this repo is

Schema-driven export engine. YAML definitions in `definitions/pingone/` declare how each PingOne API resource maps to Terraform attributes. A generic Go processing engine (`internal/core/`) reads those definitions at runtime using reflection and produces a format-agnostic intermediate representation. Two output formatters convert that to files.

The key invariant: **standard resources require zero Go code** — only a YAML definition. Custom Go handlers exist only for genuinely complex transformations (multi-field correlation, deep JSON traversal).

## Key directories

| Path | Purpose |
|------|---------|
| `definitions/pingone/` | YAML resource definitions (embedded at compile time) |
| `internal/core/` | Processing engine: processor, orchestrator, transforms, custom handlers |
| `internal/formatters/` | Output formatters: `hcl/` and `tfjson/` |
| `internal/platform/pingone/` | PingOne API client + resource handlers (single flat package) |
| `internal/schema/` | Schema types, loader, registry, validator |
| `internal/graph/` | Dependency graph (topological sort, cycle detection) |
| `internal/utils/` | Sanitization utilities — always use these, never inline |
| `contributing/` | Architecture and developer guide docs |

## Architecture and conventions

See `contributing/ARCHITECTURE.md` for the full system design and processing pipeline.
See `contributing/DEVELOPER_GUIDE.md` for workflows (adding resources, platforms, formats).

Critical rules that agents must know:
- `source_path` in YAML uses **Go struct field names** (`Environment.Id`), not JSON tags (`environment.id`)
- Name sanitization must use `utils.SanitizeResourceName()`, `SanitizeMultiKeyResourceName()`, or `SanitizeVariableName()` — never inline
- All PingOne resource handlers live in `internal/platform/pingone/` — no sub-packages
- New resources require exactly two files (YAML + Go handler) with zero edits to existing files
- Formatters must not import `internal/graph/` — references are resolved in the orchestrator before formatters run
- The `__depends_on` sentinel key is how custom handlers emit runtime `depends_on` entries

## Build, test, and verification

See `.claude/skills/build-and-test/SKILL.md` for the full command reference.

Quick reference:
- `make test` — unit tests (no live env required, ~5 min)
- `make vet` — go vet
- `go fmt ./...` — formatting
- `go run ./tools/validate-definitions definitions/` — validate YAML definitions

## Test conventions

See `.claude/skills/test-conventions/SKILL.md`.

## Commit conventions

See `.claude/skills/commit-conventions/SKILL.md`.
