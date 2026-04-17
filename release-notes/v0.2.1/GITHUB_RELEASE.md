### ENHANCEMENTS

[c98c663](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/c98c663) `resource/pingone_davinci_flow`: Flow variable dependency fetch errors are now treated as warnings instead of fatal errors. The export continues without `depends_on` references to DaVinci variables rather than failing entirely. [#9](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/9)

### BUG FIXES

[c98c663](https://github.com/pingidentity/pingcli-plugin-terraformer/commit/c98c663) `resource/pingone_davinci_flow`: Fixed flow version export URL construction to use the SDK's configured regional host instead of the default NA endpoint. This resolves 403 errors when the worker application exists in a different environment than the export target. [#9](https://github.com/pingidentity/pingcli-plugin-terraformer/pull/9)

