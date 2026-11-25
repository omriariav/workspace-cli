package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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
	"calendar move":            "moves an event to another calendar, which changes its organizer",
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
	"drive move":              "moves a file, which changes who can access it if the folders are shared differently",
	// A copy lands in --folder or, when that is omitted, beside the source,
	// and takes on that folder's sharing either way.
	"drive copy":              "creates a copy of a file, which takes on the sharing of the folder it lands in",
	"drive convert":           "creates a converted copy of a file, which takes on the sharing of the folder it lands in",
	"drive add-comment":       "posts a comment on a file, visible to everyone with access",
	"drive reply":             "posts a reply to a comment, visible to everyone with access",
	"drive comment-approval":  "posts a comment on a file approval, visible to its participants",
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
	"sheets merge":         "merges cells, discarding every value in the range except the upper-left one",
	"sheets copy-to":       "copies a sheet into another spreadsheet, where everyone with access to that spreadsheet can see it",
	"slides delete-slide":  "deletes a slide",

	// Chat
	"chat send":          "sends a chat message",
	"chat upload":        "uploads a local file to a chat space",
	"chat react":         "adds a reaction to a chat message, visible to everyone in the space",
	"chat unreact":       "removes a reaction from a chat message",
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
	"events subscriptions delete":     "deletes an event subscription",
	"events subscriptions create":     "starts ongoing delivery of Workspace events to a Pub/Sub topic",
	"events subscriptions reactivate": "resumes delivery of Workspace events to a Pub/Sub topic",
	"events subscriptions renew":      "extends how long Workspace events are delivered to a Pub/Sub topic",
	"events subscriptions update":     "changes what a subscription delivers or how long it runs",

	// Credentials
	"auth logout": "revokes the stored Google authorization and deletes the local login",
}

// conditionalSensitive lists commands that send, share or delete only when
// a specific flag carries a value: the same command without that flag is
// an ordinary create or edit.
var conditionalSensitive = map[string]conditionalRule{
	"drive update":     {flag: "trashed", effect: "moves a file to Trash"},
	"chat setup-space": {flag: "members", effect: "creates a chat space and adds members to it"},
	"calendar create":  {flag: "attendees", effect: "invites attendees to an event"},
	"calendar update":  {flag: "add-attendees", effect: "invites attendees to an event"},
	"calendar rsvp":    {flag: "message", effect: "sends a message to all attendees of an event"},
	// Placing a file in a specific folder gives it that folder's sharing.
	"drive upload":        {flag: "folder", effect: "uploads a file into a folder, where it takes on that folder's sharing"},
	"drive create-folder": {flag: "parent", effect: "creates a folder inside another, where it takes on that folder's sharing"},
	// Resolving or reopening a comment is not gated, but attaching a note
	// posts a reply that everyone with access to the file can read.
	// A message on an approval decision is shown to the other reviewers.
	"drive approve":           {flag: "message", effect: "records an approval with a message visible to the other reviewers"},
	"drive decline":           {flag: "message", effect: "declines an approval with a message visible to the other reviewers"},
	"drive cancel-approval":   {flag: "message", effect: "cancels an approval with a message visible to the reviewers"},
	"drive resolve-comment":   {flag: "content", effect: "posts a reply on a file comment, visible to everyone with access"},
	"drive unresolve-comment": {flag: "content", effect: "posts a reply on a file comment, visible to everyone with access"},
	"forms update":            {flag: "file", effect: "applies a batch of form changes from a file, which can delete questions"},
	// An empty replacement deletes every match.
	"docs replace":             {flag: "replace", effect: "deletes every occurrence of the matched text", when: isEmptyValue, values: emptyValue},
	"slides replace-text":      {flag: "replace", effect: "deletes every occurrence of the matched text", when: isEmptyValue, values: emptyValue},
	"docs replace-named-range": {flag: "text", effect: "erases the contents of every range with that name", when: isEmptyValue, values: emptyValue},
	"sheets find-replace":      {flag: "replace", effect: "deletes every occurrence of the matched text", when: isEmptyValue, values: emptyValue},
	"gmail label":              {flag: "add", effect: "moves a message to Trash or Spam", when: addsTrashOrSpam, values: trashOrSpamLabels},
	"gmail batch-modify":       {flag: "add-labels", effect: "moves messages to Trash or Spam", when: addsTrashOrSpam, values: trashOrSpamLabels},
}

// payloadSensitive lists commands that are sensitive depending on what
// their payload contains, which a single flag-and-value rule cannot
// express. flags and condition describe the rule for --schema.
var payloadSensitive = map[string]payloadRule{
	// The Sheets API treats an empty string as "clear this cell".
	"sheets write":       {flags: []string{"values", "values-json"}, condition: "a cell value is an empty string", effect: "clears the cells it writes an empty value to", check: sheetsWriteClearsCells},
	"sheets batch-write": {flags: []string{"values"}, condition: "a cell value is an empty string", effect: "clears the cells it writes an empty value to", check: sheetsBatchWriteClearsCells},
}

type payloadRule struct {
	flags     []string
	condition string
	effect    string
	check     func(cmd *cobra.Command) bool
}

// hasEmptyCell reports whether any cell is an empty string, which clears
// the cell it is written to. A JSON null is different: the Sheets API
// skips it and leaves the existing cell as it is.
func hasEmptyCell(rows [][]interface{}) bool {
	for _, row := range rows {
		for _, cell := range row {
			if text, ok := cell.(string); ok && text == "" {
				return true
			}
		}
	}
	return false
}

// sheetsWriteClearsCells reads the values exactly as `sheets write` will.
// A payload that does not parse is left to the command to reject.
func sheetsWriteClearsCells(cmd *cobra.Command) bool {
	values, err := parseValues(cmd)
	return err == nil && hasEmptyCell(values)
}

// sheetsBatchWriteClearsCells does the same for each --values of
// `sheets batch-write`.
func sheetsBatchWriteClearsCells(cmd *cobra.Command) bool {
	payloads, _ := cmd.Flags().GetStringArray("values")
	for _, payload := range payloads {
		var rows [][]interface{}
		if err := json.Unmarshal([]byte(payload), &rows); err == nil && hasEmptyCell(rows) {
			return true
		}
	}
	return false
}

// conditionalRule makes a command sensitive when flag is set, or, if when
// is given, when the flag's value satisfies it.
//
// values lists, for --schema, the flag values that when accepts, so a
// caller can tell beforehand which invocations will ask for confirmation.
// A rule with when must also set values.
type conditionalRule struct {
	flag, effect string
	when         func(value string) bool
	values       []string
}

// emptyValue is the --schema value list of a rule triggered by an
// explicitly empty flag (--replace "").
var emptyValue = []string{""}

// isEmptyValue reports whether a flag was given the empty string.
func isEmptyValue(v string) bool { return v == "" }

// trashOrSpamLabels are the label values addsTrashOrSpam looks for.
var trashOrSpamLabels = []string{"TRASH", "SPAM"}

// triggered reports whether the rule applies to the flag as currently set.
func (r conditionalRule) triggered(f *pflag.Flag) bool {
	if r.when != nil {
		// A value only counts if it was given: a default is not a choice.
		return f.Changed && r.when(f.Value.String())
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
	if rule, ok := payloadSensitive[key]; ok && rule.check(cmd) {
		return rule.effect, true
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

	// An approval only counts if the question was actually shown.
	// The exact invocation is shown too, so the person approving can see
	// which file, recipient or role this is about.
	if _, err := fmt.Fprintf(errW, "About to run: %s\ngws %s %s. Proceed? [y/N]: ", invocationSummary(cmd), key, effect); err != nil {
		return fmt.Errorf("could not display the confirmation prompt for `gws %s` (%v), so it was not run; re-run with --yes after the user approves", key, err)
	}
	// Only a complete line counts: an answer cut off by EOF or a read
	// error is not an approval.
	line, readErr := bufio.NewReader(in).ReadString('\n')
	if readErr == nil {
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			return nil
		}
	}
	return fmt.Errorf("cancelled: `gws %s` was not confirmed", key)
}

// promptValueLimit caps how much of a content value (a message body) the
// confirmation prompt shows.
const promptValueLimit = 80

// promptContentFlags are flags that carry content rather than say what
// the command acts on. Only these are shortened in the prompt: everything
// else (recipients, IDs, roles, labels) is shown in full, because it is
// what the person approving needs to check.
var promptContentFlags = map[string]bool{
	"body": true, "body-html": true, "html": true, "text": true,
	"content": true, "message": true, "description": true, "notes": true,
	"quoted-content": true, "values": true, "values-json": true,
}

// invocationSummary renders the command as it was invoked: its arguments
// and the flags that were set, for display in the confirmation prompt.
// Credentials are left out, control characters cannot reach the terminal,
// and content values (a message body) are shortened.
func invocationSummary(cmd *cobra.Command) string {
	parts := []string{"gws", commandKey(cmd)}
	for _, arg := range cmd.Flags().Args() {
		parts = append(parts, strconv.QuoteToGraphic(arg))
	}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "access-token", "yes":
			return
		}
		if f.Value.Type() == "bool" {
			if f.Value.String() == "true" {
				parts = append(parts, "--"+f.Name)
			} else {
				parts = append(parts, "--"+f.Name+"=false")
			}
			return
		}
		value := f.Value.String()
		if runes := []rune(value); promptContentFlags[f.Name] && len(runes) > promptValueLimit {
			value = string(runes[:promptValueLimit]) + "…"
		}
		parts = append(parts, "--"+f.Name, strconv.QuoteToGraphic(value))
	})
	return strings.Join(parts, " ")
}
