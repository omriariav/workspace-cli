---
name: gws-forms
version: 1.1.0
description: "Google Forms CLI operations via gws. Use when users need to get form metadata, retrieve form responses, create forms, or update forms. Triggers: google forms, form responses, survey, form data, create form, update form."
metadata:
  short-description: Google Forms CLI operations
  compatibility: claude-code, codex-cli
---

# Google Forms (gws forms)

`gws forms` provides CLI access to Google Forms with structured JSON output.

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
| Get form info | `gws forms info <form-id>` |
| Get form details (alias) | `gws forms get <form-id>` |
| Get all form responses | `gws forms responses <form-id>` |
| Get a single response | `gws forms response <form-id> --response-id <id>` |
| Create a new form | `gws forms create --title "My Form"` |
| Update a form | `gws forms update <form-id> --title "New Title"` |

## Detailed Usage

### info — Get form info

```bash
gws forms info <form-id>
```

Gets metadata about a Google Form including title, description, and question structure.

### get — Get form details

```bash
gws forms get <form-id>
```

Alias for `info`. Gets metadata about a Google Form including title, description, and question structure.

### responses — Get all form responses

```bash
gws forms responses <form-id>
```

Gets all responses submitted to a form, with question titles mapped to answers.

### response — Get a single form response

```bash
gws forms response <form-id> --response-id <response-id>
```

Gets a specific response by ID from a form.

**Flags:**
- `--response-id string` — Response ID to retrieve (required)

### create — Create a new form

```bash
gws forms create --title "Feedback Survey"
gws forms create --title "Team Poll" --description "Weekly team feedback"
```

Creates a new blank Google Form with a title and optional description.

**Flags:**
- `--title string` — Form title (required)
- `--description string` — Form description (optional)

### update — Update a form

```bash
gws forms update <form-id> --title "New Title"
gws forms update <form-id> --description "Updated description"
gws forms update <form-id> --file batch-update.json
```

Updates a Google Form. For simple changes use `--title` or `--description` flags. For advanced updates (adding questions, etc.), provide a JSON file with a batchUpdate request body.

**Flags:**
- `--title string` — New form title
- `--description string` — New form description
- `--file string` — Path to JSON file with batchUpdate request body

## Output Modes

```bash
gws forms info <form-id> --format json    # Structured JSON (default)
gws forms info <form-id> --format yaml    # YAML format
gws forms info <form-id> --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Form IDs can be extracted from Google Forms URLs: `docs.google.com/forms/d/<ID>/edit`
- Use `info` to understand the form structure before fetching responses
- Use `create` to make a new blank form, then `update --file` to add questions via batchUpdate
- The `get` command is an alias for `info` — both return the same output
- Use `--quiet` on any command to suppress JSON output (useful for scripted actions)
