# gws - Google Workspace CLI

## Project Overview

`gws` is a unified CLI for Google Workspace built in Go (Cobra/Viper). It provides structured JSON output for 10+ Google services, designed for AI agents and scripting.

## Architecture

- **Entry point**: `cmd/gws/main.go` (also `main.go` at root for `go run .`)
- **Commands**: `cmd/` — one file per service (gmail.go, calendar.go, etc.)
- **Auth**: `internal/auth/` — OAuth2 + PKCE, token stored at `~/.config/gws/token.json`
- **Clients**: `internal/client/factory.go` — lazy-initialized, mutex-protected service clients
- **Config**: `internal/config/` — Viper-based, reads from env (`GWS_*`) or `~/.config/gws/config.yaml`
- **Output**: `internal/printer/` — JSON (default) or text format via `--format` flag

## Available Commands

| Service | Commands |
|---------|----------|
| `auth` | login, logout, status |
| `gmail` | list, read, thread, send, labels, label, archive, trash |
| `calendar` | list, events, create, update, delete, rsvp |
| `tasks` | lists, list, create, complete |
| `drive` | list, search, info, download, upload, create-folder, move, delete, comments |
| `docs` | read, info, create, append, insert, replace, delete, add-table |
| `sheets` | info, list, read, create, write, append, add-sheet, delete-sheet, clear, insert-rows, delete-rows, insert-cols, delete-cols, rename-sheet, duplicate-sheet, merge, unmerge, sort, find-replace |
| `slides` | info, list, read, create, add-slide, delete-slide, duplicate-slide, add-shape, add-image, add-text, replace-text |
| `chat` | list, messages, send (needs Chat App config) |
| `forms` | info, responses |
| `search` | web search (needs API key) |
| `version` | Show version info |

## Building & Running

```bash
make build          # ./bin/gws
go run ./cmd/gws    # or go run .
```

## Credentials

- Client ID/Secret: env vars `GWS_CLIENT_ID`, `GWS_CLIENT_SECRET` or config file
- Token: `~/.config/gws/token.json` (auto-refreshes)
- All scopes requested upfront in `internal/auth/scopes.go`

## Current Version

**v1.2.2** - Fix `go install` version detection via debug.ReadBuildInfo(). Full CRUD for all services.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features including:
- Sheets: formatting, charts, named ranges, filters, conditional formatting
- Docs: text formatting, lists
- Slides: delete-object, update-shape, reorder
- Gmail/Calendar/Tasks: additional operations
- `/morning` command for daily briefings

## Implementation Patterns

When adding new commands:

1. Add command definition and `runXxx` function to `cmd/{service}.go`
2. Register in `init()` with flags
3. Add tests to `cmd/{service}_test.go` using `httptest.Server`
4. Add command name to `TestXxxCommands` in `cmd/commands_test.go`
5. Update README.md command table
6. Bump version in Makefile
