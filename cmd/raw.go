package cmd

// Programmatic-mode plumbing for `--raw` and `--params`.
//
// `--raw`    : emit the unmodified Google API response JSON (no field
//              renaming, no body decoding, no header collapsing). Default
//              ergonomic output is untouched when this flag is not set.
// `--params` : JSON object whose keys map directly to the underlying Google
//              API request parameters. Keys supplied here override the
//              equivalent CLI flags (params win), so callers can rely on the
//              JSON payload being the source of truth.
//
// Pagination under `--raw`:
//   * Without `--all` we emit the single page response verbatim.
//   * With `--all` we concatenate the top-level list field across pages and
//     drop `nextPageToken` from the final aggregated object.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// addRawParamsFlags registers `--raw` and `--params` on a leaf command.
func addRawParamsFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("raw", false, "Emit raw Google API response JSON (no transform). With --all, concatenates list fields and drops nextPageToken.")
	cmd.Flags().String("params", "", "JSON object of raw API parameters. Keys override the equivalent CLI flags (params win).")
}

// isRaw reports whether `--raw` is set on the command.
func isRaw(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if f := cmd.Flags().Lookup("raw"); f == nil {
		return false
	}
	v, _ := cmd.Flags().GetBool("raw")
	return v
}

// parseParams returns the parsed `--params` payload, or nil when not set.
func parseParams(cmd *cobra.Command) (map[string]interface{}, error) {
	if cmd == nil {
		return nil, nil
	}
	if f := cmd.Flags().Lookup("params"); f == nil {
		return nil, nil
	}
	raw, _ := cmd.Flags().GetString("params")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("--params: invalid JSON: %w", err)
	}
	// Reject trailing junk after the object so scripts don't silently
	// drop a typo'd second value. A second Decode must return io.EOF;
	// anything else (another token, syntax error) is junk.
	var extra interface{}
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("--params: unexpected trailing data after JSON object")
	}
	return m, nil
}

// printRaw marshals v to stdout using SDK JSON tags directly so the output
// preserves the Google API shape. Honors the global --quiet flag: when
// quiet is set, output is suppressed (matching GetPrinter's NullPrinter
// contract so scripts can run raw commands quietly for side effects).
func printRaw(v interface{}) error {
	if quiet {
		return nil
	}
	return writeRaw(os.Stdout, v)
}

func writeRaw(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Param accessors. All accept the parsed map (which may be nil) and a key,
// and return (value, ok). They tolerate the common JSON encodings for each
// type (numbers may decode as float64 or json.Number).

func paramString(m map[string]interface{}, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func paramInt64(m map[string]interface{}, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	case float64:
		return int64(x), true
	case int:
		return int64(x), true
	case int64:
		return x, true
	case string:
		// Some callers pass numeric strings; accept them.
		var n int64
		if _, err := fmt.Sscan(x, &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

func paramBool(m map[string]interface{}, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// paramStringSlice supports both ["a","b"] and "a,b" forms.
func paramStringSlice(m map[string]interface{}, key string) ([]string, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil, false
	}
	switch x := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out, true
	case string:
		if x == "" {
			return nil, true
		}
		return strings.Split(x, ","), true
	}
	return nil, false
}
