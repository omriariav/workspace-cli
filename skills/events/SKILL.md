---
name: gws-events
version: 1.43.1
description: "Google Workspace Events API operations via gws. Use when users need to create or manage event subscriptions for Chat, Drive, or Meet resources (Pub/Sub delivery), renew or reactivate subscriptions, or decode Pub/Sub CloudEvent notifications. Triggers: workspace events, event subscription, subscribe to chat events, drive events, meet events, pubsub notifications, cloudevent."
metadata:
  short-description: Google Workspace Events subscriptions
  compatibility: claude-code, codex-cli
---

# Google Workspace Events (gws events)

`gws events` manages Workspace Events API subscriptions (lifecycle) and decodes
Pub/Sub CloudEvent notifications.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

## Quick Command Reference

| Task | Command |
|------|---------|
| List subscriptions | `gws events subscriptions list --event-type <type> [--target <resource>]` |
| Get a subscription | `gws events subscriptions get <id>` |
| Create a subscription | `gws events subscriptions create --target <resource> --event-types <types> --topic projects/P/topics/T` |
| Update event types / TTL | `gws events subscriptions update <id> --event-types <types> --ttl 86400s` |
| Renew (extend expiration) | `gws events subscriptions renew <id>` (`--ttl 0s` = maximum) |
| Delete | `gws events subscriptions delete <id>` |
| Reactivate a suspended subscription | `gws events subscriptions reactivate <id>` |
| Decode a Pub/Sub notification | `... | gws events decode` (stdin: push envelope or bare message JSON) |

## Output Modes

All commands support `--format json` (default), `--format yaml`, and
`--format text`, plus `--quiet` to suppress output.

## Supported targets (GA only)

| Target | Format | Notes |
|--------|--------|-------|
| Chat space | `//chat.googleapis.com/spaces/SPACE` | user auth, or admin-approved app auth |
| All the user's Chat spaces | `//chat.googleapis.com/spaces/-` | user auth only |
| User (Chat or Meet user events) | `//cloudidentity.googleapis.com/users/USER` | user auth only; Chat membership + read-state, or Meet events |
| Drive file | `//drive.googleapis.com/files/FILE` | file + permission events |
| Shared drive | `//drive.googleapis.com/drives/DRIVE` | file + permission events |
| Meet space | `//meet.googleapis.com/spaces/SPACE` | Meet events |

Customer-wide Chat targets (`//admin.googleapis.com/customers/...`) are
Developer Preview and rejected.

## Supported event types (GA only)

- **Chat:** `google.workspace.chat.{message,reaction,membership,space}.v1.*`
  (created/updated/deleted as applicable) plus
  `google.workspace.chat.spaceReadState.v1.updated` and
  `google.workspace.chat.threadReadState.v1.updated` (user targets, user auth).
- **Drive (baseline):** `google.workspace.drive.file.v3.{added,moved,contentChanged,deleted,renamed,trashed,untrashed}`
  and `google.workspace.drive.permission.v3.{created,edited,deleted}`.
- **Meet:** `google.workspace.meet.{conference,participant,recording,transcript,smartNote}.v2.*`.

Excluded (Developer Preview or unsupported): Drive comment/reply/approval/
access-proposal events, Chat availability events, customer-wide Chat events,
and every `chat.app.all.*` scope. Chat `batch*` events are output-only —
they arrive automatically once the base event is subscribed and can be
decoded, but cannot be requested at creation.

## Pub/Sub topic constraint

The notification topic must **already exist** in the same Cloud project as
the subscription and is immutable: `gws` validates the
`projects/PROJECT/topics/TOPIC` format and never creates, replaces, or
modifies topics. Grant the Workspace Events service publish access to the
topic in Cloud Console.

## Auth modes and scopes

Credentials are minted with the least-privilege documented scopes for the
subscription's event classes (e.g. `chat.messages.readonly` for message
events, `drive.metadata` for Drive events, `meetings.space.readonly` — or
`meetings.space.created` with `--meet-created-scope` — for Meet).

- **OAuth (default):** run `gws auth login` (the `events` service grants the
  subscription scopes).
- **`service-account-dwd`:** acts as the impersonated Workspace user; same
  targets as OAuth.
- **`service-account-app`:** admin-approved Chat app; **individual Chat
  space subscriptions only**. Read-state and non-Chat targets are rejected
  (their app-wide variants are Preview). Read/delete lifecycle ops use
  `chat.bot`; mutations mint only the granular `chat.app.*.readonly`
  scope(s) matching the subscription's event classes (derived from the
  existing subscription via a `chat.bot` Get when not passed explicitly).

See the auth skill for configuring the modes.

## Renewal and payloads

Subscriptions expire. `--include-resource-data` payloads have **shorter**
maximum expirations — renew explicitly:

```bash
gws events subscriptions renew SUBSCRIPTION_ID          # maximum allowed
gws events subscriptions renew SUBSCRIPTION_ID --ttl 86400s
```

## Decoding notifications

```bash
gcloud pubsub subscriptions pull my-sub --format=json | jq '.[0].message' | gws events decode
```

Output fields: `event_type` (ce-type), `subscription` (ce-source — the
Workspace Events subscription), `target` (ce-subject — the changed
resource), `attributes`, `message_id`, `publish_time`, and the
base64-decoded `data` payload.

## Tips for AI Agents

- The list command requires an event-type filter — a target alone is
  rejected locally before the API is called.
- Always pass the topic as `projects/PROJECT/topics/TOPIC` and confirm it
  already exists; `gws` will not create it.
- Prefer the default least-privilege scopes; only pass
  `--meet-created-scope` when the subscription covers Meet spaces the
  authenticated user created.
- Renew before expiry: subscriptions with `--include-resource-data` expire
  sooner. `renew` with the default `--ttl 0s` requests the maximum.
- In `service-account-app` mode, stick to individual Chat space targets;
  read-state and non-Chat targets need user credentials (OAuth or DWD).
- Pipe raw Pub/Sub JSON straight into `gws events decode` — it accepts both
  push envelopes and bare messages and base64-decodes the payload for you.
