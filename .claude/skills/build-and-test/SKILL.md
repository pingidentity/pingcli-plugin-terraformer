---
name: build-and-test
description: How to build, test, vet, lint, and validate definitions in this repo. Use when running verification commands before committing or reviewing code.
---

## Build

```bash
make build         # go mod tidy + go build
go build ./...     # build without installing
```

## Unit tests (no live environment needed)

```bash
make test                          # all packages, 5 min timeout
go test ./internal/... -v -count=1 # internal packages only
go test ./internal/core/... -run TestName -v -count=1  # single test
```

## Acceptance tests (requires live PingOne environment)

```bash
make testacc   # 120 min timeout
```

Required env vars for acceptance tests:
- `PINGCLI_PINGONE_ENVIRONMENT_ID`
- `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID`
- `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID`
- `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET`
- `PINGCLI_PINGONE_REGION_CODE`

## Vet and lint

```bash
make vet          # go vet ./...
make lint         # golangci-lint run ./...
go fmt ./...      # format (or: make fmt)
```

## YAML definition validation

```bash
go run ./tools/validate-definitions definitions/
```

Run this any time a YAML definition in `definitions/` is created or modified.

## Coverage

```bash
go test ./internal/... -coverprofile=cover.out -count=1
go tool cover -func=cover.out
```

## Regression (compares current branch output vs. main)

```bash
make regression-local
```

Builds both `main` and the current branch, exports with each, and diffs output. Additions are acceptable; deletions/replacements flag a regression.

## Pre-commit checklist

```bash
make build && make vet && go fmt ./... && make test
go run ./tools/validate-definitions definitions/   # if any YAML changed
```

The `make devcheck` target runs build, vet, fmt, lint, test, and testacc in sequence — only use it when a live PingOne environment is available.
