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

The regression tests run against a static PingOne environment that must stay stable between test runs. The environment is **managed as Terraform**: its config lives in [`environment/`](environment/) and drift is repaired by the `regression-env-apply` workflow (below). Manual edits to the environment will show up as drift and should be reverted via the repair workflow, not reproduced in the config.

### Environment Requirements

The environment should contain representative configuration across all supported resource types:

- [x] DaVinci variables (multiple, with different types) — 45 in the snapshot
- [x] DaVinci flows (with node configurations and dependencies) — 35 (+ deploy/enable)
- [x] DaVinci connector instances (with properties) — 26
- [x] DaVinci applications (with flow policy assignments) — 3
- [x] DaVinci flow policies (with flow assignments) — 19

### Environment Setup

The environment is defined by the Terraform config in [`environment/`](environment/) — see that README for the layout, secrets handling, and state-adoption runbook. To recreate it from scratch: `terraform apply` the config into a new PingOne environment (see the `environment/` README), then update the secrets below to point at the new environment.

### GitHub Actions Secrets

The following secrets must be configured in the repository under a GitHub Environment named `regression`:

| Secret | Description |
|--------|-------------|
| `PINGCLI_PINGONE_ENVIRONMENT_ID` | Environment ID containing the worker application |
| `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID` | Target environment ID to export (the static regression env) |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID` | Worker application client ID |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET` | Worker application client secret |
| `PINGCLI_PINGONE_REGION_CODE` | PingOne region code (NA, EU, AP, CA, AU, SG) |

The worker application this workflow uses has **read-only** DaVinci permissions.

### GitHub Environment Protection

The `regression` environment should have **required reviewers** enabled. This ensures that pull requests from forks cannot run exports against the PingOne environment without maintainer approval.

To configure:
1. Go to **Settings → Environments → New environment** → name it `regression`
2. Enable **Required reviewers** and add maintainers
3. Add the secrets listed above to this environment

> Note: `regression.yaml` documents this gating but does not currently declare `environment: regression` in the job YAML — the export job runs with repo secrets. The repo does not use the environments feature today (the `regression-env-apply` workflow below also uses repo-level secrets), so fork-PR protection for both workflows rests on the workflow triggers themselves: both are either PR-gated (repo secrets readable in PR runs) or dispatch-only (`regression-env-apply`).

## Repairing the Environment (drift apply)

When the live environment is "out of order" — mutated by a test run, edited by hand, or partially rebuilt — converge it from the committed Terraform config:

1. Go to **Actions → regression-env-apply → Run workflow**
2. Run once with `mode: plan`. The run produces a plan summary and a downloadable plan artifact showing exactly what would change.
3. Review the plan. If it matches the expected repair, run again with `mode: apply` (the apply applies the saved plan from that run).
4. The workflow ends with a post-apply verification plan — it must report "No changes."

The workflow is `workflow_dispatch` only and never runs on schedule or on PRs. An optional `-target` input scopes a repair to a single resource. Terraform state lives in the AWS S3 bucket configured at init (from the provider secrets).

**Do not** repair the environment by hand in the PingOne/DaVinci consoles — that reintroduces drift. If a repair would change the intended baseline (e.g. the drift was actually a wanted change), change the Terraform config via PR instead of applying against the old config.

### regression-env-apply secrets

Configured as **repo-level** GitHub Actions secrets (Settings → Secrets and variables → Actions — the repo does not use the environments feature):

| Secret | Description |
|--------|-------------|
| `TERRAFORM_PROVIDER_ENV_BASE64` | Base64-encoded shell exports for the **provider and backend**: AWS credentials (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`), state bucket config (`TF_VAR_tf_state_bucket`, `TF_VAR_tf_state_region`), and the write-scoped PingOne worker credentials (`PINGONE_CLIENT_ID`, `PINGONE_CLIENT_SECRET`, `PINGONE_ENVIRONMENT_ID`, `PINGONE_REGION_CODE`). Same shape as the test repo's `localsecrets`. |
| `TERRAFORM_TFVARS_BASE64` | Base64 of the exported **`ping-export-terraform.auto.tfvars` file** — the workflow decodes it under exactly that name so Terraform auto-loads it. Generated by `environment/export.sh`; regenerate and re-upload whenever the environment's variables change. |

Upload from the 1Password-backed localsecrets file (evaluates `op://` refs and pipes — resolved values never touch disk):

```bash
op inject -i .scratch/regression-env-localsecrets.env | base64 | \
  gh secret set TERRAFORM_PROVIDER_ENV_BASE64 --repo pingidentity/pingcli-plugin-terraformer
```

Both secrets can also be managed through `environment/export.sh`: `--upload-only` uploads the reviewed tfvars as `TERRAFORM_TFVARS_BASE64` (export and upload are deliberately separate steps — always review the generated values first; requires a gh account with repository-secret write access, e.g. `gh auth switch --user samir-gandhi`).

One-off setup:
1. **AWS**: create a private, versioned S3 bucket with public access blocked, plus an IAM user whose keys can read/write that bucket. Store the keys in 1Password (the `platform-test-pingcli-terraformer-regression-env-US` item).
2. **PingOne**: create a worker application `regression-env-tf-writer` in the regression environment with DaVinci read **and write** permissions. Store its client ID/secret in the same 1Password item.
3. **Provider blob**: `.scratch/regression-env-localsecrets.env` holds the `op://` references; evaluate and upload it as shown above.
4. **Variables**: run `./tests/regression/environment/export.sh` (with `op run` and the read-only credential), review the generated `ping-export-terraform.auto.tfvars` (scrub anything unintended), then upload it base64-encoded as `TERRAFORM_TFVARS_BASE64`. Keep the master copy in 1Password.

## Adding a New Resource Type to the Environment

When a new resource type gets export support, add a representative live resource to the regression environment so the regression matrix can exercise it:

1. Land the exporter support (the usual new-resource flow).
2. Hand-author one representative resource in `environment/ping-export-module/` — new file `pingone_<type>.tf`, label following the `pingcli__<sanitized-name>` convention, literal values (not new root variables). If it needs a secret value, follow the `davinci_variable_testPW_value` pattern: sensitive variable, value in 1Password + the tfvars CI secret.
3. If the resource has API-write-only secret attributes (like connector `clientSecret`), add `lifecycle { ignore_changes = [...] }` for them with a comment.
4. Dispatch `regression-env-apply` with `mode: apply`. A new resource is absent from state, so the plan shows it as a create; use the `-target` input to scope the run to the new resource if you want a smaller blast radius.
5. After the apply, run a full `mode: plan` — it must report "No changes." If other resources show up in the plan, the new resource interacted with existing ones: fix the config, don't hand-edit the environment.
6. Confirm the regression workflow picks the type up (run the matrix or rely on the next PR run) and tick the checkbox above.
7. Changelog: `release-note:new-resource` for the exporter change (the environment/config change itself is `internal`).

The environment config is **hand-curated going forward** — do not re-export over it (that would clobber lifecycle blocks and hand additions). Re-export into a scratch directory if you want to diff the live environment against the config; see `environment/README.md`.

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
