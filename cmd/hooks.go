package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	pluginhooks "github.com/omriariav/workspace-cli/internal/hooks"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Configure opt-in agent context hooks",
	Long:  "Configure the opt-in, read-only GWS context hooks. Only user-scope configuration can enable hooks or providers; workspace configuration is limited to consent-free preferences. Configuration changes are previewed by default and require --apply to write.",
	Args:  cobra.NoArgs,
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Inspect effective hook configuration",
	Args:  cobra.NoArgs,
	RunE:  runHooksStatus,
}

var hooksEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Preview or enable one or more context providers",
	Args:  cobra.NoArgs,
	RunE:  runHooksEnable,
}

var hooksConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Preview or change hook preferences",
	Args:  cobra.NoArgs,
	RunE:  runHooksConfigure,
}

var hooksDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Preview or disable all hook fetching",
	Args:  cobra.NoArgs,
	RunE:  runHooksDisable,
}

var hooksTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Run a safe, on-demand hook health check",
	Args:  cobra.NoArgs,
	RunE:  runHooksTest,
}

var hooksRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run a packaged host hook",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE:   runHooksRun,
}

func init() {
	rootCmd.AddCommand(hooksCmd)
	hooksCmd.AddCommand(hooksStatusCmd, hooksEnableCmd, hooksConfigureCmd, hooksDisableCmd, hooksTestCmd, hooksRunCmd)

	for _, command := range []*cobra.Command{hooksEnableCmd, hooksConfigureCmd, hooksDisableCmd} {
		command.Flags().String("scope", "user", "configuration scope: user (consent) or workspace (preferences only)")
		command.Flags().Bool("apply", false, "write the previewed configuration")
	}
	hooksEnableCmd.Flags().Bool("time", false, "enable the local current-time provider (user scope only)")
	hooksEnableCmd.Flags().Bool("workspace", false, "enable the read-only Workspace provider (user scope only)")

	hooksConfigureCmd.Flags().String("timezone", "", "IANA timezone, for example America/New_York")
	hooksConfigureCmd.Flags().Bool("enabled", false, "enable or disable hook fetching (user scope only)")
	hooksConfigureCmd.Flags().Bool("time", false, "enable or disable the time provider (user scope only)")
	hooksConfigureCmd.Flags().Bool("workspace", false, "enable or disable the Workspace provider (user scope only)")
	hooksConfigureCmd.Flags().Int("refresh-minutes", 0, "Workspace refresh TTL in minutes")
	hooksConfigureCmd.Flags().Bool("calendar", false, "enable or disable Calendar context")
	hooksConfigureCmd.Flags().Int("calendar-lookahead-days", 0, "calendar context lookahead in days")
	hooksConfigureCmd.Flags().Int("calendar-limit", 0, "calendar event limit")
	hooksConfigureCmd.Flags().Bool("gmail", false, "enable or disable Gmail context")
	hooksConfigureCmd.Flags().Int("gmail-limit", 0, "important or starred Gmail thread limit")
	hooksConfigureCmd.Flags().Bool("sent-mail", false, "enable or disable sent-mail context")
	hooksConfigureCmd.Flags().Int("sent-limit", 0, "recent sent-mail thread limit")
	hooksConfigureCmd.Flags().Int("sent-lookback-days", 0, "recent sent-mail lookback in days")
	hooksConfigureCmd.Flags().Bool("tasks", false, "enable or disable Tasks context")
	hooksConfigureCmd.Flags().StringSlice("task-list", nil, "task list ID to include (repeatable)")
	hooksConfigureCmd.Flags().Int("task-discovery-limit", 0, "maximum task lists to discover when no IDs are configured")
	hooksConfigureCmd.Flags().Int("task-limit", 0, "maximum tasks per configured list")
	hooksConfigureCmd.Flags().Bool("chat", false, "enable or disable Chat context")
	hooksConfigureCmd.Flags().StringSlice("chat-space", nil, "explicit Chat space to include (repeatable)")
	hooksConfigureCmd.Flags().Int("chat-limit", 0, "maximum unread Chat messages per space")

	hooksTestCmd.Flags().String("event", string(pluginhooks.EventUserPromptSubmit), "hook event: SessionStart or UserPromptSubmit")
	hooksRunCmd.Flags().String("event", "", "hook event: SessionStart or UserPromptSubmit")
	hooksRunCmd.Flags().String("adapter", "codex", "host adapter: codex or claude")
}

func runHooksStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, err := loadHookConfig(cwd)
	if err != nil {
		return usageErrorf("unable to load hook configuration: %v", err)
	}
	home, _ := os.UserHomeDir()
	return writeHookOutput(map[string]any{
		"enabled":                  cfg.Enabled,
		"providers":                cfg.Providers,
		"sources":                  cfg.Sources,
		"timezone":                 cfg.Timezone,
		"user_config":              pluginhooks.UserConfigPath(config.GetConfigDir()),
		"workspace_config":         pluginhooks.WorkspaceConfigPath(cwd),
		"duplicate_time_injectors": pluginhooks.DetectDuplicateTimeHooks(home),
		"auth":                     hookAuthStatus(cfg),
		"source_gaps":              hookSourceGaps(cfg),
		"note":                     "No secrets, OAuth tokens, or cached context are included in hook status output.",
	})
}

func runHooksEnable(cmd *cobra.Command, args []string) error {
	if err := requireUserConsentScope(cmd); err != nil {
		return err
	}
	timeEnabled, _ := cmd.Flags().GetBool("time")
	workspaceEnabled, _ := cmd.Flags().GetBool("workspace")
	if !timeEnabled && !workspaceEnabled {
		return usageErrorf("choose at least one provider: --time and/or --workspace")
	}
	providers := map[string]any{}
	if timeEnabled {
		providers["time"] = map[string]any{"enabled": true}
	}
	if workspaceEnabled {
		providers["workspace"] = map[string]any{"enabled": true}
	}
	return previewHookUpdate(cmd, map[string]any{"enabled": true, "providers": providers})
}

func runHooksConfigure(cmd *cobra.Command, args []string) error {
	if consentFlagsChanged(cmd) {
		if err := requireUserConsentScope(cmd); err != nil {
			return err
		}
	}
	patch, err := configureHookPatch(cmd)
	if err != nil {
		return err
	}
	return previewHookUpdate(cmd, patch)
}

func configureHookPatch(cmd *cobra.Command) (map[string]any, error) {
	if err := requirePositiveHookFlags(cmd, []string{
		"refresh-minutes", "calendar-lookahead-days", "calendar-limit", "gmail-limit",
		"sent-limit", "sent-lookback-days", "task-discovery-limit", "task-limit", "chat-limit",
	}); err != nil {
		return nil, err
	}
	patch := map[string]any{}
	if cmd.Flags().Changed("enabled") {
		value, _ := cmd.Flags().GetBool("enabled")
		patch["enabled"] = value
	}
	if cmd.Flags().Changed("timezone") {
		value, _ := cmd.Flags().GetString("timezone")
		patch["timezone"] = value
	}
	providers := map[string]any{}
	if cmd.Flags().Changed("time") {
		value, _ := cmd.Flags().GetBool("time")
		providers["time"] = map[string]any{"enabled": value}
	}
	workspace := map[string]any{}
	if cmd.Flags().Changed("workspace") {
		value, _ := cmd.Flags().GetBool("workspace")
		workspace["enabled"] = value
	}
	if cmd.Flags().Changed("refresh-minutes") {
		value, _ := cmd.Flags().GetInt("refresh-minutes")
		workspace["refresh_minutes"] = value
	}
	if len(workspace) > 0 {
		providers["workspace"] = workspace
	}
	if len(providers) > 0 {
		patch["providers"] = providers
	}
	sources := map[string]any{}
	calendar := sourceNumberPatch(cmd, []string{"lookahead_days", "limit"}, []string{"calendar-lookahead-days", "calendar-limit"})
	if cmd.Flags().Changed("calendar") {
		value, _ := cmd.Flags().GetBool("calendar")
		calendar["enabled"] = value
	}
	if len(calendar) > 0 {
		sources["calendar"] = calendar
	}
	gmail := sourceNumberPatch(cmd, []string{"limit"}, []string{"gmail-limit"})
	if cmd.Flags().Changed("gmail") {
		value, _ := cmd.Flags().GetBool("gmail")
		gmail["enabled"] = value
	}
	if len(gmail) > 0 {
		sources["gmail"] = gmail
	}
	sentMail := sourceNumberPatch(cmd, []string{"lookback_days", "limit"}, []string{"sent-lookback-days", "sent-limit"})
	if cmd.Flags().Changed("sent-mail") {
		value, _ := cmd.Flags().GetBool("sent-mail")
		sentMail["enabled"] = value
	}
	if len(sentMail) > 0 {
		sources["sent_mail"] = sentMail
	}
	tasks := sourceNumberPatch(cmd, []string{"discovery_limit", "limit"}, []string{"task-discovery-limit", "task-limit"})
	if cmd.Flags().Changed("tasks") {
		value, _ := cmd.Flags().GetBool("tasks")
		tasks["enabled"] = value
	}
	if cmd.Flags().Changed("task-list") {
		value, _ := cmd.Flags().GetStringSlice("task-list")
		tasks["list_ids"] = value
	}
	if len(tasks) > 0 {
		sources["tasks"] = tasks
	}
	chat := sourceNumberPatch(cmd, []string{"limit"}, []string{"chat-limit"})
	if cmd.Flags().Changed("chat") {
		value, _ := cmd.Flags().GetBool("chat")
		chat["enabled"] = value
	}
	if cmd.Flags().Changed("chat-space") {
		value, _ := cmd.Flags().GetStringSlice("chat-space")
		chat["spaces"] = value
	}
	if len(chat) > 0 {
		sources["chat"] = chat
	}
	if len(sources) > 0 {
		patch["sources"] = sources
	}
	if len(patch) == 0 {
		return nil, usageErrorf("choose one or more configuration flags to change")
	}
	return patch, nil
}

func requirePositiveHookFlags(cmd *cobra.Command, flags []string) error {
	for _, flag := range flags {
		if !cmd.Flags().Changed(flag) {
			continue
		}
		value, _ := cmd.Flags().GetInt(flag)
		if value <= 0 {
			return usageErrorf("--%s must be positive", flag)
		}
	}
	return nil
}

func sourceNumberPatch(cmd *cobra.Command, fields, flags []string) map[string]any {
	patch := map[string]any{}
	for index, flag := range flags {
		if !cmd.Flags().Changed(flag) {
			continue
		}
		value, _ := cmd.Flags().GetInt(flag)
		patch[fields[index]] = value
	}
	return patch
}

func runHooksDisable(cmd *cobra.Command, args []string) error {
	if err := requireUserConsentScope(cmd); err != nil {
		return err
	}
	return previewHookUpdate(cmd, map[string]any{"enabled": false})
}

func consentFlagsChanged(cmd *cobra.Command) bool {
	for _, flag := range []string{"enabled", "time", "workspace"} {
		if cmd.Flags().Changed(flag) {
			return true
		}
	}
	return false
}

func requireUserConsentScope(cmd *cobra.Command) error {
	scope, _ := cmd.Flags().GetString("scope")
	if scope != "user" {
		return usageErrorf("hook and provider consent is user-scope only; workspace configuration cannot activate private reads")
	}
	return nil
}

func previewHookUpdate(cmd *cobra.Command, patch map[string]any) error {
	scope, _ := cmd.Flags().GetString("scope")
	if scope == "workspace" {
		if err := pluginhooks.ValidateWorkspacePreferences(patch); err != nil {
			return usageErrorf("invalid workspace hook configuration: %v", err)
		}
	}
	path, err := hookConfigPath(scope)
	if err != nil {
		return usageErrorf("%v", err)
	}
	apply, _ := cmd.Flags().GetBool("apply")
	update := pluginhooks.UpdateFile
	if scope == "workspace" {
		update = pluginhooks.UpdateWorkspaceFile
	}
	_, err = update(path, patch, false)
	if err != nil {
		return fmt.Errorf("update hook configuration: %w", err)
	}
	if apply {
		_, err = update(path, patch, true)
		if err != nil {
			return fmt.Errorf("update hook configuration: %w", err)
		}
	}
	return writeHookOutput(map[string]any{
		"action":  map[bool]string{true: "applied", false: "preview"}[apply],
		"scope":   scope,
		"path":    path,
		"changes": patch,
		"note":    "Review these exact requested changes and rerun with --apply only after explicit approval. Existing configuration values are not echoed.",
	})
}

func runHooksTest(cmd *cobra.Command, args []string) error {
	eventValue, _ := cmd.Flags().GetString("event")
	event := pluginhooks.Event(eventValue)
	if !event.Valid() {
		return usageErrorf("invalid --event %q", eventValue)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, err := loadHookConfig(cwd)
	if err != nil {
		return writeHookOutput(map[string]any{"healthy": false, "diagnostic": "Hook configuration could not be loaded: " + err.Error()})
	}
	rendered := renderHook(cmd.Context(), event, cfg, cwd)
	return writeHookOutput(hookHealthReport(cfg, rendered))
}

func hookHealthReport(cfg pluginhooks.Config, rendered pluginhooks.RenderedContext) map[string]any {
	return hookHealthReportWithToken(cfg, rendered, auth.TokenExists())
}

func hookHealthReportWithToken(cfg pluginhooks.Config, rendered pluginhooks.RenderedContext, tokenPresent bool) map[string]any {
	snapshotUpdatedAt := any(nil)
	if !rendered.SnapshotUpdatedAt.IsZero() {
		snapshotUpdatedAt = rendered.SnapshotUpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return map[string]any{
		"healthy":             hookHealthy(cfg, rendered, tokenPresent),
		"enabled":             cfg.Enabled,
		"providers":           cfg.Providers,
		"resolved_timezone":   rendered.ResolvedTimezone,
		"auth":                hookAuthStatusWithToken(cfg, tokenPresent),
		"source_gaps":         hookSourceGaps(cfg),
		"source_status":       rendered.Sources,
		"source_counts":       hookSourceCounts(cfg, rendered.Sources),
		"workspace_refreshed": rendered.Refreshed,
		"workspace_degraded":  rendered.WorkspaceDegraded,
		"snapshot_updated_at": snapshotUpdatedAt,
		"diagnostics":         rendered.Diagnostics,
		"note":                "Health output contains state and diagnostics only; cached Workspace content is never printed.",
	}
}

func hookHealthy(cfg pluginhooks.Config, rendered pluginhooks.RenderedContext, tokenPresent bool) bool {
	if !cfg.Enabled || !cfg.Providers.Workspace.Enabled {
		return true
	}
	if !tokenPresent {
		return false
	}
	if rendered.WorkspaceDegraded {
		return false
	}
	counts := hookSourceCounts(cfg, rendered.Sources)
	if counts["configured"] == 0 {
		return true
	}
	return counts["observed"] == counts["configured"] && counts["available"] == counts["configured"]
}

func hookSourceCounts(cfg pluginhooks.Config, statuses []pluginhooks.SourceStatus) map[string]int {
	configured := 0
	for _, enabled := range []bool{
		cfg.Sources.Calendar.Enabled,
		cfg.Sources.Gmail.Enabled,
		cfg.Sources.SentMail.Enabled,
		cfg.Sources.Tasks.Enabled,
		cfg.Sources.Chat.Enabled && len(cfg.Sources.Chat.Spaces) > 0,
	} {
		if enabled {
			configured++
		}
	}
	available := 0
	for _, status := range statuses {
		if status.Available {
			available++
		}
	}
	return map[string]int{
		"configured": configured,
		"observed":   len(statuses),
		"available":  available,
	}
}

func runHooksRun(cmd *cobra.Command, args []string) error {
	eventValue, _ := cmd.Flags().GetString("event")
	event := pluginhooks.Event(eventValue)
	if !event.Valid() {
		return nil
	}
	adapter, _ := cmd.Flags().GetString("adapter")
	if adapter != "codex" && adapter != "claude" {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	cfg, err := loadHookConfig(cwd)
	if err != nil {
		return nil
	}
	rendered := renderHook(cmd.Context(), event, cfg, cwd)
	return writeHookEnvelope(cmd, eventValue, rendered.AdditionalContext)
}

func renderHook(ctx context.Context, event pluginhooks.Event, cfg pluginhooks.Config, cwd string) pluginhooks.RenderedContext {
	return pluginhooks.Runtime{
		Run:          runSelfGWS,
		Store:        pluginhooks.FileStore{Path: hookSnapshotPath()},
		WorkspaceKey: hookWorkspaceKey(cwd),
	}.Render(ctx, event, cfg)
}

func hookWorkspaceKey(cwd string) string {
	path, err := filepath.Abs(cwd)
	if err != nil {
		path = cwd
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}

func runSelfGWS(ctx context.Context, args []string) ([]byte, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, self, args...)
	command.Env = append(os.Environ(), "GWS_NO_UPDATE_CHECK=1")
	return command.Output()
}

func loadHookConfig(cwd string) (pluginhooks.Config, error) {
	workspacePath := pluginhooks.WorkspaceConfigPath(cwd)
	if err := pluginhooks.ValidateWorkspaceConfig(workspacePath); err != nil {
		return pluginhooks.Config{}, err
	}
	return pluginhooks.LoadEffective(
		pluginhooks.UserConfigPath(config.GetConfigDir()),
		workspacePath,
		os.Getenv,
	)
}

func hookConfigPath(scope string) (string, error) {
	switch scope {
	case "user":
		return pluginhooks.UserConfigPath(config.GetConfigDir()), nil
	case "workspace":
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return pluginhooks.WorkspaceConfigPath(cwd), nil
	default:
		return "", fmt.Errorf("invalid --scope %q; expected user or workspace", scope)
	}
}

func hookSnapshotPath() string {
	dir := strings.TrimSpace(os.Getenv("PLUGIN_DATA"))
	if dir == "" {
		dir = strings.TrimSpace(os.Getenv("CLAUDE_PLUGIN_DATA"))
	}
	if dir == "" {
		if cacheDir, err := os.UserCacheDir(); err == nil {
			dir = filepath.Join(cacheDir, "gws", "hooks")
		} else {
			dir = filepath.Join(config.GetConfigDir(), "hooks-cache")
		}
	}
	return filepath.Join(dir, "workspace-snapshot.json")
}

func hookAuthStatus(cfg pluginhooks.Config) map[string]any {
	return hookAuthStatusWithToken(cfg, auth.TokenExists())
}

func hookAuthStatusWithToken(cfg pluginhooks.Config, tokenPresent bool) map[string]any {
	required := cfg.Enabled && cfg.Providers.Workspace.Enabled
	return map[string]any{
		"required_for_workspace": required,
		"token_present":          tokenPresent,
		"ready":                  !required || tokenPresent,
		"note":                   "Token presence is checked locally only; no token contents are read or displayed.",
	}
}

func hookSourceGaps(cfg pluginhooks.Config) []string {
	if !cfg.Enabled || !cfg.Providers.Workspace.Enabled {
		return []string{"Workspace provider is disabled; no Workspace sources will run."}
	}
	gaps := make([]string, 0, 3)
	if !cfg.Sources.Calendar.Enabled && !cfg.Sources.Gmail.Enabled && !cfg.Sources.SentMail.Enabled && !cfg.Sources.Tasks.Enabled && !cfg.Sources.Chat.Enabled {
		gaps = append(gaps, "All Workspace sources are disabled.")
	}
	if cfg.Sources.Chat.Enabled && len(cfg.Sources.Chat.Spaces) == 0 {
		gaps = append(gaps, "Chat is enabled but has no explicit spaces, so it will not be queried.")
	}
	if cfg.Sources.Tasks.Enabled && len(cfg.Sources.Tasks.ListIDs) == 0 {
		gaps = append(gaps, "Tasks will use bounded list discovery until task list IDs are configured.")
	}
	return gaps
}

func writeHookEnvelope(cmd *cobra.Command, event, context string) error {
	if context == "" {
		return nil
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     event,
			"additionalContext": context,
		},
	})
}

func writeHookOutput(value any) error {
	return GetPrinter().Print(value)
}
