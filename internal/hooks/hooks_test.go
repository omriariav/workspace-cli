package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type memoryStore struct {
	snapshot Snapshot
}

func (s *memoryStore) Load() (Snapshot, error) { return s.snapshot, nil }
func (s *memoryStore) Save(snapshot Snapshot) error {
	s.snapshot = snapshot
	return nil
}

type loadFailingStore struct {
	saved Snapshot
}

func (s *loadFailingStore) Load() (Snapshot, error) {
	return Snapshot{}, errors.New("cache unavailable")
}
func (s *loadFailingStore) Save(snapshot Snapshot) error {
	s.saved = snapshot
	return nil
}

func enabledConfig() Config {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Providers.Workspace.Enabled = true
	cfg.Sources.Gmail.Enabled = false
	cfg.Sources.SentMail.Enabled = false
	cfg.Sources.Tasks.Enabled = false
	cfg.Sources.Chat.Enabled = false
	return cfg
}

func TestDisabledAndTimeOnlyNeverRunWorkspaceCommands(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	calls := 0
	runtime := Runtime{
		Now:   func() time.Time { return now },
		Local: time.FixedZone("IST", 2*60*60),
		Run: func(context.Context, []string) ([]byte, error) {
			calls++
			return nil, errors.New("must not run")
		},
	}

	if got := runtime.Render(context.Background(), EventUserPromptSubmit, DefaultConfig()); got.AdditionalContext != "" || got.ResolvedTimezone != "system-local" {
		t.Fatalf("disabled config result = %#v", got)
	}
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Providers.Time.Enabled = true
	got := runtime.Render(context.Background(), EventUserPromptSubmit, cfg)
	if calls != 0 {
		t.Fatalf("time-only mode made %d Workspace calls", calls)
	}
	if !strings.Contains(got.AdditionalContext, "Monday, 2026-08-03") || !strings.Contains(got.AdditionalContext, "[system-local]") {
		t.Fatalf("unexpected time context: %q", got.AdditionalContext)
	}
}

func TestWorkspaceTTLAndStaleRefresh(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	calls := 0
	store := &memoryStore{}
	runtime := Runtime{
		Now:   func() time.Time { return now },
		Local: time.UTC,
		Store: store,
		Run: func(context.Context, []string) ([]byte, error) {
			calls++
			return []byte(`{"events":[{"id":"one","summary":"Planning"}]}`), nil
		},
	}
	cfg := enabledConfig()
	first := runtime.Render(context.Background(), EventSessionStart, cfg)
	if !first.Refreshed || calls != 1 {
		t.Fatalf("SessionStart refresh = %v, calls = %d", first.Refreshed, calls)
	}
	now = now.Add(4 * time.Minute)
	insideTTL := runtime.Render(context.Background(), EventUserPromptSubmit, cfg)
	if insideTTL.Refreshed || calls != 1 {
		t.Fatalf("cache hit refreshed = %v, calls = %d", insideTTL.Refreshed, calls)
	}
	now = now.Add(2 * time.Minute)
	stale := runtime.Render(context.Background(), EventUserPromptSubmit, cfg)
	if !stale.Refreshed || calls != 2 {
		t.Fatalf("stale prompt refresh = %v, calls = %d", stale.Refreshed, calls)
	}
}

func TestSnapshotIsBoundToWorkspaceIdentity(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	cfg := enabledConfig()
	workspaceA := Runtime{
		Now:          func() time.Time { return now },
		Local:        time.UTC,
		Store:        store,
		WorkspaceKey: "/projects/a",
		Run: func(context.Context, []string) ([]byte, error) {
			return []byte(`{"events":[{"summary":"workspace-a"}]}`), nil
		},
	}
	first := workspaceA.Render(context.Background(), EventSessionStart, cfg)
	if !first.Refreshed || !strings.Contains(first.AdditionalContext, "workspace-a") {
		t.Fatalf("workspace A did not create its snapshot: %#v", first)
	}

	workspaceBCalls := 0
	workspaceB := Runtime{
		Now:          func() time.Time { return now.Add(time.Minute) },
		Local:        time.UTC,
		Store:        store,
		WorkspaceKey: "/projects/b",
		Run: func(context.Context, []string) ([]byte, error) {
			workspaceBCalls++
			return []byte(`{"events":[{"summary":"workspace-b"}]}`), nil
		},
	}
	second := workspaceB.Render(context.Background(), EventUserPromptSubmit, cfg)
	if !second.Refreshed || workspaceBCalls != 1 || !strings.Contains(second.AdditionalContext, "workspace-b") || strings.Contains(second.AdditionalContext, "workspace-a") {
		t.Fatalf("workspace B reused workspace A context: calls=%d context=%q", workspaceBCalls, second.AdditionalContext)
	}
}

func TestSourceConfigurationChangeInvalidatesSnapshotWithinTTL(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	calls := 0
	runtime := Runtime{
		Now:          func() time.Time { return now },
		Local:        time.UTC,
		Store:        store,
		WorkspaceKey: "/projects/config-change",
		Run: func(_ context.Context, args []string) ([]byte, error) {
			calls++
			if args[2] == "calendar" {
				return []byte(`{"events":[{"summary":"calendar-only"}]}`), nil
			}
			return []byte(`{"threads":[{"snippet":"gmail-only"}]}`), nil
		},
	}
	firstConfig := enabledConfig()
	first := runtime.Render(context.Background(), EventSessionStart, firstConfig)
	if !first.Refreshed || !strings.Contains(first.AdditionalContext, "calendar-only") {
		t.Fatalf("initial snapshot failed: %#v", first)
	}

	now = now.Add(time.Minute)
	changedConfig := firstConfig
	changedConfig.Sources.Calendar.Enabled = false
	changedConfig.Sources.Gmail.Enabled = true
	second := runtime.Render(context.Background(), EventUserPromptSubmit, changedConfig)
	if !second.Refreshed || calls != 2 || !strings.Contains(second.AdditionalContext, "gmail-only") || strings.Contains(second.AdditionalContext, "calendar-only") {
		t.Fatalf("source configuration change reused stale context: calls=%d context=%q", calls, second.AdditionalContext)
	}
}

func TestChatSpaceChangeInvalidatesSnapshotWithinTTL(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	calls := 0
	runtime := Runtime{
		Now:          func() time.Time { return now },
		Local:        time.UTC,
		Store:        store,
		WorkspaceKey: "/projects/chat-space-change",
		Run: func(_ context.Context, args []string) ([]byte, error) {
			calls++
			return []byte(`{"messages":[{"text":"` + args[4] + `"}]}`), nil
		},
	}
	cfg := enabledConfig()
	cfg.Sources.Calendar.Enabled = false
	cfg.Sources.Chat.Enabled = true
	cfg.Sources.Chat.Spaces = []string{"spaces/one"}
	first := runtime.Render(context.Background(), EventSessionStart, cfg)
	if !first.Refreshed || !strings.Contains(first.AdditionalContext, "spaces/one") {
		t.Fatalf("initial Chat snapshot failed: %#v", first)
	}

	now = now.Add(time.Minute)
	cfg.Sources.Chat.Spaces = []string{"spaces/two"}
	second := runtime.Render(context.Background(), EventUserPromptSubmit, cfg)
	if !second.Refreshed || calls != 2 || !strings.Contains(second.AdditionalContext, "spaces/two") || strings.Contains(second.AdditionalContext, "spaces/one") {
		t.Fatalf("Chat space configuration reused stale context: calls=%d context=%q", calls, second.AdditionalContext)
	}
}

func TestCacheLoadFailureStillPerformsFreshBoundedRead(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	calls := 0
	store := &loadFailingStore{}
	runtime := Runtime{
		Now:   func() time.Time { return now },
		Local: time.UTC,
		Store: store,
		Run: func(context.Context, []string) ([]byte, error) {
			calls++
			return []byte(`{"events":[{"id":"fresh"}]}`), nil
		},
	}
	got := runtime.Render(context.Background(), EventSessionStart, enabledConfig())
	if calls != 1 || !got.Refreshed || store.saved.Context == "" {
		t.Fatalf("cache error must still collect fresh context: calls=%d refreshed=%v snapshot=%#v", calls, got.Refreshed, store.saved)
	}
	if len(got.Diagnostics) != 1 || !strings.Contains(got.Diagnostics[0], "fresh bounded read") {
		t.Fatalf("missing cache recovery diagnostic: %#v", got.Diagnostics)
	}
}

func TestTaskDiscoveryIsBoundedAndReadOnly(t *testing.T) {
	cfg := enabledConfig()
	cfg.Sources.Calendar.Enabled = false
	cfg.Sources.Tasks.Enabled = true
	cfg.Sources.Tasks.DiscoveryLimit = 1
	cfg.Sources.Tasks.Limit = 2
	calls := make([][]string, 0, 2)
	runtime := Runtime{
		Now:   func() time.Time { return time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC) },
		Local: time.UTC,
		Run: func(_ context.Context, args []string) ([]byte, error) {
			calls = append(calls, args)
			if args[3] == "lists" {
				return []byte(`{"tasklists":[{"id":"first"},{"id":"second"}]}`), nil
			}
			return []byte(`{"tasks":[{"id":"task-one","title":"Bounded task"}]}`), nil
		},
	}
	got := runtime.Render(context.Background(), EventSessionStart, cfg)
	if len(calls) != 2 {
		t.Fatalf("expected list discovery plus one bounded task read, got %d calls: %#v", len(calls), calls)
	}
	if calls[1][3] != "list" || calls[1][4] != "first" || !ReadOnlyArgs(calls[1]) {
		t.Fatalf("unexpected discovered task command: %#v", calls[1])
	}
	if !strings.Contains(got.AdditionalContext, "Bounded task") {
		t.Fatalf("task context missing discovered list result: %q", got.AdditionalContext)
	}
}

func TestDegradedRefreshRetainsMoreUsefulSnapshot(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	phase := 0
	runtime := Runtime{
		Now:   func() time.Time { return now },
		Local: time.UTC,
		Store: store,
		Run: func(_ context.Context, args []string) ([]byte, error) {
			if phase == 1 && args[2] == "gmail" {
				return nil, errors.New("temporary auth failure")
			}
			return []byte(`{"items":[{"id":"stable"}]}`), nil
		},
	}
	cfg := enabledConfig()
	cfg.Sources.Gmail.Enabled = true
	first := runtime.Render(context.Background(), EventSessionStart, cfg)
	if !first.Refreshed || store.snapshot.UsefulSources() != 2 {
		t.Fatalf("expected a two-source initial snapshot, got %#v", store.snapshot)
	}
	phase = 1
	now = now.Add(6 * time.Minute)
	second := runtime.Render(context.Background(), EventUserPromptSubmit, cfg)
	if second.Refreshed {
		t.Fatal("less-complete refresh should not replace the existing snapshot")
	}
	if !strings.Contains(second.AdditionalContext, first.AdditionalContext) {
		t.Fatalf("degraded refresh clobbered snapshot\nwant preserved: %q\n got: %q", first.AdditionalContext, second.AdditionalContext)
	}
	if len(second.Diagnostics) == 0 || !strings.Contains(second.Diagnostics[0], "less complete") {
		t.Fatalf("missing truthful degradation diagnostic: %#v", second.Diagnostics)
	}
}

func TestSameCountSourceLossRetainsPreviousSnapshot(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	phase := 0
	runtime := Runtime{
		Now:   func() time.Time { return now },
		Local: time.UTC,
		Store: store,
		Run: func(_ context.Context, args []string) ([]byte, error) {
			isSentMail := strings.Contains(strings.Join(args, " "), "in:sent")
			if phase == 0 && isSentMail {
				return nil, errors.New("temporary sent-mail failure")
			}
			if phase == 1 && args[2] == "gmail" && !isSentMail {
				return nil, errors.New("temporary Gmail failure")
			}
			return []byte(`{"items":[{"id":"stable"}]}`), nil
		},
	}
	cfg := enabledConfig()
	cfg.Sources.Gmail.Enabled = true
	cfg.Sources.SentMail.Enabled = true
	first := runtime.Render(context.Background(), EventSessionStart, cfg)
	if !first.Refreshed || store.snapshot.UsefulSources() != 2 {
		t.Fatalf("expected initial Calendar and Gmail snapshot, got %#v", store.snapshot)
	}
	phase = 1
	now = now.Add(6 * time.Minute)
	second := runtime.Render(context.Background(), EventUserPromptSubmit, cfg)
	if second.Refreshed || !strings.Contains(second.AdditionalContext, "Gmail") || strings.Contains(second.AdditionalContext, "Sent mail: {") {
		t.Fatalf("same-count source replacement should retain known-good snapshot: %q", second.AdditionalContext)
	}
	if len(second.Diagnostics) != 1 || !strings.Contains(second.Diagnostics[0], "less complete") {
		t.Fatalf("missing source-loss diagnostic: %#v", second.Diagnostics)
	}
}

func TestTimeFormattingUsesStableEnglishAndSafeTimezoneFallback(t *testing.T) {
	now := time.Date(2026, time.August, 3, 12, 5, 0, 0, time.UTC)
	explicit, diagnostic := FormatTimeContext(now, "America/New_York", time.UTC)
	if diagnostic != "" || !strings.Contains(explicit, "Monday, 2026-08-03") || !strings.Contains(explicit, "[America/New_York]") {
		t.Fatalf("explicit timezone result = %q, diagnostic = %q", explicit, diagnostic)
	}
	broken, diagnostic := FormatTimeContext(now, "Not/AZone", time.FixedZone("IST", 2*60*60))
	if !strings.Contains(diagnostic, "could not be resolved") || !strings.Contains(broken, "IST (+02:00) [system-local]") {
		t.Fatalf("fallback result = %q, diagnostic = %q", broken, diagnostic)
	}
}

func TestConciseJSONEscapesAndFiltersWorkingLocations(t *testing.T) {
	context := conciseJSON([]byte(`{"events":[{"event_type":"workingLocation","summary":"Home"},{"event_type":"default","summary":"Quoted \"value\""}]}`), 900)
	if strings.Contains(context, "workingLocation") || strings.Contains(context, "Home") {
		t.Fatalf("working location leaked into context: %s", context)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(context), &decoded); err != nil {
		t.Fatalf("context is not valid JSON: %v\n%s", err, context)
	}
	if !strings.Contains(context, `Quoted \"value\"`) {
		t.Fatalf("escaped source was not retained safely: %s", context)
	}
}

func TestConfigPreviewIsMergeOnlyAtomicAndPreservesUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	initial := []byte(`{"enabled":false,"future":{"kept":true},"providers":{"time":{"enabled":false}}}`)
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatal(err)
	}
	patch := map[string]any{"enabled": true, "providers": map[string]any{"time": map[string]any{"enabled": true}}}
	preview, err := UpdateFile(path, patch, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview["enabled"] != true {
		t.Fatalf("preview did not apply patch: %#v", preview)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != string(initial) {
		t.Fatal("preview unexpectedly wrote a configuration file")
	}
	if _, err := UpdateFile(path, patch, true); err != nil {
		t.Fatal(err)
	}
	updated, err := readConfigMap(path)
	if err != nil {
		t.Fatal(err)
	}
	if future, ok := updated["future"].(map[string]any); !ok || future["kept"] != true {
		t.Fatalf("unknown key was not preserved: %#v", updated)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("expected a mode-0600 config, mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestEffectiveConfigPrecedenceAndReadOnlyAllowlist(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "user.json")
	workspacePath := filepath.Join(dir, "workspace.json")
	if err := os.WriteFile(userPath, []byte(`{"enabled":true,"timezone":"America/Chicago"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workspacePath, []byte(`{"timezone":"Europe/London","providers":{"time":{"enabled":true}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadEffective(userPath, workspacePath, func(key string) string {
		if key == "GWS_HOOKS_TIMEZONE" {
			return "Asia/Tokyo"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || !cfg.Providers.Time.Enabled || cfg.Timezone != "Asia/Tokyo" {
		t.Fatalf("unexpected precedence result: %#v", cfg)
	}
	for _, args := range [][]string{
		{"--format", "json", "calendar", "events"},
		{"--format", "json", "gmail", "list"},
		{"--format", "json", "tasks", "lists"},
		{"--format", "json", "tasks", "list", "id"},
		{"--format", "json", "chat", "unread", "spaces/id"},
	} {
		if !ReadOnlyArgs(args) {
			t.Fatalf("read-only command rejected: %#v", args)
		}
	}
	if ReadOnlyArgs([]string{"--format", "json", "gmail", "send"}) {
		t.Fatal("mutating Gmail command was accepted")
	}
}

func TestWorkspaceConfigRejectsCredentialsAndCachedContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	if err := os.WriteFile(path, []byte(`{"providers":{"time":{"enabled":true}},"oauth_token":"not allowed"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWorkspaceConfig(path); err == nil || !strings.Contains(err.Error(), "oauth_token") {
		t.Fatalf("expected workspace credential rejection, got %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"sources":{"tasks":{"list_ids":[{"cached_token":"not allowed"}]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWorkspaceConfig(path); err == nil || !strings.Contains(err.Error(), "cached_token") {
		t.Fatalf("expected nested-array credential rejection, got %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"unrelated_preference":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWorkspaceConfig(path); err == nil || !strings.Contains(err.Error(), "unsupported preference") {
		t.Fatalf("expected workspace preference allowlist rejection, got %v", err)
	}
}
