package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/omriariav/workspace-cli/internal/auth"
)

// captureStderr swaps os.Stderr for a pipe, runs fn, and returns whatever was
// written. Restores the original os.Stderr on exit.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stderr = orig
	<-done
	return buf.String()
}

func TestPeopleProfile_DoesNotWarnOnMissingContactsScope(t *testing.T) {
	// Factory authenticated with --services chat (no contacts).
	f := &Factory{
		ctx:             context.Background(),
		grantedServices: []string{"chat"},
		scopeWarned:     map[string]bool{},
		mu:              sync.Mutex{},
	}

	out := captureStderr(t, func() {
		_, _ = f.PeopleProfile()
	})

	if strings.Contains(out, "contacts requires additional permissions") {
		t.Errorf("PeopleProfile must not warn about contacts scope; got %q", out)
	}
}

func TestPeople_WarnsOnMissingContactsScope(t *testing.T) {
	f := &Factory{
		ctx:             context.Background(),
		grantedServices: []string{"chat"},
		scopeWarned:     map[string]bool{},
		mu:              sync.Mutex{},
	}

	out := captureStderr(t, func() {
		_, _ = f.People()
	})

	if !strings.Contains(out, "contacts requires additional permissions") {
		t.Errorf("People must warn when contacts scope is missing; got %q", out)
	}
}

// TestPeople_PeopleAliasGrantsContactsScope verifies that users who run
// `gws auth login --services people` don't see the misleading "contacts
// requires additional permissions" warning on `gws people get` (or any
// other path that hits factory.People). The two service names share the
// same OAuth scope set per internal/auth/scopes.go.
func TestPeople_PeopleAliasGrantsContactsScope(t *testing.T) {
	f := &Factory{
		ctx:             context.Background(),
		grantedServices: []string{"people"},
		scopeWarned:     map[string]bool{},
		mu:              sync.Mutex{},
	}

	out := captureStderr(t, func() {
		_, _ = f.People()
	})
	if strings.Contains(out, "requires additional permissions") {
		t.Errorf("--services people must satisfy the contacts scope check; got %q", out)
	}
}

func TestPeopleProfile_NoWarnEvenAfterPeopleWarned(t *testing.T) {
	// Mixed flow: a code path may have already used People() (and warned).
	// PeopleProfile should still not emit a warning of its own.
	f := &Factory{
		ctx:             context.Background(),
		grantedServices: []string{"chat"},
		scopeWarned:     map[string]bool{},
		mu:              sync.Mutex{},
	}

	_ = captureStderr(t, func() { _, _ = f.People() })

	// Reset the once-flag so checkServiceScopes would warn again if invoked.
	f.scopeWarned = map[string]bool{}

	out := captureStderr(t, func() { _, _ = f.PeopleProfile() })
	if strings.Contains(out, "contacts requires additional permissions") {
		t.Errorf("PeopleProfile must not warn even after People warned; got %q", out)
	}
}

func TestAppMode_RejectsRegularServiceClients(t *testing.T) {
	f := &Factory{
		ctx:         context.Background(),
		authMode:    &auth.AuthModeConfig{Mode: auth.AuthModeApp, KeyFile: "/tmp/key.json"},
		scopeWarned: map[string]bool{},
	}

	checks := []struct {
		service string
		call    func() error
	}{
		{"gmail", func() error { _, err := f.Gmail(); return err }},
		{"calendar", func() error { _, err := f.Calendar(); return err }},
		{"drive", func() error { _, err := f.Drive(); return err }},
		{"docs", func() error { _, err := f.Docs(); return err }},
		{"sheets", func() error { _, err := f.Sheets(); return err }},
		{"slides", func() error { _, err := f.Slides(); return err }},
		{"tasks", func() error { _, err := f.Tasks(); return err }},
		{"chat", func() error { _, err := f.Chat(); return err }},
		{"forms", func() error { _, err := f.Forms(); return err }},
		{"groups", func() error { _, err := f.Admin(); return err }},
		{"keep", func() error { _, err := f.Keep(); return err }},
		{"driveactivity", func() error { _, err := f.DriveActivity(); return err }},
		{"contacts", func() error { _, err := f.People(); return err }},
		{"people-profile", func() error { _, err := f.PeopleProfile(); return err }},
	}

	for _, c := range checks {
		err := c.call()
		if err == nil {
			t.Errorf("%s: expected app-mode rejection, got nil error", c.service)
			continue
		}
		if !strings.Contains(err.Error(), "service-account-app") {
			t.Errorf("%s: error %q does not mention app mode", c.service, err)
		}
	}
}

func TestUserModes_AllowServiceClients(t *testing.T) {
	for _, mode := range []*auth.AuthModeConfig{
		nil,
		{Mode: auth.AuthModeOAuth},
		{Mode: auth.AuthModeDWD, KeyFile: "/tmp/key.json", ImpersonateUser: "u@example.com"},
	} {
		f := &Factory{
			ctx:         context.Background(),
			authMode:    mode,
			scopeWarned: map[string]bool{},
		}
		if err := f.requireUserAuth("gmail"); err != nil {
			t.Errorf("mode %+v: unexpected rejection: %v", mode, err)
		}
	}
}

func TestSetRateLimit_SpacesRequests(t *testing.T) {
	defer SetRateLimit(0)

	SetRateLimit(0)
	if d := rateLimitWait(); d != 0 {
		t.Fatalf("unlimited wait = %v, want 0", d)
	}

	SetRateLimit(10) // 100ms interval
	if d := rateLimitWait(); d > 5*time.Millisecond {
		t.Fatalf("first request wait = %v, want ~0", d)
	}
	d := rateLimitWait()
	if d < 50*time.Millisecond || d > 100*time.Millisecond {
		t.Fatalf("second request wait = %v, want ~100ms", d)
	}
}

func TestAuthOption_RateLimitedTransportThrottles(t *testing.T) {
	defer SetRateLimit(0)
	SetRateLimit(20) // 50ms interval

	rt := &rateLimitedTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})}
	start := time.Now()
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "http://example.invalid/", nil)
		if _, err := rt.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 90*time.Millisecond {
		t.Fatalf("3 requests took %v, want >= ~100ms", elapsed)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
