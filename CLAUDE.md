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
| `gmail` | list, read, thread, send, reply, labels, label, archive, trash, event-id, untrash, delete, batch-modify, batch-delete, trash-thread, untrash-thread, delete-thread, label-info, create-label, update-label, delete-label, drafts, draft, create-draft, update-draft, send-draft, delete-draft, attachment |
| `calendar` | list, events, create, update, delete, rsvp, get, quick-add, instances, move, get-calendar, create-calendar, update-calendar, delete-calendar, clear, subscribe, unsubscribe, calendar-info, update-subscription, acl, share, unshare, update-acl, freebusy, colors, settings |
| `tasks` | lists, list, list-info, create, create-list, update, update-list, delete-list, get, delete, complete, move, clear |
| `drive` | list, search, info, download, upload, create-folder, move, delete, comments, permissions, share, unshare, permission, update-permission, revisions, revision, delete-revision, replies, reply, get-reply, delete-reply, comment, add-comment, delete-comment, export, empty-trash, update, shared-drives, shared-drive, create-drive, delete-drive, update-drive, about, changes |
| `docs` | read, info, create, append, insert, replace, delete, add-table, format, set-paragraph-style, add-list, remove-list, trash |
| `sheets` | info, list, read, create, write, append, add-sheet, delete-sheet, clear, insert-rows, delete-rows, insert-cols, delete-cols, rename-sheet, duplicate-sheet, merge, unmerge, sort, find-replace, format, set-column-width, set-row-height, freeze, copy-to, batch-read, batch-write |
| `slides` | info, list, read, create, add-slide, delete-slide, duplicate-slide, add-shape, add-image, add-text, replace-text, delete-object, delete-text, update-text-style, update-transform, create-table, insert-table-rows, delete-table-row, update-table-cell, update-table-border, update-paragraph-style, update-shape, reorder-slides, thumbnail |
| `chat` | list, messages, members, send, get, update, delete, reactions, react, unreact, get-space, create-space, delete-space, update-space, search-spaces, find-dm, setup-space, get-member, add-member, remove-member, update-member, read-state, mark-read, thread-read-state, attachment, upload, download, events, event |
| `contacts` | list, search, get, create, delete |
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

**v1.23.1** - Fix batch-write/batch-read StringSlice→StringArray for JSON values with commas.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features. Remaining API parity issues:
- Gmail: Settings API — vacation, filters, forwarding, send-as, IMAP/POP (#104)
- Slides: page thumbnails (#98)
- Sheets: copy-to, batch read/write (#97)
- Docs: delete command (#96)
- Forms: create form, batch update, get single response (#95)
- Contacts: update, batch ops, directory search, photos (#94)

## Implementation Patterns

When adding new commands:

1. Add command definition and `runXxx` function to `cmd/{service}.go`
2. Register in `init()` with flags
3. Add tests to `cmd/{service}_test.go` using `httptest.Server`
4. Add command name to `TestXxxCommands` in `cmd/commands_test.go`
5. Update README.md command table
6. Bump version in Makefile

## Development Workflow

Every feature/fix follows this flow:

1. **Branch** — Create a feature branch from `main` (e.g. `feat/chat-api-params`)
2. **Implement & commit** — Make changes, commit with clear messages
3. **Open PR** — Push branch and open a PR against `main`
4. **Codex review loop** — Use Codex to review the PR; iterate on feedback until it approves
5. **Merge** — Merge the PR into `main`
6. **Release** — Bump version in Makefile, update CLAUDE.md version, tag release, update README
7. **Tweet draft** — Write a short tweet announcing the new version and key changes

## Parallel Agent Sprints

For large multi-service features, use Claude Code agent teams with git worktrees:

1. **Pre-commit shared changes** on `main` before branching (e.g. scopes.go) to avoid merge conflicts
2. **Create worktrees** — `git worktree add /tmp/{agent}-work feat/{service}` — one per agent so they don't clobber each other's working directory
3. **Spawn agents** with `cwd` pointing to their worktree
4. **Each agent**: implement → test → commit → push → open PR
5. **Review loop**: pr-reviewer agent reviews each PR, agent iterates on feedback
6. **Merge sequentially** (simplest first), rebase later PRs on updated main
7. **Version bump** after all PRs merge

Lesson learned (v1.22.0): agents sharing one working directory causes git branch switches to wipe uncommitted work. Always use separate worktrees.
