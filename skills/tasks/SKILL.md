---
name: gws-tasks
version: 1.1.0
description: "Google Tasks CLI operations via gws. Use when users need to manage task lists, view/create/update/delete tasks, move tasks, or clear completed. Triggers: google tasks, task list, todo, task management."
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
| Get task list details | `gws tasks list-info <tasklist-id>` |
| Create a task list | `gws tasks create-list --title "My List"` |
| Update a task list | `gws tasks update-list <tasklist-id> --title "New Name"` |
| Delete a task list | `gws tasks delete-list <tasklist-id>` |
| List tasks | `gws tasks list <tasklist-id>` |
| List with completed | `gws tasks list <tasklist-id> --show-completed` |
| Get a task | `gws tasks get <tasklist-id> <task-id>` |
| Create a task | `gws tasks create --title "Buy groceries"` |
| Create with due date | `gws tasks create --title "Report" --due "2024-02-01"` |
| Update a task | `gws tasks update <tasklist-id> <task-id> --title "New title"` |
| Delete a task | `gws tasks delete <tasklist-id> <task-id>` |
| Complete a task | `gws tasks complete <tasklist-id> <task-id>` |
| Move a task | `gws tasks move <tasklist-id> <task-id> --previous <sibling-id>` |
| Clear completed | `gws tasks clear <tasklist-id>` |

## Detailed Usage

### lists — List task lists

```bash
gws tasks lists
```

Lists all your task lists. The default list is `@default`.

### list-info — Get task list details

```bash
gws tasks list-info <tasklist-id>
```

Gets details for a specific task list including id, title, updated time, and self link.

**Examples:**
```bash
gws tasks list-info @default
gws tasks list-info MTIzNDU2
```

### create-list — Create a task list

```bash
gws tasks create-list --title <title>
```

**Flags:**
- `--title string` — Task list title (required)

**Examples:**
```bash
gws tasks create-list --title "Work Tasks"
gws tasks create-list --title "Shopping"
```

### update-list — Update a task list

```bash
gws tasks update-list <tasklist-id> --title <title>
```

**Flags:**
- `--title string` — New task list title (required)

**Examples:**
```bash
gws tasks update-list MTIzNDU2 --title "Renamed List"
```

### delete-list — Delete a task list

```bash
gws tasks delete-list <tasklist-id>
```

Deletes a task list and all its tasks.

**Examples:**
```bash
gws tasks delete-list MTIzNDU2
```

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

### update — Update a task

```bash
gws tasks update <tasklist-id> <task-id> [flags]
```

Updates an existing task's title, notes, or due date. At least one of `--title`, `--notes`, or `--due` is required.

**Flags:**
- `--title string` — New task title
- `--notes string` — New task notes/description
- `--due string` — New due date in RFC3339 or `YYYY-MM-DD` format

**Examples:**
```bash
gws tasks update @default dGFzay0x --title "Updated title"
gws tasks update @default dGFzay0x --notes "Added notes" --due "2024-03-01"
```

### get — Get a task

```bash
gws tasks get <tasklist-id> <task-id>
```

Gets full details for a specific task including title, notes, status, due date, parent, and completion time.

**Examples:**
```bash
gws tasks get @default dGFzay0xMjM0
```

### delete — Delete a task

```bash
gws tasks delete <tasklist-id> <task-id>
```

Deletes a specific task.

**Examples:**
```bash
gws tasks delete @default dGFzay0xMjM0
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

### move — Move a task

```bash
gws tasks move <tasklist-id> <task-id> [flags]
```

Moves a task to a different position, parent, or task list.

**Flags:**
- `--parent string` — Parent task ID (makes this a subtask)
- `--previous string` — Previous sibling task ID (positions after this task)
- `--destination-list string` — Destination task list ID (moves to another list)

**Examples:**
```bash
gws tasks move @default task-1 --previous task-2
gws tasks move @default task-1 --parent parent-task
gws tasks move @default task-1 --destination-list other-list-id
```

### clear — Clear completed tasks

```bash
gws tasks clear <tasklist-id>
```

Clears all completed tasks from a task list. Completed tasks are marked as hidden and no longer returned by default.

**Examples:**
```bash
gws tasks clear @default
```

## Output Modes

```bash
gws tasks list @default --format json    # Structured JSON (default)
gws tasks list @default --format yaml    # YAML format
gws tasks list @default --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws tasks lists` first to get task list IDs
- Use `gws tasks list <tasklist-id>` to get individual task IDs for `get`, `update`, `delete`, `complete`, and `move` commands
- Use `gws tasks get` to fetch full task details (notes, due date, parent, completion time)
- The default task list ID is `@default` — use this when users don't specify a list
- Due dates accept both RFC3339 (`2024-02-01T00:00:00Z`) and simple date (`2024-02-01`) formats
- Completed tasks are hidden by default; use `--show-completed` to include them, or `clear` to permanently hide them
- Use `move --destination-list` to transfer tasks between lists, `--parent` to make subtasks, `--previous` to reorder
- Use `--quiet` on create/update/delete/complete operations to suppress JSON output in scripts
