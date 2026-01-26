# Docs Commands Reference

Complete flag and option reference for `gws docs` commands.

> **Disclaimer:** This is an unofficial CLI tool, not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |

---

## gws docs read

Reads and displays the text content of a Google Doc.

```
Usage: gws docs read <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--include-formatting` | bool | false | Include formatting information and element positions |

### Output with `--include-formatting`

When enabled, the output includes position indices for each element. These indices are needed for `insert`, `delete`, and `add-table` commands.

---

## gws docs info

Gets metadata about a Google Doc.

```
Usage: gws docs info <document-id>
```

No additional flags.

### Output Fields (JSON)

- `documentId` — Document ID
- `title` — Document title
- `revisionId` — Current revision

---

## gws docs create

Creates a new Google Doc with optional initial content.

```
Usage: gws docs create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Document title |
| `--text` | string | | No | Initial text content |

---

## gws docs append

Appends text to the end of an existing Google Doc.

```
Usage: gws docs append <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--text` | string | | Yes | Text to append |
| `--newline` | bool | true | No | Add newline before appending |

---

## gws docs insert

Inserts text at a specific position in the document.

```
Usage: gws docs insert <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--text` | string | | Yes | Text to insert |
| `--at` | int | 1 | No | Position to insert at (1-based index) |

### Position System

- Positions are **1-based** (1 = start of document content)
- Use `gws docs read <id> --include-formatting` to see element positions
- Inserting at position 1 adds text at the very beginning of the document

---

## gws docs replace

Replaces all occurrences of a text string in the document.

```
Usage: gws docs replace <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--find` | string | | Yes | Text to find |
| `--replace` | string | | Yes | Replacement text |
| `--match-case` | bool | true | No | Case-sensitive matching |

Replaces **all** occurrences, not just the first.

---

## gws docs delete

Deletes content from a range of positions in the document.

```
Usage: gws docs delete <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--from` | int | | Yes | Start position (1-based index, inclusive) |
| `--to` | int | | Yes | End position (1-based index, exclusive) |

Use `gws docs read <id> --include-formatting` to determine correct positions.

---

## gws docs add-table

Adds a table at a specified position in the document.

```
Usage: gws docs add-table <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--rows` | int | 3 | Number of rows |
| `--cols` | int | 3 | Number of columns |
| `--at` | int | 1 | Position to insert at (1-based index) |

Use `gws docs read <id> --include-formatting` to determine correct positions.
