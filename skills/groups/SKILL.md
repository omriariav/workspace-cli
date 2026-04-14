---
name: gws-groups
version: 1.0.0
description: "Google Groups CLI operations via gws. Use when users need to list groups or view group members. Triggers: groups, google groups, admin directory, group members."
metadata:
  short-description: Google Groups CLI operations
  compatibility: claude-code, codex-cli
---

# Google Groups (gws groups)

`gws groups` provides CLI access to Google Groups via the Admin Directory API with structured JSON output.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

**Minimum version required:** v1.18.0 (Groups support added in this release)

## Prerequisites

- **Admin SDK API** must be enabled in the Google Cloud project
- **Google Workspace admin privileges** are required (not available for personal Gmail accounts)

## Authentication

Requires OAuth2 credentials. Run `gws auth status` to check.
If not authenticated: `gws auth login` (opens browser for OAuth consent).
For initial setup, see the `gws-auth` skill.

## Quick Command Reference

| Task | Command |
|------|---------|
| List groups | `gws groups list` |
| List groups in a domain | `gws groups list --domain example.com` |
| List groups for a user | `gws groups list --user-email user@example.com` |
| List more groups | `gws groups list --max 200` |
| List group members | `gws groups members group@example.com` |
| List group owners | `gws groups members group@example.com --role OWNER` |

## Detailed Usage

### list -- List groups

```bash
gws groups list [flags]
```

Lists Google Groups in your domain. By default, lists all groups for the customer account.

**Flags:**
- `--max int` -- Maximum number of groups to return (default 50)
- `--domain string` -- Filter by domain
- `--user-email string` -- Filter groups for a specific user

**Note:** `--domain` and `--user-email` are mutually exclusive. Providing both will return an error.

**Output includes:**
- `id` -- Group ID
- `email` -- Group email address
- `name` -- Group display name
- `description` -- Group description (if set)
- `member_count` -- Number of direct members (if available)

**Examples:**
```bash
gws groups list
gws groups list --max 200
gws groups list --domain example.com
gws groups list --user-email alice@example.com
gws groups list --format json | jq '.groups[] | {name, email}'
```

### members -- List group members

```bash
gws groups members <group-email> [flags]
```

Lists members of a Google Group by group email address.

**Arguments:**
- `group-email` -- Group email address (required)

**Flags:**
- `--max int` -- Maximum number of members to return (default 50)
- `--role string` -- Filter by role: `OWNER`, `MANAGER`, or `MEMBER`

**Output includes:**
- `id` -- Member ID
- `email` -- Member email address
- `role` -- Member role (OWNER, MANAGER, or MEMBER)
- `type` -- Member type (e.g., USER, GROUP)
- `status` -- Member status (if available)

**Examples:**
```bash
gws groups members engineering@example.com
gws groups members engineering@example.com --max 200
gws groups members engineering@example.com --role OWNER
gws groups members engineering@example.com --role MANAGER
gws groups members engineering@example.com --format json | jq '.members[] | select(.role == "OWNER")'
```

## Output Modes

```bash
gws groups list --format json    # Structured JSON (default)
gws groups list --format yaml    # YAML format
gws groups list --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- The `list` command returns groups for the entire customer account by default; use `--domain` or `--user-email` to narrow results
- `--domain` and `--user-email` cannot be used together -- pick one filter
- Use `--role` on the `members` command to quickly find group owners or managers
- The `members` command requires the exact group email address as a positional argument
- Use `--quiet` on any command to suppress JSON output (useful for scripted actions)
- This command requires Admin SDK API enabled and Workspace admin privileges -- it will not work with personal Gmail accounts
- For bulk membership analysis, pipe output through `jq` to filter and aggregate results
