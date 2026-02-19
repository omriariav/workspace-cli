---
name: gws-sheets
version: 1.1.0
description: "Google Sheets CLI operations via gws. Use when users need to read, write, or manage Google Sheets spreadsheets including cell values, rows, columns, sheets, sorting, merging, formatting, and find-replace. Triggers: sheets, spreadsheet, google sheets, cells, rows, columns, formulas, formatting."
metadata:
  short-description: Google Sheets CLI operations
  compatibility: claude-code, codex-cli
---

# Google Sheets (gws sheets)

`gws sheets` provides CLI access to Google Sheets with structured JSON output. This skill has 28 commands covering full spreadsheet management including batch operations, named ranges, and filters.

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

### Batch & Cross-Spreadsheet Operations
| Task | Command |
|------|---------|
| Read multiple ranges | `gws sheets batch-read <id> --ranges "A1:B5" --ranges "Sheet2!A1:C10"` |
| Write multiple ranges | `gws sheets batch-write <id> --ranges "A1:B2" --values '[[1,2],[3,4]]'` |
| Copy sheet to another | `gws sheets copy-to <id> --sheet-id 0 --destination <dest-id>` |

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

### Named Ranges
| Task | Command |
|------|---------|
| Add a named range | `gws sheets add-named-range <id> "Sheet1!A1:D10" --name "MyRange"` |
| List named ranges | `gws sheets list-named-ranges <id>` |
| Delete a named range | `gws sheets delete-named-range <id> --named-range-id "nr-123"` |

### Filters
| Task | Command |
|------|---------|
| Add a basic filter | `gws sheets add-filter <id> "Sheet1!A1:D10"` |
| Clear a basic filter | `gws sheets clear-filter <id> --sheet "Sheet1"` |
| Add a filter view | `gws sheets add-filter-view <id> "Sheet1!A1:D10" --name "My View"` |

### Cell Operations
| Task | Command |
|------|---------|
| Merge cells | `gws sheets merge <id> "Sheet1!A1:D4"` |
| Unmerge cells | `gws sheets unmerge <id> "Sheet1!A1:D4"` |
| Sort a range | `gws sheets sort <id> "A1:D10" --by B --desc` |
| Find and replace | `gws sheets find-replace <id> --find "old" --replace "new"` |
| Format cells | `gws sheets format <id> "A1:D10" --bold --bg-color "#FFFF00"` |
| Set column width | `gws sheets set-column-width <id> --sheet "Sheet1" --col A --width 200` |
| Set row height | `gws sheets set-row-height <id> --sheet "Sheet1" --row 1 --height 50` |
| Freeze panes | `gws sheets freeze <id> --sheet "Sheet1" --rows 1 --cols 1` |

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

### format — Format cells

```bash
gws sheets format <id> <range> [flags]
```

**Flags:**
- `--bold` — Make text bold
- `--italic` — Make text italic
- `--bg-color string` — Background color (hex, e.g., "#FFFF00")
- `--color string` — Text color (hex, e.g., "#FF0000")
- `--font-size int` — Font size in points

### set-column-width — Set column width

```bash
gws sheets set-column-width <id> --sheet <name> --col <letter> --width <pixels>
```

**Flags:**
- `--sheet string` — Sheet name (required)
- `--col string` — Column letter, e.g., A, B, AA (required)
- `--width int` — Column width in pixels (default: 100)

### set-row-height — Set row height

```bash
gws sheets set-row-height <id> --sheet <name> --row <number> --height <pixels>
```

**Flags:**
- `--sheet string` — Sheet name (required)
- `--row int` — Row number, 1-based (required)
- `--height int` — Row height in pixels (default: 21)

### freeze — Freeze rows and columns

```bash
gws sheets freeze <id> --sheet <name> --rows <n> --cols <n>
```

**Flags:**
- `--sheet string` — Sheet name (required)
- `--rows int` — Number of rows to freeze
- `--cols int` — Number of columns to freeze

### copy-to — Copy a sheet to another spreadsheet

```bash
gws sheets copy-to <spreadsheet-id> --sheet-id <id> --destination <dest-spreadsheet-id>
```

**Flags:**
- `--sheet-id int` — Source sheet ID to copy (required)
- `--destination string` — Destination spreadsheet ID (required)

### batch-read — Read multiple ranges

```bash
gws sheets batch-read <spreadsheet-id> --ranges "A1:B5" --ranges "Sheet2!A1:C10" [flags]
```

**Flags:**
- `--ranges strings` — Ranges to read (can be repeated, required)
- `--value-render string` — Value render option: `FORMATTED_VALUE`, `UNFORMATTED_VALUE`, `FORMULA` (default: "FORMATTED_VALUE")

### batch-write — Write to multiple ranges

```bash
gws sheets batch-write <spreadsheet-id> --ranges "A1:B2" --values '[[1,2],[3,4]]' [flags]
```

**Flags:**
- `--ranges strings` — Target ranges (can be repeated, pairs with `--values`, required)
- `--values strings` — JSON arrays of values (can be repeated, pairs with `--ranges`, required)
- `--value-input string` — Value input option: `RAW`, `USER_ENTERED` (default: "USER_ENTERED")

The nth `--ranges` pairs with the nth `--values`.

### add-named-range — Add a named range

```bash
gws sheets add-named-range <spreadsheet-id> <range> --name <name>
```

**Flags:**
- `--name string` — Name for the named range (required)

### list-named-ranges — List named ranges

```bash
gws sheets list-named-ranges <spreadsheet-id>
```

### delete-named-range — Delete a named range

```bash
gws sheets delete-named-range <spreadsheet-id> --named-range-id <id>
```

**Flags:**
- `--named-range-id string` — ID of the named range to delete (required)

Use `list-named-ranges` to find the IDs.

### add-filter — Add a basic filter

```bash
gws sheets add-filter <spreadsheet-id> <range>
```

Sets a basic filter on the specified range. Only one basic filter per sheet.

### clear-filter — Clear a basic filter

```bash
gws sheets clear-filter <spreadsheet-id> --sheet <name>
```

**Flags:**
- `--sheet string` — Sheet name (required)

### add-filter-view — Add a filter view

```bash
gws sheets add-filter-view <spreadsheet-id> <range> --name <title>
```

**Flags:**
- `--name string` — Title for the filter view (required)

Filter views are saved views that don't affect other users.

## Output Modes

```bash
gws sheets read <id> "A1:D10" --format json    # Structured JSON (default)
gws sheets read <id> "A1:D10" --format yaml    # YAML format
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
- Use `--quiet` on write/append/clear operations to suppress JSON output in scripts
