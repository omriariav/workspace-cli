# gws - Google Workspace CLI

A unified command-line interface for Google Workspace services, built in Go. Designed for AI agents and power users who need structured, token-efficient access to Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, Chat, Forms, and Custom Search.

## Features

- **10+ Google Services** - Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, Chat, Forms, Search
- **JSON & Text Output** - Machine-readable JSON (default) or human-readable text
- **OAuth2 + PKCE** - Secure authentication with automatic token refresh
- **Single Auth Flow** - Authenticate once to access all services

## Installation

### From Source

```bash
git clone https://github.com/omriariav/workspace-cli.git
cd workspace-cli/gws
go build -o gws .
```

### Prerequisites

1. Go 1.23+
2. A Google Cloud Project with OAuth 2.0 credentials
3. Enable required APIs in [Google Cloud Console](https://console.cloud.google.com/apis/library):
   - Gmail API
   - Google Calendar API
   - Google Drive API
   - Google Docs API
   - Google Sheets API
   - Google Slides API
   - Tasks API
   - (Optional) Google Chat API
   - (Optional) Google Forms API

## Configuration

### Environment Variables

```bash
export GWS_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GWS_CLIENT_SECRET="your-client-secret"
```

### Config File

Create `~/.config/gws/config.yaml`:

```yaml
client_id: "your-client-id.apps.googleusercontent.com"
client_secret: "your-client-secret"
format: json  # or "text"
```

## Authentication

```bash
# Login (opens browser for OAuth consent)
gws auth login

# Check status
gws auth status

# Logout
gws auth logout
```

## Usage

All commands support `--format=json` (default) or `--format=text`.

### Gmail

```bash
# List recent threads
gws gmail list --max=10

# Search emails
gws gmail list --query="is:unread from:someone@example.com"

# Read a message
gws gmail read <message-id>

# Send an email
gws gmail send --to="recipient@example.com" --subject="Hello" --body="Message body"
```

### Calendar

```bash
# List calendars
gws calendar list

# List upcoming events (next 7 days)
gws calendar events --days=7

# Create an event
gws calendar create --title="Meeting" --start="2024-01-15 14:00" --end="2024-01-15 15:00"
```

### Tasks

```bash
# List task lists
gws tasks lists

# List tasks in a list
gws tasks list <tasklist-id>

# Create a task
gws tasks create --title="New task" --tasklist="@default"

# Complete a task
gws tasks complete <tasklist-id> <task-id>
```

### Drive

```bash
# List files in root
gws drive list --max=20

# Search for files
gws drive search "project report"

# Get file info
gws drive info <file-id>

# Download a file
gws drive download <file-id> --output="filename.pdf"
```

### Docs

```bash
# Read document text
gws docs read <document-id>

# Get document info
gws docs info <document-id>
```

### Sheets

```bash
# Get spreadsheet info
gws sheets info <spreadsheet-id>

# List sheets in a spreadsheet
gws sheets list <spreadsheet-id>

# Read a range (returns JSON with headers)
gws sheets read <spreadsheet-id> "Sheet1!A1:D10"

# Read as CSV
gws sheets read <spreadsheet-id> "Sheet1!A1:D10" --output-format=csv
```

### Slides

```bash
# Get presentation info
gws slides info <presentation-id>

# List all slides with content
gws slides list <presentation-id>

# Read specific slide (1-indexed)
gws slides read <presentation-id> 1
```

### Chat

> Note: Requires additional setup - you need to configure a Chat App in Google Cloud Console.

```bash
# List spaces
gws chat list

# List messages in a space
gws chat messages <space-id>

# Send a message
gws chat send --space="spaces/XXXXX" --text="Hello!"
```

### Forms

> Note: Requires enabling the Google Forms API.

```bash
# Get form info
gws forms info <form-id>

# Get form responses
gws forms responses <form-id>
```

### Custom Search

> Note: Requires a Programmable Search Engine ID and API key.

```bash
export GWS_SEARCH_ENGINE_ID="your-cx-id"
export GWS_SEARCH_API_KEY="your-api-key"

gws search "golang tutorial" --max=5
```

## Output Formats

### JSON (default)

```bash
gws gmail list --max=2
```

```json
{
  "count": 2,
  "threads": [
    {"id": "abc123", "subject": "Hello", "from": "sender@example.com"},
    {"id": "def456", "subject": "Meeting", "from": "boss@example.com"}
  ]
}
```

### Text

```bash
gws gmail list --max=2 --format=text
```

```
id        subject    from
-----     -------    ----
abc123    Hello      sender@example.com
def456    Meeting    boss@example.com
```

## Token Storage

Tokens are stored at `~/.config/gws/token.json` with `0600` permissions (owner read/write only).

## License

MIT
