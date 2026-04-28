// Package updatecheck queries GitHub for the latest gws release and reports
// whether the installed binary is stale. Designed to be cheap and non-fatal:
// network and disk errors are swallowed so passive checks never break unrelated
// commands.
package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DefaultEndpoint is the GitHub API endpoint for the latest release of the
// public workspace-cli repository.
const DefaultEndpoint = "https://api.github.com/repos/omriariav/workspace-cli/releases/latest"

// DefaultTTL controls how long cached "latest version" results are considered
// fresh enough to skip the network round-trip on passive checks.
const DefaultTTL = 24 * time.Hour

// Checker performs version freshness checks against a configurable endpoint
// and cache file. Tests inject an httptest server URL and a temp cache path.
type Checker struct {
	Endpoint   string
	CachePath  string
	HTTPClient *http.Client
	TTL        time.Duration
	Now        func() time.Time
}

// Result describes the outcome of a freshness check.
type Result struct {
	Current string
	Latest  string
	Stale   bool
	// Skipped is true when the comparison was not made because the current
	// version is not a comparable release (dev build, empty, pseudo-version,
	// or unparseable). Latest may still be populated.
	Skipped       bool
	SkippedReason string
}

// CacheEntry is the on-disk shape of the version cache.
type CacheEntry struct {
	Latest    string    `json:"latest"`
	FetchedAt time.Time `json:"fetched_at"`
}

// New returns a Checker with sensible defaults. cachePath may be empty when
// caching should be disabled (e.g. forced explicit checks).
func New(cachePath string) *Checker {
	return &Checker{
		Endpoint:   DefaultEndpoint,
		CachePath:  cachePath,
		HTTPClient: &http.Client{Timeout: 3 * time.Second},
		TTL:        DefaultTTL,
		Now:        time.Now,
	}
}

// Check returns the freshness result for the given current version. forceFetch
// bypasses the cache. Errors from the network or cache are returned for
// callers that want to surface them (e.g. `version --check`); passive callers
// should ignore the error and just consult the result.
//
// For non-comparable current versions (dev / unknown / pseudo / unparseable),
// passive callers (forceFetch=false) get an early return with no network or
// cache I/O — there is nothing useful a passive notice could emit. Explicit
// callers (forceFetch=true) still fetch so `gws version --check` can report
// the latest release alongside the skip reason.
func (c *Checker) Check(ctx context.Context, current string, forceFetch bool) (*Result, error) {
	res := &Result{Current: current}

	if reason, ok := nonComparable(current); ok {
		res.Skipped = true
		res.SkippedReason = reason
		if !forceFetch {
			return res, nil
		}
	}

	latest, err := c.latest(ctx, forceFetch)
	if err != nil {
		return res, err
	}
	res.Latest = latest

	if res.Skipped {
		return res, nil
	}

	stale, cmpErr := IsStale(current, latest)
	if cmpErr != nil {
		res.Skipped = true
		res.SkippedReason = cmpErr.Error()
		return res, nil
	}
	res.Stale = stale
	return res, nil
}

func (c *Checker) latest(ctx context.Context, forceFetch bool) (string, error) {
	if !forceFetch {
		if entry, ok := c.readCache(); ok {
			if c.Now().Sub(entry.FetchedAt) < c.TTL && entry.Latest != "" {
				return entry.Latest, nil
			}
		}
	}

	latest, err := c.fetchLatest(ctx)
	if err != nil {
		return "", err
	}

	c.writeCache(CacheEntry{Latest: latest, FetchedAt: c.Now()})
	return latest, nil
}

func (c *Checker) fetchLatest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gws-version-check")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("github releases endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", err
	}

	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		tag = strings.TrimSpace(payload.Name)
	}
	if tag == "" {
		return "", errors.New("github response had no tag_name")
	}
	return tag, nil
}

func (c *Checker) readCache() (CacheEntry, bool) {
	if c.CachePath == "" {
		return CacheEntry{}, false
	}
	data, err := os.ReadFile(c.CachePath)
	if err != nil {
		return CacheEntry{}, false
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return CacheEntry{}, false
	}
	if entry.Latest == "" {
		return CacheEntry{}, false
	}
	return entry, true
}

func (c *Checker) writeCache(entry CacheEntry) {
	if c.CachePath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.CachePath), 0700); err != nil {
		return
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}
	tmp := c.CachePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return
	}
	_ = os.Rename(tmp, c.CachePath)
}

// nonComparable reports whether the given version string should be skipped
// for passive comparison. Returns the reason for skipping when ok is true.
func nonComparable(v string) (reason string, ok bool) {
	if strings.TrimSpace(v) == "" {
		return "version is empty", true
	}
	if v == "dev" || v == "unknown" {
		return fmt.Sprintf("version %q is not a release build", v), true
	}
	// Go pseudo-versions look like v0.0.0-20240101000000-abcdef0123ab and
	// embed a date+commit. Treat them as non-release.
	if strings.Contains(v, "-0.") || strings.Count(v, "-") >= 2 {
		// Heuristic: real semver may include a pre-release ("-rc.1") with a
		// single dash. Pseudo-versions and Go's "+incompatible" decorations
		// have multiple dashes or "-0." suffixes. Be conservative and skip.
		if looksPseudo(v) {
			return fmt.Sprintf("version %q looks like a pseudo-version", v), true
		}
	}
	if _, err := parseSemver(v); err != nil {
		return fmt.Sprintf("version %q is not parseable", v), true
	}
	return "", false
}

func looksPseudo(v string) bool {
	v = strings.TrimPrefix(v, "v")
	// e.g. 1.27.1-0.20240101000000-abcdef0123ab
	if strings.Contains(v, "-0.") {
		return true
	}
	parts := strings.Split(v, "-")
	if len(parts) < 3 {
		return false
	}
	last := parts[len(parts)-1]
	// short commit hash heuristic
	if len(last) >= 7 && len(last) <= 14 && isHex(last) {
		return true
	}
	return false
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// IsStale reports whether current < latest. Both inputs may carry a leading
// "v". Returns an error when either version cannot be parsed; callers should
// treat that as "skip comparison" rather than a hard failure.
func IsStale(current, latest string) (bool, error) {
	cur, err := parseSemver(current)
	if err != nil {
		return false, fmt.Errorf("parse current %q: %w", current, err)
	}
	lat, err := parseSemver(latest)
	if err != nil {
		return false, fmt.Errorf("parse latest %q: %w", latest, err)
	}
	return compare(cur, lat) < 0, nil
}

type semver struct {
	major, minor, patch int
	pre                 string
}

func parseSemver(raw string) (semver, error) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if v == "" {
		return semver{}, errors.New("empty")
	}

	core := v
	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		core = v[:i]
		// keep only the pre-release portion, drop +build metadata for compare
		rest := v[i:]
		if strings.HasPrefix(rest, "-") {
			rest = strings.SplitN(rest, "+", 2)[0]
			pre = strings.TrimPrefix(rest, "-")
		}
	}

	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return semver{}, fmt.Errorf("expected 1-3 numeric components, got %d", len(parts))
	}

	out := semver{pre: pre}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, fmt.Errorf("component %d (%q) is not a non-negative integer", i, p)
		}
		switch i {
		case 0:
			out.major = n
		case 1:
			out.minor = n
		case 2:
			out.patch = n
		}
	}
	return out, nil
}

// compare returns -1, 0, or 1.
func compare(a, b semver) int {
	switch {
	case a.major != b.major:
		return signInt(a.major - b.major)
	case a.minor != b.minor:
		return signInt(a.minor - b.minor)
	case a.patch != b.patch:
		return signInt(a.patch - b.patch)
	}
	// Pre-release < no pre-release.
	if a.pre == "" && b.pre != "" {
		return 1
	}
	if a.pre != "" && b.pre == "" {
		return -1
	}
	if a.pre == b.pre {
		return 0
	}
	if a.pre < b.pre {
		return -1
	}
	return 1
}

func signInt(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// FormatPassiveNotice returns the stderr line emitted by passive checks, or
// "" when no notice should be printed.
func FormatPassiveNotice(res *Result) string {
	if res == nil || !res.Stale {
		return ""
	}
	return fmt.Sprintf("gws: a newer version is available (%s -> %s). Run `gws version --check` for details.\n", res.Current, res.Latest)
}
