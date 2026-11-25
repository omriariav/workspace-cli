//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows)

package auth

import (
	"fmt"
	"os"
	"time"
)

// acquireLock creates path + ".lock" with O_CREATE|O_EXCL and returns a
// function that removes it. This platform has no OS-enforced file lock
// wired up, so a lock left by a process that died cannot be told from one
// that is held, and it cannot be taken over safely: two processes could
// both judge it stale, and the second remove the lock the first had just
// created. A stale lock is therefore reported, not removed.
func acquireLock(path string) (unlock func(), err error) {
	lockPath := path + lockSuffix
	deadline := time.Now().Add(lockTimeout)
	owner := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())

	for {
		f, createErr := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr == nil {
			fmt.Fprint(f, owner)
			f.Close()
			return func() {
				if held, readErr := os.ReadFile(lockPath); readErr == nil && string(held) == owner {
					os.Remove(lockPath)
				}
			}, nil
		}
		if !os.IsExist(createErr) {
			return nil, fmt.Errorf("lock file error: %w", createErr)
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > staleLockAge {
			return nil, fmt.Errorf("lock file error: %s is %s old and was probably left by a gws process that did not exit cleanly; if no gws command is running, delete it and try again", lockPath, time.Since(info.ModTime()).Round(time.Second))
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: %s", errLockTimeout, lockPath)
		}
		time.Sleep(lockPollInterval)
	}
}
