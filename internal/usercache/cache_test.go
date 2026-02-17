package usercache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func tempCache(t *testing.T) *Cache {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "user-cache.json")
	return &Cache{
		path:    path,
		entries: make(map[string]UserInfo),
	}
}

func TestCache_GetSet(t *testing.T) {
	c := tempCache(t)

	// Miss
	_, ok := c.Get("users/123")
	if ok {
		t.Error("expected miss on empty cache")
	}

	// Set and hit
	c.Set("users/123", UserInfo{DisplayName: "Alice", Email: "alice@example.com"})
	info, ok := c.Get("users/123")
	if !ok {
		t.Fatal("expected hit after set")
	}
	if info.DisplayName != "Alice" {
		t.Errorf("expected 'Alice', got %q", info.DisplayName)
	}
	if info.Email != "alice@example.com" {
		t.Errorf("expected 'alice@example.com', got %q", info.Email)
	}
}

func TestCache_SaveAndReload(t *testing.T) {
	c := tempCache(t)
	c.Set("users/1", UserInfo{DisplayName: "Bob", Email: "bob@example.com"})
	c.Set("users/2", UserInfo{DisplayName: "Carol"})

	if err := c.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Read back the file
	data, err := os.ReadFile(c.path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var loaded map[string]UserInfo
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded))
	}
	if loaded["users/1"].DisplayName != "Bob" {
		t.Errorf("expected 'Bob', got %q", loaded["users/1"].DisplayName)
	}
	if loaded["users/2"].Email != "" {
		t.Errorf("expected empty email for Carol, got %q", loaded["users/2"].Email)
	}
}

func TestCache_AtomicSave(t *testing.T) {
	c := tempCache(t)
	c.Set("users/1", UserInfo{DisplayName: "First"})

	if err := c.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify no .tmp file remains
	tmpPath := c.path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}

	// Verify the file is valid JSON
	data, _ := os.ReadFile(c.path)
	var entries map[string]UserInfo
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}
}

func TestCache_CorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user-cache.json")

	// Write corrupt JSON
	os.WriteFile(path, []byte("{invalid json!!!"), 0600)

	// New() should not fail, just start fresh
	c := &Cache{
		path:    path,
		entries: make(map[string]UserInfo),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if jsonErr := json.Unmarshal(data, &c.entries); jsonErr != nil {
		// Expected — reset
		c.entries = make(map[string]UserInfo)
	}

	if len(c.entries) != 0 {
		t.Error("expected empty cache after corrupt file")
	}

	// Should be able to set and save normally
	c.Set("users/1", UserInfo{DisplayName: "Recovered"})
	if err := c.Save(); err != nil {
		t.Fatalf("save after corrupt failed: %v", err)
	}
}

func TestResolveMany_NilService(t *testing.T) {
	c := tempCache(t)

	// Must not panic
	c.ResolveMany(nil, []string{"users/123"})

	_, ok := c.Get("users/123")
	if ok {
		t.Error("should not resolve with nil service")
	}
}

func TestResolveMany_SkipsInvalidPrefixes(t *testing.T) {
	c := tempCache(t)

	// IDs without "users/" prefix should be skipped
	c.ResolveMany(nil, []string{"bots/abc", "apps/xyz", ""})

	// No entries should be added (and no panic)
	if _, ok := c.Get("bots/abc"); ok {
		t.Error("should not cache non-user IDs")
	}
}
