package printer

import (
	"encoding/json"
	"io"
)

// JSONPrinter outputs data as indented JSON.
type JSONPrinter struct {
	w io.Writer
}

// NewJSONPrinter creates a new JSON printer.
func NewJSONPrinter(w io.Writer) *JSONPrinter {
	return &JSONPrinter{w: w}
}

// Print outputs data as indented JSON.
func (p *JSONPrinter) Print(data interface{}) error {
	encoder := json.NewEncoder(p.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintError outputs an error as JSON.
func (p *JSONPrinter) PrintError(err error) error {
	return p.Print(map[string]interface{}{
		"error": err.Error(),
	})
}
