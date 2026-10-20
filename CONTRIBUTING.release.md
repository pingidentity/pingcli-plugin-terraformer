# Release Process

How to cut a release of the Ping CLI Terraformer Plugin.

## Overview

Releases are published by the `goreleaser` GitHub workflow (`.github/workflows/goreleaser.yaml`) whenever a `v*` tag is pushed. goreleaser builds and publishes:

- Binaries for darwin/linux/windows (amd64, arm64, 386) to GitHub Releases
- Linux packages (deb, rpm, apk) as release assets
- A Homebrew **cask** to [pingidentity/homebrew-tap](https://github.com/pingidentity/homebrew-tap) (`Casks/pingcli-plugin-terraformer.rb`)

## Release checklist

1. **Verify the tap config is current** — `.goreleaser.yaml` uses the `homebrew_casks` section (goreleaser deprecated `brews`). Run `goreleaser check` locally to confirm the config validates against the goreleaser version CI resolves to.

2. **Ensure all content branches are merged to `main`.** Tagging from a branch without the latest `.goreleaser.yaml` will publish stale artifacts — e.g. tagging a branch still on the old `brews` config republishes the deprecated formula to the tap instead of the cask.

3. **Update changelog entries** — every user-facing change needs `.changelog/pr-{N}.txt` (see [Contributing](CONTRIBUTING.md)). The release notes generator collects these.

4. **Tag and push**:
   ```bash
   git tag v0.7.0 && git push origin v0.7.0
   ```
   The workflow runs on tag push. Prerelease tags (e.g. `v0.7.0-rc1`) publish a GitHub prerelease but **skip the Homebrew cask upload** (`skip_upload: auto`).

5. **Verify the release**:
   - GitHub Release published with all binaries and packages
   - `Casks/pingcli-plugin-terraformer.rb` updated in `pingidentity/homebrew-tap`
   - `brew update && brew info pingidentity/tap/pingcli-plugin-terraformer` shows the new version

## Homebrew cask notes

- The cask carries a post-install hook that strips the macOS quarantine xattr (`xattr -dr com.apple.quarantine`), because release binaries are not signed/notarized. If builds ever become notarized, remove the hook.
- If goreleaser ever logs a deprecation for the homebrew section in the release run, address it before the next release — deprecations become breakages without notice.

## History

- The tap previously shipped a **formula** (goreleaser `brews` config, deprecated by goreleaser). PR #164 migrated to `homebrew_casks`; `pingidentity/homebrew-tap#25` added `tap_migrations.json` so existing formula installs migrate to the cask on `brew update`. The stale `Formula/pingcli-plugin-terraformer.rb` in the tap is deleted in a follow-up tap PR after the first cask-publishing release.
