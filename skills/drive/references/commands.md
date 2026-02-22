# Drive Commands Reference

Complete flag and option reference for `gws drive` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws drive list

Lists files and folders in Google Drive.

```
Usage: gws drive list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--folder` | string | `root` | Folder ID to list |
| `--max` | int | 50 | Maximum number of files |
| `--order` | string | `modifiedTime desc` | Sort order |

### Sort Order Values

- `name` — Alphabetical by name
- `modifiedTime desc` — Most recently modified first (default)
- `createdTime desc` — Most recently created first
- `name desc` — Reverse alphabetical

---

## gws drive search

Searches for files in Google Drive using a query string.

```
Usage: gws drive search <query> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 50 | Maximum number of results |

---

## gws drive info

Gets detailed information about a file.

```
Usage: gws drive info <file-id>
```

No additional flags.

---

## gws drive download

Downloads a file from Google Drive.

```
Usage: gws drive download <file-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | original filename | Output file path |

---

## gws drive upload

Uploads a local file to Google Drive.

```
Usage: gws drive upload <local-file> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--folder` | string | root | Parent folder ID |
| `--name` | string | local filename | File name in Drive |
| `--mime-type` | string | auto-detected | MIME type |

---

## gws drive create-folder

Creates a new folder in Google Drive.

```
Usage: gws drive create-folder [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Folder name |
| `--parent` | string | root | No | Parent folder ID |

---

## gws drive move

Moves a file to a different folder in Google Drive.

```
Usage: gws drive move <file-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--to` | string | | Yes | Destination folder ID |

---

## gws drive delete

Deletes a file from Google Drive.

```
Usage: gws drive delete <file-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--permanent` | bool | false | Permanently delete (skip trash) |

---

## gws drive copy

Creates a copy of a file in Google Drive.

```
Usage: gws drive copy <file-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | "Copy of <original>" | Name for the copy |
| `--folder` | string | same as original | Destination folder ID |

---

## gws drive comments

Lists all comments and replies on a Google Drive file.

```
Usage: gws drive comments <file-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 100 | Maximum number of comments |
| `--include-resolved` | bool | false | Include resolved comments |
| `--include-deleted` | bool | false | Include deleted comments |

---

## gws drive export

Exports a Google Workspace file to a specified format.

```
Usage: gws drive export [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--mime-type` | string | | Yes | Export MIME type (e.g. `application/pdf`) |
| `--output` | string | | Yes | Output file path |

### Common Export MIME Types

| Format | MIME Type |
|--------|-----------|
| PDF | `application/pdf` |
| CSV | `text/csv` |
| DOCX | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| XLSX | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| PPTX | `application/vnd.openxmlformats-officedocument.presentationml.presentation` |
| Plain Text | `text/plain` |
| HTML | `text/html` |

---

## gws drive empty-trash

Permanently deletes all files in the trash. Cannot be undone.

```
Usage: gws drive empty-trash
```

No flags.

---

## gws drive update

Updates metadata of a file in Google Drive.

```
Usage: gws drive update [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--name` | string | | No | New file name |
| `--description` | string | | No | New description |
| `--starred` | bool | false | No | Star or unstar the file |
| `--trashed` | bool | false | No | Trash or untrash the file |

---

## gws drive about

Gets information about the user's Drive storage quota and account.

```
Usage: gws drive about
```

No flags. Returns user info and storage quota (limit, usage, usage in Drive, usage in trash).

---

## gws drive changes

Lists recent changes to files in Google Drive.

```
Usage: gws drive changes [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 100 | Maximum number of changes |
| `--page-token` | string | auto-fetched | Page token for pagination |

If no page token is provided, the start page token is fetched automatically.

---

## gws drive permissions

Lists all permissions on a file.

```
Usage: gws drive permissions [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |

---

## gws drive share

Shares a file with a user, group, domain, or anyone.

```
Usage: gws drive share [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--type` | string | | Yes | Permission type: `user`, `group`, `domain`, `anyone` |
| `--role` | string | | Yes | Role: `reader`, `commenter`, `writer`, `organizer`, `owner` |
| `--email` | string | | No | Email address (for user/group type) |
| `--domain` | string | | No | Domain (for domain type) |
| `--send-notification` | bool | true | No | Send notification email |

---

## gws drive unshare

Removes a permission from a file.

```
Usage: gws drive unshare [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--permission-id` | string | | Yes | Permission ID |

---

## gws drive permission

Gets details of a specific permission.

```
Usage: gws drive permission [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--permission-id` | string | | Yes | Permission ID |

---

## gws drive update-permission

Updates the role of an existing permission.

```
Usage: gws drive update-permission [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--permission-id` | string | | Yes | Permission ID |
| `--role` | string | | Yes | New role |

---

## gws drive comment

Gets a specific comment on a file.

```
Usage: gws drive comment [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--comment-id` | string | | Yes | Comment ID |

---

## gws drive add-comment

Adds a comment to a file.

```
Usage: gws drive add-comment [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--content` | string | | Yes | Comment content |

---

## gws drive delete-comment

Deletes a comment from a file.

```
Usage: gws drive delete-comment [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--comment-id` | string | | Yes | Comment ID |

---

## gws drive replies

Lists all replies to a comment.

```
Usage: gws drive replies [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--comment-id` | string | | Yes | Comment ID |

---

## gws drive reply

Creates a reply to a comment.

```
Usage: gws drive reply [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--comment-id` | string | | Yes | Comment ID |
| `--content` | string | | Yes | Reply content |

---

## gws drive get-reply

Gets a specific reply.

```
Usage: gws drive get-reply [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--comment-id` | string | | Yes | Comment ID |
| `--reply-id` | string | | Yes | Reply ID |

---

## gws drive delete-reply

Deletes a reply.

```
Usage: gws drive delete-reply [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--comment-id` | string | | Yes | Comment ID |
| `--reply-id` | string | | Yes | Reply ID |

---

## gws drive revisions

Lists all revisions of a file.

```
Usage: gws drive revisions [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |

---

## gws drive revision

Gets details of a specific revision.

```
Usage: gws drive revision [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--revision-id` | string | | Yes | Revision ID |

---

## gws drive delete-revision

Deletes a specific revision.

```
Usage: gws drive delete-revision [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file-id` | string | | Yes | File ID |
| `--revision-id` | string | | Yes | Revision ID |

---

## gws drive shared-drives

Lists all shared drives.

```
Usage: gws drive shared-drives [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 100 | Maximum number of shared drives |
| `--query` | string | | Search query |

---

## gws drive shared-drive

Gets information about a shared drive.

```
Usage: gws drive shared-drive [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Shared drive ID |

---

## gws drive create-drive

Creates a new shared drive.

```
Usage: gws drive create-drive [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Shared drive name |

---

## gws drive delete-drive

Deletes a shared drive.

```
Usage: gws drive delete-drive [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Shared drive ID |

---

## gws drive update-drive

Updates a shared drive.

```
Usage: gws drive update-drive [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Shared drive ID |
| `--name` | string | | No | New name for the shared drive |

---

## gws drive activity

Queries the Google Drive Activity API v2 for file and folder activity history.

```
Usage: gws drive activity [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--item-id` | string | | Filter by file/folder ID |
| `--folder-id` | string | | Filter by ancestor folder (all descendants) |
| `--filter` | string | | API filter string (e.g. `"detail.action_detail_case:EDIT"`) |
| `--days` | int | 0 | Last N days (auto-generates time filter) |
| `--max` | int | 50 | Page size |
| `--page-token` | string | | Pagination token |
| `--no-consolidation` | bool | false | Disable activity grouping |

### Filter Syntax

- `time >= <epoch_ms>` or `time >= "<RFC 3339>"` — filter by time
- `detail.action_detail_case:EDIT` — filter by action type
- Combine with `AND`: `time >= 1700000000000 AND detail.action_detail_case:EDIT`

### Action Types

CREATE, EDIT, MOVE, RENAME, DELETE, RESTORE, COMMENT, PERMISSION_CHANGE, SETTINGS_CHANGE, DLP_CHANGE, REFERENCE, APPLIED_LABEL_CHANGE
