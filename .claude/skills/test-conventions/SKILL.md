---
name: test-conventions
description: How tests are organised, named, and written in this repo. Use when writing new tests or assessing test coverage.
---

## Test types and locations

| Type | Location | Notes |
|------|----------|-------|
| Unit tests | `internal/{package}/*_test.go` (alongside source) | No live env required |
| Acceptance tests | `tests/acceptance/` | Build tag `acceptance`; requires live PingOne env |
| Regression tests | `tests/regression/` | Matrix-based output diffing; runs in CI on PRs |
| Test fixtures | `tests/testdata/flows/` | Real DaVinci flow JSON; use for flow-related tests |

## Frameworks

- `github.com/stretchr/testify/assert` — non-fatal assertions
- `github.com/stretchr/testify/require` — fatal assertions (stops the test immediately on failure)

## Naming

- Functions: `Test{FunctionName}_{Scenario}` — e.g., `TestProcessResource_NilInput`
- Files: `{name}_test.go` in the same package as the code under test

## Table-driven tests

Use table-driven tests for any function with multiple input cases. Always include nil, empty, and edge-case inputs.

```go
func TestSanitizeResourceName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty string", "", "pingcli__"},
        {"normal name", "my-resource", "my-resource"},
        {"special chars", "foo/bar", "pingcli__foo_2F_bar"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            require.Equal(t, tt.expected, utils.SanitizeResourceName(tt.input))
        })
    }
}
```

## Running tests

```bash
# All unit tests
go test ./internal/... -v -count=1

# Single test
go test ./internal/core/... -run TestProcessResource_NilInput -v -count=1

# With coverage
go test ./internal/... -coverprofile=cover.out -count=1
go tool cover -func=cover.out
```

Use `-count=1` to disable test result caching.

## Acceptance tests

```bash
go test -tags acceptance ./tests/acceptance/... -v -timeout 120m
```

Required env vars: `PINGCLI_PINGONE_ENVIRONMENT_ID`, `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID`, `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID`, `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET`, `PINGCLI_PINGONE_REGION_CODE`.

## Coverage target

New code: >80%. Check with `go tool cover -func=cover.out` after running tests with `-coverprofile`.
