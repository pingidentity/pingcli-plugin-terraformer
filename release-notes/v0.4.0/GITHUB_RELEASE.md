### FEATURES

[b89759d](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/b89759d) `cmd/list-outputs`: New `list-outputs` subcommand that enumerates all possible output attribute paths (`resource_type.label.attr`) for exported resources. Supports `--depth` (default 1), `--include-resources`, `--exclude-resources`, and `--include-upstream`. Output is newline-delimited on stdout and pipeable directly to `--output-attribute-file`. [#126](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/126)
[b89759d](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/b89759d) `cmd/export`: New `--output-attribute` (repeatable, glob `*` supported in label position) and `--output-attribute-file` flags that populate `outputs.tf` with Terraform output blocks for specified resource attribute paths. [#126](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/126)

### ENHANCEMENTS

[b89759d](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/b89759d) `resource/pingone_davinci_application`: Added missing `api_key.value` computed attribute to schema definition. [#126](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/126)
[f4cad53](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/f4cad53) `resource/pingone_davinci_flow`: Resolved DaVinci form node theme references (`theme.value` and the indirect `themeId.value` rich-text-wrapped UUID used when `theme.value` is "useThemeId") to `pingone_branding_theme`, matching the existing `form.value` behavior. Until `pingone_branding_theme` export support lands, the theme ID is emitted as an overridable Terraform variable. [#128](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/128)

### BUG FIXES

[03b388c](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/03b388c) `resource/pingone_davinci_flow`: Fixed export failure when `trigger.type` is `AUTHENTICATION` — the exported `input_schema` block is now suppressed, resolving the Terraform provider error "input_schema must not be configured when trigger.type is set to AUTHENTICATION" [#123](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/123)
[b89759d](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/b89759d) `formatters/hcl`, `formatters/tfjson`: Computed-only attributes inside nested object blocks (e.g. `api_key.value`) were incorrectly written into resource configuration. The computed skip guard now applies at all nesting levels. [#126](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/126)
[1b65107](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/1b65107) `resource/pingone_davinci_flow`: Fixed multi-outcome node routing (e.g. PingOne Forms nodes with multiple buttons/links) being dropped on export, which broke the flow when the exported HCL was re-applied. Requires `terraform-provider-pingone` with `outcomes` support on `graph_data.elements.nodes.*.data`. [#127](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/127)

