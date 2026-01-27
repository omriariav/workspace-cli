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

// PrintError discards the error.
func (p *NullPrinter) PrintError(err error) error {
	return nil
}
