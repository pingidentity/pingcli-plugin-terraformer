# Task 2 Summary

Status: DONE

## Changes

- Made orchestrator filter exclusion bookkeeping type-aware with a comparable `(resource type, API ID)` identity.
- Updated direct filtering, upstream expansion marking, and reference fallback lookup to use the same typed identity.
- Preserved fallback naming, canonical environment fallback behavior, IncludeUpstream behavior, and unknown graph-ID handling.
- Added focused regression coverage for equal raw IDs across resource types in direct filtering, upstream expansion, and direct reference fallback lookup.
- Defined the source mock fixture with `TargetID` for the direct filtering regression.

## Tests

- `go test ./internal/core -run 'TestResolveOneReference_ExcludedResource|TestExportOrchestrator_Export_FilterExclusionIsTypeAware|TestExportOrchestrator_Export_IncludeUpstreamExclusionIsTypeAware' -count=1` — passed.
- `go test ./internal/core -count=1` — failed only at the pre-existing unrelated `TestResolveEmbeddedReferences_DisambiguatedName_StableAcrossEnvironments` failure in `internal/core/embedded_references_test.go:1316`.
- `git diff --check` — passed.

## Commit

`9c9f100 core: Scope filter exclusions by resource type`
