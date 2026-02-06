# gws Roadmap

Feature roadmap for the Google Workspace CLI. Items are organized by priority and complexity.

## Legend

- **P1** - High priority, significant user value
- **P2** - Medium priority, nice to have
- **P3** - Low priority, future consideration
- **Complexity**: Simple (S), Medium (M), Complex (C)

---

## Completed

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

### Sheets Formatting (P2, M)

Cell and range formatting capabilities.

```bash
gws sheets format <id> <range> --bold --italic --bg-color "#FFFF00"
# API: batchUpdate → repeatCell with CellFormat

gws sheets set-column-width <id> --sheet "Sheet1" --col A --width 200
# API: batchUpdate → updateDimensionProperties

gws sheets set-row-height <id> --sheet "Sheet1" --row 1 --height 50
# API: batchUpdate → updateDimensionProperties

gws sheets freeze <id> --sheet "Sheet1" --rows 1 --cols 1
# API: batchUpdate → updateSheetProperties (gridProperties.frozenRowCount/frozenColumnCount)
```

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

### Docs Formatting (P2, M)

Text formatting in documents.

```bash
gws docs format <id> --from 10 --to 50 --bold --italic --font-size 14
# API: batchUpdate → updateTextStyle

gws docs set-paragraph-style <id> --from 10 --to 100 --alignment CENTER --line-spacing 1.5
# API: batchUpdate → updateParagraphStyle
```

### Docs Lists (P2, M)

Create and manage lists.

```bash
gws docs add-list <id> --at 10 --type bullet --items "Item1;Item2;Item3"
# API: batchUpdate → insertText + createParagraphBullets

gws docs remove-list <id> --from 10 --to 50
# API: batchUpdate → deleteParagraphBullets
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

## CLI Infrastructure (Competitive Gaps)

Identified from comparison with [jenkins-cli](https://github.com/avivsinai/jenkins-cli) and [bitbucket-cli](https://github.com/avivsinai/bitbucket-cli).

### Add golangci-lint (P1, S)

Replace basic `go vet` with golangci-lint for comprehensive static analysis. Both `jk` and `bkt` use it.

```makefile
lint:
	golangci-lint run ./...
```

### Add YAML output format (P1, M)

Add `--format yaml` alongside existing JSON and text. Both `jk` and `bkt` support YAML output.

```bash
gws gmail list --max 5 --format yaml
```

### OS keychain token storage (P2, M)

Store OAuth tokens in OS keychain (macOS Keychain, Linux Secret Service) instead of plain JSON file at `~/.config/gws/token.json`. `bkt` uses [go-keyring](https://github.com/zalando/go-keyring) for this.

### Multi-account support (P2, C)

Support multiple Google accounts with context switching, similar to `jk context use` and `bkt context create`.

```bash
gws context add work --client-id=xxx
gws context add personal --client-id=yyy
gws context use work
# or: GWS_CONTEXT=personal gws gmail list
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
