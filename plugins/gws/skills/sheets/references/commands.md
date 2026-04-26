# Sheets Commands Reference

Complete flag and option reference for `gws sheets` commands — 38 commands total.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json`, `yaml`, or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

## Range Format Reference

Ranges are used by `read`, `write`, `append`, `clear`, `merge`, `unmerge`, `sort`, `format`, `add-named-range`, `add-filter`, and `add-filter-view`.

| Format | Example | Description |
|--------|---------|-------------|
| `Sheet!Cell:Cell` | `Sheet1!A1:D10` | Specific range in a named sheet |
| `Sheet!Col:Col` | `Sheet1!A:D` | Full columns in a named sheet |
| `Sheet` | `Sheet1` | All data in a sheet |
| `Cell:Cell` | `A1:D10` | Range in the first sheet |

**Note:** Unbounded ranges (`A:A`, `1:1`) are NOT supported for `merge`, `unmerge`, and `sort`.

---

## gws sheets info

Gets metadata about a Google Sheets spreadsheet.

```
Usage: gws sheets info <spreadsheet-id>
```

---

## gws sheets list

Lists all sheets in a spreadsheet.

```
Usage: gws sheets list <spreadsheet-id>
```

Returns sheet names and IDs — useful for identifying sheets before other operations.

---

## gws sheets read

Reads cell values from a spreadsheet range.

```
Usage: gws sheets read <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--headers` | bool | true | Treat first row as headers (for JSON output) |
| `--output-format` | string | `json` | Output format: `json` or `csv` |

When `--headers` is true (default), the first row values become JSON object keys.

---

## gws sheets create

Creates a new Google Sheets spreadsheet.

```
Usage: gws sheets create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Spreadsheet title |
| `--sheet-names` | strings | `Sheet1` | No | Sheet names (comma-separated) |

---

## gws sheets write

Writes values to a range of cells.

```
Usage: gws sheets write <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--values` | string | | Values (comma-separated; semicolon for rows) |
| `--values-json` | string | | Values as JSON array |

One of `--values` or `--values-json` is required.

### Values Format

**Simple format** (`--values`):
- Single row: `"a,b,c"`
- Multiple rows: `"a,b,c;d,e,f"`

**JSON format** (`--values-json`):
- `'[["a","b"],["c","d"]]'`
- Supports mixed types: `'[["Name",30,true]]'`

Use `--values-json` if your data contains commas or semicolons.

---

## gws sheets append

Appends rows after the last row with data.

```
Usage: gws sheets append <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--values` | string | | Values (comma-separated; semicolon for rows) |
| `--values-json` | string | | Values as JSON array |

The range identifies the table to append to. Data is added after the last row containing data.

---

## gws sheets add-sheet

Adds a new sheet to an existing spreadsheet.

```
Usage: gws sheets add-sheet <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Sheet name |
| `--rows` | int | 1000 | No | Number of rows |
| `--cols` | int | 26 | No | Number of columns |

---

## gws sheets delete-sheet

Deletes a sheet from a spreadsheet.

```
Usage: gws sheets delete-sheet <spreadsheet-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | | Sheet name to delete |
| `--sheet-id` | int | -1 | Sheet ID to delete (alternative to `--name`) |

One of `--name` or `--sheet-id` is required.

---

## gws sheets clear

Clears all values from a range (keeps formatting).

```
Usage: gws sheets clear <spreadsheet-id> <range>
```

No additional flags.

---

## gws sheets insert-rows

Inserts empty rows at a specified position.

```
Usage: gws sheets insert-rows <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--at` | int | 0 | No | Row index to insert at (0-based) |
| `--count` | int | 1 | No | Number of rows to insert |

---

## gws sheets delete-rows

Deletes rows from a specified range.

```
Usage: gws sheets delete-rows <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--from` | int | | Yes | Start row index (0-based, inclusive) |
| `--to` | int | | Yes | End row index (0-based, exclusive) |

---

## gws sheets insert-cols

Inserts empty columns at a specified position.

```
Usage: gws sheets insert-cols <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--at` | int | 0 | No | Column index to insert at (0-based) |
| `--count` | int | 1 | No | Number of columns to insert |

---

## gws sheets delete-cols

Deletes columns from a specified range.

```
Usage: gws sheets delete-cols <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--from` | int | | Yes | Start column index (0-based, inclusive) |
| `--to` | int | | Yes | End column index (0-based, exclusive) |

---

## gws sheets rename-sheet

Renames a sheet within a spreadsheet.

```
Usage: gws sheets rename-sheet <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Current sheet name |
| `--name` | string | | Yes | New sheet name |

---

## gws sheets duplicate-sheet

Creates a copy of an existing sheet.

```
Usage: gws sheets duplicate-sheet <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name to duplicate |
| `--new-name` | string | | No | Name for the new sheet |

---

## gws sheets merge

Merges a range of cells into a single cell.

```
Usage: gws sheets merge <spreadsheet-id> <range>
```

No additional flags. Unbounded ranges (`A:A`, `1:1`) are not supported.

---

## gws sheets unmerge

Unmerges previously merged cells.

```
Usage: gws sheets unmerge <spreadsheet-id> <range>
```

No additional flags. Unbounded ranges are not supported.

---

## gws sheets sort

Sorts data in a range by a specified column.

```
Usage: gws sheets sort <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--by` | string | `A` | Column to sort by (e.g., `A`, `B`, `C`) |
| `--desc` | bool | false | Sort in descending order |
| `--has-header` | bool | false | First row is a header (excluded from sort) |

Unbounded ranges are not supported.

---

## gws sheets find-replace

Finds and replaces text across the spreadsheet or within a specific sheet.

```
Usage: gws sheets find-replace <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--find` | string | | Yes | Text to find |
| `--replace` | string | | Yes | Replacement text |
| `--sheet` | string | | No | Limit to specific sheet |
| `--match-case` | bool | false | No | Case-sensitive matching |
| `--entire-cell` | bool | false | No | Match entire cell contents only |

---

## gws sheets format

Formats cells in a range with text and background styles (v1.14.0).

```
Usage: gws sheets format <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--bold` | bool | false | Make text bold |
| `--italic` | bool | false | Make text italic |
| `--bg-color` | string | | Background color (hex, e.g., `#FFFF00`) |
| `--color` | string | | Text color (hex, e.g., `#FF0000`) |
| `--font-size` | int | 0 | Font size in points |

### Examples

```bash
# Make header row bold
gws sheets format 1abc123xyz "Sheet1!A1:Z1" --bold

# Highlight cells in yellow
gws sheets format 1abc123xyz "Sheet1!A2:D10" --bg-color "#FFFF00"

# Red text with larger font
gws sheets format 1abc123xyz "Sheet1!A1:A1" --color "#FF0000" --font-size 14

# Apply multiple styles
gws sheets format 1abc123xyz "Sheet1!B2:B100" --bold --italic --color "#0000FF"
```

### Notes

- At least one formatting flag is required
- Colors must be in hex format: `#RRGGBB`
- Unbounded ranges (e.g., `A:A`, `1:1`) are not supported
- Font size is in points (typical sizes: 8, 10, 11, 12, 14, 18)

---

## gws sheets set-column-width

Sets the width of a column in pixels (v1.14.0).

```
Usage: gws sheets set-column-width <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--col` | string | | Yes | Column letter (e.g., `A`, `B`, `AA`) |
| `--width` | int | 100 | No | Column width in pixels |

### Examples

```bash
# Set column A to 200 pixels wide
gws sheets set-column-width 1abc123xyz --sheet "Sheet1" --col A --width 200

# Set column B to default width (100 pixels)
gws sheets set-column-width 1abc123xyz --sheet "Sheet1" --col B

# Set double-letter column width
gws sheets set-column-width 1abc123xyz --sheet "Data" --col AA --width 150
```

### Notes

- Column letters are case-insensitive (`A` = `a`)
- Default width is 100 pixels (Google Sheets standard)
- Typical widths: narrow (50-80px), standard (100px), wide (150-250px)
- Multi-letter columns supported (e.g., `AA`, `AB`, `ZZ`)

---

## gws sheets set-row-height

Sets the height of a row in pixels (v1.14.0).

```
Usage: gws sheets set-row-height <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--row` | int | 1 | Yes | Row number (1-based) |
| `--height` | int | 21 | No | Row height in pixels |

### Examples

```bash
# Set row 1 (header) to 40 pixels tall
gws sheets set-row-height 1abc123xyz --sheet "Sheet1" --row 1 --height 40

# Set row 5 to default height (21 pixels)
gws sheets set-row-height 1abc123xyz --sheet "Sheet1" --row 5

# Make row 10 taller for wrapped text
gws sheets set-row-height 1abc123xyz --sheet "Data" --row 10 --height 60
```

### Notes

- Row numbers are 1-based (row 1 is the first row)
- Default height is 21 pixels (Google Sheets standard)
- Typical heights: compact (15-18px), standard (21px), tall (30-50px)
- Useful for header rows or cells with wrapped text

---

## gws sheets freeze

Freezes rows and/or columns so they remain visible when scrolling (v1.14.0).

```
Usage: gws sheets freeze <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--rows` | int | 0 | No | Number of rows to freeze |
| `--cols` | int | 0 | No | Number of columns to freeze |

### Examples

```bash
# Freeze the first row (header row)
gws sheets freeze 1abc123xyz --sheet "Sheet1" --rows 1

# Freeze the first column
gws sheets freeze 1abc123xyz --sheet "Sheet1" --cols 1

# Freeze first 2 rows and first column
gws sheets freeze 1abc123xyz --sheet "Data" --rows 2 --cols 1

# Unfreeze all (set both to 0)
gws sheets freeze 1abc123xyz --sheet "Sheet1" --rows 0 --cols 0
```

### Notes

- At least one of `--rows` or `--cols` must be specified (unless both are 0 to unfreeze)
- Frozen rows remain visible when scrolling vertically
- Frozen columns remain visible when scrolling horizontally
- Common pattern: freeze 1 row (header) and/or 1 column (labels)
- To unfreeze completely, set both `--rows 0 --cols 0`

---

## gws sheets copy-to

Copies a sheet tab from one spreadsheet to another.

```
Usage: gws sheets copy-to <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet-id` | int | 0 | Yes | Source sheet ID to copy |
| `--destination` | string | | Yes | Destination spreadsheet ID |

### Examples

```bash
# Copy sheet 0 to another spreadsheet
gws sheets copy-to 1abc123xyz --sheet-id 0 --destination 2def456uvw

# Copy sheet by ID (get IDs from gws sheets list)
gws sheets copy-to 1abc123xyz --sheet-id 12345 --destination 2def456uvw
```

### Notes

- Use `gws sheets list <id>` to find sheet IDs
- The copied sheet appears as a new tab in the destination spreadsheet
- The copy inherits all data, formatting, and conditional formatting

---

## gws sheets batch-read

Reads multiple ranges from a spreadsheet in a single API call.

```
Usage: gws sheets batch-read <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--ranges` | strings | | Yes | Ranges to read (can be repeated) |
| `--value-render` | string | `FORMATTED_VALUE` | No | Value render option |

**Value render options:**
- `FORMATTED_VALUE` — Values as displayed in the UI (default)
- `UNFORMATTED_VALUE` — Raw unformatted values
- `FORMULA` — Formulas instead of computed values

### Examples

```bash
# Read two ranges
gws sheets batch-read 1abc123xyz --ranges "Sheet1!A1:B5" --ranges "Sheet2!A1:C10"

# Read with formulas visible
gws sheets batch-read 1abc123xyz --ranges "A1:D10" --ranges "E1:F10" --value-render FORMULA

# Read from multiple sheets
gws sheets batch-read 1abc123xyz --ranges "Sales!A1:D100" --ranges "Inventory!A1:C50" --ranges "Summary!A1:B10"
```

### Notes

- More efficient than multiple `gws sheets read` calls
- Each range in the response includes its own data array
- Ranges can span different sheets within the same spreadsheet

---

## gws sheets batch-write

Writes values to multiple ranges in a single API call.

```
Usage: gws sheets batch-write <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--ranges` | strings | | Yes | Target ranges (pairs with `--values`) |
| `--values` | strings | | Yes | JSON arrays of values (pairs with `--ranges`) |
| `--value-input` | string | `USER_ENTERED` | No | Value input option |

**Value input options:**
- `USER_ENTERED` — Values parsed as if typed by a user (default)
- `RAW` — Values stored exactly as provided

### Examples

```bash
# Write to two ranges
gws sheets batch-write 1abc123xyz \
  --ranges "A1:B2" --values '[[1,2],[3,4]]' \
  --ranges "Sheet2!A1:B1" --values '[["x","y"]]'

# Write raw values (no formula parsing)
gws sheets batch-write 1abc123xyz \
  --ranges "A1:C1" --values '[["=SUM(B1:B10)","hello",42]]' \
  --value-input RAW
```

### Notes

- The nth `--ranges` flag pairs with the nth `--values` flag
- Number of `--ranges` flags must match number of `--values` flags
- Values must be JSON arrays (e.g., `'[["a","b"],["c","d"]]'`)
- More efficient than multiple `gws sheets write` calls

---

## gws sheets add-named-range

Adds a named range to a spreadsheet.

```
Usage: gws sheets add-named-range <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Name for the named range |

### Examples

```bash
# Create a named range for a data table
gws sheets add-named-range 1abc123xyz "Sheet1!A1:D100" --name "SalesData"

# Create a named range in a specific sheet
gws sheets add-named-range 1abc123xyz "Inventory!B2:F50" --name "StockLevels"
```

### Notes

- The response includes the `named_range_id` which is needed for deletion
- Named ranges can be used in formulas (e.g., `=SUM(SalesData)`)
- Range must include both start and end cells (e.g., `A1:D10`)

---

## gws sheets list-named-ranges

Lists all named ranges in a spreadsheet.

```
Usage: gws sheets list-named-ranges <spreadsheet-id>
```

No additional flags.

### Examples

```bash
# List all named ranges
gws sheets list-named-ranges 1abc123xyz
```

### Notes

- Returns name, `named_range_id`, and range coordinates for each named range
- Use the `named_range_id` with `delete-named-range` to remove a range

---

## gws sheets delete-named-range

Deletes a named range from a spreadsheet.

```
Usage: gws sheets delete-named-range <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--named-range-id` | string | | Yes | ID of the named range to delete |

### Examples

```bash
# Delete a named range by ID (get IDs from list-named-ranges)
gws sheets delete-named-range 1abc123xyz --named-range-id "nr-abc123"
```

### Notes

- Use `list-named-ranges` to find named range IDs
- Deleting a named range does not delete the underlying data

---

## gws sheets add-filter

Sets a basic filter on a range in a spreadsheet.

```
Usage: gws sheets add-filter <spreadsheet-id> <range>
```

No additional flags — the range is positional.

### Examples

```bash
# Add a filter to a data range
gws sheets add-filter 1abc123xyz "Sheet1!A1:D100"

# Add a filter in the first sheet
gws sheets add-filter 1abc123xyz "A1:F50"
```

### Notes

- Only one basic filter is allowed per sheet
- Setting a new filter replaces any existing basic filter on the sheet
- Basic filters add dropdown arrows to the header row for column filtering
- Range must include both start and end cells

---

## gws sheets clear-filter

Clears the basic filter from a sheet.

```
Usage: gws sheets clear-filter <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |

### Examples

```bash
# Clear the basic filter from Sheet1
gws sheets clear-filter 1abc123xyz --sheet "Sheet1"

# Clear filter from a named sheet
gws sheets clear-filter 1abc123xyz --sheet "Data"
```

### Notes

- Only removes the filter — does not affect the underlying data
- If no filter exists on the sheet, this is a no-op

---

## gws sheets add-filter-view

Creates a new filter view for a range in a spreadsheet.

```
Usage: gws sheets add-filter-view <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--name` | string | | Yes | Title for the filter view |

### Examples

```bash
# Create a filter view
gws sheets add-filter-view 1abc123xyz "Sheet1!A1:D100" --name "Active Items"

# Create a filter view in the first sheet
gws sheets add-filter-view 1abc123xyz "A1:F50" --name "Q1 Data"
```

### Notes

- Filter views are saved named views that don't affect other users
- Multiple filter views can exist on the same sheet
- The response includes the `filter_view_id`
- Unlike basic filters, filter views are per-user and don't change the shared view

---

## gws sheets add-chart

Adds an embedded chart to a spreadsheet.

```
Usage: gws sheets add-chart <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--type` | string | | Yes | Chart type: `BAR`, `LINE`, `AREA`, `COLUMN`, `SCATTER`, `PIE`, `COMBO` |
| `--data` | string | | Yes | Data range (e.g., `Sheet1!A1:B10`) |
| `--title` | string | | No | Chart title |
| `--sheet` | string | | No | Sheet to place chart on (defaults to new chart sheet) |

### Examples

```bash
# Add a bar chart from data in A1:B10
gws sheets add-chart 1abc123xyz --type BAR --data "Sheet1!A1:B10" --title "Sales"

# Add a pie chart
gws sheets add-chart 1abc123xyz --type PIE --data "Sheet1!A1:B5" --title "Distribution"

# Add a line chart overlaid on an existing sheet
gws sheets add-chart 1abc123xyz --type LINE --data "Sheet1!A1:C20" --sheet "Sheet1"
```

### Notes

- PIE charts use a different internal spec than other chart types
- Without `--sheet`, the chart is placed on a new dedicated chart sheet
- With `--sheet`, the chart is overlaid on the specified sheet at position A1
- Valid types: BAR, LINE, AREA, COLUMN, SCATTER, PIE, COMBO

---

## gws sheets list-charts

Lists all embedded charts in a spreadsheet.

```
Usage: gws sheets list-charts <spreadsheet-id>
```

No additional flags. Returns chart IDs, titles, types, and which sheet each chart is on.

---

## gws sheets delete-chart

Deletes an embedded chart by its chart ID.

```
Usage: gws sheets delete-chart <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--chart-id` | int | | Yes | Chart ID to delete |

### Examples

```bash
# Delete chart with ID 12345
gws sheets delete-chart 1abc123xyz --chart-id 12345

# List charts first, then delete
gws sheets list-charts 1abc123xyz
gws sheets delete-chart 1abc123xyz --chart-id <id-from-list>
```

---

## gws sheets add-conditional-format

Adds a conditional formatting rule to a range of cells.

```
Usage: gws sheets add-conditional-format <spreadsheet-id> <range> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--rule` | string | | Yes | Condition type (see table below) |
| `--value` | string | | Depends | Comparison value |
| `--bg-color` | string | | No | Background color (hex, e.g., `#FFFF00`) |
| `--color` | string | | No | Text color (hex, e.g., `#FF0000`) |
| `--bold` | bool | false | No | Make matching text bold |
| `--italic` | bool | false | No | Make matching text italic |

### Rule Types

| Rule | API Type | Needs `--value`? |
|------|----------|-----------------|
| `>` | NUMBER_GREATER | Yes |
| `<` | NUMBER_LESS | Yes |
| `=` | NUMBER_EQ | Yes |
| `!=` | NUMBER_NOT_EQ | Yes |
| `contains` | TEXT_CONTAINS | Yes |
| `not-contains` | TEXT_NOT_CONTAINS | Yes |
| `blank` | BLANK | No |
| `not-blank` | NOT_BLANK | No |
| `formula` | CUSTOM_FORMULA | Yes (formula string) |

### Examples

```bash
# Highlight cells > 100 in yellow
gws sheets add-conditional-format 1abc123xyz "Sheet1!A1:D10" --rule ">" --value "100" --bg-color "#FFFF00"

# Bold cells containing "URGENT"
gws sheets add-conditional-format 1abc123xyz "Sheet1!A1:A100" --rule "contains" --value "URGENT" --bold

# Red text for negative numbers
gws sheets add-conditional-format 1abc123xyz "B2:B100" --rule "<" --value "0" --color "#FF0000"

# Highlight blank cells
gws sheets add-conditional-format 1abc123xyz "Sheet1!A1:D10" --rule "blank" --bg-color "#FFCCCC"

# Custom formula
gws sheets add-conditional-format 1abc123xyz "A1:A100" --rule "formula" --value "=A1>B1" --bold --italic
```

### Notes

- If no format flags are specified, defaults to yellow background
- Colors must be in hex format: `#RRGGBB`
- New rules are inserted at index 0 (highest priority)
- Unbounded ranges (`A:A`, `1:1`) are not supported

---

## gws sheets list-conditional-formats

Lists all conditional formatting rules for a specific sheet.

```
Usage: gws sheets list-conditional-formats <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |

### Examples

```bash
# List all conditional format rules on Sheet1
gws sheets list-conditional-formats 1abc123xyz --sheet "Sheet1"
```

### Notes

- Returns rule index, condition type, values, ranges, and format details
- Use the index from the output with `delete-conditional-format`

---

## gws sheets delete-conditional-format

Deletes a conditional formatting rule by its index within a sheet.

```
Usage: gws sheets delete-conditional-format <spreadsheet-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--sheet` | string | | Yes | Sheet name |
| `--index` | int | | Yes | 0-based index of the rule to delete |

### Examples

```bash
# Delete the first (highest priority) conditional format rule
gws sheets delete-conditional-format 1abc123xyz --sheet "Sheet1" --index 0

# List rules first, then delete by index
gws sheets list-conditional-formats 1abc123xyz --sheet "Sheet1"
gws sheets delete-conditional-format 1abc123xyz --sheet "Sheet1" --index 2
```

### Notes

- Get rule indices from `list-conditional-formats`
- Indices are 0-based
- Deleting a rule shifts the indices of subsequent rules
