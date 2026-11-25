//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package auth

// withLockGuard runs fn. This platform has no advisory file lock wired
// up, so stale-lock takeover keeps its best-effort behaviour here.
func withLockGuard(_ string, fn func()) {
	fn()
}
