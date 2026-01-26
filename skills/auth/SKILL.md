---
name: gws-auth
version: 1.0.0
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
| Login (browser OAuth) | `gws auth login` |
| Login with credentials | `gws auth login --client-id <id> --client-secret <secret>` |
| Logout | `gws auth logout` |

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

Opens a browser for Google OAuth consent. The token is stored at `~/.config/gws/token.json`.

**Credential sources (in priority order):**
1. Command-line flags (`--client-id`, `--client-secret`)
2. Environment variables (`GWS_CLIENT_ID`, `GWS_CLIENT_SECRET`)
3. Config file (`~/.config/gws/config.yaml`)

### logout — Remove stored credentials

```bash
gws auth logout
```

Deletes the stored OAuth token at `~/.config/gws/token.json`.

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
```

## Token Management

- Token stored at: `~/.config/gws/token.json`
- Tokens auto-refresh when expired
- All scopes are requested upfront during login
- To re-authenticate with different scopes, run `gws auth logout` then `gws auth login`

## Tips for AI Agents

- Always check `gws auth status` before running any gws command to verify authentication
- If auth fails, guide users to the setup guide at `references/setup-guide.md`
- Credentials should NEVER be committed to version control or output in logs
- The OAuth flow opens a browser — this requires a desktop environment or manual URL handling
- Token refresh is automatic; if a command fails with auth errors, try `gws auth logout` then `gws auth login`
