# Qoder custom branch

`qoder-custom` preserves the Qoder integration used by the matching custom CPA management panel and Keeper forks:

- Mixed text/image content and conversation fields survive request conversion.
- International model discovery has a region-aware fallback catalog.
- The Credits endpoint exposes Teams/base, dedicated/SOTA and shared pools, decimals, availability, expiry, and unknown capacity.

These changes do not grant model entitlements or implement organization identity support. Display names and aliases remain CPA configuration (`oauth-model-alias.qoder`), separate from plugin code.

Merged upstream stable baseline: `v0.12.64`; Qoder build version: `0.8.13-qoder.1`. The upstream message passthrough supersedes the earlier custom image parser, while the mixed-content regression test remains. Upstream null-content, tool, reasoning and large-input handling are retained.

## Updating

Keep `upstream` pointed at `mmqz/cpa-multi-plugins` and `origin` at this fork. Fetch official stable tags, switch to `qoder-custom`, then merge the selected stable tag. Resolve conflicts semantically; keep upstream fixes instead of replacing whole files with old copies.

Run `go test ./...` in `plugins/qoder`, then build the shared library for the target CPU and libc. Check the library in an isolated container using the deployed CPA version before installing it. Back up the existing library, replace it atomically, restart CPA and verify the management Credits endpoint. Model-inference checks require available account quota and are not part of unit tests.

Keep the three forks coordinated: this plugin supplies Credits data, the management fork displays it on both pages, and the Keeper fork consumes it for quota refresh. Official plugin updates overwrite custom binaries; deploy the tested custom build instead. Never publish server configuration, tokens, private keys or runtime auth files.
