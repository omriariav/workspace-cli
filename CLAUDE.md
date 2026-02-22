# gws - Google Workspace CLI

## Project Overview

`gws` is a unified CLI for Google Workspace built in Go (Cobra/Viper). It provides structured JSON output for 12 Google services, designed for AI agents and scripting.

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
| `drive` | list, search, info, download, upload, create-folder, move, delete, comments, permissions, share, unshare, permission, update-permission, revisions, revision, delete-revision, replies, reply, get-reply, delete-reply, comment, add-comment, delete-comment, export, empty-trash, update, shared-drives, shared-drive, create-drive, delete-drive, update-drive, about, changes, activity |
| `docs` | read, info, create, append, insert, replace, delete, add-table, format, set-paragraph-style, add-list, remove-list, trash |
| `sheets` | info, list, read, create, write, append, add-sheet, delete-sheet, clear, insert-rows, delete-rows, insert-cols, delete-cols, rename-sheet, duplicate-sheet, merge, unmerge, sort, find-replace, format, set-column-width, set-row-height, freeze, copy-to, batch-read, batch-write, add-named-range, list-named-ranges, delete-named-range, add-filter, clear-filter, add-filter-view, add-chart, list-charts, delete-chart, add-conditional-format, list-conditional-formats, delete-conditional-format |
| `slides` | info, list, read, create, add-slide, delete-slide, duplicate-slide, add-shape, add-image, add-text, replace-text, delete-object, delete-text, update-text-style, update-transform, create-table, insert-table-rows, delete-table-row, update-table-cell, update-table-border, update-paragraph-style, update-shape, reorder-slides, thumbnail |
| `chat` | list, messages, members, send, get, update, delete, reactions, react, unreact, get-space, create-space, delete-space, update-space, search-spaces, find-dm, setup-space, get-member, add-member, remove-member, update-member, read-state, mark-read, thread-read-state, attachment, upload, download, events, event |
| `contacts` | list, search, get, create, delete, update, batch-create, batch-update, batch-delete, directory, directory-search, photo, delete-photo, resolve |
| `groups` | list, members |
| `keep` | list, get, create |
| `forms` | info, get, responses, response, create, update |
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
- Groups requires Admin SDK API enabled + Workspace admin privileges
- Keep requires Keep API enabled + Workspace Enterprise plan

## Current Version

**v1.27.0** - Drive Activity API v2 command (query file/folder activity history).

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features. Remaining items:
- Gmail: Settings API — vacation, filters, forwarding, send-as (#104)
- Classroom: courses, assignments, submissions (#121)
- Apps Script: list, get, run (#122)
- Infrastructure: keychain storage, multi-account, Homebrew, jq filtering

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
4. **Codex review loop** — Codex CI reviews automatically; iterate on warnings/issues until the review is clean (no warnings, no critical/suggestion items that need fixing)
5. **Merge** — `gh pr merge <N> --squash` only after Codex approves
6. **Release** — Follow the release checklist below immediately after merge
7. **Tweet** — `/tweet` announcing the new version and key changes

## Release Checklist

Run these steps immediately after merging a PR. Do not skip any step.

```
1. git checkout main && git pull --rebase
2. Edit Makefile: bump VERSION (e.g. 1.26.0 → 1.27.0)
3. Edit CLAUDE.md: update "Current Version" line
4. git add Makefile CLAUDE.md
5. git commit -m "release: vX.Y.Z — <short description>"
6. git push
7. git tag vX.Y.Z && git push origin vX.Y.Z
8. gh release create vX.Y.Z --title "vX.Y.Z — <title>" --notes "<release notes>"
9. Build and upload binaries:
   GOOS=darwin GOARCH=arm64 go build -ldflags "..." -o bin/release/gws-darwin-arm64 ./cmd/gws
   GOOS=darwin GOARCH=amd64 go build -ldflags "..." -o bin/release/gws-darwin-amd64 ./cmd/gws
   GOOS=linux  GOARCH=amd64 go build -ldflags "..." -o bin/release/gws-linux-amd64 ./cmd/gws
   GOOS=linux  GOARCH=arm64 go build -ldflags "..." -o bin/release/gws-linux-arm64 ./cmd/gws
   gh release upload vX.Y.Z bin/release/gws-* --clobber
10. /tweet about the release
```

Lesson learned (v1.27.0): forgetting the release step means GitHub releases page is stale and users see outdated versions. Always release immediately after merge. Always upload binaries — source-only releases are not downloadable.

## Parallel Agent Sprints

For large multi-service features, use Claude Code agent teams with git worktrees:

1. **Pre-commit shared changes** on `main` before branching (e.g. scopes.go) to avoid merge conflicts
2. **Create worktrees** — `git worktree add /tmp/{agent}-work feat/{service}` — one per agent so they don't clobber each other's working directory
3. **Spawn agents** with `cwd` pointing to their worktree
4. **Each agent**: implement → test → commit → push → open PR
5. **Codex CI reviews** each PR automatically; agent fixes findings and pushes
6. **Merge sequentially** (simplest first), rebase later PRs on updated main
7. **Version bump** after all PRs merge

Lesson learned (v1.22.0): agents sharing one working directory causes git branch switches to wipe uncommitted work. Always use separate worktrees.

Lesson learned (v1.26.0): when branching from main with shared test expectations (e.g. expected subcommands list), each branch must only expect its own new commands — the full list is restored after merge.

## PR Review Policy

**No PR may be merged without a passing Codex review.**

### Automated review (CI)
- `openai/codex-action` runs automatically on every PR open/update
- Codex posts review comments on the PR
- Review prompt lives in `.github/codex/prompts/review.md`

### Agent team workflow
1. Implementation agent: branch > implement > test > commit > push > `gh pr create`
2. CI triggers Codex review automatically
3. If changes requested: implementation agent fixes issues, pushes (triggers re-review)
4. Only merge after reviewer approves: `gh pr merge <N> --squash`

### Rules
- NEVER run `gh pr merge` without a prior review on the latest commit
- Implementation agent must NOT self-review (Codex provides independent review)
- For local pre-flight review: run `/code-review` before pushing
