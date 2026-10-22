package client

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/spf13/viper"
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

func TestServiceAccountScopes_DefaultsToAllScopes(t *testing.T) {
	viper.Set(config.KeyServices, nil)
	t.Cleanup(func() { viper.Set(config.KeyServices, nil) })

	got := serviceAccountScopes(&auth.AuthModeConfig{Mode: auth.AuthModeDWD})
	if len(got) == 0 {
		t.Fatal("expected scopes, got none")
	}
	if strings.Join(got, ",") != strings.Join(auth.AllScopes, ",") {
		t.Errorf("expected auth.AllScopes when no services configured, got %v", got)
	}
}

func TestServiceAccountScopes_HonorsConfiguredServices(t *testing.T) {
	viper.Set(config.KeyServices, []string{"gmail"})
	t.Cleanup(func() { viper.Set(config.KeyServices, nil) })

	got := serviceAccountScopes(&auth.AuthModeConfig{Mode: auth.AuthModeDWD})
	want := auth.ScopesForServices([]string{"gmail"})
	if len(got) == 0 {
		t.Fatal("expected gmail scopes, got none")
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("scopes = %v, want %v", got, want)
	}
	if len(got) >= len(auth.AllScopes) {
		t.Errorf("configured services should narrow scopes: got %d, all %d", len(got), len(auth.AllScopes))
	}
}

func TestFactoryAuthMode(t *testing.T) {
	cfg := &auth.AuthModeConfig{Mode: auth.AuthModeDWD, KeyFile: "/tmp/key.json", ImpersonateUser: "u@example.com"}

	if got := (&Factory{authMode: cfg}).AuthMode(); got != cfg {
		t.Errorf("AuthMode() = %+v, want the configured pointer %+v", got, cfg)
	}
	if got := (&Factory{}).AuthMode(); got != nil {
		t.Errorf("AuthMode() = %+v, want nil when unset", got)
	}
}
