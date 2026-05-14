package printer

import (
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLPrinter outputs data as YAML.
type YAMLPrinter struct {
	w    io.Writer
	errW io.Writer
}

// NewYAMLPrinter creates a new YAML printer with separate output and error
// writers.
func NewYAMLPrinter(w, errW io.Writer) *YAMLPrinter {
	return &YAMLPrinter{w: w, errW: errW}
}

// Print outputs data as YAML to stdout.
func (p *YAMLPrinter) Print(data interface{}) error {
	enc := yaml.NewEncoder(p.w)
	enc.SetIndent(2)
	return enc.Encode(data)
}

// PrintError writes a structured error to the error writer (stderr) and
// returns the original error wrapped in AlreadyPrintedError.
func (p *YAMLPrinter) PrintError(err error) error {
	enc := yaml.NewEncoder(p.errW)
	enc.SetIndent(2)
	_ = enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
	return &AlreadyPrintedError{Err: err}
}
