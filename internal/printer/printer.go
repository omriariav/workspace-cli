package printer

import (
	"io"
	"os"
)

// Printer is the interface for output formatters.
type Printer interface {
	Print(data interface{}) error
	PrintError(err error) error
}

// New creates a Printer that writes success output to w and errors to
// os.Stderr (the standard unix convention).
func New(w io.Writer, format string) Printer {
	return NewWithWriters(w, os.Stderr, format)
}

// NewWithWriters creates a Printer with explicit output and error writers.
// Useful in tests to capture stderr independently.
func NewWithWriters(w, errW io.Writer, format string) Printer {
	switch format {
	case "text":
		return NewTextPrinter(w, errW)
	case "yaml":
		return NewYAMLPrinter(w, errW)
	default:
		return NewJSONPrinter(w, errW)
	}
}
