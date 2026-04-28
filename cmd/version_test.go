package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/updatecheck"
)

func TestVersionCheck_ReportsStale(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v9.99.0"})
	}))
	defer srv.Close()

	checker := updatecheck.New(filepath.Join(t.TempDir(), "v.json"))
	checker.Endpoint = srv.URL
	checker.HTTPClient = srv.Client()

	prev := testVersionChecker
	testVersionChecker = checker
	t.Cleanup(func() { testVersionChecker = prev })

	prevVersion := Version
	Version = "1.36.0"
	t.Cleanup(func() { Version = prevVersion })

	cmd := versionCmd
	if err := cmd.Flags().Set("check", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("check", "false") })

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE err: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "newer version is available") {
		t.Errorf("expected stale notice, got %q", out)
	}
	if !strings.Contains(out, "v9.99.0") {
		t.Errorf("expected latest version in output, got %q", out)
	}
}

func TestVersionCheck_ReportsUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	checker := updatecheck.New(filepath.Join(t.TempDir(), "v.json"))
	checker.Endpoint = srv.URL
	checker.HTTPClient = srv.Client()

	prev := testVersionChecker
	testVersionChecker = checker
	t.Cleanup(func() { testVersionChecker = prev })

	prevVersion := Version
	Version = "1.37.0"
	t.Cleanup(func() { Version = prevVersion })

	cmd := versionCmd
	if err := cmd.Flags().Set("check", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("check", "false") })

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE err: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "up to date") {
		t.Errorf("expected up-to-date notice, got %q", out)
	}
}

func TestVersionCheck_DevBuildSkips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.37.0"})
	}))
	defer srv.Close()

	checker := updatecheck.New(filepath.Join(t.TempDir(), "v.json"))
	checker.Endpoint = srv.URL
	checker.HTTPClient = srv.Client()

	prev := testVersionChecker
	testVersionChecker = checker
	t.Cleanup(func() { testVersionChecker = prev })

	prevVersion := Version
	Version = "dev"
	t.Cleanup(func() { Version = prevVersion })

	cmd := versionCmd
	if err := cmd.Flags().Set("check", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("check", "false") })

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE err: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "skipped") {
		t.Errorf("expected skip notice, got %q", out)
	}
}

func TestVersionCheck_NetworkErrorIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := updatecheck.New("")
	checker.Endpoint = srv.URL
	checker.HTTPClient = srv.Client()

	prev := testVersionChecker
	testVersionChecker = checker
	t.Cleanup(func() { testVersionChecker = prev })

	prevVersion := Version
	Version = "1.36.0"
	t.Cleanup(func() { Version = prevVersion })

	cmd := versionCmd
	if err := cmd.Flags().Set("check", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("check", "false") })

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE should be non-fatal on network failure, got %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "failed to query GitHub") {
		t.Errorf("expected failure note, got %q", out)
	}
}
