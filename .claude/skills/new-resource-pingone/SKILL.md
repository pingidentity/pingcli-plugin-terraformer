---
name: new-resource-pingone
description: PingOne-specific supplement to the new-resource skill. Covers the correct SDK package, API service naming conventions, pagination patterns, HAL link dependency analysis, and provider documentation location. Read alongside new-resource/SKILL.md.
---

## SDK package

All PingOne resources (base and DaVinci) share one SDK package:

```
github.com/pingidentity/pingone-go-client/pingone
```

The client struct in this repo is `internal/platform/pingone.Client`, which holds `apiClient *pingone.APIClient`. The API groups on `APIClient` are:

| Field | Covers |
|---|---|
| `DaVinciVariablesApi` | DaVinci variables |
| `DaVinciFlowsApi` | DaVinci flows |
| `DaVinciApplicationsApi` | DaVinci applications + flow policies |
| `DaVinciConnectorsApi` | DaVinci connector instances |
| `DaVinciFlowVersionsApi` | Flow version details |
| `FlowPoliciesApi` | Flow policies (standalone) |
| `EnvironmentsApi` | PingOne environments |
| `ConfigurationManagementApi` | Configuration resources |

Use `go doc` to inspect what's available:
```bash
go doc github.com/pingidentity/pingone-go-client/pingone APIClient
go doc github.com/pingidentity/pingone-go-client/pingone <TypeName>
go doc -src github.com/pingidentity/pingone-go-client/pingone <TypeName>
```

## Terraform provider

All PingOne and DaVinci resources are in the `pingidentity/pingone` provider (DaVinci resources were migrated from the legacy `pingidentity/davinci` provider — do not reference that provider).

Provider documentation:
```
https://registry.terraform.io/providers/pingidentity/pingone/latest/docs/resources/<resource_name>
```

For DaVinci resources the resource name is typically `pingone_davinci_*`.

## SDK struct inspection: what to look for

### Go type conventions

The SDK is generated — type names follow consistent patterns:

| What | Pattern | Example |
|---|---|---|
| Single resource response | `<Resource>Response` | `DaVinciVariableResponse` |
| Collection response | `<Resource>CollectionResponse` | `DaVinciVariableCollectionResponse` |
| HAL links on response | `<Resource>ResponseLinks` | `DaVinciVariableResponseLinks` |
| Embedded in collection | `<Resource>CollectionResponseEmbedded` | `DaVinciVariableCollectionResponseEmbedded` |
| Cross-resource relationship | `ResourceRelationshipDaVinciReadOnly` | (see below) |

`source_path` must use Go field names, not JSON tags. Common pitfall — the SDK uses `Id` (not `ID`), `Href` (not `href`).

### SDK type coercions handled automatically by the processor

- `uuid.UUID` → `string` via `.String()`
- SDK enum types (e.g., `DaVinciVariableResponseDataType`) implementing `fmt.Stringer` → `string`
- Pointer types → dereferenced to underlying value

### Relationship fields

Cross-resource references appear as typed relationship structs, not raw strings:
- `ResourceRelationshipDaVinciReadOnly` — has `.Id` field (string UUID)
- `ResourceRelationshipReadOnly` — similar pattern for PingOne base resources

These fields indicate a dependency on another exported resource type. The `source_path` to the UUID is `<FieldName>.Id` (e.g., `Flow.Id` for a variable's parent flow).

## HAL link analysis: identifying dependencies

PingOne API responses follow the HAL standard. Every response includes a `_links` field containing `JSONHALLink` entries. The `Href` string encodes the full URI path, which reveals parent resource IDs.

### Reading dependency signals from `_links`

Inspect the `*ResponseLinks` struct for the resource type:
```bash
go doc github.com/pingidentity/pingone-go-client/pingone <Resource>ResponseLinks
```

Named HAL links (not just `self` and `environment`) indicate parent resource relationships. For example, if `FlowPolicyResponseLinks` has a `DavinciApplication` link, the flow policy is a child of an application — look for the application ID in that link's `Href` path.

**Href path structure**: `https://<host>/v1/environments/<envID>/.../<parentID>/<resourcePath>`

Parse the path segments to extract parent IDs when a typed relationship field is absent but the link `Href` contains the parent resource's ID. This is the signal that a dependency exists even when no explicit ID field appears in the struct.

### Collection response `_links`

Collection responses have `self`, `environment`, and optionally `next` links. The presence of `next` means the API paginates — the SDK wraps this in `PagedIterator`.

## Pagination patterns

Two patterns are used across PingOne resources:

### Paginated (PagedIterator) — use when `{Resource}ApiService.Get{Resources}Execute` returns `PagedIterator[T]`

```go
func listThings(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
    iterator := c.apiClient.ThingsApi.GetThings(ctx, c.environmentID).Execute()

    var result []interface{}
    for pageCursor, err := range iterator {
        if err != nil {
            return nil, fmt.Errorf("list things: %w", err)
        }
        embedded := pageCursor.Data.Embedded
        things, ok := embedded.GetThingsOk()
        if ok && things != nil {
            for i := range things {
                result = append(result, &things[i])
            }
        }
    }
    return result, nil
}
```

### Non-paginated — use when `Execute` returns `(*CollectionResponse, *http.Response, error)`

```go
func listThings(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
    resp, _, err := c.apiClient.ThingsApi.GetThings(ctx, c.environmentID).Execute()
    if err != nil {
        return nil, fmt.Errorf("list things: %w", err)
    }
    embedded := resp.GetEmbedded()
    things := embedded.GetThings()
    result := make([]interface{}, len(things))
    for i := range things {
        result[i] = &things[i]
    }
    return result, nil
}
```

To determine which pattern applies:
```bash
go doc github.com/pingidentity/pingone-go-client/pingone <Resource>ApiService
```

Look at the `Get<Resources>Execute` method signature. `PagedIterator[T]` → use iterator. `(*CollectionResponse, *http.Response, error)` → use direct.

### Get (single resource)

```go
func getThing(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
    thingUUID, err := uuid.Parse(resourceID)
    if err != nil {
        return nil, fmt.Errorf("invalid thing ID: %w", err)
    }
    thing, _, err := c.apiClient.ThingsApi.GetThingById(ctx, c.environmentID, thingUUID).Execute()
    if err != nil {
        return nil, fmt.Errorf("get thing: %w", err)
    }
    return thing, nil
}
```

Note: some APIs take the resource ID as `string`, others as `uuid.UUID`. Check the method signature with `go doc`.

## List-then-get

Some list APIs return summary structs with fewer fields than the individual get endpoint. The `DaVinciFlowsApi` is the current example — `GetFlows` returns summaries, so `listFlows` calls `GetFlowById` for each item.

To detect this: compare the fields on the list response's embedded item type vs. the single response type. If the list type is missing fields the TF provider needs, use list-then-get.

## Dependency identification checklist

For the resource under analysis:

1. **Typed relationship fields** — any field of type `ResourceRelationshipDaVinciReadOnly` or similar. The `source_path` to the UUID is `<FieldName>.Id`. Add `references_type` in the YAML pointing to the resource type that owns that relationship.

2. **HAL `_links` named entries** — inspect `<Resource>ResponseLinks`. Non-`self`/non-`environment` named links indicate parent resources. Parse their `Href` path segments to determine if the parent's ID should be surfaced as a `references_type` attribute or a `depends_on` entry.

3. **Embedded JSON blobs** — attributes using `jsonencode_raw` may contain UUIDs inside JSON structures. Register an `EmbeddedReferenceRule` in `init()` for each. Use the flow resource's `subFlowId` rule in `resource_flow.go` as the reference implementation.

4. **Flow version endpoint** — for DaVinci flows specifically, variable dependencies are not in the standard GET response. They require a POST to the flow version endpoint (see `fetchFlowVariableDeps` in `resource_flow.go`). If a resource similarly requires a non-standard supplemental API call to discover dependencies, follow that pattern with a custom handler + `__depends_on` sentinel.

## Environment credentials

Required env vars for acceptance tests and live validation:

| Variable | Purpose |
|---|---|
| `PINGCLI_PINGONE_ENVIRONMENT_ID` | Auth/worker app environment |
| `PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID` | Target environment to export (defaults to auth env if unset) |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID` | OAuth client ID |
| `PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET` | OAuth client secret |
| `PINGCLI_PINGONE_REGION_CODE` | Region: `NA`, `EU`, `AP`, `CA`, `AU` |

The acceptance test helpers in `tests/acceptance/helpers.go` use these exact variable names. New acceptance tests should use `createTestClient(t)` from that package.

## Definition directory placement

The subdirectory under `definitions/pingone/` mirrors the **category grouping in the Terraform provider documentation sidebar**. This keeps schema comparisons easy — find the resource in the docs, use the same category name as the directory.

```
https://registry.terraform.io/providers/pingidentity/pingone/latest/docs/resources/<resource_name>
```

Look at the sidebar category the resource falls under. Use that as the directory name, lowercased with underscores. Examples:

| TF docs sidebar category | Definition directory |
|---|---|
| DaVinci | `definitions/pingone/davinci/` |
| SSO | `definitions/pingone/sso/` |
| MFA | `definitions/pingone/mfa/` |
| Protect | `definitions/pingone/protect/` |

**Exception**: `definitions/pingone/base/` contains `pingone_environment` (TF docs category: "Platform"). It was not renamed to `platform/` to avoid confusion with the Go package path `internal/platform/pingone/`. Do not rename it, and do not use `base/` as a template for new categories — new resources in the TF "Platform" category go in `definitions/pingone/platform/`.

**If the category directory doesn't exist yet**, create it and add it to the `go:embed` directive in `definitions/embed.go`:

```go
//go:embed pingone/base/*.yaml pingone/davinci/*.yaml pingone/sso/*.yaml
var FS embed.FS
```

All handlers go in `internal/platform/pingone/` regardless of category — single flat package, no sub-packages.
