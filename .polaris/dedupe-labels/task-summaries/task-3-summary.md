# Task 3 Summary

Status: DONE

## Changes

- Added deterministic orchestrator regressions for equal labels across two target resource types, covering both target definition insertion orders and asserting the typed standard reference uses the target type's unsuffixed canonical label.
- Added an embedded `RawHCLValue` regression for equal labels and IDs across target types, asserting typed replacement and the dependency graph edge target.
- Added runtime `RuntimeDependsOn` coverage for equal labels across types and preserved empty labels for unknown IDs.

## Tests

- `go test ./internal/core -run 'TestExportOrchestrator_Export_EqualLabelsAcrossTypes|TestResolveEmbeddedReferences_EqualLabelsAcrossTypes|TestResolveDependsOnResources_EqualLabelsAcrossTypes|TestExportOrchestrator_Export_ResolvesReferences|TestResolveEmbeddedReferences_SingleSubFlow|TestResolveDependsOnResources' -count=1` — passed.
- `go test ./internal/core -count=1` — failed only at the known unrelated `TestResolveEmbeddedReferences_DisambiguatedName_StableAcrossEnvironments` regression in `internal/core/embedded_references_test.go:1357` (the test was previously reported at line 1316 before these additions).
- `gofmt -w internal/core/orchestrator_test.go internal/core/embedded_references_test.go` — passed.
- `git diff --check` — passed.

## Deviations

- No production behavior was changed. No YAML definitions, live environment, formatter, or public reference shape changes were made.
- No changelog entry was added because this commit only adds internal regression tests.

## Commit

To be recorded in the single task commit using the repository convention.
