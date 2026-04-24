# Regression Testing

This directory contains configuration and documentation for the PR regression testing process.

## Overview

The regression test system compares Terraform export output between the base branch and a PR branch to detect unintended changes. It runs automatically on pull requests via GitHub Actions.

### What it checks

- **Acceptable changes (pass)**: New resources, new attributes, new files — additions to export output
- **Breaking changes (fail)**: Removed resources, removed attributes, changed values, removed files — deletions or modifications to existing output

### Example

Adding a new `pingone_environment` resource to the export is acceptable. However, changing an existing reference from `var.pingone_environment_id` to `pingone_environment.my_env.id` is a breaking change because it modifies existing output that users may depend on.

## Static Regression Environment

The regression tests run against a static PingOne environment that should **not be modified** between test runs. Changes to the environment will cause false positives in regression detection.

### Environment Requirements

The environment should contain representative configuration across all supported resource types:

- [ ] DaVinci variables (multiple, with different types)
- [ ] DaVinci flows (with node configurations and dependencies)
- [ ] DaVinci connector instances (with properties)
- [ ] DaVinci applications (with flow policy assignments)
- [ ] DaVinci flow policies (with flow assignments)

### Environment Setup

<!-- TODO: Add specific environment setup instructions or Terraform files -->

To recreate the regression environment:

1. Create a new PingOne environment with DaVinci enabled
2. Configure the resources listed above
3. Create a worker application with DaVinci API Read permissions
4. Store the credentials as GitHub Actions secrets (see below)

### GitHub Actions Secrets

The following secrets must be configured in the repository under a GitHub Environment named `regression`:

| Secret | Description |
|--------|-------------|
| `PINGCLI_PINGONE_ENVIRONMENT_ID` | Environment ID containing the worker application |
| `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID` | Target environment ID to export (the static regression env) |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID` | Worker application client ID |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET` | Worker application client secret |
| `PINGCLI_PINGONE_REGION_CODE` | PingOne region code (NA, EU, AP, CA, AU) |

### GitHub Environment Protection

The `regression` environment should have **required reviewers** enabled. This ensures that pull requests from forks cannot run exports against the PingOne environment without maintainer approval.

To configure:
1. Go to **Settings → Environments → New environment** → name it `regression`
2. Enable **Required reviewers** and add maintainers
3. Add the secrets listed above to this environment

## Flag Matrix

The file [`matrix.json`](matrix.json) defines the export flag combinations tested in the regression workflow. Each entry runs a separate export with both the base and PR binaries.

Current matrix entries:

| Name | Format | Skip Deps | Skip Imports | Include Imports | Include Values |
|------|--------|-----------|-------------|-----------------|----------------|
| `default-hcl` | hcl | no | no | no | no |
| `hcl-skip-deps` | hcl | yes | no | no | no |
| `hcl-include-all` | hcl | no | no | yes | yes |
| `hcl-skip-imports` | hcl | no | yes | no | no |
| `default-tfjson` | tfjson | no | no | no | no |
| `tfjson-skip-deps` | tfjson | yes | no | no | no |

To add a new flag combination, add an entry to `matrix.json`.

## Running Locally

### Using the shell script (recommended)

The `tests/regression/run-local.sh` script runs the full regression test matrix locally, comparing base branch vs current branch exports across all flag combinations defined in `matrix.json`.

#### Prerequisites

Before running the script, set the required environment variables:

```bash
export PINGCLI_PINGONE_ENVIRONMENT_ID="<environment-id>"
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="<client-id>"
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="<client-secret>"

# Optional:
export PINGCLI_PINGONE_REGION_CODE="NA"                    # Default: NA
export PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID="<export-env-id>"  # Default: uses PINGCLI_PINGONE_ENVIRONMENT_ID
export REGRESSION_BASE="main"                              # Default: main
```

#### Running the test

```bash
./tests/regression/run-local.sh
```

This command:
1. Validates prerequisites (required environment variables and `jq`)
2. Creates a temporary `git worktree` for the base branch
3. Builds a binary from the base branch
4. Builds the current branch's binary
5. Iterates through all matrix entries, exporting resources with both binaries
6. Compares outputs using the regression-compare tool
7. Prints a summary table showing pass/fail status for each matrix entry
8. Copies JSON reports to `regression-reports/` directory
9. Cleans up temporary files and worktree

The script exits with code 0 if all comparisons pass (no breaking changes), or 1 if any breaking changes are detected.

### Using the Makefile target

The `make regression-local` target provides an alternative approach to automated local regression testing:

```bash
make regression-local
```

This command:
1. Creates a temporary `git worktree` for the base branch (`main` or override with `REGRESSION_BASE`)
2. Builds a binary from the base branch
3. Builds the current branch's binary (already done by `build` dependency)
4. Exports resources with both binaries
5. Compares the outputs using the regression-compare tool
6. Cleans up the worktree and temporary files

#### Overriding the base branch

By default, regression tests compare against `main`. To test against a different branch:

```bash
REGRESSION_BASE=develop make regression-local
```

### Manual approach

To compare outputs from two different branches manually:

```bash
# On the current branch: build and export
make build
./pingcli-terraformer export --out ./output-current-branch

# Switch to base branch
git checkout main
make build
./pingcli-terraformer export --out ./output-base-branch

# Switch back and compare
git checkout -
go run ./tools/regression-compare/ \
  --base-dir ./output-base-branch \
  --pr-dir ./output-current-branch \
  --report-file report.json
```

## Comparison Tool

The comparison tool lives at `tools/regression-compare/`. It:

1. Walks both output directories recursively
2. Compares `.tf` files using HCL-aware semantic comparison (not line-by-line diff)
3. Compares `.tf.json` files using JSON-aware comparison
4. Classifies each difference as **acceptable** (addition) or **breaking** (deletion/modification)
5. Produces a human-readable summary on stdout
6. Optionally writes a JSON report via `--report-file`
7. Exits with code 0 (no regressions) or 1 (breaking changes found)
