package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/config"
	"golang.org/x/oauth2"
)

const (
	lockSuffix       = ".lock"
	lockTimeout      = 5 * time.Second
	lockPollInterval = 50 * time.Millisecond
	staleLockAge     = 30 * time.Second
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
// Uses atomic write (temp file + rename) and file locking for safety.
func SaveToken(token *oauth2.Token) error {
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tokenPath := config.GetTokenPath()

	unlock, err := acquireLock(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Atomic write: create temp file in same directory, then rename
	dir := filepath.Dir(tokenPath)
	tmp, err := os.CreateTemp(dir, ".token-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Set permissions before writing data
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, tokenPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// MergeToken merges an incoming token with an existing one, preserving the
// refresh token from the existing token if the incoming one lacks it.
func MergeToken(existing, incoming *oauth2.Token) *oauth2.Token {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming
	}

	if incoming.RefreshToken == "" && existing.RefreshToken != "" {
		incoming.RefreshToken = existing.RefreshToken
	}

	return incoming
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

// acquireLock creates a .lock file for the given path using O_CREATE|O_EXCL
// for cross-platform mutual exclusion. Returns an unlock function.
func acquireLock(path string) (unlock func(), err error) {
	lockPath := path + lockSuffix
	deadline := time.Now().Add(lockTimeout)

	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			// Write PID to lock file for debugging
			fmt.Fprintf(f, "%d", os.Getpid())
			f.Close()

			return func() {
				os.Remove(lockPath)
			}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("lock file error: %w", err)
		}

		// Check for stale lock
		if removeStaleLock(lockPath) {
			continue // Retry immediately after cleaning stale lock
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for lock: %s", lockPath)
		}

		time.Sleep(lockPollInterval)
	}
}

// removeStaleLock removes a lock file if it's older than staleLockAge.
// Returns true if the lock was removed.
func removeStaleLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil {
		return false
	}

	if time.Since(info.ModTime()) > staleLockAge {
		// Read PID for logging (best effort)
		if data, err := os.ReadFile(lockPath); err == nil {
			pidStr := strings.TrimSpace(string(data))
			if pid, err := strconv.Atoi(pidStr); err == nil {
				// Check if process is still running
				if p, err := os.FindProcess(pid); err == nil {
					// On Unix, FindProcess always succeeds, but Signal(0) checks existence
					if p.Signal(nil) == nil {
						return false // Process still running, lock is not stale
					}
				}
			}
		}

		os.Remove(lockPath)
		return true
	}

	return false
}
