---
name: gws-gmail
version: 1.1.0
description: "Google Gmail CLI operations via gws. Use when users need to list emails, read messages, send email, manage labels, archive, or trash messages. Triggers: gmail, email, inbox, send email, mail, labels, archive, trash."
metadata:
  short-description: Google Gmail CLI operations
  compatibility: claude-code, codex-cli
---

# Google Gmail (gws gmail)

`gws gmail` provides CLI access to Gmail with structured JSON output.

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

## Quick Command Reference

| Task | Command |
|------|---------|
| List recent emails | `gws gmail list` |
| List unread emails | `gws gmail list --query "is:unread"` |
| Search emails | `gws gmail list --query "from:user@example.com"` |
| Read a message | `gws gmail read <message-id>` |
| Read full thread | `gws gmail thread <thread-id>` |
| Send an email | `gws gmail send --to user@example.com --subject "Hi" --body "Hello"` |
| List all labels | `gws gmail labels` |
| Add labels | `gws gmail label <message-id> --add "STARRED"` |
| Remove labels | `gws gmail label <message-id> --remove "UNREAD"` |
| Archive a message | `gws gmail archive <message-id>` |
| Archive a thread | `gws gmail archive-thread <thread-id>` |
| Trash a message | `gws gmail trash <message-id>` |

## Detailed Usage

### list — List recent messages/threads

```bash
gws gmail list [flags]
```

Lists recent email threads from your inbox. Each result includes:
- `thread_id` — Thread ID (use with `gws gmail thread`)
- `message_id` — Latest message ID (use with `read`, `label`, `archive`, `trash`)
- `message_count` — Number of messages in the thread

**Flags:**
- `--max int` — Maximum number of results (default 10)
- `--query string` — Gmail search query (e.g., `is:unread`, `from:someone@example.com`)

**Examples:**
```bash
gws gmail list --max 5
gws gmail list --query "is:unread"
gws gmail list --query "from:boss@company.com subject:urgent"
gws gmail list --query "after:2024/01/01 has:attachment"
```

### thread — Read a full thread

```bash
gws gmail thread <thread-id>
```

Reads and displays all messages in a Gmail thread (conversation). Use the `thread_id` from `gws gmail list` output. Returns all messages with headers, body, and labels.

### read — Read a message

```bash
gws gmail read <message-id>
```

Reads and displays the content of a specific email message. Use the `message_id` from `gws gmail list` output.

### send — Send an email

```bash
gws gmail send --to <email> --subject <subject> --body <body> [flags]
```

**Flags:**
- `--to string` — Recipient email address (required)
- `--subject string` — Email subject (required)
- `--body string` — Email body (required)
- `--cc string` — CC recipients (comma-separated)
- `--bcc string` — BCC recipients (comma-separated)

**Examples:**
```bash
gws gmail send --to user@example.com --subject "Meeting" --body "Let's meet at 3pm"
gws gmail send --to user@example.com --cc team@example.com --subject "Update" --body "Status update"
```

### labels — List all labels

```bash
gws gmail labels
```

Lists all Gmail labels in the account, including system labels (INBOX, SENT, etc.) and user-created labels.

### label — Add or remove labels

```bash
gws gmail label <message-id> [flags]
```

**Flags:**
- `--add string` — Label names to add (comma-separated)
- `--remove string` — Label names to remove (comma-separated)

Use `gws gmail labels` to see available label names.

**Examples:**
```bash
gws gmail label 18abc123 --add "STARRED"
gws gmail label 18abc123 --add "ActionNeeded,IMPORTANT" --remove "INBOX"
gws gmail label 18abc123 --remove "UNREAD"
```

### archive — Archive a message

```bash
gws gmail archive <message-id>
```

Archives a Gmail message by removing the INBOX label. The message remains accessible via search and labels.

### archive-thread — Archive all messages in a thread

```bash
gws gmail archive-thread <thread-id>
```

Archives all messages in a Gmail thread by removing the INBOX label and marking all messages as read. Use the `thread_id` from `gws gmail list` output. More efficient than archiving individual messages for multi-message threads.

### trash — Trash a message

```bash
gws gmail trash <message-id>
```

Moves a Gmail message to the trash. Messages in trash are permanently deleted after 30 days.

## Output Modes

```bash
gws gmail list --format json    # Structured JSON (default)
gws gmail list --format text    # Human-readable text
```

## Tips for AI Agents

### Search Strategy for Finding Emails to/from a Person
- **Get the exact email first**: If using employee lookup tools, use the returned email address — don't guess the format (e.g., `lior.g@` not `lior.golan@`)
- **Use broad search first**: `from:me lior.g@taboola.com subject:keyword` searches both to/cc fields
- **Narrow with `to:` only if needed**: `to:` only matches the To header, not CC — use the bare email address to match both
- **When `list` isn't enough**: The `message_id` in list results is the *latest* message in the thread (could be from anyone). To find a specific person's message, use `gws gmail thread <thread-id>` and scan all messages.

### Common Gotchas
- `list` returns the latest `message_id` per thread — reading it may show someone else's reply, not the user's email
- Gmail `to:` operator only matches the To header, not CC recipients
- For multi-message threads, always prefer `thread` over `read` when searching for a specific message within a conversation

- Always use `--format json` (the default) for programmatic parsing
- Use `gws gmail list` to get IDs: `message_id` for `read`/`label`/`archive`/`trash`, `thread_id` for `thread`
- Use `gws gmail thread <thread-id>` to view full conversations with all messages
- Gmail search query syntax supports operators like `is:`, `from:`, `to:`, `subject:`, `after:`, `before:`, `has:`, `label:`
- When managing labels, run `gws gmail labels` first to see available label names and IDs
- Archive is a shortcut for `gws gmail label <id> --remove "INBOX"`
- Use `gws gmail archive-thread <thread-id>` to archive all messages in a conversation at once (archives + marks read)
- To mark as read: `gws gmail label <id> --remove "UNREAD"`
- To star a message: `gws gmail label <id> --add "STARRED"`
