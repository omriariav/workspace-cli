# gws Release Plan

Planning snapshot for the post-v1.37.0 backlog. This file is a proposed sequence, not release authorization. Do not start implementation, merge, tag, publish, or close release-scoped issues from this file alone; wait for explicit user or CTO direction.

## Release Requirements

Every release-scoped issue must include matching tests and skill/docs updates when applicable, even if the GitHub issue text does not explicitly mention them.

- Tests should cover the changed command behavior, request construction, output shape, and important error paths.
- Skills and command references should be updated for any new flag, command, output field, workflow, or behavioral caveat.
- If no skill/docs change is needed, the PR should say why in its testing or implementation notes.

## Current Baseline

- Current released version: `v1.37.0`
- Released on: 2026-04-28
- Shipped issues:
  - [#174](https://github.com/omriariav/workspace-cli/issues/174): tell users about newer CLI versions
  - [#175](https://github.com/omriariav/workspace-cli/issues/175): resolve chat senders and flag self

## v1.38.0 - Bug Fixes And Chat Recap

Recommended next release. Prioritizes currently open bug fixes, then adds a focused Chat recap enhancement that builds on v1.37.0 sender attribution.

### [#179](https://github.com/omriariav/workspace-cli/issues/179): Drive resolve-comment fails

Scope:
- Fix `gws drive resolve-comment` and `gws drive unresolve-comment`.
- Use the Drive replies API action path for state transitions: `Replies.Create` with `action: "resolve"` or `action: "reopen"`.
- Keep optional `--content` support as a closing/reopening note if useful.
- Avoid overwriting the original comment content while marking a comment resolved or reopened.
- Update `skills/drive/SKILL.md` for the corrected behavior.

Acceptance:
- Resolve and unresolve commands succeed against the documented Drive API flow.
- Tests assert `POST /files/{fileID}/comments/{commentID}/replies` with the expected action.
- Default behavior remains one-shot for users who only want to change resolved state.
- Original comment content is not replaced by placeholder text.

### [#181](https://github.com/omriariav/workspace-cli/issues/181): Gmail attachment IDs are not discoverable

Scope:
- Expose Gmail attachment metadata from message-reading surfaces.
- Add an `attachments` array to `gws gmail read` and `gws gmail thread` message output.
- Include filename, MIME type, size, and attachment ID.
- Update Gmail skill/docs so users can discover an attachment ID and pass it to `gws gmail attachment`.

Acceptance:
- Messages with attachments expose usable `attachment_id` values in JSON/YAML/text-safe output.
- Existing message output remains backward-compatible except for additive fields.
- `gws gmail attachment --message-id <id> --id <attachment-id>` is reachable from CLI output alone.
- Tests cover single-message and thread output with attachments.

### [#182](https://github.com/omriariav/workspace-cli/issues/182): Chat recent messages across active spaces

Scope:
- Add `gws chat recent` to recap messages across all spaces active within a time window.
- Parse `--since` as durations such as `2h`, `12h`, `7d`, or as an RFC3339 timestamp.
- Use `spaces.list` and `lastActiveTime` as the cheap active-space prefilter.
- Fetch messages only from active spaces with `createTime > since` and `orderBy=createTime DESC`.
- Flatten results, sort globally by newest message first, and include space metadata on each message.
- Include both sent and received messages by default; support optional `--exclude-self`.
- Reuse existing `--resolve-senders` behavior for sender names and self detection.

Acceptance:
- `gws chat recent --since 2h` returns recent messages from spaces active in the last two hours.
- Spaces outside the time window are not queried for messages.
- Output includes `since`, `spaces_scanned`, `active_spaces`, `count`, and message rows with space metadata.
- `--max`, `--max-per-space`, and `--max-spaces` provide safety caps.
- `--exclude-self` omits authenticated-user messages when self detection is available.
- Tests cover duration parsing, active-space filtering, per-space message filtering, sorting, caps, and self exclusion.

## v1.39.0 - Homebrew Distribution

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

## v1.40.0 - OS Keychain Token Storage

### [#112](https://github.com/omriariav/workspace-cli/issues/112): OS keychain token storage

Scope:
- Add an optional keychain-backed token store for macOS and supported Linux environments.
- Preserve compatibility with the existing JSON token file.
- Provide migration or fallback behavior that does not strand existing users.

Acceptance:
- Existing installs keep working without manual migration.
- Keychain storage can be enabled and verified.
- Failure modes fall back or report clear remediation.

## v1.41.0 - Multi-Account Contexts

### [#113](https://github.com/omriariav/workspace-cli/issues/113): Multi-account support with context switching

Scope:
- Add named auth contexts for multiple Google accounts.
- Support context selection by command, config, or environment variable.
- Keep context-aware token/config storage compatible with the v1.40 token-store decision.

Acceptance:
- Users can add, list, use, and remove contexts.
- Commands run against the selected context.
- Existing single-account config remains the default path.

## v1.42.0 - Gmail Settings API

### [#104](https://github.com/omriariav/workspace-cli/issues/104): Gmail settings API

Scope:
- Add Gmail settings commands for vacation responder, filters, forwarding, IMAP/POP, and send-as where supported by the API and scopes.
- Keep the first slice focused on read/list plus the most common writes if the full surface is too large.

Acceptance:
- Commands follow existing `gmail` command patterns.
- Required scopes are documented and covered by auth validation.
- Tests cover request construction and output shape.

## v1.43.0 - Service Account Support

### [#116](https://github.com/omriariav/workspace-cli/issues/116): Service account support with domain-wide delegation

Scope:
- Add service-account authentication for Workspace automation.
- Support subject impersonation for domain-wide delegation.
- Make credential loading and config explicit, with clear safety boundaries.

Acceptance:
- Service-account login/status works separately from user OAuth.
- Commands can run with the delegated subject where APIs support it.
- Errors explain missing delegation, scopes, or admin setup.

## v1.44.0 - Output Filtering

### [#117](https://github.com/omriariav/workspace-cli/issues/117): jq / Go template output filtering

Scope:
- Add structured output filtering flags such as `--jq` and/or `--template`.
- Apply after API response normalization and before final printing.
- Keep behavior consistent across services.

Acceptance:
- Filters work with JSON output and fail clearly on invalid expressions.
- Template output is deterministic and documented.
- Tests cover success and invalid-filter paths.

## v1.45.0 - Cross-Service Batch Operations

### [#123](https://github.com/omriariav/workspace-cli/issues/123): Cross-service batch operations

Scope:
- Add batch workflows for common multi-item operations across supported services.
- Start with narrowly scoped operations that already have stable single-item commands.
- Include dry-run and confirmation controls for destructive operations.

Acceptance:
- Batch operations are scriptable and safe by default.
- Destructive paths support dry-run or explicit confirmation.
- Partial failures are reported in structured output.

## v1.46.0 - Apps Script

### [#122](https://github.com/omriariav/workspace-cli/issues/122): Google Apps Script - list, get, run

Scope:
- Add initial Apps Script service support.
- Include project listing, content inspection, and function invocation if scopes and API enablement allow it.

Acceptance:
- New service follows existing command, client, scope, and test patterns.
- API enablement or permission errors are understandable.

## v1.47.0 - Classroom

### [#121](https://github.com/omriariav/workspace-cli/issues/121): Google Classroom - courses, assignments, submissions

Scope:
- Add initial Classroom service support for courses, assignments, and submissions.
- Keep the first release read-focused unless write operations become clearly required.

Acceptance:
- New service follows existing command, client, scope, and test patterns.
- Outputs are useful for agents and scripts.

## v1.48.0 - Extension System

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
