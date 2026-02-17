package usercache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"google.golang.org/api/people/v1"
)

// UserInfo stores resolved user metadata.
type UserInfo struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
}

// Cache provides a persistent user ID → display name cache.
type Cache struct {
	mu      sync.Mutex
	path    string
	entries map[string]UserInfo
}

// New loads or creates a user cache at ~/.config/gws/user-cache.json.
func New() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".config", "gws", "user-cache.json")

	c := &Cache{
		path:    path,
		entries: make(map[string]UserInfo),
	}

	data, err := os.ReadFile(path)
	if err == nil {
		if jsonErr := json.Unmarshal(data, &c.entries); jsonErr != nil {
			// Corrupted cache — log and start fresh
			fmt.Fprintf(os.Stderr, "warning: user cache corrupted, resetting: %v\n", jsonErr)
			c.entries = make(map[string]UserInfo)
		}
	}

	return c, nil
}

// Get returns cached info for a user ID (e.g., "users/123"), or false if not cached.
func (c *Cache) Get(userID string) (UserInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info, ok := c.entries[userID]
	return info, ok
}

// Set stores a user entry.
func (c *Cache) Set(userID string, info UserInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = info
}

// Save persists the cache to disk using atomic temp-file + rename.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// ResolveMany looks up unknown user IDs via the People API in batches.
// Only processes IDs with the "users/" prefix. Skips bots and invalid IDs.
// Caches results including email-only entries (no display name required).
func (c *Cache) ResolveMany(peopleSvc *people.Service, userIDs []string) {
	if peopleSvc == nil {
		return
	}

	// Filter to only unknown IDs with valid "users/" prefix
	var unknown []string
	for _, id := range userIDs {
		if !strings.HasPrefix(id, "users/") {
			continue
		}
		if _, ok := c.Get(id); !ok {
			unknown = append(unknown, id)
		}
	}

	if len(unknown) == 0 {
		return
	}

	// People API getBatchGet supports up to 50 resource names per call
	for i := 0; i < len(unknown); i += 50 {
		end := i + 50
		if end > len(unknown) {
			end = len(unknown)
		}
		batch := unknown[i:end]

		// Convert "users/123" to "people/123" for People API
		resourceNames := make([]string, len(batch))
		for j, id := range batch {
			resourceNames[j] = "people/" + strings.TrimPrefix(id, "users/")
		}

		resp, err := peopleSvc.People.GetBatchGet().
			ResourceNames(resourceNames...).
			PersonFields("names,emailAddresses").
			Do()
		if err != nil {
			continue // Best-effort: skip batch on error
		}

		for _, pr := range resp.Responses {
			if pr.Person == nil || pr.Person.ResourceName == "" {
				continue
			}
			if !strings.HasPrefix(pr.Person.ResourceName, "people/") {
				continue
			}
			userID := "users/" + strings.TrimPrefix(pr.Person.ResourceName, "people/")

			info := UserInfo{}
			if len(pr.Person.Names) > 0 {
				info.DisplayName = pr.Person.Names[0].DisplayName
			}
			if len(pr.Person.EmailAddresses) > 0 {
				info.Email = pr.Person.EmailAddresses[0].Value
			}
			// Cache if we got any useful data (name or email)
			if info.DisplayName != "" || info.Email != "" {
				c.Set(userID, info)
			}
		}
	}

	_ = c.Save()
}
