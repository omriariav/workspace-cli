package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultOverallTimeout = 8 * time.Second
	defaultSourceTimeout  = 3 * time.Second
	maxSnapshotAge        = 24 * time.Hour
)

type Event string

const (
	EventSessionStart     Event = "SessionStart"
	EventUserPromptSubmit Event = "UserPromptSubmit"
)

func (e Event) Valid() bool {
	return e == EventSessionStart || e == EventUserPromptSubmit
}

type Snapshot struct {
	UpdatedAt time.Time      `json:"updated_at"`
	Context   string         `json:"context"`
	Sources   []SourceResult `json:"sources"`
}

func (s Snapshot) UsefulSources() int {
	count := 0
	for _, source := range s.Sources {
		if source.Error == "" && source.Context != "" {
			count++
		}
	}
	return count
}

type SourceResult struct {
	Name    string `json:"name"`
	Context string `json:"context,omitempty"`
	Error   string `json:"error,omitempty"`
}

type SnapshotStore interface {
	Load() (Snapshot, error)
	Save(Snapshot) error
}

type FileStore struct {
	Path string
}

func (s FileStore) Load() (Snapshot, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (s FileStore) Save(snapshot Snapshot) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".snapshot-*.json")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, s.Path)
}

// Runner receives only gws arguments. Implementations must never invoke a
// shell, which keeps configuration values from becoming executable input.
type Runner func(context.Context, []string) ([]byte, error)

type Runtime struct {
	Now            func() time.Time
	Local          *time.Location
	Run            Runner
	Store          SnapshotStore
	OverallTimeout time.Duration
	SourceTimeout  time.Duration
}

type RenderedContext struct {
	AdditionalContext string
	Refreshed         bool
	Diagnostics       []string
	ResolvedTimezone  string
	Sources           []SourceStatus
	SnapshotUpdatedAt time.Time
}

// SourceStatus is intentionally metadata-only so health checks can report
// source availability without exposing cached Workspace content.
type SourceStatus struct {
	Name       string `json:"name"`
	Available  bool   `json:"available"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

func (r Runtime) Render(ctx context.Context, event Event, cfg Config) RenderedContext {
	if !event.Valid() {
		return RenderedContext{}
	}
	if r.Local == nil {
		r.Local = time.Local
	}
	_, resolvedTimezone, _ := resolveTimezone(cfg.Timezone, r.Local)
	if resolvedTimezone == "" {
		resolvedTimezone = "system-local"
	}
	if !cfg.Enabled {
		return RenderedContext{ResolvedTimezone: resolvedTimezone}
	}
	if r.Now == nil {
		r.Now = time.Now
	}
	if r.OverallTimeout <= 0 {
		r.OverallTimeout = defaultOverallTimeout
	}
	if r.SourceTimeout <= 0 {
		r.SourceTimeout = defaultSourceTimeout
	}

	now := r.Now()
	sections := make([]string, 0, 2)
	diagnostics := make([]string, 0, 2)
	if cfg.Providers.Time.Enabled {
		timeContext, diagnostic := FormatTimeContext(now, cfg.Timezone, r.Local)
		sections = append(sections, timeContext)
		if diagnostic != "" {
			diagnostics = append(diagnostics, diagnostic)
		}
	}

	refreshed := false
	workspace := Snapshot{}
	if cfg.Providers.Workspace.Enabled {
		var didRefresh bool
		var diagnostic string
		workspace, didRefresh, diagnostic = r.workspace(ctx, event, cfg, now)
		if workspace.Context != "" {
			sections = append(sections, workspace.Context)
		}
		refreshed = didRefresh
		if diagnostic != "" {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	if len(diagnostics) > 0 {
		sections = append(sections, "Hook diagnostics: "+strings.Join(diagnostics, "; "))
	}
	return RenderedContext{
		AdditionalContext: strings.Join(sections, "\n\n"),
		Refreshed:         refreshed,
		Diagnostics:       diagnostics,
		ResolvedTimezone:  resolvedTimezone,
		Sources:           sourceStatuses(workspace.Sources),
		SnapshotUpdatedAt: workspace.UpdatedAt,
	}
}

func (r Runtime) workspace(ctx context.Context, event Event, cfg Config, now time.Time) (Snapshot, bool, string) {
	previous := Snapshot{}
	cacheDiagnostic := ""
	if r.Store != nil {
		loaded, err := r.Store.Load()
		if err != nil {
			cacheDiagnostic = "Workspace cache is unavailable; completed a fresh bounded read"
		} else if !loaded.UpdatedAt.IsZero() && now.Sub(loaded.UpdatedAt) <= maxSnapshotAge {
			previous = loaded
		}
	}

	refreshAfter := time.Duration(cfg.Providers.Workspace.RefreshMinutes) * time.Minute
	needsRefresh := event == EventSessionStart || previous.UpdatedAt.IsZero() || now.Sub(previous.UpdatedAt) >= refreshAfter
	if !needsRefresh {
		return previous, false, ""
	}

	refreshCtx, cancel := context.WithTimeout(ctx, r.OverallTimeout)
	defer cancel()
	fresh := r.collectWorkspace(refreshCtx, cfg, now)
	fresh.UpdatedAt = now
	if fresh.UsefulSources() == 0 && previous.Context != "" {
		return previous, false, "Workspace refresh failed; retained the previous context"
	}
	if previous.Context != "" && refreshLostUsefulSource(previous, fresh, cfg) {
		return previous, false, "Workspace refresh was less complete; retained the previous context"
	}
	if r.Store != nil {
		if err := r.Store.Save(fresh); err != nil {
			if cacheDiagnostic != "" {
				return fresh, true, cacheDiagnostic + "; Workspace refreshed but its volatile cache could not be saved"
			}
			return fresh, true, "Workspace refreshed but its volatile cache could not be saved"
		}
	}
	return fresh, true, cacheDiagnostic
}

func refreshLostUsefulSource(previous, fresh Snapshot, cfg Config) bool {
	freshByName := make(map[string]SourceResult, len(fresh.Sources))
	for _, source := range fresh.Sources {
		freshByName[source.Name] = source
	}
	for _, source := range previous.Sources {
		if !sourceEnabled(cfg, source.Name) || source.Error != "" || source.Context == "" {
			continue
		}
		updated, ok := freshByName[source.Name]
		if !ok || updated.Error != "" || updated.Context == "" {
			return true
		}
	}
	return false
}

func sourceEnabled(cfg Config, name string) bool {
	switch name {
	case "Calendar":
		return cfg.Sources.Calendar.Enabled
	case "Gmail":
		return cfg.Sources.Gmail.Enabled
	case "Sent mail":
		return cfg.Sources.SentMail.Enabled
	case "Tasks":
		return cfg.Sources.Tasks.Enabled
	case "Chat":
		return cfg.Sources.Chat.Enabled && len(cfg.Sources.Chat.Spaces) > 0
	default:
		return false
	}
}

func sourceStatuses(sources []SourceResult) []SourceStatus {
	statuses := make([]SourceStatus, 0, len(sources))
	for _, source := range sources {
		statuses = append(statuses, SourceStatus{
			Name:       source.Name,
			Available:  source.Error == "" && source.Context != "",
			Diagnostic: source.Error,
		})
	}
	return statuses
}

func (r Runtime) collectWorkspace(ctx context.Context, cfg Config, now time.Time) Snapshot {
	type sourceCall struct {
		name string
		call func(context.Context) SourceResult
	}
	calls := make([]sourceCall, 0, 5)
	if cfg.Sources.Calendar.Enabled {
		calls = append(calls, sourceCall{"Calendar", func(ctx context.Context) SourceResult { return r.calendar(ctx, cfg, now) }})
	}
	if cfg.Sources.Gmail.Enabled {
		calls = append(calls, sourceCall{"Gmail", func(ctx context.Context) SourceResult { return r.gmail(ctx, cfg) }})
	}
	if cfg.Sources.SentMail.Enabled {
		calls = append(calls, sourceCall{"Sent mail", func(ctx context.Context) SourceResult { return r.sentMail(ctx, cfg) }})
	}
	if cfg.Sources.Tasks.Enabled {
		calls = append(calls, sourceCall{"Tasks", func(ctx context.Context) SourceResult { return r.tasks(ctx, cfg) }})
	}
	if cfg.Sources.Chat.Enabled && len(cfg.Sources.Chat.Spaces) > 0 {
		calls = append(calls, sourceCall{"Chat", func(ctx context.Context) SourceResult { return r.chat(ctx, cfg) }})
	}

	results := make([]SourceResult, len(calls))
	var group sync.WaitGroup
	for index, source := range calls {
		group.Add(1)
		go func(index int, source sourceCall) {
			defer group.Done()
			results[index] = source.call(ctx)
		}(index, source)
	}
	group.Wait()
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return Snapshot{Sources: results, Context: renderWorkspaceContext(results)}
}

func (r Runtime) calendar(ctx context.Context, cfg Config, now time.Time) SourceResult {
	start := now.AddDate(0, 0, -1).Format("2006-01-02")
	return r.runSource(ctx, "Calendar", []string{"--format", "json", "calendar", "events", "--from", start, "--days", fmt.Sprint(cfg.Sources.Calendar.LookaheadDays + 1), "--max", fmt.Sprint(cfg.Sources.Calendar.Limit)})
}

func (r Runtime) gmail(ctx context.Context, cfg Config) SourceResult {
	return r.runSource(ctx, "Gmail", []string{"--format", "json", "gmail", "list", "--query", "is:starred OR (is:important is:unread)", "--max", fmt.Sprint(cfg.Sources.Gmail.Limit)})
}

func (r Runtime) sentMail(ctx context.Context, cfg Config) SourceResult {
	query := fmt.Sprintf("in:sent newer_than:%dd", cfg.Sources.SentMail.LookbackDays)
	return r.runSource(ctx, "Sent mail", []string{"--format", "json", "gmail", "list", "--query", query, "--max", fmt.Sprint(cfg.Sources.SentMail.Limit)})
}

func (r Runtime) tasks(ctx context.Context, cfg Config) SourceResult {
	listIDs := cfg.Sources.Tasks.ListIDs
	if len(listIDs) == 0 {
		listData, err := r.runRaw(ctx, []string{"--format", "json", "tasks", "lists"})
		if err != nil {
			return SourceResult{Name: "Tasks", Error: boundedError(err)}
		}
		listIDs = discoveredTaskListIDs(listData, cfg.Sources.Tasks.DiscoveryLimit)
		if len(listIDs) == 0 {
			return SourceResult{Name: "Tasks", Context: conciseJSON(listData, 900)}
		}
	}
	parts := make([]string, 0, len(listIDs))
	for _, id := range listIDs {
		result := r.runSource(ctx, "Tasks", []string{"--format", "json", "tasks", "list", id, "--max", fmt.Sprint(cfg.Sources.Tasks.Limit)})
		if result.Error != "" {
			return result
		}
		parts = append(parts, result.Context)
	}
	return SourceResult{Name: "Tasks", Context: strings.Join(parts, " ")}
}

func (r Runtime) chat(ctx context.Context, cfg Config) SourceResult {
	parts := make([]string, 0, len(cfg.Sources.Chat.Spaces))
	for _, space := range cfg.Sources.Chat.Spaces {
		result := r.runSource(ctx, "Chat", []string{"--format", "json", "chat", "unread", space, "--max", fmt.Sprint(cfg.Sources.Chat.Limit)})
		if result.Error != "" {
			return result
		}
		parts = append(parts, result.Context)
	}
	return SourceResult{Name: "Chat", Context: strings.Join(parts, " ")}
}

func (r Runtime) runSource(parent context.Context, name string, args []string) SourceResult {
	output, err := r.runRaw(parent, args)
	if err != nil {
		return SourceResult{Name: name, Error: boundedError(err)}
	}
	return SourceResult{Name: name, Context: conciseJSON(output, 900)}
}

func (r Runtime) runRaw(parent context.Context, args []string) ([]byte, error) {
	if r.Run == nil {
		return nil, errors.New("gws runner is unavailable")
	}
	if !ReadOnlyArgs(args) {
		return nil, errors.New("unsafe hook command rejected")
	}
	ctx, cancel := context.WithTimeout(parent, r.SourceTimeout)
	defer cancel()
	return r.Run(ctx, args)
}

func discoveredTaskListIDs(data []byte, limit int) []string {
	var payload struct {
		TaskLists []struct {
			ID string `json:"id"`
		} `json:"tasklists"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}
	if limit <= 0 || limit > len(payload.TaskLists) {
		limit = len(payload.TaskLists)
	}
	ids := make([]string, 0, limit)
	for _, taskList := range payload.TaskLists[:limit] {
		if taskList.ID != "" {
			ids = append(ids, taskList.ID)
		}
	}
	return ids
}

func renderWorkspaceContext(results []SourceResult) string {
	lines := make([]string, 0, len(results))
	for _, result := range results {
		if result.Error != "" {
			lines = append(lines, fmt.Sprintf("- %s: unavailable (%s)", result.Name, result.Error))
			continue
		}
		if result.Context != "" {
			lines = append(lines, fmt.Sprintf("- %s: %s", result.Name, result.Context))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "Workspace context (read-only, event-driven):\n" + strings.Join(lines, "\n")
}

func conciseJSON(data []byte, limit int) string {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return truncate(strings.TrimSpace(string(data)), limit)
	}
	filtered, include := removeWorkingLocations(value)
	if !include {
		return "No matching items"
	}
	encoded, err := json.Marshal(filtered)
	if err != nil {
		return truncate(strings.TrimSpace(string(data)), limit)
	}
	return truncate(string(encoded), limit)
}

func removeWorkingLocations(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if kind, ok := typed["event_type"].(string); ok && kind == "workingLocation" {
			return nil, false
		}
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			filtered, include := removeWorkingLocations(child)
			if include {
				result[key] = filtered
			}
		}
		return result, true
	case []any:
		result := make([]any, 0, len(typed))
		for _, child := range typed {
			filtered, include := removeWorkingLocations(child)
			if include {
				result = append(result, filtered)
			}
		}
		return result, true
	default:
		return value, true
	}
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

func boundedError(err error) string {
	value := strings.TrimSpace(err.Error())
	return truncate(value, 160)
}

// ReadOnlyArgs is an allowlist for every command the hook runtime can issue.
// It is intentionally public so tests and future source additions must prove
// their commands cannot mutate a Workspace account.
func ReadOnlyArgs(args []string) bool {
	if len(args) < 3 || args[0] != "--format" || args[1] != "json" {
		return false
	}
	switch args[2] {
	case "calendar":
		return len(args) >= 4 && args[3] == "events"
	case "gmail":
		return len(args) >= 4 && args[3] == "list"
	case "tasks":
		return len(args) >= 4 && (args[3] == "lists" || args[3] == "list")
	case "chat":
		return len(args) >= 4 && args[3] == "unread"
	default:
		return false
	}
}
