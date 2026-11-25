//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows)

package auth

import (
	"fmt"
	"os"
	"time"
)

// acquireLock creates path + ".lock" with O_CREATE|O_EXCL and returns a
// function that removes it. This platform has no OS-enforced file lock
// wired up, so a lock left by a process that died is taken over once it
// is older than staleLockAge. That is best effort: a holder paused for
// longer than that can lose its lock.
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
			os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: %s", errLockTimeout, lockPath)
		}
		time.Sleep(lockPollInterval)
	}
}
