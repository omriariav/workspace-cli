# Search Commands Reference

Complete flag and option reference for `gws search` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

## Prerequisites

Requires separate API credentials (not standard OAuth):

1. Create a Programmable Search Engine at https://programmablesearchengine.google.com/
2. Get an API key from Google Cloud Console (APIs & Services > Credentials)
3. Configure credentials via environment variables or config file

### Environment Variables

```bash
export GWS_SEARCH_ENGINE_ID="your-search-engine-id"
export GWS_SEARCH_API_KEY="your-api-key"
```

### Config File (`~/.config/gws/config.yaml`)

```yaml
search_engine_id: "your-search-engine-id"
search_api_key: "your-api-key"
```

---

## gws search

Searches using Google Programmable Search Engine.

```
Usage: gws search <query> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--max` | int | 10 | No | Maximum results (1-10) |
| `--site` | string | | No | Restrict to a specific site |
| `--type` | string | | No | Search type: `image` or empty for web |
| `--start` | int | 1 | No | Start index for pagination |
| `--api-key` | string | | No | API Key (overrides config) |
| `--engine-id` | string | | No | Search Engine ID (overrides config) |

### Pagination

Google Custom Search returns a maximum of 10 results per request. To get more results, use the `--start` flag:

- Page 1: `--start 1` (default)
- Page 2: `--start 11`
- Page 3: `--start 21`

### Output Fields (JSON)

Each result includes:
- `title` — Page title
- `link` — URL
- `snippet` — Text excerpt
- `displayLink` — Display URL

For image results (`--type image`):
- `image.contextLink` — Page containing the image
- `image.thumbnailLink` — Thumbnail URL
- `image.width` / `image.height` — Image dimensions
