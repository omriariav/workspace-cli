//go:build windows

package auth

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// guardPollInterval is how often a busy guard is retried. The guard is
// only ever held for a few file operations.
const guardPollInterval = 5 * time.Millisecond

// withLockGuard runs fn while holding an exclusive lock (LockFileEx) on a
// guard file beside lockPath, and reports whether fn ran. Windows
// releases the lock when the handle closes or the process dies, so it can
// never be left behind. The lock is requested without blocking and
// retried until the deadline; if it stays busy, fn does not run. If the
// guard cannot be set up at all, fn still runs, but is told so (guarded
// is false) and must then not do anything that needs the guard.
func withLockGuard(lockPath string, deadline time.Time, fn func(guarded bool)) bool {
	guard, err := os.OpenFile(lockPath+".guard", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fn(false)
		return true
	}
	defer guard.Close()
	handle := windows.Handle(guard.Fd())
	overlapped := new(windows.Overlapped)
	for {
		err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped)
		if err == nil {
			break
		}
		if err != windows.ERROR_LOCK_VIOLATION && err != windows.ERROR_IO_PENDING {
			fn(false)
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(guardPollInterval)
	}
	defer windows.UnlockFileEx(handle, 0, 1, 0, overlapped) //nolint:errcheck // released on close anyway
	fn(true)
	return true
}
