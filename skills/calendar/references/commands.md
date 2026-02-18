# Calendar Commands Reference

Complete flag and option reference for `gws calendar` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws calendar list

Lists all calendars you have access to.

```
Usage: gws calendar list
```

No additional flags.

### Output Fields (JSON)

Returns an array of calendars with:
- `id` -- Calendar ID (e.g., `primary`, `user@group.calendar.google.com`)
- `summary` -- Calendar name
- `primary` -- `true` if this is the user's primary calendar
- `description` -- Calendar description (if set)

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
| `--pending` | bool | false | Only show events with pending RSVP (needsAction). Tip: increase `--max` for long date ranges. |

### Output Fields (JSON)

Fields are omitted when empty/nil to keep output compact.

**Core:**
- `id` -- Event ID (used for update/delete/rsvp)
- `summary` -- Event title
- `status` -- Event status (`confirmed`, `tentative`, `cancelled`)

**Time:**
- `start` -- Start time (dateTime or date for all-day events)
- `end` -- End time
- `all_day` -- `true` for all-day events (omitted for timed events)

**Details:**
- `description` -- Event description (HTML allowed by Google)
- `location` -- Event location
- `hangout_link` -- Legacy Hangouts link
- `html_link` -- Link to event in Google Calendar web UI
- `created` -- Creation timestamp
- `updated` -- Last-modified timestamp
- `color_id` -- Calendar color ID
- `visibility` -- `default`, `public`, `private`, `confidential`
- `transparency` -- `opaque` (busy) or `transparent` (free)
- `event_type` -- `default`, `outOfOffice`, `focusTime`, `workingLocation`

**People:**
- `organizer` -- Organizer email address
- `creator` -- Event creator email address
- `response_status` -- Current user's RSVP status: `accepted`, `declined`, `tentative`, `needsAction`

**Attendees:**
- `attendees[]` -- Full attendee list, each with: `email`, `response_status`, `optional` (bool), `organizer` (bool), `self` (bool)

**Conference:**
- `conference` -- `{ conference_id, solution, entry_points[]: { type, uri } }`

**Attachments:**
- `attachments[]` -- `{ file_url, title, mime_type, file_id }`

**Recurrence:**
- `recurrence[]` -- RRULE/EXRULE/RDATE/EXDATE strings

**Reminders:**
- `reminders` -- `{ use_default (bool), overrides[]: { method, minutes } }`

---

## gws calendar get

Gets a single event by its ID.

```
Usage: gws calendar get [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Event ID |
| `--calendar-id` | string | `primary` | No | Calendar ID |

Returns the same output fields as `events`.

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

## gws calendar quick-add

Creates an event from natural language text.

```
Usage: gws calendar quick-add [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--text` | string | | Yes | Text describing the event (e.g. "Lunch with John tomorrow at noon") |
| `--calendar-id` | string | `primary` | No | Calendar ID |

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

At least one update flag is required.

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
| `--message` | string | | No | Optional message (notifies all attendees) |

---

## gws calendar instances

Lists instances of a recurring event.

```
Usage: gws calendar instances [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Recurring event ID |
| `--calendar-id` | string | `primary` | No | Calendar ID |
| `--max` | int | 50 | No | Maximum number of instances |
| `--from` | string | | No | Start of time range (RFC3339 or YYYY-MM-DD) |
| `--to` | string | | No | End of time range (RFC3339 or YYYY-MM-DD) |

---

## gws calendar move

Moves an event to another calendar.

```
Usage: gws calendar move [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Event ID |
| `--calendar-id` | string | `primary` | No | Source calendar ID |
| `--destination` | string | | Yes | Destination calendar ID |

---

## gws calendar get-calendar

Gets metadata for a calendar.

```
Usage: gws calendar get-calendar [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID |

### Output Fields

- `id` -- Calendar ID
- `summary` -- Calendar name
- `description` -- Calendar description (if set)
- `timezone` -- Calendar timezone
- `location` -- Calendar location (if set)
- `etag` -- Calendar etag

---

## gws calendar create-calendar

Creates a new secondary calendar.

```
Usage: gws calendar create-calendar [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--summary` | string | | Yes | Calendar name |
| `--description` | string | | No | Calendar description |
| `--timezone` | string | | No | Calendar timezone (e.g. `America/New_York`) |

---

## gws calendar update-calendar

Updates an existing calendar's metadata.

```
Usage: gws calendar update-calendar [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID |
| `--summary` | string | | No | New calendar name |
| `--description` | string | | No | New calendar description |
| `--timezone` | string | | No | New calendar timezone |

---

## gws calendar delete-calendar

Deletes a secondary calendar.

```
Usage: gws calendar delete-calendar [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID |

---

## gws calendar clear

Clears all events from a calendar.

```
Usage: gws calendar clear [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--calendar-id` | string | `primary` | Calendar ID |

---

## gws calendar subscribe

Subscribes to a public calendar (adds to your calendar list).

```
Usage: gws calendar subscribe [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID to subscribe to |

---

## gws calendar unsubscribe

Unsubscribes from a calendar (removes from your calendar list).

```
Usage: gws calendar unsubscribe [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID to unsubscribe from |

---

## gws calendar calendar-info

Gets the calendar list entry (subscription settings, color, visibility).

```
Usage: gws calendar calendar-info [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID |

### Output Fields

- `id`, `summary`, `primary`, `description`, `timezone`
- `color_id`, `background_color`, `foreground_color`
- `summary_override`, `hidden`, `selected`, `access_role`

---

## gws calendar update-subscription

Updates subscription settings for a calendar in your list.

```
Usage: gws calendar update-subscription [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--id` | string | | Yes | Calendar ID |
| `--color-id` | string | | No | Color ID (use `gws calendar colors` to list valid IDs) |
| `--hidden` | bool | false | No | Hide calendar from the list |
| `--summary-override` | string | | No | Custom display name |

---

## gws calendar acl

Lists access control rules for a calendar.

```
Usage: gws calendar acl [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--calendar-id` | string | `primary` | Calendar ID |

### Output Fields

Returns array of rules with: `id`, `role`, `scope_type`, `scope_value`.

---

## gws calendar share

Shares a calendar with a user.

```
Usage: gws calendar share [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--calendar-id` | string | `primary` | No | Calendar ID |
| `--email` | string | | Yes | Email address to share with |
| `--role` | string | | Yes | Access role: `reader`, `writer`, `owner`, `freeBusyReader` |

---

## gws calendar unshare

Removes an access control rule from a calendar.

```
Usage: gws calendar unshare [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--calendar-id` | string | `primary` | No | Calendar ID |
| `--rule-id` | string | | Yes | ACL rule ID (e.g. `user:user@example.com`) |

---

## gws calendar update-acl

Updates an existing access control rule.

```
Usage: gws calendar update-acl [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--calendar-id` | string | `primary` | No | Calendar ID |
| `--rule-id` | string | | Yes | ACL rule ID |
| `--role` | string | | Yes | New role: `reader`, `writer`, `owner`, `freeBusyReader` |

---

## gws calendar freebusy

Queries free/busy information for one or more calendars.

```
Usage: gws calendar freebusy [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--from` | string | | Yes | Start of time range |
| `--to` | string | | Yes | End of time range |
| `--calendars` | string | `primary` | No | Comma-separated calendar IDs |

### Output

```json
{
  "time_min": "...",
  "time_max": "...",
  "calendars": {
    "primary": {
      "busy": [{"start": "...", "end": "..."}]
    }
  }
}
```

---

## gws calendar colors

Lists all available calendar and event colors.

```
Usage: gws calendar colors
```

No additional flags.

### Output

Returns `calendar_colors` and `event_colors` maps, each keyed by color ID with `background` and `foreground` hex values.

---

## gws calendar settings

Lists all user calendar settings.

```
Usage: gws calendar settings
```

No additional flags.

### Output

Returns `settings` map of key-value pairs (e.g. `timezone`, `locale`, `weekStart`).
