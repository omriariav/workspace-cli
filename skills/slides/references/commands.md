# Slides Commands Reference

Complete flag and option reference for `gws slides` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws slides info

Gets metadata about a Google Slides presentation.

```
Usage: gws slides info <presentation-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--notes` | bool | `false` | Include speaker notes in output |

---

## gws slides list

Lists all slides in a presentation with their content and object IDs.

```
Usage: gws slides list <presentation-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--notes` | bool | `false` | Include speaker notes in output |

Returns slide details including object IDs for elements — needed for `add-text`.

---

## gws slides read

Reads the text content of a specific slide or all slides.

```
Usage: gws slides read <presentation-id> [slide-number] [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--notes` | bool | `false` | Include speaker notes in output |

Slide numbers are **1-indexed**. Omit the slide number to read all slides.

---

## gws slides create

Creates a new Google Slides presentation.

```
Usage: gws slides create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Presentation title |

---

## gws slides add-slide

Adds a new slide to an existing presentation.

```
Usage: gws slides add-slide <presentation-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--title` | string | | Slide title |
| `--body` | string | | Slide body text |
| `--layout` | string | `TITLE_AND_BODY` | Slide layout (predefined) |
| `--layout-id` | string | | Custom layout ID from presentation's masters (overrides --layout) |

Use `gws slides list-layouts <id>` to discover available custom layout IDs.

### Available Predefined Layouts

| Layout | Description |
|--------|-------------|
| `TITLE_AND_BODY` | Title at top, body text below (default) |
| `TITLE_ONLY` | Title at top, empty body area |
| `BLANK` | Completely empty slide |
| `SECTION_HEADER` | Section divider |
| `TITLE` | Large centered title |
| `ONE_COLUMN_TEXT` | Single column text layout |
| `MAIN_POINT` | Main point emphasis |
| `BIG_NUMBER` | Large number display |

---

## gws slides delete-slide

Deletes a slide from a presentation.

```
Usage: gws slides delete-slide <presentation-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--slide-number` | int | | Slide number (1-indexed) |
| `--slide-id` | string | | Slide object ID (alternative) |

One of `--slide-number` or `--slide-id` is required.

---

## gws slides duplicate-slide

Creates a copy of an existing slide.

```
Usage: gws slides duplicate-slide <presentation-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--slide-number` | int | | Slide number (1-indexed) |
| `--slide-id` | string | | Slide object ID (alternative) |

One of `--slide-number` or `--slide-id` is required.

---

## gws slides add-shape

Adds a shape to a slide at specified position.

```
Usage: gws slides add-shape <presentation-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--slide-number` | int | | Slide number (1-indexed) |
| `--slide-id` | string | | Slide object ID |
| `--type` | string | `RECTANGLE` | Shape type |
| `--x` | float | 100 | X position in points |
| `--y` | float | 100 | Y position in points |
| `--width` | float | 200 | Width in points |
| `--height` | float | 100 | Height in points |

### Shape Types

`RECTANGLE`, `ELLIPSE`, `TEXT_BOX`, `ROUND_RECTANGLE`, `TRIANGLE`, `ARROW`, and more.

### Coordinate System

- All positions and sizes are in **points (PT)**
- Standard slide dimensions: **720 x 405 points**
- Origin (0,0) is the top-left corner

---

## gws slides add-image

Adds an image to a slide from a URL.

```
Usage: gws slides add-image <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--url` | string | | Yes | Image URL (must be publicly accessible) |
| `--slide-number` | int | | | Slide number (1-indexed) |
| `--slide-id` | string | | | Slide object ID |
| `--x` | float | 100 | No | X position in points |
| `--y` | float | 100 | No | Y position in points |
| `--width` | float | 400 | No | Width in points (height auto-calculated) |

Height is automatically calculated to maintain aspect ratio based on width.

---

## gws slides add-text

Inserts text into an existing shape, text box, table cell, or speaker notes.

```
Usage: gws slides add-text <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | | Object ID to insert text into (mutually exclusive with --table-id and --notes) |
| `--table-id` | string | | | Table object ID (requires --row and --col) |
| `--row` | int | -1 | | Row index, 0-based (required with --table-id) |
| `--col` | int | -1 | | Column index, 0-based (required with --table-id) |
| `--notes` | bool | false | | Target speaker notes (mutually exclusive with --object-id and --table-id) |
| `--slide-id` | string | | | Slide object ID (required with --notes) |
| `--slide-number` | int | 0 | | Slide number, 1-indexed (required with --notes) |
| `--text` | string | | Yes | Text to insert |
| `--at` | int | 0 | No | Position to insert at (0 = beginning) |

One of `--object-id`, `--table-id`, or `--notes` is required. Get object IDs from `gws slides list <id>` output.

---

## gws slides replace-text

Replaces all occurrences of text across ALL slides.

```
Usage: gws slides replace-text <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--find` | string | | Yes | Text to find |
| `--replace` | string | | Yes | Replacement text |
| `--match-case` | bool | true | No | Case-sensitive matching |

Operates across every slide in the presentation — useful for template variable substitution (e.g., replace `{{name}}` with a value).

---

## gws slides delete-object

Deletes any page element (shape, image, table, etc.) from a presentation.

```
Usage: gws slides delete-object <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Object ID to delete |

Get object IDs from `gws slides list <id>` output.

---

## gws slides delete-text

Clears text from a shape or speaker notes, optionally within a specific range.

```
Usage: gws slides delete-text <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | | Shape containing text (required unless --notes) |
| `--notes` | bool | false | | Target speaker notes (alternative to --object-id) |
| `--slide-id` | string | | | Slide object ID (required with --notes) |
| `--slide-number` | int | 0 | | Slide number, 1-indexed (required with --notes) |
| `--from` | int | 0 | No | Start index |
| `--to` | int | | No | End index (if omitted, deletes to end) |

One of `--object-id` or `--notes` is required.

---

## gws slides update-text-style

Updates text styling within a shape (bold, italic, font, color).

```
Usage: gws slides update-text-style <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Shape containing text |
| `--from` | int | 0 | No | Start index |
| `--to` | int | | No | End index (if omitted, applies to all) |
| `--bold` | bool | false | No | Make text bold |
| `--italic` | bool | false | No | Make text italic |
| `--underline` | bool | false | No | Underline text |
| `--font-size` | float | | No | Font size in points |
| `--font-family` | string | | No | Font family name |
| `--color` | string | | No | Text color as hex `#RRGGBB` |

---

## gws slides update-transform

Updates the position, scale, or rotation of a page element.

```
Usage: gws slides update-transform <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Element to transform |
| `--x` | float | 0 | No | X position in points |
| `--y` | float | 0 | No | Y position in points |
| `--scale-x` | float | 1 | No | Scale factor X |
| `--scale-y` | float | 1 | No | Scale factor Y |
| `--rotate` | float | 0 | No | Rotation in degrees |

---

## gws slides create-table

Creates a new table on a slide.

```
Usage: gws slides create-table <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--slide-number` | int | | | Slide number (1-indexed) |
| `--slide-id` | string | | | Slide object ID |
| `--rows` | int | | Yes | Number of rows |
| `--cols` | int | | Yes | Number of columns |
| `--x` | float | 100 | No | X position in points |
| `--y` | float | 100 | No | Y position in points |
| `--width` | float | 400 | No | Width in points |
| `--height` | float | 200 | No | Height in points |

One of `--slide-number` or `--slide-id` is required.

---

## gws slides insert-table-rows

Inserts rows into an existing table.

```
Usage: gws slides insert-table-rows <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-id` | string | | Yes | Table object ID |
| `--at` | int | | Yes | Row index to insert at |
| `--count` | int | 1 | No | Number of rows to insert |
| `--below` | bool | true | No | Insert below the index |

---

## gws slides delete-table-row

Removes a row from a table.

```
Usage: gws slides delete-table-row <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-id` | string | | Yes | Table object ID |
| `--row` | int | | Yes | Row index to delete |

---

## gws slides update-table-cell

Updates table cell background color.

```
Usage: gws slides update-table-cell <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-id` | string | | Yes | Table object ID |
| `--row` | int | | Yes | Row index |
| `--col` | int | | Yes | Column index |
| `--background-color` | string | | Yes | Background color as hex `#RRGGBB` |

---

## gws slides update-table-border

Styles table cell borders.

```
Usage: gws slides update-table-border <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--table-id` | string | | Yes | Table object ID |
| `--row` | int | | Yes | Row index |
| `--col` | int | | Yes | Column index |
| `--border` | string | all | No | Border to style: `top`, `bottom`, `left`, `right`, `all` |
| `--color` | string | | No | Border color as hex `#RRGGBB` |
| `--width` | float | 1 | No | Border width in points |
| `--style` | string | solid | No | Border style: `solid`, `dashed`, `dotted` |

---

## gws slides update-paragraph-style

Updates paragraph-level formatting (alignment, spacing).

```
Usage: gws slides update-paragraph-style <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Shape containing text |
| `--from` | int | 0 | No | Start index |
| `--to` | int | | No | End index (if omitted, applies to all) |
| `--alignment` | string | | No | Text alignment: `START`, `CENTER`, `END`, `JUSTIFIED` |
| `--line-spacing` | float | | No | Line spacing percentage (100 = single) |
| `--space-above` | float | | No | Space above paragraph in points |
| `--space-below` | float | | No | Space below paragraph in points |

---

## gws slides update-shape

Modifies shape properties (fill color, outline).

```
Usage: gws slides update-shape <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Shape to update |
| `--background-color` | string | | No | Fill color as hex `#RRGGBB` |
| `--outline-color` | string | | No | Outline color as hex `#RRGGBB` |
| `--outline-width` | float | 0 | No | Outline width in points |

---

## gws slides reorder-slides

Moves slides to a new position within the presentation.

```
Usage: gws slides reorder-slides <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--slide-ids` | string | | Yes | Comma-separated slide IDs to move |
| `--to` | int | | Yes | Target position (0-indexed) |

---

## gws slides update-slide-background

Sets the background of a slide to a solid color or an image URL.

```
Usage: gws slides update-slide-background <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--slide-number` | int | | | Slide number (1-indexed) |
| `--slide-id` | string | | | Slide object ID |
| `--color` | string | | | Background color as hex `#RRGGBB` |
| `--image-url` | string | | | Background image URL |

One of `--slide-number` or `--slide-id` is required. One of `--color` or `--image-url` is required (mutually exclusive).

---

## gws slides list-layouts

Lists all available slide layouts from the presentation's masters.

```
Usage: gws slides list-layouts <presentation-id>
```

No additional flags. Returns layout ID, name, display name, and master ID for each layout.

---

## gws slides add-line

Creates a line or connector on a slide.

```
Usage: gws slides add-line <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--slide-number` | int | | | Slide number (1-indexed) |
| `--slide-id` | string | | | Slide object ID |
| `--type` | string | `STRAIGHT_CONNECTOR_1` | No | Line type |
| `--start-x` | float | 0 | No | Start X position in points |
| `--start-y` | float | 0 | No | Start Y position in points |
| `--end-x` | float | 200 | No | End X position in points |
| `--end-y` | float | 200 | No | End Y position in points |
| `--color` | string | | No | Line color as hex `#RRGGBB` |
| `--weight` | float | 1 | No | Line thickness in points |

One of `--slide-number` or `--slide-id` is required. Line category is determined from the type prefix: `STRAIGHT_*`, `BENT_*`, or `CURVED_*`.

---

## gws slides group

Groups multiple page elements into a single group.

```
Usage: gws slides group <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-ids` | string | | Yes | Comma-separated element IDs to group (minimum 2) |

---

## gws slides ungroup

Ungroups a group element back into individual elements.

```
Usage: gws slides ungroup <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--group-id` | string | | Yes | Object ID of the group to ungroup |

---

## gws slides thumbnail

Gets a thumbnail image URL for a specific slide page.

```
Usage: gws slides thumbnail <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--slide` | string | | Yes | Slide object ID or 1-based slide number |
| `--size` | string | `MEDIUM` | No | Thumbnail size: SMALL, MEDIUM, LARGE |
| `--download` | string | | No | Download thumbnail to file path |
