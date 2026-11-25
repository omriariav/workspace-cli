//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows

package auth

import (
	"fmt"
	"os"
	"time"
)

// acquireLock takes an exclusive, OS-enforced lock on path + ".lock" and
// holds it until the returned function is called. The kernel ties the
// lock to this process's open file: it is released if the process dies,
// so a lock can never be left behind, and — unlike a lock judged by its
// age — it is never taken over while its holder is alive, however long
// that holder is paused.
//
// The lock file itself is never removed. Removing it would let a waiting
// process lock a file that no longer has a name while a third creates and
// locks a new one. An existing lock file with nobody holding it (left by
// this or an older version of gws) is simply locked.
func acquireLock(path string) (unlock func(), err error) {
	lockPath := path + lockSuffix
	deadline := time.Now().Add(lockTimeout)

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("lock file error: %w", err)
	}
	for {
		lockErr := tryLockFile(f)
		if lockErr == nil {
			break
		}
		if !lockIsBusy(lockErr) {
			f.Close()
			return nil, fmt.Errorf("lock file error: %w", lockErr)
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("%w: %s", errLockTimeout, lockPath)
		}
		time.Sleep(lockPollInterval)
	}
	return func() {
		unlockFile(f)
		f.Close()
	}, nil
}
