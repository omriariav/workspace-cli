package printer

// AlreadyPrintedError wraps an error that has already been emitted to
// the error writer by a Printer. Callers (Execute, main) should NOT
// re-print the message — only inspect it for exit-code mapping.
type AlreadyPrintedError struct {
	Err error
}

func (e *AlreadyPrintedError) Error() string { return e.Err.Error() }
func (e *AlreadyPrintedError) Unwrap() error { return e.Err }
