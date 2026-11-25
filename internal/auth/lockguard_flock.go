//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package auth

import (
	"os"
	"syscall"
)

// withLockGuard runs fn while holding an exclusive advisory lock on a
// guard file beside lockPath. The kernel enforces the lock and releases
// it if the process dies, so it can never be left behind. If the guard
// cannot be set up, fn runs unguarded rather than not at all.
func withLockGuard(lockPath string, fn func()) {
	guard, err := os.OpenFile(lockPath+".guard", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fn()
		return
	}
	defer guard.Close()
	if err := syscall.Flock(int(guard.Fd()), syscall.LOCK_EX); err != nil {
		fn()
		return
	}
	defer syscall.Flock(int(guard.Fd()), syscall.LOCK_UN) //nolint:errcheck // released on close anyway
	fn()
}
