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

// A lock file with nobody holding it is not a held lock, whatever its
// age: the process that created it is gone, or it was left by an older
// version of gws that used the file's existence as the lock.
func TestAcquireLock_LeftoverLockFileIsNotHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-file")
	if err := os.WriteFile(path+lockSuffix, []byte("999999999"), 0600); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	unlock, err := acquireLock(path)
	if err != nil {
		t.Fatalf("a leftover lock file blocked the lock: %v", err)
	}
	unlock()
	if waited := time.Since(start); waited > time.Second {
		t.Errorf("waited %v for a lock nobody holds", waited)
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

	// Make the lock file look long abandoned.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path+lockSuffix, old, old); err != nil {
		t.Fatal(err)
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
