// Package cmd implements the commands of gws, an unofficial Google
// Workspace CLI written in Go.
//
// The project is independent and is not affiliated with, endorsed by, or
// sponsored by Google LLC.
//
// Each Google Workspace service has one file defining its Cobra command
// group: Gmail, Google Calendar, Google Drive, Docs, Sheets, Slides, Tasks,
// Chat, Contacts, Groups, Keep, Forms, Workspace Events, and Custom Search.
// Commands print structured JSON by default (YAML or text with --format)
// and map Google API errors to stable exit codes.
//
// The package is the implementation of the gws binary rather than a
// library with a stable API. Install the binary with:
//
//	go install github.com/omriariav/workspace-cli/cmd/gws@latest
package cmd
