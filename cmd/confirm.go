package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// The opt-in confirmation gate (confirm_sensitive / GWS_CONFIRM_SENSITIVE)
// covers two kinds of command:
//
//   - every command whose name starts with one of destructivePrefixes —
//     a naming rule, so a newly added delete command is covered without
//     anyone remembering to list it;
//   - the commands listed in sensitiveCommands, which send messages or
//     change who can access data, and whose names follow no single pattern;
//   - the commands in conditionalSensitive, when their triggering flag is
//     used (e.g. `drive update --trashed`).
//
// sensitiveCommands also supplies a specific description for some
// destructive commands. Keys are command paths without the leading "gws ".
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
	"calendar unsubscribe":     "removes a calendar from your calendar list",
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
	"drive start-approval":    "sends an approval request to reviewers",
	"drive reassign-approval": "changes the reviewers of an approval request",
	"drive delete-revision":   "deletes a file revision",
	"drive delete-comment":    "deletes a comment",
	"drive delete-reply":      "deletes a comment reply",
	"drive delete-drive":      "deletes a shared drive",

	// Docs / Sheets / Slides
	"docs trash":           "moves a document to Trash",
	"docs replace-content": "replaces the entire body of a document",
	"sheets delete-sheet":  "deletes a sheet",
	"slides delete-slide":  "deletes a slide",

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

// conditionalSensitive lists commands that send, share or delete only when
// a specific flag carries a value: the same command without that flag is
// an ordinary create or edit.
var conditionalSensitive = map[string]conditionalRule{
	"drive update":       {flag: "trashed", effect: "moves a file to Trash"},
	"chat setup-space":   {flag: "members", effect: "creates a chat space and adds members to it"},
	"calendar create":    {flag: "attendees", effect: "invites attendees to an event"},
	"calendar update":    {flag: "add-attendees", effect: "invites attendees to an event"},
	"calendar rsvp":      {flag: "message", effect: "sends a message to all attendees of an event"},
	"gmail label":        {flag: "add", effect: "moves a message to Trash or Spam", when: addsTrashOrSpam},
	"gmail batch-modify": {flag: "add-labels", effect: "moves messages to Trash or Spam", when: addsTrashOrSpam},
}

// conditionalRule makes a command sensitive when flag is set, or, if when
// is given, when the flag's value satisfies it.
type conditionalRule struct {
	flag, effect string
	when         func(value string) bool
}

// triggered reports whether the rule applies to the flag as currently set.
func (r conditionalRule) triggered(f *pflag.Flag) bool {
	if r.when != nil {
		return r.when(f.Value.String())
	}
	return flagIsSet(f)
}

// addsTrashOrSpam reports whether a comma-separated label list includes
// the TRASH or SPAM system label, in any letter case.
func addsTrashOrSpam(labels string) bool {
	for _, label := range strings.Split(labels, ",") {
		switch strings.ToUpper(strings.TrimSpace(label)) {
		case "TRASH", "SPAM":
			return true
		}
	}
	return false
}

// flagIsSet reports whether a flag carries a real value, judged by the
// flag's type: a bool must be true, a slice non-empty, and anything else a
// non-empty string (so a literal message of "false", "[]" or " " counts,
// exactly as it does for the command's own handler).
func flagIsSet(f *pflag.Flag) bool {
	v := f.Value.String()
	switch f.Value.Type() {
	case "bool":
		return v == "true"
	case "stringSlice", "stringArray", "intSlice":
		return v != "[]"
	default:
		// Not trimmed: handlers act on any non-empty value, so a
		// whitespace-only RSVP message still notifies attendees.
		return v != ""
	}
}

// destructivePrefixes are the command-name prefixes that mark a command as
// deleting or clearing data. "untrash" and "unshare" style names are not
// matched by prefix; unshare commands are listed explicitly above.
var destructivePrefixes = []string{"delete", "trash", "remove", "clear", "empty", "batch-delete"}

// sensitiveEffect reports whether cmd is covered by the confirmation gate
// and describes its effect for the prompt.
func sensitiveEffect(cmd *cobra.Command) (string, bool) {
	if cmd == nil {
		return "", false
	}
	key := commandKey(cmd)
	if effect, ok := sensitiveCommands[key]; ok {
		return effect, true
	}
	// Commands that are sensitive only when a particular flag is used.
	if rule, ok := conditionalSensitive[key]; ok {
		if f := cmd.Flags().Lookup(rule.flag); f != nil && rule.triggered(f) {
			return rule.effect, true
		}
	}
	// Only runnable leaf commands: a group such as `gws events
	// subscriptions` is never the thing being confirmed.
	if cmd.HasSubCommands() || !cmd.HasParent() || cmd.Parent() == cmd.Root() {
		return "", false
	}
	name := cmd.Name()
	for _, prefix := range destructivePrefixes {
		if strings.HasPrefix(name, prefix) {
			return "deletes or clears data", true
		}
	}
	return "", false
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
	effect, sensitive := sensitiveEffect(cmd)
	if !sensitive {
		return nil
	}
	key := commandKey(cmd)

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
