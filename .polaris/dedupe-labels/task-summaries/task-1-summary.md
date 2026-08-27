# Task 1 Summary

Status: DONE

## Changes

- Scoped `DependencyGraph` label usage by `(resource type, requested label)` using a private comparable key.
- Preserved type-plus-ID node lookup, graph APIs, concurrency locking, and same-type `_2`, `_3`, ... suffix allocation.
- Added focused graph regressions for equal labels across types, insertion-order independence, same-type suffixes, and type-plus-ID lookup with edges.

## Tests

- `go test ./internal/graph -count=1`
- `go test ./internal/... -count=1`
- `make vet`
- `go build ./...`
- `git diff --check`

All checks passed.

## Commit

Included in the single task commit using the repository commit convention: `graph: Scope label allocation by resource type`.
