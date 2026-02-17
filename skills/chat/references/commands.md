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

Lists all Chat spaces (rooms, DMs, group chats) you have access to.

```
Usage: gws chat list
```

No additional flags.

### Output Fields (JSON)

Each space includes:
- `name` — Space resource name (e.g., `spaces/AAAA1234`)
- `displayName` — Human-readable space name
- `type` — Space type (`ROOM`, `DM`, `GROUP_CHAT`)

---

## gws chat messages

Lists recent messages in a Chat space.

```
Usage: gws chat messages <space-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 25 | Maximum number of messages to return |

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
