package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/config"
	"golang.org/x/oauth2"
)

// LoadToken loads the OAuth token from the token file.
func LoadToken() (*oauth2.Token, error) {
	tokenPath := config.GetTokenPath()

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not authenticated, run: gws auth login")
		}
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return &token, nil
}

// SaveToken saves the OAuth token to the token file with secure permissions.
func SaveToken(token *oauth2.Token) error {
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tokenPath := config.GetTokenPath()

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with restrictive permissions (0600 = owner read/write only)
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// DeleteToken removes the token file.
func DeleteToken() error {
	tokenPath := config.GetTokenPath()

	if err := os.Remove(tokenPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone, that's fine
		}
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}

// TokenExists checks if a token file exists.
func TokenExists() bool {
	tokenPath := config.GetTokenPath()
	_, err := os.Stat(tokenPath)
	return err == nil
}
