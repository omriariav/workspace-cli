package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi"
)

func TestExitCodeForError_GenericError(t *testing.T) {
	err := errors.New("something failed")
	if got := exitCodeForError(err); got != ExitError {
		t.Errorf("expected ExitError (1), got %d", got)
	}
}

func TestExitCodeForError_Auth401(t *testing.T) {
	err := &googleapi.Error{Code: 401, Message: "unauthorized"}
	if got := exitCodeForError(err); got != ExitAuth {
		t.Errorf("expected ExitAuth (3), got %d", got)
	}
}

func TestExitCodeForError_Auth403(t *testing.T) {
	err := &googleapi.Error{Code: 403, Message: "forbidden"}
	if got := exitCodeForError(err); got != ExitAuth {
		t.Errorf("expected ExitAuth (3), got %d", got)
	}
}

func TestExitCodeForError_Transient429(t *testing.T) {
	err := &googleapi.Error{Code: 429, Message: "rate limited"}
	if got := exitCodeForError(err); got != ExitTransient {
		t.Errorf("expected ExitTransient (4), got %d", got)
	}
}

func TestExitCodeForError_Transient500(t *testing.T) {
	err := &googleapi.Error{Code: 500, Message: "internal error"}
	if got := exitCodeForError(err); got != ExitTransient {
		t.Errorf("expected ExitTransient (4), got %d", got)
	}
}

func TestExitCodeForError_Transient503(t *testing.T) {
	err := &googleapi.Error{Code: 503, Message: "service unavailable"}
	if got := exitCodeForError(err); got != ExitTransient {
		t.Errorf("expected ExitTransient (4), got %d", got)
	}
}

func TestExitCodeForError_Client404(t *testing.T) {
	err := &googleapi.Error{Code: 404, Message: "not found"}
	if got := exitCodeForError(err); got != ExitError {
		t.Errorf("expected ExitError (1) for 404, got %d", got)
	}
}

func TestExitCodeForError_WrappedGoogleAPIError(t *testing.T) {
	inner := &googleapi.Error{Code: 403, Message: "forbidden"}
	wrapped := errors.Join(errors.New("context"), inner)
	if got := exitCodeForError(wrapped); got != ExitAuth {
		t.Errorf("expected ExitAuth (3) for wrapped 403, got %d", got)
	}
}

// --- resolveExitError: full dispatch contract ----------------------------

func TestResolveExitError_NilReturnsOK(t *testing.T) {
	var buf bytes.Buffer
	if got := resolveExitError(nil, &buf); got != ExitOK {
		t.Errorf("expected ExitOK for nil error, got %d", got)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no stderr write for nil, got %q", buf.String())
	}
}

func TestResolveExitError_PrintedGenericMapsToOne(t *testing.T) {
	var buf bytes.Buffer
	printed := &printer.AlreadyPrintedError{Err: errors.New("something failed")}
	if got := resolveExitError(printed, &buf); got != ExitError {
		t.Errorf("expected ExitError (1) for AlreadyPrintedError, got %d", got)
	}
	// AlreadyPrintedError means the printer already wrote to stderr;
	// resolveExitError must not re-print.
	if buf.Len() != 0 {
		t.Errorf("expected no extra stderr write for printed error, got %q", buf.String())
	}
}

func TestResolveExitError_PrintedAuthMapsToThree(t *testing.T) {
	var buf bytes.Buffer
	printed := &printer.AlreadyPrintedError{Err: &googleapi.Error{Code: 403, Message: "forbidden"}}
	if got := resolveExitError(printed, &buf); got != ExitAuth {
		t.Errorf("expected ExitAuth (3) for printed 403, got %d", got)
	}
}

func TestResolveExitError_PrintedTransientMapsToFour(t *testing.T) {
	var buf bytes.Buffer
	printed := &printer.AlreadyPrintedError{Err: &googleapi.Error{Code: 503, Message: "unavailable"}}
	if got := resolveExitError(printed, &buf); got != ExitTransient {
		t.Errorf("expected ExitTransient (4) for printed 503, got %d", got)
	}
}

func TestResolveExitError_CobraUsageMapsToTwoAndWritesStderr(t *testing.T) {
	// Plain error (not AlreadyPrintedError) simulates a Cobra arg/flag
	// validation failure that came back from rootCmd.Execute.
	var buf bytes.Buffer
	usageErr := fmt.Errorf("unknown flag: --bogus")
	if got := resolveExitError(usageErr, &buf); got != ExitUsage {
		t.Errorf("expected ExitUsage (2) for unprinted error, got %d", got)
	}
	out := buf.String()
	if !strings.Contains(out, "Error: unknown flag: --bogus") {
		t.Errorf("expected Cobra error on stderr, got %q", out)
	}
}

// TestResolveExitError_UnwrappedRuntimeErrorMapsToOne verifies that an
// unprinted error which is NOT a Cobra usage error (e.g. a RunE returning
// a write error from p.Print) maps to ExitError, not ExitUsage. This is
// the round-3 fix — previously every non-AlreadyPrintedError mapped to 2.
func TestResolveExitError_UnwrappedRuntimeErrorMapsToOne(t *testing.T) {
	var buf bytes.Buffer
	runtimeErr := fmt.Errorf("write /dev/stdout: broken pipe")
	if got := resolveExitError(runtimeErr, &buf); got != ExitError {
		t.Errorf("expected ExitError (1) for unwrapped runtime error, got %d", got)
	}
	if !strings.Contains(buf.String(), "broken pipe") {
		t.Errorf("expected stderr to include error message, got %q", buf.String())
	}
}

// TestResolveExitError_CobraEndToEnd_UnknownFlag exercises a real Cobra
// command tree end-to-end so we catch regressions in SilenceUsage and the
// usage-error classification. Uses an isolated *cobra.Command so we don't
// touch the package-level rootCmd.
func TestResolveExitError_CobraEndToEnd_UnknownFlag(t *testing.T) {
	root := &cobra.Command{
		Use:           "test-root",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	leaf := &cobra.Command{
		Use:  "leaf",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	root.AddCommand(leaf)
	root.SetArgs([]string{"leaf", "--bogus"})

	// Capture root's own stderr/usage outputs (Cobra writes usage via
	// SetErr / SetOut). SilenceUsage:true should keep these empty.
	var cobraOut, cobraErr bytes.Buffer
	root.SetOut(&cobraOut)
	root.SetErr(&cobraErr)

	var resolverErr bytes.Buffer
	code := resolveExitError(root.Execute(), &resolverErr)

	if code != ExitUsage {
		t.Errorf("expected ExitUsage (2) for unknown flag, got %d", code)
	}
	if !strings.Contains(resolverErr.String(), "unknown flag") {
		t.Errorf("expected resolver stderr to mention unknown flag, got %q", resolverErr.String())
	}
	// Auto-usage dump must be suppressed.
	if strings.Contains(cobraErr.String(), "Usage:") || strings.Contains(cobraOut.String(), "Usage:") {
		t.Errorf("expected no Cobra usage dump under SilenceUsage; got out=%q err=%q", cobraOut.String(), cobraErr.String())
	}
}

// TestResolveExitError_EndToEnd_RealCommandUsageValidation exercises a
// real `gws` command path end-to-end: invoke chat setup-space with an
// invalid --type (a runtime validation that uses usageErrorf), and
// verify resolveExitError returns ExitUsage. This is the integration
// test codex requested — it touches the actual command registry, not
// a synthetic *cobra.Command.
func TestResolveExitError_EndToEnd_RealCommandUsageValidation(t *testing.T) {
	cmd := findSubcommand(chatCmd, "setup-space")
	if cmd == nil {
		t.Fatal("chat setup-space command not registered")
	}
	if err := cmd.Flags().Set("display-name", "x"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("type", "BOGUS_TYPE"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cmd.Flags().Set("display-name", "")
		cmd.Flags().Set("type", "")
	})

	// usageErrorf writes to os.Stderr. We don't capture it here — the
	// point of this test is the exit code, not the message format
	// (covered elsewhere).
	oldStderr := os.Stderr
	devnull, _ := os.Open(os.DevNull)
	os.Stderr = devnull
	t.Cleanup(func() {
		os.Stderr = oldStderr
		_ = devnull.Close()
	})

	err := cmd.RunE(cmd, []string{})
	var resolverErr bytes.Buffer
	code := resolveExitError(err, &resolverErr)
	if code != ExitUsage {
		t.Errorf("expected ExitUsage (2) for invalid --type, got %d", code)
	}
}

// TestResolveExitError_CobraEndToEnd_BadArgs covers the other common
// Cobra validation path (cobra.ExactArgs).
func TestResolveExitError_CobraEndToEnd_BadArgs(t *testing.T) {
	root := &cobra.Command{Use: "test-root", SilenceErrors: true, SilenceUsage: true}
	leaf := &cobra.Command{
		Use:  "leaf",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	root.AddCommand(leaf)
	root.SetArgs([]string{"leaf"}) // missing required arg
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	var resolverErr bytes.Buffer
	code := resolveExitError(root.Execute(), &resolverErr)
	if code != ExitUsage {
		t.Errorf("expected ExitUsage (2) for missing arg, got %d", code)
	}
	if !strings.Contains(resolverErr.String(), "accepts") {
		t.Errorf("expected stderr to mention 'accepts ... arg(s)', got %q", resolverErr.String())
	}
}

// TestResolveExitError_UsageErrorfMapsToTwo verifies that the runtime
// input-validation helper produces the same exit code as Cobra's own
// usage errors. usageErrorf already wrote to stderr, so resolveExitError
// must NOT re-print.
func TestResolveExitError_UsageErrorfMapsToTwo(t *testing.T) {
	// Simulate what usageErrorf returns (its stderr write is a side
	// effect we don't drive here).
	wrapped := &printer.AlreadyPrintedError{Err: &usageError{msg: "people get: resourceName is required"}}

	var buf bytes.Buffer
	code := resolveExitError(wrapped, &buf)
	if code != ExitUsage {
		t.Errorf("expected ExitUsage (2) for usageError, got %d", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no extra stderr write (usageErrorf already wrote), got %q", buf.String())
	}
}

func TestIsCobraUsageError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"unknown command \"foo\" for \"gws\"", true},
		{"unknown flag: --bogus", true},
		{"unknown shorthand flag: 'x'", true},
		{"flag needs an argument: --to", true},
		{"invalid argument \"abc\" for \"--max\"", true},
		{"required flag(s) \"to\" not set", true},
		{"accepts 1 arg(s), received 0", true},
		{"requires at least 1 arg(s)", true},
		{"requires exactly 2 arg(s)", true},
		{"subcommand is required", true},

		{"failed to get person: googleapi: Error 403", false},
		{"write /dev/stdout: broken pipe", false},
		{"something random", false},
		{"", false},
	}
	for _, c := range cases {
		got := isCobraUsageError(fmt.Errorf("%s", c.msg))
		if c.msg == "" {
			got = isCobraUsageError(nil)
		}
		if got != c.want {
			t.Errorf("isCobraUsageError(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

func TestResolveExitError_JoinedEncodeFailureStillMapsCorrectly(t *testing.T) {
	// Mirrors the new "join encode error" behavior of PrintError: even
	// if the printer's stderr encode failed, AlreadyPrintedError is
	// still in the chain and the exit code must be correct.
	var buf bytes.Buffer
	chain := errors.Join(
		&printer.AlreadyPrintedError{Err: &googleapi.Error{Code: 429, Message: "rate limited"}},
		errors.New("stderr broken: write: broken pipe"),
	)
	if got := resolveExitError(chain, &buf); got != ExitTransient {
		t.Errorf("expected ExitTransient (4) for joined error, got %d", got)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no stderr write for joined AlreadyPrintedError, got %q", buf.String())
	}
}
