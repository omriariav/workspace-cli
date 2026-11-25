//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows

package auth

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// osLockSuffix names the file the OS lock is taken on. It is deliberately
// not the ".lock" file: versions of gws before this scheme treat that
// file's existence as the lock and delete it once it looks stale, which
// would pull the locked file out from under its holder.
const osLockSuffix = ".flock"

// osLockMarkerPrefix starts the contents of the ".lock" marker written by
// versions that hold the OS lock, which tells them apart from the lock
// files of older versions.
const osLockMarkerPrefix = "gws-flock:"

// acquireLock takes an exclusive lock for path and holds it until the
// returned function is called.
//
// The lock proper is an OS-enforced lock on path + ".flock". The kernel
// ties it to this process's open file: it is released if the process
// dies, so it can never be left behind, and — unlike a lock judged by its
// age — it is never taken over while its holder is alive, however long
// that holder is paused. That file is never removed: removing it would
// let a waiting process lock a file that no longer has a name while a
// third creates and locks a new one.
//
// For versions of gws that predate it, the holder also keeps the lock
// file they understand, path + ".lock" (created exclusively, removed on
// release), and waits for one of theirs to go away. That part works the
// way it always did between those versions, stale takeover after
// staleLockAge included; done under the OS lock, the takeover cannot race
// another current process.
func acquireLock(path string) (unlock func(), err error) {
	deadline := time.Now().Add(lockTimeout)
	osLockPath := path + osLockSuffix
	legacyPath := path + lockSuffix

	f, err := os.OpenFile(osLockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("lock file error: %w", err)
	}
	release := func() {
		unlockFile(f)
		f.Close()
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
			return nil, fmt.Errorf("%w: %s", errLockTimeout, osLockPath)
		}
		time.Sleep(lockPollInterval)
	}

	// Holding the OS lock: now the lock file older versions look for.
	owner := fmt.Sprintf("%s%d-%d", osLockMarkerPrefix, os.Getpid(), time.Now().UnixNano())
	for {
		legacy, createErr := os.OpenFile(legacyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr == nil {
			fmt.Fprint(legacy, owner)
			legacy.Close()
			break
		}
		if !os.IsExist(createErr) {
			release()
			return nil, fmt.Errorf("lock file error: %w", createErr)
		}
		// A marker written by a version that uses the OS lock was created
		// by a process holding that lock. This process holds it now, so
		// that one is gone (it crashed before removing its marker): the
		// marker is abandoned whatever its age, and is replaced at once.
		if held, readErr := os.ReadFile(legacyPath); readErr == nil && strings.HasPrefix(string(held), osLockMarkerPrefix) {
			if rmErr := os.Remove(legacyPath); rmErr == nil || os.IsNotExist(rmErr) {
				continue
			}
		}
		// Held by an older version, or left behind by one that died.
		if info, statErr := os.Stat(legacyPath); statErr == nil && time.Since(info.ModTime()) > staleLockAge {
			if rmErr := os.Remove(legacyPath); rmErr == nil || os.IsNotExist(rmErr) {
				continue
			} else {
				// A stale lock that cannot be removed will not go away by
				// retrying: give up, and let go of the OS lock so other
				// commands are not held up behind this one.
				release()
				return nil, fmt.Errorf("lock file error: stale lock %s could not be removed: %w", legacyPath, rmErr)
			}
		}
		if time.Now().After(deadline) {
			release()
			return nil, fmt.Errorf("%w: %s", errLockTimeout, legacyPath)
		}
		time.Sleep(lockPollInterval)
	}

	return func() {
		// Only our own: an older version may have judged it stale and
		// replaced it while this process held the lock for a long time.
		if held, readErr := os.ReadFile(legacyPath); readErr == nil && string(held) == owner {
			os.Remove(legacyPath)
		}
		release()
	}, nil
}
