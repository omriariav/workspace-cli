package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsStale(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"1.36.0", "1.37.0", true},
		{"v1.36.0", "v1.37.0", true},
		{"1.37.0", "1.37.0", false},
		{"1.37.1", "1.37.0", false},
		{"1.36.5", "2.0.0", true},
		{"1.36.0-rc.1", "1.36.0", true},
		{"1.36.0", "1.36.0-rc.1", false},
	}
	for _, c := range cases {
		got, err := IsStale(c.current, c.latest)
		if err != nil {
			t.Fatalf("IsStale(%q,%q) unexpected err: %v", c.current, c.latest, err)
		}
		if got != c.want {
			t.Errorf("IsStale(%q,%q) = %v; want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestIsStaleParseError(t *testing.T) {
	if _, err := IsStale("not-a-version", "1.0.0"); err == nil {
		t.Errorf("expected error for unparseable current version")
	}
	if _, err := IsStale("1.0.0", "totally bad"); err == nil {
		t.Errorf("expected error for unparseable latest version")
	}
}

func TestNonComparable(t *testing.T) {
	cases := []struct{ in string }{
		{""},
		{"dev"},
		{"unknown"},
		{"v1.27.1-0.20240101000000-abcdef0123ab"},
	}
	for _, c := range cases {
		if _, ok := nonComparable(c.in); !ok {
			t.Errorf("nonComparable(%q) = false; want true", c.in)
		}
	}
	for _, good := range []string{"1.36.0", "v1.36.0", "1.36.0-rc.1"} {
		if _, ok := nonComparable(good); ok {
			t.Errorf("nonComparable(%q) = true; want false", good)
		}
	}
}

func TestCheck_FetchesAndCaches(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "version-cache.json")

	c := New(cachePath)
	c.Endpoint = srv.URL
	c.HTTPClient = srv.Client()
	c.Now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	res, err := c.Check(context.Background(), "1.36.0", false)
	if err != nil {
		t.Fatalf("Check err: %v", err)
	}
	if !res.Stale || res.Latest != "v1.37.0" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if calls != 1 {
		t.Fatalf("expected 1 network call, got %d", calls)
	}

	// Cache should be hit on the next call within TTL.
	res2, err := c.Check(context.Background(), "1.36.0", false)
	if err != nil {
		t.Fatalf("second Check err: %v", err)
	}
	if !res2.Stale || res2.Latest != "v1.37.0" {
		t.Fatalf("unexpected cached result: %+v", res2)
	}
	if calls != 1 {
		t.Fatalf("expected cache hit, got %d calls", calls)
	}

	// Force fetch should bypass cache.
	if _, err := c.Check(context.Background(), "1.36.0", true); err != nil {
		t.Fatalf("forced Check err: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected forced fetch, got %d calls", calls)
	}
}

func TestCheck_NetworkErrorReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New("")
	c.Endpoint = srv.URL
	c.HTTPClient = srv.Client()

	res, err := c.Check(context.Background(), "1.36.0", false)
	if err == nil {
		t.Fatalf("expected network error, got nil")
	}
	if res == nil || res.Current != "1.36.0" {
		t.Fatalf("expected partial result populated, got %+v", res)
	}
	if res.Stale {
		t.Fatalf("stale should be false on network failure, got %+v", res)
	}
}

func TestCheck_PassiveDevSkipsNetwork(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "v.json")

	c := New(cachePath)
	c.Endpoint = srv.URL
	c.HTTPClient = srv.Client()

	res, err := c.Check(context.Background(), "dev", false)
	if err != nil {
		t.Fatalf("Check err: %v", err)
	}
	if calls != 0 {
		t.Errorf("passive dev check must not hit the endpoint, got %d calls", calls)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true for dev build, got %+v", res)
	}
	if res.Latest != "" {
		t.Errorf("expected Latest empty when passive dev skip short-circuits, got %q", res.Latest)
	}
	// The cache should not have been written either.
	if _, err := os.Stat(cachePath); err == nil {
		t.Errorf("passive dev check should not write cache; file exists at %s", cachePath)
	}
}

func TestCheck_ForceFetchDevStillFetches(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	c := New("")
	c.Endpoint = srv.URL
	c.HTTPClient = srv.Client()

	res, err := c.Check(context.Background(), "dev", true)
	if err != nil {
		t.Fatalf("Check err: %v", err)
	}
	if calls != 1 {
		t.Errorf("explicit dev check should still fetch, got %d calls", calls)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true for dev build, got %+v", res)
	}
	if res.Latest != "v1.37.0" {
		t.Errorf("expected Latest populated for explicit dev check, got %q", res.Latest)
	}
}

func TestCheck_DevVersionSkipsComparison(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	c := New("")
	c.Endpoint = srv.URL
	c.HTTPClient = srv.Client()

	// Use forceFetch=true so the latest release is still populated even
	// though the comparison itself is skipped.
	res, err := c.Check(context.Background(), "dev", true)
	if err != nil {
		t.Fatalf("Check err: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true for dev build, got %+v", res)
	}
	if res.Stale {
		t.Errorf("expected Stale=false for dev build, got %+v", res)
	}
	if res.Latest != "v1.37.0" {
		t.Errorf("expected latest still populated: %+v", res)
	}
}

func TestCheck_ExpiredCacheRefetches(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "v.json")

	// Pre-populate stale cache entry.
	old := CacheEntry{Latest: "v1.30.0", FetchedAt: time.Unix(0, 0)}
	data, _ := json.Marshal(old)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	c := New(cachePath)
	c.Endpoint = srv.URL
	c.HTTPClient = srv.Client()
	c.TTL = time.Second
	c.Now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	res, err := c.Check(context.Background(), "1.36.0", false)
	if err != nil {
		t.Fatalf("Check err: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected refetch, got %d calls", calls)
	}
	if res.Latest != "v1.37.0" {
		t.Fatalf("expected refreshed latest, got %+v", res)
	}
}

func TestFormatPassiveNotice(t *testing.T) {
	if got := FormatPassiveNotice(nil); got != "" {
		t.Errorf("nil result should produce empty notice, got %q", got)
	}
	if got := FormatPassiveNotice(&Result{Stale: false}); got != "" {
		t.Errorf("non-stale result should produce empty notice, got %q", got)
	}
	if got := FormatPassiveNotice(&Result{Stale: true, Current: "1.36.0", Latest: "v1.37.0"}); got == "" {
		t.Error("stale result should produce notice")
	}
}
