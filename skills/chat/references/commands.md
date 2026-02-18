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

---

## gws chat get-space

Retrieves details about a Chat space.

```
Usage: gws chat get-space <space>
```

### Output Fields (JSON)

- `name` — Space resource name
- `display_name` — Human-readable space name
- `type` — Space type
- `description` — Space description (if set)
- `create_time` — Space creation timestamp

---

## gws chat create-space

Creates a new Chat space.

```
Usage: gws chat create-space [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--display-name` | string | | Yes | Space display name |
| `--type` | string | SPACE | No | Space type: `SPACE` or `GROUP_CHAT` |
| `--description` | string | | No | Space description |

---

## gws chat delete-space

Deletes a Chat space.

```
Usage: gws chat delete-space <space>
```

---

## gws chat update-space

Updates a Chat space's display name or description.

```
Usage: gws chat update-space <space> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--display-name` | string | | New display name |
| `--description` | string | | New description |

At least one of `--display-name` or `--description` must be provided.

---

## gws chat search-spaces

Searches for Chat spaces using a query.

```
Usage: gws chat search-spaces [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--query` | string | | Yes | Search query |
| `--page-size` | int | 100 | No | Number of results per page |

---

## gws chat find-dm

Finds a direct message space with a specific user.

```
Usage: gws chat find-dm [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--user` | string | | Yes | User resource name (e.g. `users/123` or `users/user@example.com`) |

---

## gws chat setup-space

Creates a space and adds initial members in one call.

```
Usage: gws chat setup-space [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--display-name` | string | | Yes | Space display name |
| `--members` | string | | No | Comma-separated user resource names |

---

## gws chat get-member

Retrieves details about a space member.

```
Usage: gws chat get-member <member-name>
```

The member name format is `spaces/AAAA/members/111` (get from `gws chat members`).

---

## gws chat add-member

Adds a user as a member of a Chat space.

```
Usage: gws chat add-member <space> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--user` | string | | Yes | User resource name (e.g. `users/123`) |
| `--role` | string | ROLE_MEMBER | No | Member role: `ROLE_MEMBER` or `ROLE_MANAGER` |

---

## gws chat remove-member

Removes a member from a Chat space.

```
Usage: gws chat remove-member <member-name>
```

---

## gws chat update-member

Updates a member's role in a Chat space.

```
Usage: gws chat update-member <member-name> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--role` | string | | Yes | New role: `ROLE_MEMBER` or `ROLE_MANAGER` |

---

## gws chat read-state

Gets the read state for a space (when you last read it).

```
Usage: gws chat read-state <space>
```

Space IDs are auto-expanded: `AAAA` → `users/me/spaces/AAAA/spaceReadState`.

### Output Fields (JSON)

- `name` — Read state resource name
- `last_read_time` — When the space was last read (RFC-3339)

---

## gws chat mark-read

Updates the read state for a space to mark it as read.

```
Usage: gws chat mark-read <space> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--time` | string | now | Read time in RFC-3339 format |

---

## gws chat thread-read-state

Gets the read state for a thread.

```
Usage: gws chat thread-read-state <thread>
```

Full resource name required (e.g. `users/me/spaces/AAAA/threads/thread1/threadReadState`).

---

## gws chat attachment

Retrieves metadata for a message attachment.

```
Usage: gws chat attachment <attachment-name>
```

### Output Fields (JSON)

- `name` — Attachment resource name
- `content_name` — Original filename
- `content_type` — MIME type
- `source` — `DRIVE_FILE` or `UPLOADED_CONTENT`
- `download_uri` — Download URL (if available)
- `thumbnail_uri` — Thumbnail URL (if available)

---

## gws chat upload

Uploads a file as an attachment to a Chat space.

```
Usage: gws chat upload <space> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file` | string | | Yes | Path to file to upload |

---

## gws chat download

Downloads a media attachment to a local file.

```
Usage: gws chat download <resource-name> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--output` | string | | Yes | Output file path |

---

## gws chat events

Lists events in a Chat space. The filter flag is required by the API.

```
Usage: gws chat events <space> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--filter` | string | | Yes | Event type filter (e.g. `event_types:"google.workspace.chat.message.v1.created"`) |
| `--page-size` | int | 100 | No | Number of events per page |

---

## gws chat event

Retrieves details about a single space event.

```
Usage: gws chat event <event-name>
```

### Output Fields (JSON)

- `name` — Event resource name
- `event_type` — Event type string
- `event_time` — When the event occurred (RFC-3339)
