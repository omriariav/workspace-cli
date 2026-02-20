# gws Roadmap

Feature roadmap for the Google Workspace CLI. Items are organized by priority and complexity.

## Legend

- **P1** - High priority, significant user value
- **P2** - Medium priority, nice to have
- **P3** - Low priority, future consideration
- **Complexity**: Simple (S), Medium (M), Complex (C)

---

## Completed

### v1.26.0
- [x] Groups: list groups, list members via Admin Directory API (2 new commands, PR #133)
- [x] Keep: list notes, get note, create note via Keep API (3 new commands, PR #134)
- [x] Auth: added admin.directory.group and keep scopes
- [x] CI: Codex PR review workflow (openai/codex-action)

### v1.25.0
- [x] Sheets: Named Ranges — add, list, delete (3 new commands, PR #127)
- [x] Sheets: Filters — set basic filter, clear filter, add filter view (3 new commands, PR #128)
- [x] Sheets: Charts — add-chart, list-charts, delete-chart (3 new commands, PR #129)
- [x] Sheets: Conditional Formatting — add, list, delete rules (3 new commands, PR #130)

### v1.24.0
- [x] Contacts: full People API parity — update, batch-create/update/delete, directory search, photos, resolve (9 new commands, PR #125)
- [x] Forms: full API parity — create, update, get, response (4 new commands, PR #124)
- [x] Auth: added forms.body and directory.readonly scopes

### v1.23.0
- [x] Docs: trash command with permanent delete option (PR #106)
- [x] Sheets: copy-to, batch-read, batch-write (PR #105)
- [x] Slides: page thumbnails with download (PR #107)

### v1.22.0
- [x] Gmail: full API parity — drafts CRUD, label CRUD, batch-modify/batch-delete, thread trash/untrash/delete, attachment download (18 new commands, PR #101)
- [x] Calendar: full API parity — event get/quick-add/instances/move, calendar CRUD, ACL/sharing, subscriptions, freebusy, colors, settings (20 new commands, PR #103)
- [x] Drive: full API parity — permissions/sharing, revisions, replies, comments CRUD, export, empty-trash, update metadata, shared drives CRUD, about, changes (25 new commands, PR #102)
- [x] Auth: broader scopes for gmail.settings, full calendar, full drive

### v1.21.0
- [x] Tasks: full API parity — task list CRUD (create-list, update-list, delete-list), task get/delete, move/reorder, clear completed (8 new commands, PR #100)

### v1.20.0
- [x] Chat: full API parity — space CRUD, member management, read states, reactions, file upload/download, space events (28 new commands, PR #99)

### v1.15.0
- [x] Slides: update-slide-background (solid color + image URL)
- [x] Slides: list-layouts (discover custom master layouts)
- [x] Slides: add-slide --layout-id (custom layout support)
- [x] Slides: add-line (lines and connectors with position, color, weight)
- [x] Slides: group / ungroup elements
- [x] Slides: replace-text --slide-number/--slide-id (slide-scoped replacement)
- [x] Slides: replace-text --object-id (element-level replacement via delete+insert)
- [x] Slides: read --elements (expose element IDs, types, text per slide)
- [x] Drive: copy command (duplicate files / templates)
- [x] Drive: comments pagination + anchor data for Slides
- [x] Chat: fix empty sender field (fallback to resource name, add sender_type)

### v1.14.0
- [x] Contacts / People API: list, search, get, create, delete
- [x] Sheets Formatting: format cells, set-column-width, set-row-height, freeze panes
- [x] Docs Formatting & Lists: format text, set-paragraph-style, add-list, remove-list

### v1.13.0
- [x] Add YAML output format (`--format yaml`) alongside existing JSON and text

### v1.12.0
- [x] Add golangci-lint with `.golangci.yml` config (replaces basic `go vet` in CI and Makefile)
- [x] Fix lint findings across codebase (unchecked errors, unused consts, gofmt, staticcheck)
- [x] Add test coverage for chat, forms, and search commands

### v1.11.0
- [x] Slides: `--notes` flag on `info`, `list`, `read` to include speaker notes in output
- [x] Slides: `--notes` mode on `add-text` and `delete-text` to write/clear speaker notes (with `--slide-id` or `--slide-number`)

### v0.7.0
- [x] `gws drive create-folder` - Create folder
- [x] `gws drive move` - Move file to folder
- [x] `gws drive delete` - Delete/trash file

### v0.6.0
- [x] Sheets: insert-rows, delete-rows, insert-cols, delete-cols
- [x] Sheets: rename-sheet, duplicate-sheet, merge, unmerge, sort, find-replace
- [x] Slides: add-shape, add-image, add-text, replace-text
- [x] Docs: delete, add-table

### v0.5.0
- [x] Sheets: add-sheet, delete-sheet, clear
- [x] Docs: insert, replace
- [x] Slides: delete-slide, duplicate-slide

### v0.4.0
- [x] Drive: upload
- [x] Sheets: create, write, append

---

## Remaining API Gaps

### Gmail Settings API (P2, M) — #104

Vacation responder, filters, forwarding rules, send-as aliases, IMAP/POP settings.

```bash
gws gmail vacation --enable --subject "OOO" --body "Back Jan 5"
gws gmail filters
gws gmail create-filter --from "noreply@" --action archive
gws gmail forwarding --add user@example.com
gws gmail send-as --list
```

---

## New Services

### Classroom (P3, M) — #121

Access Google Classroom courses and assignments.

```bash
gws classroom courses
# API: courses.list

gws classroom assignments <course-id>
# API: courseWork.list

gws classroom submissions <course-id> <coursework-id>
# API: studentSubmissions.list
```

### Apps Script (P3, M) — #122

Inspect and execute Google Apps Script projects.

```bash
gws apps-script list
# API: projects.list

gws apps-script get <script-id>
# API: projects.getContent

gws apps-script run <script-id> --function "myFunction"
# API: scripts.run
```

---

## CLI Infrastructure

### OS keychain token storage (P2, M) — #112

Store OAuth tokens in OS keychain (macOS Keychain, Linux Secret Service) instead of plain JSON file at `~/.config/gws/token.json`. Both `bkt` ([go-keyring](https://github.com/zalando/go-keyring)) and `gogcli` use keyring-based credential storage.

### Multi-account support (P2, C) — #113

Support multiple Google accounts with context switching, similar to `jk context use`, `bkt context create`, and `gogcli`'s email alias system.

```bash
gws context add work --client-id=xxx
gws context add personal --client-id=yyy
gws context use work
# or: GWS_CONTEXT=personal gws gmail list
```

### Scoped authentication (P2, S) — #114

Least-privilege auth — `gws auth login --services gmail,calendar` already exists. Could add `--readonly` for read-only scopes only.

### Homebrew distribution (P2, S) — #115

Publish gws via Homebrew tap for easy installation.

```bash
brew install omriariav/tap/gws
```

### Service account support (P3, M) — #116

Support Google Workspace domain-wide delegation via service accounts, enabling server-side automation without interactive OAuth.

```bash
gws auth login --service-account key.json --subject admin@company.com
```

### jq / Go template filtering (P3, M) — #117

Add `--jq` and `--template` flags for output filtering.

```bash
gws gmail list --max 5 --jq '.[].subject'
gws calendar events --template '{{range .}}{{.summary}} at {{.start}}{{end}}'
```

### Extension / plugin system (P3, C) — #118

Allow custom commands via git-cloned extensions.

```bash
gws extension install github.com/user/gws-calendar-sync
gws calendar-sync  # runs extension
```

### Cross-service batch operations (P3, C) — #123

Bulk operations across multiple files/items.

```bash
gws drive batch-delete --query "trashed=true"
gws gmail batch-archive --query "older_than:30d"
```

---

## Contributing

When implementing new features:

1. Follow existing patterns in `cmd/{service}.go`
2. Add tests in `cmd/{service}_test.go`
3. Update command list in `TestXxxCommands` in `commands_test.go`
4. Update README.md command table
5. Bump version in Makefile

See `CLAUDE.md` for architecture details.
