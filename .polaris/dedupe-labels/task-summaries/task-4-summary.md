# Task 4 Summary

Status: DONE

## Changes

- Added HCL formatter boundary coverage for equal labels across resource types, canonical `ResourceData.Label` usage, type-qualified references, runtime `depends_on`, and omission of unresolved dependencies.
- Added Terraform JSON formatter coverage for the same equal-label/reference/dependency behavior, including JSON resource keys and dependency expressions.
- Added HCL and Terraform JSON import coverage proving canonical labels are used while environment, parent, resource, and attribute import IDs remain raw values.
- Added schema-driven import generator coverage for canonical labels and raw import IDs.
- Added command output matching coverage proving equal labels are selected independently by resource type and output values remain type-qualified.
- No production code, formatter-side deduplication, YAML definitions, or list command paths were changed; existing `--list-resources` and `list-outputs` construction remains type-qualified by inspection.

## Tests

- `go test ./internal/formatters/... ./internal/imports ./cmd -count=1` — passed.
- `gofmt -w internal/formatters/hcl/formatter_test.go internal/formatters/tfjson/formatter_test.go internal/imports/generator_test.go cmd/export_outputs_test.go` — passed.
- `git diff --check` — passed.

## Deviations

- No narrowly focused testability adjustment was required in `cmd/export.go` or `cmd/list_outputs.go`.
- No changelog entry was added because this commit only adds internal regression tests.

## Commit

`60a4291 tests: Verify type-qualified formatter and output addresses`
