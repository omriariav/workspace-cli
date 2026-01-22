package config

import (
	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	Format       string `mapstructure:"format"`
}

// Keys for configuration values.
const (
	KeyClientID     = "client_id"
	KeyClientSecret = "client_secret"
	KeyFormat       = "format"
)

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
