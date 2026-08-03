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
