package config

import (
	"os"
	"path/filepath"
)

const (
	appName       = "gws"
	configDirName = "gws"
	tokenFileName = "token.json"
	configName    = "config.yaml"
)

// GetConfigDir returns the configuration directory path.
// On macOS/Linux: ~/.config/gws/
// On Windows: %APPDATA%/gws/
func GetConfigDir() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, configDirName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", configDirName)
	}

	return filepath.Join(homeDir, ".config", configDirName)
}

// GetTokenPath returns the full path to the token file.
func GetTokenPath() string {
	return filepath.Join(GetConfigDir(), tokenFileName)
}

// GetConfigPath returns the full path to the config file.
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), configName)
}

// EnsureConfigDir creates the configuration directory if it doesn't exist.
func EnsureConfigDir() error {
	configDir := GetConfigDir()
	return os.MkdirAll(configDir, 0700)
}
