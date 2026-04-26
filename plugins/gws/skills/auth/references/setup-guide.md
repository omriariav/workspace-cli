# Google Workspace CLI — Authentication Setup Guide

Step-by-step guide to set up Google Cloud credentials for `gws`.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Overview

`gws` uses OAuth 2.0 to authenticate with Google Workspace APIs. You need to:

1. Create a Google Cloud project
2. Enable the required APIs
3. Create OAuth 2.0 credentials
4. Configure `gws` with your credentials
5. Run the initial login flow

## Step 1: Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top of the page
3. Click **New Project**
4. Enter a project name (e.g., "GWS CLI")
5. Click **Create**
6. Select the new project from the project dropdown

## Step 2: Enable Google Workspace APIs

Navigate to **APIs & Services > Library** and enable each API you plan to use:

| API | Required For |
|-----|-------------|
| Gmail API | `gws gmail` commands |
| Google Calendar API | `gws calendar` commands |
| Google Drive API | `gws drive` commands |
| Google Docs API | `gws docs` commands |
| Google Sheets API | `gws sheets` commands |
| Google Slides API | `gws slides` commands |
| Google Tasks API | `gws tasks` commands |
| Google Chat API | `gws chat` commands |
| Google Forms API | `gws forms` commands |
| Custom Search API | `gws search` commands |

For each API:
1. Search for the API name in the Library
2. Click on it
3. Click **Enable**

**Tip:** Enable all APIs you might use now to avoid repeated re-authentication later.

## Step 3: Configure OAuth Consent Screen

1. Go to **APIs & Services > OAuth consent screen**
2. Select **External** user type (or **Internal** if using Google Workspace organization)
3. Click **Create**
4. Fill in the required fields:
   - **App name**: e.g., "GWS CLI"
   - **User support email**: your email
   - **Developer contact email**: your email
5. Click **Save and Continue**
6. On the **Scopes** page, click **Add or Remove Scopes**
7. Add the scopes for the APIs you enabled (or skip — `gws` requests scopes at runtime)
8. Click **Save and Continue**
9. On the **Test users** page, add your Google account email
10. Click **Save and Continue**

**Important:** While the app is in "Testing" status, only test users you add can authenticate. This is fine for personal use.

## Step 4: Create OAuth 2.0 Credentials

1. Go to **APIs & Services > Credentials**
2. Click **+ Create Credentials > OAuth client ID**
3. Select **Desktop app** as the application type
4. Enter a name (e.g., "GWS CLI Desktop")
5. Click **Create**
6. **Copy the Client ID and Client Secret** — you'll need these next

## Step 5: Configure gws with Credentials

Choose one of these methods:

### Option A: Environment Variables (Recommended)

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
export GWS_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GWS_CLIENT_SECRET="your-client-secret"
```

Then reload: `source ~/.zshrc` (or restart your terminal).

### Option B: Config File

Create or edit `~/.config/gws/config.yaml`:

```yaml
client_id: "your-client-id.apps.googleusercontent.com"
client_secret: "your-client-secret"
```

### Option C: Command-Line Flags

Pass credentials directly (not recommended for regular use):

```bash
gws auth login --client-id "your-id" --client-secret "your-secret"
```

## Step 6: Authenticate

Run the login command:

```bash
gws auth login
```

This will:
1. Open your default browser
2. Show Google's OAuth consent screen
3. Ask you to grant the requested permissions
4. Redirect back to confirm authentication

The token is saved to `~/.config/gws/token.json` and will auto-refresh.

## Step 7: Verify

Check your authentication status:

```bash
gws auth status
```

Try a command:

```bash
gws gmail list --max 3
```

## Token Management

| File | Purpose |
|------|---------|
| `~/.config/gws/config.yaml` | Client credentials and settings |
| `~/.config/gws/token.json` | OAuth token (auto-refreshes) |

- Tokens auto-refresh when expired
- Scopes are requested based on the `--services` flag, config defaults, or all scopes by default
- To re-authenticate: `gws auth logout` then `gws auth login`
- To switch accounts: logout and login with a different Google account

## Additional Setup: Google Search

`gws search` requires separate credentials:

1. Create a [Programmable Search Engine](https://programmablesearchengine.google.com/)
2. Get an API key from **APIs & Services > Credentials > + Create Credentials > API key**
3. Configure:

```bash
export GWS_SEARCH_ENGINE_ID="your-search-engine-id"
export GWS_SEARCH_API_KEY="your-api-key"
```

Or in `~/.config/gws/config.yaml`:

```yaml
search_engine_id: "your-search-engine-id"
search_api_key: "your-api-key"
```

## Additional Setup: Google Chat

Google Chat API may require additional configuration:

1. Enable the Chat API in your Google Cloud project
2. In some environments, Chat requires a service account with domain-wide delegation
3. Contact your Google Workspace admin if you're in an organization

## Troubleshooting

### "Error: oauth2: cannot fetch token"
- Verify your Client ID and Client Secret are correct
- Check that the credentials are for a "Desktop app" type
- Ensure the API is enabled in your Google Cloud project

### "Error: access_denied"
- Make sure your email is added as a test user in the OAuth consent screen
- If using External user type, the app must be in "Testing" mode with your email listed

### "Error: invalid_scope"
- The requested API may not be enabled in your Google Cloud project
- Go to APIs & Services > Library and enable the missing API

### Token expired and won't refresh
- Delete the token file: `rm ~/.config/gws/token.json`
- Re-authenticate: `gws auth login`

### "Error: redirect_uri_mismatch"
- Ensure the credential type is "Desktop app" (not "Web application")
- Desktop apps use `http://localhost` for the redirect URI automatically
