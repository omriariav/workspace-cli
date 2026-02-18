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

## gws tasks list-info

Gets details for a specific task list.

```
Usage: gws tasks list-info <tasklist-id>
```

No additional flags.

### Output Fields (JSON)

- `id` — Task list ID
- `title` — Task list name
- `updated` — Last update time
- `selfLink` — API self link

---

## gws tasks create-list

Creates a new task list.

```
Usage: gws tasks create-list [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Task list title |

### Output Fields (JSON)

- `status` — Always `"created"`
- `id` — New task list ID
- `title` — Task list title

---

## gws tasks update-list

Updates a task list's title.

```
Usage: gws tasks update-list <tasklist-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | New task list title |

### Output Fields (JSON)

- `status` — Always `"updated"`
- `id` — Task list ID
- `title` — Updated title

---

## gws tasks delete-list

Deletes a task list and all its tasks.

```
Usage: gws tasks delete-list <tasklist-id>
```

No additional flags.

### Output Fields (JSON)

- `status` — Always `"deleted"`
- `id` — Deleted task list ID

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

## gws tasks get

Gets details for a specific task.

```
Usage: gws tasks get <tasklist-id> <task-id>
```

No additional flags. Both the task list ID and task ID are required positional arguments.

### Output Fields (JSON)

- `id` — Task ID
- `title` — Task title
- `status` — `needsAction` or `completed`
- `notes` — Task notes (if set)
- `due` — Due date in RFC3339 (if set)
- `parent` — Parent task ID (if subtask)
- `completed` — Completion time (if completed)
- `updated` — Last update time

---

## gws tasks delete

Deletes a specific task.

```
Usage: gws tasks delete <tasklist-id> <task-id>
```

No additional flags. Both the task list ID and task ID are required positional arguments.

### Output Fields (JSON)

- `status` — Always `"deleted"`
- `id` — Deleted task ID

---

## gws tasks complete

Marks a specific task as completed.

```
Usage: gws tasks complete <tasklist-id> <task-id>
```

No additional flags. Both the task list ID and task ID are required positional arguments.

---

## gws tasks move

Moves a task to a different position, parent, or task list.

```
Usage: gws tasks move <tasklist-id> <task-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--parent` | string | | No | Parent task ID (makes this a subtask) |
| `--previous` | string | | No | Previous sibling task ID (positions after this task) |
| `--destination-list` | string | | No | Destination task list ID (moves to another list) |

### Output Fields (JSON)

- `status` — Always `"moved"`
- `id` — Task ID
- `title` — Task title
- `parent` — Parent task ID (if set)

---

## gws tasks clear

Clears all completed tasks from a task list. Completed tasks are marked as hidden and no longer returned by default.

```
Usage: gws tasks clear <tasklist-id>
```

No additional flags.

### Output Fields (JSON)

- `status` — Always `"cleared"`
- `list_id` — Task list ID
