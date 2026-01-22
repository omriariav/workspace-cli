# **Phase 2: Core Communication (Gmail & Chat)**

## **1. Manual Setup (Required)**

Before generating code, you must enable the APIs in your Google Cloud Project. The CLI will fail without these.

1. Go to the [Google Cloud Console](https://console.cloud.google.com/apis/library).
2. Select the project you created for your OAuth Client ID.
3. **Search for and Enable** the following APIs:
   * **Gmail API**
   * **Google Chat API**
4. (Optional but recommended) Go to **APIs & Services > OAuth consent screen** and ensure your user is added to "Test Users" if the app is still in Testing mode.

## **2. Prompt for Claude Code**

Copy and paste the block below into your terminal to trigger the implementation.

**Note:** We are adding scopes for Gmail and Chat. You will likely need to re-login (gws auth login) after this update to grant the new permissions.

Read PLAN.md and implementing Phase 2 (Gmail & Chat).

Please perform the following changes:

1.  **Update Auth Scopes** (`internal/auth/auth.go`):
    - Add the following scopes to your config:
      - `gmail.GmailReadonlyScope`
      - `gmail.GmailSendScope`
      - `chat.ChatSpacesReadonlyScope`
      - `chat.ChatMessagesScope`

2.  **Implement Gmail Client** (`internal/client/gmail.go`):
    - Create a wrapper struct `GmailClient`.
    - Implement `ListThreads(maxResults int64, query string)`.
    - Implement `GetThread(id string)`.
    - Implement `SendEmail(to, subject, body string)`.

3.  **Implement Gmail Commands** (`cmd/gmail.go`):
    - `gws gmail list`: Flags: `--max`, `--query`. Output: Table of ID, Snippet, Date.
    - `gws gmail read <id>`: Output: The full email body text.
    - `gws gmail send`: Flags: `--to`, `--subject`, `--body`.

4.  **Implement Chat Client** (`internal/client/chat.go`):
    - Create a wrapper struct `ChatClient`.
    - Implement `ListSpaces()`.
    - Implement `SendMessage(spaceName string, text string)`.

5.  **Implement Chat Commands** (`cmd/chat.go`):
    - `gws chat list`: Output: Table of Space Name, Display Name, Type.
    - `gws chat send <space-name> <message>`: Sends a text message.

Refactor `cmd/root.go` or `main.go` if necessary to register these new commands. Ensure all commands support the global `--format=json` flag by returning structs in `RunE`.

## **3. Verification**

Once Claude finishes, run these commands to verify:

1. **Re-authenticate** (to accept new scopes):
   rm ~/.config/gws/token.json  # Clear old token if needed
   go run main.go auth login

2. **Test Gmail:**
   go run main.go gmail list --max 5

3. **Test Chat:**
   go run main.go chat list

review and execute
