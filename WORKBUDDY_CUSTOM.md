# WorkBuddy integration

The `qoder-custom` branch also carries WorkBuddy integration. Keep `main` aligned with upstream and merge selected stable releases into this branch. Do not replace custom builds with stock images without carrying these changes forward.

- WorkBuddy native plugin `0.9.19-workbuddy.4`: propagate stream errors through the host error channel, reject empty completion streams, and retain upstream model display names with Credits multipliers. Multipliers are informational, not per-request billing formulas; model IDs remain stable.
- [CPA core](https://github.com/libo0118/CLIProxyAPI/tree/qoder-custom): capture the original `usage.credit` JSON number before translation and publish optional `workbuddy_credits` metadata only for WorkBuddy. Missing metadata is not zero. The existing Qoder billing path is unchanged.
- [Keeper](https://github.com/libo0118/cpa-usage-keeper/tree/qoder-custom): query the authenticated WorkBuddy credits endpoint; preserve separate resource packages and expiry times; retain per-request billing in hot and archive tables. Show the upstream amount rounded to two decimals. Original/discounted values and USD are not inferred. Historical records are not backfilled.
- [CPA management panel](https://github.com/libo0118/Cli-Proxy-API-Management-Center/tree/qoder-custom): quota and auth-file cards support WorkBuddy, including per-account refresh, total Credits, and expandable resource packages. Credentials remain server-side.

The deployed model aliases should map `workbuddy/Auto`, `workbuddy/Hy3`, and other provider-prefixed names to IDs returned by the account's own model catalog. Preserve existing explicit mappings. CN resource timestamps without an offset use UTC+08:00; never interpret them in the viewer's local timezone.

Model discovery accepts both nested and flat upstream `supportedEfforts`. Preserve provider-specific levels: WorkBuddy V4.1-Flash currently offers `low/high/max`, while its V4-Flash route offers `low/high/xhigh`. The advertised Hy3 `low` setting is preserved during request preparation instead of being forcibly raised to `high`. No thinking-off selector is added.

Chat requests supply the `X-IDE-Name`, `X-IDE-Type`, and `X-IDE-Version` identity observed in WorkBuddy 5.5.6 (`WorkBuddy`, `WorkBuddy`, `5.5.6`). This populates the official usage history's `client` field for new requests; prior blank records are not rewritten. Adopted CodeBuddy IDE and Intl account headers keep their existing overrides. Login, billing-query, and catalog headers are unchanged. The official desktop's bundled CLI uses the WorkBuddy identity; a separate WorkBuddy CLI billing classification has not been verified.

Before deploying, run plugin tests and the CPA helper/queue/SDK tests, compile the server and native library, run panel verification and Keeper tests/build, and verify the new Keeper image against a database copy. Back up the active images/configuration, native plugin, panel asset and database. Verify both single and bulk quota refresh, Responses SSE/WS, and Credits persistence using minimal requests to an account-supported model.
