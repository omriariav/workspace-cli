# Slides Commands Reference

Complete flag and option reference for `gws slides` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |

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
