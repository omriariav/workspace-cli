//go:build windows

package auth

import (
	"os"

	"golang.org/x/sys/windows"
)

// tryLockFile takes an exclusive lock on f without blocking.
func tryLockFile(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, new(windows.Overlapped))
}

// lockIsBusy reports whether err means another holder has the lock.
func lockIsBusy(err error) bool {
	return err == windows.ERROR_LOCK_VIOLATION || err == windows.ERROR_IO_PENDING
}

// unlockFile releases the lock taken by tryLockFile.
func unlockFile(f *os.File) {
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
