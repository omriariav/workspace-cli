package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	ClientID          string   `mapstructure:"client_id"`
	ClientSecret      string   `mapstructure:"client_secret"`
	Format            string   `mapstructure:"format"`
	Services          []string `mapstructure:"services"`
	AuthMode          string   `mapstructure:"auth_mode"`
	ServiceAccountKey string   `mapstructure:"service_account_key_file"`
	ImpersonateUser   string   `mapstructure:"impersonate_user"`
}

// Keys for configuration values.
const (
	KeyClientID          = "client_id"
	KeyClientSecret      = "client_secret"
	KeyFormat            = "format"
	KeyServices          = "services"
	KeyAuthMode          = "auth_mode"
	KeyServiceAccountKey = "service_account_key_file"
	KeyImpersonateUser   = "impersonate_user"
	KeyConfirmSensitive  = "confirm_sensitive"
)

// AccessTokenEnv is the environment variable that carries an externally
// managed access token.
const AccessTokenEnv = "GWS_ACCESS_TOKEN"

// AccessTokenFlag returns the value of the --access-token flag. The cmd
// package sets it; it is nil where there is no flag (library use, tests).
var AccessTokenFlag func() string

// GetAccessToken returns an externally managed OAuth access token. When
// set, gws uses it as-is: no login, no refresh.
//
// Only the --access-token flag and the GWS_ACCESS_TOKEN environment
// variable are read. The config file is deliberately not a source: a
// token left in config.yaml would silently take precedence over the
// stored login or a service account long after it was put there.
func GetAccessToken() string {
	if AccessTokenFlag != nil {
		if v := strings.TrimSpace(AccessTokenFlag()); v != "" {
			return v
		}
	}
	return strings.TrimSpace(os.Getenv(AccessTokenEnv))
}

// GetConfirmSensitive reports whether sensitive commands (send, share,
// delete) must be confirmed before they run (GWS_CONFIRM_SENSITIVE or
// confirm_sensitive in the config file).
func GetConfirmSensitive() bool {
	return viper.GetBool(KeyConfirmSensitive)
}

// GetClientID returns the OAuth client ID from config or environment.
func GetClientID() string {
	return viper.GetString(KeyClientID)
}

// GetClientSecret returns the OAuth client secret from config or environment.
func GetClientSecret() string {
	return viper.GetString(KeyClientSecret)
}

// GetFormat returns the output format (json or text).
func GetFormat() string {
	format := viper.GetString(KeyFormat)
	if format == "" {
		return "json"
	}
	return format
}

// GetServices returns the list of services from config.
func GetServices() []string {
	return viper.GetStringSlice(KeyServices)
}

// GetAuthMode returns the configured authentication mode. Empty means the
// default interactive OAuth flow.
func GetAuthMode() string {
	return viper.GetString(KeyAuthMode)
}

// GetServiceAccountKeyFile returns the path to the service-account JSON key
// file. Only the path is ever surfaced; key material is never read into
// config or printed.
func GetServiceAccountKeyFile() string {
	return viper.GetString(KeyServiceAccountKey)
}

// GetImpersonateUser returns the Workspace user email to impersonate under
// domain-wide delegation.
func GetImpersonateUser() string {
	return viper.GetString(KeyImpersonateUser)
}

// SetDefaults sets default configuration values.
func SetDefaults() {
	viper.SetDefault(KeyFormat, "json")
}

// Load loads the configuration from all sources.
func Load() (*Config, error) {
	SetDefaults()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
