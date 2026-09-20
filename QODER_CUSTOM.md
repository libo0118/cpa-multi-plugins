# Qoder custom branch

`qoder-custom` preserves the Qoder integration used by the matching custom CPA management panel and Keeper forks:

- Mixed text/image content and conversation fields survive request conversion.
- International model discovery has a region-aware fallback catalog.
- The Credits endpoint exposes Teams/base, dedicated/SOTA and shared pools, decimals, availability, expiry, and unknown capacity.
- Nested upstream errors are surfaced through the host RPC error field; empty streams cannot silently become successful completions.

These changes do not grant model entitlements or implement organization identity support. Routing aliases remain CPA configuration (`oauth-model-alias.qoder`). Model display names can include upstream catalog multipliers; remove redundant `display-name` overrides equal to the alias if they hide these labels.

Merged upstream stable baseline: `v0.12.64`; Qoder custom build version: `0.8.13-qoder.4` (inject with `-ldflags "-X main.version=0.8.13-qoder.4"`). The upstream message passthrough supersedes the earlier custom image parser, while the mixed-content regression test remains. Upstream null-content, tool, reasoning, large-input and account-status handling are retained.

## Catalog multipliers

The model-list GET request signs its actual empty body. Signing an encoded JSON object while sending no body caused `403 Signature invalid` and silently forced the fallback catalog. The chat-scene `price_factor` and optional `original_price_factor` now populate display labels, including valid zero-rate offers and original-to-current comparisons. Missing factors remain unknown; they are never treated as zero or used to estimate request charges.

Catalog caches are scoped to credentials. The existing international compatibility IDs remain when omitted by the catalog, without invented multipliers. Explicitly disabled entries in a valid catalog are not reintroduced by this merge. Current model IDs and upstream routing remain separate from display metadata. Unit tests cover the scene, zero/missing values, discount display, compatibility IDs and credential cache isolation; no model calls are required to refresh metadata.

## Updating

The chat catalog's `thinking_config.enabled.efforts` supplies per-model `Thinking.Levels` to CPA and Codex. For example, DeepSeek-Flash advertises `low/high/max`, while Qwen3.8-Max advertises `low/medium/xhigh`. Missing effort metadata is not filled from another model family. Request conversion now preserves `high` and `max` alongside the previously supported levels, without conflating them with `xhigh`. No new thinking-off control is exposed by this change.

Keep `upstream` pointed at `mmqz/cpa-multi-plugins` and `origin` at this fork. Fetch official stable tags, switch to `qoder-custom`, then merge the selected stable tag. Resolve conflicts semantically; keep upstream fixes instead of replacing whole files with old copies.

Run `go test ./...` in `plugins/qoder`, then build the shared library for the target CPU and libc. Check the library in an isolated container using the deployed CPA version before installing it. Back up the existing library, replace it atomically, restart CPA and verify the management Credits endpoint. Model-inference checks require available account quota and are not part of unit tests.

Keep the forks coordinated: this plugin supplies quota data, the management fork displays it on both pages, and the Keeper fork consumes it for quota refresh. Per-request billing additionally requires the [CPA core custom branch](https://github.com/libo0118/CLIProxyAPI/tree/qoder-custom), which carries the existing upstream usage fields to Keeper. Official plugin updates overwrite custom binaries; deploy the tested custom build instead. Never publish server configuration, tokens, private keys or runtime auth files.
