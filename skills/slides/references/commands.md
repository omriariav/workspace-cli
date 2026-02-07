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
Usage: gws slides info <presentation-id>
```

---

## gws slides list

Lists all slides in a presentation with their content and object IDs.

```
Usage: gws slides list <presentation-id>
```

Returns slide details including object IDs for elements — needed for `add-text`.

---

## gws slides read

Reads the text content of a specific slide or all slides.

```
Usage: gws slides read <presentation-id> [slide-number]
```

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
| `--layout` | string | `TITLE_AND_BODY` | Slide layout |

### Available Layouts

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

Inserts text into an existing shape or text box.

```
Usage: gws slides add-text <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Object ID to insert text into |
| `--text` | string | | Yes | Text to insert |
| `--at` | int | 0 | No | Position to insert at (0 = beginning) |

Get object IDs from `gws slides list <id>` output.

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

Clears text from a shape, optionally within a specific range.

```
Usage: gws slides delete-text <presentation-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--object-id` | string | | Yes | Shape containing text |
| `--from` | int | 0 | No | Start index |
| `--to` | int | | No | End index (if omitted, deletes to end) |

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
