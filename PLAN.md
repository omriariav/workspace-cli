# gws Release Plan

Planning snapshot for the post-v1.36.0 backlog. This file is a proposed sequence, not release authorization. Do not start implementation, merge, tag, publish, or close release-scoped issues from this file alone; wait for explicit user or CTO direction.

## Current Baseline

- Current released version: `v1.36.0`
- Released on: 2026-04-28
- Shipped issues:
  - [#170](https://github.com/omriariav/workspace-cli/issues/170): `chat find-space --name` via the local space cache
  - [#171](https://github.com/omriariav/workspace-cli/issues/171): chat attachment metadata in message output
  - [#176](https://github.com/omriariav/workspace-cli/issues/176): calendar create adds the authenticated user as an accepted attendee by default

## v1.37.0 - Update Notice And Chat Attribution

Recommended next release. Small enough for one PR, with two user-facing quality-of-life fixes.

### [#174](https://github.com/omriariav/workspace-cli/issues/174): Tell users about newer CLI versions

Scope:
- Add a version freshness check against GitHub releases.
- Prefer a manual command path such as `gws version --check`, plus a low-noise passive notice when the installed version is stale.
- Cache the latest-version result so normal CLI usage does not call the network on every invocation.
- Print passive notices to stderr and respect script-friendly modes such as `--quiet`.
- Treat network failures as non-fatal for normal commands.

Acceptance:
- Current version reports no update available.
- Older version reports the latest release and a clear upgrade hint.
- `--quiet` suppresses passive notices.
- Network failure does not break unrelated commands.
- Unit tests cover version comparison, cache behavior, quiet suppression, and failure handling.

### [#175](https://github.com/omriariav/workspace-cli/issues/175): Resolve chat senders and flag self

Scope:
- Improve sender attribution on chat message surfaces where sender data appears.
- Add a `self` marker when the sender can be identified as the authenticated user.
- Add display-name resolution behind an explicit flag, likely `--resolve-senders`, to avoid surprise API cost.
- Apply consistently to `chat messages`, `chat get`, and `chat unread` where those outputs include message sender data.
- Resolve sender display names once per space per invocation and expose unresolved senders predictably.

Acceptance:
- Default output remains fast and backward-compatible except for additive fields.
- `--resolve-senders` adds display names for resolvable members.
- Self messages are marked consistently.
- Unresolvable senders do not fail the whole command.
- Tests cover resolved, unresolved, self, and non-self senders.

## v1.38.0 - Homebrew Distribution

### [#115](https://github.com/omriariav/workspace-cli/issues/115): Homebrew distribution

Scope:
- Publish `gws` via a Homebrew tap.
- Start with a formula that consumes the existing GitHub release binaries and checksums.
- Add install/upgrade instructions to README and release notes.
- Update release process documentation for tap updates.
- Evaluate GoReleaser only if the current Makefile release flow becomes a bottleneck.

Acceptance:
- `brew tap omriariav/tap` and `brew install gws` work on macOS.
- Formula points at published release assets and verifies checksums.
- Release process clearly states how the formula is updated.

## v1.39.0 - OS Keychain Token Storage

### [#112](https://github.com/omriariav/workspace-cli/issues/112): OS keychain token storage

Scope:
- Add an optional keychain-backed token store for macOS and supported Linux environments.
- Preserve compatibility with the existing JSON token file.
- Provide migration or fallback behavior that does not strand existing users.

Acceptance:
- Existing installs keep working without manual migration.
- Keychain storage can be enabled and verified.
- Failure modes fall back or report clear remediation.

## v1.40.0 - Multi-Account Contexts

### [#113](https://github.com/omriariav/workspace-cli/issues/113): Multi-account support with context switching

Scope:
- Add named auth contexts for multiple Google accounts.
- Support context selection by command, config, or environment variable.
- Keep context-aware token/config storage compatible with the v1.39 token-store decision.

Acceptance:
- Users can add, list, use, and remove contexts.
- Commands run against the selected context.
- Existing single-account config remains the default path.

## v1.41.0 - Gmail Settings API

### [#104](https://github.com/omriariav/workspace-cli/issues/104): Gmail settings API

Scope:
- Add Gmail settings commands for vacation responder, filters, forwarding, IMAP/POP, and send-as where supported by the API and scopes.
- Keep the first slice focused on read/list plus the most common writes if the full surface is too large.

Acceptance:
- Commands follow existing `gmail` command patterns.
- Required scopes are documented and covered by auth validation.
- Tests cover request construction and output shape.

## v1.42.0 - Service Account Support

### [#116](https://github.com/omriariav/workspace-cli/issues/116): Service account support with domain-wide delegation

Scope:
- Add service-account authentication for Workspace automation.
- Support subject impersonation for domain-wide delegation.
- Make credential loading and config explicit, with clear safety boundaries.

Acceptance:
- Service-account login/status works separately from user OAuth.
- Commands can run with the delegated subject where APIs support it.
- Errors explain missing delegation, scopes, or admin setup.

## v1.43.0 - Output Filtering

### [#117](https://github.com/omriariav/workspace-cli/issues/117): jq / Go template output filtering

Scope:
- Add structured output filtering flags such as `--jq` and/or `--template`.
- Apply after API response normalization and before final printing.
- Keep behavior consistent across services.

Acceptance:
- Filters work with JSON output and fail clearly on invalid expressions.
- Template output is deterministic and documented.
- Tests cover success and invalid-filter paths.

## v1.44.0 - Cross-Service Batch Operations

### [#123](https://github.com/omriariav/workspace-cli/issues/123): Cross-service batch operations

Scope:
- Add batch workflows for common multi-item operations across supported services.
- Start with narrowly scoped operations that already have stable single-item commands.
- Include dry-run and confirmation controls for destructive operations.

Acceptance:
- Batch operations are scriptable and safe by default.
- Destructive paths support dry-run or explicit confirmation.
- Partial failures are reported in structured output.

## v1.45.0 - Apps Script

### [#122](https://github.com/omriariav/workspace-cli/issues/122): Google Apps Script - list, get, run

Scope:
- Add initial Apps Script service support.
- Include project listing, content inspection, and function invocation if scopes and API enablement allow it.

Acceptance:
- New service follows existing command, client, scope, and test patterns.
- API enablement or permission errors are understandable.

## v1.46.0 - Classroom

### [#121](https://github.com/omriariav/workspace-cli/issues/121): Google Classroom - courses, assignments, submissions

Scope:
- Add initial Classroom service support for courses, assignments, and submissions.
- Keep the first release read-focused unless write operations become clearly required.

Acceptance:
- New service follows existing command, client, scope, and test patterns.
- Outputs are useful for agents and scripts.

## v1.47.0 - Extension System

### [#118](https://github.com/omriariav/workspace-cli/issues/118): Extension / plugin system

Scope:
- Design and implement a minimal extension mechanism for custom commands.
- Treat this as a larger architecture release because it affects command discovery, trust, install paths, and execution boundaries.

Acceptance:
- Extension install/list/remove flows are explicit.
- Execution model is documented and constrained.
- Core commands remain stable and unaffected.

## Closed Or Deferred Items

- [#114](https://github.com/omriariav/workspace-cli/issues/114): scoped auth is already addressed by `gws auth login --services`.
- [#164](https://github.com/omriariav/workspace-cli/issues/164): quiet flag enforcement shipped in v1.35.0.
- [#172](https://github.com/omriariav/workspace-cli/issues/172): docs replace-content already exists.
