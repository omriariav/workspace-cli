package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

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

func TestSaveToken_And_LoadToken(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	// Create a test token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Save it
	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Load it back
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

	// Check file permissions
	configDir := os.Getenv("XDG_CONFIG_HOME")
	tokenPath := filepath.Join(configDir, "gws", "token.json")

	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}

	// Should be 0600 (owner read/write only)
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

	// Create config dir and write invalid JSON
	configDir := os.Getenv("XDG_CONFIG_HOME")
	gwsDir := filepath.Join(configDir, "gws")
	os.MkdirAll(gwsDir, 0700)

	tokenPath := filepath.Join(gwsDir, "token.json")
	os.WriteFile(tokenPath, []byte("not valid json"), 0600)

	_, err := LoadToken()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Verify the error indicates a parsing issue
	// Note: We check for "parse" as the error wraps json.Unmarshal errors
	errStr := err.Error()
	if !strings.Contains(errStr, "parse") && !strings.Contains(errStr, "unmarshal") && !strings.Contains(errStr, "invalid") {
		t.Errorf("expected JSON parse-related error, got: %v", err)
	}
}

func TestDeleteToken(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	// First save a token
	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	if err := SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Verify it exists
	if !TokenExists() {
		t.Fatal("token should exist after saving")
	}

	// Delete it
	if err := DeleteToken(); err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify it's gone
	if TokenExists() {
		t.Error("token should not exist after deletion")
	}
}

func TestDeleteToken_NotExists(t *testing.T) {
	cleanup := setupTempConfigDir(t)
	defer cleanup()

	// Should not error when token doesn't exist
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

	// Verify directory doesn't exist yet
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

	// Verify directory was created
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

	// Read the raw file
	configDir := os.Getenv("XDG_CONFIG_HOME")
	tokenPath := filepath.Join(configDir, "gws", "token.json")
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("saved token is not valid JSON: %v", err)
	}

	// Verify it's indented (pretty printed)
	if !strings.Contains(string(data), "\n") {
		t.Error("expected token to be pretty-printed with newlines")
	}
}
