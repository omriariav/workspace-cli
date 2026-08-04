---
name: gws-auth
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
- Granted services tracked in: `~/.config/gws/granted_services.json`
- Tokens auto-refresh when expired; refresh tokens preserved across re-auth
- Scoped login: use `--services` to request only needed scopes (smaller consent screen)
- Default services can be set in config.yaml: `services: [gmail, calendar, chat]`
- To add more scopes, re-run `gws auth login --services gmail,calendar,chat,drive`
- Logout revokes the token server-side before deleting locally

## Tips for AI Agents

- Always check `gws auth status` before running any gws command to verify authentication
- If auth fails, guide users to the setup guide at `references/setup-guide.md`
- Credentials should NEVER be committed to version control or output in logs
- The OAuth flow opens a browser — this requires a desktop environment or manual URL handling
- Token refresh is automatic; if a command fails with auth errors, try `gws auth logout` then `gws auth login`
