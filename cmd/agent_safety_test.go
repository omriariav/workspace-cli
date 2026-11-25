package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/auth"
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
	if result["login_readonly"] != "gws auth login --services gmail --readonly" {
		t.Errorf("login_readonly = %v", result["login_readonly"])
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
	old := &oauth2.Token{AccessToken: "old", RefreshToken: "old-refresh"}

	// Replacing login without a refresh token must not inherit the old grant.
	if _, err := loginTokenToStore(old, &oauth2.Token{AccessToken: "new"}, false); err == nil {
		t.Error("replacing login without a refresh token should fail")
	}
	got, err := loginTokenToStore(old, &oauth2.Token{AccessToken: "new", RefreshToken: "new-refresh"}, false)
	if err != nil || got.RefreshToken != "new-refresh" {
		t.Errorf("replacing login = %+v, %v", got, err)
	}
	// Incremental login may keep the previous refresh token.
	got, err = loginTokenToStore(old, &oauth2.Token{AccessToken: "new"}, true)
	if err != nil || got.AccessToken != "new" || got.RefreshToken != "old-refresh" {
		t.Errorf("incremental login = %+v, %v", got, err)
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

	// A read-only request is never reported as full, even when the
	// service's full and read-only scope sets coincide.
	groups, _ := deriveGrantedAccess(map[string]string{"groups": auth.AccessReadOnly}, auth.ScopesForServices([]string{"groups"}))
	if groups["groups"] != auth.AccessReadOnly {
		t.Errorf("groups = %q, want readonly", groups["groups"])
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
