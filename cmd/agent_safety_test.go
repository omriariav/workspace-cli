package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

// --- confirmation gate ---

func TestSensitiveCommands_Exist(t *testing.T) {
	for key := range sensitiveCommands {
		cmd, _, err := rootCmd.Find(strings.Fields(key))
		if err != nil || cmd == nil || commandKey(cmd) != key {
			t.Errorf("sensitive command %q does not resolve to a real command", key)
		}
	}
}

func TestConfirmGate(t *testing.T) {
	send, _, _ := rootCmd.Find([]string{"gmail", "send"})
	list, _, _ := rootCmd.Find([]string{"gmail", "list"})

	tests := []struct {
		name        string
		cmd         *cobra.Command
		enabled     bool
		yes         bool
		input       string
		interactive bool
		wantErr     string
		wantPrompt  bool
	}{
		{name: "gate off runs sensitive command", cmd: send},
		{name: "non-sensitive command runs", cmd: list, enabled: true},
		{name: "--yes skips prompt", cmd: send, enabled: true, yes: true},
		{name: "no terminal fails closed", cmd: send, enabled: true, wantErr: "confirmation required"},
		{name: "approved", cmd: send, enabled: true, interactive: true, input: "y\n", wantPrompt: true},
		{name: "approved yes", cmd: send, enabled: true, interactive: true, input: "YES\n", wantPrompt: true},
		{name: "declined", cmd: send, enabled: true, interactive: true, input: "n\n", wantErr: "cancelled", wantPrompt: true},
		{name: "empty answer declines", cmd: send, enabled: true, interactive: true, input: "\n", wantErr: "cancelled", wantPrompt: true},
		{name: "closed stdin declines", cmd: send, enabled: true, interactive: true, input: "", wantErr: "cancelled", wantPrompt: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errW bytes.Buffer
			err := confirmGate(tt.cmd, tt.enabled, tt.yes, strings.NewReader(tt.input), &errW, tt.interactive)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
			if gotPrompt := strings.Contains(errW.String(), "Proceed?"); gotPrompt != tt.wantPrompt {
				t.Errorf("prompt shown = %v, want %v (stderr: %q)", gotPrompt, tt.wantPrompt, errW.String())
			}
		})
	}
}

// failingWriter rejects every write, like a closed stderr.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stderr closed") }

func TestConfirmGate_NoApprovalWithoutAVisiblePrompt(t *testing.T) {
	send, _, _ := rootCmd.Find([]string{"gmail", "send"})
	// A "y" is already waiting on stdin, but the prompt cannot be shown.
	err := confirmGate(send, true, false, strings.NewReader("y\n"), failingWriter{}, true)
	if err == nil || !strings.Contains(err.Error(), "could not display the confirmation prompt") {
		t.Fatalf("error = %v, want a prompt-display failure", err)
	}
}

func TestConfirmSensitive_ReadsEnv(t *testing.T) {
	t.Setenv("GWS_CONFIRM_SENSITIVE", "1")
	viper.SetEnvPrefix("GWS")
	viper.AutomaticEnv()
	if !viper.GetBool("confirm_sensitive") {
		t.Error("GWS_CONFIRM_SENSITIVE=1 should enable the gate")
	}
}

// --- introspection ---

func runIntrospection(t *testing.T, args ...string) (map[string]interface{}, bool) {
	t.Helper()
	var out bytes.Buffer
	handled, err := handleIntrospection(rootCmd, args, &out)
	if err != nil {
		t.Fatalf("handleIntrospection(%v) error: %v", args, err)
	}
	if !handled {
		return nil, false
	}
	var result map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	return result, true
}

func TestIntrospection_NotHandledWithoutFlags(t *testing.T) {
	if _, handled := runIntrospection(t, "gmail", "list"); handled {
		t.Error("expected normal execution without --schema/--show-scopes")
	}
	// A literal after "--" is an argument, not a flag.
	if _, handled := runIntrospection(t, "gmail", "list", "--", "--schema"); handled {
		t.Error("--schema after -- must not trigger introspection")
	}
}

func TestIntrospection_Schema(t *testing.T) {
	// Works without the command's required flags or positional args.
	result, handled := runIntrospection(t, "calendar", "freebusy", "--schema")
	if !handled {
		t.Fatal("expected --schema to be handled")
	}
	if result["command"] != "gws calendar freebusy" {
		t.Errorf("command = %v", result["command"])
	}
	flags, _ := result["flags"].([]interface{})
	byName := map[string]map[string]interface{}{}
	for _, f := range flags {
		m := f.(map[string]interface{})
		byName[m["name"].(string)] = m
	}
	if f := byName["from"]; f == nil || f["required"] != true || f["type"] != "string" {
		t.Errorf("--from should be a required string flag, got %v", f)
	}
	if f := byName["calendars"]; f == nil || f["required"] != false || f["default"] != "primary" {
		t.Errorf("--calendars should be optional with default primary, got %v", f)
	}
	if f := byName["format"]; f == nil || f["global"] != true {
		t.Errorf("--format should be reported as a global flag, got %v", f)
	}
	if _, ok := result["sensitive"]; ok {
		t.Error("freebusy must not be marked sensitive")
	}
}

func TestIntrospection_SchemaMarksSensitiveAndListsSubcommands(t *testing.T) {
	result, _ := runIntrospection(t, "gmail", "send", "--schema")
	if result["sensitive"] != true {
		t.Errorf("gmail send should be marked sensitive: %v", result)
	}
	group, _ := runIntrospection(t, "gmail", "--schema")
	subs, _ := group["subcommands"].([]interface{})
	if len(subs) < 10 {
		t.Errorf("gmail group should list its subcommands, got %d", len(subs))
	}
}

func TestIntrospection_SchemaReportsConditionalSensitivity(t *testing.T) {
	// With or without the triggering flag on the command line.
	for _, args := range [][]string{
		{"calendar", "create", "--schema"},
		{"calendar", "create", "--attendees", "a@example.com", "--schema"},
	} {
		result, handled := runIntrospection(t, args...)
		if !handled {
			t.Fatalf("%v: expected introspection", args)
		}
		when, _ := result["sensitive_when"].(map[string]interface{})
		if when == nil || when["flag"] != "attendees" || when["effect"] == "" {
			t.Errorf("%v: sensitive_when = %v", args, result["sensitive_when"])
		}
	}
	plain, _ := runIntrospection(t, "calendar", "events", "--schema")
	if _, ok := plain["sensitive_when"]; ok {
		t.Error("calendar events has no conditional rule")
	}
}

func TestIntrospection_ShowScopes(t *testing.T) {
	result, handled := runIntrospection(t, "gmail", "send", "--show-scopes")
	if !handled {
		t.Fatal("expected --show-scopes to be handled")
	}
	scopes := toStrings(result["scopes"])
	ro := toStrings(result["readonly_scopes"])
	if !containsStr(scopes, "https://www.googleapis.com/auth/gmail.send") {
		t.Errorf("scopes missing gmail.send: %v", scopes)
	}
	if containsStr(ro, "https://www.googleapis.com/auth/gmail.send") || !containsStr(ro, "https://www.googleapis.com/auth/gmail.readonly") {
		t.Errorf("readonly_scopes wrong: %v", ro)
	}
	// Suggested logins extend the stored authorization, never replace it.
	if result["login"] != "gws auth login --add --services gmail" || result["login_readonly"] != "gws auth login --add --services gmail --readonly" {
		t.Errorf("login = %v, login_readonly = %v", result["login"], result["login_readonly"])
	}
	if note, _ := result["login_note"].(string); !strings.Contains(note, "replaces") {
		t.Errorf("login_note = %v", result["login_note"])
	}

	activity, _ := runIntrospection(t, "drive", "activity", "--show-scopes")
	activityScopes := toStrings(activity["scopes"])
	if !containsStr(activityScopes, "https://www.googleapis.com/auth/drive.activity.readonly") {
		t.Errorf("drive activity should include the Drive Activity scope: %v", activityScopes)
	}
	// It uses only the Drive Activity API, so it must not suggest granting
	// Drive file access.
	for _, s := range activityScopes {
		if strings.HasSuffix(s, "/auth/drive") || strings.HasSuffix(s, "/auth/drive.file") || strings.HasSuffix(s, "/auth/drive.readonly") {
			t.Errorf("drive activity should not request %s", s)
		}
	}
	if activity["login"] != "gws auth login --add --services driveactivity" {
		t.Errorf("drive activity login = %v", activity["login"])
	}

	// docs trash goes through the Drive API, so it needs Drive scopes.
	trash, _ := runIntrospection(t, "docs", "trash", "--show-scopes")
	if !containsStr(toStrings(trash["scopes"]), "https://www.googleapis.com/auth/drive") || containsStr(toStrings(trash["scopes"]), "https://www.googleapis.com/auth/documents") {
		t.Errorf("docs trash scopes = %v, want Drive scopes", trash["scopes"])
	}
	docsRead, _ := runIntrospection(t, "docs", "read", "--show-scopes")
	if !containsStr(toStrings(docsRead["scopes"]), "https://www.googleapis.com/auth/documents") {
		t.Errorf("docs read scopes = %v, want Docs scopes", docsRead["scopes"])
	}

	// auth login is what requests scopes: it must not claim to use none.
	login, _ := runIntrospection(t, "auth", "login", "--show-scopes")
	loginNote, _ := login["note"].(string)
	if len(toStrings(login["scopes"])) == 0 || len(toStrings(login["readonly_scopes"])) == 0 ||
		!strings.Contains(loginNote, "--services") || !strings.Contains(loginNote, "--readonly") || !strings.Contains(loginNote, "--add") {
		t.Errorf("auth login --show-scopes = %v", login)
	}
	if strings.Contains(loginNote, "does not use Google OAuth scopes") {
		t.Errorf("auth login note = %q", loginNote)
	}
	// Other auth commands really use none.
	status, _ := runIntrospection(t, "auth", "status", "--show-scopes")
	if len(toStrings(status["scopes"])) != 0 {
		t.Errorf("auth status scopes = %v", status["scopes"])
	}

	version, _ := runIntrospection(t, "version", "--show-scopes")
	if len(toStrings(version["scopes"])) != 0 || version["note"] == nil {
		t.Errorf("version uses no OAuth scopes: %v", version)
	}
}

func TestIntrospection_BothFlags(t *testing.T) {
	result, _ := runIntrospection(t, "tasks", "list", "--schema", "--show-scopes")
	if result["flags"] == nil || result["scopes"] == nil {
		t.Errorf("expected both schema and scopes: %v", result)
	}
}

func TestIntrospection_UnknownCommandFallsThrough(t *testing.T) {
	// Normal execution reports the unknown command (exit code 2).
	var out bytes.Buffer
	handled, err := handleIntrospection(rootCmd, []string{"nope", "--schema"}, &out)
	if handled || err != nil || out.Len() != 0 {
		t.Errorf("unknown command should fall through, got handled=%v err=%v out=%q", handled, err, out.String())
	}
}

func TestIntrospection_FlagValueIsNotTheFlag(t *testing.T) {
	// "--schema" here is the value of --body: the command must run.
	cases := [][]string{
		{"gmail", "send", "--to", "a@example.com", "--subject", "Hi", "--body", "--schema"},
		{"gmail", "send", "--to", "a@example.com", "--subject", "--show-scopes", "--body", "x"},
		{"gmail", "send", "--body=--schema", "--to", "a@example.com"},
		{"gmail", "list", "--schema=false"},
		{"gmail", "list", "--no-such-flag", "--schema"},
	}
	for _, args := range cases {
		var out bytes.Buffer
		handled, err := handleIntrospection(rootCmd, args, &out)
		if handled || err != nil || out.Len() != 0 {
			t.Errorf("%v: expected fall-through, got handled=%v err=%v out=%q", args, handled, err, out.String())
		}
	}

	// The real flag still works alongside flags that take values.
	for _, args := range [][]string{
		{"gmail", "send", "--to", "a@example.com", "--schema"},
		{"gmail", "send", "--schema=true", "--body", "x"},
		{"--format", "yaml", "gmail", "send", "--schema"},
	} {
		if _, handled := runIntrospection(t, args...); !handled {
			t.Errorf("%v: expected introspection", args)
		}
	}
}

func TestIntrospection_DoesNotMutateCommandFlags(t *testing.T) {
	send, _, _ := rootCmd.Find([]string{"gmail", "send"})
	runIntrospection(t, "gmail", "send", "--attachment", "a.pdf", "--to", "x@example.com", "--schema")
	if send.Flags().Changed("attachment") || send.Flags().Changed("to") {
		t.Error("introspection must not set the command's real flags")
	}
}

func TestLoginTokenToStore(t *testing.T) {
	// A login without its own refresh token is never stored, whether it
	// replaces or extends the previous authorization.
	for _, tok := range []*oauth2.Token{nil, {AccessToken: "new"}} {
		if _, err := loginTokenToStore(tok); err == nil {
			t.Errorf("login %+v without a refresh token should fail", tok)
		}
	}
	got, err := loginTokenToStore(&oauth2.Token{AccessToken: "new", RefreshToken: "new-refresh"})
	if err != nil || got.RefreshToken != "new-refresh" {
		t.Errorf("login = %+v, %v", got, err)
	}
}

func TestSensitiveEffect_CoversDestructiveCommandsByName(t *testing.T) {
	covered := [][]string{
		{"docs", "delete"}, {"docs", "delete-object"}, {"docs", "delete-table-row"}, {"docs", "delete-tab"}, {"docs", "remove-list"},
		{"slides", "delete-object"}, {"slides", "delete-text"}, {"slides", "delete-table-row"},
		{"sheets", "delete-rows"}, {"sheets", "clear"}, {"sheets", "delete-chart"},
		{"drive", "empty-trash"}, {"gmail", "batch-delete"}, {"gmail", "trash-thread"},
		{"tasks", "clear"}, {"chat", "remove-member"}, {"contacts", "delete-photo"},
		{"calendar", "update-acl"}, {"chat", "update-member"}, {"drive", "update-permission"},
		{"gmail", "send"}, {"chat", "send"}, {"drive", "share"}, {"calendar", "unshare"},
		{"events", "subscriptions", "delete"},
		{"events", "subscriptions", "create"},
		{"events", "subscriptions", "reactivate"},
		{"drive", "start-approval"}, {"drive", "reassign-approval"},
		{"calendar", "unsubscribe"},
		{"docs", "replace-content"},
		{"drive", "move"},
	}
	for _, path := range covered {
		cmd, _, err := rootCmd.Find(path)
		if err != nil || commandKey(cmd) != strings.Join(path, " ") {
			t.Fatalf("command %v not found", path)
		}
		if _, ok := sensitiveEffect(cmd); !ok {
			t.Errorf("gws %s should require confirmation", strings.Join(path, " "))
		}
	}

	notCovered := [][]string{
		{"gmail", "list"}, {"gmail", "untrash"}, {"gmail", "untrash-thread"}, {"gmail", "create-draft"},
		{"drive", "download"}, {"docs", "read"}, {"sheets", "write"}, {"calendar", "create"},
		{"auth", "logout"}, {"hooks", "disable"}, {"gmail"}, {"events", "subscriptions"},
	}
	for _, path := range notCovered {
		cmd, _, err := rootCmd.Find(path)
		if err != nil {
			t.Fatalf("command %v not found", path)
		}
		if _, ok := sensitiveEffect(cmd); ok {
			t.Errorf("gws %s should not require confirmation", strings.Join(path, " "))
		}
	}
	if _, ok := sensitiveEffect(nil); ok {
		t.Error("nil command is not sensitive")
	}
}

// TestSensitiveEffect_EveryDestructiveNameIsGated walks the whole command
// tree: any runnable command whose name starts with a destructive prefix
// must be gated, so a new delete command cannot be added ungated.
func TestSensitiveEffect_EveryDestructiveNameIsGated(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.HasSubCommands() {
				walk(sub)
				continue
			}
			for _, prefix := range destructivePrefixes {
				if strings.HasPrefix(sub.Name(), prefix) {
					if _, ok := sensitiveEffect(sub); !ok {
						t.Errorf("gws %s has a destructive name but is not gated", commandKey(sub))
					}
				}
			}
		}
	}
	walk(rootCmd)
}

func TestSensitiveEffect_DriveUpdateOnlyWhenTrashing(t *testing.T) {
	update, _, _ := rootCmd.Find([]string{"drive", "update"})
	t.Cleanup(func() { _ = update.Flags().Set("trashed", "false") })

	if _, ok := sensitiveEffect(update); ok {
		t.Error("drive update without --trashed should not require confirmation")
	}
	if err := update.Flags().Set("trashed", "true"); err != nil {
		t.Fatal(err)
	}
	if _, ok := sensitiveEffect(update); !ok {
		t.Error("drive update --trashed should require confirmation")
	}
	// Fails closed without a terminal.
	var errW bytes.Buffer
	if err := confirmGate(update, true, false, strings.NewReader(""), &errW, false); err == nil || !strings.Contains(err.Error(), "confirmation required") {
		t.Errorf("gate error = %v", err)
	}
	// --trashed=false untrashes: no confirmation.
	_ = update.Flags().Set("trashed", "false")
	if _, ok := sensitiveEffect(update); ok {
		t.Error("drive update --trashed=false should not require confirmation")
	}
}

func TestSensitiveEffect_ChatSetupSpaceOnlyWithMembers(t *testing.T) {
	setup, _, _ := rootCmd.Find([]string{"chat", "setup-space"})
	t.Cleanup(func() { _ = setup.Flags().Set("members", "") })

	if _, ok := sensitiveEffect(setup); ok {
		t.Error("chat setup-space without --members should not require confirmation")
	}
	if err := setup.Flags().Set("members", "users/123,users/456"); err != nil {
		t.Fatal(err)
	}
	if _, ok := sensitiveEffect(setup); !ok {
		t.Error("chat setup-space --members should require confirmation")
	}
	var errW bytes.Buffer
	if err := confirmGate(setup, true, false, strings.NewReader(""), &errW, false); err == nil || !strings.Contains(err.Error(), "confirmation required") {
		t.Errorf("gate error = %v", err)
	}
}

func TestConditionalSensitive_LiteralFalseAndBracketMessages(t *testing.T) {
	// "false" and "[]" are real RSVP messages, not empty values.
	rsvp, _, _ := rootCmd.Find([]string{"calendar", "rsvp"})
	t.Cleanup(func() { _ = rsvp.Flags().Set("message", "") })
	for _, msg := range []string{"false", "[]", "ok", " ", "\t"} {
		if err := rsvp.Flags().Set("message", msg); err != nil {
			t.Fatal(err)
		}
		if _, ok := sensitiveEffect(rsvp); !ok {
			t.Errorf("calendar rsvp --message %q should require confirmation", msg)
		}
	}
	// Only a truly empty message sends no update.
	_ = rsvp.Flags().Set("message", "")
	if _, ok := sensitiveEffect(rsvp); ok {
		t.Error("calendar rsvp with an empty --message should not require confirmation")
	}
	// Bool flags are judged as bools.
	update, _, _ := rootCmd.Find([]string{"drive", "update"})
	_ = update.Flags().Set("trashed", "false")
	if _, ok := sensitiveEffect(update); ok {
		t.Error("drive update --trashed=false should not require confirmation")
	}
}

func TestConditionalSensitive_GmailLabelsToTrashOrSpam(t *testing.T) {
	batch, _, _ := rootCmd.Find([]string{"gmail", "batch-modify"})
	label, _, _ := rootCmd.Find([]string{"gmail", "label"})
	t.Cleanup(func() {
		_ = batch.Flags().Set("add-labels", "")
		_ = label.Flags().Set("add", "")
	})

	for _, labels := range []string{"TRASH", "trash", "STARRED, Spam", "INBOX,TRASH"} {
		_ = batch.Flags().Set("add-labels", labels)
		if _, ok := sensitiveEffect(batch); !ok {
			t.Errorf("gmail batch-modify --add-labels %q should require confirmation", labels)
		}
		_ = label.Flags().Set("add", labels)
		if _, ok := sensitiveEffect(label); !ok {
			t.Errorf("gmail label --add %q should require confirmation", labels)
		}
	}
	// Ordinary labels, including names that merely contain the word.
	for _, labels := range []string{"", "STARRED", "IMPORTANT,Work", "Trash-Reports", "NotSpam"} {
		_ = batch.Flags().Set("add-labels", labels)
		if _, ok := sensitiveEffect(batch); ok {
			t.Errorf("gmail batch-modify --add-labels %q should not require confirmation", labels)
		}
	}
	// Fails closed without a terminal.
	_ = batch.Flags().Set("add-labels", "TRASH")
	var errW bytes.Buffer
	if err := confirmGate(batch, true, false, strings.NewReader(""), &errW, false); err == nil || !strings.Contains(err.Error(), "confirmation required") {
		t.Errorf("gate error = %v", err)
	}
}

func TestConditionalSensitive_RulesResolveAndTrigger(t *testing.T) {
	values := map[string]string{"trashed": "true", "members": "users/1", "attendees": "a@example.com", "add-attendees": "a@example.com", "message": "running late", "add": "TRASH", "add-labels": "STARRED,trash", "event-types": "google.workspace.chat.message.v1.created"}
	for key, rule := range conditionalSensitive {
		cmd, _, err := rootCmd.Find(strings.Fields(key))
		if err != nil || commandKey(cmd) != key {
			t.Errorf("conditional rule %q does not resolve to a real command", key)
			continue
		}
		f := cmd.Flags().Lookup(rule.flag)
		if f == nil {
			t.Errorf("gws %s has no --%s flag", key, rule.flag)
			continue
		}
		if _, ok := sensitiveEffect(cmd); ok {
			t.Errorf("gws %s without --%s should not require confirmation", key, rule.flag)
		}
		original := f.Value.String()
		if err := cmd.Flags().Set(rule.flag, values[rule.flag]); err != nil {
			t.Fatalf("set --%s: %v", rule.flag, err)
		}
		if _, ok := sensitiveEffect(cmd); !ok {
			t.Errorf("gws %s --%s should require confirmation", key, rule.flag)
		}
		// Restore the flag for other tests (slice flags need replacing).
		if sv, ok := f.Value.(interface{ Replace([]string) error }); ok {
			_ = sv.Replace(nil)
		} else {
			_ = cmd.Flags().Set(rule.flag, original)
		}
		f.Changed = false
		if _, ok := sensitiveEffect(cmd); ok {
			t.Errorf("gws %s should not require confirmation after the flag is cleared", key)
		}
	}
}

func TestStoredOAuthLoginActive(t *testing.T) {
	if !storedOAuthLoginActive() {
		t.Error("default configuration uses the stored OAuth login")
	}
	viper.Set("access_token", "tok")
	if storedOAuthLoginActive() {
		t.Error("an external access token is not the stored OAuth login")
	}
	viper.Set("access_token", "")
	setAuthModeConfig(t, "service-account-dwd", "/tmp/key.json", "user@example.com")
	if storedOAuthLoginActive() {
		t.Error("a service-account mode is not the stored OAuth login")
	}
}

func TestExistingGrantedAccess_IgnoresMetadataWithoutToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, nil, "", tok); err != nil {
		t.Fatal(err)
	}
	if got := existingGrantedAccess(); len(got) != 0 {
		t.Errorf("metadata without a stored token must be ignored, got %v", got)
	}
	if err := auth.SaveToken(tok); err != nil {
		t.Fatal(err)
	}
	if got := existingGrantedAccess(); got["gmail"] != auth.AccessFull {
		t.Errorf("metadata with a stored token should load, got %v", got)
	}
}

func TestExistingGrantedAccess_LegacyVersusInterruptedLogin(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	current := &oauth2.Token{AccessToken: "a", RefreshToken: "current"}
	if err := auth.SaveToken(current); err != nil {
		t.Fatal(err)
	}
	if err := auth.SaveGrantedServices([]string{"gmail", "drive"}); err != nil {
		t.Fatal(err)
	}

	// Legacy login: a service list and no access record at all. Those
	// logins always granted full access.
	if got := existingGrantedAccess(); got["gmail"] != auth.AccessFull || got["drive"] != auth.AccessFull {
		t.Errorf("legacy login = %v, want gmail and drive full", got)
	}

	// Interrupted login: an access record exists but belongs to another
	// token. The service list must not be read as full access.
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessReadOnly}, nil, "", &oauth2.Token{RefreshToken: "other"}); err != nil {
		t.Fatal(err)
	}
	if got := existingGrantedAccess(); len(got) != 0 {
		t.Errorf("after an interrupted login nothing may be assumed, got %v", got)
	}

	// Once the record matches the stored token it is used as written.
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessReadOnly}, nil, "", current); err != nil {
		t.Fatal(err)
	}
	if got := existingGrantedAccess(); len(got) != 1 || got["gmail"] != auth.AccessReadOnly {
		t.Errorf("matching record = %v, want gmail readonly", got)
	}
}

func TestLogout_ClearsGrantedMetadata(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessReadOnly}, nil, "", &oauth2.Token{RefreshToken: "r"}); err != nil {
		t.Fatal(err)
	}
	if err := auth.SaveGrantedServices([]string{"gmail"}); err != nil {
		t.Fatal(err)
	}
	// No token stored: logout makes no network call and still clears metadata.
	if _, err := captureStdout(t, func() error { return runLogout(logoutCmd, nil) }); err != nil {
		t.Fatalf("runLogout: %v", err)
	}
	if _, err := os.Stat(config.GetGrantedAccessPath()); !os.IsNotExist(err) {
		t.Error("logout should delete the granted-access file")
	}
	if auth.LoadGrantedAccess() != nil || auth.LoadGrantedServices() != nil {
		t.Error("logout should remove granted-services and granted-access metadata")
	}
}

func TestCommandScopeOverrides_AreKnown(t *testing.T) {
	for key, services := range commandScopeOverrides {
		cmd, _, err := rootCmd.Find(strings.Fields(key))
		if err != nil || commandKey(cmd) != key {
			t.Errorf("scope override %q does not resolve to a real command", key)
		}
		for _, svc := range services {
			if _, ok := auth.ServiceScopes[svc]; !ok {
				t.Errorf("override %q maps to unknown scope service %q", key, svc)
			}
		}
	}
}

func TestCommandScopeServices_AreKnown(t *testing.T) {
	for group, services := range commandScopeServices {
		if findSubcommand(rootCmd, group) == nil {
			t.Errorf("command group %q does not exist", group)
		}
		for _, svc := range services {
			if _, ok := auth.ServiceScopes[svc]; !ok {
				t.Errorf("group %q maps to unknown scope service %q", group, svc)
			}
		}
	}
}

func TestReadOnlyScopeHint(t *testing.T) {
	send, _, _ := rootCmd.Find([]string{"gmail", "send"})
	insufficient := errors.New("googleapi: Error 403: Request had insufficient authentication scopes.")

	if hint := readOnlyScopeHint(send, insufficient, map[string]string{"gmail": auth.AccessReadOnly}); !strings.Contains(hint, "gws auth login --add --services gmail") {
		t.Errorf("hint = %q", hint)
	}
	if hint := readOnlyScopeHint(send, insufficient, map[string]string{"gmail": auth.AccessPartial}); !strings.Contains(hint, "--add --services gmail") {
		t.Errorf("partial-access hint = %q", hint)
	}
	if hint := readOnlyScopeHint(send, insufficient, map[string]string{"gmail": auth.AccessFull}); hint != "" {
		t.Errorf("no hint expected for full access, got %q", hint)
	}
	if hint := readOnlyScopeHint(send, errors.New("googleapi: Error 403: forbidden"), map[string]string{"gmail": auth.AccessReadOnly}); hint != "" {
		t.Errorf("no hint expected for unrelated 403, got %q", hint)
	}
	if hint := readOnlyScopeHint(nil, insufficient, map[string]string{"gmail": auth.AccessReadOnly}); hint != "" {
		t.Errorf("no hint expected without a command, got %q", hint)
	}
}

func TestWithScopeHint_IsPartOfTheStructuredError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
	if err := auth.SaveToken(tok); err != nil {
		t.Fatal(err)
	}
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessReadOnly}, nil, "", tok); err != nil {
		t.Fatal(err)
	}
	send, _, _ := rootCmd.Find([]string{"gmail", "send"})
	previous := executedCmd
	executedCmd = send
	t.Cleanup(func() { executedCmd = previous })

	apiErr := fmt.Errorf("failed to send: %w", &googleapi.Error{Code: 403, Message: "Request had insufficient authentication scopes."})

	hinted := withScopeHint(apiErr)
	if !strings.Contains(hinted.Error(), "gws auth login --add --services gmail") {
		t.Fatalf("hint missing: %v", hinted)
	}
	// Exit-code mapping still sees the API error underneath.
	if exitCodeForError(hinted) != ExitAuth {
		t.Errorf("exit code = %d, want %d", exitCodeForError(hinted), ExitAuth)
	}

	// Printed as JSON, stderr is exactly one document, hint included.
	var stderr bytes.Buffer
	p := &hintingPrinter{Printer: printer.NewWithWriters(io.Discard, &stderr, "json")}
	_ = p.PrintError(apiErr)
	var doc map[string]string
	if err := json.Unmarshal(stderr.Bytes(), &doc); err != nil {
		t.Fatalf("stderr is not a single JSON document: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(doc["error"], "insufficient authentication scopes") || !strings.Contains(doc["error"], "--add --services gmail") {
		t.Errorf("error field = %q", doc["error"])
	}

	// No hint for an unrelated failure, a full-access service, or when
	// another credential is in use.
	if got := withScopeHint(errors.New("boom")); got.Error() != "boom" {
		t.Errorf("non-auth error changed: %v", got)
	}
	viper.Set("access_token", "external")
	if got := withScopeHint(apiErr); got != apiErr {
		t.Errorf("hint added although an external token is in use: %v", got)
	}
	viper.Set("access_token", "")
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, nil, "", tok); err != nil {
		t.Fatal(err)
	}
	if got := withScopeHint(apiErr); got != apiErr {
		t.Errorf("hint added for a full-access service: %v", got)
	}
	if withScopeHint(nil) != nil {
		t.Error("nil should stay nil")
	}
}

// --- login scope resolution ---

func TestLoginScopes(t *testing.T) {
	if got := loginScopes(auth.ValidServiceNames(), false, false); len(got) != len(auth.AllScopes) {
		t.Errorf("unscoped full login should request AllScopes, got %d scopes", len(got))
	}
	ro := loginScopes([]string{"gmail", "sheets"}, true, true)
	want := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/spreadsheets.readonly",
	}
	if strings.Join(ro, " ") != strings.Join(want, " ") {
		t.Errorf("readonly scopes = %v, want %v", ro, want)
	}
	for _, s := range loginScopes(auth.ValidServiceNames(), true, false) {
		if strings.HasSuffix(s, "/gmail.send") || strings.HasSuffix(s, "/drive") || strings.HasSuffix(s, "/calendar") {
			t.Errorf("readonly login must not request write scope %s", s)
		}
	}
}

func TestMergeGrantedAccess(t *testing.T) {
	// Replacing login: only the new services remain.
	got := mergeGrantedAccess(nil, []string{"gmail"}, true)
	if len(got) != 1 || got["gmail"] != auth.AccessReadOnly {
		t.Errorf("replace = %v", got)
	}

	existing := map[string]string{"gmail": auth.AccessFull, "drive": auth.AccessReadOnly}

	// Adding read-only never downgrades a service that already has full access.
	got = mergeGrantedAccess(existing, []string{"gmail", "tasks"}, true)
	if got["gmail"] != auth.AccessFull || got["tasks"] != auth.AccessReadOnly || got["drive"] != auth.AccessReadOnly {
		t.Errorf("add readonly = %v", got)
	}

	// Adding full access upgrades a read-only service.
	got = mergeGrantedAccess(existing, []string{"drive"}, false)
	if got["drive"] != auth.AccessFull || got["gmail"] != auth.AccessFull {
		t.Errorf("add full = %v", got)
	}
	if existing["drive"] != auth.AccessReadOnly {
		t.Error("mergeGrantedAccess must not mutate its input")
	}

	if !allReadOnly(map[string]string{"gmail": auth.AccessReadOnly}) || allReadOnly(existing) || allReadOnly(nil) {
		t.Error("allReadOnly gave the wrong answer")
	}
	names := grantedServiceNames(map[string]string{"tasks": "full", "gmail": "full"})
	if strings.Join(names, ",") != "gmail,tasks" {
		t.Errorf("grantedServiceNames order = %v", names)
	}
}

func TestMissingScopes_ComparesScopeByScope(t *testing.T) {
	const pfx = "https://www.googleapis.com/auth/"
	previous := []string{pfx + "userinfo.email", pfx + "gmail.readonly", pfx + "gmail.send", pfx + "drive.readonly"}

	// Everything kept, more added: nothing missing.
	kept := append(append([]string(nil), previous...), pfx+"tasks.readonly")
	if got := missingScopes(previous, kept); len(got) != 0 {
		t.Errorf("missing = %v, want none", got)
	}

	// gmail.send swapped for gmail.modify: both grants are "partial"
	// Gmail, but sending would stop working, so it counts as a loss.
	swapped := []string{pfx + "userinfo.email", pfx + "gmail.readonly", pfx + "gmail.modify", pfx + "drive.readonly"}
	before, _ := deriveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, previous)
	after, _ := deriveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, swapped)
	if before["gmail"] != auth.AccessPartial || after["gmail"] != auth.AccessPartial {
		t.Fatalf("test premise: both should be partial, got %q and %q", before["gmail"], after["gmail"])
	}
	if got := strings.Join(shortScopeNames(missingScopes(previous, swapped)), ","); got != "gmail.send" {
		t.Errorf("missing = %q, want gmail.send", got)
	}

	// No recorded previous scopes (replacing login, or a login from an
	// older version): nothing can be verified, so nothing is reported.
	if got := missingScopes(nil, swapped); len(got) != 0 {
		t.Errorf("missing with unknown previous scopes = %v", got)
	}
	// A scope gws no longer requests is not worth blocking a login over.
	if got := missingScopes([]string{pfx + "some.retired.scope"}, swapped); len(got) != 0 {
		t.Errorf("unknown scopes should be ignored, got %v", got)
	}
}

func TestAddGrantedAccess(t *testing.T) {
	// Used by `auth status` on both the success and the failed-lookup
	// path, so the fields never depend on the user-info call.
	result := map[string]interface{}{"authenticated": true}
	addGrantedAccess(result, map[string]string{"gmail": auth.AccessReadOnly})
	if result["read_only"] != true || result["services"] == nil {
		t.Errorf("read-only login = %v", result)
	}
	result = map[string]interface{}{}
	addGrantedAccess(result, map[string]string{"gmail": auth.AccessReadOnly, "drive": auth.AccessFull})
	if result["read_only"] != false {
		t.Errorf("mixed login = %v", result)
	}
	result = map[string]interface{}{}
	addGrantedAccess(result, nil)
	if len(result) != 0 {
		t.Errorf("no recorded access should add nothing: %v", result)
	}
}

func TestCheckSameAccount(t *testing.T) {
	if err := checkSameAccount("acct-A", "acct-A", nil); err != nil {
		t.Errorf("same account rejected: %v", err)
	}
	// A different account must never replace the stored authorization.
	if err := checkSameAccount("acct-A", "acct-B", nil); err == nil || !strings.Contains(err.Error(), "different Google account") {
		t.Errorf("different account: error = %v", err)
	}
	// Known stored account, unidentified new one: not accepted on trust.
	if err := checkSameAccount("acct-A", "", nil); err == nil {
		t.Error("an unidentified account should be rejected when the stored one is known")
	}
	if err := checkSameAccount("acct-A", "", errors.New("network down")); err == nil || !strings.Contains(err.Error(), "network down") {
		t.Errorf("lookup failure: error = %v", err)
	}
	// Nothing stored to compare against (first login, or unknown).
	if err := checkSameAccount("", "acct-B", nil); err != nil {
		t.Errorf("no stored account: %v", err)
	}
	if err := checkSameAccount("", "", errors.New("x")); err != nil {
		t.Errorf("no stored account and no lookup: %v", err)
	}
}

func TestStoredAccountID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	original := lookupAccountID
	t.Cleanup(func() { lookupAccountID = original })
	lookups := 0
	var lookupErr error
	lookupAccountID = func(*oauth2.Token) (string, error) { lookups++; return "acct-from-google", lookupErr }

	// Nothing stored: no account, no error, no network call.
	if got, err := storedAccountID(); got != "" || err != nil || lookups != 0 {
		t.Errorf("no stored token: got %q, %v after %d lookups", got, err, lookups)
	}

	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
	if err := auth.SaveToken(tok); err != nil {
		t.Fatal(err)
	}
	// Login from an older version: no recorded account, so Google is asked.
	if got, err := storedAccountID(); got != "acct-from-google" || err != nil || lookups != 1 {
		t.Errorf("legacy login: got %q, %v after %d lookups", got, err, lookups)
	}
	// A stored token whose account cannot be determined is an error, not
	// "nothing to compare against" — otherwise --add could switch accounts.
	lookupErr = errors.New("network down")
	if got, err := storedAccountID(); err == nil || got != "" {
		t.Errorf("failed lookup with a stored token: got %q, %v; want an error", got, err)
	}
	lookupErr = nil

	// Recorded account: used directly, no network call.
	before := lookups
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, nil, "acct-recorded", tok); err != nil {
		t.Fatal(err)
	}
	if got, err := storedAccountID(); got != "acct-recorded" || err != nil || lookups != before {
		t.Errorf("recorded account: got %q, %v after %d lookups", got, err, lookups-before)
	}
}

func TestUnrequestedScopes(t *testing.T) {
	const pfx = "https://www.googleapis.com/auth/"
	requested := []string{pfx + "userinfo.email", pfx + "gmail.readonly", pfx + "calendar.readonly"}

	// Exactly what was asked for, plus the identity scopes Google adds.
	withIdentity := append([]string{"openid", "email", "profile", pfx + "userinfo.profile"}, requested...)
	if extra := unrequestedScopes(requested, withIdentity); len(extra) != 0 {
		t.Errorf("extra = %v, want none", extra)
	}
	// Write scopes from an earlier grant coming back must be caught.
	returned := append(append([]string(nil), requested...), pfx+"gmail.send", pfx+"drive")
	if extra := strings.Join(shortScopeNames(unrequestedScopes(requested, returned)), ","); extra != "gmail.send,drive" {
		t.Errorf("extra = %q, want gmail.send,drive", extra)
	}
	// So must a scope gws has never heard of: it could grant access a
	// read-only login was meant to exclude.
	unknown := append(append([]string(nil), requested...), pfx+"cloud-platform", pfx+"some.future.scope")
	if extra := strings.Join(shortScopeNames(unrequestedScopes(requested, unknown)), ","); extra != "cloud-platform,some.future.scope" {
		t.Errorf("extra = %q, want the unknown scopes reported", extra)
	}
	// Fewer scopes than requested is not this check's concern.
	if extra := unrequestedScopes(requested, requested[:1]); len(extra) != 0 {
		t.Errorf("extra = %v", extra)
	}
}

func TestWarnIfRefreshNotSaved(t *testing.T) {
	var out bytes.Buffer
	warnIfRefreshNotSaved(&out, nil)
	if out.Len() != 0 {
		t.Errorf("no warning expected on success: %q", out.String())
	}
	warnIfRefreshNotSaved(&out, errors.New("disk full"))
	if !strings.Contains(out.String(), "could not be saved (disk full)") || !strings.Contains(out.String(), "gws auth login") {
		t.Errorf("warning = %q", out.String())
	}
}

func TestSnapshotGrantedAccess_RestoresPreviousRecord(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldTok := &oauth2.Token{AccessToken: "a", RefreshToken: "old"}
	if err := auth.SaveToken(oldTok); err != nil {
		t.Fatal(err)
	}
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, []string{"s1"}, "acct-A", oldTok); err != nil {
		t.Fatal(err)
	}

	// A login writes the new record, then fails to save its token.
	restore := auth.SnapshotGrantedAccess()
	if err := auth.SaveGrantedAccess(map[string]string{"drive": auth.AccessReadOnly}, []string{"s2"}, "acct-A", &oauth2.Token{RefreshToken: "new"}); err != nil {
		t.Fatal(err)
	}
	if auth.LoadGrantedAccess() != nil {
		t.Fatal("test premise: the new record should not match the old token")
	}
	restore()

	// The still-stored old token has its record back.
	if got := auth.LoadGrantedAccess(); got["gmail"] != auth.AccessFull || len(got) != 1 {
		t.Errorf("restored access = %v", got)
	}
	if got := auth.LoadGrantedScopes(); len(got) != 1 || got[0] != "s1" {
		t.Errorf("restored scopes = %v", got)
	}

	// With no record beforehand, restoring removes the one written.
	if err := os.Remove(config.GetGrantedAccessPath()); err != nil {
		t.Fatal(err)
	}
	restore = auth.SnapshotGrantedAccess()
	_ = auth.SaveGrantedAccess(map[string]string{"drive": auth.AccessFull}, nil, "", &oauth2.Token{RefreshToken: "new"})
	restore()
	if _, err := os.Stat(config.GetGrantedAccessPath()); !os.IsNotExist(err) {
		t.Error("restore should remove a record that did not exist before")
	}
}

func TestUnionScopes(t *testing.T) {
	got := unionScopes([]string{"a", "b"}, []string{"b", "c", "a", "d"})
	if strings.Join(got, ",") != "a,b,c,d" {
		t.Errorf("union = %v", got)
	}
	if got := unionScopes(nil, []string{"x"}); len(got) != 1 || got[0] != "x" {
		t.Errorf("union with nil = %v", got)
	}
}

func TestRedactSecrets_KeepsTokensOutOfErrors(t *testing.T) {
	token := "ya29.a0Af/secret+token=="
	requestURL := "https://oauth2.googleapis.com/tokeninfo?access_token=" + url.QueryEscape(token) + "&alt=json"
	// What net/http returns for a transport failure: it quotes the URL.
	transportErr := &url.Error{Op: "Get", URL: requestURL, Err: errors.New("dial tcp: connection refused")}
	if !strings.Contains(transportErr.Error(), url.QueryEscape(token)) {
		t.Fatal("test premise: the raw error should contain the token")
	}

	got := redactSecrets(transportErr, token, "")
	for _, leaked := range []string{token, url.QueryEscape(token), url.PathEscape(token), "secret"} {
		if strings.Contains(got.Error(), leaked) {
			t.Errorf("redacted error still contains %q: %s", leaked, got)
		}
	}
	if !strings.Contains(got.Error(), "connection refused") || !strings.Contains(got.Error(), "[redacted]") {
		t.Errorf("redacted error lost its cause: %s", got)
	}
	// The original error must not be reachable by unwrapping.
	if errors.Unwrap(got) != nil {
		t.Error("the redacted error must not wrap the original")
	}
	var urlErr *url.Error
	if errors.As(got, &urlErr) {
		t.Error("the original url.Error must not be recoverable")
	}

	// The plain form is redacted too, and nil stays nil.
	if got := redactSecrets(errors.New("bad token "+token), token); strings.Contains(got.Error(), token) {
		t.Errorf("plain token not redacted: %s", got)
	}
	if redactSecrets(nil, token) != nil {
		t.Error("nil error should stay nil")
	}
}

// A failed scope lookup surfaces through `auth login --add`; whatever the
// lookup returns, the token must not appear in what the command prints.
func TestStoredScopes_ErrorDoesNotLeakToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	secret := "ya29.super-secret-access-token"
	if err := auth.SaveToken(&oauth2.Token{AccessToken: secret, RefreshToken: "1//refresh-secret"}); err != nil {
		t.Fatal(err)
	}
	original := lookupTokenScopes
	t.Cleanup(func() { lookupTokenScopes = original })
	lookupTokenScopes = func(tok *oauth2.Token) ([]string, error) {
		raw := &url.Error{Op: "Get", URL: "https://oauth2.googleapis.com/tokeninfo?access_token=" + url.QueryEscape(tok.AccessToken), Err: errors.New("timeout")}
		return nil, redactSecrets(raw, tok.AccessToken, tok.RefreshToken)
	}
	_, err := storedScopes()
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "refresh-secret") {
		t.Errorf("error leaks a token: %s", err)
	}
}

func TestStoredScopes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	original := lookupTokenScopes
	t.Cleanup(func() { lookupTokenScopes = original })
	lookups := 0
	var reported []string
	var lookupErr error
	lookupTokenScopes = func(*oauth2.Token) ([]string, error) { lookups++; return reported, lookupErr }

	// Nothing stored: nothing to preserve, no network call.
	if got, err := storedScopes(); got != nil || err != nil || lookups != 0 {
		t.Errorf("no stored token: %v, %v after %d lookups", got, err, lookups)
	}

	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
	if err := auth.SaveToken(tok); err != nil {
		t.Fatal(err)
	}
	// A login from an older version has no recorded list, so Google is
	// asked what the token really holds — here, Gmail without gmail.send,
	// which the user declined back then. Nothing wider is assumed.
	const pfx = "https://www.googleapis.com/auth/"
	reported = []string{pfx + "userinfo.email", pfx + "gmail.readonly", pfx + "gmail.modify"}
	got, err := storedScopes()
	if err != nil || strings.Join(shortScopeNames(got), ",") != "userinfo.email,gmail.readonly,gmail.modify" {
		t.Fatalf("legacy scopes = %v, %v", got, err)
	}
	request := unionScopes(got, auth.ScopesForServices([]string{"tasks"}))
	if containsStr(request, pfx+"gmail.send") {
		t.Errorf("--add must not ask again for a scope the user declined: %v", request)
	}
	if !containsStr(request, pfx+"tasks") || !containsStr(request, pfx+"gmail.modify") {
		t.Errorf("--add request should keep held scopes and add the new service: %v", request)
	}

	// The scopes cannot be determined: an error, never a guess.
	lookupErr = errors.New("network down")
	if got, err := storedScopes(); err == nil || got != nil {
		t.Errorf("failed lookup: %v, %v; want an error", got, err)
	}
	lookupErr, reported = nil, nil
	if _, err := storedScopes(); err == nil {
		t.Error("an empty scope report should be an error")
	}

	// A recorded list is used directly, without asking Google.
	before := lookups
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, []string{"recorded-scope"}, "acct", tok); err != nil {
		t.Fatal(err)
	}
	if got, err := storedScopes(); err != nil || len(got) != 1 || got[0] != "recorded-scope" || lookups != before {
		t.Errorf("recorded scopes = %v, %v after %d lookups", got, err, lookups-before)
	}
}

// Logout holds the same lock as login, so it cannot delete metadata while
// a login is writing it: with the lock held elsewhere, logout must wait
// and then fail rather than proceed.
func TestLogout_WaitsForTheLoginLock(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the lock timeout")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, nil, "acct", tok); err != nil {
		t.Fatal(err)
	}

	unlock, err := auth.LockLoginState()
	if err != nil {
		t.Fatal(err)
	}
	logoutErr := captureStderrStr(t, func() {
		if err := runLogout(logoutCmd, nil); err == nil {
			t.Error("logout should not proceed while a login holds the lock")
		}
	})
	if !strings.Contains(logoutErr, "login appears to be in progress") {
		t.Errorf("logout error output = %q", logoutErr)
	}
	// Nothing was deleted behind the login's back.
	if _, err := os.Stat(config.GetGrantedAccessPath()); err != nil {
		t.Errorf("metadata should be untouched while the lock is held: %v", err)
	}

	// Once the login is done, logout goes through and clears it.
	unlock()
	if _, err := captureStdout(t, func() error { return runLogout(logoutCmd, nil) }); err != nil {
		t.Fatalf("logout after release: %v", err)
	}
	if _, err := os.Stat(config.GetGrantedAccessPath()); !os.IsNotExist(err) {
		t.Error("logout should delete the metadata once it holds the lock")
	}
}

func TestLockLoginState_CanBeReacquiredAfterRelease(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	unlock, err := auth.LockLoginState()
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	// Released locks can be taken again.
	unlock()
	unlock2, err := auth.LockLoginState()
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	unlock2()
}

func TestMergeGrantedAccess_ReadOnlyAddKeepsPartialWriteAccess(t *testing.T) {
	const pfx = "https://www.googleapis.com/auth/"
	existing := map[string]string{"gmail": auth.AccessPartial, "chat": auth.AccessPartialReadOnly}
	got := mergeGrantedAccess(existing, []string{"gmail", "chat"}, true)
	if got["gmail"] != auth.AccessPartial {
		t.Errorf("gmail = %q, want partial kept", got["gmail"])
	}
	if got["chat"] != auth.AccessReadOnly {
		t.Errorf("chat = %q, want readonly", got["chat"])
	}
	// The consent request keeps the write scopes the token already holds.
	previous := []string{pfx + "userinfo.email", pfx + "gmail.readonly", pfx + "gmail.send"}
	request := unionScopes(previous, auth.ReadOnlyScopesForServices([]string{"gmail"}))
	if !containsStr(request, pfx+"gmail.send") {
		t.Errorf("request dropped an existing write scope: %v", request)
	}
}

func TestDeriveGrantedAccess(t *testing.T) {
	const pfx = "https://www.googleapis.com/auth/"
	granted := append(auth.ScopesForServices([]string{"gmail"}), // gmail: everything
		pfx+"drive.readonly",    // drive: read-only only, although full was requested
		pfx+"tasks.readonly",    // tasks: read-only as requested
		pfx+"calendar.readonly", // calendar: read-only scope only
	)
	requested := map[string]string{
		"gmail":    auth.AccessFull,
		"drive":    auth.AccessFull,
		"tasks":    auth.AccessReadOnly,
		"calendar": auth.AccessReadOnly,
		"docs":     auth.AccessFull,     // nothing granted
		"sheets":   auth.AccessReadOnly, // nothing granted
	}

	access, notGranted := deriveGrantedAccess(requested, granted)

	want := map[string]string{"gmail": auth.AccessFull, "drive": auth.AccessReadOnly, "tasks": auth.AccessReadOnly, "calendar": auth.AccessReadOnly}
	if len(access) != len(want) {
		t.Fatalf("access = %v, want %v", access, want)
	}
	for svc, level := range want {
		if access[svc] != level {
			t.Errorf("access[%s] = %q, want %q", svc, access[svc], level)
		}
	}
	if got := strings.Join(notGranted, "; "); got != "drive (write access); docs; sheets" {
		t.Errorf("notGranted = %q", got)
	}

	// An incremental read-only login can return write scopes granted
	// earlier; with no local metadata the level must still be full.
	prior := append(auth.ScopesForServices([]string{"drive"}), auth.ReadOnlyScopesForServices([]string{"drive", "sheets"})...)
	up, missing := deriveGrantedAccess(map[string]string{"drive": auth.AccessReadOnly, "sheets": auth.AccessReadOnly}, prior)
	if up["drive"] != auth.AccessFull || up["sheets"] != auth.AccessReadOnly || len(missing) != 0 {
		t.Errorf("readonly request with prior write scopes = %v (missing %v), want drive full", up, missing)
	}

	// Write scopes for a service that was never requested and is unknown
	// to local metadata (e.g. the local token was deleted without revoking
	// the grant) must still be reported: the token is not read-only.
	leftover := append(auth.ReadOnlyScopesForServices([]string{"tasks"}), auth.ScopesForServices([]string{"gmail"})...)
	seen, missing := deriveGrantedAccess(map[string]string{"tasks": auth.AccessReadOnly}, leftover)
	if seen["gmail"] != auth.AccessFull || seen["tasks"] != auth.AccessReadOnly || len(missing) != 0 {
		t.Errorf("unrequested write scopes = %v (missing %v), want gmail full", seen, missing)
	}
	if allReadOnly(seen) {
		t.Error("a token carrying gmail write scopes must not be reported read-only")
	}

	// A partial grant that includes a write scope (gmail.readonly +
	// gmail.send, the rest unticked) is never reported as read-only.
	partial, missing := deriveGrantedAccess(map[string]string{"gmail": auth.AccessFull, "tasks": auth.AccessReadOnly},
		[]string{pfx + "userinfo.email", pfx + "gmail.readonly", pfx + "gmail.send", pfx + "tasks.readonly"})
	if partial["gmail"] != auth.AccessPartial || partial["tasks"] != auth.AccessReadOnly {
		t.Errorf("partial grant = %v", partial)
	}
	if allReadOnly(partial) {
		t.Error("a token that can send mail must not be reported read-only")
	}
	if strings.Join(missing, "; ") != "gmail (only some scopes)" {
		t.Errorf("missing = %v", missing)
	}
	// A write scope alone (read-only scope unticked) is still partial.
	writeOnly, _ := deriveGrantedAccess(map[string]string{"gmail": auth.AccessReadOnly}, []string{pfx + "userinfo.email", pfx + "gmail.send"})
	if writeOnly["gmail"] != auth.AccessPartial {
		t.Errorf("write-only grant = %v", writeOnly)
	}
	// Full contacts access does not surface a spurious partial `people`.
	contacts, _ := deriveGrantedAccess(map[string]string{"contacts": auth.AccessFull}, auth.ScopesForServices([]string{"contacts"}))
	if _, ok := contacts["people"]; ok || contacts["contacts"] != auth.AccessFull {
		t.Errorf("contacts grant = %v", contacts)
	}

	// Some read scopes unticked, no write scope: the service is kept as
	// partial-readonly so the scope hint still has something to go on,
	// and the login as a whole still counts as read-only.
	chatScopes := []string{pfx + "userinfo.email", pfx + "chat.messages.readonly", pfx + "tasks.readonly"}
	chat, missing := deriveGrantedAccess(map[string]string{"chat": auth.AccessReadOnly, "tasks": auth.AccessReadOnly}, chatScopes)
	if chat["chat"] != auth.AccessPartialReadOnly || strings.Join(missing, "; ") != "chat (only some scopes)" {
		t.Errorf("partial read grant = %v (missing %v)", chat, missing)
	}
	if !allReadOnly(chat) {
		t.Error("a login with only read scopes is read-only")
	}
	// ...but an unrequested service sharing those read scopes is not
	// invented (events shares chat's read scopes).
	if _, ok := chat["events"]; ok {
		t.Errorf("unrequested events should not appear: %v", chat)
	}
	chatMembers, _, _ := rootCmd.Find([]string{"chat", "members"})
	insufficientErr := errors.New("googleapi: Error 403: Request had insufficient authentication scopes.")
	if hint := readOnlyScopeHint(chatMembers, insufficientErr, chat); !strings.Contains(hint, "--add --services chat") {
		t.Errorf("partial-readonly hint = %q", hint)
	}

	// Sheets has exactly one scope in each set. A previously granted
	// `spreadsheets` scope with no local metadata is full, not partial.
	sheets, _ := deriveGrantedAccess(map[string]string{"tasks": auth.AccessReadOnly},
		[]string{pfx + "userinfo.email", pfx + "spreadsheets", pfx + "tasks.readonly"})
	if sheets["sheets"] != auth.AccessFull {
		t.Errorf("sheets with the spreadsheets scope = %q, want full", sheets["sheets"])
	}
	sheetsRO, _ := deriveGrantedAccess(nil, []string{pfx + "userinfo.email", pfx + "spreadsheets.readonly"})
	if sheetsRO["sheets"] != auth.AccessReadOnly {
		t.Errorf("sheets with spreadsheets.readonly = %q, want readonly", sheetsRO["sheets"])
	}

	// Groups has only read-only scopes: it is read-only whatever was
	// requested, so a groups-only login reports read_only: true.
	for _, requestedLevel := range []string{auth.AccessFull, auth.AccessReadOnly} {
		groups, missing := deriveGrantedAccess(map[string]string{"groups": requestedLevel}, auth.ScopesForServices([]string{"groups"}))
		if groups["groups"] != auth.AccessReadOnly || len(missing) != 0 || !allReadOnly(groups) {
			t.Errorf("groups requested %s = %v (missing %v), want readonly with nothing missing", requestedLevel, groups, missing)
		}
	}
}

// --add refreshes the stored login once, before it looks anything up, and
// stops there if that fails — the stored login is never left half-used.
func TestAuthLogin_AddStopsIfTheStoredLoginCannotBeRefreshed(t *testing.T) {
	viper.Set("client_id", "id")
	viper.Set("client_secret", "secret")
	_ = loginCmd.Flags().Set("add", "true")
	_ = loginCmd.Flags().Set("services", "tasks")
	originalRefresh, originalAccount, originalScopes := refreshStoredToken, lookupAccountID, lookupTokenScopes
	t.Cleanup(func() {
		viper.Set("client_id", "")
		viper.Set("client_secret", "")
		_ = loginCmd.Flags().Set("add", "false")
		loginCmd.Flags().Lookup("add").Changed = false
		_ = loginCmd.Flags().Set("services", "")
		loginCmd.Flags().Lookup("services").Changed = false
		refreshStoredToken, lookupAccountID, lookupTokenScopes = originalRefresh, originalAccount, originalScopes
	})

	var order []string
	refreshStoredToken = func() error { order = append(order, "refresh"); return errors.New("invalid_grant") }
	lookupAccountID = func(*oauth2.Token) (string, error) { order = append(order, "account"); return "acct", nil }
	lookupTokenScopes = func(*oauth2.Token) ([]string, error) { order = append(order, "scopes"); return nil, nil }

	var err error
	out := captureStderrStr(t, func() { err = runLogin(loginCmd, nil) })
	if err == nil || !strings.Contains(out, "could not refresh the stored authorization") {
		t.Fatalf("error = %v, output = %q", err, out)
	}
	// The refresh came first and nothing was looked up after it failed.
	if strings.Join(order, ",") != "refresh" {
		t.Errorf("call order = %v, want only the refresh", order)
	}
}

func TestAuthLogin_AddRequiresServices(t *testing.T) {
	viper.Set("client_id", "id")
	viper.Set("client_secret", "secret")
	viper.Set("services", nil)
	_ = loginCmd.Flags().Set("add", "true")
	t.Cleanup(func() {
		viper.Set("client_id", "")
		viper.Set("client_secret", "")
		_ = loginCmd.Flags().Set("add", "false")
		loginCmd.Flags().Lookup("add").Changed = false
	})

	// Fails before any browser flow starts — with the flag absent, empty,
	// or made only of blanks (Cobra marks all but the first as "changed").
	t.Cleanup(func() {
		_ = loginCmd.Flags().Set("services", "")
		loginCmd.Flags().Lookup("services").Changed = false
	})
	for _, value := range []string{"(unset)", "", " ", ",", " , "} {
		if value != "(unset)" {
			_ = loginCmd.Flags().Set("services", value)
		}
		err := runLogin(loginCmd, nil)
		want := "names no service"
		if value == "(unset)" {
			want = "--add requires --services"
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("--services %q: error = %v, want %q", value, err, want)
		}
	}

	// Without --add, an explicit but empty --services is rejected too,
	// instead of silently becoming "every service".
	_ = loginCmd.Flags().Set("add", "false")
	for _, value := range []string{"", " , "} {
		_ = loginCmd.Flags().Set("services", value)
		err := runLogin(loginCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "names no service") {
			t.Fatalf("--services %q: error = %v, want names-no-service", value, err)
		}
	}
	_ = loginCmd.Flags().Set("add", "true")

	// And an empty --services never counts as a scoped selection.
	if hasServiceSelection(loginCmd) {
		t.Error("blank --services must not count as a selection")
	}
	_ = loginCmd.Flags().Set("services", "gmail, ,drive")
	if got := strings.Join(resolveServices(loginCmd), ","); got != "gmail,drive" || !hasServiceSelection(loginCmd) {
		t.Errorf("resolveServices = %q", got)
	}
}

func TestAuthLoginCommand_HasReadonlyAndAddFlags(t *testing.T) {
	for _, name := range []string{"readonly", "add", "services"} {
		if loginCmd.Flags().Lookup(name) == nil {
			t.Errorf("auth login missing --%s", name)
		}
	}
	for _, name := range []string{"access-token", "yes", "schema", "show-scopes"} {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("root missing global --%s", name)
		}
	}
}

// --- external access token ---

func TestAuthStatus_ReportsAccessTokenWithoutPrintingIt(t *testing.T) {
	viper.Set("access_token", "ya29.secret-token-value")
	t.Cleanup(func() { viper.Set("access_token", "") })

	out, err := captureStdout(t, func() error { return runStatus(statusCmd, nil) })
	if err != nil {
		t.Fatalf("runStatus error: %v", err)
	}
	if !strings.Contains(out, `"auth_mode": "access-token"`) {
		t.Errorf("status should report access-token mode: %s", out)
	}
	if strings.Contains(out, "ya29.secret-token-value") {
		t.Errorf("status must never print the token: %s", out)
	}
}

func TestCurrentAuthMode_AccessTokenOverridesServiceAccount(t *testing.T) {
	setAuthModeConfig(t, "service-account-app", "/tmp/key.json", "")
	viper.Set("access_token", "tok")
	t.Cleanup(func() { viper.Set("access_token", "") })
	if got := currentAuthMode(); got != auth.AuthModeOAuth {
		t.Errorf("currentAuthMode = %q, want %q", got, auth.AuthModeOAuth)
	}
}

func toStrings(v interface{}) []string {
	items, _ := v.([]interface{})
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
