# gws

<p align="center"><em>Unified CLI for Google Workspace — Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, and more from your terminal.</em></p>

<p align="center">
  <a href="https://github.com/omriariav/workspace-cli/actions/workflows/ci.yml"><img src="https://github.com/omriariav/workspace-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="go.mod"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8.svg" alt="Go Version"></a>
</p>

`gws` gives developers and AI agents a structured, token-efficient interface to 10+ Google Workspace services. Every command returns consistent JSON (or YAML or human-readable text), making it ideal for scripting, automation, and agent toolchains.

**Built for AI & automation:** Drop `gws` into Claude Code, Codex, or shell scripts and they inherit structured output, predictable flags, and safe defaults — no wrapper code required.

## Features

- **10+ Google services** — Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, Chat, Forms, Contacts, Custom Search.
- **Scriptable output** — `--format json` (default), `--format yaml`, `--format text` for human-readable tables, or `--quiet` to suppress output.
- **OAuth2 + PKCE** — Secure browser-based auth with automatic token refresh and `0600` file permissions.
- **Single auth flow** — Authenticate once to access all services; scopes based on `--services` flag, config, or all by default.
- **Lazy clients** — Service clients are initialized on-demand with mutex protection.

## Installation

### Go Install

```bash
go install github.com/omriariav/workspace-cli/cmd/gws@latest
```

### From Source

```bash
git clone https://github.com/omriariav/workspace-cli.git
cd workspace-cli
make build    # produces ./bin/gws
./bin/gws --help
```

### Prerequisites

1. A [Google Cloud Project](https://console.cloud.google.com/) with an **OAuth 2.0 Client ID** (Desktop type).
2. Enable the APIs you need in the [API Library](https://console.cloud.google.com/apis/library):
   - Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks (core)
   - Chat, Forms (optional — require additional setup)

## Quickstart

### 1. Configure credentials

```bash
export GWS_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GWS_CLIENT_SECRET="your-client-secret"
```

Or create `~/.config/gws/config.yaml`:

```yaml
client_id: "your-client-id.apps.googleusercontent.com"
client_secret: "your-client-secret"
```

### 2. Authenticate

```bash
gws auth login          # Opens browser for OAuth consent
gws auth status         # Verify: shows email and token expiry
```

### 3. Use it

```bash
gws gmail list --max 5 --query "is:unread"
gws calendar events --days 7
gws drive search "quarterly report" --max 10
gws docs read <document-id>
gws sheets read <spreadsheet-id> "Sheet1!A1:D10"
gws tasks lists
```

Add `--format text` for human-readable output, or `--format yaml` for YAML.

## Commands

### Auth

| Command | Description |
|---------|-------------|
| `gws auth login` | Authenticate via OAuth2 + PKCE |
| `gws auth status` | Show current auth status and email |
| `gws auth logout` | Remove stored credentials |

### Gmail

| Command | Description |
|---------|-------------|
| `gws gmail list` | List threads with `thread_id` and `message_id` (`--max`, `--query`, `--all`, `--include-labels`) |
| `gws gmail read <id>` | Read message body and headers |
| `gws gmail thread <id>` | Read full thread conversation |
| `gws gmail send` | Send email (`--to`, `--subject`, `--body`, `--cc`, `--bcc`, `--thread-id`, `--reply-to-message-id`) |
| `gws gmail reply <id>` | Reply to message (`--body`, `--cc`, `--bcc`, `--all`) |
| `gws gmail event-id <id>` | Extract calendar event ID from invite email |
| `gws gmail labels` | List all labels |
| `gws gmail label <id>` | Add/remove labels (`--add`, `--remove`) |
| `gws gmail archive <id>` | Archive message (remove from inbox) |
| `gws gmail archive-thread <id>` | Archive all messages in thread (archives + marks read) |
| `gws gmail trash <id>` | Move message to trash |
| `gws gmail untrash <id>` | Remove message from trash |
| `gws gmail delete <id>` | Permanently delete message |
| `gws gmail batch-modify` | Modify labels on multiple messages (`--ids`, `--add-labels`, `--remove-labels`) |
| `gws gmail batch-delete` | Permanently delete multiple messages (`--ids`) |
| `gws gmail trash-thread <id>` | Move thread to trash |
| `gws gmail untrash-thread <id>` | Remove thread from trash |
| `gws gmail delete-thread <id>` | Permanently delete thread |
| `gws gmail label-info <id>` | Get label details |
| `gws gmail create-label` | Create label (`--name`, `--visibility`, `--list-visibility`) |
| `gws gmail update-label` | Update label (`--id`, `--name`, `--visibility`, `--list-visibility`) |
| `gws gmail delete-label <id>` | Delete a label |
| `gws gmail drafts` | List drafts (`--max`, `--query`) |
| `gws gmail draft <id>` | Get draft by ID |
| `gws gmail create-draft` | Create draft (`--to`, `--subject`, `--body`, `--cc`, `--bcc`) |
| `gws gmail update-draft` | Update draft (`--id`, `--to`, `--subject`, `--body`) |
| `gws gmail send-draft <id>` | Send an existing draft |
| `gws gmail delete-draft <id>` | Delete a draft |
| `gws gmail attachment` | Download attachment (`--message-id`, `--id`, `--output`) |

### Calendar

| Command | Description |
|---------|-------------|
| `gws calendar list` | List all calendars |
| `gws calendar events` | List upcoming events with expanded details (`--days`, `--calendar-id`, `--max`, `--pending`) |
| `gws calendar create` | Create event (`--title`, `--start`, `--end`, `--attendees`) |
| `gws calendar update <id>` | Update event (`--title`, `--start`, `--end`, `--add-attendees`) |
| `gws calendar delete <id>` | Delete event |
| `gws calendar rsvp <id>` | RSVP to invite (`--response accepted/declined/tentative`, `--message`) |
| `gws calendar get` | Get event by ID (`--calendar-id`, `--id`) |
| `gws calendar quick-add` | Quick add event from text (`--text`) |
| `gws calendar instances` | List instances of recurring event (`--id`, `--max`, `--from`, `--to`) |
| `gws calendar move` | Move event to another calendar (`--id`, `--destination`) |
| `gws calendar get-calendar` | Get calendar metadata (`--id`) |
| `gws calendar create-calendar` | Create secondary calendar (`--summary`, `--description`, `--timezone`) |
| `gws calendar update-calendar` | Update calendar (`--id`, `--summary`, `--description`, `--timezone`) |
| `gws calendar delete-calendar` | Delete secondary calendar (`--id`) |
| `gws calendar clear` | Clear all events from a calendar (`--calendar-id`) |
| `gws calendar subscribe` | Subscribe to a public calendar (`--id`) |
| `gws calendar unsubscribe` | Unsubscribe from calendar (`--id`) |
| `gws calendar calendar-info` | Get calendar subscription info (`--id`) |
| `gws calendar update-subscription` | Update subscription settings (`--id`, `--color-id`, `--hidden`) |
| `gws calendar acl` | List access control rules (`--calendar-id`) |
| `gws calendar share` | Share calendar (`--calendar-id`, `--email`, `--role`) |
| `gws calendar unshare` | Remove access (`--calendar-id`, `--rule-id`) |
| `gws calendar update-acl` | Update access rule (`--calendar-id`, `--rule-id`, `--role`) |
| `gws calendar freebusy` | Query free/busy (`--from`, `--to`, `--calendars`) |
| `gws calendar colors` | List available calendar colors |
| `gws calendar settings` | List user calendar settings |

### Tasks

| Command | Description |
|---------|-------------|
| `gws tasks lists` | List task lists |
| `gws tasks list <id>` | List tasks in a list (`--show-completed`) |
| `gws tasks list-info <id>` | Get task list details |
| `gws tasks create` | Create task (`--title`, `--tasklist`, `--due`); accepts YYYY-MM-DD dates |
| `gws tasks create-list` | Create a task list (`--title`) |
| `gws tasks update <list> <task>` | Update task (`--title`, `--notes`, `--due`) |
| `gws tasks update-list <id>` | Update task list title (`--title`) |
| `gws tasks delete-list <id>` | Delete a task list |
| `gws tasks get <list> <task>` | Get task details |
| `gws tasks delete <list> <task>` | Delete a task |
| `gws tasks complete <list> <task>` | Mark task as done |
| `gws tasks move <list> <task>` | Move/reorder task (`--parent`, `--previous`, `--destination-list`) |
| `gws tasks clear <id>` | Clear completed tasks from a list |

### Drive

| Command | Description |
|---------|-------------|
| `gws drive list` | List files (`--folder`, `--max`, `--order`) |
| `gws drive search <query>` | Full-text search |
| `gws drive info <id>` | File metadata, owners, permissions |
| `gws drive download <id>` | Download file (`--output`); auto-exports Google formats |
| `gws drive upload <file>` | Upload file (`--folder`, `--name`, `--mime-type`); supports Shared Drives |
| `gws drive create-folder` | Create folder (`--name`, `--parent`) |
| `gws drive move <id>` | Move file to folder (`--to`) |
| `gws drive delete <id>` | Delete file (`--permanent` for hard delete) |
| `gws drive comments <id>` | List comments and replies (`--include-resolved`, `--include-deleted`) |
| `gws drive permissions` | List permissions on a file (`--file-id`) |
| `gws drive share` | Share file (`--file-id`, `--type`, `--role`, `--email`, `--domain`) |
| `gws drive unshare` | Remove permission (`--file-id`, `--permission-id`) |
| `gws drive permission` | Get permission details (`--file-id`, `--permission-id`) |
| `gws drive update-permission` | Update permission role (`--file-id`, `--permission-id`, `--role`) |
| `gws drive revisions` | List file revisions (`--file-id`) |
| `gws drive revision` | Get revision details (`--file-id`, `--revision-id`) |
| `gws drive delete-revision` | Delete a revision (`--file-id`, `--revision-id`) |
| `gws drive replies` | List replies to a comment (`--file-id`, `--comment-id`) |
| `gws drive reply` | Create reply (`--file-id`, `--comment-id`, `--content`) |
| `gws drive get-reply` | Get a specific reply (`--file-id`, `--comment-id`, `--reply-id`) |
| `gws drive delete-reply` | Delete a reply (`--file-id`, `--comment-id`, `--reply-id`) |
| `gws drive comment` | Get a single comment (`--file-id`, `--comment-id`) |
| `gws drive add-comment` | Add comment to file (`--file-id`, `--content`) |
| `gws drive delete-comment` | Delete comment (`--file-id`, `--comment-id`) |
| `gws drive export` | Export Google Workspace file (`--file-id`, `--mime-type`, `--output`) |
| `gws drive empty-trash` | Empty trash permanently |
| `gws drive update` | Update file metadata (`--file-id`, `--name`, `--description`, `--starred`, `--trashed`) |
| `gws drive shared-drives` | List shared drives (`--max`, `--query`) |
| `gws drive shared-drive` | Get shared drive details (`--id`) |
| `gws drive create-drive` | Create shared drive (`--name`) |
| `gws drive delete-drive` | Delete shared drive (`--id`) |
| `gws drive update-drive` | Update shared drive (`--id`, `--name`) |
| `gws drive about` | Get drive storage and user info |
| `gws drive changes` | List recent file changes (`--max`, `--page-token`) |
| `gws drive activity` | Query activity history (`--item-id`, `--folder-id`, `--filter`, `--days`, `--max`) |

### Docs

| Command | Description |
|---------|-------------|
| `gws docs read <id>` | Extract document text (`--include-formatting`) |
| `gws docs info <id>` | Document metadata and styles |
| `gws docs create` | Create new document (`--title`, `--text`) |
| `gws docs append <id>` | Append text to document (`--text`, `--newline`) |
| `gws docs insert <id>` | Insert text at position (`--text`, `--at`) |
| `gws docs replace <id>` | Find and replace text (`--find`, `--replace`, `--match-case`) |
| `gws docs delete <id>` | Delete content range (`--from`, `--to`) |
| `gws docs add-table <id>` | Insert table (`--rows`, `--cols`, `--at`) |
| `gws docs format <id>` | Format text (`--from`, `--to`, `--bold`, `--italic`, `--font-size`, `--color`) |
| `gws docs set-paragraph-style <id>` | Paragraph style (`--from`, `--to`, `--alignment`, `--line-spacing`) |
| `gws docs add-list <id>` | Add bullet/numbered list (`--at`, `--type`, `--items`) |
| `gws docs remove-list <id>` | Remove list formatting (`--from`, `--to`) |
| `gws docs trash <id>` | Trash document (`--permanent` for hard delete) |
| `gws docs add-tab <id>` | Add a tab (`--title`, `--index`) |
| `gws docs delete-tab <id>` | Delete a tab (`--tab-id`) |
| `gws docs rename-tab <id>` | Rename a tab (`--tab-id`, `--title`) |
| `gws docs add-image <id>` | Insert image (`--uri`, `--at`, `--width`, `--height`) |
| `gws docs insert-table-row <id>` | Insert table row (`--table-start`, `--row`, `--col`) |
| `gws docs delete-table-row <id>` | Delete table row (`--table-start`, `--row`, `--col`) |
| `gws docs insert-table-col <id>` | Insert table column (`--table-start`, `--row`, `--col`) |
| `gws docs delete-table-col <id>` | Delete table column (`--table-start`, `--row`, `--col`) |
| `gws docs merge-cells <id>` | Merge table cells (`--table-start`, `--row`, `--col`, `--row-span`, `--col-span`) |
| `gws docs unmerge-cells <id>` | Unmerge table cells (`--table-start`, `--row`, `--col`, `--row-span`, `--col-span`) |
| `gws docs pin-rows <id>` | Pin header rows (`--table-start`, `--count`) |
| `gws docs page-break <id>` | Insert page break (`--at`) |
| `gws docs section-break <id>` | Insert section break (`--at`, `--type`) |
| `gws docs add-header <id>` | Add header (`--type`) |
| `gws docs delete-header <id> <hid>` | Delete header |
| `gws docs add-footer <id>` | Add footer (`--type`) |
| `gws docs delete-footer <id> <fid>` | Delete footer |
| `gws docs add-named-range <id>` | Create named range (`--name`, `--from`, `--to`) |
| `gws docs delete-named-range <id>` | Delete named range (`--name` or `--id`) |
| `gws docs add-footnote <id>` | Insert footnote (`--at`) |
| `gws docs delete-object <id> <oid>` | Delete positioned object |
| `gws docs replace-image <id>` | Replace image (`--object-id`, `--uri`) |
| `gws docs replace-named-range <id>` | Replace named range text (`--name`/`--id`, `--text`) |
| `gws docs update-style <id>` | Update doc margins (`--margin-top/bottom/left/right`) |
| `gws docs update-section-style <id>` | Update section style (`--from`, `--to`, `--column-count`) |
| `gws docs update-table-cell-style <id>` | Update cell style (`--table-start`, `--row`, `--col`, `--bg-color`) |
| `gws docs update-table-col-properties <id>` | Update column width (`--table-start`, `--col-index`, `--width`) |
| `gws docs update-table-row-style <id>` | Update row style (`--table-start`, `--row`, `--min-height`) |

### Sheets

| Command | Description |
|---------|-------------|
| `gws sheets info <id>` | Spreadsheet metadata |
| `gws sheets list <id>` | List sheets in a spreadsheet |
| `gws sheets read <id> <range>` | Read cell values (`--output-format=csv`, `--headers`) |
| `gws sheets create` | Create spreadsheet (`--title`, `--sheet-names`) |
| `gws sheets write <id> <range>` | Write cell values (`--values`, `--values-json`) |
| `gws sheets append <id> <range>` | Append rows (`--values`, `--values-json`) |
| `gws sheets add-sheet <id>` | Add sheet (`--name`, `--rows`, `--cols`) |
| `gws sheets delete-sheet <id>` | Delete sheet (`--name` or `--sheet-id`) |
| `gws sheets clear <id> <range>` | Clear cell values (keeps formatting) |
| `gws sheets insert-rows <id>` | Insert rows (`--sheet`, `--at`, `--count`) |
| `gws sheets delete-rows <id>` | Delete rows (`--sheet`, `--from`, `--to`) |
| `gws sheets insert-cols <id>` | Insert columns (`--sheet`, `--at`, `--count`) |
| `gws sheets delete-cols <id>` | Delete columns (`--sheet`, `--from`, `--to`) |
| `gws sheets rename-sheet <id>` | Rename sheet (`--sheet`, `--name`) |
| `gws sheets duplicate-sheet <id>` | Duplicate sheet (`--sheet`, `--new-name`) |
| `gws sheets merge <id> <range>` | Merge cells |
| `gws sheets unmerge <id> <range>` | Unmerge cells |
| `gws sheets sort <id> <range>` | Sort data (`--by`, `--desc`, `--has-header`) |
| `gws sheets find-replace <id>` | Find and replace (`--find`, `--replace`, `--sheet`, `--match-case`) |
| `gws sheets format <id> <range>` | Format cells (`--bold`, `--italic`, `--bg-color`, `--color`, `--font-size`) |
| `gws sheets set-column-width <id>` | Set column width (`--sheet`, `--col`, `--width`) |
| `gws sheets set-row-height <id>` | Set row height (`--sheet`, `--row`, `--height`) |
| `gws sheets freeze <id>` | Freeze panes (`--sheet`, `--rows`, `--cols`) |
| `gws sheets copy-to <id>` | Copy sheet to another spreadsheet (`--sheet-id`, `--destination`) |
| `gws sheets batch-read <id>` | Read multiple ranges (`--ranges`, `--value-render`) |
| `gws sheets batch-write <id>` | Write multiple ranges (`--ranges`, `--values`, `--value-input`) |
| `gws sheets add-named-range <id> <range>` | Add named range (`--name`) |
| `gws sheets list-named-ranges <id>` | List all named ranges |
| `gws sheets delete-named-range <id>` | Delete named range (`--named-range-id`) |
| `gws sheets add-filter <id> <range>` | Set basic filter on range |
| `gws sheets clear-filter <id>` | Clear basic filter (`--sheet`) |
| `gws sheets add-filter-view <id> <range>` | Add filter view (`--name`) |
| `gws sheets add-chart <id>` | Add embedded chart (`--type`, `--data`, `--title`, `--sheet`) |
| `gws sheets list-charts <id>` | List all charts in a spreadsheet |
| `gws sheets delete-chart <id>` | Delete a chart (`--chart-id`) |
| `gws sheets add-conditional-format <id> <range>` | Add conditional format rule (`--rule`, `--value`, `--bg-color`, `--bold`) |
| `gws sheets list-conditional-formats <id>` | List conditional format rules (`--sheet`) |
| `gws sheets delete-conditional-format <id>` | Delete conditional format rule (`--sheet`, `--index`) |

### Slides

| Command | Description |
|---------|-------------|
| `gws slides info <id>` | Presentation metadata (`--notes` for speaker notes) |
| `gws slides list <id>` | List slides with text content (`--notes` for speaker notes) |
| `gws slides read <id> [n]` | Read slide text (specific or all, `--notes` for speaker notes) |
| `gws slides create` | Create new presentation (`--title`) |
| `gws slides add-slide <id>` | Add slide (`--title`, `--body`, `--layout`, `--layout-id`) |
| `gws slides delete-slide <id>` | Delete slide (`--slide-id` or `--slide-number`) |
| `gws slides duplicate-slide <id>` | Duplicate slide (`--slide-id` or `--slide-number`) |
| `gws slides add-shape <id>` | Add shape (`--slide-id/--slide-number`, `--type`, `--x`, `--y`, `--width`, `--height`) |
| `gws slides add-image <id>` | Add image (`--slide-id/--slide-number`, `--url`, `--x`, `--y`, `--width`) |
| `gws slides add-text <id>` | Insert text into shape, table cell, or speaker notes (`--object-id`, `--table-id`/`--row`/`--col`, or `--notes`/`--slide-number`) |
| `gws slides replace-text <id>` | Find and replace text (`--find`, `--replace`, `--match-case`) |
| `gws slides delete-object <id>` | Delete any page element (`--object-id`) |
| `gws slides delete-text <id>` | Clear text from shape or speaker notes (`--object-id` or `--notes`/`--slide-number`) |
| `gws slides update-text-style <id>` | Style text (`--object-id`, `--bold`, `--italic`, `--font-size`, `--color`) |
| `gws slides update-transform <id>` | Move/scale/rotate element (`--object-id`, `--x`, `--y`, `--scale-x`, `--rotate`) |
| `gws slides create-table <id>` | Add table (`--slide-id/--slide-number`, `--rows`, `--cols`) |
| `gws slides insert-table-rows <id>` | Insert rows (`--table-id`, `--at`, `--count`) |
| `gws slides delete-table-row <id>` | Delete row (`--table-id`, `--row`) |
| `gws slides update-table-cell <id>` | Style cell (`--table-id`, `--row`, `--col`, `--background-color`) |
| `gws slides update-table-border <id>` | Style border (`--table-id`, `--row`, `--col`, `--border`, `--color`) |
| `gws slides update-paragraph-style <id>` | Paragraph style (`--object-id`, `--alignment`, `--line-spacing`) |
| `gws slides update-shape <id>` | Shape properties (`--object-id`, `--background-color`, `--outline-color`) |
| `gws slides reorder-slides <id>` | Reorder slides (`--slide-ids`, `--to`) |
| `gws slides update-slide-background <id>` | Set slide background (`--slide-id/--slide-number`, `--color` or `--image-url`) |
| `gws slides list-layouts <id>` | List available layouts from presentation masters |
| `gws slides add-line <id>` | Add line/connector (`--slide-id/--slide-number`, `--type`, `--start-x/y`, `--end-x/y`) |
| `gws slides group <id>` | Group elements (`--object-ids`) |
| `gws slides ungroup <id>` | Ungroup elements (`--group-id`) |
| `gws slides thumbnail <id>` | Get slide thumbnail (`--slide`, `--size`, `--download`) |

### Chat

> Requires [Chat App configuration](https://console.cloud.google.com/apis/api/chat.googleapis.com/hangouts-chat) in Google Cloud Console.

| Command | Description |
|---------|-------------|
| `gws chat list` | List spaces (`--filter`, `--page-size`) |
| `gws chat messages <space>` | List messages (`--max`, `--filter`, `--order-by`, `--show-deleted`) |
| `gws chat members <space>` | List members with display names + emails via People API (`--max`, `--filter`, `--show-groups`, `--show-invited`) |
| `gws chat send` | Send message (`--space`, `--text`) |
| `gws chat get <message>` | Get a single message |
| `gws chat update <message>` | Update message text (`--text`) |
| `gws chat delete <message>` | Delete a message (`--force`) |
| `gws chat reactions <message>` | List reactions (`--filter`, `--page-size`) |
| `gws chat react <message>` | Add emoji reaction (`--emoji`) |
| `gws chat unreact <reaction>` | Remove a reaction |
| `gws chat get-space <space>` | Get space details |
| `gws chat create-space` | Create a space (`--display-name`, `--type`, `--description`) |
| `gws chat delete-space <space>` | Delete a space |
| `gws chat update-space <space>` | Update a space (`--display-name`, `--description`) |
| `gws chat search-spaces` | Search spaces — admin only (`--query`, `--page-size`) |
| `gws chat find-dm` | Find DM space with a user (`--user`) |
| `gws chat setup-space` | Create space with initial members (`--display-name`, `--members`) |
| `gws chat get-member <member>` | Get member details |
| `gws chat add-member <space>` | Add a member (`--user`, `--role`) |
| `gws chat remove-member <member>` | Remove a member |
| `gws chat update-member <member>` | Update member role (`--role`) |
| `gws chat read-state <space>` | Get space read state |
| `gws chat mark-read <space>` | Mark space as read (`--time`) |
| `gws chat thread-read-state <thread>` | Get thread read state |
| `gws chat attachment <attachment>` | Get attachment metadata |
| `gws chat upload <space>` | Upload a file (`--file`) |
| `gws chat download <resource>` | Download media (`--output`) |
| `gws chat events <space>` | List space events (`--filter`, `--page-size`) |
| `gws chat event <event>` | Get event details |

### Forms

> Requires enabling the [Google Forms API](https://console.cloud.google.com/apis/api/forms.googleapis.com).

| Command | Description |
|---------|-------------|
| `gws forms info <id>` | Form structure and questions |
| `gws forms responses <id>` | All form responses with answers |

### Contacts

| Command | Description |
|---------|-------------|
| `gws contacts list` | List contacts (`--max`) |
| `gws contacts search <query>` | Search contacts by name/email/phone |
| `gws contacts get <resource-name>` | Get contact details |
| `gws contacts create` | Create contact (`--name`, `--email`, `--phone`) |
| `gws contacts delete <resource-name>` | Delete a contact |

### Custom Search

> Requires a [Programmable Search Engine](https://programmablesearchengine.google.com/) ID and API key.

| Command | Description |
|---------|-------------|
| `gws search <query>` | Search the web (`--max`, `--site`, `--type`) |

## Structured Output

Every command returns JSON by default for machine consumption:

```bash
$ gws calendar events --days 1
{
  "count": 3,
  "events": [
    {"id": "abc123", "summary": "Team standup", "start": "2024-01-15T09:00:00Z"},
    ...
  ]
}
```

## Development

### Project Layout

```
cmd/              # Cobra command implementations
internal/
  auth/           # OAuth2 + PKCE flow, token management
  client/         # Lazy-initialized Google API service factory
  config/         # Viper configuration and path resolution
  printer/        # JSON, YAML, and text output formatters
main.go           # Entry point
```

### Building & Testing

```bash
make build      # Build binary to ./bin/gws
make test       # Run unit tests
make test-race  # Run tests with race detector
make vet        # Static analysis
make fmt        # Format code
make tidy       # Tidy go modules
```

## Credential Storage

| File | Permissions | Contents |
|------|-------------|----------|
| `~/.config/gws/config.yaml` | `0600` | OAuth client ID/secret, preferences |
| `~/.config/gws/token.json` | `0600` | OAuth access/refresh tokens |

**Note:** After upgrading `gws` to a version with new features (e.g., Docs/Slides write commands), you may need to re-authenticate to grant the new OAuth scopes:

```bash
gws auth logout && gws auth login
```

## Claude Code Plugin

`gws` ships with a Claude Code plugin that teaches Claude how to use every command. Install it to get context-aware help for all 10+ services:

```bash
# From Claude Code, run:
/install-plugin omriariav/workspace-cli
```

The plugin includes 11 skills (one per Google service + auth setup), each with quick reference tables, detailed flag documentation, and AI agent tips.

## License

`gws` is available under the [MIT License](LICENSE).
