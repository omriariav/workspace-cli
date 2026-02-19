# Contacts Commands Reference

Complete flag and option reference for `gws contacts` commands — 14 commands total.

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

## gws contacts update

Updates an existing contact by resource name. Specify fields to update via flags.

```
Usage: gws contacts update <resource-name> [flags]
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `resource-name` | string | Yes | Resource identifier (e.g., `people/c1234567890`) |

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | No | Updated contact name |
| `--email` | string | | No | Updated email address |
| `--phone` | string | | No | Updated phone number |
| `--organization` | string | | No | Updated organization name |
| `--title` | string | | No | Updated job title |
| `--etag` | string | | No | Etag for concurrency control (from get command) |

### Output Fields (JSON)

Returns the updated contact object with:
- `status` — Always `"updated"`
- `resource_name` — Contact's resource identifier
- `etag` — New etag after update
- All standard contact fields

### Examples

```bash
# Update contact name
gws contacts update people/c1234567890 --name "Jane Doe"

# Update email and phone
gws contacts update people/c1234567890 --email "new@example.com" --phone "555-9999"

# Update organization info
gws contacts update people/c1234567890 --organization "Acme Inc" --title "Manager"

# Update with etag for concurrency control
ETAG=$(gws contacts get people/c1234567890 --format json | jq -r '.etag')
gws contacts update people/c1234567890 --name "Updated Name" --etag "$ETAG"
```

### Notes

- At least one field to update must be specified
- The `updatePersonFields` mask is automatically derived from the flags provided
- Use `--etag` for optimistic concurrency control — the update will fail if the contact was modified since the etag was fetched
- Only specified fields are updated; unspecified fields remain unchanged

---

## gws contacts batch-create

Creates multiple contacts from a JSON file.

```
Usage: gws contacts batch-create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file` | string | | Yes | Path to JSON file with contacts array |

### File Format

The JSON file should contain an array of Person objects:

```json
[
  {
    "names": [{"unstructuredName": "John Doe"}],
    "emailAddresses": [{"value": "john@example.com"}]
  },
  {
    "names": [{"unstructuredName": "Jane Smith"}],
    "phoneNumbers": [{"value": "555-1234"}]
  }
]
```

### Output Fields (JSON)

Returns:
- `status` — Always `"created"`
- `contacts` — Array of created contact objects
- `count` — Number of contacts created

### Examples

```bash
# Batch create contacts from file
gws contacts batch-create --file contacts.json
```

### Notes

- Allows up to 200 contacts in a single request
- Each contact must have at least a name

---

## gws contacts batch-update

Updates multiple contacts from a JSON file.

```
Usage: gws contacts batch-update [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file` | string | | Yes | Path to JSON file with contacts map |

### File Format

The JSON file should contain a map of resource names to Person objects plus an update mask:

```json
{
  "contacts": {
    "people/c123": {
      "etag": "abc",
      "names": [{"unstructuredName": "Updated Name"}]
    },
    "people/c456": {
      "etag": "def",
      "emailAddresses": [{"value": "new@example.com"}]
    }
  },
  "update_mask": "names,emailAddresses"
}
```

### Output Fields (JSON)

Returns:
- `status` — Always `"updated"`
- `results` — Map of resource names to updated contact objects
- `count` — Number of contacts updated

### Examples

```bash
# Batch update contacts from file
gws contacts batch-update --file updates.json
```

### Notes

- Allows up to 200 contacts in a single request
- The `update_mask` field is required
- Each contact should include its etag for concurrency control

---

## gws contacts batch-delete

Deletes multiple contacts by resource names.

```
Usage: gws contacts batch-delete [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--resources` | string[] | | Yes | Resource names to delete (repeatable) |

### Output Fields (JSON)

Returns:
- `status` — Always `"deleted"`
- `resource_names` — Array of deleted resource names
- `count` — Number of contacts deleted

### Examples

```bash
# Delete multiple contacts
gws contacts batch-delete --resources people/c1 --resources people/c2

# Delete contacts from search results
gws contacts search "test" --format json | \
  jq -r '.contacts[].resource_name' | \
  xargs -I {} echo "--resources {}" | \
  xargs gws contacts batch-delete
```

### Notes

- **This operation is permanent and cannot be undone**
- Allows up to 500 resource names in a single request
- Use `--resources` flag repeatedly for each resource name

---

## gws contacts directory

Lists people in the organization's directory.

```
Usage: gws contacts directory [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 50 | Maximum number of directory people to return |
| `--query` | string | | Filter directory results |

### Output Fields (JSON)

Returns an object with:
- `contacts` — Array of directory people objects
- `count` — Number of people returned

### Examples

```bash
# List directory people
gws contacts directory

# List more directory people
gws contacts directory --max 200
```

### Notes

- Requires the `directory.readonly` scope
- Only available for Google Workspace accounts (not personal Gmail)
- Returns people from the organization's domain directory

---

## gws contacts directory-search

Searches people in the organization's directory by query.

```
Usage: gws contacts directory-search [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--query` | string | | Yes | Search query |
| `--max` | int | 50 | No | Maximum number of results to return |

### Output Fields (JSON)

Returns an object with:
- `contacts` — Array of matching directory people objects
- `count` — Number of results returned
- `query` — The search query used

### Examples

```bash
# Search directory by name
gws contacts directory-search --query "John"

# Search directory with more results
gws contacts directory-search --query "engineering" --max 100
```

### Notes

- Requires the `directory.readonly` scope
- Only available for Google Workspace accounts
- Searches across names and email addresses in the directory

---

## gws contacts photo

Updates a contact's photo from an image file.

```
Usage: gws contacts photo <resource-name> [flags]
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `resource-name` | string | Yes | Resource identifier (e.g., `people/c1234567890`) |

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--file` | string | | Yes | Path to image file, JPEG or PNG |

### Output Fields (JSON)

Returns:
- `status` — Always `"photo_updated"`
- `resource_name` — Contact's resource identifier
- All standard contact fields (if available)

### Examples

```bash
# Update contact photo
gws contacts photo people/c1234567890 --file photo.jpg
gws contacts photo people/c1234567890 --file avatar.png
```

### Notes

- Only JPEG and PNG formats are supported
- The image is base64-encoded before sending to the API
- Large images may be resized by the API

---

## gws contacts delete-photo

Deletes a contact's photo by resource name.

```
Usage: gws contacts delete-photo <resource-name>
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `resource-name` | string | Yes | Resource identifier (e.g., `people/c1234567890`) |

No additional flags.

### Output Fields (JSON)

Returns:
- `status` — Always `"photo_deleted"`
- `resource_name` — Contact's resource identifier

### Examples

```bash
# Delete contact photo
gws contacts delete-photo people/c1234567890
```

---

## gws contacts resolve

Gets multiple contacts by their resource names in a single batch request.

```
Usage: gws contacts resolve [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--ids` | string[] | | Yes | Resource names to resolve (repeatable) |

### Output Fields (JSON)

Returns an object with:
- `contacts` — Array of resolved contact objects
- `count` — Number of contacts resolved

### Examples

```bash
# Resolve multiple contacts
gws contacts resolve --ids people/c1 --ids people/c2

# Resolve contacts from a list
gws contacts resolve --ids people/c1234567890 --ids people/c9876543210
```

### Notes

- More efficient than making individual `get` calls for multiple contacts
- Uses `people.getBatchGet` API method
- Returns contacts in the order they were requested

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

### Update Contact

```bash
# Direct update using the update command
gws contacts update people/c1234567890 --name "Jane Smith" --email "jane.new@example.com" --phone "555-9999"

# Search and update
RESOURCE=$(gws contacts search "Jane" --format json | jq -r '.contacts[0].resource_name')
gws contacts update $RESOURCE --name "Jane Smith-Doe"
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
