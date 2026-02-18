---
name: gws-chat
version: 1.3.0
description: "Google Chat CLI operations via gws. Use when users need to list chat spaces, read messages, send/update/delete messages, or manage reactions in Google Chat. Triggers: google chat, gchat, chat spaces, chat messages."
metadata:
  short-description: Google Chat CLI operations
  compatibility: claude-code, codex-cli
---

# Google Chat (gws chat)

`gws chat` provides CLI access to Google Chat with structured JSON output.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

## Authentication

Requires OAuth2 credentials. Run `gws auth status` to check.
If not authenticated: `gws auth login` (opens browser for OAuth consent).
For initial setup, see the `gws-auth` skill.

**Note:** Google Chat API requires additional setup:
1. Enable the Chat API in your Google Cloud project
2. Configure the OAuth consent screen for Chat scopes
3. For some operations, you may need a service account with domain-wide delegation

## Quick Command Reference

| Task | Command |
|------|---------|
| List chat spaces | `gws chat list` |
| List spaces (filtered) | `gws chat list --filter 'spaceType = "SPACE"'` |
| Read messages | `gws chat messages <space-id>` |
| Read recent messages (ordered) | `gws chat messages <space-id> --order-by "createTime DESC" --max 10` |
| List space members | `gws chat members <space-id>` |
| Send a message | `gws chat send --space <space-id> --text "Hello"` |
| Get a single message | `gws chat get <message-name>` |
| Update a message | `gws chat update <message-name> --text "New text"` |
| Delete a message | `gws chat delete <message-name>` |
| List reactions | `gws chat reactions <message-name>` |
| Add a reaction | `gws chat react <message-name> --emoji "👍"` |
| Remove a reaction | `gws chat unreact <reaction-name>` |

## Detailed Usage

### list — List chat spaces

```bash
gws chat list [flags]
```

Lists all Chat spaces (rooms, DMs, group chats) you have access to. Supports filtering and pagination.

**Flags:**
- `--filter string` — Filter spaces (e.g. `spaceType = "SPACE"`)
- `--page-size int` — Number of spaces per page (default 100)

### messages — List messages in a space

```bash
gws chat messages <space-id> [flags]
```

**Flags:**
- `--max int` — Maximum number of messages to return (default 25)
- `--filter string` — Filter messages (e.g. `createTime > "2024-01-01T00:00:00Z"`)
- `--order-by string` — Order messages (e.g. `createTime DESC`)
- `--show-deleted` — Include deleted messages in results

### members — List space members

```bash
gws chat members <space-id> [flags]
```

Lists all members of a Chat space with display names, emails, roles, and user types.

Display names and emails are auto-resolved via the People API and cached locally at `~/.config/gws/user-cache.json`. The cache grows over time, avoiding repeat API calls.

**Flags:**
- `--max int` — Maximum number of members to return (default 100)
- `--filter string` — Filter members (e.g. `member.type = "HUMAN"`)
- `--show-groups` — Include Google Group memberships
- `--show-invited` — Include invited memberships

**Output includes:**
- `display_name` — Member's display name (resolved via People API)
- `email` — Member's email address (resolved via People API, if available)
- `user` — User resource name (e.g., `users/123456789`)
- `type` — `HUMAN` or `BOT`
- `role` — `ROLE_MEMBER` or `ROLE_MANAGER`
- `joined` — When the member joined the space

### send — Send a message

```bash
gws chat send --space <space-id> --text <message>
```

**Flags:**
- `--space string` — Space ID or name (required)
- `--text string` — Message text (required)

### get — Get a single message

```bash
gws chat get <message-name>
```

Retrieves a single message by its resource name (e.g. `spaces/AAAA/messages/msg1`).

### update — Update a message

```bash
gws chat update <message-name> --text "New text"
```

**Flags:**
- `--text string` — New message text (required)

### delete — Delete a message

```bash
gws chat delete <message-name> [flags]
```

**Flags:**
- `--force` — Force delete even if message has replies

### reactions — List reactions on a message

```bash
gws chat reactions <message-name> [flags]
```

**Flags:**
- `--filter string` — Filter reactions (e.g. `emoji.unicode = "😀"`)
- `--page-size int` — Number of reactions per page (default 25)

### react — Add a reaction

```bash
gws chat react <message-name> --emoji "👍"
```

**Flags:**
- `--emoji string` — Emoji unicode character (required)

### unreact — Remove a reaction

```bash
gws chat unreact <reaction-name>
```

Removes a reaction by its resource name (e.g. `spaces/AAAA/messages/msg1/reactions/rxn1`).

## Output Modes

```bash
gws chat list --format json    # Structured JSON (default)
gws chat list --format yaml    # YAML format
gws chat list --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws chat list` first to get space IDs
- Space IDs are in the format `spaces/AAAA1234`
- Message names are in the format `spaces/AAAA/messages/msg1`
- `members` auto-resolves display names via People API — first call may be slower, subsequent calls use cache
- Use `--order-by "createTime DESC"` with messages to get newest first
- Chat API requires additional GCP setup beyond standard OAuth — see the `gws-auth` skill
