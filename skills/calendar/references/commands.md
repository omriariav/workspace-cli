# Calendar Commands Reference

Complete flag and option reference for `gws calendar` commands.

> **Disclaimer:** This is an unofficial CLI tool, not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |

---

## gws calendar list

Lists all calendars you have access to.

```
Usage: gws calendar list
```

No additional flags.

### Output Fields (JSON)

Returns an array of calendars with:
- `id` — Calendar ID (e.g., `primary`, `user@group.calendar.google.com`)
- `summary` — Calendar name
- `description` — Calendar description
- `accessRole` — Your access level (`owner`, `writer`, `reader`)

---

## gws calendar events

Lists upcoming events from a calendar.

```
Usage: gws calendar events [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--calendar-id` | string | `primary` | Calendar ID |
| `--days` | int | 7 | Number of days to look ahead |
| `--max` | int | 50 | Maximum number of events |

### Output Fields (JSON)

Each event includes:
- `id` — Event ID (used for update/delete/rsvp)
- `summary` — Event title
- `start` — Start time (dateTime or date for all-day events)
- `end` — End time
- `location` — Event location
- `description` — Event description
- `attendees` — List of attendees with response status
- `htmlLink` — Link to event in Google Calendar

---

## gws calendar create

Creates a new calendar event.

```
Usage: gws calendar create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Event title |
| `--start` | string | | Yes | Start time (RFC3339 or `YYYY-MM-DD HH:MM`) |
| `--end` | string | | Yes | End time (RFC3339 or `YYYY-MM-DD HH:MM`) |
| `--calendar-id` | string | `primary` | No | Calendar ID |
| `--description` | string | | No | Event description |
| `--location` | string | | No | Event location |
| `--attendees` | strings | | No | Attendee email addresses (repeatable) |

### Time Format

Both formats are accepted:
- RFC3339: `2024-02-01T14:00:00Z` or `2024-02-01T14:00:00-05:00`
- Simple: `2024-02-01 14:00` (uses local timezone)

---

## gws calendar update

Updates an existing calendar event. Uses PATCH (only changed fields are sent).

```
Usage: gws calendar update <event-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--title` | string | | New event title |
| `--start` | string | | New start time |
| `--end` | string | | New end time |
| `--description` | string | | New event description |
| `--location` | string | | New event location |
| `--add-attendees` | strings | | Attendee emails to add (repeatable) |
| `--calendar-id` | string | `primary` | Calendar ID |

At least one update flag is required (`--title`, `--start`, `--end`, `--description`, `--location`, `--add-attendees`).

### API Behavior

The update command uses Google Calendar's **Patch** API (not Update/PUT). This means:
- Only fields you specify are changed
- Unchanged fields are preserved
- Avoids sending unnecessary re-invitation notifications to attendees

---

## gws calendar delete

Deletes a calendar event.

```
Usage: gws calendar delete <event-id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--calendar-id` | string | `primary` | Calendar ID |

---

## gws calendar rsvp

Sets your RSVP status for a calendar event.

```
Usage: gws calendar rsvp <event-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--response` | string | | Yes | Response: `accepted`, `declined`, `tentative` |
| `--calendar-id` | string | `primary` | No | Calendar ID |

### Valid Responses

| Value | Meaning |
|-------|---------|
| `accepted` | Accept the invitation |
| `declined` | Decline the invitation |
| `tentative` | Maybe / tentatively accept |
