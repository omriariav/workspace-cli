# Contacts Commands Reference

Complete flag and option reference for `gws contacts` commands — 5 commands total.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

These flags apply to all `gws contacts` commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json`, `yaml`, or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws contacts list

Lists contacts from your Google account.

```
Usage: gws contacts list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 50 | Maximum number of contacts to return |

### Output Fields (JSON)

Returns an object with:
- `contacts` — Array of contact objects
- `count` — Number of contacts returned

Each contact includes:
- `resource_name` — Resource identifier (e.g., `people/c1234567890`)
- `name` — Contact's display name
- `emails` — Array of email addresses (if available)
- `phones` — Array of phone numbers (if available)
- `organization` — Object with `name` and `title` (if available)

### Examples

```bash
# List default 50 contacts
gws contacts list

# List up to 100 contacts
gws contacts list --max 100

# List contacts with text output
gws contacts list --format text

# Extract just names and emails
gws contacts list --format json | jq '.contacts[] | {name, emails}'

# Find contacts with email addresses
gws contacts list --format json | jq '.contacts[] | select(.emails != null)'
```

### Notes

- Pagination is handled automatically up to the `--max` limit
- Results are ordered by last modified date (most recent first)
- The API page size is capped at 1000 per request
- Organization info is read-only (from Google Workspace Directory if available)

---

## gws contacts search

Searches contacts by name, email, or phone number.

```
Usage: gws contacts search <query>
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `query` | string | Yes | Search string |

No additional flags.

### Output Fields (JSON)

Returns an object with:
- `contacts` — Array of matching contact objects (same fields as `list`)
- `count` — Number of results returned
- `query` — The search query used

### Search Behavior

The search is performed by the Google People API with these characteristics:
- Searches across: names, email addresses, phone numbers
- Case-insensitive matching
- Partial match support (e.g., "john" matches "John Doe" and "johnson@example.com")
- Returns contacts from both user's contacts and directory (if applicable)

### Examples

```bash
# Search by name
gws contacts search "John"

# Search by email
gws contacts search "john@example.com"

# Search by phone
gws contacts search "555-1234"

# Search by company
gws contacts search "Company Inc"

# Get resource name of first match
gws contacts search "Jane Smith" --format json | jq -r '.contacts[0].resource_name'
```

### Notes

- Search is more efficient than listing all contacts and filtering client-side
- Results are limited by the API (typically returns top 10 matches)
- For exact matches, use the full email address or full name
- Quote the query if it contains spaces

---

## gws contacts get

Gets detailed information about a specific contact by resource name.

```
Usage: gws contacts get <resource-name>
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `resource-name` | string | Yes | Resource identifier (e.g., `people/c1234567890`) |

No additional flags.

### Output Fields (JSON)

Returns a single contact object with:
- `resource_name` — Resource identifier
- `name` — Contact's display name
- `emails` — Array of email addresses (if available)
- `phones` — Array of phone numbers (if available)
- `organization` — Object with `name` and `title` (if available)

### Resource Name Format

Resource names follow the pattern: `people/c<numeric-id>`

Examples:
- `people/c1234567890`
- `people/c9876543210`

### Examples

```bash
# Get a specific contact
gws contacts get people/c1234567890

# Search and then get details
RESOURCE=$(gws contacts search "Jane" --format json | jq -r '.contacts[0].resource_name')
gws contacts get $RESOURCE

# Pipeline: search -> extract resource name -> get details
gws contacts search "Jane Smith" --format json | \
  jq -r '.contacts[0].resource_name' | \
  xargs gws contacts get
```

### Notes

- The resource name is required and must be exact
- Use `list` or `search` to find the resource name
- Invalid resource names return an error from the API

---

## gws contacts create

Creates a new contact with a name, email, and/or phone number.

```
Usage: gws contacts create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Contact name |
| `--email` | string | | No | Contact email address |
| `--phone` | string | | No | Contact phone number |

### Output Fields (JSON)

Returns the created contact object with:
- `status` — Always `"created"`
- `resource_name` — New contact's resource identifier
- `name` — Contact's display name
- `emails` — Array with the email (if provided)
- `phones` — Array with the phone (if provided)

### Examples

```bash
# Create contact with name only
gws contacts create --name "Jane Smith"

# Create contact with name and email
gws contacts create --name "John Doe" --email "john@example.com"

# Create contact with all fields
gws contacts create --name "Bob Wilson" --email "bob@example.com" --phone "555-1234"

# Create and capture resource name for further operations
RESOURCE=$(gws contacts create --name "Alice Brown" --email "alice@example.com" --format json | jq -r '.resource_name')
echo "Created: $RESOURCE"
```

### Notes

- The `--name` flag is required
- Email and phone are optional
- Currently supports one email and one phone per create operation
- Multiple emails/phones can be added via the Google Contacts web UI
- Organization info cannot be set via create (it's pulled from Google Workspace Directory)
- The API returns the full contact object including the new resource name

---

## gws contacts delete

Deletes a contact by resource name.

```
Usage: gws contacts delete <resource-name>
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `resource-name` | string | Yes | Resource identifier (e.g., `people/c1234567890`) |

No additional flags.

### Output Fields (JSON)

Returns:
- `status` — Always `"deleted"`
- `resource_name` — The deleted contact's resource identifier

### Examples

```bash
# Delete a contact
gws contacts delete people/c1234567890

# Review before deletion
gws contacts get people/c1234567890  # Review details first
gws contacts delete people/c1234567890  # Then delete

# Search, confirm, and delete
RESOURCE=$(gws contacts search "Old Contact" --format json | jq -r '.contacts[0].resource_name')
gws contacts get $RESOURCE  # Review
gws contacts delete $RESOURCE  # Delete

# Bulk delete with confirmation (careful!)
gws contacts search "test-contact" --format json | \
  jq -r '.contacts[].resource_name' | \
  while read resource; do
    echo "Deleting $resource"
    gws contacts delete $resource --quiet
  done
```

### Notes

- **This operation is permanent and cannot be undone**
- The resource name must be exact
- No confirmation prompt is shown — deletion happens immediately
- Use `get` to review the contact details before deleting
- The `--quiet` flag suppresses output (useful in loops)
- Invalid resource names return an error from the API

---

## Common Workflows

### Import Contacts from CSV

```bash
# Given a CSV with columns: name,email,phone
tail -n +2 contacts.csv | while IFS=, read name email phone; do
  gws contacts create --name "$name" --email "$email" --phone "$phone" --quiet
done
```

### Export Contacts to JSON

```bash
gws contacts list --max 1000 --format json > contacts.json
```

### Find Contacts Without Email

```bash
gws contacts list --format json | jq '.contacts[] | select(.emails == null)'
```

### Update Contact (Get, Delete, Recreate)

```bash
# Note: There's no direct update command, so update = delete + create
OLD=$(gws contacts search "Jane" --format json | jq -r '.contacts[0].resource_name')
gws contacts delete $OLD
gws contacts create --name "Jane Smith" --email "jane.new@example.com" --phone "555-9999"
```

### Merge Duplicate Contacts

```bash
# Find duplicates by name
DUPES=$(gws contacts list --format json | jq -r '.contacts | group_by(.name) | map(select(length > 1)) | .[][] | .resource_name')

# Review and manually delete duplicates
for resource in $DUPES; do
  gws contacts get $resource
done
```
