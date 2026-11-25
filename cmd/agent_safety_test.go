package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
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
	if !containsStr(toStrings(activity["scopes"]), "https://www.googleapis.com/auth/drive.activity.readonly") {
		t.Errorf("drive activity should include the Drive Activity scope: %v", activity["scopes"])
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
		{"drive", "start-approval"}, {"drive", "reassign-approval"},
		{"calendar", "unsubscribe"},
		{"docs", "replace-content"},
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

func TestConditionalSensitive_RulesResolveAndTrigger(t *testing.T) {
	values := map[string]string{"trashed": "true", "members": "users/1", "attendees": "a@example.com", "add-attendees": "a@example.com", "message": "running late"}
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

func TestScopesForAccess_RequestsTheWholeMergedGrant(t *testing.T) {
	const pfx = "https://www.googleapis.com/auth/"
	// Existing: gmail full, drive read-only. Adding tasks read-only must
	// re-request all three, because an incremental grant cannot be relied
	// on to carry earlier scopes over.
	access := mergeGrantedAccess(map[string]string{"gmail": auth.AccessFull, "drive": auth.AccessReadOnly}, []string{"tasks"}, true)
	scopes := scopesForAccess(access)

	for _, want := range []string{"userinfo.email", "gmail.send", "gmail.readonly", "drive.readonly", "tasks.readonly"} {
		if !containsStr(scopes, pfx+want) {
			t.Errorf("merged request is missing %s: %v", want, scopes)
		}
	}
	for _, unwanted := range []string{"drive", "drive.file", "tasks"} {
		if containsStr(scopes, pfx+unwanted) {
			t.Errorf("merged request must not widen to %s: %v", unwanted, scopes)
		}
	}
	seen := map[string]bool{}
	for _, s := range scopes {
		if seen[s] {
			t.Errorf("duplicate scope %s", s)
		}
		seen[s] = true
	}
	// A partial grant is re-requested in full; a partial read grant read-only.
	partial := scopesForAccess(map[string]string{"gmail": auth.AccessPartial, "chat": auth.AccessPartialReadOnly})
	if !containsStr(partial, pfx+"gmail.modify") || containsStr(partial, pfx+"chat.messages") || !containsStr(partial, pfx+"chat.messages.readonly") {
		t.Errorf("partial levels = %v", partial)
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
	lookupAccountID = func(*oauth2.Token) (string, error) { lookups++; return "acct-from-google", nil }

	if got := storedAccountID(); got != "" || lookups != 0 {
		t.Errorf("no stored token: got %q after %d lookups", got, lookups)
	}

	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
	if err := auth.SaveToken(tok); err != nil {
		t.Fatal(err)
	}
	// Login from an older version: no recorded account, so Google is asked.
	if got := storedAccountID(); got != "acct-from-google" || lookups != 1 {
		t.Errorf("legacy login: got %q after %d lookups", got, lookups)
	}
	// Recorded account: used directly, no network call.
	if err := auth.SaveGrantedAccess(map[string]string{"gmail": auth.AccessFull}, nil, "acct-recorded", tok); err != nil {
		t.Fatal(err)
	}
	if got := storedAccountID(); got != "acct-recorded" || lookups != 1 {
		t.Errorf("recorded account: got %q after %d lookups", got, lookups)
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

// A login from an older version recorded every service as granted in full.
// Re-deriving labels from the scopes Google returns relabels groups and
// driveactivity as read-only; that must not be mistaken for lost access.
func TestAddOnLegacyLogin_IsNotRejected(t *testing.T) {
	legacy := map[string]string{}
	for _, svc := range auth.ValidServiceNames() {
		legacy[svc] = auth.AccessFull
	}
	// No recorded scope list for a legacy login.
	var previousScopes []string
	carried := scopesForAccess(legacy)
	requested := unionScopes(carried, auth.ScopesForServices([]string{"tasks"}))

	// Google returns exactly what was requested.
	if lost := missingScopes(previousScopes, requested); len(lost) != 0 {
		t.Fatalf("legacy --add rejected for %v", lost)
	}
	access, notGranted := deriveGrantedAccess(mergeGrantedAccess(legacy, []string{"tasks"}, false), requested)
	if access["groups"] != auth.AccessReadOnly || access["gmail"] != auth.AccessFull || len(notGranted) != 0 {
		t.Errorf("access = %v, notGranted = %v", access, notGranted)
	}
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
