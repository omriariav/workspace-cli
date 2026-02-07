# Tasks Commands Reference

Complete flag and option reference for `gws tasks` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws tasks lists

Lists all your task lists.

```
Usage: gws tasks lists
```

No additional flags.

### Output Fields (JSON)

Returns an array of task lists with:
- `id` — Task list ID (use this for other commands)
- `title` — Task list name
- `updated` — Last update time

The default task list has the special ID `@default`.

---

## gws tasks list

Lists all tasks in a specific task list.

```
Usage: gws tasks list <tasklist-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | 100 | Maximum number of tasks |
| `--show-completed` | bool | false | Include completed tasks |

### Output Fields (JSON)

Each task includes:
- `id` — Task ID
- `title` — Task title
- `notes` — Task notes/description
- `status` — `needsAction` or `completed`
- `due` — Due date (RFC3339)
- `completed` — Completion time (if completed)

---

## gws tasks create

Creates a new task in a task list.

```
Usage: gws tasks create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Task title |
| `--tasklist` | string | `@default` | No | Task list ID |
| `--notes` | string | | No | Task notes/description |
| `--due` | string | | No | Due date (RFC3339 or `YYYY-MM-DD`) |

### Date Format

- Simple: `2024-02-01`
- RFC3339: `2024-02-01T00:00:00Z`

---

## gws tasks update

Updates an existing task's title, notes, or due date.

```
Usage: gws tasks update <tasklist-id> <task-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | No | New task title |
| `--notes` | string | | No | New task notes/description |
| `--due` | string | | No | New due date (RFC3339 or `YYYY-MM-DD`) |

At least one of `--title`, `--notes`, or `--due` is required.

### Output Fields (JSON)

- `status` — Always `"updated"`
- `id` — Task ID
- `title` — Task title (updated or existing)
- `notes` — Task notes (if set)
- `due` — Due date (if set)

---

## gws tasks complete

Marks a specific task as completed.

```
Usage: gws tasks complete <tasklist-id> <task-id>
```

No additional flags. Both the task list ID and task ID are required positional arguments.
