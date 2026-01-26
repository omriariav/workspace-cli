# Drive Commands Reference

Complete flag and option reference for `gws drive` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |

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

The query is a full-text search across file names and content.

---

## gws drive info

Gets detailed information about a file.

```
Usage: gws drive info <file-id>
```

No additional flags.

### Output Fields (JSON)

- `id` — File ID
- `name` — File name
- `mimeType` — MIME type
- `size` — File size in bytes
- `createdTime` — Creation time
- `modifiedTime` — Last modified time
- `owners` — File owners
- `webViewLink` — Link to view in browser

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

### MIME Type Auto-Detection

The MIME type is auto-detected from the file extension. Override with `--mime-type` if needed.

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

Use `root` as the folder ID to move to the root of Drive.

---

## gws drive delete

Deletes a file from Google Drive.

```
Usage: gws drive delete <file-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--permanent` | bool | false | Permanently delete (skip trash) |

By default, files are moved to trash (recoverable for 30 days). Use `--permanent` only when explicitly intended.

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

### Notes

- Works on any Drive file type: Docs, Sheets, Slides, PDFs, etc.
- Resolved comments are excluded by default
- When filtering resolved comments, the actual result count may be less than `--max` since filtering happens after fetching from the API
- Each comment includes its replies
