//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package auth

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A guard held by a process that is not making progress must not make
// another caller wait past its own deadline: the guard is requested
// without blocking, and when it stays busy fn is not run.
func TestWithLockGuard_GivesUpAtTheDeadline(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "token.json.lock")

	holder, err := os.OpenFile(lockPath+".guard", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}

	ran := false
	start := time.Now()
	if withLockGuard(lockPath, start.Add(100*time.Millisecond), func() { ran = true }) {
		t.Error("withLockGuard reported success while the guard was held elsewhere")
	}
	if ran {
		t.Error("fn ran without the guard")
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Errorf("waited %v on a busy guard, want about the 100ms deadline", waited)
	}

	// Once the holder lets go, the guard is available again.
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if !withLockGuard(lockPath, time.Now().Add(time.Second), func() { ran = true }) || !ran {
		t.Error("withLockGuard should run fn once the guard is free")
	}
}
