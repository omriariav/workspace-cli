---
name: gws-tasks
version: 1.0.0
description: "Google Tasks CLI operations via gws. Use when users need to manage task lists, view tasks, create tasks, or mark tasks complete. Triggers: google tasks, task list, todo, task management."
metadata:
  short-description: Google Tasks CLI operations
  compatibility: claude-code, codex-cli
---

# Google Tasks (gws tasks)

`gws tasks` provides CLI access to Google Tasks with structured JSON output.

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
| List task lists | `gws tasks lists` |
| List tasks | `gws tasks list <tasklist-id>` |
| List with completed | `gws tasks list <tasklist-id> --show-completed` |
| Create a task | `gws tasks create --title "Buy groceries"` |
| Create with due date | `gws tasks create --title "Report" --due "2024-02-01"` |
| Complete a task | `gws tasks complete <tasklist-id> <task-id>` |

## Detailed Usage

### lists — List task lists

```bash
gws tasks lists
```

Lists all your task lists. The default list is `@default`.

### list — List tasks in a task list

```bash
gws tasks list <tasklist-id> [flags]
```

**Flags:**
- `--max int` — Maximum number of tasks (default 100)
- `--show-completed` — Include completed tasks

**Examples:**
```bash
gws tasks list @default
gws tasks list @default --show-completed
gws tasks list @default --max 10
```

### create — Create a task

```bash
gws tasks create --title <title> [flags]
```

**Flags:**
- `--title string` — Task title (required)
- `--tasklist string` — Task list ID (default: "@default")
- `--notes string` — Task notes/description
- `--due string` — Due date in RFC3339 or `YYYY-MM-DD` format

**Examples:**
```bash
gws tasks create --title "Buy groceries"
gws tasks create --title "Finish report" --due "2024-02-01" --notes "Include Q4 data"
gws tasks create --title "Team task" --tasklist MTIzNDU2
```

### complete — Mark a task as completed

```bash
gws tasks complete <tasklist-id> <task-id>
```

Marks a specific task as completed.

**Examples:**
```bash
gws tasks complete @default dGFzay0xMjM0
```

## Output Modes

```bash
gws tasks list @default --format json    # Structured JSON (default)
gws tasks list @default --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws tasks lists` first to get task list IDs
- Use `gws tasks list <tasklist-id>` to get individual task IDs for the `complete` command
- The default task list ID is `@default` — use this when users don't specify a list
- Due dates accept both RFC3339 (`2024-02-01T00:00:00Z`) and simple date (`2024-02-01`) formats
- Completed tasks are hidden by default; use `--show-completed` to include them
