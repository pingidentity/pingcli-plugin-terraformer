### ENHANCEMENTS

[a6b5fae](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/a6b5fae) `resource/pingone_davinci_flow`: Added support for the flow settings `custom_timeout_error_screen_css`, `custom_timeout_error_screen_html`, `custom_timeout_error_screen_message`, and `use_custom_timeout_error_screen` attributes (see #144). Requires pingone-go-client v0.13.0, now pinned in go.mod. [#152](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/152)

### BUG FIXES

[c0d54b7](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/c0d54b7) Fixed fallback Terraform variables for not-yet-exported reference targets (e.g. `pingone_branding_theme`, `pingone_password_policy`) and embedded DaVinci node references silently colliding by derived name instead of by the underlying UUID. Two resources referencing different instances of the same missing target type, or two DaVinci nodes sharing a title but referencing different UUIDs, now produce distinct variables instead of one resource silently inheriting another's ID. [#139](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/139)
[45ae223](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/45ae223) Fixed Terraform references and filter fallback handling when different resource types share the same label or API resource ID. [#146](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/146)
[df08efb](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/df08efb) `resource/pingone_davinci_flow`: Fixed a regression where exporting an environment with a legacy `jsLinks` format flow showed a raw SDK unmarshal error instead of the actionable hint (see #10). [#147](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/147)
[c14d0b5](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/c14d0b5) Fixed `--list-resources` printing resource addresses to the logger (stderr) instead of stdout — addresses are now pipeable, matching the `list-outputs` command behaviour. [#148](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/148)

### NOTES

[be61cb3](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/be61cb3) The static PingOne regression environment is now managed as Terraform: its config lives in `tests/regression/environment/` with remote S3 state, and a new manually-triggered `regression-env-apply` GitHub Actions workflow can repair environment drift on demand. Variable values are no longer committed (they ride in repo-level CI secrets); see `tests/regression/README.md`. [#149](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/149)

