package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAuthMode_Valid(t *testing.T) {
	tests := []struct {
		name            string
		mode, key, user string
		wantMode        string
	}{
		{"empty defaults to oauth", "", "", "", AuthModeOAuth},
		{"explicit oauth", "oauth", "", "", AuthModeOAuth},
		{"oauth case-insensitive", "OAuth", "", "", AuthModeOAuth},
		{"dwd complete", "service-account-dwd", "/tmp/key.json", "user@example.com", AuthModeDWD},
		{"app complete", "service-account-app", "/tmp/key.json", "", AuthModeApp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ResolveAuthMode(tt.mode, tt.key, tt.user)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", cfg.Mode, tt.wantMode)
			}
			if cfg.KeyFile != tt.key {
				t.Errorf("keyFile = %q, want %q", cfg.KeyFile, tt.key)
			}
		})
	}
}

func TestResolveAuthMode_Invalid(t *testing.T) {
	tests := []struct {
		name            string
		mode, key, user string
		wantErrContains string
	}{
		{"oauth with key file", "oauth", "/tmp/key.json", "", "service-account settings are present"},
		{"oauth with impersonation", "", "", "user@example.com", "service-account settings are present"},
		{"dwd without key", "service-account-dwd", "", "user@example.com", "requires service_account_key_file"},
		{"dwd without impersonation", "service-account-dwd", "/tmp/key.json", "", "requires impersonate_user"},
		{"app without key", "service-account-app", "", "", "requires service_account_key_file"},
		{"app with impersonation", "service-account-app", "/tmp/key.json", "user@example.com", "does not support impersonation"},
		{"unknown mode", "service-account", "/tmp/key.json", "", "unknown auth_mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveAuthMode(tt.mode, tt.key, tt.user)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

// writeTestServiceAccountKey writes a syntactically valid service-account
// JSON key (freshly generated test-only RSA key, never a real credential)
// and returns its path.
func writeTestServiceAccountKey(t *testing.T) string {
	t.Helper()

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	})

	key := map[string]string{
		"type":         "service_account",
		"project_id":   "test-project",
		"private_key":  string(keyPEM),
		"client_email": "gws-test@test-project.iam.gserviceaccount.com",
		"token_uri":    "https://oauth2.googleapis.com/token",
	}
	data, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "sa-key.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestServiceAccountTokenSource_DWDAndApp(t *testing.T) {
	keyFile := writeTestServiceAccountKey(t)

	dwd := &AuthModeConfig{Mode: AuthModeDWD, KeyFile: keyFile, ImpersonateUser: "user@example.com"}
	ts, err := ServiceAccountTokenSource(context.Background(), dwd, []string{"https://www.googleapis.com/auth/chat.messages.readonly"})
	if err != nil {
		t.Fatalf("DWD: unexpected error: %v", err)
	}
	if ts == nil {
		t.Fatal("DWD: nil token source")
	}

	app := &AuthModeConfig{Mode: AuthModeApp, KeyFile: keyFile}
	ts, err = ServiceAccountTokenSource(context.Background(), app, []string{"https://www.googleapis.com/auth/chat.app.messages.readonly"})
	if err != nil {
		t.Fatalf("app: unexpected error: %v", err)
	}
	if ts == nil {
		t.Fatal("app: nil token source")
	}
}

func TestServiceAccountTokenSource_Errors(t *testing.T) {
	keyFile := writeTestServiceAccountKey(t)
	scopes := []string{"https://www.googleapis.com/auth/chat.messages.readonly"}

	t.Run("oauth mode rejected", func(t *testing.T) {
		_, err := ServiceAccountTokenSource(context.Background(), &AuthModeConfig{Mode: AuthModeOAuth}, scopes)
		if err == nil || !strings.Contains(err.Error(), "requires auth_mode") {
			t.Errorf("expected mode error, got %v", err)
		}
	})

	t.Run("nil config rejected", func(t *testing.T) {
		_, err := ServiceAccountTokenSource(context.Background(), nil, scopes)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("empty scopes rejected", func(t *testing.T) {
		cfg := &AuthModeConfig{Mode: AuthModeDWD, KeyFile: keyFile, ImpersonateUser: "u@example.com"}
		_, err := ServiceAccountTokenSource(context.Background(), cfg, nil)
		if err == nil || !strings.Contains(err.Error(), "at least one scope") {
			t.Errorf("expected scope error, got %v", err)
		}
	})

	t.Run("missing key file", func(t *testing.T) {
		cfg := &AuthModeConfig{Mode: AuthModeDWD, KeyFile: filepath.Join(t.TempDir(), "missing.json"), ImpersonateUser: "u@example.com"}
		_, err := ServiceAccountTokenSource(context.Background(), cfg, scopes)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected not-found error, got %v", err)
		}
	})

	t.Run("invalid key file never echoes content", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		secret := `{"type":"service_account","private_key":"SECRET-MATERIAL"`
		if err := os.WriteFile(path, []byte(secret), 0600); err != nil {
			t.Fatal(err)
		}
		cfg := &AuthModeConfig{Mode: AuthModeApp, KeyFile: path}
		_, err := ServiceAccountTokenSource(context.Background(), cfg, scopes)
		if err == nil {
			t.Fatal("expected error for invalid key")
		}
		if strings.Contains(err.Error(), "SECRET-MATERIAL") {
			t.Error("error message must not include key file content")
		}
	})
}
