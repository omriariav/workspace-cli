---
name: gws-drive
version: 1.0.0
description: "Google Drive CLI operations via gws. Use when users need to list, search, upload, download, or manage files and folders in Google Drive. Triggers: drive, files, upload, download, folders, google drive, file management."
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

| Task | Command |
|------|---------|
| List files | `gws drive list` |
| List files in folder | `gws drive list --folder <folder-id>` |
| Search files | `gws drive search "quarterly report"` |
| Get file info | `gws drive info <file-id>` |
| Download a file | `gws drive download <file-id>` |
| Upload a file | `gws drive upload report.pdf` |
| Upload to folder | `gws drive upload data.xlsx --folder <folder-id>` |
| Create a folder | `gws drive create-folder --name "Project Files"` |
| Move a file | `gws drive move <file-id> --to <folder-id>` |
| Delete a file | `gws drive delete <file-id>` |
| Permanently delete | `gws drive delete <file-id> --permanent` |
| List comments | `gws drive comments <file-id>` |

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

**Examples:**
```bash
gws drive upload report.pdf
gws drive upload data.xlsx --folder 1abc123xyz
gws drive upload document.docx --name "My Report"
```

### create-folder — Create a new folder

```bash
gws drive create-folder --name <name> [flags]
```

**Flags:**
- `--name string` — Folder name (required)
- `--parent string` — Parent folder ID (default: root)

**Examples:**
```bash
gws drive create-folder --name "Project Files"
gws drive create-folder --name "Subproject" --parent 1abc123xyz
```

### move — Move a file

```bash
gws drive move <file-id> --to <folder-id>
```

**Flags:**
- `--to string` — Destination folder ID (required)

**Examples:**
```bash
gws drive move 1abc123xyz --to 2def456uvw
gws drive move 1abc123xyz --to root
```

### delete — Delete a file

```bash
gws drive delete <file-id> [flags]
```

By default, moves the file to trash. Use `--permanent` to permanently delete.

**Flags:**
- `--permanent` — Permanently delete (skip trash)

**Examples:**
```bash
gws drive delete 1abc123xyz
gws drive delete 1abc123xyz --permanent
```

### comments — List comments on a file

```bash
gws drive comments <file-id> [flags]
```

Lists all comments and replies on a Google Drive file (works with Docs, Sheets, Slides).

**Flags:**
- `--max int` — Maximum number of comments (default 100)
- `--include-resolved` — Include resolved comments
- `--include-deleted` — Include deleted comments

**Examples:**
```bash
gws drive comments 1abc123xyz
gws drive comments 1abc123xyz --include-resolved
```

## Output Modes

```bash
gws drive list --format json    # Structured JSON (default)
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
