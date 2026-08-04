package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Authentication modes. The default (empty or "oauth") is the interactive
// OAuth2 + PKCE flow. The two service-account modes are explicit and
// mutually exclusive:
//
//   - AuthModeDWD uses a service-account key with domain-wide delegation,
//     acting as a named Workspace user (impersonation subject).
//   - AuthModeApp uses a service-account key as an admin-approved Google
//     Chat app, without impersonating any user.
const (
	AuthModeOAuth = "oauth"
	AuthModeDWD   = "service-account-dwd"
	AuthModeApp   = "service-account-app"
)

// ValidAuthModes lists the accepted auth_mode values.
func ValidAuthModes() []string {
	return []string{AuthModeOAuth, AuthModeDWD, AuthModeApp}
}

// AuthModeConfig is the resolved, validated authentication configuration.
type AuthModeConfig struct {
	Mode            string
	KeyFile         string // path only; key material is never held here
	ImpersonateUser string
}

// ResolveAuthMode validates the auth-mode configuration fail-closed and
// returns the resolved config. Every rejected combination gets an
// actionable error that names the offending keys, without ever including
// key material.
func ResolveAuthMode(mode, keyFile, impersonateUser string) (*AuthModeConfig, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		normalized = AuthModeOAuth
	}

	switch normalized {
	case AuthModeOAuth:
		if keyFile != "" || impersonateUser != "" {
			return nil, fmt.Errorf("auth_mode is %q but service-account settings are present: unset service_account_key_file/impersonate_user (GWS_SERVICE_ACCOUNT_KEY_FILE/GWS_IMPERSONATE_USER), or set auth_mode to %q or %q",
				AuthModeOAuth, AuthModeDWD, AuthModeApp)
		}
		return &AuthModeConfig{Mode: AuthModeOAuth}, nil

	case AuthModeDWD:
		if keyFile == "" {
			return nil, fmt.Errorf("auth_mode %q requires service_account_key_file (GWS_SERVICE_ACCOUNT_KEY_FILE) pointing to a service-account JSON key with domain-wide delegation enabled", AuthModeDWD)
		}
		if impersonateUser == "" {
			return nil, fmt.Errorf("auth_mode %q requires impersonate_user (GWS_IMPERSONATE_USER): the Workspace user email the service account acts as", AuthModeDWD)
		}
		return &AuthModeConfig{Mode: AuthModeDWD, KeyFile: keyFile, ImpersonateUser: impersonateUser}, nil

	case AuthModeApp:
		if keyFile == "" {
			return nil, fmt.Errorf("auth_mode %q requires service_account_key_file (GWS_SERVICE_ACCOUNT_KEY_FILE) pointing to the Chat app's service-account JSON key", AuthModeApp)
		}
		if impersonateUser != "" {
			return nil, fmt.Errorf("auth_mode %q does not support impersonation: unset impersonate_user (GWS_IMPERSONATE_USER), or use auth_mode %q to act as a Workspace user", AuthModeApp, AuthModeDWD)
		}
		return &AuthModeConfig{Mode: AuthModeApp, KeyFile: keyFile}, nil

	default:
		return nil, fmt.Errorf("unknown auth_mode %q: valid modes are %s", mode, strings.Join(ValidAuthModes(), ", "))
	}
}

// ServiceAccountTokenSource builds an oauth2.TokenSource from a
// service-account JSON key file for the given scopes. For AuthModeDWD the
// impersonated user is set as the JWT subject; AuthModeApp mints app
// credentials with no subject. The key file is read here and never logged
// or echoed back in errors.
func ServiceAccountTokenSource(ctx context.Context, cfg *AuthModeConfig, scopes []string) (oauth2.TokenSource, error) {
	if cfg == nil || (cfg.Mode != AuthModeDWD && cfg.Mode != AuthModeApp) {
		return nil, fmt.Errorf("service-account token source requires auth_mode %q or %q", AuthModeDWD, AuthModeApp)
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("service-account token source requires at least one scope")
	}

	keyJSON, err := os.ReadFile(cfg.KeyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("service-account key file not found: %s", cfg.KeyFile)
		}
		return nil, fmt.Errorf("failed to read service-account key file %s: %w", cfg.KeyFile, err)
	}

	jwtCfg, err := google.JWTConfigFromJSON(keyJSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("invalid service-account key file %s: not a valid service-account JSON key", cfg.KeyFile)
	}

	if cfg.Mode == AuthModeDWD {
		jwtCfg.Subject = cfg.ImpersonateUser
	}

	return jwtCfg.TokenSource(ctx), nil
}
