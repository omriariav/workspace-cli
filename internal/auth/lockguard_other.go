//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows)

package auth

import "time"

// withLockGuard runs fn. This platform has no advisory file lock wired
// up, so stale-lock takeover keeps its best-effort behaviour here.
func withLockGuard(_ string, _ time.Time, fn func()) bool {
	fn()
	return true
}
