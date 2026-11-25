package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
// Uses file locking to coordinate with SaveToken.
func DeleteToken() error {
	tokenPath := config.GetTokenPath()

	// Check if the token file even exists before locking
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return nil // Already gone
	}

	unlock, err := acquireLock(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for deletion: %w", err)
	}
	defer unlock()

	if err := os.Remove(tokenPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Removed between stat and lock
		}
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}

// SaveGrantedServices persists the list of services that were granted during login.
func SaveGrantedServices(services []string) error {
	path := config.GetGrantedServicesPath()
	data, err := json.Marshal(services)
	if err != nil {
		return fmt.Errorf("failed to marshal granted services: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// LoadGrantedServices loads the persisted list of granted services.
func LoadGrantedServices() []string {
	path := config.GetGrantedServicesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var services []string
	if err := json.Unmarshal(data, &services); err != nil {
		return nil
	}
	return services
}

// SaveGrantedAccess persists the access level ("full" or "readonly") granted
// to each service at login time. It is local metadata only: the token's real
// scopes are always enforced by Google.
//
// The record carries a fingerprint of the token it describes, so metadata
// left behind by another login (an interrupted write, two logins racing)
// is recognised as stale and ignored on load.
//
// scopes is the exact scope list the token was granted. It is what later
// logins compare against: access levels are a summary for people, and two
// different scope sets can share a level.
//
// account is the stable Google account ID the token belongs to ("" if it
// could not be determined).
func SaveGrantedAccess(access map[string]string, scopes []string, account string, token *oauth2.Token) error {
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	data, err := json.Marshal(grantedAccessRecord{Token: tokenFingerprint(token), Services: access, Scopes: scopes, Account: account})
	if err != nil {
		return fmt.Errorf("failed to marshal granted access: %w", err)
	}
	// Atomic replace, so a failed or interrupted write never leaves a
	// truncated or half-old file behind.
	path := config.GetGrantedAccessPath()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".granted-access-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
		if cerr := tmp.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Rename(tmpPath, path)
		}
		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to write granted access: %w", err)
		}
		return nil
	}
	tmp.Close()
	os.Remove(tmpPath)
	return fmt.Errorf("failed to set temp file permissions")
}

// GrantedServicesForToken returns the services granted to the stored
// token. It prefers the token-bound access record, which cannot describe
// another login, and falls back to the plain service list written by
// versions that did not record access levels.
func GrantedServicesForToken() []string {
	// An access record that exists but does not match the stored token
	// was written by another login, and so was the plain service list
	// beside it: use neither.
	if _, err := os.Stat(config.GetGrantedAccessPath()); err == nil {
		access := LoadGrantedAccess()
		if access == nil {
			return nil
		}
		var services []string
		for _, svc := range ValidServiceNames() {
			if _, ok := access[svc]; ok {
				services = append(services, svc)
			}
		}
		return services
	}
	return LoadGrantedServices()
}

// grantedAccessRecord is the on-disk form of the access metadata.
type grantedAccessRecord struct {
	// Token is a fingerprint of the refresh token the levels describe.
	Token    string            `json:"token"`
	Services map[string]string `json:"services"`
	// Scopes is the exact scope list granted to that token.
	Scopes []string `json:"scopes,omitempty"`
	// Account is the stable Google account ID the token belongs to.
	Account string `json:"account,omitempty"`
}

// tokenFingerprint identifies a stored authorization without revealing
// it: a truncated SHA-256 of the refresh token, which stays the same
// across access-token refreshes and changes with every login.
func tokenFingerprint(token *oauth2.Token) string {
	if token == nil || token.RefreshToken == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token.RefreshToken))
	return hex.EncodeToString(sum[:8])
}

// LoadGrantedAccess loads the per-service access levels for the stored
// token. Returns nil when there is no metadata, no stored token, or the
// metadata describes a different token than the one stored.
func LoadGrantedAccess() map[string]string {
	if record := loadGrantedRecord(); record != nil {
		return record.Services
	}
	return nil
}

// LoadGrantedScopes returns the exact scopes granted to the stored token,
// or nil when they are not known (no record, or a record for another
// token).
func LoadGrantedScopes() []string {
	if record := loadGrantedRecord(); record != nil {
		return record.Scopes
	}
	return nil
}

// LoadGrantedAccount returns the Google account ID recorded for the
// stored token, or "" when it is not known.
func LoadGrantedAccount() string {
	if record := loadGrantedRecord(); record != nil {
		return record.Account
	}
	return ""
}

// SnapshotGrantedAccess captures the access record as it is on disk and
// returns a function that puts it back (or removes the file if there was
// none). Used to undo a metadata write when the token it describes could
// not be saved.
func SnapshotGrantedAccess() (restore func()) {
	path := config.GetGrantedAccessPath()
	previous, err := os.ReadFile(path)
	existed := err == nil
	return func() {
		if !existed {
			os.Remove(path)
			return
		}
		tmp, err := os.CreateTemp(filepath.Dir(path), ".granted-access-*.tmp")
		if err != nil {
			return
		}
		tmpPath := tmp.Name()
		_ = tmp.Chmod(0600)
		_, werr := tmp.Write(previous)
		cerr := tmp.Close()
		if werr != nil || cerr != nil || os.Rename(tmpPath, path) != nil {
			os.Remove(tmpPath)
		}
	}
}

// loadGrantedRecord reads the access record and returns it only if it
// belongs to the stored token.
func loadGrantedRecord() *grantedAccessRecord {
	data, err := os.ReadFile(config.GetGrantedAccessPath())
	if err != nil {
		return nil
	}
	var record grantedAccessRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil
	}
	token, err := LoadToken()
	if err != nil {
		return nil
	}
	if fp := tokenFingerprint(token); fp == "" || fp != record.Token {
		return nil
	}
	return &record
}

// DeleteGrantedMetadata removes the granted-services and granted-access
// files. Missing files are not an error.
func DeleteGrantedMetadata() {
	os.Remove(config.GetGrantedServicesPath())
	os.Remove(config.GetGrantedAccessPath())
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
// A token write takes milliseconds, so a 30s-old lock is reliably stale.
// We use age-only detection (no PID probing) for cross-platform safety.
// Returns true if the lock was removed.
func removeStaleLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil {
		return false
	}

	if time.Since(info.ModTime()) > staleLockAge {
		os.Remove(lockPath)
		return true
	}

	return false
}
