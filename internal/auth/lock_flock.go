//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package auth

import (
	"os"
	"syscall"
)

// tryLockFile takes an exclusive advisory lock on f without blocking.
func tryLockFile(f *os.File) error {
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != syscall.EINTR {
			return err
		}
	}
}

// lockIsBusy reports whether err means another holder has the lock.
func lockIsBusy(err error) bool {
	return err == syscall.EWOULDBLOCK
}

// unlockFile releases the lock taken by tryLockFile.
func unlockFile(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
