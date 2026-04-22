### ENHANCEMENTS

[020c949](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/020c949) `resource/pingone_davinci_flow`: Added user-friendly error message when the PingOne API returns `jsLinks` in a legacy string format that the SDK cannot parse. The error now directs users to a known issue with workaround steps to convert the affected flow's `jsLinks` to the current object format. See [#10](https://github.com/pingidentity/pingcli-plugin-terraformer/issues/10). [#11](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/11)
[e535e6b](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/e535e6b) `resource/pingone_davinci_flow`: Added support for new flow attributes: `multi_value_source_id` (edge data), `capability_class` (node data), `subtype` (trigger), and `preview_form_rendering_updates` (settings) [#12](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/12)

