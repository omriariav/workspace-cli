---
name: gws-calendar
version: 1.0.0
description: "Google Calendar CLI operations via gws. Use when users need to list calendars, view events, create/update/delete events, or RSVP to invitations. Triggers: calendar, events, meetings, schedule, rsvp, invite."
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

| Task | Command |
|------|---------|
| List calendars | `gws calendar list` |
| View upcoming events | `gws calendar events` |
| View next 14 days | `gws calendar events --days 14` |
| View pending invites | `gws calendar events --days 30 --pending` |
| Create an event | `gws calendar create --title "Meeting" --start "2024-02-01 14:00" --end "2024-02-01 15:00"` |
| Update an event | `gws calendar update <event-id> --title "New Title"` |
| Delete an event | `gws calendar delete <event-id>` |
| RSVP to an event | `gws calendar rsvp <event-id> --response accepted` |

## Detailed Usage

### list — List calendars

```bash
gws calendar list
```

Lists all calendars you have access to, including shared calendars and subscriptions.

### events — List events

```bash
gws calendar events [flags]
```

Lists upcoming events from a calendar.

**Flags:**
- `--calendar-id string` — Calendar ID (default: "primary")
- `--days int` — Number of days to look ahead (default 7)
- `--max int` — Maximum number of events (default 50)
- `--pending` — Only show events with pending RSVP (needsAction). Tip: increase `--max` when using `--pending` over long date ranges, since `--max` limits the API fetch before client-side filtering.

**Output includes:**
- `response_status` — User's RSVP status (`accepted`, `declined`, `tentative`, `needsAction`) when user is an attendee
- `organizer` — Organizer's email address

**Examples:**
```bash
gws calendar events
gws calendar events --days 14 --max 20
gws calendar events --calendar-id work@group.calendar.google.com
gws calendar events --days 30 --pending    # Pending invites only
```

### create — Create an event

```bash
gws calendar create --title <title> --start <time> --end <time> [flags]
```

**Flags:**
- `--title string` — Event title (required)
- `--start string` — Start time in RFC3339 or `YYYY-MM-DD HH:MM` format (required)
- `--end string` — End time in RFC3339 or `YYYY-MM-DD HH:MM` format (required)
- `--calendar-id string` — Calendar ID (default: "primary")
- `--description string` — Event description
- `--location string` — Event location
- `--attendees strings` — Attendee email addresses

**Examples:**
```bash
gws calendar create --title "Team Standup" --start "2024-02-01 09:00" --end "2024-02-01 09:30"
gws calendar create --title "Lunch" --start "2024-02-01 12:00" --end "2024-02-01 13:00" --location "Cafe"
gws calendar create --title "Review" --start "2024-02-01 14:00" --end "2024-02-01 15:00" --attendees user1@example.com --attendees user2@example.com
```

### update — Update an event

```bash
gws calendar update <event-id> [flags]
```

Updates an existing calendar event. Only specified fields are changed (uses PATCH, not PUT — avoids sending unnecessary notifications).

**Flags:**
- `--title string` — New event title
- `--start string` — New start time
- `--end string` — New end time
- `--description string` — New event description
- `--location string` — New event location
- `--add-attendees strings` — Attendee emails to add
- `--calendar-id string` — Calendar ID (default: "primary")

At least one update flag is required.

**Examples:**
```bash
gws calendar update abc123 --title "New Title"
gws calendar update abc123 --start "2024-02-01 14:00" --end "2024-02-01 15:00"
gws calendar update abc123 --add-attendees user@example.com
gws calendar update abc123 --location "Room 42" --description "Updated agenda"
```

### delete — Delete an event

```bash
gws calendar delete <event-id> [flags]
```

Deletes a calendar event.

**Flags:**
- `--calendar-id string` — Calendar ID (default: "primary")

**Examples:**
```bash
gws calendar delete abc123
gws calendar delete abc123 --calendar-id work@group.calendar.google.com
```

### rsvp — Respond to an event invitation

```bash
gws calendar rsvp <event-id> --response <status> [flags]
```

Sets your RSVP status for a calendar event.

**Flags:**
- `--response string` — Response: `accepted`, `declined`, `tentative` (required)
- `--calendar-id string` — Calendar ID (default: "primary")

**Examples:**
```bash
gws calendar rsvp abc123 --response accepted
gws calendar rsvp abc123 --response declined
gws calendar rsvp abc123 --response tentative
```

## Output Modes

```bash
gws calendar events --format json    # Structured JSON (default)
gws calendar events --format yaml    # YAML format
gws calendar events --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- Use `gws calendar events` to get event IDs, then use those IDs for update/delete/rsvp
- Time format accepts both RFC3339 (`2024-02-01T14:00:00Z`) and human-friendly (`2024-02-01 14:00`)
- The `update` command uses PATCH (not PUT), so only changed fields are sent — this avoids re-sending invitations to attendees
- For non-primary calendars, get the calendar ID from `gws calendar list` first
- Default event window is 7 days; increase with `--days` for broader views
