package config

import (
	"fmt"
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

// AccessTokenFlagSet reports whether --access-token was given on the
// command line, whatever its value. Set by the cmd package alongside
// AccessTokenFlag.
var AccessTokenFlagSet func() bool

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

// CheckAccessToken rejects an external access token that was supplied but
// is empty or blank, for example because the command that was meant to
// produce it printed nothing. Treating that as "no token" would quietly
// run the command with the stored login or a service account instead: a
// different identity than the caller selected.
func CheckAccessToken() error {
	if AccessTokenFlagSet != nil && AccessTokenFlagSet() {
		if AccessTokenFlag == nil || strings.TrimSpace(AccessTokenFlag()) == "" {
			return fmt.Errorf("--access-token was given but is empty; not falling back to another credential. Pass a token, or remove the flag to use the stored login")
		}
		return nil
	}
	if v, ok := os.LookupEnv(AccessTokenEnv); ok && strings.TrimSpace(v) == "" {
		return fmt.Errorf("%s is set but empty; not falling back to another credential. Set it to a token, or unset it to use the stored login", AccessTokenEnv)
	}
	return nil
}

// ConfirmSensitiveEnv is the environment variable that sets the
// confirmation policy, overriding the config file.
const ConfirmSensitiveEnv = "GWS_CONFIRM_SENSITIVE"

// ParseConfirmSetting reads an on/off value for the confirmation gate.
// Only unambiguous spellings are accepted. Anything else (a typo such as
// "treu", or an empty value) is an error: for a safety switch, an
// unreadable setting must not quietly mean "off".
func ParseConfirmSetting(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a valid on/off value (use true or false)", v)
}

// ConfirmSensitivePolicy reports whether sensitive commands (send, share,
// delete) must be confirmed before they run. The environment variable
// wins over confirm_sensitive in the config file. An error means a value
// was given but could not be understood; the caller must then treat the
// policy as unknown rather than off.
func ConfirmSensitivePolicy() (bool, error) {
	if v, set := os.LookupEnv(ConfirmSensitiveEnv); set {
		on, err := ParseConfirmSetting(v)
		if err != nil {
			return false, fmt.Errorf("%s: %w", ConfirmSensitiveEnv, err)
		}
		return on, nil
	}
	if !viper.InConfig(KeyConfirmSensitive) {
		return false, nil
	}
	switch value := viper.Get(KeyConfirmSensitive).(type) {
	case bool:
		return value, nil
	case nil:
		return false, fmt.Errorf("%s in the config file has no value", KeyConfirmSensitive)
	default:
		on, err := ParseConfirmSetting(fmt.Sprint(value))
		if err != nil {
			return false, fmt.Errorf("%s in the config file: %w", KeyConfirmSensitive, err)
		}
		return on, nil
	}
}

// GetConfirmSensitive reports whether the confirmation gate is on. A
// setting that cannot be understood counts as on here; callers that can
// report the problem should use ConfirmSensitivePolicy.
func GetConfirmSensitive() bool {
	on, err := ConfirmSensitivePolicy()
	return on || err != nil
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
