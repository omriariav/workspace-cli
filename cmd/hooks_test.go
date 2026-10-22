package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pluginhooks "github.com/omriariav/workspace-cli/internal/hooks"
	"github.com/spf13/cobra"
)

func TestConfigureHookPatchSetsIndependentProviderAndSourceStates(t *testing.T) {
	cmd := newHookConfigureTestCommand()
	if err := cmd.Flags().Parse([]string{
		"--enabled=true", "--time=false", "--workspace=true",
		"--calendar=false", "--gmail=false", "--sent-mail=true", "--tasks=false", "--chat=true",
	}); err != nil {
		t.Fatal(err)
	}
	patch, err := configureHookPatch(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if patch["enabled"] != true {
		t.Fatalf("global enabled state missing: %#v", patch)
	}
	providers := patch["providers"].(map[string]any)
	if providers["time"].(map[string]any)["enabled"] != false || providers["workspace"].(map[string]any)["enabled"] != true {
		t.Fatalf("provider states were not independently configurable: %#v", providers)
	}
	sources := patch["sources"].(map[string]any)
	for source, want := range map[string]bool{"calendar": false, "gmail": false, "sent_mail": true, "tasks": false, "chat": true} {
		if got := sources[source].(map[string]any)["enabled"]; got != want {
			t.Errorf("%s enabled=%v, want %v", source, got, want)
		}
	}
}

func TestConfigureHookPatchRejectsNonpositiveLimits(t *testing.T) {
	cmd := newHookConfigureTestCommand()
	if err := cmd.Flags().Parse([]string{"--task-limit=0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := configureHookPatch(cmd); err == nil || !strings.Contains(err.Error(), "--task-limit must be positive") {
		t.Fatalf("expected positive-limit validation, got %v", err)
	}
}

func TestWorkspaceScopeRejectsConsentPatches(t *testing.T) {
	for name, patch := range map[string]map[string]any{
		"global hook gate":   {"enabled": true},
		"time provider":      {"providers": map[string]any{"time": map[string]any{"enabled": true}}},
		"workspace provider": {"providers": map[string]any{"workspace": map[string]any{"enabled": true}}},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("scope", "workspace", "")
			cmd.Flags().Bool("apply", false, "")
			err := previewHookUpdate(cmd, patch)
			if err == nil || !strings.Contains(err.Error(), "invalid workspace hook configuration") {
				t.Fatalf("workspace scope accepted consent patch %#v: %v", patch, err)
			}
		})
	}
}

func TestHookConsentCommandsRequireUserScope(t *testing.T) {
	enable := &cobra.Command{}
	enable.Flags().String("scope", "workspace", "")
	enable.Flags().Bool("time", true, "")
	enable.Flags().Bool("workspace", false, "")
	if err := runHooksEnable(enable, nil); err == nil || !strings.Contains(err.Error(), "user-scope only") {
		t.Fatalf("workspace-scoped enable was not rejected: %v", err)
	}

	disable := &cobra.Command{}
	disable.Flags().String("scope", "workspace", "")
	if err := runHooksDisable(disable, nil); err == nil || !strings.Contains(err.Error(), "user-scope only") {
		t.Fatalf("workspace-scoped disable was not rejected: %v", err)
	}

	configure := newHookConfigureTestCommand()
	configure.Flags().String("scope", "user", "")
	if err := configure.Flags().Parse([]string{"--scope=workspace", "--enabled=true"}); err != nil {
		t.Fatal(err)
	}
	if err := runHooksConfigure(configure, nil); err == nil || !strings.Contains(err.Error(), "user-scope only") {
		t.Fatalf("workspace-scoped provider consent was not rejected: %v", err)
	}
}

func TestHookHealthReportNeverContainsWorkspaceContext(t *testing.T) {
	secretContext := "Calendar subject and message body must never appear here"
	report := hookHealthReport(pluginhooks.DefaultConfig(), pluginhooks.RenderedContext{
		AdditionalContext: secretContext,
		ResolvedTimezone:  "Asia/Jerusalem",
		Refreshed:         true,
		SnapshotUpdatedAt: time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC),
		Sources: []pluginhooks.SourceStatus{{
			Name: "Calendar", Available: true,
		}},
	})
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secretContext) || strings.Contains(string(encoded), "additionalContext") {
		t.Fatalf("health report exposed rendered Workspace context: %s", encoded)
	}
	if !strings.Contains(string(encoded), "resolved_timezone") || !strings.Contains(string(encoded), "source_status") || !strings.Contains(string(encoded), "source_counts") {
		t.Fatalf("health report omitted safe metadata: %s", encoded)
	}
}

func TestHookHealthReportReflectsAuthAndSourceOutcomes(t *testing.T) {
	workspace := pluginhooks.DefaultConfig()
	workspace.Enabled = true
	workspace.Providers.Workspace.Enabled = true
	workspace.Sources.Gmail.Enabled = false
	workspace.Sources.SentMail.Enabled = false
	workspace.Sources.Tasks.Enabled = false
	workspace.Sources.Chat.Enabled = false

	missingAuth := hookHealthReportWithToken(workspace, pluginhooks.RenderedContext{}, false)
	if missingAuth["healthy"] != false || missingAuth["auth"].(map[string]any)["ready"] != false {
		t.Fatalf("missing Workspace auth must be unhealthy: %#v", missingAuth)
	}

	allFailed := hookHealthReportWithToken(workspace, pluginhooks.RenderedContext{
		Sources: []pluginhooks.SourceStatus{{Name: "Calendar", Diagnostic: "auth unavailable"}},
	}, true)
	if allFailed["healthy"] != false {
		t.Fatalf("all configured sources failing must be unhealthy: %#v", allFailed)
	}

	degradedConfig := workspace
	degradedConfig.Sources.Gmail.Enabled = true
	degraded := hookHealthReportWithToken(degradedConfig, pluginhooks.RenderedContext{
		Sources: []pluginhooks.SourceStatus{
			{Name: "Calendar", Available: true},
			{Name: "Gmail", Diagnostic: "temporary failure"},
		},
	}, true)
	if degraded["healthy"] != false {
		t.Fatalf("a degraded configured source set must be unhealthy: %#v", degraded)
	}

	allAvailable := hookHealthReportWithToken(workspace, pluginhooks.RenderedContext{
		Sources: []pluginhooks.SourceStatus{{Name: "Calendar", Available: true}},
	}, true)
	if allAvailable["healthy"] != true {
		t.Fatalf("ready auth and all configured sources available must be healthy: %#v", allAvailable)
	}

	retainedDegraded := hookHealthReportWithToken(workspace, pluginhooks.RenderedContext{
		WorkspaceDegraded: true,
		Sources:           []pluginhooks.SourceStatus{{Name: "Calendar", Available: true}},
	}, true)
	if retainedDegraded["healthy"] != false || retainedDegraded["workspace_degraded"] != true {
		t.Fatalf("a retained degraded refresh must be unhealthy despite stale available sources: %#v", retainedDegraded)
	}

	timeOnly := pluginhooks.DefaultConfig()
	timeOnly.Enabled = true
	timeOnly.Providers.Time.Enabled = true
	if report := hookHealthReportWithToken(timeOnly, pluginhooks.RenderedContext{}, false); report["healthy"] != true {
		t.Fatalf("time-only hooks must remain healthy without Workspace auth: %#v", report)
	}
}

func TestHooksStatusUsesStandardFormatsAndQuiet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runStatus := func() error { return runHooksStatus(&cobra.Command{}, nil) }
	if output := captureHookCommandOutput(t, "text", false, runStatus); !strings.Contains(output, "enabled:") || strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("text format was not applied: %q", output)
	}
	if output := captureHookCommandOutput(t, "yaml", false, runStatus); !strings.Contains(output, "enabled: false") {
		t.Fatalf("yaml format was not applied: %q", output)
	}
	if output := captureHookCommandOutput(t, "json", true, runStatus); output != "" {
		t.Fatalf("quiet output must be empty, got %q", output)
	}
}

func TestHooksCommandsRejectPositionalArguments(t *testing.T) {
	commands := []*cobra.Command{hooksCmd, hooksStatusCmd, hooksEnableCmd, hooksConfigureCmd, hooksDisableCmd, hooksTestCmd, hooksRunCmd}
	for _, command := range commands {
		t.Run(command.Name(), func(t *testing.T) {
			if err := command.Args(command, []string{"unexpected"}); err == nil {
				t.Fatalf("%s accepted a positional argument", command.CommandPath())
			}
		})
	}
}

func TestHookAdaptersAreSilentAndNonblockingWithoutWorkingGWS(t *testing.T) {
	paths := []string{
		filepath.Join("..", "hooks", "claude-hook.sh"),
		filepath.Join("..", "plugins", "gws", "hooks", "codex-hook.sh"),
	}
	for _, script := range paths {
		t.Run(filepath.Base(filepath.Dir(script)), func(t *testing.T) {
			for _, withFailingGWS := range []bool{false, true} {
				t.Run(map[bool]string{false: "missing", true: "failing"}[withFailingGWS], func(t *testing.T) {
					binDir := t.TempDir()
					if withFailingGWS {
						fake := filepath.Join(binDir, "gws")
						if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
							t.Fatal(err)
						}
					}
					command := exec.Command("/bin/sh", script, "SessionStart")
					command.Env = []string{"PATH=" + binDir + ":/usr/bin:/bin"}
					output, err := command.CombinedOutput()
					if err != nil {
						t.Fatalf("adapter must exit zero when gws is unavailable: %v\n%s", err, output)
					}
					if len(output) != 0 {
						t.Fatalf("adapter must be silent when gws is unavailable: %q", output)
					}
				})
			}
		})
	}
}

func TestHookRunIsSilentForInvalidHostInvocation(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("event", "UnexpectedEvent", "")
	cmd.Flags().String("adapter", "codex", "")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := runHooksRun(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid hook invocation must not emit a diagnostic envelope: %q", output.String())
	}
}

func captureHookCommandOutput(t *testing.T, outputFormat string, quietValue bool, run func() error) string {
	t.Helper()
	formatFlag := rootCmd.PersistentFlags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("root --format flag is not registered")
	}
	oldFormat, oldQuiet, oldStdout := formatFlag.Value.String(), quiet, os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := rootCmd.PersistentFlags().Set("format", outputFormat); err != nil {
		t.Fatal(err)
	}
	quiet, os.Stdout = quietValue, writer
	defer func() {
		_ = rootCmd.PersistentFlags().Set("format", oldFormat)
		quiet, os.Stdout = oldQuiet, oldStdout
		_ = reader.Close()
		_ = writer.Close()
	}()
	if err := run(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func newHookConfigureTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	flags := cmd.Flags()
	flags.Bool("enabled", false, "")
	flags.Bool("time", false, "")
	flags.Bool("workspace", false, "")
	flags.Int("refresh-minutes", 0, "")
	flags.Bool("calendar", false, "")
	flags.Int("calendar-lookahead-days", 0, "")
	flags.Int("calendar-limit", 0, "")
	flags.Bool("gmail", false, "")
	flags.Int("gmail-limit", 0, "")
	flags.Bool("sent-mail", false, "")
	flags.Int("sent-limit", 0, "")
	flags.Int("sent-lookback-days", 0, "")
	flags.Bool("tasks", false, "")
	flags.StringSlice("task-list", nil, "")
	flags.Int("task-discovery-limit", 0, "")
	flags.Int("task-limit", 0, "")
	flags.Bool("chat", false, "")
	flags.StringSlice("chat-space", nil, "")
	flags.Int("chat-limit", 0, "")
	return cmd
}

// isolateHookEnv points every path hook helpers consult at temp dirs so tests
// never read or write the real home directory or gws config.
func isolateHookEnv(t *testing.T) (configHome, pluginData string) {
	t.Helper()
	home := t.TempDir()
	configHome = filepath.Join(home, "xdg")
	pluginData = filepath.Join(home, "plugin-data")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("PLUGIN_DATA", "")
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	return configHome, pluginData
}

func TestHookWorkspaceKeyCleansAndResolvesSymlinks(t *testing.T) {
	root := t.TempDir()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(resolvedRoot, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(resolvedRoot, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if got := hookWorkspaceKey(link); got != real {
		t.Errorf("symlink key = %q, want %q", got, real)
	}
	if got := hookWorkspaceKey(real + string(filepath.Separator)); got != real {
		t.Errorf("trailing separator key = %q, want %q", got, real)
	}
	if got := hookWorkspaceKey(filepath.Join(real, "sub", "..")); got != real {
		t.Errorf("dotdot key = %q, want %q", got, real)
	}

	missing := filepath.Join(resolvedRoot, "does-not-exist")
	if got := hookWorkspaceKey(missing); got != missing {
		t.Errorf("missing path key = %q, want %q", got, missing)
	}
}

func TestHookWorkspaceKeyMakesRelativePathsAbsolute(t *testing.T) {
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if got := hookWorkspaceKey("."); got != resolved {
		t.Errorf("relative key = %q, want %q", got, resolved)
	}
}

func TestHookConfigPathScopes(t *testing.T) {
	configHome, _ := isolateHookEnv(t)
	cwd := t.TempDir()
	t.Chdir(cwd)

	userPath, err := hookConfigPath("user")
	if err != nil {
		t.Fatal(err)
	}
	if want := pluginhooks.UserConfigPath(filepath.Join(configHome, "gws")); userPath != want {
		t.Errorf("user path = %q, want %q", userPath, want)
	}

	workspacePath, err := hookConfigPath("workspace")
	if err != nil {
		t.Fatal(err)
	}
	gotCwd, _ := os.Getwd()
	if want := pluginhooks.WorkspaceConfigPath(gotCwd); workspacePath != want {
		t.Errorf("workspace path = %q, want %q", workspacePath, want)
	}
	if userPath == workspacePath {
		t.Error("user and workspace config paths must differ")
	}

	for _, scope := range []string{"", "global", "USER"} {
		path, err := hookConfigPath(scope)
		if err == nil || path != "" {
			t.Errorf("scope %q: got (%q, %v), want empty path and error", scope, path, err)
		} else if !strings.Contains(err.Error(), "invalid --scope") {
			t.Errorf("scope %q: unexpected error %v", scope, err)
		}
	}
}

func TestHookSnapshotPathPrecedence(t *testing.T) {
	configHome, _ := isolateHookEnv(t)
	pluginDir := t.TempDir()
	claudeDir := t.TempDir()

	t.Setenv("CLAUDE_PLUGIN_DATA", claudeDir)
	if got, want := hookSnapshotPath(), filepath.Join(claudeDir, "workspace-snapshot.json"); got != want {
		t.Errorf("CLAUDE_PLUGIN_DATA path = %q, want %q", got, want)
	}

	t.Setenv("PLUGIN_DATA", "  "+pluginDir+"  ")
	if got, want := hookSnapshotPath(), filepath.Join(pluginDir, "workspace-snapshot.json"); got != want {
		t.Errorf("PLUGIN_DATA must win and be trimmed: got %q, want %q", got, want)
	}

	t.Setenv("PLUGIN_DATA", "   ")
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	got := hookSnapshotPath()
	if filepath.Base(got) != "workspace-snapshot.json" {
		t.Errorf("fallback file name = %q", filepath.Base(got))
	}
	if strings.HasPrefix(got, configHome) {
		// os.UserCacheDir failed; the config-dir fallback applies.
		if want := filepath.Join(configHome, "gws", "hooks-cache", "workspace-snapshot.json"); got != want {
			t.Errorf("config fallback = %q, want %q", got, want)
		}
	} else if want := filepath.Join("gws", "hooks", "workspace-snapshot.json"); !strings.HasSuffix(got, want) {
		t.Errorf("cache fallback = %q, want suffix %q", got, want)
	}
}

func TestRenderHookDisabledAndInvalidEventAreEmpty(t *testing.T) {
	_, _ = isolateHookEnv(t)
	pluginData := t.TempDir()
	t.Setenv("PLUGIN_DATA", pluginData)
	cwd := t.TempDir()

	disabled := pluginhooks.DefaultConfig()
	disabled.Enabled = false
	got := renderHook(t.Context(), pluginhooks.EventSessionStart, disabled, cwd)
	if got.AdditionalContext != "" || got.Refreshed || got.WorkspaceDegraded {
		t.Errorf("disabled config rendered %+v, want empty", got)
	}

	enabled := pluginhooks.DefaultConfig()
	enabled.Enabled = true
	got = renderHook(t.Context(), pluginhooks.Event("Bogus"), enabled, cwd)
	if got.AdditionalContext != "" {
		t.Errorf("invalid event rendered %q, want empty", got.AdditionalContext)
	}

	if _, err := os.Stat(filepath.Join(pluginData, "workspace-snapshot.json")); !os.IsNotExist(err) {
		t.Errorf("snapshot must not be written for no-op renders, stat err = %v", err)
	}
}

func TestRenderHookTimeOnlyNeverRunsWorkspaceSources(t *testing.T) {
	_, _ = isolateHookEnv(t)
	pluginData := t.TempDir()
	t.Setenv("PLUGIN_DATA", pluginData)
	cwd := t.TempDir()

	cfg := pluginhooks.DefaultConfig()
	cfg.Enabled = true
	cfg.Timezone = "UTC"
	cfg.Providers.Time.Enabled = true
	cfg.Providers.Workspace.Enabled = false

	got := renderHook(t.Context(), pluginhooks.EventSessionStart, cfg, cwd)
	if strings.TrimSpace(got.AdditionalContext) == "" {
		t.Fatal("expected time context for enabled time provider")
	}
	if got.Refreshed || got.WorkspaceDegraded {
		t.Errorf("workspace provider is off; got Refreshed=%v WorkspaceDegraded=%v", got.Refreshed, got.WorkspaceDegraded)
	}
	if got.ResolvedTimezone != "UTC" {
		t.Errorf("ResolvedTimezone = %q, want UTC", got.ResolvedTimezone)
	}
}
