package cmd

import (
	"errors"
	"testing"

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
