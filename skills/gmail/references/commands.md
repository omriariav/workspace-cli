# Gmail Commands Reference

Complete flag and option reference for `gws gmail` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

These flags apply to all `gws gmail` commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws gmail list

Lists recent email threads from your inbox.

```
Usage: gws gmail list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 10 | Maximum number of results (use `--all` for unlimited) |
| `--all` | bool | false | Fetch all matching results (may take time for large result sets) |
| `--query` | string | | Gmail search query |
| `--include-labels` | bool | false | Include Gmail label IDs in output |

### Output Fields (JSON)

Each thread includes:
- `thread_id` — Thread ID (use with `gws gmail thread`)
- `message_id` — Latest message ID (use with `read`, `label`, `archive`, `trash`)
- `message_count` — Number of messages in the thread
- `subject` — Thread subject
- `from` — Original sender
- `date` — Date of first message
- `snippet` — Preview text
- `labels` — Array of label IDs (only when `--include-labels` is used)

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

No additional flags. Use the `message_id` from `gws gmail list` output.

### Output Fields (JSON)

- `id` — Message ID
- `headers` — Object with `subject`, `from`, `to`, `date`, `cc`, `bcc`
- `body` — Message body text
- `labels` — Applied label IDs

---

## gws gmail thread

Reads and displays all messages in a Gmail thread (conversation).

```
Usage: gws gmail thread <thread-id>
```

No additional flags. Use the `thread_id` from `gws gmail list` output.

### Output Fields (JSON)

- `thread_id` — Thread ID
- `message_count` — Number of messages in thread
- `messages` — Array of messages, each with:
  - `id` — Message ID
  - `headers` — Object with `subject`, `from`, `to`, `date`, `cc`, `bcc`
  - `body` — Message body text
  - `labels` — Applied label IDs

---

## gws gmail send

Sends a new email message. Supports replying within an existing thread.

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
| `--thread-id` | string | | No | Thread ID to reply in |
| `--reply-to-message-id` | string | | No | Message ID to reply to (sets In-Reply-To/References headers) |

---

## gws gmail reply

Replies to an existing email message within its thread.

```
Usage: gws gmail reply <message-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--body` | string | | Yes | Reply body |
| `--cc` | string | | No | CC recipients (comma-separated) |
| `--bcc` | string | | No | BCC recipients (comma-separated) |
| `--all` | bool | false | No | Reply to all recipients |

### Output Fields (JSON)

- `status` — Always `"sent"`
- `message_id` — New reply message ID
- `thread_id` — Thread ID
- `in_reply_to` — Original message ID

---

## gws gmail event-id

Extracts the Google Calendar event ID from a calendar invite email.

```
Usage: gws gmail event-id <message-id>
```

No additional flags. Parses the `eid` parameter from Google Calendar URLs in the email body and base64 decodes it.

### Output Fields (JSON)

- `message_id` — Source message ID
- `event_id` — Extracted calendar event ID
- `subject` — Email subject (for context)

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

## gws gmail archive-thread

Archives all messages in a Gmail thread by removing the INBOX label and marking all messages as read.

```
Usage: gws gmail archive-thread <thread-id>
```

No additional flags. Use the `thread_id` from `gws gmail list` output. More efficient than archiving individual messages for multi-message threads.

### Output Fields (JSON)

- `status` — Always `"archived"`
- `thread_id` — Thread ID
- `archived` — Number of messages successfully archived
- `failed` — Number of messages that failed to archive
- `total` — Total messages in the thread

---

## gws gmail trash

Moves a Gmail message to the trash.

```
Usage: gws gmail trash <message-id>
```

No additional flags. Messages in trash are permanently deleted after 30 days.

---

## gws gmail untrash

Removes a Gmail message from the trash.

```
Usage: gws gmail untrash <message-id>
```

No additional flags. Restores the message to its previous location.

---

## gws gmail delete

Permanently deletes a Gmail message. This action cannot be undone.

```
Usage: gws gmail delete <message-id>
```

No additional flags.

---

## gws gmail batch-modify

Modifies labels on multiple Gmail messages at once.

```
Usage: gws gmail batch-modify [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--ids` | string | | Yes | Comma-separated message IDs |
| `--add-labels` | string | | No | Label names to add (comma-separated) |
| `--remove-labels` | string | | No | Label names to remove (comma-separated) |

At least one of `--add-labels` or `--remove-labels` is required.

---

## gws gmail batch-delete

Permanently deletes multiple Gmail messages at once. This action cannot be undone.

```
Usage: gws gmail batch-delete [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--ids` | string | | Yes | Comma-separated message IDs |

---

## gws gmail trash-thread

Moves all messages in a Gmail thread to the trash.

```
Usage: gws gmail trash-thread <thread-id>
```

No additional flags.

---

## gws gmail untrash-thread

Removes all messages in a Gmail thread from the trash.

```
Usage: gws gmail untrash-thread <thread-id>
```

No additional flags.

---

## gws gmail delete-thread

Permanently deletes all messages in a Gmail thread. This action cannot be undone.

```
Usage: gws gmail delete-thread <thread-id>
```

No additional flags.

---

## gws gmail label-info

Gets detailed information about a specific Gmail label.

```
Usage: gws gmail label-info [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Label ID |

### Output Fields (JSON)

- `id` — Label ID
- `name` — Label name
- `type` — Label type (`system` or `user`)
- `message_list_visibility` — Visibility in message list
- `label_list_visibility` — Visibility in label list
- `messages_total` — Total messages with this label
- `messages_unread` — Unread messages with this label
- `threads_total` — Total threads with this label
- `threads_unread` — Unread threads with this label

---

## gws gmail create-label

Creates a new Gmail label.

```
Usage: gws gmail create-label [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Label name |
| `--visibility` | string | | No | Message visibility: `labelShow`, `labelShowIfUnread`, `labelHide` |
| `--list-visibility` | string | | No | Label list visibility: `labelShow`, `labelHide` |

---

## gws gmail update-label

Updates an existing Gmail label.

```
Usage: gws gmail update-label [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Label ID |
| `--name` | string | | No | New label name |
| `--visibility` | string | | No | Message visibility: `labelShow`, `labelShowIfUnread`, `labelHide` |
| `--list-visibility` | string | | No | Label list visibility: `labelShow`, `labelHide` |

---

## gws gmail delete-label

Permanently deletes a Gmail label. Messages with this label are not deleted.

```
Usage: gws gmail delete-label [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Label ID |

---

## gws gmail drafts

Lists Gmail drafts.

```
Usage: gws gmail drafts [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--max` | int | 10 | No | Maximum number of results |
| `--query` | string | | No | Gmail search query |

### Output Fields (JSON)

- `drafts` — Array of drafts, each with:
  - `id` — Draft ID
  - `message_id` — Associated message ID
- `count` — Total number of drafts returned

---

## gws gmail draft

Gets the content of a specific Gmail draft.

```
Usage: gws gmail draft [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Draft ID |

### Output Fields (JSON)

- `id` — Draft ID
- `message_id` — Associated message ID
- `headers` — Object with `subject`, `from`, `to`, `date`, `cc`, `bcc`
- `body` — Draft body text

---

## gws gmail create-draft

Creates a new Gmail draft message.

```
Usage: gws gmail create-draft [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--to` | string | | Yes | Recipient email address |
| `--subject` | string | | No | Email subject |
| `--body` | string | | No | Email body |
| `--cc` | string | | No | CC recipients (comma-separated) |
| `--bcc` | string | | No | BCC recipients (comma-separated) |
| `--thread-id` | string | | No | Thread ID for reply draft |

---

## gws gmail update-draft

Replaces the content of an existing Gmail draft.

```
Usage: gws gmail update-draft [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Draft ID |
| `--to` | string | | No | Recipient email address |
| `--subject` | string | | No | Email subject |
| `--body` | string | | No | Email body |
| `--cc` | string | | No | CC recipients (comma-separated) |
| `--bcc` | string | | No | BCC recipients (comma-separated) |

---

## gws gmail send-draft

Sends an existing Gmail draft.

```
Usage: gws gmail send-draft [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Draft ID |

### Output Fields (JSON)

- `status` — Always `"sent"`
- `message_id` — Sent message ID
- `thread_id` — Thread ID

---

## gws gmail delete-draft

Permanently deletes a Gmail draft.

```
Usage: gws gmail delete-draft [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Draft ID |

---

## gws gmail attachment

Downloads a Gmail message attachment to a local file.

```
Usage: gws gmail attachment [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--message-id` | string | | Yes | Message ID |
| `--id` | string | | Yes | Attachment ID |
| `--output` | string | | Yes | Output file path |

### Output Fields (JSON)

- `status` — Always `"downloaded"`
- `file` — Output file path
- `size` — File size in bytes
