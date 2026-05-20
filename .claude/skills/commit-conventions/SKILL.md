---
name: commit-conventions
description: Commit message format and changelog entry process for this repo. Use when committing changes or preparing a PR.
---

## Commit message format

```
{scope}: {Short description}

{Optional body explaining what changed and why.}
```

Common scopes: `resource/{resource_type}`, package name (e.g., `core`, `formatters/hcl`), or `docs`.

Examples:
```
resource/pingone_davinci_flow: Add embedded sub-flow reference resolution
core: Fix nil pointer in processor when source_path is missing
docs: Update ARCHITECTURE.md with embedded reference pipeline
```

## Changelog entries

Every user-facing change (feature, bug fix, new resource, docs, etc.) requires a changelog entry in `.changelog/pr-{N}.txt`.

**Create via script:**
```bash
./shared-configs/release-notes/scripts/create-changelog-entry.sh <PR-NUMBER> <TYPE> "Description"
```

**Or create manually** — file: `.changelog/pr-{N}.txt`:
````
```release-note:{type}
Description of the change
```
````

For resource-scoped changes, prefix the description:
````
```release-note:enhancement
`resource/pingone_davinci_variable`: Added support for masked secret detection
```
````

## Changelog entry types

| Type | When to use |
|------|-------------|
| `feature` | New user-facing capability |
| `enhancement` | Improvement to existing functionality |
| `bug` | Bug fix |
| `new-resource` | New Terraform resource supported |
| `breaking-change` | Requires user action on upgrade |
| `documentation` | Documentation-only change |
| `internal` | Internal change (excluded from release notes) |
| `security` | Security-related change |
| `deprecation` | Deprecated feature or functionality |

Multiple entries in one file are fine — use separate code blocks.
