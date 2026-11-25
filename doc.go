// Command workspace-cli is an unofficial Google Workspace CLI written in Go.
//
// It is an independent open-source project and is not affiliated with,
// endorsed by, or sponsored by Google LLC. Google Workspace, Gmail, Google
// Drive, and Google Calendar are trademarks of Google LLC.
//
// The CLI is distributed as a single static binary named gws. It gives
// developers and AI agents structured JSON, YAML, or text output for Gmail,
// Google Calendar, Google Drive, Docs, Sheets, Slides, Tasks, Chat,
// Contacts, Groups, Keep, Forms, Workspace Events, and Custom Search.
//
// # Installation
//
// Install the gws binary with Go 1.25 or later:
//
//	go install github.com/omriariav/workspace-cli/cmd/gws@latest
//
// Prebuilt binaries for macOS and Linux are attached to each release at
// https://github.com/omriariav/workspace-cli/releases.
//
// # Usage
//
// Configure an OAuth client, authenticate, and run commands:
//
//	export GWS_CLIENT_ID="your-client-id.apps.googleusercontent.com"
//	export GWS_CLIENT_SECRET="your-client-secret"
//	gws auth login
//
//	gws gmail list --max 5 --query "is:unread"
//	gws calendar events --days 7
//	gws drive search "quarterly report"
//	gws sheets read <spreadsheet-id> "Sheet1!A1:D10"
//
// Every command accepts --format json (the default), yaml, or text, and
// --schema or --show-scopes to describe a command without running it.
//
// # AI agents
//
// The repository also ships a Claude Code plugin and a Codex plugin with
// one skill per service. See the README for setup:
// https://github.com/omriariav/workspace-cli#readme
//
// This root package builds the same binary as cmd/gws; prefer the cmd/gws
// path above so the installed binary is named gws.
package main
