---
name: gws-docs
version: 1.1.0
description: "Google Docs CLI operations via gws. Use when users need to read, create, or edit Google Docs documents. Triggers: google docs, document, gdoc, word processing."
metadata:
  short-description: Google Docs CLI operations
  compatibility: claude-code, codex-cli
---

# Google Docs (gws docs)

`gws docs` provides CLI access to Google Docs with structured JSON output.

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
| Read a document | `gws docs read <doc-id>` |
| Read with formatting | `gws docs read <doc-id> --include-formatting` |
| Get document info | `gws docs info <doc-id>` |
| Create a document | `gws docs create --title "My Doc"` |
| Create with markdown | `gws docs create --title "My Doc" --text "# Heading\n\n**Bold**"` |
| Create with plain text | `gws docs create --title "My Doc" --text "Plain" --content-format plaintext` |
| Append text | `gws docs append <doc-id> --text "New paragraph"` |
| Insert text at position | `gws docs insert <doc-id> --text "Hello" --at 1` |
| Find and replace | `gws docs replace <doc-id> --find "old" --replace "new"` |
| Delete content | `gws docs delete <doc-id> --from 5 --to 10` |
| Add a table | `gws docs add-table <doc-id> --rows 3 --cols 4` |
| Format text | `gws docs format <doc-id> --from 1 --to 10 --bold` |
| Set paragraph style | `gws docs set-paragraph-style <doc-id> --from 1 --to 100 --alignment CENTER` |
| Add a list | `gws docs add-list <doc-id> --at 1 --type bullet --items "A;B;C"` |
| Remove list | `gws docs remove-list <doc-id> --from 1 --to 50` |

## Detailed Usage

### read — Read document content

```bash
gws docs read <document-id> [flags]
```

**Flags:**
- `--include-formatting` — Include formatting information and element positions

Use `--include-formatting` to see position indices needed for `insert`, `delete`, and `add-table`.

### info — Get document info

```bash
gws docs info <document-id>
```

Gets metadata about a Google Doc (title, ID, revision info).

### create — Create a new document

```bash
gws docs create --title <title> [flags]
```

**Flags:**
- `--title string` — Document title (required)
- `--text string` — Initial text content
- `--content-format string` — Content format: `markdown` (default), `plaintext`, or `richformat`

**Examples:**
```bash
gws docs create --title "Meeting Notes"
gws docs create --title "Report" --text "# Q4 Summary\n\n**Revenue** grew 15%"
gws docs create --title "Plain" --text "No formatting" --content-format plaintext
```

### append — Append text

```bash
gws docs append <document-id> --text <text> [flags]
```

**Flags:**
- `--text string` — Text to append (required)
- `--newline` — Add newline before appending (default: true)
- `--content-format string` — Content format: `markdown` (default), `plaintext`, or `richformat`

### insert — Insert text at position

```bash
gws docs insert <document-id> --text <text> [flags]
```

**Flags:**
- `--text string` — Text to insert (required)
- `--at int` — Position to insert at (1-based index, default: 1)
- `--content-format string` — Content format: `markdown` (default), `plaintext`, or `richformat`

Position 1 is the start of the document content. Use `gws docs read <id> --include-formatting` to see element positions.

### replace — Find and replace text

```bash
gws docs replace <document-id> --find <text> --replace <text> [flags]
```

**Flags:**
- `--find string` — Text to find (required)
- `--replace string` — Replacement text (required)
- `--match-case` — Case-sensitive matching (default: true)

### delete — Delete content

```bash
gws docs delete <document-id> --from <pos> --to <pos>
```

**Flags:**
- `--from int` — Start position (1-based index, required)
- `--to int` — End position (1-based index, required)

Use `gws docs read <id> --include-formatting` to see position indices.

### add-table — Add a table

```bash
gws docs add-table <document-id> [flags]
```

**Flags:**
- `--rows int` — Number of rows (default: 3)
- `--cols int` — Number of columns (default: 3)
- `--at int` — Position to insert at (1-based index, default: 1)

### format — Format text style

```bash
gws docs format <document-id> [flags]
```

**Flags:**
- `--from int` — Start position (1-based index, required)
- `--to int` — End position (1-based index, required)
- `--bold` — Make text bold
- `--italic` — Make text italic
- `--font-size int` — Font size in points
- `--color string` — Text color (hex, e.g., "#FF0000")

### set-paragraph-style — Set paragraph style

```bash
gws docs set-paragraph-style <document-id> [flags]
```

**Flags:**
- `--from int` — Start position (1-based index, required)
- `--to int` — End position (1-based index, required)
- `--alignment string` — Paragraph alignment: START, CENTER, END, JUSTIFIED
- `--line-spacing float` — Line spacing multiplier (e.g., 1.15, 1.5, 2.0)

### add-list — Add a bullet or numbered list

```bash
gws docs add-list <document-id> [flags]
```

**Flags:**
- `--at int` — Position to insert at (1-based index, default: 1)
- `--type string` — List type: bullet or numbered (default: bullet)
- `--items string` — List items separated by semicolons (required)

### remove-list — Remove list formatting

```bash
gws docs remove-list <document-id> [flags]
```

**Flags:**
- `--from int` — Start position (1-based index, required)
- `--to int` — End position (1-based index, required)

## Content Formats

The `--content-format` flag controls how `--text` input is handled for `create`, `append`, and `insert`.

| Format | Description |
|--------|-------------|
| `markdown` (default) | Text is inserted as-is with markdown syntax preserved. Select the text in Google Docs and use "Paste from Markdown" to apply formatting. |
| `plaintext` | Text is inserted as-is with no markdown syntax expected. |
| `richformat` | `--text` is parsed as a JSON array of Google Docs API `Request` objects, passed directly to `BatchUpdate`. |

**Markdown example:**
```bash
gws docs create --title "Report" --text "# Summary\n\n- Item **one**\n- Item *two*"
```

**Richformat example:**
```bash
gws docs create --title "Styled" --text '[{"insertText":{"location":{"index":1},"text":"Hello World"}},{"updateTextStyle":{"range":{"startIndex":1,"endIndex":6},"textStyle":{"bold":true},"fields":"bold"}}]' --content-format richformat
```

## Output Modes

```bash
gws docs read <doc-id> --format json    # Structured JSON (default)
gws docs read <doc-id> --format yaml    # YAML format
gws docs read <doc-id> --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- To get position indices for insert/delete/add-table, first run `gws docs read <id> --include-formatting`
- Positions are 1-based (1 = start of document content)
- `append` is the simplest way to add content — it adds to the end with an automatic newline
- `replace` replaces ALL occurrences in the document, not just the first one
- Document IDs can be extracted from Google Docs URLs: `docs.google.com/document/d/<ID>/edit`
- For comments on a doc, use `gws drive comments <doc-id>`
- Use `--quiet` on write operations to suppress JSON output in scripts
