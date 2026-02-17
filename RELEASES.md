# Releases

## v1.16.0

**Security Hardening + Scoped Auth**

### Security
- Atomic token writes: temp-file + `os.Rename` prevents partial/corrupt token files
- File locking: `.lock` file with `O_CREATE|O_EXCL`, age-based stale lock cleanup (30s)
- Refresh token preservation: `MergeToken()` keeps existing refresh token when re-auth omits it
- Server-side token revocation on logout via Google's `/revoke` endpoint (best-effort)

### Scoped Authorization
- New `--services` flag on `gws auth login` — request only the scopes you need (e.g. `--services gmail,calendar,chat`)
- Service-to-scope mapping in `ServiceScopes` map with helpers (`ScopesForServices`, `ServiceForScope`)
- Config file support: set default services in `config.yaml` (`services: [gmail, calendar]`)
- Lazy scope detection: warns at runtime if a command needs scopes not yet authorized
- Backward compatible: omitting `--services` requests all scopes (existing behavior)

### New Scope
- Added `chat.memberships.readonly` — enables future `gws chat members` command

## v1.0.0

**Claude Code Plugin — Workspace Skills**

- `.claude-plugin/plugin.json` manifest for plugin distribution
- 11 per-service skills (gmail, calendar, drive, docs, sheets, slides, tasks, chat, forms, search, auth)
  - Each skill: SKILL.md with YAML frontmatter, quick reference table, detailed usage, output modes, AI agent tips
  - Each service: `references/commands.md` with exhaustive flag documentation
  - Auth skill: `references/setup-guide.md` — step-by-step GCP project + OAuth setup walkthrough
- Cross-referencing test suite (`cmd/skills_test.go`) validates skills match actual CLI commands
- README: added Claude Code Plugin install section
- Unofficial/not-endorsed-by-Google disclaimer on all skill files

## v0.9.0

**Calendar Management**

- `gws calendar update` - Update event fields (`--title`, `--start`, `--end`, `--description`, `--location`, `--add-attendees`)
  - Uses `Events.Patch` to send only changed fields (avoids unnecessary attendee notifications)
  - No-op guard when no update flags specified
- `gws calendar delete` - Delete event from calendar
- `gws calendar rsvp` - Accept/decline/tentative response to invites
  - Finds current user via `Self` attendee field
  - Client-side validation of response values

## v0.8.0

**Gmail Label Management**

- `gws gmail labels` - List all Gmail labels (system + user)
- `gws gmail label` - Add/remove labels by name (`--add`, `--remove`, comma-separated)
  - Case-insensitive label name resolution via `fetchLabelMap` + `resolveFromMap`
  - Single API call for label lookup even with both `--add` and `--remove`
- `gws gmail archive` - Archive message (removes INBOX label)
- `gws gmail trash` - Move message to trash

## v0.7.0

**Drive Write Commands**

- `gws drive create-folder` - Create folder with `--name` and optional `--parent`
- `gws drive move` - Move file to folder with `--to` flag
- `gws drive delete` - Trash file by default, `--permanent` for hard delete

## v0.6.0

**P1 Commands: Sheets, Slides, Docs Management**

### Sheets (10 new commands)
- `insert-rows` / `delete-rows` - Row dimension operations
- `insert-cols` / `delete-cols` - Column dimension operations
- `rename-sheet` / `duplicate-sheet` - Sheet management
- `merge` / `unmerge` - Cell merging
- `sort` - Sort data by column with `--has-header` support
- `find-replace` - Find and replace across sheets

### Slides (4 new commands)
- `add-shape` - Create shapes with position/size and type validation
- `add-image` - Add images from URL
- `add-text` - Insert text into objects
- `replace-text` - Find/replace across presentation

### Docs (2 new commands)
- `delete` - Delete content range
- `add-table` - Insert table at position

### Infrastructure
- Added `getSheetID`, `parseRange`, `parseCellRef`, `columnLetterToIndex`, `getSlideID` helper functions
- Added `validShapeTypes` validation map

## v0.5.0

**P0 Commands: Sheets, Docs, Slides Management**

- `gws sheets add-sheet` - Add sheet with `--name`, `--rows`, `--cols`
- `gws sheets delete-sheet` - Delete sheet by `--name` or `--sheet-id`
- `gws sheets clear` - Clear cell values (keeps formatting)
- `gws docs insert` - Insert text at position
- `gws docs replace` - Find and replace text with `--match-case`
- `gws slides delete-slide` - Delete slide by ID or number
- `gws slides duplicate-slide` - Duplicate slide by ID or number

## v0.4.0

**Sheets Write & Drive Upload**

- `gws sheets create` - Create new spreadsheet with `--title`, `--sheet-names`
- `gws sheets write` - Write cell values with `--values` or `--values-json`
- `gws sheets append` - Append rows with `--values` or `--values-json`
- `gws drive upload` - Upload file with `--folder`, `--name`, `--mime-type` and auto-detection
- Added `spreadsheets` write scope (replaces redundant `spreadsheets.readonly`)

## v0.3.0

**Docs & Slides Write Capabilities**

- `gws docs create` - Create new document with `--title` and optional `--text`
- `gws docs append` - Append text to document with `--text` and `--newline`
- `gws slides create` - Create new presentation with `--title`
- `gws slides add-slide` - Add slide with `--title`, `--body`, `--layout` and layout validation
- Added `documents` and `presentations` write scopes
- Comprehensive mock API tests for Docs and Slides

## v0.2.0

**Drive Comments & Unit Tests**

- `gws drive comments` - List comments and replies on Drive files
  - `--include-resolved`, `--include-deleted`, `--max` flags
- Comprehensive unit tests for internal packages (auth, config, printer)
- Command structure tests for all services

## v0.1.0

**Initial Release**

Core read operations for Google Workspace services:

- **Auth**: `login`, `logout`, `status`
- **Gmail**: `list`, `read`, `send`
- **Calendar**: `list`, `events`, `create`
- **Tasks**: `lists`, `list`, `create`, `complete`
- **Drive**: `list`, `search`, `info`, `download`
- **Docs**: `read`, `info`
- **Sheets**: `info`, `list`, `read`
- **Slides**: `info`, `list`, `read`
- **Chat**: `list`, `messages`, `send`
- **Forms**: `info`, `responses`
- **Search**: web search
- **Version**: `version` command with build-time injection

### Infrastructure
- OAuth2 + PKCE authentication
- Lazy-initialized service client factory
- JSON (default) and text output formats
- Viper-based config (env + YAML)
- GitHub Actions CI workflow
- Makefile with build, test, vet, fmt targets
