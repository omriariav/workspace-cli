---
name: gws-auth
version: 1.43.1
description: "Google Workspace CLI authentication setup and management. Use when users need to set up OAuth credentials, authenticate with Google APIs, or troubleshoot authentication issues. Triggers: gws auth, google workspace setup, oauth, credentials, client id, api setup."
metadata:
  short-description: Google Workspace CLI authentication
  compatibility: claude-code, codex-cli
---

# Google Workspace Auth (gws auth)

`gws auth` manages OAuth2 authentication for all Google Workspace services.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

## Quick Command Reference

| Task | Command |
|------|---------|
| Check auth status | `gws auth status` |
| Login (all scopes) | `gws auth login` |
| Login (specific services) | `gws auth login --services gmail,calendar,chat` |
| Login read-only | `gws auth login --readonly` |
| Login read-only for some services | `gws auth login --services gmail,drive --readonly` |
| Add a service to the existing login | `gws auth login --add --services tasks` |
| Use an existing access token | `GWS_ACCESS_TOKEN=<token> gws gmail list` |
| Scopes a command needs | `gws gmail send --show-scopes` |
| Describe a command's flags as JSON | `gws gmail send --schema` |
| Login with credentials | `gws auth login --client-id <id> --client-secret <secret>` |
| Logout (revokes token) | `gws auth logout` |

## Service-Account Modes (v1.43.0+)

Besides the default interactive OAuth flow, `gws` supports two explicit
service-account modes via `auth_mode` (`GWS_AUTH_MODE` or config file).
Modes are mutually exclusive and validated fail-closed with actionable
errors; key material is never persisted, printed, or echoed.

| Mode | Required settings | Behavior |
|------|-------------------|----------|
| `oauth` (default) | none | Interactive OAuth2 + PKCE via `gws auth login` |
| `service-account-dwd` | `service_account_key_file` (`GWS_SERVICE_ACCOUNT_KEY_FILE`) + `impersonate_user` (`GWS_IMPERSONATE_USER`) | Domain-wide delegation: the service account acts as the named Workspace user across all services |
| `service-account-app` | `service_account_key_file` | Admin-approved Google Chat app credentials; limited to `gws events` subscriptions on individual Chat spaces — every other service command is rejected |

```bash
# Domain-wide delegation (requires DWD enabled for the service account
# and the scopes authorized in the Admin console)
export GWS_AUTH_MODE=service-account-dwd
export GWS_SERVICE_ACCOUNT_KEY_FILE=~/keys/sa.json
export GWS_IMPERSONATE_USER=user@yourdomain.com
gws gmail list   # runs as the impersonated user, no login needed

# Chat app auth (events subscriptions only)
export GWS_AUTH_MODE=service-account-app
export GWS_SERVICE_ACCOUNT_KEY_FILE=~/keys/chat-app-sa.json
gws events subscriptions create --target //chat.googleapis.com/spaces/AAA ...
```

Notes:
- `gws auth login` fails fast in service-account modes (no login needed);
  `gws auth status` reports the configured mode instead of token state.
- Incomplete or mixed configuration is rejected: DWD without
  `impersonate_user`, app mode with `impersonate_user`, or OAuth mode with
  leftover service-account settings all produce clear errors.

## Least-Privilege Login

Start read-only and add write access only for the services that need it.

```bash
gws auth login --readonly                       # read-only, every service
gws auth login --services gmail,drive --readonly  # read-only, two services
gws auth login --add --services gmail           # later: upgrade gmail to full access
```

- `--readonly` requests only read-only scopes. Commands that create, change,
  or delete data then fail with an insufficient-scope error (exit code 3) and
  a hint naming the login that grants write access.
- `--add` merges the selected services into the existing authorization
  instead of replacing it. Without `--add`, a login replaces the previous
  grant.
- Every login must return its own refresh token; if Google returns none,
  the login is rejected and the previous authorization is left unchanged.
- The reported access levels come from the scopes Google actually granted,
  so unticking a scope on the consent screen is reflected in `auth status`.
- Adding a service read-only never downgrades a service that already has
  full access. To reduce access, run `gws auth logout` (which revokes the
  old grant) and then log in again with the narrower options.
- `gws auth status` reports the access level per service (`full` or
  `readonly`) and `read_only: true` when nothing has write access.

## External Access Token

If another system already manages Google OAuth tokens (CI, a secrets broker,
`gcloud auth print-access-token`, a domain-wide-delegation token service),
hand the token to `gws` directly:

```bash
export GWS_ACCESS_TOKEN="$(my-token-broker)"
gws calendar events --days 1
```

- The token takes precedence over the stored login and over the
  service-account modes. No login is needed.
- `gws` uses it as-is: it is never refreshed, stored, or printed. An expired
  or under-scoped token surfaces the API error (exit code 3).
- Check what a command needs first with `--show-scopes`.
- `--access-token <token>` also works, but a flag value is visible in process
  listings and shell history — prefer the environment variable.

## Inspecting Commands Without Running Them

Two global flags print JSON and exit without calling any Google API and
without needing credentials:

```bash
gws drive share --schema        # flags (name, type, default, required), usage, subcommands
gws drive share --show-scopes   # full and read-only scopes of the command's service
```

`--schema` also marks commands covered by the confirmation gate with
`"sensitive": true`. Scope output is per service, not per command.

## Confirmation Gate for Sensitive Commands

Off by default. When turned on, commands that send messages, change sharing,
or delete data ask for approval before they run.

```bash
export GWS_CONFIRM_SENSITIVE=1     # or confirm_sensitive: true in config.yaml
gws gmail send --to a@example.com --subject Hi --body Hello
# stderr: gws gmail send sends an email. Proceed? [y/N]:
```

- In a terminal, the prompt is written to stderr and reads `y`/`n` from stdin.
- Without a terminal (agents, scripts), the command fails with
  `confirmation required` instead of hanging. **Ask the user for explicit
  approval, then re-run the same command with `--yes`.** Never add `--yes`
  on your own initiative.
- Covered by name: every command that starts with `delete`, `trash`,
  `remove`, `clear`, `empty` or `batch-delete`, in any service (including
  content deletes such as `docs delete-table-row` or `sheets clear`).
- Covered when a flag is used: `drive update --trashed`, `chat setup-space
  --members`, `calendar create --attendees`, `calendar update
  --add-attendees`, `calendar rsvp --message`.
- Covered explicitly: commands that send (Gmail `send`, `reply`, `forward`,
  `send-draft`; Chat `send`) and commands that change access (Calendar
  `share`, `unshare`, `update-acl`; Drive `share`, `unshare`,
  `update-permission`; Chat `add-member`, `update-member`).
- Not covered: creating or editing content, `untrash`, and Drive approval
  actions. `gws <command> --schema` shows `"sensitive": true` for a covered
  command.

## First-Time Setup

If you haven't set up Google Cloud credentials yet, see the detailed setup guide:
**[Setup Guide](references/setup-guide.md)**

Quick summary:
1. Create a Google Cloud project
2. Enable the required Workspace APIs
3. Create OAuth 2.0 credentials (Desktop app type)
4. Set credentials via environment variables or config file
5. Run `gws auth login`

## Detailed Usage

### status — Check authentication status

```bash
gws auth status
```

Shows whether you're authenticated, the current user email, and token expiry info.

### login — Authenticate with Google

```bash
gws auth login [flags]
```

**Flags:**
- `--client-id string` — OAuth client ID (overrides env/config)
- `--client-secret string` — OAuth client secret (overrides env/config)
- `--services string` — Comma-separated services to authorize (e.g. `gmail,calendar,chat`). Omit for all scopes.
- `--readonly` — Request only read-only scopes for the selected services
- `--add` — Add the selected services to the existing authorization instead of replacing it

Opens a browser for Google OAuth consent. The token is stored at `~/.config/gws/token.json`.

**Available services:** gmail, calendar, drive, docs, sheets, slides, tasks, chat, forms, contacts

**Credential sources (in priority order):**
1. Command-line flags (`--client-id`, `--client-secret`)
2. Environment variables (`GWS_CLIENT_ID`, `GWS_CLIENT_SECRET`)
3. Config file (`~/.config/gws/config.yaml`)

### logout — Remove stored credentials

```bash
gws auth logout
```

Revokes the token server-side with Google, then deletes the local token at `~/.config/gws/token.json`.

## Configuration

### Environment Variables

```bash
export GWS_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GWS_CLIENT_SECRET="your-client-secret"
```

### Config File (`~/.config/gws/config.yaml`)

```yaml
client_id: "your-client-id.apps.googleusercontent.com"
client_secret: "your-client-secret"
services:
  - gmail
  - calendar
  - chat
```

## Token Management

- Token stored at: `~/.config/gws/token.json` (atomic writes, file-locked)
- Granted services tracked in: `~/.config/gws/granted_services.json`; per-service access level (`full` / `readonly`) in `~/.config/gws/granted_access.json`
- Tokens auto-refresh when expired; refresh tokens preserved across re-auth
- Scoped login: use `--services` to request only needed scopes (smaller consent screen)
- Default services can be set in config.yaml: `services: [gmail, calendar, chat]`
- To add more scopes, run `gws auth login --add --services drive` (or re-run the full list without `--add`)
- Logout revokes the token server-side before deleting locally

## Tips for AI Agents

- Always check `gws auth status` before running any gws command to verify authentication
- Prefer `gws auth login --readonly` for tasks that only read data; ask the user before requesting write access
- If a command fails with `confirmation required`, ask the user to approve the action, then re-run it with `--yes`
- Use `--schema` and `--show-scopes` to learn a command's flags and scopes instead of guessing
- If auth fails, guide users to the setup guide at `references/setup-guide.md`
- Credentials should NEVER be committed to version control or output in logs
- The OAuth flow opens a browser — this requires a desktop environment or manual URL handling
- Token refresh is automatic; if a command fails with auth errors, try `gws auth logout` then `gws auth login`
