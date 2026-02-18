---
name: gws-chat
version: 2.0.0
description: "Google Chat CLI operations via gws. Use when users need to list/create/manage chat spaces, read/send messages, manage members, track read state, handle attachments, or monitor space events. Triggers: google chat, gchat, chat spaces, chat messages."
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
| **Spaces** | |
| List chat spaces | `gws chat list` |
| List spaces (filtered) | `gws chat list --filter 'spaceType = "SPACE"'` |
| Get space details | `gws chat get-space <space-id>` |
| Create a space | `gws chat create-space --display-name "Team" --type SPACE` |
| Delete a space | `gws chat delete-space <space-id>` |
| Update a space | `gws chat update-space <space-id> --display-name "New Name"` |
| Search spaces (admin only) | `gws chat search-spaces --query "Engineering"` |
| Find DM with user | `gws chat find-dm --user users/123` |
| Create space + members | `gws chat setup-space --display-name "Team" --members "users/1,users/2"` |
| **Messages** | |
| Read messages | `gws chat messages <space-id>` |
| Read recent messages | `gws chat messages <space-id> --order-by "createTime DESC" --max 10` |
| Send a message | `gws chat send --space <space-id> --text "Hello"` |
| Get a single message | `gws chat get <message-name>` |
| Update a message | `gws chat update <message-name> --text "New text"` |
| Delete a message | `gws chat delete <message-name>` |
| **Members** | |
| List space members | `gws chat members <space-id>` |
| Get member details | `gws chat get-member <member-name>` |
| Add a member | `gws chat add-member <space-id> --user users/123` |
| Remove a member | `gws chat remove-member <member-name>` |
| Update member role | `gws chat update-member <member-name> --role ROLE_MANAGER` |
| **Reactions** | |
| List reactions | `gws chat reactions <message-name>` |
| Add a reaction | `gws chat react <message-name> --emoji "👍"` |
| Remove a reaction | `gws chat unreact <reaction-name>` |
| **Read State** | |
| Get read state | `gws chat read-state <space-id>` |
| Mark space as read | `gws chat mark-read <space-id>` |
| Get thread read state | `gws chat thread-read-state <thread-name>` |
| **Attachments & Media** | |
| Get attachment info | `gws chat attachment <attachment-name>` |
| Upload a file | `gws chat upload <space-id> --file ./report.pdf` |
| Download media | `gws chat download <resource-name> --output ./file.pdf` |
| **Events** | |
| List space events | `gws chat events <space-id> --filter 'event_types:"google.workspace.chat.message.v1.created"'` |
| Get event details | `gws chat event <event-name>` |

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

### get-space — Get space details

```bash
gws chat get-space <space>
```

Retrieves details about a Chat space including name, type, description.

### create-space — Create a space

```bash
gws chat create-space --display-name "Team Chat" [flags]
```

**Flags:**
- `--display-name string` — Space display name (required)
- `--type string` — Space type: SPACE or GROUP_CHAT (default SPACE)
- `--description string` — Space description

### delete-space — Delete a space

```bash
gws chat delete-space <space>
```

### update-space — Update a space

```bash
gws chat update-space <space> [flags]
```

**Flags:**
- `--display-name string` — New display name
- `--description string` — New description

### search-spaces — Search for spaces (admin only)

> Requires Workspace admin privileges and `chat.admin.spaces` scope. Not available with regular user OAuth.

```bash
gws chat search-spaces --query "Engineering" [flags]
```

**Flags:**
- `--query string` — Search query (required)
- `--page-size int` — Results per page (default 100)

### find-dm — Find direct message space

```bash
gws chat find-dm --user users/123
```

**Flags:**
- `--user string` — User resource name or email (required, e.g. `users/123` or `users/user@example.com`)

### setup-space — Create space with members

```bash
gws chat setup-space --display-name "Project Team" --members "users/111,users/222"
```

**Flags:**
- `--display-name string` — Space display name (required)
- `--members string` — Comma-separated user resource names

### get-member — Get member details

```bash
gws chat get-member <member-name>
```

### add-member — Add a member to a space

```bash
gws chat add-member <space> --user users/123 [flags]
```

**Flags:**
- `--user string` — User resource name (required)
- `--role string` — Member role: ROLE_MEMBER or ROLE_MANAGER (default ROLE_MEMBER)

### remove-member — Remove a member

```bash
gws chat remove-member <member-name>
```

### update-member — Update member role

```bash
gws chat update-member <member-name> --role ROLE_MANAGER
```

**Flags:**
- `--role string` — New role: ROLE_MEMBER or ROLE_MANAGER (required)

### read-state — Get space read state

```bash
gws chat read-state <space>
```

Returns when you last read the space. Space ID is auto-expanded to the full read state resource name.

### mark-read — Mark space as read

```bash
gws chat mark-read <space> [flags]
```

**Flags:**
- `--time string` — Read time in RFC-3339 format (defaults to now)

### thread-read-state — Get thread read state

```bash
gws chat thread-read-state <thread-name>
```

Full resource name required (e.g. `users/me/spaces/AAAA/threads/thread1/threadReadState`).

### attachment — Get attachment metadata

```bash
gws chat attachment <attachment-name>
```

Returns metadata: name, content_name, content_type, source, download_uri, thumbnail_uri.

### upload — Upload a file

```bash
gws chat upload <space> --file ./report.pdf
```

**Flags:**
- `--file string` — Path to file to upload (required)

### download — Download media

```bash
gws chat download <resource-name> --output ./file.pdf
```

**Flags:**
- `--output string` — Output file path (required)

### events — List space events

```bash
gws chat events <space> --filter 'event_types:"google.workspace.chat.message.v1.created"' [flags]
```

**Flags:**
- `--filter string` — Event type filter (required — API requires it)
- `--page-size int` — Events per page (default 100)

### event — Get event details

```bash
gws chat event <event-name>
```

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
- `read-state` auto-expands bare space IDs (e.g. `AAAA` → `users/me/spaces/AAAA/spaceReadState`)
- `events` requires a `--filter` with event types — see [API docs](https://developers.google.com/workspace/chat/api/reference/rest/v1/spaces.spaceEvents/list)
- Chat API requires additional GCP setup beyond standard OAuth — see the `gws-auth` skill
