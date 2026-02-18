---
name: gws-drive
version: 2.0.0
description: "Google Drive CLI operations via gws. Use when users need to list, search, upload, download, manage files/folders, permissions, revisions, comments, shared drives, and more. Triggers: drive, files, upload, download, folders, google drive, file management, permissions, share, shared drives."
metadata:
  short-description: Google Drive CLI operations
  compatibility: claude-code, codex-cli
---

# Google Drive (gws drive)

`gws drive` provides CLI access to Google Drive with structured JSON output.

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

### Files & Folders
| Task | Command |
|------|---------|
| List files | `gws drive list` |
| List files in folder | `gws drive list --folder <folder-id>` |
| Search files | `gws drive search "quarterly report"` |
| Get file info | `gws drive info <file-id>` |
| Download a file | `gws drive download <file-id>` |
| Upload a file | `gws drive upload report.pdf` |
| Create a folder | `gws drive create-folder --name "Project Files"` |
| Move a file | `gws drive move <file-id> --to <folder-id>` |
| Delete a file | `gws drive delete <file-id>` |
| Copy a file | `gws drive copy <file-id>` |
| Export file | `gws drive export --file-id <id> --mime-type application/pdf --output report.pdf` |
| Update metadata | `gws drive update --file-id <id> --name "New Name"` |
| Empty trash | `gws drive empty-trash` |
| Drive info | `gws drive about` |
| Recent changes | `gws drive changes` |

### Permissions
| Task | Command |
|------|---------|
| List permissions | `gws drive permissions --file-id <id>` |
| Share with user | `gws drive share --file-id <id> --type user --role writer --email user@example.com` |
| Share with anyone | `gws drive share --file-id <id> --type anyone --role reader` |
| Get permission | `gws drive permission --file-id <id> --permission-id <perm-id>` |
| Update permission | `gws drive update-permission --file-id <id> --permission-id <perm-id> --role reader` |
| Remove permission | `gws drive unshare --file-id <id> --permission-id <perm-id>` |

### Comments & Replies
| Task | Command |
|------|---------|
| List comments | `gws drive comments <file-id>` |
| Get comment | `gws drive comment --file-id <id> --comment-id <cid>` |
| Add comment | `gws drive add-comment --file-id <id> --content "Great work!"` |
| Delete comment | `gws drive delete-comment --file-id <id> --comment-id <cid>` |
| List replies | `gws drive replies --file-id <id> --comment-id <cid>` |
| Reply to comment | `gws drive reply --file-id <id> --comment-id <cid> --content "Thanks!"` |
| Get reply | `gws drive get-reply --file-id <id> --comment-id <cid> --reply-id <rid>` |
| Delete reply | `gws drive delete-reply --file-id <id> --comment-id <cid> --reply-id <rid>` |

### Revisions
| Task | Command |
|------|---------|
| List revisions | `gws drive revisions --file-id <id>` |
| Get revision | `gws drive revision --file-id <id> --revision-id <rid>` |
| Delete revision | `gws drive delete-revision --file-id <id> --revision-id <rid>` |

### Shared Drives
| Task | Command |
|------|---------|
| List shared drives | `gws drive shared-drives` |
| Get shared drive | `gws drive shared-drive --id <drive-id>` |
| Create shared drive | `gws drive create-drive --name "Engineering"` |
| Update shared drive | `gws drive update-drive --id <drive-id> --name "New Name"` |
| Delete shared drive | `gws drive delete-drive --id <drive-id>` |

## Detailed Usage

### list — List files

```bash
gws drive list [flags]
```

**Flags:**
- `--folder string` — Folder ID to list (default: "root")
- `--max int` — Maximum number of files (default 50)
- `--order string` — Sort order (default: "modifiedTime desc")

**Examples:**
```bash
gws drive list
gws drive list --folder 1abc123xyz --max 20
gws drive list --order "name"
```

### search — Search for files

```bash
gws drive search <query> [flags]
```

**Flags:**
- `--max int` — Maximum number of results (default 50)

**Examples:**
```bash
gws drive search "quarterly report"
gws drive search "budget 2024" --max 10
```

### info — Get file info

```bash
gws drive info <file-id>
```

Gets detailed information about a file including name, type, size, owners, and permissions.

### download — Download a file

```bash
gws drive download <file-id> [flags]
```

**Flags:**
- `--output string` — Output file path (default: original filename)

**Examples:**
```bash
gws drive download 1abc123xyz
gws drive download 1abc123xyz --output ./local-copy.pdf
```

### upload — Upload a file

```bash
gws drive upload <local-file> [flags]
```

**Flags:**
- `--folder string` — Parent folder ID (default: root)
- `--name string` — File name in Drive (default: local filename)
- `--mime-type string` — MIME type (auto-detected if not specified)

### create-folder — Create a new folder

```bash
gws drive create-folder --name <name> [flags]
```

**Flags:**
- `--name string` — Folder name (required)
- `--parent string` — Parent folder ID (default: root)

### move — Move a file

```bash
gws drive move <file-id> --to <folder-id>
```

### delete — Delete a file

```bash
gws drive delete <file-id> [flags]
```

By default, moves to trash. Use `--permanent` to permanently delete.

### copy — Copy a file

```bash
gws drive copy <file-id> [flags]
```

**Flags:**
- `--name string` — Name for the copy
- `--folder string` — Destination folder ID

### export — Export a Google Workspace file

```bash
gws drive export --file-id <id> --mime-type <mime> --output <path>
```

Exports Docs, Sheets, Slides to formats like PDF, CSV, DOCX, etc.

**Flags:**
- `--file-id string` — File ID (required)
- `--mime-type string` — Export MIME type (required, e.g. `application/pdf`, `text/csv`)
- `--output string` — Output file path (required)

### update — Update file metadata

```bash
gws drive update --file-id <id> [flags]
```

**Flags:**
- `--file-id string` — File ID (required)
- `--name string` — New file name
- `--description string` — New description
- `--starred` — Star or unstar the file
- `--trashed` — Trash or untrash the file

### empty-trash — Empty trash

```bash
gws drive empty-trash
```

Permanently deletes all files in the trash. Cannot be undone.

### about — Drive storage and user info

```bash
gws drive about
```

Returns user info and storage quota (limit, usage, usage in Drive, usage in trash).

### changes — List recent file changes

```bash
gws drive changes [flags]
```

Polling pattern: the first call (without `--page-token`) fetches the current start token
and typically returns zero results. Save the returned `new_start_page_token` and pass it
in subsequent calls to detect new changes.

**Flags:**
- `--max int` — Maximum number of changes (default 100)
- `--page-token string` — Page token from a previous call (auto-fetches start token if empty)

### permissions — List permissions

```bash
gws drive permissions --file-id <id>
```

### share — Share a file

```bash
gws drive share --file-id <id> --type <type> --role <role> [flags]
```

**Flags:**
- `--file-id string` — File ID (required)
- `--type string` — Permission type: `user`, `group`, `domain`, `anyone` (required)
- `--role string` — Role: `reader`, `commenter`, `writer`, `organizer`, `owner` (required)
- `--email string` — Email address (for user/group type)
- `--domain string` — Domain (for domain type)
- `--send-notification` — Send notification email (default: true)

### unshare — Remove a permission

```bash
gws drive unshare --file-id <id> --permission-id <perm-id>
```

### permission — Get permission details

```bash
gws drive permission --file-id <id> --permission-id <perm-id>
```

### update-permission — Update a permission

```bash
gws drive update-permission --file-id <id> --permission-id <perm-id> --role <role>
```

### comments — List comments

```bash
gws drive comments <file-id> [flags]
```

**Flags:**
- `--max int` — Maximum number of comments (default 100)
- `--include-resolved` — Include resolved comments
- `--include-deleted` — Include deleted comments

### comment — Get a single comment

```bash
gws drive comment --file-id <id> --comment-id <cid>
```

### add-comment — Add a comment

```bash
gws drive add-comment --file-id <id> --content "comment text"
```

### delete-comment — Delete a comment

```bash
gws drive delete-comment --file-id <id> --comment-id <cid>
```

### replies — List replies

```bash
gws drive replies --file-id <id> --comment-id <cid>
```

### reply — Create a reply

```bash
gws drive reply --file-id <id> --comment-id <cid> --content "reply text"
```

### get-reply — Get a reply

```bash
gws drive get-reply --file-id <id> --comment-id <cid> --reply-id <rid>
```

### delete-reply — Delete a reply

```bash
gws drive delete-reply --file-id <id> --comment-id <cid> --reply-id <rid>
```

### revisions — List revisions

```bash
gws drive revisions --file-id <id>
```

### revision — Get revision details

```bash
gws drive revision --file-id <id> --revision-id <rid>
```

### delete-revision — Delete a revision

```bash
gws drive delete-revision --file-id <id> --revision-id <rid>
```

### shared-drives — List shared drives

```bash
gws drive shared-drives [flags]
```

**Flags:**
- `--max int` — Maximum number of drives (default 100)
- `--query string` — Search query

### shared-drive — Get shared drive info

```bash
gws drive shared-drive --id <drive-id>
```

### create-drive — Create a shared drive

```bash
gws drive create-drive --name "Drive Name"
```

### update-drive — Update a shared drive

```bash
gws drive update-drive --id <drive-id> --name "New Name"
```

### delete-drive — Delete a shared drive

```bash
gws drive delete-drive --id <drive-id>
```

## Output Modes

```bash
gws drive list --format json    # Structured JSON (default)
gws drive list --format yaml    # YAML format
gws drive list --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws drive search` to find files by name, then `gws drive info <id>` for details
- File IDs from Google Docs/Sheets/Slides URLs can be extracted from the URL path
- Delete moves to trash by default — use `--permanent` only when explicitly requested
- When uploading, MIME type is auto-detected from the file extension
- The `comments` command works on any Drive file type (Docs, Sheets, Slides, etc.)
- Resolved comments are excluded by default; use `--include-resolved` to see them
- Use `gws drive about` to check storage quota before large uploads
- Use `gws drive changes` to monitor recent file activity
- For sharing, use `--type anyone --role reader` for public access
- For Workspace file exports, common MIME types: `application/pdf`, `text/csv`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
