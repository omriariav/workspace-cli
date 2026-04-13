---
name: gws-keep
version: 1.0.0
description: "Google Keep CLI operations via gws. Use when users need to list, view, or create Google Keep notes. Triggers: keep, google keep, notes, sticky notes."
metadata:
  short-description: Google Keep CLI operations
  compatibility: claude-code, codex-cli
---

# Google Keep (gws keep)

`gws keep` provides CLI access to Google Keep via the Keep API with structured JSON output.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

**Minimum version required:** v1.18.0 (Keep support added in this release)

## Prerequisites

- **Keep API** must be enabled in the Google Cloud project
- **Google Workspace Enterprise plan** is required (the Keep API is not available for personal Gmail or standard Workspace accounts)

## Authentication

Requires OAuth2 credentials. Run `gws auth status` to check.
If not authenticated: `gws auth login` (opens browser for OAuth consent).
For initial setup, see the `gws-auth` skill.

## Quick Command Reference

| Task | Command |
|------|---------|
| List notes | `gws keep list` |
| List more notes | `gws keep list --max 50` |
| Get a specific note | `gws keep get <note-id>` |
| Create a note | `gws keep create --title "Title" --text "Content"` |

## Detailed Usage

### list -- List notes

```bash
gws keep list [flags]
```

Lists notes from Google Keep.

**Flags:**
- `--max int` -- Maximum number of notes to return (default 20)

**Output includes:**
- `name` -- Note resource name (e.g., `notes/abc123`)
- `title` -- Note title
- `text` -- Note text content (if available)
- `create_time` -- Creation timestamp (if available)
- `update_time` -- Last update timestamp (if available)
- `trashed` -- Whether the note is trashed (only present if true)

**Examples:**
```bash
gws keep list
gws keep list --max 50
gws keep list --format json | jq '.notes[] | {name, title}'
```

### get -- Get a note

```bash
gws keep get <note-id>
```

Gets a specific note from Google Keep by its ID.

**Arguments:**
- `note-id` -- Note identifier (required, e.g., `notes/abc123` or just `abc123`)

**Output includes:**
- `name` -- Note resource name
- `title` -- Note title
- `text` -- Note text content (if available)
- `create_time` -- Creation timestamp (if available)
- `update_time` -- Last update timestamp (if available)
- `trashed` -- Whether the note is trashed (only present if true)

**Examples:**
```bash
gws keep get notes/abc123
gws keep get abc123
```

**Tip:** The `notes/` prefix is optional -- the CLI automatically prepends it if missing:
```bash
gws keep list --format json | jq -r '.notes[0].name' | xargs gws keep get
```

### create -- Create a note

```bash
gws keep create --title <title> --text <text>
```

Creates a new note in Google Keep.

**Flags:**
- `--title string` -- Note title (required)
- `--text string` -- Note text content (required)

**Output includes:**
- `name` -- New note's resource name
- `title` -- Note title
- `text` -- Note text content
- `create_time` -- Creation timestamp
- `update_time` -- Last update timestamp

**Examples:**
```bash
gws keep create --title "Shopping List" --text "Milk, eggs, bread"
gws keep create --title "Meeting Notes" --text "Discuss Q1 goals"
gws keep create --title "Reminder" --text "Call dentist at 3pm"
```

## Output Modes

```bash
gws keep list --format json    # Structured JSON (default)
gws keep list --format yaml    # YAML format
gws keep list --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws keep list` to discover note IDs, then use `get` for full details
- Note IDs follow the format `notes/<alphanumeric-id>` (e.g., `notes/abc123`)
- The `get` command accepts both `notes/abc123` and `abc123` -- the `notes/` prefix is added automatically if missing
- Both `--title` and `--text` are required when creating a note
- The `list` command default is 20 notes; increase with `--max` if you need more
- Use `--quiet` on any command to suppress JSON output (useful for scripted actions)
- This command requires the Keep API enabled and a Google Workspace Enterprise plan -- it will not work with personal Gmail or standard Workspace accounts
- The Keep API is read/create only via this CLI -- note updates and deletions are not currently supported
