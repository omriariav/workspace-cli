# gws events — Command Reference

Workspace Events API subscription lifecycle and CloudEvent decoding.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global flags

All commands accept the standard `gws` global flags:

| Flag | Description |
|------|-------------|
| `--format <fmt>` | Output format: `json`, `yaml`, or `text` (default `json`) |
| `--quiet` | Suppress output |
| `--config <path>` | Alternate config file |

Subscription IDs may be passed bare (`abc123`) or as the full resource name
(`subscriptions/abc123`).

## gws events subscriptions list

Lists subscriptions. The API **requires** at least one `event_types` clause,
so `--event-type` (or a raw `--filter` containing `event_types`) is
mandatory; `--target` is an optional additional AND clause.

| Flag | Description |
|------|-------------|
| `--event-type <type>` | Filter by event type (required unless `--filter` is given) |
| `--target <resource>` | Additionally filter by target resource |
| `--filter <expr>` | Raw filter expression; must contain an `event_types` clause |
| `--max <n>` | Maximum subscriptions to return (default 50) |

Output: `{"count": N, "subscriptions": [...]}` with full subscription
objects (name, target, event types, state, expire time, etc.).

```bash
gws events subscriptions list --event-type google.workspace.chat.message.v1.created
gws events subscriptions list \
  --event-type google.workspace.chat.message.v1.created \
  --target //chat.googleapis.com/spaces/AAA
```

## gws events subscriptions get

```bash
gws events subscriptions get SUBSCRIPTION_ID
```

Output: the subscription object, including `state`
(`ACTIVE`/`SUSPENDED`/`DELETED`), `suspensionReason`, `expireTime`,
`targetResource`, `eventTypes`, and the Pub/Sub `notificationEndpoint`.

## gws events subscriptions create

| Flag | Description |
|------|-------------|
| `--target <resource>` | Target resource (required; see targets below) |
| `--event-types <t1,t2>` | Comma-separated GA event types (required) |
| `--topic projects/P/topics/T` | Existing Pub/Sub topic (required; validated, never created) |
| `--include-resource-data` | Include the changed resource in payloads (shorter max expiration) |
| `--ttl <dur>` | TTL, e.g. `86400s`; empty/`0s` = maximum allowed |
| `--meet-created-scope` | Use `meetings.space.created` instead of `meetings.space.readonly` |

Output: `{"status":"created","operation":{...},"scopes":[...]}` — the
long-running operation and the least-privilege scopes that were used.

```bash
gws events subscriptions create \
  --target //chat.googleapis.com/spaces/AAA \
  --event-types google.workspace.chat.message.v1.created \
  --topic projects/my-project/topics/chat-events
```

## gws events subscriptions update / renew

`update` changes event types and/or TTL (at least one required):

| Flag | Description |
|------|-------------|
| `--event-types <t1,t2>` | Replace the event-type list |
| `--ttl <dur>` | New TTL; `0s` = maximum allowed |
| `--meet-created-scope` | Narrower Meet scope (see create) |

`renew` is the TTL-only shortcut (`--ttl` defaults to `0s` = maximum).
Subscriptions with resource data in payloads expire sooner — renew them
explicitly.

Both first read the existing subscription to derive its exact event
classes, then mint only the matching scopes (in app mode the read uses
`chat.bot` and the mutation uses the granular `chat.app.*` scopes).

```bash
gws events subscriptions renew SUBSCRIPTION_ID
gws events subscriptions update SUBSCRIPTION_ID --ttl 86400s
```

Output: `{"status":"updated","operation":{...},"scopes":[...]}`.

## gws events subscriptions delete

```bash
gws events subscriptions delete SUBSCRIPTION_ID
```

Output: `{"status":"deleted","operation":{...}}`.

## gws events subscriptions reactivate

Reactivates a `SUSPENDED` subscription after fixing the suspension reason
(for example restoring the Pub/Sub topic permission).

```bash
gws events subscriptions reactivate SUBSCRIPTION_ID
```

Output: `{"status":"reactivated","operation":{...},"scopes":[...]}`.

## gws events decode

Reads a Pub/Sub message JSON from stdin — either a push-delivery envelope
(`{"message": {...}, "subscription": "..."}`) or a bare message object —
and decodes the CloudEvent attributes plus the base64 `data` payload.

```bash
gcloud pubsub subscriptions pull my-sub --format=json | jq '.[0].message' | gws events decode
cat push-body.json | gws events decode
```

Output fields:

| Field | Source |
|-------|--------|
| `event_type` | `ce-type` attribute |
| `subscription` | `ce-source` attribute (the Workspace Events subscription) |
| `target` | `ce-subject` attribute (the changed target resource) |
| `time` | `ce-time` attribute |
| `attributes` | All Pub/Sub attributes |
| `message_id`, `publish_time` | Pub/Sub message metadata |
| `pubsub_subscription` | Push envelope `subscription` (the Pub/Sub subscription) |
| `data` | Base64-decoded event payload (JSON passthrough; non-JSON re-encoded as a JSON string) |

Batch event types (`*.batchCreated` etc.) are output-only: decode accepts
them, but they cannot be requested at subscription creation.

## Accepted targets (GA)

| Target | Event classes |
|--------|---------------|
| `//chat.googleapis.com/spaces/SPACE` | Chat space/membership/message/reaction |
| `//chat.googleapis.com/spaces/-` (user auth only) | same, across the user's spaces |
| `//cloudidentity.googleapis.com/users/USER` | Chat membership, spaceReadState/threadReadState, or Meet events |
| `//drive.googleapis.com/files/FILE` | Drive file + permission |
| `//drive.googleapis.com/drives/DRIVE` | Drive file + permission |
| `//meet.googleapis.com/spaces/SPACE` | Meet events |

Rejected: customer-wide Chat targets (`//admin.googleapis.com/customers/…`,
Developer Preview), prefix-only targets with an empty resource ID.

## Accepted event types (GA)

- Chat: `google.workspace.chat.message.v1.{created,updated,deleted}`,
  `google.workspace.chat.reaction.v1.{created,deleted}`,
  `google.workspace.chat.membership.v1.{created,updated,deleted}`,
  `google.workspace.chat.space.v1.{updated,deleted}`,
  `google.workspace.chat.spaceReadState.v1.updated`,
  `google.workspace.chat.threadReadState.v1.updated`
- Drive: `google.workspace.drive.file.v3.{added,moved,contentChanged,deleted,renamed,trashed,untrashed}`,
  `google.workspace.drive.permission.v3.{created,edited,deleted}`
- Meet: `google.workspace.meet.conference.v2.{started,ended}`,
  `google.workspace.meet.participant.v2.{joined,left}`,
  `google.workspace.meet.{recording,transcript,smartNote}.v2.{started,ended,fileGenerated}`

Rejected with explanatory errors: Drive comment/reply/approval/
access-proposal events, Chat availability events (all Developer Preview or
unsupported), and Chat batch events (output-only).

## Pub/Sub topic constraint

`--topic` must name an **existing** topic in the same Cloud project as the
subscription, in `projects/PROJECT/topics/TOPIC` form. `gws` validates the
format and never creates, replaces, or modifies topics. The Workspace
Events service account needs publish permission on the topic.

## Auth modes and scopes

| Mode | Behavior |
|------|----------|
| OAuth (default) | Stored user token; grant subscription scopes via `gws auth login --services events,...` |
| `service-account-dwd` | Acts as the impersonated user; scopes minted per subscription class |
| `service-account-app` | Individual Chat space subscriptions only; read/delete ops use `chat.bot`, mutations mint granular `chat.app.*.readonly` scopes; read-state rejected (app-wide variant is Preview) |

Least-privilege user-auth scopes per class: `chat.spaces.readonly`,
`chat.memberships.readonly`, `chat.messages.readonly`,
`chat.messages.reactions.readonly`, `chat.users.readstate.readonly`,
`drive.metadata` (the readonly variant is not supported for subscriptions),
`meetings.space.readonly` or `meetings.space.created`.
