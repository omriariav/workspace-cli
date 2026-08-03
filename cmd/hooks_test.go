package cmd

import (
	"bytes"
	"encoding/json"
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

	timeOnly := pluginhooks.DefaultConfig()
	timeOnly.Enabled = true
	timeOnly.Providers.Time.Enabled = true
	if report := hookHealthReportWithToken(timeOnly, pluginhooks.RenderedContext{}, false); report["healthy"] != true {
		t.Fatalf("time-only hooks must remain healthy without Workspace auth: %#v", report)
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
