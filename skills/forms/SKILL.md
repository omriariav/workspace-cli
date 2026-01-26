---
name: gws-forms
version: 1.0.0
description: "Google Forms CLI operations via gws. Use when users need to get form metadata or retrieve form responses. Read-only access. Triggers: google forms, form responses, survey, form data."
metadata:
  short-description: Google Forms CLI operations (read-only)
  compatibility: claude-code, codex-cli
---

# Google Forms (gws forms)

`gws forms` provides read-only CLI access to Google Forms with structured JSON output.

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
| Get form info | `gws forms info <form-id>` |
| Get form responses | `gws forms responses <form-id>` |

## Detailed Usage

### info — Get form info

```bash
gws forms info <form-id>
```

Gets metadata about a Google Form including title, description, and question structure.

### responses — Get form responses

```bash
gws forms responses <form-id>
```

Gets all responses submitted to a form.

## Output Modes

```bash
gws forms info <form-id> --format json    # Structured JSON (default)
gws forms info <form-id> --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- This is read-only — forms cannot be created or modified via the CLI
- Form IDs can be extracted from Google Forms URLs: `docs.google.com/forms/d/<ID>/edit`
- Use `info` to understand the form structure before fetching responses
