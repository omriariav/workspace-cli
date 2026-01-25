package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGetConfigDir_Default(t *testing.T) {
	// Clear XDG_CONFIG_HOME to test default behavior
	original := os.Getenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	dir := GetConfigDir()

	// Should end with .config/gws
	if !strings.HasSuffix(dir, filepath.Join(".config", "gws")) {
		t.Errorf("expected path to end with .config/gws, got: %s", dir)
	}
}

func TestGetConfigDir_XDGConfigHome(t *testing.T) {
	// Set XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	dir := GetConfigDir()

	expected := filepath.Join("/custom/config", "gws")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetTokenPath(t *testing.T) {
	path := GetTokenPath()

	if !strings.HasSuffix(path, "token.json") {
		t.Errorf("expected path to end with token.json, got: %s", path)
	}

	// Should be inside config dir
	configDir := GetConfigDir()
	if !strings.HasPrefix(path, configDir) {
		t.Errorf("expected path to be inside config dir, got: %s", path)
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()

	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("expected path to end with config.yaml, got: %s", path)
	}

	// Should be inside config dir
	configDir := GetConfigDir()
	if !strings.HasPrefix(path, configDir) {
		t.Errorf("expected path to be inside config dir, got: %s", path)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()
	original := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", original)

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory was created
	configDir := GetConfigDir()
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected config dir to be a directory")
	}

	// Verify permissions (0700)
	if info.Mode().Perm() != 0700 {
		t.Errorf("expected permissions 0700, got %o", info.Mode().Perm())
	}
}

func TestGetFormat_Default(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	format := GetFormat()
	if format != "json" {
		t.Errorf("expected default format 'json', got '%s'", format)
	}
}

func TestGetFormat_Set(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set(KeyFormat, "text")
	format := GetFormat()
	if format != "text" {
		t.Errorf("expected format 'text', got '%s'", format)
	}
}

func TestGetClientID(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set(KeyClientID, "test-client-id")
	id := GetClientID()
	if id != "test-client-id" {
		t.Errorf("expected 'test-client-id', got '%s'", id)
	}
}

func TestGetClientSecret(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set(KeyClientSecret, "test-secret")
	secret := GetClientSecret()
	if secret != "test-secret" {
		t.Errorf("expected 'test-secret', got '%s'", secret)
	}
}

func TestSetDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	SetDefaults()
	format := viper.GetString(KeyFormat)
	if format != "json" {
		t.Errorf("expected default format 'json', got '%s'", format)
	}
}

func TestLoad(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set(KeyClientID, "load-test-id")
	viper.Set(KeyClientSecret, "load-test-secret")
	viper.Set(KeyFormat, "text")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ClientID != "load-test-id" {
		t.Errorf("expected ClientID 'load-test-id', got '%s'", cfg.ClientID)
	}

	if cfg.ClientSecret != "load-test-secret" {
		t.Errorf("expected ClientSecret 'load-test-secret', got '%s'", cfg.ClientSecret)
	}

	if cfg.Format != "text" {
		t.Errorf("expected Format 'text', got '%s'", cfg.Format)
	}
}

func TestConfigConstants(t *testing.T) {
	// Verify constants are set correctly
	if KeyClientID != "client_id" {
		t.Errorf("unexpected KeyClientID: %s", KeyClientID)
	}
	if KeyClientSecret != "client_secret" {
		t.Errorf("unexpected KeyClientSecret: %s", KeyClientSecret)
	}
	if KeyFormat != "format" {
		t.Errorf("unexpected KeyFormat: %s", KeyFormat)
	}
}
