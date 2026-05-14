package printer

import (
	"errors"
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
// returns the original error wrapped in AlreadyPrintedError. Encode errors
// are joined onto the returned chain.
func (p *YAMLPrinter) PrintError(err error) error {
	enc := yaml.NewEncoder(p.errW)
	enc.SetIndent(2)
	encErr := enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
	printed := &AlreadyPrintedError{Err: err}
	if encErr != nil {
		return errors.Join(printed, encErr)
	}
	return printed
}
