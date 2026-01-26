---
name: gws-chat
version: 1.0.0
description: "Google Chat CLI operations via gws. Use when users need to list chat spaces, read messages, or send messages in Google Chat. Triggers: google chat, gchat, chat spaces, chat messages."
metadata:
  short-description: Google Chat CLI operations
  compatibility: claude-code, codex-cli
---

# Google Chat (gws chat)

`gws chat` provides CLI access to Google Chat with structured JSON output.

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

**Note:** Google Chat API requires additional setup:
1. Enable the Chat API in your Google Cloud project
2. Configure the OAuth consent screen for Chat scopes
3. For some operations, you may need a service account with domain-wide delegation

## Quick Command Reference

| Task | Command |
|------|---------|
| List chat spaces | `gws chat list` |
| Read messages | `gws chat messages <space-id>` |
| Read recent messages | `gws chat messages <space-id> --max 10` |
| Send a message | `gws chat send --space <space-id> --text "Hello"` |

## Detailed Usage

### list — List chat spaces

```bash
gws chat list
```

Lists all Chat spaces (rooms, DMs, group chats) you have access to.

### messages — List messages in a space

```bash
gws chat messages <space-id> [flags]
```

**Flags:**
- `--max int` — Maximum number of messages to return (default 25)

### send — Send a message

```bash
gws chat send --space <space-id> --text <message> [flags]
```

**Flags:**
- `--space string` — Space ID or name (required)
- `--text string` — Message text (required)

**Examples:**
```bash
gws chat send --space spaces/AAAA1234 --text "Hello team!"
```

## Output Modes

```bash
gws chat list --format json    # Structured JSON (default)
gws chat list --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws chat list` first to get space IDs
- Space IDs are in the format `spaces/AAAA1234`
- Chat API requires additional GCP setup beyond standard OAuth — see the `gws-auth` skill
