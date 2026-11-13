# DaVinci Variable Contexts: API, Provider, and Exporter Behavior

Reference for the four DaVinci variable context types and how the exporter
handles each — including the duplicate-name semantics that motivated
[issue #166](https://github.com/pingidentity/pingcli-plugin-terraformer/issues/166)
and its fix (PR #167).

Evidence key:
- **[API]** — observed against a live PingOne environment via the PingOne
  surface of the DaVinci API (`/v1/environments/{id}/variables`), 2026-11.
- **[TF]** — observed with `pingidentity/pingone` provider **1.21.1**
  (the version pinned in this repo's test data), apply-tested.
- **[Export]** — exporter source + live export runs.
- **[Docs]** — https://docs.pingidentity.com/davinci/variables/davinci_variables.md
- **⚠ Unverified** — claimed but not yet tested; see the open-questions
  section at the end.

## Summary table

| Context | Created by | Uniqueness domain [API] | Owner exposed on GET? [API] | Value readable? | Provider can create duplicates? [TF] | Exporter label [Export] |
|---|---|---|---|---|---|---|
| `company` | UI (Variables tab), API | `(name, context)` — enforced, `UNIQUENESS_VIOLATION` | n/a (no owner) | Yes (unless secret) | N/A — duplicates rejected | `pingcli__<name>_company` |
| `flow` | **Only from within a flow** (UI Variables tab cannot create them) | `(name, context, flow.id)` — same name allowed across different flows | Yes — `flow.id` on the response | Yes | **Yes** — same name + different flows applies cleanly (provider 1.21.1) | `pingcli__<name>_flow_<flowLabel>` |
| `flowInstance` | UI Variables tab, flows, and **system** (DaVinci auto-creates some per deployed flow) | `(name, context)` — enforced, `UNIQUENESS_VIOLATION` | **No** — shared across flows by design; `flow.id` absent from list/GET and silently ignored on PUT | Yes (unless secret); system vars (`cancelUrl`, `logoUrl`, `ldapApiUrl`, …) return no value | **No** — provider surfaces the API's `UNIQUENESS_VIOLATION` | `pingcli__<name>_flowInstance` |
| `user` | UI / flows | Not tested | Not tested | Not tested | Not tested | Not handled distinctly — same schema path |

## Per-context detail

### `company`

One shared value across the whole environment, usable in all flows [Docs].

- **API [API]**: creation requires `mutable`; duplicates are rejected with
  400 `UNIQUENESS_VIOLATION` ("Construct already exists"). Secret-type
  company variables are supported (value never readable after creation).
- **Lifecycle**: independent of flows. Deleting a flow never deletes a
  company variable.
- **Terraform [TF]**: standard resource; no flow attribute.
- **Exporter [Export]**: labels on `[name, context, flow.id]`, but `Flow`
  is always absent → label is the two-part `pingcli__<name>_company`,
  unchanged by the #167 fix. `value` is variable-eligible; secret values
  export as `secret_string` backed by a sensitive Terraform variable.

### `flow`

Bound to one specific flow. The value persists until changed; both the
Variables tab and the flow's own execution can update it (latest wins)
[Docs].

- **API [API]**: cannot be created without `flow.id` (400
  `REQUIRED_VALUE` "must have required property 'flow.id'"). Uniqueness is
  enforced per `(name, context, flow)`: the same name on two different
  flows **succeeds**; the same name twice on the same flow fails with
  `UNIQUENESS_VIOLATION`. The `flow.id` is returned on both list and GET.
- **Lifecycle [user observation, ⚠ API-unverified]**: when the owning flow
  removes its reference, the variable is deleted with it. (Consistent with
  the API making `flow.id` required and immutable-ish, but no direct
  delete-on-unlink test was run.)
- **Terraform [TF]**: provider 1.21.1 has a `flow` attribute (nested,
  `flow.id` required). **Two same-named flow-context variables on
  different flows apply cleanly** — the provider does not add uniqueness
  beyond the API's.
- **Exporter [Export]**: `Flow.Id` is processed into the nested `flow`
  object attribute (already used for the variable→flow `depends_on` edge).
  Since #167 the label includes it: `pingcli__<name>_flow_<flowId>` —
  matching the API's true uniqueness domain. Before #167 two such
  variables produced identical labels and the export aborted.

### `flowInstance`

Usable across multiple flows. Each flow's instance takes its value from
the flow execution if set there, otherwise inherits the value set on the
Variables tab [Docs].

- **API [API]**: creation requires `mutable`; duplicates are rejected with
  400 `UNIQUENESS_VIOLATION` — the uniqueness domain is `(name, context)`
  across the whole environment. **No owner/flow link exists**: the
  response carries no `flow` field, `PUT`ing a `flow.id` onto a
  flowInstance variable returns 200 but silently drops the field, and the
  flow-version export shows flows referencing flowInstance variables **by
  `name##SK##context`**, not by ID (two different flows' exports both
  resolved to the same variable ID). Sharing is the feature: one variable,
  many flows.
- **System flowInstance variables [API/Export]**: DaVinci auto-creates
  per-deployed-flow system variables such as `cancelUrl`, `logoUrl`,
  `ldapApiUrl` (historically per flow; see legacy note below). These
  typically have no readable value via the API.
- **Terraform [TF]**: duplicates are impossible to create — the provider
  surfaces the API's `UNIQUENESS_VIOLATION` and the apply aborts.
- **Exporter [Export]**: label `pingcli__<name>_flowInstance` (no flow
  segment — correct, since no owner exists). Value-less variables get the
  `mutable → true` conditional default (provider requirement). Duplicate
  `(name, context)` pairs **cannot be created by the current API or
  provider**, but **may exist in environments provisioned by the legacy
  DaVinci API**, which reportedly did not enforce name uniqueness. Since
  #167 the exporter degrades rather than aborts: deterministic `_2`,
  `_3`, … suffixes (resources sorted by stable API ID) plus a warning —
  the "tool remediates" behavior requested in #166.

### `user`

One value per user; the user must be identified before the variable can
be used in a flow [Docs]. **Not exercised in the #166 investigation** —
no API, provider, or exporter testing was done. The exporter treats it
through the same schema path (`[name, context]` labels, no flow
attribute), which is expected to be correct since user variables are not
flow-owned, but this is unverified.

## The legacy API question (unverified)

The current PingOne-surface API provably enforces `(name, context)`
uniqueness for `company` and `flowInstance` (verified 2026-11, both via
raw API and the provider). Yet #166 reporters observe environments
containing duplicate flowInstance variables. The working hypothesis:

> The **legacy DaVinci API** (pre-migration, the one the
> `legacyDaVinciAPI` worker app and the old `davinci` host surface served)
> did not enforce `(name, context)` uniqueness for flowInstance variables,
> so environments created/managed through it can hold duplicates the
> current API would never produce.

**This has not been verified.** We did not test the legacy API surface
(it requires the legacy DaVinci host/API-key auth, and the duplicate
`legacyDaVinciAPI` worker app in the acctest environment was not used).
What we did establish:

- The current API rejects duplicate creation (API + provider evidence
  above).
- System flowInstance variables are per-deployed-flow constructs with
  stable names (`cancelUrl` etc.), which is exactly the pattern that
  would collide if creation went through a non-enforcing path — e.g. two
  flows deployed at times when creation was allowed to double-create.
- The reporter's environment (v0.6.0 export) exhibited the duplicates,
  and their `--exclude-resources` workaround listed
  `pingcli__<name>_flowInstance` for both a "flowInstance" and "company"
  spelling of each variable, indicating both contexts were duplicated.

If the legacy hypothesis is ever worth confirming: exercise
`POST /variables` against the legacy DaVinci host with a duplicate
`(name, context)` pair, or inspect a legacy-managed environment's
variables for same-`(name, context)` pairs. Until then, treat "legacy
artifacts" as the assumption behind the suffix-fallback behavior, not a
proven fact.

## Exporter handling recap (post-#167)

1. **Fetch**: one flat list, `GET /environments/{id}/variables` — every
   variable appears exactly once (1:1 with API objects; the exporter never
   duplicates).
2. **Process**: standard schema pipeline. `Flow.Id` → nested `flow`
   object (present only for flow context). Secret values →
   `secret_string` + sensitive variable; value-less variables →
   `mutable` forced `true` (provider requirement).
3. **Label**: `label_fields: [name, context, flow.id]`:
   - `company` → `pingcli__<name>_company`
   - `flow` → `pingcli__<name>_flow_<flowLabel>` (the owning flow's own Terraform label, prefix-stripped — name-derived and environment-portable, never the UUID; the `@type` label-field form resolves it through the dependency graph)
   - `flowInstance` → `pingcli__<name>_flowInstance`
   - `user` → `pingcli__<name>_user` (untested; same mechanism)
4. **Collisions**: `assignLabels` never aborts. Colliding labels get
   deterministic `_2`/`_3` suffixes (ID-ordered, stable across exports)
   and a warning. For the current API this can only fire on legacy data.
5. **Extracted variables** (tfvars): names are disambiguated by resource
   label when two distinct resources would derive the same name — same
   collision class, same trigger set.

## Open questions

1. Legacy DaVinci API duplicate-creation behavior (see above).
2. `user` context: API semantics, provider behavior, exporter output —
   all untested.
3. Flow-context delete-on-unlink lifecycle (user-observed, not
   API-verified).
4. Whether flowInstance system variables should be exported at all
   (provider import support for them is unconfirmed; dropping them via
   `enabled: false` was considered and deferred in #167).
