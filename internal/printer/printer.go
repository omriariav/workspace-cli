package printer

import (
	"io"
)

// Printer is the interface for output formatters.
type Printer interface {
	Print(data interface{}) error
	PrintError(err error) error
}

// New creates a new Printer based on the format string.
func New(w io.Writer, format string) Printer {
	switch format {
	case "text":
		return NewTextPrinter(w)
	case "yaml":
		return NewYAMLPrinter(w)
	default:
		return NewJSONPrinter(w)
	}
}
