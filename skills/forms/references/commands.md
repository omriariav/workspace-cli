# Forms Commands Reference

Complete flag and option reference for `gws forms` commands — 6 commands total.

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

- `id` — Form ID
- `title` — Form title
- `document_title` — Document title
- `description` — Form description (if set)
- `responder_uri` — URL for respondents
- `items` — List of form questions with their types and options
- `item_count` — Number of items in the form

---

## gws forms get

Gets metadata about a Google Form. Alias for `info`.

```
Usage: gws forms get <form-id>
```

No additional flags.

### Output Fields (JSON)

Same as `info`.

---

## gws forms responses

Gets all responses submitted to a form.

```
Usage: gws forms responses <form-id>
```

No additional flags.

### Output Fields (JSON)

Returns an object with:
- `form_id` — Form ID
- `form_title` — Form title
- `responses` — Array of response objects
- `response_count` — Number of responses

Each response includes:
- `id` — Response ID
- `create_time` — Submission time
- `last_submit` — Last submission time
- `email` — Respondent email (if collected)
- `answers` — Map of question titles to submitted answers

---

## gws forms response

Gets a specific response by ID from a form.

```
Usage: gws forms response <form-id> --response-id <id>
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--response-id` | string | | Yes | Response ID to retrieve |

### Output Fields (JSON)

- `id` — Response ID
- `form_id` — Form ID
- `form_title` — Form title
- `create_time` — Submission time
- `last_submit` — Last submission time
- `email` — Respondent email (if collected)
- `answers` — Map of question titles to submitted answers

---

## gws forms create

Creates a new blank Google Form with a title and optional description.

```
Usage: gws forms create [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | Yes | Form title |
| `--description` | string | | No | Form description |

### Output Fields (JSON)

- `id` — New form ID
- `title` — Form title
- `document_title` — Document title
- `description` — Form description (if set)
- `responder_uri` — URL for respondents

### Examples

```bash
# Create a simple form
gws forms create --title "Feedback Survey"

# Create a form with description
gws forms create --title "Team Poll" --description "Weekly team feedback"
```

---

## gws forms update

Updates a Google Form using title/description flags or a JSON batchUpdate file.

```
Usage: gws forms update <form-id> [flags]
```

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--title` | string | | No | New form title |
| `--description` | string | | No | New form description |
| `--file` | string | | No | Path to JSON file with batchUpdate request body |

At least one of `--title`, `--description`, or `--file` must be provided.

### Output Fields (JSON)

- `id` — Form ID
- `title` — Form title
- `document_title` — Document title
- `description` — Form description (if set)
- `responder_uri` — URL for respondents
- `status` — Always `"updated"`
- `replies_count` — Number of batch update replies

### Examples

```bash
# Update title
gws forms update <form-id> --title "New Title"

# Update description
gws forms update <form-id> --description "Updated description"

# Advanced update with JSON file (add questions, etc.)
gws forms update <form-id> --file batch-update.json
```

### Notes

- For advanced updates (adding questions, modifying settings), use `--file` with a JSON batchUpdate body
- See https://developers.google.com/forms/api/reference/rest/v1/forms/batchUpdate for the request format
- Form IDs can be extracted from Google Forms URLs: `docs.google.com/forms/d/<ID>/edit`
