package spacecache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	// Load from non-existent file returns empty cache
	cache, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error loading non-existent cache: %v", err)
	}
	if len(cache.Spaces) != 0 {
		t.Errorf("expected empty cache, got %d spaces", len(cache.Spaces))
	}

	// Save and reload
	cache.LastUpdated = time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	cache.Spaces["spaces/AAAA"] = SpaceEntry{
		Type:        "GROUP_CHAT",
		DisplayName: "",
		Members:     []string{"alice@example.com", "bob@example.com"},
		MemberCount: 2,
	}
	cache.Spaces["spaces/BBBB"] = SpaceEntry{
		Type:        "SPACE",
		DisplayName: "Engineering",
		Members:     []string{"alice@example.com", "charlie@example.com"},
		MemberCount: 2,
	}

	if err := Save(path, cache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("failed to reload cache: %v", err)
	}

	if len(loaded.Spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(loaded.Spaces))
	}
	if loaded.Spaces["spaces/AAAA"].MemberCount != 2 {
		t.Errorf("expected member count 2, got %d", loaded.Spaces["spaces/AAAA"].MemberCount)
	}
	if loaded.Spaces["spaces/BBBB"].DisplayName != "Engineering" {
		t.Errorf("expected display name 'Engineering', got %q", loaded.Spaces["spaces/BBBB"].DisplayName)
	}
}

func TestLoadCorruptedCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	os.WriteFile(path, []byte("not json"), 0600)

	cache, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cache.Spaces) != 0 {
		t.Errorf("expected empty cache after corruption, got %d", len(cache.Spaces))
	}
}

func TestFindByMembers(t *testing.T) {
	cache := &CacheData{
		Spaces: map[string]SpaceEntry{
			"spaces/AAAA": {
				Type:    "GROUP_CHAT",
				Members: []string{"alice@example.com", "bob@example.com", "charlie@example.com"},
			},
			"spaces/BBBB": {
				Type:    "GROUP_CHAT",
				Members: []string{"alice@example.com", "dave@example.com"},
			},
			"spaces/CCCC": {
				Type:    "SPACE",
				Members: []string{"alice@example.com", "bob@example.com"},
			},
		},
	}

	// Search for alice + bob — should match AAAA and CCCC
	results := FindByMembers(cache, []string{"alice@example.com", "bob@example.com"})
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if _, ok := results["spaces/AAAA"]; !ok {
		t.Error("expected spaces/AAAA in results")
	}
	if _, ok := results["spaces/CCCC"]; !ok {
		t.Error("expected spaces/CCCC in results")
	}

	// Search for alice + dave — only BBBB
	results = FindByMembers(cache, []string{"alice@example.com", "dave@example.com"})
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if _, ok := results["spaces/BBBB"]; !ok {
		t.Error("expected spaces/BBBB in results")
	}

	// Search for unknown — no results
	results = FindByMembers(cache, []string{"nobody@example.com"})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFindByMembers_CaseInsensitive(t *testing.T) {
	cache := &CacheData{
		Spaces: map[string]SpaceEntry{
			"spaces/AAAA": {
				Members: []string{"Alice@Example.com", "Bob@Example.com"},
			},
		},
	}

	results := FindByMembers(cache, []string{"alice@example.com"})
	if len(results) != 1 {
		t.Errorf("expected case-insensitive match, got %d results", len(results))
	}
}

func TestFindByMembers_EmptySearch(t *testing.T) {
	cache := &CacheData{
		Spaces: map[string]SpaceEntry{
			"spaces/AAAA": {Members: []string{"alice@example.com"}},
		},
	}

	// Empty search matches everything (all of zero needles are present)
	results := FindByMembers(cache, []string{})
	if len(results) != 1 {
		t.Errorf("expected all spaces matched with empty search, got %d", len(results))
	}
}
