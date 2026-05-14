package cmd

// Runtime input-validation errors. A handler that catches "you forgot a
// required field" can return usageErrorf(...) to signal:
//   - exit code = ExitUsage (2) — same as Cobra's own arg/flag errors
//   - output    = plain text "Error: <msg>" on stderr (not structured JSON)
//
// This complements Cobra-level validation: Cobra catches what it can
// declaratively (arg counts, unknown flags); usageError covers checks
// that depend on flag values, --params content, or any data Cobra cannot
// validate up front.
//
// CONVENTION for new code:
//   - Validating user input ("must specify X", "X is required",
//     "--from must be >= 1", invalid enum value)  → usageErrorf(...)
//   - Reporting state/post-condition failure ("no cache found",
//     "document has no content", "sheet 'X' not found",
//     "you are not an attendee", "unexpected empty response")
//                                                  → p.PrintError(...)
//   - Wrapping a Google API error (fmt.Errorf("failed to X: %w", err))
//                                                  → p.PrintError(...)
//     (exitCodeForError will map 401/403/429/5xx to 3/4 automatically)
//
// State-only sites intentionally exit 1, since their cause is server-
// or file-side, not the user's invocation.

import (
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/printer"
)

// usageError marks an error as a CLI-input mistake. resolveExitError
// surfaces these as ExitUsage.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// usageErrorf prints a plain-text error line to stderr and returns
// AlreadyPrintedError wrapping a usageError. Callsites use it exactly
// like fmt.Errorf so the migration from "p.PrintError(fmt.Errorf(...))"
// to "usageErrorf(...)" is a single-line swap.
func usageErrorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	return &printer.AlreadyPrintedError{Err: &usageError{msg: msg}}
}
