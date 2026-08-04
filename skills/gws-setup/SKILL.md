---
name: gws-setup
version: 1.43.1
description: "Safely inspect, preview, and explicitly configure optional GWS authentication and session-context hooks. Use for gws setup, gws hooks, time context, Workspace context, or hook configuration."
metadata:
  short-description: Safe GWS hook setup
  compatibility: claude-code, codex-cli
---

# GWS Setup

Use this workflow to inspect and configure `gws` and its optional context hooks. Installation is inert: no Workspace API calls or context injection occur until the user explicitly approves an `--apply` command.

> `gws` is not the official Google CLI and is not endorsed by or affiliated with Google.

## Safety contract

- Begin with inspection and a preview. Do not authenticate, open a browser, or write configuration until the user approves the exact planned action.
- Never print or copy OAuth tokens, client secrets, cached Workspace content, or existing credential files.
- Time-only mode needs no Google authentication. Ask about authentication only if the user elects Workspace context.
- Workspace hooks are read-only and event-driven. Never send, modify, delete, or mark data as read.
- Do not edit, disable, or replace personal hooks. If status detects a duplicate time injector, explain it and offer a separately approved migration plan.

## Inspect first

Run only read-only checks:

```bash
command -v gws
gws version
gws hooks status
```

If the user is considering Workspace context, also run `gws auth status`. Report paths, enabled providers, timezone, auth readiness, source gaps, and duplicate time injectors without including credentials or cached context.

## Output formats

Use `--format json` for structured automation, `--format yaml` for readable structured output, or `--format text` for a concise terminal view. Hook status and health output never includes cached Workspace content in any format.

## Scope and providers

Configuration precedence is: built-in defaults, user config `~/.config/gws/hooks.json`, workspace config `.gws/hooks.json`, then `GWS_HOOKS_ENABLED` and `GWS_HOOKS_TIMEZONE` environment overrides. The user config exclusively owns the global hook gate and both provider enablement settings. Workspace config contains only allowlisted, consent-free preferences such as timezone, Workspace refresh TTL, and source settings; it cannot activate a provider, private reads, credentials, tokens, or cached content.

Ask the user to choose one or both providers:

- `time`: computes current local time for every supported event; no Google auth required.
- `workspace`: reads only configured Calendar, Gmail, sent-mail, Tasks, and explicit Chat sources, refreshing only on an event after its TTL expires.

## Preview, then apply

Preview every mutation without `--apply`:

```bash
gws hooks enable --time --scope user
gws hooks enable --workspace --scope user
gws hooks enable --time --workspace --scope user
gws hooks configure --scope user --enabled=true --time=true --workspace=false
gws hooks configure --scope workspace --calendar=false --gmail=false --sent-mail=false --tasks=true --chat=false
gws hooks configure --scope workspace --timezone America/New_York --refresh-minutes 5 --calendar-lookahead-days 7 --gmail-limit 8 --sent-lookback-days 7 --sent-limit 5 --task-discovery-limit 3 --task-limit 10
gws hooks configure --scope workspace --task-list TASK_LIST_ID --chat-space spaces/SPACE_ID --chat-limit 5
```

Review the exact path and requested keys. Existing configuration values are deliberately not echoed; use `gws hooks status` to inspect the safe effective provider state and auth requirement.

Only after the user explicitly approves the exact preview, repeat that command with `--apply`:

```bash
gws hooks enable --time --workspace --scope user --apply
gws hooks configure --scope workspace --timezone America/New_York --refresh-minutes 5 --apply
```

Writes are atomic, merge-only, and mode-restricted. They never write secret material. Do not use workspace scope for `hooks enable`, `hooks disable`, or `hooks configure --enabled/--time/--workspace`: those consent controls are user-scope only.

## Host hook approval

Applying GWS configuration makes the providers eligible, but it does not by itself activate a lifecycle hook in the host. The host must also review and approve the packaged hook entrypoints.

- In Codex, open `/hooks`, inspect the `gws` `SessionStart` and `UserPromptSubmit` entries, and explicitly approve them when the user approves that separate host action.
- Do not claim hooks are active until that host review/approval is complete. Without it, the GWS configuration remains inert.

## Hook command reference

See the focused [hook command reference](references/commands.md) for the user-facing command surface and safety constraints.

## Verify, reconfigure, and disable

Run a safe health check after an approved change:

```bash
gws hooks status
gws hooks test --event UserPromptSubmit
```

The test reports provider state, resolved timezone, auth/source diagnostics, source availability, and refresh metadata without printing cached Workspace or message content. It does not mutate Workspace state. For reconfiguration, preview `gws hooks configure` and get new approval before `--apply`. To stop all hook fetches immediately while retaining normal `gws` commands and authentication:

```bash
gws hooks disable --scope user
gws hooks disable --scope user --apply
```

## Tips for AI Agents

- Treat an inspect or preview as read-only evidence, never as approval to apply it.
- Time-only mode must not prompt for OAuth credentials or invoke Workspace commands.
- When a duplicate personal time injector is detected, report it without changing the personal hook.
