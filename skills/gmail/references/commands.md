# Gmail Commands Reference

Complete flag and option reference for `gws gmail` commands.

> **Disclaimer:** This is an unofficial CLI tool, not endorsed by or affiliated with Google.

## Global Flags

These flags apply to all `gws gmail` commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |

---

## gws gmail list

Lists recent email threads from your inbox.

```
Usage: gws gmail list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 10 | Maximum number of results |
| `--query` | string | | Gmail search query |

### Gmail Search Query Syntax

The `--query` flag supports Gmail's full search syntax:

| Operator | Example | Description |
|----------|---------|-------------|
| `is:unread` | `--query "is:unread"` | Unread messages |
| `is:starred` | `--query "is:starred"` | Starred messages |
| `is:important` | `--query "is:important"` | Important messages |
| `from:` | `--query "from:user@example.com"` | From a specific sender |
| `to:` | `--query "to:user@example.com"` | To a specific recipient |
| `subject:` | `--query "subject:meeting"` | Subject contains |
| `has:attachment` | `--query "has:attachment"` | Has attachments |
| `after:` | `--query "after:2024/01/01"` | After a date |
| `before:` | `--query "before:2024/12/31"` | Before a date |
| `label:` | `--query "label:work"` | Has a specific label |
| `in:anywhere` | `--query "in:anywhere search"` | Search all mail including spam/trash |
| Combined | `--query "from:boss is:unread after:2024/01/01"` | Multiple conditions |

---

## gws gmail read

Reads and displays the content of a specific email message.

```
Usage: gws gmail read <message-id>
```

No additional flags. The message ID is obtained from `gws gmail list` output.

### Output Fields (JSON)

- `id` — Message ID
- `threadId` — Thread ID
- `from` — Sender
- `to` — Recipients
- `subject` — Subject line
- `date` — Date sent
- `body` — Message body text
- `labels` — Applied label IDs

---

## gws gmail send

Sends a new email message.

```
Usage: gws gmail send [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--to` | string | | Yes | Recipient email address |
| `--subject` | string | | Yes | Email subject |
| `--body` | string | | Yes | Email body |
| `--cc` | string | | No | CC recipients (comma-separated) |
| `--bcc` | string | | No | BCC recipients (comma-separated) |

---

## gws gmail labels

Lists all Gmail labels in the account.

```
Usage: gws gmail labels
```

No additional flags.

### Output Fields (JSON)

Returns an array of labels, each with:
- `id` — Label ID (e.g., `INBOX`, `SENT`, `Label_123`)
- `name` — Label name
- `type` — Label type (`system` or `user`)

---

## gws gmail label

Adds or removes labels from a Gmail message.

```
Usage: gws gmail label <message-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--add` | string | | Label names to add (comma-separated) |
| `--remove` | string | | Label names to remove (comma-separated) |

At least one of `--add` or `--remove` is required.

### Label Name Resolution

Label names are resolved case-insensitively. Use `gws gmail labels` to see available names. Both system labels (`INBOX`, `STARRED`, `IMPORTANT`, `UNREAD`) and user-created labels are supported.

### Common Patterns

| Action | Command |
|--------|---------|
| Star a message | `gws gmail label <id> --add "STARRED"` |
| Mark as read | `gws gmail label <id> --remove "UNREAD"` |
| Mark as unread | `gws gmail label <id> --add "UNREAD"` |
| Mark important | `gws gmail label <id> --add "IMPORTANT"` |
| Move to inbox | `gws gmail label <id> --add "INBOX"` |
| Remove from inbox | `gws gmail label <id> --remove "INBOX"` |

---

## gws gmail archive

Archives a Gmail message by removing the INBOX label.

```
Usage: gws gmail archive <message-id>
```

No additional flags. Equivalent to `gws gmail label <id> --remove "INBOX"`.

---

## gws gmail trash

Moves a Gmail message to the trash.

```
Usage: gws gmail trash <message-id>
```

No additional flags. Messages in trash are permanently deleted after 30 days.
