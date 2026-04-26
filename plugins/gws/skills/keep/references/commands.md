# Keep Commands Reference

Complete flag and option reference for `gws keep` commands -- 3 commands total.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

These flags apply to all `gws keep` commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json`, `yaml`, or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws keep list

Lists notes from Google Keep.

```
Usage: gws keep list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 20 | Maximum number of notes to return |

### Output Fields (JSON)

Returns an object with:
- `notes` -- Array of note objects
- `count` -- Number of notes returned

Each note includes:
- `name` -- Note resource name (e.g., `notes/abc123`)
- `title` -- Note title
- `text` -- Note text content (only present if the note has a text body)
- `create_time` -- Creation timestamp (only present if available)
- `update_time` -- Last update timestamp (only present if available)
- `trashed` -- Boolean, only present if the note is trashed

### Examples

```bash
# List default 20 notes
gws keep list

# List up to 50 notes
gws keep list --max 50

# List notes with text output
gws keep list --format text

# Extract just titles
gws keep list --format json | jq '.notes[] | .title'

# Find notes by title keyword
gws keep list --max 100 --format json | jq '.notes[] | select(.title | test("meeting"; "i"))'

# Get IDs of all notes
gws keep list --format json | jq -r '.notes[].name'
```

### Notes

- Requires Keep API enabled in the Google Cloud project
- Requires Google Workspace Enterprise plan
- Default page size is 20; use `--max` to increase
- Results include both active and trashed notes (check the `trashed` field)

---

## gws keep get

Gets a specific note from Google Keep by its ID.

```
Usage: gws keep get <note-id>
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `note-id` | string | Yes | Note identifier (e.g., `notes/abc123` or `abc123`) |

No additional flags.

### Output Fields (JSON)

Returns a single note object with:
- `name` -- Note resource name (e.g., `notes/abc123`)
- `title` -- Note title
- `text` -- Note text content (only present if the note has a text body)
- `create_time` -- Creation timestamp (only present if available)
- `update_time` -- Last update timestamp (only present if available)
- `trashed` -- Boolean, only present if the note is trashed

### Note ID Format

Note IDs follow the pattern: `notes/<alphanumeric-id>`

The `notes/` prefix is optional -- the CLI automatically prepends it if missing:
- `notes/abc123` and `abc123` are both valid inputs

### Examples

```bash
# Get a specific note by full resource name
gws keep get notes/abc123

# Get a note by short ID (notes/ prefix added automatically)
gws keep get abc123

# Pipeline: list notes and get the first one
gws keep list --format json | jq -r '.notes[0].name' | xargs gws keep get
```

### Notes

- The note ID is a required positional argument
- If the note does not exist or has been permanently deleted, an error is returned
- The `notes/` prefix is automatically added if not provided

---

## gws keep create

Creates a new note in Google Keep.

```
Usage: gws keep create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Note title |
| `--text` | string | | Yes | Note text content |

### Output Fields (JSON)

Returns the created note object with:
- `name` -- New note's resource name (e.g., `notes/abc123`)
- `title` -- Note title
- `text` -- Note text content
- `create_time` -- Creation timestamp
- `update_time` -- Last update timestamp

### Examples

```bash
# Create a simple note
gws keep create --title "Shopping List" --text "Milk, eggs, bread"

# Create meeting notes
gws keep create --title "Meeting Notes" --text "Discuss Q1 goals"

# Create a reminder
gws keep create --title "Reminder" --text "Call dentist at 3pm"

# Create and capture the note ID
NOTE_ID=$(gws keep create --title "New Note" --text "Content here" --format json | jq -r '.name')
echo "Created: $NOTE_ID"
```

### Notes

- Both `--title` and `--text` are required flags
- The note body is created as a single text section
- The API returns the full note object including the assigned resource name
- Currently only text notes are supported (no list/checkbox notes via CLI)

---

## Common Workflows

### Quick Note Capture

```bash
# Create a note from a one-liner
gws keep create --title "Quick Note" --text "Remember to review PR #42"
```

### Export Notes to JSON

```bash
# Export all notes
gws keep list --max 100 --format json > notes.json
```

### Search Notes by Title

```bash
# List notes and filter by title
gws keep list --max 100 --format json | jq '.notes[] | select(.title | test("shopping"; "i"))'
```

### Get Full Content of Recent Notes

```bash
# List note IDs, then get each one
gws keep list --max 5 --format json | jq -r '.notes[].name' | while read note; do
  echo "=== $note ==="
  gws keep get "$note" --format text
done
```
