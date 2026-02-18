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
		"https://www.googleapis.com/auth/calendar.readonly",
		"https://www.googleapis.com/auth/calendar.events",
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
