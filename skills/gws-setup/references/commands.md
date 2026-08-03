# GWS Hook Commands

These commands configure optional, disabled-by-default context hooks. All configuration commands preview by default; add `--apply` only after the user approves the exact planned change.

> `gws` is not the official Google CLI and is not endorsed by or affiliated with Google.

| Goal | Command | Safety |
| --- | --- | --- |
| Inspect effective state | `gws hooks status` | Metadata only; never prints tokens or cached context. |
| Preview provider enablement | `gws hooks enable --time --workspace --scope user` | User scope only; no file change without `--apply`. |
| Preview source or TTL changes | `gws hooks configure --scope workspace --calendar=false --task-limit 10` | Workspace scope is consent-free and cannot enable hooks or providers; all numeric limits must be positive. |
| Safe health check | `gws hooks test --event UserPromptSubmit` | Reports state, gaps, and refresh metadata only. |
| Stop hook fetching | `gws hooks disable --scope user` | Normal GWS commands and authentication are unchanged. |

`gws hooks run` is the packaged host adapter entrypoint. It is not a setup command and must not be used to bypass host approval. In Codex, separately use `/hooks` to review and explicitly approve the `gws` lifecycle entries before treating hooks as active.

The user config owns `enabled` and `providers.*.enabled`. A repository `.gws/hooks.json` may set only consent-free preferences (timezone, Workspace refresh TTL, and source settings) and can never activate private reads.

## Global flags

| Flag | Description |
| --- | --- |
| `--config <path>` | Override the main GWS configuration file. |
| `--format <format>` | Output format: `json`, `yaml`, or `text`. |
| `--quiet` | Suppress normal output for scripted use. |
