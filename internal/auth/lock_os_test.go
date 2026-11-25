//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows

package auth

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireLock_BasicAcquireRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-file")
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	unlock()
	// Released: it can be taken again at once.
	again, err := acquireLock(path)
	if err != nil {
		t.Fatalf("failed to re-acquire a released lock: %v", err)
	}
	again()
}

// Versions of gws before the OS lock use the ".lock" file's existence as
// the lock. One of those files that is recent means such a version holds
// the lock right now: it is waited for, not ignored.
func TestAcquireLock_WaitsForAnOlderVersionsLockFile(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the lock timeout")
	}
	path := filepath.Join(t.TempDir(), "test-file")
	if err := os.WriteFile(path+lockSuffix, []byte("older-version"), 0600); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireLock(path)
	if err == nil {
		unlock()
		t.Fatal("acquired the lock while an older version's lock file was present")
	}
	if !errors.Is(err, errLockTimeout) {
		t.Errorf("err = %v, want a lock timeout", err)
	}
	if held, _ := os.ReadFile(path + lockSuffix); string(held) != "older-version" {
		t.Errorf("the older version's lock file was disturbed: %q", held)
	}
	// The failed attempt must not keep the OS lock: once the older
	// version's file is gone, the lock is free.
	if err := os.Remove(path + lockSuffix); err != nil {
		t.Fatal(err)
	}
	unlock, err = acquireLock(path)
	if err != nil {
		t.Fatalf("after the older version released: %v", err)
	}
	unlock()
}

// One left behind by an older version that died is taken over once it is
// stale, as those versions do among themselves.
func TestAcquireLock_TakesOverAStaleOlderLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-file")
	lockPath := path + lockSuffix
	if err := os.WriteFile(lockPath, []byte("999999999"), 0600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-staleLockAge - time.Second)
	if err := os.Chtimes(lockPath, past, past); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("a stale lock file blocked the lock: %v", err)
	}
	// While held, the file older versions look for exists, so they wait.
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("no lock file for older versions while held: %v", err)
	}
	unlock()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("the older-version lock file was not removed on release: %v", err)
	}
	// The OS lock file stays: it is never removed.
	if _, err := os.Stat(path + osLockSuffix); err != nil {
		t.Errorf("the OS lock file should remain: %v", err)
	}
}

// A lock that is held is never taken over, however old its file looks:
// a holder that is paused in the middle of its work (a logout that has
// read the token it is about to delete) must not find that another
// process went ahead and saved a new login in the meantime.
func TestAcquireLock_HeldLockIsNeverTakenOver(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the lock timeout")
	}
	path := filepath.Join(t.TempDir(), "test-file")
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	// Make both lock files look long abandoned.
	old := time.Now().Add(-time.Hour)
	for _, suffix := range []string{lockSuffix, osLockSuffix} {
		if err := os.Chtimes(path+suffix, old, old); err != nil {
			t.Fatal(err)
		}
	}

	start := time.Now()
	second, err := acquireLock(path)
	if err == nil {
		second()
		t.Fatal("a held lock was taken over")
	}
	if !errors.Is(err, errLockTimeout) {
		t.Errorf("err = %v, want a lock timeout", err)
	}
	if waited := time.Since(start); waited < lockTimeout || waited > lockTimeout+3*time.Second {
		t.Errorf("gave up after %v, want about %v", waited, lockTimeout)
	}
}

// Many callers contend for one lock: only one may hold it at a time.
func TestAcquireLock_IsExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-file")
	var holders, maxHolders, acquired int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := acquireLock(path)
			if err != nil {
				return // timed out behind the others; not what is tested
			}
			atomic.AddInt32(&acquired, 1)
			n := atomic.AddInt32(&holders, 1)
			for {
				seen := atomic.LoadInt32(&maxHolders)
				if n <= seen || atomic.CompareAndSwapInt32(&maxHolders, seen, n) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt32(&holders, -1)
			unlock()
		}()
	}
	wg.Wait()
	if maxHolders != 1 {
		t.Errorf("%d callers held the lock at once", maxHolders)
	}
	if acquired == 0 {
		t.Error("nobody acquired the lock")
	}
}
