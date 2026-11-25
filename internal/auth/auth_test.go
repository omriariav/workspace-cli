package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/omriariav/workspace-cli/internal/config"
	"golang.org/x/oauth2"
)

// Helper to set up a temp config directory for token tests
func setupTempConfigDir(t *testing.T) (cleanup func()) {
	t.Helper()
	tempDir := t.TempDir()
	original := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	return func() {
		os.Setenv("XDG_CONFIG_HOME", original)
	}
}

// --- Scope Tests ---

func TestAllScopes_NotEmpty(t *testing.T) {
	if len(AllScopes) == 0 {
		t.Error("AllScopes should not be empty")
	}
}

func TestAllScopes_ValidURLs(t *testing.T) {
	for _, scope := range AllScopes {
		if !strings.HasPrefix(scope, "https://www.googleapis.com/auth/") {
			t.Errorf("unexpected scope format: %s", scope)
		}
	}
}

func TestAllScopes_ContainsRequiredScopes(t *testing.T) {
	requiredPrefixes := []string{
		"gmail",
		"calendar",
		"drive",
		"documents",
		"spreadsheets",
		"presentations",
		"tasks",
		"userinfo",
	}

	for _, prefix := range requiredPrefixes {
		found := false
		for _, scope := range AllScopes {
			if strings.Contains(scope, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing scope for: %s", prefix)
		}
	}
}

func TestAllScopes_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, scope := range AllScopes {
		if seen[scope] {
			t.Errorf("duplicate scope: %s", scope)
		}
		seen[scope] = true
	}
}

func TestAllScopes_MatchesServiceScopesUnion(t *testing.T) {
	// AllScopes should be exactly the union of all ServiceScopes entries
	fromServices := make(map[string]bool)
	for _, scopes := range ServiceScopes {
		for _, s := range scopes {
			fromServices[scopePrefix+s] = true
		}
	}

	allScopesSet := make(map[string]bool)
	for _, s := range AllScopes {
		allScopesSet[s] = true
	}

	for s := range fromServices {
		if !allScopesSet[s] {
			t.Errorf("ServiceScopes contains %s but AllScopes does not", s)
		}
	}
	for s := range allScopesSet {
		if !fromServices[s] {
			t.Errorf("AllScopes contains %s but ServiceScopes does not", s)
		}
	}
}

func TestServiceScopes_ChatIncludesMemberships(t *testing.T) {
	chatScopes := ServiceScopes["chat"]
	found := false
	for _, s := range chatScopes {
		if s == "chat.memberships" {
			found = true
			break
		}
	}
	if !found {
		t.Error("chat service should include chat.memberships scope")
	}
}

func TestScopesForServices(t *testing.T) {
	scopes := ScopesForServices([]string{"gmail", "calendar"})

	expected := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/gmail.settings.basic",
		"https://www.googleapis.com/auth/gmail.settings.sharing",
		"https://www.googleapis.com/auth/calendar.readonly",
		"https://www.googleapis.com/auth/calendar.events",
		"https://www.googleapis.com/auth/calendar",
	}

	if len(scopes) != len(expected) {
		t.Fatalf("expected %d scopes, got %d: %v", len(expected), len(scopes), scopes)
	}

	scopeSet := make(map[string]bool)
	for _, s := range scopes {
		scopeSet[s] = true
	}
	for _, e := range expected {
		if !scopeSet[e] {
			t.Errorf("missing expected scope: %s", e)
		}
	}
}

func TestScopesForServices_AlwaysIncludesUserinfo(t *testing.T) {
	scopes := ScopesForServices([]string{"gmail"})
	found := false
	for _, s := range scopes {
		if s == "https://www.googleapis.com/auth/userinfo.email" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ScopesForServices should always include userinfo.email")
	}
}

func TestScopesForServices_UnknownService(t *testing.T) {
	scopes := ScopesForServices([]string{"nonexistent"})
	// Should still have userinfo
	if len(scopes) != 1 {
		t.Errorf("expected 1 scope (userinfo), got %d: %v", len(scopes), scopes)
	}
}

func TestServiceForScope(t *testing.T) {
	tests := []struct {
		scope string
		want  string
	}{
		{"https://www.googleapis.com/auth/gmail.readonly", "gmail"},
		{"https://www.googleapis.com/auth/calendar.events", "calendar"},
		{"https://www.googleapis.com/auth/chat.memberships", "chat"},
		{"https://www.googleapis.com/auth/userinfo.email", "userinfo"},
		{"https://www.googleapis.com/auth/unknown.scope", ""},
	}

	for _, tt := range tests {
		got := ServiceForScope(tt.scope)
		if got != tt.want {
			t.Errorf("ServiceForScope(%s) = %s, want %s", tt.scope, got, tt.want)
		}
	}
}

// --- Token Tests ---

func TestSaveToken_And_LoadToken(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	loaded, err := LoadToken()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("access token mismatch: got %s, want %s", loaded.AccessToken, token.AccessToken)
	}

	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("refresh token mismatch: got %s, want %s", loaded.RefreshToken, token.RefreshToken)
	}

	if loaded.TokenType != token.TokenType {
		t.Errorf("token type mismatch: got %s, want %s", loaded.TokenType, token.TokenType)
	}
}

func TestSaveToken_AtomicWrite_NoTmpResidue(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Check no .tmp files remain in the config dir
	configDir := os.Getenv("XDG_CONFIG_HOME")
	gwsDir := filepath.Join(configDir, "gws")
	entries, err := os.ReadDir(gwsDir)
	if err != nil {
		t.Fatalf("failed to read config dir: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("found leftover temp file: %s", entry.Name())
		}
	}
}

func TestSaveToken_Overwrite(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token1 := &oauth2.Token{AccessToken: "first", TokenType: "Bearer"}
	token2 := &oauth2.Token{AccessToken: "second", TokenType: "Bearer"}

	if err := SaveToken(token1); err != nil {
		t.Fatalf("failed to save first token: %v", err)
	}

	if err := SaveToken(token2); err != nil {
		t.Fatalf("failed to save second token: %v", err)
	}

	loaded, err := LoadToken()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}

	if loaded.AccessToken != "second" {
		t.Errorf("expected overwritten token, got: %s", loaded.AccessToken)
	}
}

func TestSaveToken_Permissions(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	tokenPath := filepath.Join(configDir, "gws", "token.json")

	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestLoadToken_NotExists(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	_, err := LoadToken()
	if err == nil {
		t.Error("expected error when token doesn't exist")
	}

	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("expected 'not authenticated' error, got: %v", err)
	}
}

func TestLoadToken_InvalidJSON(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	configDir := os.Getenv("XDG_CONFIG_HOME")
	gwsDir := filepath.Join(configDir, "gws")
	os.MkdirAll(gwsDir, 0700)

	tokenPath := filepath.Join(gwsDir, "token.json")
	os.WriteFile(tokenPath, []byte("not valid json"), 0600)

	_, err := LoadToken()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "parse") && !strings.Contains(errStr, "unmarshal") && !strings.Contains(errStr, "invalid") {
		t.Errorf("expected JSON parse-related error, got: %v", err)
	}
}

func TestDeleteToken(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	if !TokenExists() {
		t.Fatal("token should exist after saving")
	}

	if err := DeleteToken(); err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	if TokenExists() {
		t.Error("token should not exist after deletion")
	}
}

func TestDeleteToken_NotExists(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	err := DeleteToken()
	if err != nil {
		t.Errorf("unexpected error deleting non-existent token: %v", err)
	}
}

func TestTokenExists_True(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	if !TokenExists() {
		t.Error("TokenExists should return true when token exists")
	}
}

func TestTokenExists_False(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	if TokenExists() {
		t.Error("TokenExists should return false when token doesn't exist")
	}
}

func TestSaveToken_CreatesConfigDir(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	configDir := os.Getenv("XDG_CONFIG_HOME")
	gwsDir := filepath.Join(configDir, "gws")

	if _, err := os.Stat(gwsDir); !os.IsNotExist(err) {
		t.Skip("gws directory already exists")
	}

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	if _, err := os.Stat(gwsDir); os.IsNotExist(err) {
		t.Error("expected config dir to be created")
	}
}

func TestSaveToken_JSONFormat(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{
		AccessToken:  "access-123",
		TokenType:    "Bearer",
		RefreshToken: "refresh-456",
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	tokenPath := filepath.Join(configDir, "gws", "token.json")
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("saved token is not valid JSON: %v", err)
	}

	if !strings.Contains(string(data), "\n") {
		t.Error("expected token to be pretty-printed with newlines")
	}
}

// --- Lock Tests ---

func TestAcquireLock_BasicAcquireRelease(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-file")

	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Lock file should exist
	lockPath := path + lockSuffix
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should exist while held")
	}

	unlock()

	// Lock file should be removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should be removed after unlock")
	}
}

func TestAcquireLock_StaleLockCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-file")
	lockPath := path + lockSuffix

	// Create a stale lock file with a non-existent PID
	os.WriteFile(lockPath, []byte("999999999"), 0600)
	// Set modification time to the past
	past := time.Now().Add(-staleLockAge - time.Second)
	os.Chtimes(lockPath, past, past)

	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("should acquire lock after cleaning stale lock: %v", err)
	}
	unlock()
}

// --- MergeToken Tests ---

func TestMergeToken_PreservesRefresh(t *testing.T) {
	existing := &oauth2.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
	}
	incoming := &oauth2.Token{
		AccessToken:  "new-access",
		RefreshToken: "",
	}

	merged := MergeToken(existing, incoming)

	if merged.AccessToken != "new-access" {
		t.Errorf("expected new access token, got %s", merged.AccessToken)
	}
	if merged.RefreshToken != "old-refresh" {
		t.Errorf("expected preserved refresh token, got %s", merged.RefreshToken)
	}
}

func TestMergeToken_UsesNewRefresh(t *testing.T) {
	existing := &oauth2.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
	}
	incoming := &oauth2.Token{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
	}

	merged := MergeToken(existing, incoming)

	if merged.RefreshToken != "new-refresh" {
		t.Errorf("expected new refresh token, got %s", merged.RefreshToken)
	}
}

func TestMergeToken_NilExisting(t *testing.T) {
	incoming := &oauth2.Token{
		AccessToken: "new-access",
	}

	merged := MergeToken(nil, incoming)
	if merged != incoming {
		t.Error("expected incoming token when existing is nil")
	}
}

func TestMergeToken_NilIncoming(t *testing.T) {
	existing := &oauth2.Token{
		AccessToken: "old-access",
	}

	merged := MergeToken(existing, nil)
	if merged != existing {
		t.Error("expected existing token when incoming is nil")
	}
}

// --- Lock Timeout Test ---

func TestAcquireLock_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-file")

	// Hold the lock
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("failed to acquire initial lock: %v", err)
	}
	defer unlock()

	// Try to acquire again — should timeout
	_, err = acquireLock(path)
	if err == nil {
		t.Fatal("expected timeout error when lock is held")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAcquireLock_RecentLockNotStale(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-file")
	lockPath := path + lockSuffix

	// Create a recent lock (should NOT be cleaned as stale)
	os.WriteFile(lockPath, []byte("999999999"), 0600)

	// Should timeout, not clean the recent lock
	_, err := acquireLock(path)
	if err == nil {
		t.Fatal("expected timeout — recent lock should not be treated as stale")
	}

	// Lock file should still exist
	if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
		t.Error("recent lock file should not have been removed")
	}
}

// --- DeleteToken Lock Coordination Test ---

func TestDeleteToken_Locked(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	token := &oauth2.Token{AccessToken: "test", TokenType: "Bearer"}
	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Delete should succeed (acquires and releases its own lock)
	if err := DeleteToken(); err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	if TokenExists() {
		t.Error("token should not exist after deletion")
	}
}

// --- Validate Services Tests ---

func TestValidateServices_AllValid(t *testing.T) {
	unknown := ValidateServices([]string{"gmail", "calendar", "chat"})
	if len(unknown) != 0 {
		t.Errorf("expected no unknown services, got: %v", unknown)
	}
}

func TestValidateServices_SomeInvalid(t *testing.T) {
	unknown := ValidateServices([]string{"gmail", "fakesvc", "calendar", "nope"})
	if len(unknown) != 2 {
		t.Fatalf("expected 2 unknown services, got %d: %v", len(unknown), unknown)
	}

	has := func(s string) bool {
		for _, u := range unknown {
			if u == s {
				return true
			}
		}
		return false
	}
	if !has("fakesvc") || !has("nope") {
		t.Errorf("expected fakesvc and nope, got: %v", unknown)
	}
}

func TestValidateServices_Empty(t *testing.T) {
	unknown := ValidateServices(nil)
	if len(unknown) != 0 {
		t.Errorf("expected no unknown services for nil input, got: %v", unknown)
	}
}

// --- Granted Services Persistence Tests ---

func TestSaveAndLoadGrantedServices(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	// Ensure config dir exists
	if err := config.EnsureConfigDir(); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	services := []string{"gmail", "calendar", "chat"}
	if err := SaveGrantedServices(services); err != nil {
		t.Fatalf("failed to save granted services: %v", err)
	}

	loaded := LoadGrantedServices()
	if len(loaded) != len(services) {
		t.Fatalf("expected %d services, got %d", len(services), len(loaded))
	}
	for i, s := range services {
		if loaded[i] != s {
			t.Errorf("service %d: expected %s, got %s", i, s, loaded[i])
		}
	}
}

func TestLoadGrantedServices_NotExists(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	loaded := LoadGrantedServices()
	if loaded != nil {
		t.Errorf("expected nil for non-existent file, got: %v", loaded)
	}
}

func TestReadOnlyScopesForServices(t *testing.T) {
	scopes := ReadOnlyScopesForServices([]string{"gmail", "drive"})
	expected := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	}
	if len(scopes) != len(expected) {
		t.Fatalf("expected %d scopes, got %d: %v", len(expected), len(scopes), scopes)
	}
	for i, want := range expected {
		if scopes[i] != want {
			t.Errorf("scope %d = %s, want %s", i, scopes[i], want)
		}
	}
}

func TestReadOnlyServiceScopes_CoverEveryService(t *testing.T) {
	for svc := range ServiceScopes {
		ro, ok := ReadOnlyServiceScopes[svc]
		if !ok || len(ro) == 0 {
			t.Errorf("service %q has no read-only scopes", svc)
		}
	}
	for svc := range ReadOnlyServiceScopes {
		if _, ok := ServiceScopes[svc]; !ok {
			t.Errorf("read-only table has unknown service %q", svc)
		}
	}
}

func TestReadOnlyServiceScopes_GrantNoWriteAccess(t *testing.T) {
	// Scopes that are read-only without carrying a ".readonly" suffix.
	allowed := map[string]bool{"userinfo.email": true, "userinfo.profile": true}
	for svc, scopes := range ReadOnlyServiceScopes {
		for _, s := range scopes {
			if !strings.HasSuffix(s, ".readonly") && !allowed[s] {
				t.Errorf("service %q read-only scope %q is not read-only", svc, s)
			}
		}
	}
}

func TestReadOnlyEventsScopes_OmitUnsupportedDriveScope(t *testing.T) {
	// The Workspace Events API does not accept drive.metadata.readonly for
	// Drive subscriptions, so requesting it would be dead weight.
	for _, s := range ReadOnlyServiceScopes["events"] {
		if strings.HasPrefix(s, "drive.") {
			t.Errorf("read-only events scopes should not include %q", s)
		}
	}
}

func TestServiceForScope_ResolvesReadOnlyOnlyScopes(t *testing.T) {
	if got := ServiceForScope("https://www.googleapis.com/auth/spreadsheets.readonly"); got != "sheets" {
		t.Errorf("ServiceForScope(spreadsheets.readonly) = %q, want sheets", got)
	}
}

func TestGrantedAccess_RoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := LoadGrantedAccess(); got != nil {
		t.Errorf("expected nil before any login, got %v", got)
	}
	tok := &oauth2.Token{AccessToken: "a1", RefreshToken: "refresh-1"}
	want := map[string]string{"gmail": AccessReadOnly, "drive": AccessFull}
	wantScopes := []string{"scope-a", "scope-b"}
	if err := SaveGrantedAccess(want, wantScopes, "acct-1", tok); err != nil {
		t.Fatalf("SaveGrantedAccess: %v", err)
	}

	// Metadata without a stored token is ignored.
	if got := LoadGrantedAccess(); got != nil {
		t.Errorf("expected nil without a stored token, got %v", got)
	}
	if err := SaveToken(tok); err != nil {
		t.Fatal(err)
	}
	got := LoadGrantedAccess()
	if len(got) != 2 || got["gmail"] != AccessReadOnly || got["drive"] != AccessFull {
		t.Errorf("round trip = %v", got)
	}
	if account := LoadGrantedAccount(); account != "acct-1" {
		t.Errorf("LoadGrantedAccount = %q", account)
	}
	// Bound to a specific token: the right one sees it, another does not,
	// whatever happens to be on disk.
	if got := LoadGrantedAccessFor(tok); got["gmail"] != AccessReadOnly {
		t.Errorf("LoadGrantedAccessFor(matching token) = %v", got)
	}
	if got := LoadGrantedAccessFor(&oauth2.Token{RefreshToken: "someone-else"}); got != nil {
		t.Errorf("LoadGrantedAccessFor(other token) = %v, want nil", got)
	}
	if got := LoadGrantedAccessFor(nil); got != nil {
		t.Errorf("LoadGrantedAccessFor(nil) = %v, want nil", got)
	}
	if scopes := LoadGrantedScopes(); strings.Join(scopes, ",") != "scope-a,scope-b" {
		t.Errorf("LoadGrantedScopes = %v", scopes)
	}

	// An access-token refresh keeps the refresh token: still valid.
	if err := SaveToken(&oauth2.Token{AccessToken: "a2", RefreshToken: "refresh-1"}); err != nil {
		t.Fatal(err)
	}
	if got := LoadGrantedAccess(); len(got) != 2 {
		t.Errorf("metadata should survive an access-token refresh, got %v", got)
	}

	if got := strings.Join(GrantedServicesForToken(), ","); got != "gmail,drive" {
		t.Errorf("GrantedServicesForToken = %q, want the token-bound record", got)
	}

	// A different login's token: the metadata describes another grant
	// (interrupted write, or two logins racing) and must be ignored.
	if err := SaveToken(&oauth2.Token{AccessToken: "b1", RefreshToken: "refresh-2"}); err != nil {
		t.Fatal(err)
	}
	if got := LoadGrantedAccess(); got != nil {
		t.Errorf("metadata for another token must be ignored, got %v", got)
	}
	if scopes := LoadGrantedScopes(); scopes != nil {
		t.Errorf("scopes for another token must be ignored, got %v", scopes)
	}
	// The plain service list beside a mismatched record came from the same
	// other login: it must not be used either.
	if err := SaveGrantedServices([]string{"tasks"}); err != nil {
		t.Fatal(err)
	}
	if got := GrantedServicesForToken(); got != nil {
		t.Errorf("GrantedServicesForToken with a stale record = %v, want nil", got)
	}
	// With no access record at all (a login from an older version), the
	// plain list is the only metadata there is.
	if err := os.Remove(config.GetGrantedAccessPath()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(GrantedServicesForToken(), ","); got != "tasks" {
		t.Errorf("GrantedServicesForToken legacy fallback = %q, want tasks", got)
	}
}

func TestStoredTokenFingerprint_ChangesOnlyWhenTheLoginIsReplaced(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if StoredTokenFingerprint() != "" {
		t.Error("nothing stored should give an empty fingerprint")
	}
	if err := SaveToken(&oauth2.Token{AccessToken: "a1", RefreshToken: "login-1"}); err != nil {
		t.Fatal(err)
	}
	base := StoredTokenFingerprint()
	if base == "" {
		t.Fatal("a stored login should have a fingerprint")
	}
	// An access-token refresh is the same login.
	if err := SaveToken(&oauth2.Token{AccessToken: "a2", RefreshToken: "login-1"}); err != nil {
		t.Fatal(err)
	}
	if StoredTokenFingerprint() != base {
		t.Error("an access-token refresh must not look like a new login")
	}
	// Another login finishing in the meantime is detected.
	if err := SaveToken(&oauth2.Token{AccessToken: "b1", RefreshToken: "login-2"}); err != nil {
		t.Fatal(err)
	}
	if StoredTokenFingerprint() == base {
		t.Error("a replaced login must change the fingerprint")
	}
}

func TestTokenRefreshed(t *testing.T) {
	stored := &oauth2.Token{AccessToken: "a1", RefreshToken: "r1"}
	cases := []struct {
		name    string
		current *oauth2.Token
		want    bool
	}{
		{"unchanged", &oauth2.Token{AccessToken: "a1", RefreshToken: "r1"}, false},
		{"refresh token omitted from response", &oauth2.Token{AccessToken: "a1"}, false},
		{"new access token", &oauth2.Token{AccessToken: "a2", RefreshToken: "r1"}, true},
		{"rotated refresh token, same access token", &oauth2.Token{AccessToken: "a1", RefreshToken: "r2"}, true},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := TokenRefreshed(stored, c.current); got != c.want {
			t.Errorf("%s: TokenRefreshed = %v, want %v", c.name, got, c.want)
		}
	}
	if TokenRefreshed(nil, stored) {
		t.Error("nil stored token cannot have been refreshed")
	}
}

func TestSaveRefreshedToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stored := &oauth2.Token{AccessToken: "a1", RefreshToken: "login-1"}
	if err := SaveToken(stored); err != nil {
		t.Fatal(err)
	}

	// Ordinary refresh: saved, and the refresh token is kept even when
	// the refresh response omits it.
	if err := SaveRefreshedToken(stored, &oauth2.Token{AccessToken: "a2"}); err != nil {
		t.Fatal(err)
	}
	got, _ := LoadToken()
	if got.AccessToken != "a2" || got.RefreshToken != "login-1" {
		t.Errorf("after refresh = %+v", got)
	}

	// A read-only login replaces the authorization while an API command
	// that started earlier is still refreshing the old, wider token.
	if err := SaveToken(&oauth2.Token{AccessToken: "ro1", RefreshToken: "login-2-readonly"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveRefreshedToken(stored, &oauth2.Token{AccessToken: "a3", RefreshToken: "login-1"}); err != nil {
		t.Fatal(err)
	}
	got, _ = LoadToken()
	if got.AccessToken != "ro1" || got.RefreshToken != "login-2-readonly" {
		t.Errorf("a stale refresh overwrote the new login: %+v", got)
	}

	// After logout there is nothing to refresh into.
	if err := DeleteToken(); err != nil {
		t.Fatal(err)
	}
	if err := SaveRefreshedToken(stored, &oauth2.Token{AccessToken: "a4", RefreshToken: "login-1"}); err != nil {
		t.Fatal(err)
	}
	if TokenExists() {
		t.Error("a refresh must not bring back a token after logout")
	}
	// Refresh-token rotation: the provider hands back a new refresh token
	// for the same grant. It must be saved (the old one may be dead), and
	// the access record must follow it.
	current := &oauth2.Token{AccessToken: "c1", RefreshToken: "rotating-1"}
	if err := SaveToken(current); err != nil {
		t.Fatal(err)
	}
	if err := SaveGrantedAccess(map[string]string{"gmail": AccessReadOnly}, []string{"scope-x"}, "acct-9", current); err != nil {
		t.Fatal(err)
	}
	if err := SaveRefreshedToken(current, &oauth2.Token{AccessToken: "c2", RefreshToken: "rotating-2"}); err != nil {
		t.Fatal(err)
	}
	got, _ = LoadToken()
	if got.AccessToken != "c2" || got.RefreshToken != "rotating-2" {
		t.Errorf("rotated token not saved: %+v", got)
	}
	if access := LoadGrantedAccess(); access["gmail"] != AccessReadOnly {
		t.Errorf("access record did not follow the rotated token: %v", access)
	}
	if scopes := LoadGrantedScopes(); len(scopes) != 1 || scopes[0] != "scope-x" || LoadGrantedAccount() != "acct-9" {
		t.Errorf("scopes/account did not follow the rotated token: %v / %q", scopes, LoadGrantedAccount())
	}
	// Interrupted rotation: the record was rewritten for the new token but
	// the process stopped before the token itself was saved. The old token
	// is still on disk and must still have its record.
	if err := SaveToken(&oauth2.Token{AccessToken: "c2", RefreshToken: "rotating-1"}); err != nil {
		t.Fatal(err)
	}
	if access := LoadGrantedAccess(); access["gmail"] != AccessReadOnly {
		t.Errorf("after an interrupted rotation the old token lost its record: %v", access)
	}
	// ...and completing it later (the token write) keeps the record too.
	if err := SaveToken(&oauth2.Token{AccessToken: "c2", RefreshToken: "rotating-2"}); err != nil {
		t.Fatal(err)
	}
	if access := LoadGrantedAccess(); access["gmail"] != AccessReadOnly {
		t.Errorf("after the rotation completes the record is missing: %v", access)
	}
	// An unrelated token matches neither fingerprint.
	if err := SaveToken(&oauth2.Token{AccessToken: "x", RefreshToken: "someone-else"}); err != nil {
		t.Fatal(err)
	}
	if LoadGrantedAccess() != nil {
		t.Error("a record mid-rotation must still be ignored for an unrelated token")
	}
	if err := SaveToken(&oauth2.Token{AccessToken: "c2", RefreshToken: "rotating-2"}); err != nil {
		t.Fatal(err)
	}

	// A second command still holding the pre-rotation token must not put
	// it back.
	if err := SaveRefreshedToken(current, &oauth2.Token{AccessToken: "c3", RefreshToken: "rotating-1"}); err != nil {
		t.Fatal(err)
	}
	if got, _ = LoadToken(); got.RefreshToken != "rotating-2" {
		t.Errorf("a stale holder overwrote the rotated token: %+v", got)
	}

	// Nil arguments are a no-op.
	if err := SaveRefreshedToken(nil, stored); err != nil {
		t.Errorf("nil stored: %v", err)
	}
}

func TestTokenFingerprint(t *testing.T) {
	if tokenFingerprint(nil) != "" || tokenFingerprint(&oauth2.Token{AccessToken: "a"}) != "" {
		t.Error("no refresh token means no fingerprint")
	}
	a := tokenFingerprint(&oauth2.Token{RefreshToken: "refresh-1"})
	b := tokenFingerprint(&oauth2.Token{RefreshToken: "refresh-2"})
	if a == "" || a == b || strings.Contains(a, "refresh") {
		t.Errorf("fingerprints: %q %q", a, b)
	}
}

func TestAuthURLOptions_NeverAsksForPreviouslyGrantedScopes(t *testing.T) {
	c := NewOAuthClient("id", "secret", []string{"scope-a"})
	url := c.config.AuthCodeURL("state", c.authURLOptions("challenge")...)
	if strings.Contains(url, "include_granted_scopes") {
		t.Errorf("login must not pull in scopes from earlier grants: %s", url)
	}
	if !strings.Contains(url, "code_challenge=challenge") || !strings.Contains(url, "prompt=consent") || !strings.Contains(url, "access_type=offline") {
		t.Errorf("auth URL lost PKCE/consent/offline params: %s", url)
	}
}
