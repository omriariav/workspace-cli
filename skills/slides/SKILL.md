---
name: gws-slides
version: 1.1.0
description: "Google Slides CLI operations via gws. Use when users need to create, read, or edit Google Slides presentations. Triggers: slides, presentation, google slides, deck."
metadata:
  short-description: Google Slides CLI operations
  compatibility: claude-code, codex-cli
---

# Google Slides (gws slides)

`gws slides` provides CLI access to Google Slides with structured JSON output.

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
| Get presentation info | `gws slides info <id>` |
| List all slides | `gws slides list <id>` |
| Read slide content | `gws slides read <id>` |
| Read specific slide | `gws slides read <id> 3` |
| Create presentation | `gws slides create --title "My Deck"` |
| Add a slide | `gws slides add-slide <id> --title "Slide Title" --body "Content"` |
| Add blank slide | `gws slides add-slide <id> --layout BLANK` |
| Delete a slide | `gws slides delete-slide <id> --slide-number 3` |
| Duplicate a slide | `gws slides duplicate-slide <id> --slide-number 2` |
| Add a shape | `gws slides add-shape <id> --slide-number 1 --type RECTANGLE` |
| Add an image | `gws slides add-image <id> --slide-number 1 --url "https://..."` |
| Add text to object | `gws slides add-text <id> --object-id <obj-id> --text "Hello"` |
| Find and replace | `gws slides replace-text <id> --find "old" --replace "new"` |
| Delete any element | `gws slides delete-object <id> --object-id <obj-id>` |
| Clear text from shape | `gws slides delete-text <id> --object-id <obj-id>` |
| Style text | `gws slides update-text-style <id> --object-id <obj-id> --bold --color "#FF0000"` |
| Move/resize element | `gws slides update-transform <id> --object-id <obj-id> --x 200 --y 100` |
| Create a table | `gws slides create-table <id> --slide-number 1 --rows 3 --cols 4` |
| Insert table rows | `gws slides insert-table-rows <id> --table-id <tbl-id> --at 1 --count 2` |
| Delete table row | `gws slides delete-table-row <id> --table-id <tbl-id> --row 2` |
| Style table cell | `gws slides update-table-cell <id> --table-id <tbl-id> --row 0 --col 0 --background-color "#FFFF00"` |
| Style table border | `gws slides update-table-border <id> --table-id <tbl-id> --row 0 --col 0 --color "#000000"` |
| Paragraph style | `gws slides update-paragraph-style <id> --object-id <obj-id> --alignment CENTER` |
| Shape properties | `gws slides update-shape <id> --object-id <obj-id> --background-color "#0000FF"` |
| Reorder slides | `gws slides reorder-slides <id> --slide-ids "slide1,slide2" --to 0` |

## Detailed Usage

### info — Get presentation info

```bash
gws slides info <presentation-id>
```

### list — List all slides

```bash
gws slides list <presentation-id>
```

Lists all slides with their content and object IDs.

### read — Read slide content

```bash
gws slides read <presentation-id> [slide-number]
```

Reads text content. Omit slide number to read all slides. Slide numbers are **1-indexed**.

### create — Create a presentation

```bash
gws slides create --title <title>
```

### add-slide — Add a slide

```bash
gws slides add-slide <presentation-id> [flags]
```

**Flags:**
- `--title string` — Slide title
- `--body string` — Slide body text
- `--layout string` — Layout type (default: "TITLE_AND_BODY")

**Available layouts:** `TITLE_AND_BODY`, `TITLE_ONLY`, `BLANK`, `SECTION_HEADER`, `TITLE`, `ONE_COLUMN_TEXT`, `MAIN_POINT`, `BIG_NUMBER`

### delete-slide — Delete a slide

```bash
gws slides delete-slide <presentation-id> [flags]
```

**Flags:**
- `--slide-number int` — Slide number (1-indexed)
- `--slide-id string` — Slide object ID (alternative)

### duplicate-slide — Duplicate a slide

```bash
gws slides duplicate-slide <presentation-id> [flags]
```

**Flags:**
- `--slide-number int` — Slide number (1-indexed)
- `--slide-id string` — Slide object ID (alternative)

### add-shape — Add a shape

```bash
gws slides add-shape <presentation-id> [flags]
```

**Flags:**
- `--slide-number int` — Slide number (1-indexed)
- `--slide-id string` — Slide object ID
- `--type string` — Shape type (default: "RECTANGLE")
- `--x float` — X position in points (default: 100)
- `--y float` — Y position in points (default: 100)
- `--width float` — Width in points (default: 200)
- `--height float` — Height in points (default: 100)

**Shape types:** `RECTANGLE`, `ELLIPSE`, `TEXT_BOX`, `ROUND_RECTANGLE`, `TRIANGLE`, `ARROW`, etc.

### add-image — Add an image

```bash
gws slides add-image <presentation-id> --url <image-url> [flags]
```

**Flags:**
- `--url string` — Image URL (required, must be publicly accessible)
- `--slide-number int` — Slide number (1-indexed)
- `--slide-id string` — Slide object ID
- `--x float` — X position in points (default: 100)
- `--y float` — Y position in points (default: 100)
- `--width float` — Width in points (default: 400; height auto-calculated)

### add-text — Add text to an object

```bash
gws slides add-text <presentation-id> --object-id <id> --text <text> [flags]
```

**Flags:**
- `--object-id string` — Object ID to insert text into (required)
- `--text string` — Text to insert (required)
- `--at int` — Position to insert at (0 = beginning)

Get object IDs from `gws slides list <id>` output.

### replace-text — Find and replace text

```bash
gws slides replace-text <presentation-id> --find <text> --replace <text> [flags]
```

**Flags:**
- `--find string` — Text to find (required)
- `--replace string` — Replacement text (required)
- `--match-case` — Case-sensitive matching (default: true)

Replaces across ALL slides in the presentation.

### delete-object — Delete any page element

```bash
gws slides delete-object <presentation-id> --object-id <id>
```

Deletes shapes, images, tables, or any page element by object ID.

### delete-text — Clear text from shape

```bash
gws slides delete-text <presentation-id> --object-id <id> [flags]
```

**Flags:**
- `--object-id string` — Shape containing text (required)
- `--from int` — Start index (default: 0)
- `--to int` — End index (if omitted, deletes to end)

### update-text-style — Style text formatting

```bash
gws slides update-text-style <presentation-id> --object-id <id> [flags]
```

**Flags:**
- `--object-id string` — Shape containing text (required)
- `--from int` / `--to int` — Text range (optional)
- `--bold` / `--italic` / `--underline` — Boolean styles
- `--font-size float` — Size in points
- `--font-family string` — Font name
- `--color string` — Hex color `#RRGGBB`

### update-transform — Move, scale, or rotate elements

```bash
gws slides update-transform <presentation-id> --object-id <id> [flags]
```

**Flags:**
- `--object-id string` — Element to transform (required)
- `--x` / `--y float` — Position in points
- `--scale-x` / `--scale-y float` — Scale factors
- `--rotate float` — Rotation in degrees

### create-table — Add a table

```bash
gws slides create-table <presentation-id> --rows <n> --cols <n> [flags]
```

**Flags:**
- `--slide-number int` or `--slide-id string` — Target slide
- `--rows int` — Number of rows (required)
- `--cols int` — Number of columns (required)
- `--x` / `--y` / `--width` / `--height float` — Position and size

### insert-table-rows — Insert rows into table

```bash
gws slides insert-table-rows <presentation-id> --table-id <id> --at <row> [flags]
```

**Flags:**
- `--table-id string` — Table object ID (required)
- `--at int` — Row index to insert at (required)
- `--count int` — Number of rows (default: 1)
- `--below` — Insert below the index (default: true)

### delete-table-row — Remove row from table

```bash
gws slides delete-table-row <presentation-id> --table-id <id> --row <index>
```

### update-table-cell — Style table cell

```bash
gws slides update-table-cell <presentation-id> --table-id <id> --row <r> --col <c> [flags]
```

**Flags:**
- `--table-id string` — Table object ID (required)
- `--row int` / `--col int` — Cell location (required)
- `--background-color string` — Hex color `#RRGGBB`

### update-table-border — Style table borders

```bash
gws slides update-table-border <presentation-id> --table-id <id> --row <r> --col <c> [flags]
```

**Flags:**
- `--table-id string` — Table object ID (required)
- `--row int` / `--col int` — Cell location (required)
- `--border string` — `top`, `bottom`, `left`, `right`, or `all`
- `--color string` — Hex color `#RRGGBB`
- `--width float` — Border width in points
- `--style string` — `solid`, `dashed`, or `dotted`

### update-paragraph-style — Paragraph formatting

```bash
gws slides update-paragraph-style <presentation-id> --object-id <id> [flags]
```

**Flags:**
- `--object-id string` — Shape containing text (required)
- `--from int` / `--to int` — Text range (optional)
- `--alignment string` — `START`, `CENTER`, `END`, `JUSTIFIED`
- `--line-spacing float` — Line spacing percentage (100 = single)
- `--space-above` / `--space-below float` — Paragraph spacing in points

### update-shape — Shape properties

```bash
gws slides update-shape <presentation-id> --object-id <id> [flags]
```

**Flags:**
- `--object-id string` — Shape to update (required)
- `--background-color string` — Fill color `#RRGGBB`
- `--outline-color string` — Outline color `#RRGGBB`
- `--outline-width float` — Outline width in points

### reorder-slides — Change slide order

```bash
gws slides reorder-slides <presentation-id> --slide-ids <ids> --to <position>
```

**Flags:**
- `--slide-ids string` — Comma-separated slide IDs to move (required)
- `--to int` — Target position, 0-indexed (required)

## Output Modes

```bash
gws slides list <id> --format json    # Structured JSON (default)
gws slides list <id> --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws slides list <id>` to get slide object IDs and element object IDs
- Slide numbers are **1-indexed** (first slide is 1, not 0)
- Positions and sizes are in **points (PT)**: standard slide is 720x405 points
- Image URLs must be publicly accessible — Google Slides fetches them server-side
- `replace-text` operates across ALL slides — useful for template variable substitution
- `add-text` inserts into an existing object; use `add-shape --type TEXT_BOX` to create a text container first
- Presentation IDs can be extracted from URLs: `docs.google.com/presentation/d/<ID>/edit`
- For comments on a presentation, use `gws drive comments <presentation-id>`
