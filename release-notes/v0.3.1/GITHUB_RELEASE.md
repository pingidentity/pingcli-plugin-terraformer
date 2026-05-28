### ENHANCEMENTS

[a82baf2](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/a82baf2) Added support for AU (Australia) and SG (Singapore) PingOne region codes. [#22](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/22)
[98a12a7](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/98a12a7) `resource/pingone_davinci_flow`: Use SDK `DaVinciExportFlowVersionResponse` type for flow variable dependency extraction; bump SDK to v0.11.0 [#23](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/23)

### BUG FIXES

[be58c20](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/be58c20) `resource/pingone_davinci_flow`: Fixed export failure (HTTP 500) in environments with many or large DaVinci flows. The SDK's `GetFlows()` list endpoint returns full flow graph data in a single unbounded response, which exceeds the 10 MB AWS API Gateway limit at scale. The list call now uses a minimal raw HTTP request with `?attributes=id,name` (~120 KB regardless of environment size), then fetches each flow individually via `GetFlowById` as before. [#15](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/15)

