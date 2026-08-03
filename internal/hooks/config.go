// Package hooks owns the opt-in, local configuration used by the gws plugin
// lifecycle hooks. It deliberately does not reuse the OAuth config file: hook
// preferences never need to contain credentials.
package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ConfigFileName = "hooks.json"
	configDirName  = ".gws"
)

// Config is intentionally a small, secret-free contract. Unknown JSON fields
// are preserved by UpdateFile so newer versions can safely share a config
// directory with older versions.
type Config struct {
	Version   int       `json:"version"`
	Enabled   bool      `json:"enabled"`
	Timezone  string    `json:"timezone,omitempty"`
	Providers Providers `json:"providers"`
	Sources   Sources   `json:"sources"`
}

type Providers struct {
	Time      ProviderConfig    `json:"time"`
	Workspace WorkspaceProvider `json:"workspace"`
}

type ProviderConfig struct {
	Enabled bool `json:"enabled"`
}

type WorkspaceProvider struct {
	Enabled        bool `json:"enabled"`
	RefreshMinutes int  `json:"refresh_minutes"`
}

type Sources struct {
	Calendar CalendarSource `json:"calendar"`
	Gmail    GmailSource    `json:"gmail"`
	SentMail SentMailSource `json:"sent_mail"`
	Tasks    TasksSource    `json:"tasks"`
	Chat     ChatSource     `json:"chat"`
}

type CalendarSource struct {
	Enabled       bool `json:"enabled"`
	LookaheadDays int  `json:"lookahead_days"`
	Limit         int  `json:"limit"`
}

type GmailSource struct {
	Enabled bool `json:"enabled"`
	Limit   int  `json:"limit"`
}

type SentMailSource struct {
	Enabled      bool `json:"enabled"`
	LookbackDays int  `json:"lookback_days"`
	Limit        int  `json:"limit"`
}

type TasksSource struct {
	Enabled        bool     `json:"enabled"`
	ListIDs        []string `json:"list_ids,omitempty"`
	DiscoveryLimit int      `json:"discovery_limit"`
	Limit          int      `json:"limit"`
}

type ChatSource struct {
	Enabled bool     `json:"enabled"`
	Spaces  []string `json:"spaces,omitempty"`
	Limit   int      `json:"limit"`
}

func DefaultConfig() Config {
	return Config{
		Version: 1,
		Providers: Providers{
			Time:      ProviderConfig{},
			Workspace: WorkspaceProvider{RefreshMinutes: 5},
		},
		Sources: Sources{
			Calendar: CalendarSource{Enabled: true, LookaheadDays: 7, Limit: 8},
			Gmail:    GmailSource{Enabled: true, Limit: 8},
			SentMail: SentMailSource{Enabled: true, LookbackDays: 7, Limit: 5},
			Tasks:    TasksSource{Enabled: true, DiscoveryLimit: 3, Limit: 10},
			Chat:     ChatSource{Enabled: true, Limit: 5},
		},
	}
}

func UserConfigPath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

func WorkspaceConfigPath(cwd string) string {
	return filepath.Join(cwd, configDirName, ConfigFileName)
}

// LoadEffective applies the documented precedence: defaults, user config,
// workspace config, then the safe environment overrides. Missing config files
// are normal and do not cause a hook to fail.
func LoadEffective(userPath, workspacePath string, getenv func(string) string) (Config, error) {
	merged, err := configMap(DefaultConfig())
	if err != nil {
		return Config{}, err
	}
	for _, path := range []string{userPath, workspacePath} {
		if path == "" {
			continue
		}
		file, err := readConfigMap(path)
		if err != nil {
			return Config{}, err
		}
		deepMerge(merged, file)
	}

	if getenv != nil {
		if value := strings.TrimSpace(getenv("GWS_HOOKS_ENABLED")); value != "" {
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return Config{}, fmt.Errorf("invalid GWS_HOOKS_ENABLED: %w", err)
			}
			merged["enabled"] = enabled
		}
		if value := strings.TrimSpace(getenv("GWS_HOOKS_TIMEZONE")); value != "" {
			merged["timezone"] = value
		}
	}

	bytes, err := json.Marshal(merged)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode hook configuration: %w", err)
	}
	normalize(&cfg)
	return cfg, nil
}

func normalize(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Providers.Workspace.RefreshMinutes <= 0 {
		cfg.Providers.Workspace.RefreshMinutes = 5
	}
	if cfg.Sources.Calendar.LookaheadDays <= 0 {
		cfg.Sources.Calendar.LookaheadDays = 7
	}
	if cfg.Sources.Calendar.Limit <= 0 {
		cfg.Sources.Calendar.Limit = 8
	}
	if cfg.Sources.Gmail.Limit <= 0 {
		cfg.Sources.Gmail.Limit = 8
	}
	if cfg.Sources.SentMail.LookbackDays <= 0 {
		cfg.Sources.SentMail.LookbackDays = 7
	}
	if cfg.Sources.SentMail.Limit <= 0 {
		cfg.Sources.SentMail.Limit = 5
	}
	if cfg.Sources.Tasks.DiscoveryLimit <= 0 {
		cfg.Sources.Tasks.DiscoveryLimit = 3
	}
	if cfg.Sources.Tasks.Limit <= 0 {
		cfg.Sources.Tasks.Limit = 10
	}
	if cfg.Sources.Chat.Limit <= 0 {
		cfg.Sources.Chat.Limit = 5
	}
}

// UpdateFile returns the planned, merge-only JSON document. Callers must pass
// apply=true to persist it. This keeps setup preview-first by construction.
func UpdateFile(path string, patch map[string]any, apply bool) (map[string]any, error) {
	if path == "" {
		return nil, errors.New("hook configuration path is required")
	}
	current, err := readConfigMap(path)
	if err != nil {
		return nil, err
	}
	if len(current) == 0 {
		current, err = configMap(DefaultConfig())
		if err != nil {
			return nil, err
		}
	}
	deepMerge(current, patch)
	if !apply {
		return current, nil
	}
	if err := writeAtomicJSON(path, current); err != nil {
		return nil, err
	}
	return current, nil
}

func readConfigMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read hook configuration %s: %w", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("parse hook configuration %s: %w", path, err)
	}
	if value == nil {
		return map[string]any{}, nil
	}
	return value, nil
}

func configMap(cfg Config) (map[string]any, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func deepMerge(dst, src map[string]any) {
	for key, value := range src {
		if sourceMap, ok := value.(map[string]any); ok {
			if destinationMap, ok := dst[key].(map[string]any); ok {
				deepMerge(destinationMap, sourceMap)
				continue
			}
		}
		dst[key] = value
	}
}

func writeAtomicJSON(path string, value map[string]any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create hook configuration directory: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(dir, ".hooks-*.json")
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
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

// DetectDuplicateTimeHooks reports only existing known personal time hooks.
// It never changes, disables, or reads their contents.
func DetectDuplicateTimeHooks(home string) []string {
	paths := []string{
		filepath.Join(home, ".codex", "hooks", "inject-time.sh"),
		filepath.Join(home, ".claude", "hooks", "inject-time.sh"),
	}
	found := make([]string, 0, len(paths))
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			found = append(found, path)
		}
	}
	return found
}

// ValidateWorkspaceConfig accepts only documented, shareable preferences. A
// workspace config must never become a credential store, context cache, or a
// catch-all file shared by unrelated tools.
func ValidateWorkspaceConfig(path string) error {
	value, err := readConfigMap(path)
	if err != nil {
		return err
	}
	return ValidateWorkspacePreferences(value)
}

// ValidateWorkspacePreferences is also used before a workspace-scoped
// preview is written, so a rejected key can never be persisted by this CLI.
func ValidateWorkspacePreferences(value map[string]any) error {
	return validateWorkspacePreferences(value, workspacePreferenceSchema, "")
}

var workspacePreferenceSchema = map[string]any{
	"version":  nil,
	"enabled":  nil,
	"timezone": nil,
	"providers": map[string]any{
		"time": map[string]any{
			"enabled": nil,
		},
		"workspace": map[string]any{
			"enabled":         nil,
			"refresh_minutes": nil,
		},
	},
	"sources": map[string]any{
		"calendar": map[string]any{
			"enabled": nil, "lookahead_days": nil, "limit": nil,
		},
		"gmail": map[string]any{
			"enabled": nil, "limit": nil,
		},
		"sent_mail": map[string]any{
			"enabled": nil, "lookback_days": nil, "limit": nil,
		},
		"tasks": map[string]any{
			"enabled": nil, "list_ids": nil, "discovery_limit": nil, "limit": nil,
		},
		"chat": map[string]any{
			"enabled": nil, "spaces": nil, "limit": nil,
		},
	},
}

func validateWorkspacePreferences(value, allowed map[string]any, parent string) error {
	for key, child := range value {
		path := key
		if parent != "" {
			path = parent + "." + key
		}
		lower := strings.ToLower(key)
		for _, forbidden := range []string{"secret", "token", "credential", "snapshot", "cache"} {
			if strings.Contains(lower, forbidden) {
				return fmt.Errorf("workspace hook configuration must not contain %q", path)
			}
		}
		allowedChild, ok := allowed[key]
		if !ok {
			return fmt.Errorf("workspace hook configuration contains unsupported preference %q", path)
		}
		nestedAllowed, nested := allowedChild.(map[string]any)
		if nested {
			childMap, ok := child.(map[string]any)
			if !ok {
				return fmt.Errorf("workspace hook configuration preference %q must be an object", path)
			}
			if err := validateWorkspacePreferences(childMap, nestedAllowed, path); err != nil {
				return err
			}
			continue
		}
		if err := rejectSensitiveWorkspaceValue(child, path); err != nil {
			return err
		}
	}
	return nil
}

func rejectSensitiveWorkspaceValue(value any, parent string) error {
	switch typed := value.(type) {
	case map[string]any:
		return rejectSensitiveWorkspaceMap(typed, parent)
	case []any:
		for index, child := range typed {
			if err := rejectSensitiveWorkspaceValue(child, fmt.Sprintf("%s[%d]", parent, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectSensitiveWorkspaceMap(value map[string]any, parent string) error {
	for key, child := range value {
		path := key
		if parent != "" {
			path = parent + "." + key
		}
		lower := strings.ToLower(key)
		for _, forbidden := range []string{"secret", "token", "credential", "snapshot", "cache"} {
			if strings.Contains(lower, forbidden) {
				return fmt.Errorf("workspace hook configuration must not contain %q", path)
			}
		}
		if err := rejectSensitiveWorkspaceValue(child, path); err != nil {
			return err
		}
	}
	return nil
}
