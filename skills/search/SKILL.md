---
name: gws-search
version: 1.0.0
description: "Google Custom Search CLI operations via gws. Use when users need to perform web searches using Google Programmable Search Engine. Triggers: google search, custom search, web search."
metadata:
  short-description: Google Custom Search CLI operations
  compatibility: claude-code, codex-cli
---

# Google Search (gws search)

`gws search` provides CLI access to Google Programmable Search Engine (Custom Search) with structured JSON output.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

## Authentication

Requires separate API credentials beyond standard OAuth:
1. Create a Programmable Search Engine at https://programmablesearchengine.google.com/
2. Get an API key from Google Cloud Console
3. Set `GWS_SEARCH_ENGINE_ID` and `GWS_SEARCH_API_KEY` environment variables, or add `search_engine_id` and `search_api_key` to your config file

## Quick Command Reference

| Task | Command |
|------|---------|
| Web search | `gws search "query"` |
| Limit results | `gws search "query" --max 5` |
| Search a site | `gws search "query" --site example.com` |
| Image search | `gws search "query" --type image` |
| Paginate results | `gws search "query" --start 11` |
| Override API key | `gws search "query" --api-key <key> --engine-id <id>` |

## Detailed Usage

### search — Web search

```bash
gws search <query> [flags]
```

**Flags:**
- `--max int` — Maximum number of results, 1-10 (default 10)
- `--site string` — Restrict search to a specific site
- `--type string` — Search type: `image` or empty for web
- `--start int` — Start index for results pagination (default 1)
- `--api-key string` — API Key (overrides config)
- `--engine-id string` — Search Engine ID (overrides config)

**Examples:**
```bash
gws search "golang best practices"
gws search "kubernetes deployment" --max 5
gws search "release notes" --site github.com
gws search "logo" --type image
gws search "query" --start 11    # Page 2 of results
```

## Output Modes

```bash
gws search "query" --format json    # Structured JSON (default)
gws search "query" --format yaml    # YAML format
gws search "query" --format text    # Human-readable text
```

## Tips for AI Agents

- Always use `--format json` (the default) for programmatic parsing
- This command uses Google's Custom Search API, not standard Google search — it requires a separate API key and search engine ID
- Max 10 results per request; use `--start` to paginate
- The `--site` flag is useful for searching within a specific domain
- Image search (`--type image`) returns image URLs and metadata
