# Chat Commands Reference

Complete flag and option reference for `gws chat` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

## Prerequisites

Google Chat API requires additional setup beyond standard OAuth:

1. Enable the Chat API in your Google Cloud project
2. Configure the OAuth consent screen for Chat scopes
3. For some operations, you may need a service account with domain-wide delegation

---

## gws chat list

Lists all Chat spaces (rooms, DMs, group chats) you have access to. Supports filtering and pagination.

```
Usage: gws chat list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--filter` | string | | Filter spaces (e.g. `spaceType = "SPACE"`) |
| `--page-size` | int | 100 | Number of spaces per page |

### Output Fields (JSON)

Each space includes:
- `name` — Space resource name (e.g., `spaces/AAAA1234`)
- `displayName` — Human-readable space name
- `type` — Space type (`ROOM`, `DM`, `GROUP_CHAT`)

---

## gws chat messages

Lists recent messages in a Chat space. Supports filtering, ordering, pagination, and showing deleted messages.

```
Usage: gws chat messages <space-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 25 | Maximum number of messages to return |
| `--filter` | string | | Filter messages (e.g. `createTime > "2024-01-01T00:00:00Z"`) |
| `--order-by` | string | | Order messages (e.g. `createTime DESC`) |
| `--show-deleted` | bool | false | Include deleted messages in results |

The space ID format is `spaces/AAAA1234` (get from `gws chat list`).

---

## gws chat members

Lists all members of a Chat space with display names and emails (auto-resolved via People API, cached locally).

```
Usage: gws chat members <space-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 100 | Maximum number of members to return |
| `--filter` | string | | Filter members (e.g. `member.type = "HUMAN"`) |
| `--show-groups` | bool | false | Include Google Group memberships |
| `--show-invited` | bool | false | Include invited memberships |

The space ID format is `spaces/AAAA1234` (get from `gws chat list`).

Requires the `chat.memberships.readonly` scope (included by default since v1.16.0).

### Output Fields (JSON)

Each member includes (optional fields omitted when empty):
- `name` — Membership resource name (e.g., `spaces/AAAA/members/111`)
- `role` — `ROLE_MEMBER` or `ROLE_MANAGER`
- `display_name` — Member's display name (resolved via People API, cached at `~/.config/gws/user-cache.json`)
- `email` — Member's email address (resolved via People API, if available)
- `user` — User resource name, e.g., `users/123456789` (if available)
- `type` — User type: `HUMAN` or `BOT` (if available)
- `joined` — Membership creation timestamp (if available)

### Name Resolution

Display names and emails are auto-resolved via the Google People API (`people.getBatchGet`) in batches of up to 50. Results are cached persistently — the first call to a new space may be slower, but subsequent calls use the local cache. BOT members are skipped (not sent to People API).

---

## gws chat send

Sends a text message to a Chat space.

```
Usage: gws chat send [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--space` | string | | Yes | Space ID or name |
| `--text` | string | | Yes | Message text |

---

## gws chat get

Retrieves a single message by its resource name.

```
Usage: gws chat get <message-name>
```

The message name format is `spaces/AAAA/messages/msg1` (get from `gws chat messages`).

### Output Fields (JSON)

- `name` — Message resource name
- `text` — Message text content
- `create_time` — Message creation timestamp
- `sender` — Sender display name (falls back to resource name)
- `sender_type` — `HUMAN` or `BOT`
- `thread` — Thread resource name (if part of a thread)

---

## gws chat update

Updates the text of an existing message.

```
Usage: gws chat update <message-name> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--text` | string | | Yes | New message text |

---

## gws chat delete

Deletes a message by its resource name.

```
Usage: gws chat delete <message-name> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | false | Force delete even if message has replies |

---

## gws chat reactions

Lists all reactions on a message.

```
Usage: gws chat reactions <message-name> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--filter` | string | | Filter reactions (e.g. `emoji.unicode = "😀"`) |
| `--page-size` | int | 25 | Number of reactions per page |

### Output Fields (JSON)

Each reaction includes:
- `name` — Reaction resource name
- `emoji` — Emoji unicode character
- `user` — User display name who reacted

---

## gws chat react

Adds an emoji reaction to a message.

```
Usage: gws chat react <message-name> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--emoji` | string | | Yes | Emoji unicode character (e.g. `😀`) |

---

## gws chat unreact

Removes a reaction by its resource name.

```
Usage: gws chat unreact <reaction-name>
```

The reaction name format is `spaces/AAAA/messages/msg1/reactions/rxn1` (get from `gws chat reactions`).
