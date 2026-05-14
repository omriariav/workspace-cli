package printer

import (
	"encoding/json"
	"io"
)

// JSONPrinter outputs data as indented JSON.
type JSONPrinter struct {
	w    io.Writer
	errW io.Writer
}

// NewJSONPrinter creates a new JSON printer with separate output and error
// writers. Success data goes to w; errors go to errW.
func NewJSONPrinter(w, errW io.Writer) *JSONPrinter {
	return &JSONPrinter{w: w, errW: errW}
}

// Print outputs data as indented JSON to stdout.
func (p *JSONPrinter) Print(data interface{}) error {
	encoder := json.NewEncoder(p.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintError writes a structured error to the error writer (stderr) and
// returns the original error wrapped in AlreadyPrintedError so callers
// propagate a non-zero exit without re-printing.
func (p *JSONPrinter) PrintError(err error) error {
	enc := json.NewEncoder(p.errW)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
	return &AlreadyPrintedError{Err: err}
}
