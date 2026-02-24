package spacecache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
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

func TestBuild_MockServer(t *testing.T) {
	// Mock Chat API: spaces list + members for each space
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/spaces", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"spaces": []map[string]interface{}{
				{"name": "spaces/GC1", "displayName": "", "spaceType": "GROUP_CHAT"},
				{"name": "spaces/GC2", "displayName": "Eng Team", "spaceType": "GROUP_CHAT"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/spaces/GC1/members", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"memberships": []map[string]interface{}{
				{"name": "spaces/GC1/members/1", "member": map[string]interface{}{"name": "users/111", "type": "HUMAN"}},
				{"name": "spaces/GC1/members/2", "member": map[string]interface{}{"name": "users/222", "type": "HUMAN"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/spaces/GC2/members", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"memberships": []map[string]interface{}{
				{"name": "spaces/GC2/members/1", "member": map[string]interface{}{"name": "users/111", "type": "HUMAN"}},
				{"name": "spaces/GC2/members/3", "member": map[string]interface{}{"name": "users/333", "type": "HUMAN"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	chatSvc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	// Build cache (no People service — emails won't resolve, IDs used as fallback)
	cache, err := Build(context.Background(), chatSvc, nil, "GROUP_CHAT", nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(cache.Spaces) != 2 {
		t.Fatalf("expected 2 spaces in cache, got %d", len(cache.Spaces))
	}

	gc1 := cache.Spaces["spaces/GC1"]
	if gc1.MemberCount != 2 {
		t.Errorf("expected 2 members in GC1, got %d", gc1.MemberCount)
	}
	if gc1.Type != "GROUP_CHAT" {
		t.Errorf("expected type GROUP_CHAT, got %q", gc1.Type)
	}

	gc2 := cache.Spaces["spaces/GC2"]
	if gc2.DisplayName != "Eng Team" {
		t.Errorf("expected display name 'Eng Team', got %q", gc2.DisplayName)
	}

	// Save to temp file and reload
	tmpPath := filepath.Join(t.TempDir(), "cache.json")
	if err := Save(tmpPath, cache); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(tmpPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Spaces) != 2 {
		t.Fatalf("expected 2 spaces after reload, got %d", len(loaded.Spaces))
	}

	// Search: users/111 is in both spaces
	results := FindByMembers(loaded, []string{"users/111"})
	if len(results) != 2 {
		t.Errorf("expected 2 matches for users/111, got %d", len(results))
	}

	// Search: users/222 is only in GC1
	results = FindByMembers(loaded, []string{"users/222"})
	if len(results) != 1 {
		t.Errorf("expected 1 match for users/222, got %d", len(results))
	}
	if _, ok := results["spaces/GC1"]; !ok {
		t.Error("expected spaces/GC1 in results")
	}
}

func TestBuild_SkipsSpaceOnMemberFetchFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/spaces", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"spaces": []map[string]interface{}{
				{"name": "spaces/OK", "spaceType": "GROUP_CHAT"},
				{"name": "spaces/FAIL", "spaceType": "GROUP_CHAT"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/spaces/OK/members", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"memberships": []map[string]interface{}{
				{"name": "spaces/OK/members/1", "member": map[string]interface{}{"name": "users/111", "type": "HUMAN"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/spaces/FAIL/members", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "forbidden"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	chatSvc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	cache, err := Build(context.Background(), chatSvc, nil, "GROUP_CHAT", nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// spaces/FAIL should be skipped (0 members due to error)
	if _, ok := cache.Spaces["spaces/FAIL"]; ok {
		t.Error("expected spaces/FAIL to be skipped due to member fetch failure")
	}

	// spaces/OK should be present
	if _, ok := cache.Spaces["spaces/OK"]; !ok {
		t.Error("expected spaces/OK to be in cache")
	}
}
