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
	if withLockGuard(lockPath, start.Add(100*time.Millisecond), func(bool) { ran = true }) {
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
	if !withLockGuard(lockPath, time.Now().Add(time.Second), func(bool) { ran = true }) || !ran {
		t.Error("withLockGuard should run fn once the guard is free")
	}
}

// Taking over a stale lock is only safe under the guard. When the guard
// cannot be set up (here its path is a directory), a stale lock is waited
// for like any other instead of being removed: two unguarded processes
// could otherwise both remove it and both believe they hold the lock.
func TestAcquireLock_NoStaleTakeoverWithoutTheGuard(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the lock timeout")
	}
	path := filepath.Join(t.TempDir(), "login")
	lockPath := path + lockSuffix
	if err := os.Mkdir(lockPath+".guard", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("someone-else"), 0600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-staleLockAge - time.Minute)
	if err := os.Chtimes(lockPath, past, past); err != nil {
		t.Fatal(err)
	}

	if unlock, err := acquireLock(path); err == nil {
		unlock()
		t.Fatal("a stale lock was taken over without the guard")
	}
	if held, err := os.ReadFile(lockPath); err != nil || string(held) != "someone-else" {
		t.Errorf("the existing lock was disturbed: %q, %v", held, err)
	}

	// With no lock in the way, acquiring and releasing still works
	// unguarded.
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("acquire without the guard: %v", err)
	}
	unlock()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("the lock was not released: %v", err)
	}
}
