//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package auth

import (
	"os"
	"syscall"
	"time"
)

// guardPollInterval is how often a busy guard is retried. The guard is
// only ever held for a few file operations.
const guardPollInterval = 5 * time.Millisecond

// withLockGuard runs fn while holding an exclusive advisory lock on a
// guard file beside lockPath, and reports whether fn ran. The kernel
// enforces the lock and releases it if the process dies, so it can never
// be left behind. The guard is requested without blocking and retried
// until the deadline, so a suspended guard holder cannot make the caller
// wait longer than its own timeout; in that case fn does not run. If the
// guard cannot be set up at all, fn still runs, but is told so (guarded
// is false) and must then not do anything that needs the guard.
func withLockGuard(lockPath string, deadline time.Time, fn func(guarded bool)) bool {
	guard, err := os.OpenFile(lockPath+".guard", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fn(false)
		return true
	}
	defer guard.Close()
	for {
		err := syscall.Flock(int(guard.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err == syscall.EINTR {
			continue
		}
		if err != syscall.EWOULDBLOCK {
			fn(false)
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(guardPollInterval)
	}
	defer syscall.Flock(int(guard.Fd()), syscall.LOCK_UN) //nolint:errcheck // released on close anyway
	fn(true)
	return true
}
