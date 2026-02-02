# gws

<p align="center"><em>Unified CLI for Google Workspace — Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, and more from your terminal.</em></p>

<p align="center">
  <a href="https://github.com/omriariav/workspace-cli/actions/workflows/ci.yml"><img src="https://github.com/omriariav/workspace-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="go.mod"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8.svg" alt="Go Version"></a>
</p>

`gws` gives developers and AI agents a structured, token-efficient interface to 10+ Google Workspace services. Every command returns consistent JSON (or human-readable text), making it ideal for scripting, automation, and agent toolchains.

**Built for AI & automation:** Drop `gws` into Claude Code, Codex, or shell scripts and they inherit structured output, predictable flags, and safe defaults — no wrapper code required.

## Features

- **10+ Google services** — Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, Chat, Forms, Custom Search.
- **Scriptable output** — `--format json` (default) or `--format text` for human-readable tables.
- **OAuth2 + PKCE** — Secure browser-based auth with automatic token refresh and `0600` file permissions.
- **Single auth flow** — Authenticate once to access all services; all scopes requested upfront.
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

Add `--format text` to any command for human-readable output.

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
| `gws gmail list` | List threads with `thread_id` and `message_id` (`--max`, `--query`) |
| `gws gmail read <id>` | Read message body and headers |
| `gws gmail thread <id>` | Read full thread conversation |
| `gws gmail send` | Send email (`--to`, `--subject`, `--body`, `--cc`, `--bcc`, `--thread-id`, `--reply-to-message-id`) |
| `gws gmail reply <id>` | Reply to message (`--body`, `--cc`, `--bcc`, `--all`) |
| `gws gmail event-id <id>` | Extract calendar event ID from invite email |
| `gws gmail labels` | List all labels |
| `gws gmail label <id>` | Add/remove labels (`--add`, `--remove`) |
| `gws gmail archive <id>` | Archive message (remove from inbox) |
| `gws gmail trash <id>` | Move message to trash |

### Calendar

| Command | Description |
|---------|-------------|
| `gws calendar list` | List all calendars |
| `gws calendar events` | List upcoming events (`--days`, `--calendar-id`, `--max`) |
| `gws calendar create` | Create event (`--title`, `--start`, `--end`, `--attendees`) |
| `gws calendar update <id>` | Update event (`--title`, `--start`, `--end`, `--add-attendees`) |
| `gws calendar delete <id>` | Delete event |
| `gws calendar rsvp <id>` | RSVP to invite (`--response accepted/declined/tentative`, `--message`) |

### Tasks

| Command | Description |
|---------|-------------|
| `gws tasks lists` | List task lists |
| `gws tasks list <id>` | List tasks in a list (`--show-completed`) |
| `gws tasks create` | Create task (`--title`, `--tasklist`, `--due`); accepts YYYY-MM-DD dates |
| `gws tasks complete <list> <task>` | Mark task as done |

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

### Slides

| Command | Description |
|---------|-------------|
| `gws slides info <id>` | Presentation metadata |
| `gws slides list <id>` | List slides with text content |
| `gws slides read <id> [n]` | Read slide text (specific or all) |
| `gws slides create` | Create new presentation (`--title`) |
| `gws slides add-slide <id>` | Add slide (`--title`, `--body`, `--layout`) |
| `gws slides delete-slide <id>` | Delete slide (`--slide-id` or `--slide-number`) |
| `gws slides duplicate-slide <id>` | Duplicate slide (`--slide-id` or `--slide-number`) |
| `gws slides add-shape <id>` | Add shape (`--slide-id/--slide-number`, `--type`, `--x`, `--y`, `--width`, `--height`) |
| `gws slides add-image <id>` | Add image (`--slide-id/--slide-number`, `--url`, `--x`, `--y`, `--width`) |
| `gws slides add-text <id>` | Insert text into object (`--object-id`, `--text`, `--at`) |
| `gws slides replace-text <id>` | Find and replace text (`--find`, `--replace`, `--match-case`) |
| `gws slides delete-object <id>` | Delete any page element (`--object-id`) |
| `gws slides delete-text <id>` | Clear text from shape (`--object-id`, `--from`, `--to`) |
| `gws slides update-text-style <id>` | Style text (`--object-id`, `--bold`, `--italic`, `--font-size`, `--color`) |
| `gws slides update-transform <id>` | Move/resize element (`--object-id`, `--x`, `--y`, `--scale-x`, `--rotate`) |
| `gws slides create-table <id>` | Add table (`--slide-id/--slide-number`, `--rows`, `--cols`) |
| `gws slides insert-table-rows <id>` | Insert rows (`--table-id`, `--at`, `--count`) |
| `gws slides delete-table-row <id>` | Delete row (`--table-id`, `--row`) |
| `gws slides update-table-cell <id>` | Style cell (`--table-id`, `--row`, `--col`, `--background-color`) |
| `gws slides update-table-border <id>` | Style border (`--table-id`, `--row`, `--col`, `--border`, `--color`) |
| `gws slides update-paragraph-style <id>` | Paragraph style (`--object-id`, `--alignment`, `--line-spacing`) |
| `gws slides update-shape <id>` | Shape properties (`--object-id`, `--background-color`, `--outline-color`) |
| `gws slides reorder-slides <id>` | Reorder slides (`--slide-ids`, `--to`) |

### Chat

> Requires [Chat App configuration](https://console.cloud.google.com/apis/api/chat.googleapis.com/hangouts-chat) in Google Cloud Console.

| Command | Description |
|---------|-------------|
| `gws chat list` | List spaces |
| `gws chat messages <space>` | List messages in a space |
| `gws chat send` | Send message (`--space`, `--text`) |

### Forms

> Requires enabling the [Google Forms API](https://console.cloud.google.com/apis/api/forms.googleapis.com).

| Command | Description |
|---------|-------------|
| `gws forms info <id>` | Form structure and questions |
| `gws forms responses <id>` | All form responses with answers |

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
  printer/        # JSON and text output formatters
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
