package printer

// NullPrinter discards all output. Used with --quiet flag.
type NullPrinter struct{}

// NewNullPrinter creates a new NullPrinter.
func NewNullPrinter() *NullPrinter {
	return &NullPrinter{}
}

// Print discards the data.
func (p *NullPrinter) Print(data interface{}) error {
	return nil
}

// PrintError suppresses output but returns the error wrapped in
// AlreadyPrintedError so the process still exits non-zero.
func (p *NullPrinter) PrintError(err error) error {
	return &AlreadyPrintedError{Err: err}
}
