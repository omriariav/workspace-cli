package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestRevokeToken_Success(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got %s", ct)
		}
		r.ParseForm()
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	token := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
	}

	err := revokeWithEndpoint(context.Background(), server.URL, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should prefer refresh token
	if receivedToken != "refresh-456" {
		t.Errorf("expected refresh token to be sent, got: %s", receivedToken)
	}
}

func TestRevokeToken_PrefersRefreshToken(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	token := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	err := revokeWithEndpoint(context.Background(), server.URL, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedToken != "refresh-token" {
		t.Errorf("should prefer refresh token, got: %s", receivedToken)
	}
}

func TestRevokeToken_FallbackToAccessToken(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	token := &oauth2.Token{
		AccessToken:  "access-only",
		RefreshToken: "",
	}

	err := revokeWithEndpoint(context.Background(), server.URL, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedToken != "access-only" {
		t.Errorf("should fall back to access token, got: %s", receivedToken)
	}
}

func TestRevokeToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_token"}`))
	}))
	defer server.Close()

	token := &oauth2.Token{
		RefreshToken: "bad-token",
	}

	err := revokeWithEndpoint(context.Background(), server.URL, token)
	if err == nil {
		t.Fatal("expected error for server error response")
	}

	if !contains(err.Error(), "400") {
		t.Errorf("error should mention status code, got: %v", err)
	}
}

func TestRevokeToken_NilToken(t *testing.T) {
	err := revokeWithEndpoint(context.Background(), "http://unused", nil)
	if err == nil {
		t.Fatal("expected error for nil token")
	}
}

func TestRevokeToken_EmptyToken(t *testing.T) {
	token := &oauth2.Token{}
	err := revokeWithEndpoint(context.Background(), "http://unused", token)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
