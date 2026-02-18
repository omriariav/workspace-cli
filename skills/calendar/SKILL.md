---
name: gws-calendar
version: 2.0.0
description: "Google Calendar CLI operations via gws. Use when users need to list calendars, view events, create/update/delete events, RSVP to invitations, manage calendar CRUD, subscriptions, ACL, free/busy queries, colors, and settings. Triggers: calendar, events, meetings, schedule, rsvp, invite, share, freebusy."
metadata:
  short-description: Google Calendar CLI operations
  compatibility: claude-code, codex-cli
---

# Google Calendar (gws calendar)

`gws calendar` provides CLI access to Google Calendar with structured JSON output.

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

### Events

| Task | Command |
|------|---------|
| List calendars | `gws calendar list` |
| View upcoming events | `gws calendar events` |
| View next 14 days | `gws calendar events --days 14` |
| View pending invites | `gws calendar events --days 30 --pending` |
| Get event by ID | `gws calendar get --id <event-id>` |
| Create an event | `gws calendar create --title "Meeting" --start "2024-02-01 14:00" --end "2024-02-01 15:00"` |
| Quick add from text | `gws calendar quick-add --text "Lunch with John tomorrow at noon"` |
| Update an event | `gws calendar update <event-id> --title "New Title"` |
| Delete an event | `gws calendar delete <event-id>` |
| RSVP to an event | `gws calendar rsvp <event-id> --response accepted` |
| List recurring instances | `gws calendar instances --id <event-id>` |
| Move event to calendar | `gws calendar move --id <event-id> --destination <cal-id>` |

### Calendar Management

| Task | Command |
|------|---------|
| Get calendar metadata | `gws calendar get-calendar --id <cal-id>` |
| Create secondary calendar | `gws calendar create-calendar --summary "Projects"` |
| Update calendar | `gws calendar update-calendar --id <cal-id> --summary "New Name"` |
| Delete secondary calendar | `gws calendar delete-calendar --id <cal-id>` |
| Clear all events | `gws calendar clear` |

### Subscriptions

| Task | Command |
|------|---------|
| Subscribe to calendar | `gws calendar subscribe --id <cal-id>` |
| Unsubscribe | `gws calendar unsubscribe --id <cal-id>` |
| Get subscription info | `gws calendar calendar-info --id <cal-id>` |
| Update subscription | `gws calendar update-subscription --id <cal-id> --color "#0000ff"` |

### Access Control (ACL)

| Task | Command |
|------|---------|
| List ACL rules | `gws calendar acl` |
| Share with user | `gws calendar share --email user@example.com --role reader` |
| Remove access | `gws calendar unshare --rule-id "user:user@example.com"` |
| Update access | `gws calendar update-acl --rule-id "user:user@example.com" --role writer` |

### Other

| Task | Command |
|------|---------|
| Query free/busy | `gws calendar freebusy --from "2024-03-01 09:00" --to "2024-03-01 17:00"` |
| List colors | `gws calendar colors` |
| List settings | `gws calendar settings` |

## Detailed Usage

### list -- List calendars

```bash
gws calendar list
```

Lists all calendars you have access to, including shared calendars and subscriptions.

### events -- List events

```bash
gws calendar events [flags]
```

Lists upcoming events from a calendar.

**Flags:**
- `--calendar-id string` -- Calendar ID (default: "primary")
- `--days int` -- Number of days to look ahead (default 7)
- `--max int` -- Maximum number of events (default 50)
- `--pending` -- Only show events with pending RSVP (needsAction). Tip: increase `--max` when using `--pending` over long date ranges, since `--max` limits the API fetch before client-side filtering.

**Output includes** (fields omitted when empty):

| Category | Fields |
|----------|--------|
| **Core** | `id`, `summary`, `status` |
| **Time** | `start`, `end`, `all_day` |
| **Details** | `description`, `location`, `hangout_link`, `html_link`, `created`, `updated`, `color_id`, `visibility`, `transparency`, `event_type` |
| **People** | `organizer` (email), `creator` (email), `response_status` (your RSVP) |
| **Attendees** | `attendees[]` -- `{ email, response_status, optional, organizer, self }` |
| **Conference** | `conference` -- `{ conference_id, solution, entry_points[]: { type, uri } }` |
| **Attachments** | `attachments[]` -- `{ file_url, title, mime_type, file_id }` |
| **Recurrence** | `recurrence[]` -- RRULE strings |
| **Reminders** | `reminders` -- `{ use_default, overrides[]: { method, minutes } }` |

### get -- Get event by ID

```bash
gws calendar get --id <event-id> [--calendar-id <cal-id>]
```

Returns full event details using `mapEventToOutput` format (same fields as `events`).

### create -- Create an event

```bash
gws calendar create --title <title> --start <time> --end <time> [flags]
```

**Flags:**
- `--title string` -- Event title (required)
- `--start string` -- Start time in RFC3339 or `YYYY-MM-DD HH:MM` format (required)
- `--end string` -- End time in RFC3339 or `YYYY-MM-DD HH:MM` format (required)
- `--calendar-id string` -- Calendar ID (default: "primary")
- `--description string` -- Event description
- `--location string` -- Event location
- `--attendees strings` -- Attendee email addresses

### quick-add -- Quick add from text

```bash
gws calendar quick-add --text "Lunch with John tomorrow at noon" [--calendar-id <cal-id>]
```

Uses Google's natural language processing to create an event from a text string.

### update -- Update an event

```bash
gws calendar update <event-id> [flags]
```

Updates an existing calendar event. Uses PATCH (only changed fields are sent).

**Flags:**
- `--title string` -- New event title
- `--start string` -- New start time
- `--end string` -- New end time
- `--description string` -- New event description
- `--location string` -- New event location
- `--add-attendees strings` -- Attendee emails to add
- `--calendar-id string` -- Calendar ID (default: "primary")

### delete -- Delete an event

```bash
gws calendar delete <event-id> [--calendar-id <cal-id>]
```

### rsvp -- Respond to an event invitation

```bash
gws calendar rsvp <event-id> --response <status> [--message <text>] [--calendar-id <cal-id>]
```

Valid responses: `accepted`, `declined`, `tentative`. If `--message` is provided, notifies all attendees.

### instances -- List recurring event instances

```bash
gws calendar instances --id <event-id> [--max 50] [--from <time>] [--to <time>] [--calendar-id <cal-id>]
```

Lists all instances of a recurring event within an optional time range.

### move -- Move event to another calendar

```bash
gws calendar move --id <event-id> --destination <cal-id> [--calendar-id <cal-id>]
```

### get-calendar -- Get calendar metadata

```bash
gws calendar get-calendar --id <cal-id>
```

Returns: `id`, `summary`, `description`, `timezone`, `location`, `etag`.

### create-calendar -- Create a secondary calendar

```bash
gws calendar create-calendar --summary <name> [--description <text>] [--timezone <tz>]
```

### update-calendar -- Update a calendar

```bash
gws calendar update-calendar --id <cal-id> [--summary <name>] [--description <text>] [--timezone <tz>]
```

### delete-calendar -- Delete a secondary calendar

```bash
gws calendar delete-calendar --id <cal-id>
```

### clear -- Clear all events from a calendar

```bash
gws calendar clear [--calendar-id <cal-id>]
```

### subscribe -- Subscribe to a public calendar

```bash
gws calendar subscribe --id <cal-id>
```

### unsubscribe -- Unsubscribe from a calendar

```bash
gws calendar unsubscribe --id <cal-id>
```

### calendar-info -- Get subscription info

```bash
gws calendar calendar-info --id <cal-id>
```

Returns: `id`, `summary`, `primary`, `description`, `timezone`, `color_id`, `background_color`, `foreground_color`, `summary_override`, `hidden`, `selected`, `access_role`.

### update-subscription -- Update subscription settings

```bash
gws calendar update-subscription --id <cal-id> [--color <hex>] [--hidden] [--summary-override <name>]
```

### acl -- List access control rules

```bash
gws calendar acl [--calendar-id <cal-id>]
```

Returns array of rules with: `id`, `role`, `scope_type`, `scope_value`.

### share -- Share calendar with a user

```bash
gws calendar share --email <email> --role <role> [--calendar-id <cal-id>]
```

Valid roles: `reader`, `writer`, `owner`, `freeBusyReader`.

### unshare -- Remove calendar access

```bash
gws calendar unshare --rule-id <rule-id> [--calendar-id <cal-id>]
```

### update-acl -- Update access control rule

```bash
gws calendar update-acl --rule-id <rule-id> --role <role> [--calendar-id <cal-id>]
```

### freebusy -- Query free/busy information

```bash
gws calendar freebusy --from <time> --to <time> [--calendars "primary,user@example.com"]
```

Returns busy periods for each requested calendar.

### colors -- List available colors

```bash
gws calendar colors
```

Returns `calendar_colors` and `event_colors` maps with `background`/`foreground` hex values.

### settings -- List user calendar settings

```bash
gws calendar settings
```

Returns all user calendar settings as key-value pairs.

## Output Modes

```bash
gws calendar events --format json    # Structured JSON (default)
gws calendar events --format yaml    # YAML format
gws calendar events --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws calendar events` to get event IDs, then use those IDs for update/delete/rsvp/get/move
- Time format accepts both RFC3339 (`2024-02-01T14:00:00Z`) and human-friendly (`2024-02-01 14:00`)
- The `update` command uses PATCH (not PUT), so only changed fields are sent
- For non-primary calendars, get the calendar ID from `gws calendar list` first
- Use `quick-add` for natural language event creation (e.g. "Meeting with Bob next Tuesday 2pm")
- Use `freebusy` to check availability before creating events
- ACL roles: `freeBusyReader` < `reader` < `writer` < `owner`
- Default event window is 7 days; increase with `--days` for broader views
