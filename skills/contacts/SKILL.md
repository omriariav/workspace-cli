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
| Update a contact | `gws contacts update <resource-name> --name "New Name"` |
| Delete a contact | `gws contacts delete <resource-name>` |
| Batch create contacts | `gws contacts batch-create --file contacts.json` |
| Batch update contacts | `gws contacts batch-update --file updates.json` |
| Batch delete contacts | `gws contacts batch-delete --resources people/c1 --resources people/c2` |
| List directory people | `gws contacts directory` |
| Search directory | `gws contacts directory-search --query "John"` |
| Update contact photo | `gws contacts photo <resource-name> --file photo.jpg` |
| Delete contact photo | `gws contacts delete-photo <resource-name>` |
| Resolve multiple contacts | `gws contacts resolve --ids people/c1 --ids people/c2` |

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

### update — Update a contact

```bash
gws contacts update <resource-name> [flags]
```

Updates an existing contact by resource name. Specify fields to update via flags.

**Arguments:**
- `resource-name` — Resource identifier (required, e.g., `people/c1234567890`)

**Flags:**
- `--name string` — Updated contact name
- `--email string` — Updated email address
- `--phone string` — Updated phone number
- `--organization string` — Updated organization name
- `--title string` — Updated job title
- `--etag string` — Etag for concurrency control (from get command)

**Examples:**
```bash
gws contacts update people/c1234567890 --name "Jane Doe"
gws contacts update people/c1234567890 --email "new@example.com" --phone "555-9999"
gws contacts update people/c1234567890 --organization "Acme Inc" --title "Manager"
```

### batch-create — Batch create contacts

```bash
gws contacts batch-create --file <path>
```

Creates multiple contacts from a JSON file. The file should contain an array of contact objects.

**Flags:**
- `--file string` — Path to JSON file with contacts array (required)

**File format:**
```json
[
  {"names": [{"unstructuredName": "John Doe"}], "emailAddresses": [{"value": "john@example.com"}]},
  {"names": [{"unstructuredName": "Jane Smith"}], "phoneNumbers": [{"value": "555-1234"}]}
]
```

**Examples:**
```bash
gws contacts batch-create --file contacts.json
```

### batch-update — Batch update contacts

```bash
gws contacts batch-update --file <path>
```

Updates multiple contacts from a JSON file. The file should contain a map of resource names to contact objects.

**Flags:**
- `--file string` — Path to JSON file with contacts map (required)

**File format:**
```json
{
  "contacts": {
    "people/c123": {"etag": "...", "names": [{"unstructuredName": "Updated Name"}]},
    "people/c456": {"etag": "...", "emailAddresses": [{"value": "new@example.com"}]}
  },
  "update_mask": "names,emailAddresses"
}
```

**Examples:**
```bash
gws contacts batch-update --file updates.json
```

### batch-delete — Batch delete contacts

```bash
gws contacts batch-delete --resources <resource-name> [--resources <resource-name> ...]
```

Deletes multiple contacts by resource names.

**Flags:**
- `--resources string` — Resource names to delete (repeatable)

**Examples:**
```bash
gws contacts batch-delete --resources people/c1 --resources people/c2
```

### directory — List directory people

```bash
gws contacts directory [flags]
```

Lists people in the organization's directory. Requires directory.readonly scope.

**Flags:**
- `--max int` — Maximum number of directory people to return (default 50)
- `--query string` — Filter directory results

**Examples:**
```bash
gws contacts directory
gws contacts directory --max 100
```

### directory-search — Search directory people

```bash
gws contacts directory-search --query <query> [flags]
```

Searches people in the organization's directory by query.

**Flags:**
- `--query string` — Search query (required)
- `--max int` — Maximum number of results to return (default 50)

**Examples:**
```bash
gws contacts directory-search --query "John"
gws contacts directory-search --query "engineering" --max 100
```

### photo — Update contact photo

```bash
gws contacts photo <resource-name> --file <path>
```

Updates a contact's photo from an image file (JPEG or PNG).

**Arguments:**
- `resource-name` — Resource identifier (required)

**Flags:**
- `--file string` — Path to image file, JPEG or PNG (required)

**Examples:**
```bash
gws contacts photo people/c1234567890 --file photo.jpg
gws contacts photo people/c1234567890 --file avatar.png
```

### delete-photo — Delete contact photo

```bash
gws contacts delete-photo <resource-name>
```

Deletes a contact's photo by resource name.

**Arguments:**
- `resource-name` — Resource identifier (required)

**Examples:**
```bash
gws contacts delete-photo people/c1234567890
```

### resolve — Resolve multiple contacts

```bash
gws contacts resolve --ids <resource-name> [--ids <resource-name> ...]
```

Gets multiple contacts by their resource names in a single batch request.

**Flags:**
- `--ids string` — Resource names to resolve (repeatable)

**Examples:**
```bash
gws contacts resolve --ids people/c1 --ids people/c2
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
- Organization info can be set via `update` command (`--organization`, `--title` flags)
- Use `--quiet` on any command to suppress JSON output (useful for scripted actions)
- For bulk operations, use `batch-create`, `batch-update`, `batch-delete` for better efficiency
- Use `resolve` to fetch multiple contacts in one request instead of multiple `get` calls
- Directory commands (`directory`, `directory-search`) require Google Workspace and `directory.readonly` scope
