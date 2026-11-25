package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// sensitiveCommands lists the commands that send messages, change who can
// access data, or delete data. When the opt-in confirmation gate is on
// (confirm_sensitive / GWS_CONFIRM_SENSITIVE), these commands require an
// explicit approval before they run. Keys are command paths without the
// leading "gws "; values describe the effect shown in the prompt.
//
// TestSensitiveCommands_Exist fails if an entry no longer resolves to a
// real command, so renames cannot silently drop a command from the gate.
var sensitiveCommands = map[string]string{
	// Gmail
	"gmail send":          "sends an email",
	"gmail reply":         "sends an email reply",
	"gmail forward":       "forwards an email",
	"gmail send-draft":    "sends a draft email",
	"gmail trash":         "moves a message to Trash",
	"gmail trash-thread":  "moves a thread to Trash",
	"gmail delete":        "permanently deletes a message",
	"gmail delete-thread": "permanently deletes a thread",
	"gmail batch-delete":  "permanently deletes messages",
	"gmail delete-label":  "deletes a label",
	"gmail delete-draft":  "deletes a draft",

	// Calendar
	"calendar delete":          "deletes an event",
	"calendar delete-calendar": "deletes a calendar",
	"calendar clear":           "deletes every event on a calendar",
	"calendar share":           "shares a calendar",
	"calendar unshare":         "removes access to a calendar",
	"calendar update-acl":      "changes who can access a calendar",

	// Tasks
	"tasks delete":      "deletes a task",
	"tasks delete-list": "deletes a task list",
	"tasks clear":       "clears completed tasks",

	// Drive
	"drive delete":            "deletes a file",
	"drive empty-trash":       "permanently deletes everything in Trash",
	"drive share":             "shares a file",
	"drive unshare":           "removes access to a file",
	"drive update-permission": "changes who can access a file",
	"drive delete-revision":   "deletes a file revision",
	"drive delete-comment":    "deletes a comment",
	"drive delete-reply":      "deletes a comment reply",
	"drive delete-drive":      "deletes a shared drive",

	// Docs / Sheets / Slides
	"docs trash":          "moves a document to Trash",
	"sheets delete-sheet": "deletes a sheet",
	"slides delete-slide": "deletes a slide",

	// Chat
	"chat send":          "sends a chat message",
	"chat delete":        "deletes a chat message",
	"chat delete-space":  "deletes a chat space",
	"chat add-member":    "adds a member to a chat space",
	"chat remove-member": "removes a member from a chat space",
	"chat update-member": "changes a member's role in a chat space",

	// Contacts
	"contacts delete":       "deletes a contact",
	"contacts batch-delete": "deletes contacts",
	"contacts delete-photo": "deletes a contact photo",

	// Workspace Events
	"events subscriptions delete": "deletes an event subscription",
}

// commandKey returns the command path without the root command name,
// e.g. "gmail send" for `gws gmail send`.
func commandKey(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := cmd.CommandPath()
	if i := strings.Index(path, " "); i >= 0 {
		return path[i+1:]
	}
	return ""
}

// stdinIsTerminal reports whether stdin is an interactive terminal.
// (A character-device check is not enough: /dev/null is one too.)
func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// confirmGate enforces the opt-in confirmation gate. It returns nil when
// the command may run. The prompt is written to errW so stdout stays clean
// for structured output. Without a terminal there is nobody to ask, so the
// gate fails closed and tells the caller to re-run with --yes once the
// user has approved.
func confirmGate(cmd *cobra.Command, enabled, yes bool, in io.Reader, errW io.Writer, interactive bool) error {
	if !enabled || yes {
		return nil
	}
	key := commandKey(cmd)
	effect, sensitive := sensitiveCommands[key]
	if !sensitive {
		return nil
	}

	if !interactive {
		return fmt.Errorf("confirmation required: `gws %s` %s and confirm_sensitive is on; ask the user to approve, then re-run with --yes", key, effect)
	}

	fmt.Fprintf(errW, "gws %s %s. Proceed? [y/N]: ", key, effect)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return nil
	}
	return fmt.Errorf("cancelled: `gws %s` was not confirmed", key)
}
