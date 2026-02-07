# Forms Commands Reference

Complete flag and option reference for `gws forms` commands.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json` or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws forms info

Gets metadata about a Google Form.

```
Usage: gws forms info <form-id>
```

No additional flags.

### Output Fields (JSON)

- `formId` — Form ID
- `info` — Form title and description
- `items` — List of form questions with their types and options

---

## gws forms responses

Gets all responses submitted to a form.

```
Usage: gws forms responses <form-id>
```

No additional flags.

### Output Fields (JSON)

Returns an array of responses, each containing:
- `responseId` — Response ID
- `createTime` — Submission time
- `answers` — Map of question IDs to submitted answers

### Notes

- This is read-only — forms cannot be created or modified via the CLI
- Form IDs can be extracted from Google Forms URLs: `docs.google.com/forms/d/<ID>/edit`
