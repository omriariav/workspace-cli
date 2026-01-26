---
name: gws-sheets
version: 1.0.0
description: "Google Sheets CLI operations via gws. Use when users need to read, write, or manage Google Sheets spreadsheets including cell values, rows, columns, sheets, sorting, merging, and find-replace. Triggers: sheets, spreadsheet, google sheets, cells, rows, columns, formulas."
metadata:
  short-description: Google Sheets CLI operations
  compatibility: claude-code, codex-cli
---

# Google Sheets (gws sheets)

`gws sheets` provides CLI access to Google Sheets with structured JSON output. This is the largest skill with 19 commands covering full spreadsheet management.

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

### Reading Data
| Task | Command |
|------|---------|
| Get spreadsheet info | `gws sheets info <id>` |
| List sheets | `gws sheets list <id>` |
| Read a range | `gws sheets read <id> "Sheet1!A1:D10"` |
| Read entire sheet | `gws sheets read <id> "Sheet1"` |

### Writing Data
| Task | Command |
|------|---------|
| Create spreadsheet | `gws sheets create --title "My Sheet"` |
| Write to cells | `gws sheets write <id> "Sheet1!A1" --values "a,b,c"` |
| Write multiple rows | `gws sheets write <id> "A1" --values "a,b,c;d,e,f"` |
| Write JSON data | `gws sheets write <id> "A1" --values-json '[["a","b"],["c","d"]]'` |
| Append rows | `gws sheets append <id> "Sheet1" --values "x,y,z"` |
| Clear cells | `gws sheets clear <id> "Sheet1!A1:D10"` |

### Sheet Management
| Task | Command |
|------|---------|
| Add a sheet | `gws sheets add-sheet <id> --name "New Sheet"` |
| Delete a sheet | `gws sheets delete-sheet <id> --name "Old Sheet"` |
| Rename a sheet | `gws sheets rename-sheet <id> --sheet "Old" --name "New"` |
| Duplicate a sheet | `gws sheets duplicate-sheet <id> --sheet "Template"` |

### Row/Column Operations
| Task | Command |
|------|---------|
| Insert rows | `gws sheets insert-rows <id> --sheet "Sheet1" --at 5 --count 3` |
| Delete rows | `gws sheets delete-rows <id> --sheet "Sheet1" --from 5 --to 8` |
| Insert columns | `gws sheets insert-cols <id> --sheet "Sheet1" --at 2 --count 1` |
| Delete columns | `gws sheets delete-cols <id> --sheet "Sheet1" --from 2 --to 4` |

### Cell Operations
| Task | Command |
|------|---------|
| Merge cells | `gws sheets merge <id> "Sheet1!A1:D4"` |
| Unmerge cells | `gws sheets unmerge <id> "Sheet1!A1:D4"` |
| Sort a range | `gws sheets sort <id> "A1:D10" --by B --desc` |
| Find and replace | `gws sheets find-replace <id> --find "old" --replace "new"` |

## Detailed Usage

### info — Get spreadsheet info

```bash
gws sheets info <spreadsheet-id>
```

### list — List sheets

```bash
gws sheets list <spreadsheet-id>
```

### read — Read cell values

```bash
gws sheets read <spreadsheet-id> <range> [flags]
```

**Flags:**
- `--headers` — Treat first row as headers for JSON output (default: true)
- `--output-format string` — Output format: `json` or `csv` (default: "json")

**Range format:**
- `Sheet1!A1:D10` — Specific range in Sheet1
- `Sheet1!A:D` — Columns A through D
- `Sheet1` — All data in Sheet1
- `A1:D10` — Range in first sheet

### create — Create a spreadsheet

```bash
gws sheets create --title <title> [flags]
```

**Flags:**
- `--title string` — Spreadsheet title (required)
- `--sheet-names strings` — Sheet names (comma-separated, default: Sheet1)

### write — Write values to cells

```bash
gws sheets write <spreadsheet-id> <range> [flags]
```

**Flags:**
- `--values string` — Values (comma-separated, semicolon for rows)
- `--values-json string` — Values as JSON array

**Examples:**
```bash
gws sheets write <id> "Sheet1!A1" --values "Hello"
gws sheets write <id> "A1:C1" --values "Name,Age,City"
gws sheets write <id> "A1:C2" --values "Name,Age,City;Alice,30,NYC"
gws sheets write <id> "A1" --values-json '[["Name","Age"],["Alice",30]]'
```

### append — Append rows

```bash
gws sheets append <spreadsheet-id> <range> [flags]
```

Appends rows after the last row with data. The range identifies the table to append to.

**Flags:**
- `--values string` — Values (comma-separated, semicolon for rows)
- `--values-json string` — Values as JSON array

### add-sheet / delete-sheet / rename-sheet / duplicate-sheet

```bash
gws sheets add-sheet <id> --name "New Sheet" [--rows 1000] [--cols 26]
gws sheets delete-sheet <id> --name "Sheet Name"     # or --sheet-id 123
gws sheets rename-sheet <id> --sheet "Current" --name "New Name"
gws sheets duplicate-sheet <id> --sheet "Template" [--new-name "Copy"]
```

### insert-rows / delete-rows / insert-cols / delete-cols

```bash
gws sheets insert-rows <id> --sheet "Sheet1" --at 5 --count 3
gws sheets delete-rows <id> --sheet "Sheet1" --from 5 --to 8
gws sheets insert-cols <id> --sheet "Sheet1" --at 2 --count 1
gws sheets delete-cols <id> --sheet "Sheet1" --from 2 --to 4
```

Row/column indices are **0-based**. For delete, `--from` is inclusive and `--to` is exclusive.

### merge / unmerge

```bash
gws sheets merge <id> "Sheet1!A1:D4"
gws sheets unmerge <id> "Sheet1!A1:D4"
```

Unbounded ranges (`A:A`, `1:1`) are not supported for merge/unmerge.

### sort — Sort a range

```bash
gws sheets sort <id> <range> [flags]
```

**Flags:**
- `--by string` — Column to sort by (e.g., "A", "B", "C") (default: "A")
- `--desc` — Sort in descending order
- `--has-header` — First row is a header (excluded from sort)

### find-replace — Find and replace

```bash
gws sheets find-replace <id> --find <text> --replace <text> [flags]
```

**Flags:**
- `--find string` — Text to find (required)
- `--replace string` — Replacement text (required)
- `--sheet string` — Limit to specific sheet (optional)
- `--match-case` — Case-sensitive matching
- `--entire-cell` — Match entire cell contents only

## Output Modes

```bash
gws sheets read <id> "A1:D10" --format json    # Structured JSON (default)
gws sheets read <id> "A1:D10" --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Always specify the sheet name in ranges for multi-sheet spreadsheets (e.g., `Sheet1!A1:D10`)
- Use `gws sheets list <id>` to discover sheet names before operating on them
- Row/column indices for insert/delete operations are **0-based**
- For delete operations, `--from` is inclusive and `--to` is exclusive
- Use `--values-json` for data that might contain commas or semicolons
- `append` finds the last row with data and adds below it — great for log-style data
- `read --headers` (default true) uses the first row as JSON keys — disable with `--headers=false` for raw arrays
- Spreadsheet IDs can be extracted from URLs: `docs.google.com/spreadsheets/d/<ID>/edit`
- Unbounded ranges (`A:A`, `1:1`) are not supported for merge/unmerge/sort
