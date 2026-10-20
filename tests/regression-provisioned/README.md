# Provisioned-Environment E2E Testing

This directory contains the runner for the provisioned-environment E2E test
tier. It provisions a throwaway PingOne environment via Terraform itself
(`terraform-test-data/`, authenticated with an org-admin credential scoped to
this run's lifecycle), applies hand-authored fixture resources into it, then
runs the same base-vs-PR export/compare pipeline as `tests/regression/`
against that fresh environment. It is additive to the static-environment tier
in `tests/regression/`, which is unchanged.

## Running

Prerequisites: `terraform`, `jq`, and the five org-admin environment
variables below (in CI these are repository secrets under the
`regression-provision` GitHub Environment).

```
PINGCLI_PINGONE_ORGADMIN_CLIENT_ID
PINGCLI_PINGONE_ORGADMIN_CLIENT_SECRET
PINGCLI_PINGONE_ORGADMIN_ENVIRONMENT_ID
PINGCLI_PINGONE_ORGADMIN_REGION_CODE
PINGCLI_PINGONE_ORGADMIN_LICENSE_ID
```

Run locally with:

```
make e2e
```

The runner provisions the environment (`tools/tf-regression-provision`),
builds base and PR export binaries, runs the export matrix from
`tests/regression/matrix.json` against the provisioned environment, compares
outputs, and always tears the environment down. Set
`E2E_KEEP_ENVIRONMENT=1` to skip teardown for debugging — destroy manually
afterwards with `terraform destroy` in `terraform-test-data/root`. Reports
land in `regression-reports/`.

CI runs this tier on demand via the `regression-provisioned` workflow
(`workflow_dispatch` only — per-run environment creation has cost and
rate-limit implications that haven't been signed off on for a schedule).

## Fixture-module contract

Every fixture lives in `terraform-test-data/<category>/<short_name>/`
(e.g. `terraform-test-data/sso/group/`) with exactly:

- `main.tf` — one `variable "environment_id"` (string) plus one
  `resource "pingone_<type>" "e2e"` block. Resource names use the
  `pingcli-terraformer-e2e-*` convention. Consumers take referenced resource
  IDs as additional input variables.
- `versions.tf` — an unversioned `required_providers { pingone = ... }`
  block. The provider **version pin lives only in
  `terraform-test-data/root/versions.tf`**; modules inherit it.

Producers expose IDs via `outputs.tf` (e.g. `resource_id`,
`application_id`) — the one sanctioned extension to the two-file pattern.
Every fixture must be exportable by the exporter: before adding one, check
the matching `definitions/pingone/<platform>/<short_name>.yaml` exists and
that the fixture's attribute shape passes the provider's validators (run
`terraform validate` in `terraform-test-data/root` — no credentials needed).

Wiring: add a `module` stanza to `terraform-test-data/root/main.tf` in
dependency order. Terraform resolves cross-module references through the
output→input wiring; the stanza order is for readability. A fixture may be a
self-contained multi-resource module only when the dependency is internal
(e.g. `group_nesting` needs two groups, but both belong to that fixture's
concern).

Some child resources need no fixture of their own because PingOne generates
them when their parent is created and the exporter reads them as 1:1
singletons — `resource_secret` and `application_secret` are covered this way
by the `resource` and `application` fixtures.

## SSO fixture coverage

| State | Resources |
|---|---|
| Fixtured | `population`, `group`, `group_nesting`, `sign_on_policy`, `password_policy`, `population_default`, `resource`, `resource_scope`, `resource_attribute`, `resource_scope_openid`, `resource_scope_pingone_api`, `application`, `application_sign_on_policy_assignment`, `application_attribute_mapping` |
| Auto-covered (no fixture needed) | `resource_secret`, `application_secret` |
| Known gaps — not fixturable in an SSO-only environment | `population_default_identity_provider` (needs a `pingone_identity_provider`; no IdP exists here), `application_flow_policy_assignment` (needs the DaVinci service; the throwaway environment enables SSO only) |
| Follow-up candidates | `application_resource`, `application_resource_grant` (both fixturable with the output→input pattern; chain three modules) |

Note on `pingone_resource_scope_pingone_api`: the exporter silently filters
every built-in scope not matching `^p1:(read|update):user(:...)*$` (the
provider's name validator rejects them), so the fixture scope is named
`p1:read:user:e2e` to land inside the exported subset. A fixture name outside
that pattern would provision fine but provide zero export coverage.

Note on `pingone_population_default`: the provider adopts the environment's
auto-created default population on create (import id is the environment ID),
so it renames rather than fights the bootstrap population. The default
population consequently appears in the export twice — once as
`pingone_population`, once as `pingone_population_default` — which is
intentional dual coverage, not duplication.
