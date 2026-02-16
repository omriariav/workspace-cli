---
name: gws-contacts
version: 1.0.0
description: "Google Contacts CLI operations via gws. Use when users need to list, search, view, create, or delete contacts. Triggers: contacts, google contacts, people api, contact management."
metadata:
  short-description: Google Contacts CLI operations
  compatibility: claude-code, codex-cli
---

# Google Contacts (gws contacts)

`gws contacts` provides CLI access to Google Contacts via the People API with structured JSON output.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

**Minimum version required:** v1.14.0 (Contacts support added in this release)

## Authentication

Requires OAuth2 credentials. Run `gws auth status` to check.
If not authenticated: `gws auth login` (opens browser for OAuth consent).
For initial setup, see the `gws-auth` skill.

## Quick Command Reference

| Task | Command |
|------|---------|
| List contacts | `gws contacts list` |
| List more contacts | `gws contacts list --max 100` |
| Search by name | `gws contacts search "John Doe"` |
| Search by email | `gws contacts search "john@example.com"` |
| Get contact details | `gws contacts get <resource-name>` |
| Create a contact | `gws contacts create --name "Jane Smith" --email "jane@example.com"` |
| Delete a contact | `gws contacts delete <resource-name>` |

## Detailed Usage

### list — List contacts

```bash
gws contacts list [flags]
```

Lists contacts from your Google account.

**Flags:**
- `--max int` — Maximum number of contacts to return (default 50)

**Output includes:**
- `resource_name` — Resource identifier (e.g., `people/c1234567890`)
- `name` — Contact's display name
- `emails` — Array of email addresses
- `phones` — Array of phone numbers
- `organization` — Organization info (name and title)

**Examples:**
```bash
gws contacts list
gws contacts list --max 100
gws contacts list --format json | jq '.contacts[] | select(.name | contains("Smith"))'
```

### search — Search contacts

```bash
gws contacts search <query>
```

Searches contacts by name, email, or phone number.

**Arguments:**
- `query` — Search string (required)

**Search behavior:**
- Searches across names, email addresses, and phone numbers
- Case-insensitive matching
- Returns contacts that match any field

**Examples:**
```bash
gws contacts search "John"
gws contacts search "john@example.com"
gws contacts search "555-1234"
gws contacts search "Company Inc"
```

### get — Get contact details

```bash
gws contacts get <resource-name>
```

Gets detailed information about a specific contact by resource name.

**Arguments:**
- `resource-name` — Resource identifier (required, e.g., `people/c1234567890`)

**Output includes:**
- `resource_name` — Resource identifier
- `name` — Contact's display name
- `emails` — Array of email addresses
- `phones` — Array of phone numbers
- `organization` — Organization info (name and title)

**Examples:**
```bash
gws contacts get people/c1234567890
```

**Tip:** Get the `resource_name` from `list` or `search` output:
```bash
gws contacts search "Jane" --format json | jq -r '.contacts[0].resource_name' | xargs gws contacts get
```

### create — Create a new contact

```bash
gws contacts create --name <name> [flags]
```

Creates a new contact with a name, email, and/or phone number.

**Flags:**
- `--name string` — Contact name (required)
- `--email string` — Contact email address (optional)
- `--phone string` — Contact phone number (optional)

**Output includes:**
- `status` — Always `"created"`
- `resource_name` — New contact's resource identifier
- All contact fields

**Examples:**
```bash
gws contacts create --name "Jane Smith"
gws contacts create --name "John Doe" --email "john@example.com"
gws contacts create --name "Bob Wilson" --email "bob@example.com" --phone "555-1234"
```

### delete — Delete a contact

```bash
gws contacts delete <resource-name>
```

Deletes a contact by resource name.

**Arguments:**
- `resource-name` — Resource identifier (required, e.g., `people/c1234567890`)

**Output includes:**
- `status` — Always `"deleted"`
- `resource_name` — Deleted contact's resource identifier

**Examples:**
```bash
gws contacts delete people/c1234567890
```

**Warning:** This operation is permanent and cannot be undone. Consider confirming before deletion:
```bash
gws contacts get people/c1234567890  # Review first
gws contacts delete people/c1234567890  # Then delete
```

## Output Modes

```bash
gws contacts list --format json    # Structured JSON (default)
gws contacts list --format yaml    # YAML format
gws contacts list --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws contacts list` or `search` to get resource names, then use those with `get` or `delete`
- Resource names have the format `people/c<numeric-id>` (e.g., `people/c1234567890`)
- The `list` command paginates automatically up to the `--max` limit (default 50)
- Search is more efficient than listing all contacts and filtering client-side
- When creating contacts, `--name` is required, but `--email` and `--phone` are optional
- Organization info is read-only (returned by list/get/search but not settable via create)
- Use `--quiet` on any command to suppress JSON output (useful for scripted actions)
- For bulk operations, pipe JSON output to `jq` for filtering and extracting resource names
