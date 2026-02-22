# Docs Commands Reference

Complete flag and option reference for `gws docs` commands — 41 commands total.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json`, `yaml`, or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

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
| `--content-format` | string | `markdown` | No | Content format: `markdown`, `plaintext`, or `richformat` |

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
| `--content-format` | string | `markdown` | No | Content format: `markdown`, `plaintext`, or `richformat` |

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
| `--content-format` | string | `markdown` | No | Content format: `markdown`, `plaintext`, or `richformat` |

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

---

## gws docs format

Applies text formatting to a range of positions in the document (v1.14.0).

```
Usage: gws docs format <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--from` | int | | Yes | Start position (1-based index) |
| `--to` | int | | Yes | End position (1-based index) |
| `--bold` | bool | false | No | Make text bold |
| `--italic` | bool | false | No | Make text italic |
| `--font-size` | int | 0 | No | Font size in points |
| `--color` | string | | No | Text color (hex, e.g., `#FF0000`) |

### Examples

```bash
# Make text bold
gws docs format 1abc123xyz --from 10 --to 50 --bold

# Make text italic and red
gws docs format 1abc123xyz --from 100 --to 150 --italic --color "#FF0000"

# Change font size
gws docs format 1abc123xyz --from 1 --to 20 --font-size 18

# Apply multiple styles
gws docs format 1abc123xyz --from 200 --to 250 --bold --italic --font-size 14 --color "#0000FF"
```

### Notes

- At least one formatting flag (`--bold`, `--italic`, `--font-size`, or `--color`) is required
- Color must be in hex format: `#RRGGBB` (e.g., `#FF0000` for red, `#0000FF` for blue)
- Font size is in points (typical sizes: 10, 11, 12, 14, 18, 24)
- Use `gws docs read <id> --include-formatting` to identify positions to format

---

## gws docs set-paragraph-style

Sets paragraph style properties for a range of positions (v1.14.0).

```
Usage: gws docs set-paragraph-style <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--from` | int | | Yes | Start position (1-based index) |
| `--to` | int | | Yes | End position (1-based index) |
| `--alignment` | string | | No | Paragraph alignment: `START`, `CENTER`, `END`, `JUSTIFIED` |
| `--line-spacing` | float | 0 | No | Line spacing multiplier (e.g., 1.15, 1.5, 2.0) |

### Alignment Values

| Value | Behavior |
|-------|----------|
| `START` | Left-aligned (default for LTR) |
| `CENTER` | Center-aligned |
| `END` | Right-aligned (default for RTL) |
| `JUSTIFIED` | Justified (flush left and right) |

### Examples

```bash
# Center-align a paragraph
gws docs set-paragraph-style 1abc123xyz --from 50 --to 150 --alignment CENTER

# Set double line spacing
gws docs set-paragraph-style 1abc123xyz --from 1 --to 500 --line-spacing 2.0

# Right-align with 1.5 line spacing
gws docs set-paragraph-style 1abc123xyz --from 200 --to 300 --alignment END --line-spacing 1.5

# Justify entire document
gws docs set-paragraph-style 1abc123xyz --from 1 --to 999999 --alignment JUSTIFIED
```

### Notes

- At least one style flag (`--alignment` or `--line-spacing`) is required
- Alignment values are case-insensitive
- Line spacing is a multiplier: 1.0 = single, 1.15 = default, 1.5 = 1.5x, 2.0 = double
- Use `gws docs read <id> --include-formatting` to identify paragraph positions

---

## gws docs add-list

Inserts text items as a bullet or numbered list at a specified position (v1.14.0).

```
Usage: gws docs add-list <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--at` | int | 1 | No | Position to insert at (1-based index) |
| `--type` | string | `bullet` | No | List type: `bullet` or `numbered` |
| `--items` | string | | Yes | List items separated by semicolons |

### Examples

```bash
# Add bullet list at beginning
gws docs add-list 1abc123xyz --items "First item;Second item;Third item"

# Add numbered list at specific position
gws docs add-list 1abc123xyz --at 200 --type numbered --items "Step one;Step two;Step three"

# Add single-item list
gws docs add-list 1abc123xyz --items "Single bullet point"
```

### Notes

- Items are separated by semicolons (`;`)
- Each item becomes a separate list entry
- Use `bullet` for unordered lists, `numbered` for ordered lists
- Position is 1-based; use `gws docs read <id> --include-formatting` to find positions

---

## gws docs remove-list

Removes bullet or numbered list formatting from a range (v1.14.0).

```
Usage: gws docs remove-list <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--from` | int | | Yes | Start position (1-based index) |
| `--to` | int | | Yes | End position (1-based index) |

### Examples

```bash
# Remove list formatting from a range
gws docs remove-list 1abc123xyz --from 100 --to 200

# Remove list formatting from entire document
gws docs remove-list 1abc123xyz --from 1 --to 999999
```

### Notes

- Removes list bullets/numbers but preserves the text content
- Use `gws docs read <id> --include-formatting` to identify list positions
- To convert between bullet and numbered lists, use `remove-list` followed by `add-list`

---

## gws docs trash

Moves a Google Doc to the trash via the Drive API.

```
Usage: gws docs trash <document-id> [flags]
```

### Flags

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--permanent` | bool | false | No | Permanently delete (skip trash) |

### Examples

```bash
# Move a document to trash
gws docs trash 1abc123xyz

# Permanently delete a document (cannot be undone)
gws docs trash 1abc123xyz --permanent
```

### Notes

- Default behavior moves the document to Drive trash (recoverable)
- `--permanent` bypasses trash and permanently deletes the document
- Uses the Drive API since the Docs API does not have a native delete endpoint

---

## Content Formats

The `--content-format` flag is available on `create`, `append`, and `insert` commands.

| Format | Behavior |
|--------|----------|
| `markdown` | Default. Text inserted as-is with markdown syntax. Select in Google Docs and use "Paste from Markdown" to format. |
| `plaintext` | Text inserted as-is. No markdown syntax expected. |
| `richformat` | `--text` parsed as JSON array of Google Docs API `Request` objects, sent directly to `BatchUpdate`. |

**Tip:** With `richformat`, the `--text` value must be a valid JSON array of [Google Docs API Request](https://developers.google.com/docs/api/reference/rest/v1/documents/request) objects. The `--newline` flag is ignored in `richformat` mode for `append`.

---

## Tab Support

All subcommands inherit the persistent `--tab` flag from the `docs` parent command. Use `--tab` to target a specific tab by ID or title.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--tab` | string | | Tab ID or title to target (omit for first tab) |

The `read` command also supports `--tab-index` for zero-based index access.

---

## gws docs add-tab

Adds a new tab to the document.

```
Usage: gws docs add-tab <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Tab title |
| `--index` | int | -1 | No | Position index for the new tab |

---

## gws docs delete-tab

Deletes a tab from the document.

```
Usage: gws docs delete-tab <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--tab-id` | string | | Yes | Tab ID to delete |

---

## gws docs rename-tab

Renames a tab in the document.

```
Usage: gws docs rename-tab <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--tab-id` | string | | Yes | Tab ID to rename |
| `--title` | string | | Yes | New tab title |

---

## gws docs add-image

Inserts an inline image into the document.

```
Usage: gws docs add-image <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--uri` | string | | Yes | Image URI |
| `--at` | int | 1 | No | Position to insert at (1-based index) |
| `--width` | float | 0 | No | Image width in points |
| `--height` | float | 0 | No | Image height in points |

---

## gws docs insert-table-row

Inserts a row into a table.

```
Usage: gws docs insert-table-row <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |
| `--below` | bool | true | No | Insert below the reference cell |

---

## gws docs delete-table-row

Deletes a row from a table.

```
Usage: gws docs delete-table-row <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |

---

## gws docs insert-table-col

Inserts a column into a table.

```
Usage: gws docs insert-table-col <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |
| `--right` | bool | true | No | Insert to the right of the reference cell |

---

## gws docs delete-table-col

Deletes a column from a table.

```
Usage: gws docs delete-table-col <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |

---

## gws docs merge-cells

Merges table cells.

```
Usage: gws docs merge-cells <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |
| `--row-span` | int | 1 | No | Number of rows to merge |
| `--col-span` | int | 1 | No | Number of columns to merge |

---

## gws docs unmerge-cells

Unmerges table cells.

```
Usage: gws docs unmerge-cells <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |
| `--row-span` | int | 1 | No | Number of rows to unmerge |
| `--col-span` | int | 1 | No | Number of columns to unmerge |

---

## gws docs pin-rows

Pins header rows in a table.

```
Usage: gws docs pin-rows <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--count` | int | | Yes | Number of rows to pin |

---

## gws docs page-break

Inserts a page break.

```
Usage: gws docs page-break <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--at` | int | 1 | Position to insert at (1-based index) |

---

## gws docs section-break

Inserts a section break.

```
Usage: gws docs section-break <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--at` | int | 1 | Position to insert at (1-based index) |
| `--type` | string | NEXT_PAGE | Section break type: NEXT_PAGE or CONTINUOUS |

---

## gws docs add-header

Adds a header to the document.

```
Usage: gws docs add-header <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | DEFAULT | Header type |

---

## gws docs delete-header

Deletes a header from the document.

```
Usage: gws docs delete-header <document-id> <header-id>
```

No additional flags. Header ID is passed as a positional argument.

---

## gws docs add-footer

Adds a footer to the document.

```
Usage: gws docs add-footer <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | DEFAULT | Footer type |

---

## gws docs delete-footer

Deletes a footer from the document.

```
Usage: gws docs delete-footer <document-id> <footer-id>
```

No additional flags. Footer ID is passed as a positional argument.

---

## gws docs add-named-range

Creates a named range in the document.

```
Usage: gws docs add-named-range <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Range name |
| `--from` | int | | Yes | Start position |
| `--to` | int | | Yes | End position |

---

## gws docs delete-named-range

Deletes a named range.

```
Usage: gws docs delete-named-range <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | No | Named range name (mutually exclusive with --id) |
| `--id` | string | | No | Named range ID (mutually exclusive with --name) |

One of `--name` or `--id` is required.

---

## gws docs add-footnote

Inserts a footnote at a position.

```
Usage: gws docs add-footnote <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--at` | int | | Yes | Insertion index |

---

## gws docs delete-object

Deletes a positioned object from the document.

```
Usage: gws docs delete-object <document-id> <object-id>
```

No additional flags. Object ID is passed as a positional argument.

---

## gws docs replace-image

Replaces an inline image with a new one.

```
Usage: gws docs replace-image <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Inline object ID |
| `--uri` | string | | Yes | New image URI |

---

## gws docs replace-named-range

Replaces text content in a named range.

```
Usage: gws docs replace-named-range <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | No | Named range name (mutually exclusive with --id) |
| `--id` | string | | No | Named range ID (mutually exclusive with --name) |
| `--text` | string | | Yes | Replacement text |

---

## gws docs update-style

Updates document-level style properties (margins).

```
Usage: gws docs update-style <document-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--margin-top` | float | -1 | Top margin in points |
| `--margin-bottom` | float | -1 | Bottom margin in points |
| `--margin-left` | float | -1 | Left margin in points |
| `--margin-right` | float | -1 | Right margin in points |

At least one margin flag must be specified.

---

## gws docs update-section-style

Updates section style properties.

```
Usage: gws docs update-section-style <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--from` | int | | Yes | Start position |
| `--to` | int | | Yes | End position |
| `--column-count` | int | 0 | No | Number of columns |
| `--content-direction` | string | | No | Content direction: LEFT_TO_RIGHT or RIGHT_TO_LEFT |

---

## gws docs update-table-cell-style

Updates table cell style properties.

```
Usage: gws docs update-table-cell-style <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--col` | int | | Yes | Zero-based column index |
| `--row-span` | int | 1 | No | Number of rows |
| `--col-span` | int | 1 | No | Number of columns |
| `--bg-color` | string | | No | Background color (#RRGGBB) |
| `--padding` | float | -1 | No | Cell padding in points |

---

## gws docs update-table-col-properties

Updates table column width.

```
Usage: gws docs update-table-col-properties <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--col-index` | int | | Yes | Column index |
| `--width` | float | | Yes | Column width in points |

---

## gws docs update-table-row-style

Updates table row style.

```
Usage: gws docs update-table-row-style <document-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-start` | int | | Yes | Table start index |
| `--row` | int | | Yes | Zero-based row index |
| `--min-height` | float | | Yes | Minimum row height in points |
