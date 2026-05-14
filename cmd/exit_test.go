package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/printer"
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
