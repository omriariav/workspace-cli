---
name: gws-slides
version: 1.0.0
description: "Google Slides CLI operations via gws. Use when users need to create, read, or edit Google Slides presentations. Triggers: slides, presentation, google slides, deck."
metadata:
  short-description: Google Slides CLI operations
  compatibility: claude-code, codex-cli
---

# Google Slides (gws slides)

`gws slides` provides CLI access to Google Slides with structured JSON output.

> **Disclaimer:** This is an unofficial CLI tool, not endorsed by or affiliated with Google.

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
