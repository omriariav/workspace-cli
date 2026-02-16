# gws Roadmap

Feature roadmap for the Google Workspace CLI. Items are organized by priority and complexity.

## Legend

- **P1** - High priority, significant user value
- **P2** - Medium priority, nice to have
- **P3** - Low priority, future consideration
- **Complexity**: Simple (S), Medium (M), Complex (C)

---

## Completed

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

## Planned Features

### Sheets Charts (P2, C)

Create and manage charts in spreadsheets.

```bash
gws sheets add-chart <id> --type PIE --data "Sheet1!A1:B10" --title "Sales"
# API: batchUpdate → addChart
# Types: PIE, BAR, LINE, AREA, COLUMN, SCATTER

gws sheets list-charts <id>
# API: spreadsheets.get with fields=sheets.charts

gws sheets delete-chart <id> --chart-id 123456
# API: batchUpdate → deleteEmbeddedObject
```

### Sheets Named Ranges (P2, S)

Manage named ranges for easier formula references.

```bash
gws sheets add-named-range <id> --name "SalesData" --range "Sheet1!A1:D100"
# API: batchUpdate → addNamedRange

gws sheets list-named-ranges <id>
# API: spreadsheets.get with fields=namedRanges

gws sheets delete-named-range <id> --name "SalesData"
# API: batchUpdate → deleteNamedRange
```

### Sheets Filters (P2, M)

Add and manage filter views.

```bash
gws sheets add-filter <id> <range>
# API: batchUpdate → setBasicFilter

gws sheets clear-filter <id> --sheet "Sheet1"
# API: batchUpdate → clearBasicFilter

gws sheets add-filter-view <id> --name "Q1 Data" --range "Sheet1!A1:D100"
# API: batchUpdate → addFilterView
```

### Sheets Conditional Formatting (P2, M)

Apply conditional formatting rules.

```bash
gws sheets conditional-format <id> <range> --rule ">" --value 100 --bg-color green
# API: batchUpdate → addConditionalFormatRule

gws sheets list-conditional-formats <id> --sheet "Sheet1"
# API: spreadsheets.get with fields=sheets.conditionalFormats

gws sheets delete-conditional-format <id> --index 0 --sheet "Sheet1"
# API: batchUpdate → deleteConditionalFormatRule
```

### Slides Advanced (P2, M)

Additional slide manipulation commands.

```bash
gws slides delete-object <id> --object-id "g123abc"
# API: batchUpdate → deleteObject

gws slides update-shape <id> --object-id "g123abc" --fill-color "#FF0000"
# API: batchUpdate → updateShapeProperties

gws slides reorder <id> --slide-id "g123abc" --position 2
# API: batchUpdate → updateSlidesPosition

gws slides update-text-style <id> --object-id "g123abc" --bold --font-size 24
# API: batchUpdate → updateTextStyle
```

### Gmail Advanced (P2, S)

Additional Gmail operations.

```bash
gws gmail labels
# API: users.labels.list

gws gmail label <message-id> --add "Important" --remove "Inbox"
# API: users.messages.modify

gws gmail trash <message-id>
# API: users.messages.trash

gws gmail archive <message-id>
# API: users.messages.modify (remove INBOX label)
```

### Calendar Advanced (P2, S)

Additional calendar operations.

```bash
gws calendar update <event-id> --title "New Title" --start "2024-01-20T10:00:00"
# API: events.patch

gws calendar delete <event-id>
# API: events.delete

gws calendar rsvp <event-id> --response accepted
# API: events.patch (attendees[].responseStatus)
```

### Tasks Advanced (P2, S)

Additional task operations.

```bash
gws tasks update <list-id> <task-id> --title "New Title" --due "2024-01-20"
# API: tasks.patch

gws tasks delete <list-id> <task-id>
# API: tasks.delete

gws tasks move <list-id> <task-id> --parent <parent-task-id>
# API: tasks.move
```

---

## New Services (Competitive Gaps — gogcli)

Identified from comparison with [gogcli](https://github.com/steipete/gogcli).

### Groups (P3, S) — Workspace only

List and inspect Google Groups.

```bash
gws groups list
# API: directory.groups.list (Admin SDK)

gws groups members <group-email>
# API: directory.members.list
```

### Keep (P3, M) — Workspace only

Access Google Keep notes (Workspace accounts only).

```bash
gws keep list --max 20
# API: notes.list

gws keep get <note-id>
# API: notes.get

gws keep create --title "TODO" --text "Buy milk"
# API: notes.create
```

### Classroom (P3, M)

Access Google Classroom courses and assignments.

```bash
gws classroom courses
# API: courses.list

gws classroom assignments <course-id>
# API: courseWork.list

gws classroom submissions <course-id> <coursework-id>
# API: studentSubmissions.list
```

### Apps Script (P3, M)

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

## CLI Infrastructure (Competitive Gaps)

Identified from comparison with [jenkins-cli](https://github.com/avivsinai/jenkins-cli) and [bitbucket-cli](https://github.com/avivsinai/bitbucket-cli).

### OS keychain token storage (P2, M)

Store OAuth tokens in OS keychain (macOS Keychain, Linux Secret Service) instead of plain JSON file at `~/.config/gws/token.json`. Both `bkt` ([go-keyring](https://github.com/zalando/go-keyring)) and `gogcli` use keyring-based credential storage.

### Multi-account support (P2, C)

Support multiple Google accounts with context switching, similar to `jk context use`, `bkt context create`, and `gogcli`'s email alias system.

```bash
gws context add work --client-id=xxx
gws context add personal --client-id=yyy
gws context use work
# or: GWS_CONTEXT=personal gws gmail list
```

### Scoped authentication (P2, S)

Least-privilege auth with `--readonly` and `--drive-scope` flags to request only necessary permissions during login. gogcli implements this to avoid over-scoping. Currently gws requests all scopes upfront.

```bash
gws auth login --readonly                    # read-only access
gws auth login --scopes gmail,calendar       # specific services only
```

### Homebrew distribution (P2, S)

Publish gws via Homebrew tap for easy installation. gogcli uses `brew install steipete/tap/gogcli`.

```bash
brew install omriariav/tap/gws
```

### Service account support (P3, M)

Support Google Workspace domain-wide delegation via service accounts, enabling server-side automation without interactive OAuth. gogcli supports this for Workspace admins.

```bash
gws auth login --service-account key.json --subject admin@company.com
```

### jq / Go template filtering (P3, M)

Add `--jq` and `--template` flags for output filtering, matching `jk`'s approach.

```bash
gws gmail list --max 5 --jq '.[].subject'
gws calendar events --template '{{range .}}{{.summary}} at {{.start}}{{end}}'
```

### Extension / plugin system (P3, C)

Allow custom commands via git-cloned extensions, similar to `bkt`'s `bkt-<name>` convention.

```bash
gws extension install github.com/user/gws-calendar-sync
gws calendar-sync  # runs extension
```

---

## Feature Ideas (P3)

### Forms Write (P3, C)

Limited by Google Forms API capabilities.

```bash
gws forms create --title "Feedback Form"
# API: forms.create (very limited)
```

Note: Forms API is primarily read-only. Programmatic form creation is limited.

### Drive Sharing (P3, M)

Manage file permissions.

```bash
gws drive share <file-id> --email user@example.com --role writer
# API: permissions.create

gws drive unshare <file-id> --email user@example.com
# API: permissions.delete

gws drive permissions <file-id>
# API: permissions.list
```

### Batch Operations (P3, C)

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
