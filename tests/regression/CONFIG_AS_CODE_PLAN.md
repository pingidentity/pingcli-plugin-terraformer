# Config-as-Code Plan for the Regression Environment

> **Status: SUPERSEDED (2026-09-02).** The "baseline, never applied" policy
> below was replaced by the managed-environment design: the config in
> `environment/` now has an S3 backend, the live resources are imported into
> that state, and the `regression-env-apply` workflow applies drift repairs
> from CI. See `environment/README.md` ("State and apply policy", "Adopting
> state") and `tests/regression/README.md` ("Repairing the Environment") for
> the current process. This document is kept as the historical record of the
> original approach.

Status: **plan only — not yet executed** (credentials not wired up at time of writing).

## Why

`tests/regression/README.md` documents that PR regression testing runs against a
**static PingOne environment that must not be modified between test runs** — but
the environment currently exists only by hand in PingOne. If that environment is
ever lost, drifted, or rebuilt, the static baseline is gone and regression
comparisons become meaningless. Storing the environment's own Terraform config
in this repo is the insurance policy: the environment becomes reproducible, and
future drift becomes visible as a `terraform plan` difference instead of a
mystery.

## Where the config lives

`tests/regression/environment/` in this repo, committed to the default branch.

```text
tests/regression/environment/
├── README.md                 # what this env is, how to rebuild it
├── versions.tf               # required terraform + provider versions
├── variables.tf              # generated; env ID/region via input variables
├── main.tf                   # generated module.tf + resource definitions
├── ping-export-module/       # generated module resources (see below)
└── .gitignore                # terraform state, plan, .terraform/
```

The directory is hand-curated, not blindly committed: the export output is
reviewed, secrets (client secrets, API keys, anything `--include-values`
captures) are scrubbed or converted to variables, then committed. The
`.gitignore` must exclude all Terraform state and `.terraform/`.

## What gets exported

Per `README.md`, the static environment should contain:

- DaVinci variables (multiple, with different types)
- DaVinci flows (with node configurations and dependencies)
- DaVinci connector instances (with properties)
- DaVinci applications (with flow policy assignments)
- DaVinci flow policies (with flow assignments)

Because the regression matrix also runs `tfjson` format, the exported config
should be captured in **both** `hcl` and `tfjson` formats to prove each path
works — or, more simply, exported once in `hcl` (the canonical config-as-code)
and the `tfjson` path is exercised by CI anyway. **Decision: export in `hcl`
only for the stored config-as-code**; the tfjson formatter is already covered
by the regression matrix itself.

## Prerequisites (not yet done)

1. **Credentials** — worker app credentials for the regression environment,
   stored in 1Password (per user convention) and referenced with `op://` env
   references, not committed:
   - `PINGCLI_PINGONE_ENVIRONMENT_ID`
   - `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID`
   - `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID`
   - `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET`
   - `PINGCLI_PINGONE_REGION_CODE`
   There is an existing `ping-cli-terraformer` item template at
   `~/projects/config-automation/.env/op-template-ping-cli-terraformer.json` —
   create a `ping-cli-terraformer-regression` item from it rather than a new
   template.
2. **1Password CLI signed in** (`op whoami` must succeed).
3. **Landed session work first** (see ordering below) so the config is
   generated with the fixed label-identity logic.

## Generation procedure (once credentials are ready)

Run from the repo root, after `polaris/dedupe-labels` (or its successor) is
merged to `main`:

```bash
# 1Password injects the secrets only into this subprocess
op run --env-file=<op:// env file> -- ./pingcli-terraformer export \
  --output-format hcl \
  --out ./tests/regression/environment/ \
  --include-values \
  --skip-dependencies=false
```

Then:
1. `terraform -chdir=tests/regression/environment init && plan` to verify it
   round-trips clean against the live environment.
2. Scrub/parametrize any literal secrets in the output (the `--include-values`
   flag may embed them — inspect before commit).
3. Commit, and note in the PR that this is the stored baseline of the static
   regression environment.

## Prerequisite session work — completion order

| # | Session | Work | Status | Blocks config-as-code? |
|---|---------|------|--------|------------------------|
| 1 | **dedupe-labels** | Fix cross-type label collisions in graph label allocation + filter exclusions, 5 commits on `polaris/dedupe-labels` | ✅ **Merged to main as PR #146** | ~~Yes~~ — resolved |
| 2 | **ref-pipe-platform-QoL-updates** | Pipeline template hardening, PR #30 in `pipeline-example-platform` | PR awaiting human merge approval | No |
| 3 | **migrate-davinci-export** | DaVinci export scripts + `--include-values` support in the `pipeline-example-platform-test` repo | Unpushed commits + one pending commit; no PR yet | No |

The only blocking prerequisite (PR #146) is done. Remaining blockers are purely
credential setup (1Password signin + item creation).
