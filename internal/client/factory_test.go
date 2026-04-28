package client

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
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
