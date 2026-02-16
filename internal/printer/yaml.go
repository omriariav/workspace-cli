package printer

import (
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLPrinter outputs data as YAML.
type YAMLPrinter struct {
	w io.Writer
}

// NewYAMLPrinter creates a new YAML printer.
func NewYAMLPrinter(w io.Writer) *YAMLPrinter {
	return &YAMLPrinter{w: w}
}

// Print outputs data as YAML.
func (p *YAMLPrinter) Print(data interface{}) error {
	enc := yaml.NewEncoder(p.w)
	enc.SetIndent(2)
	return enc.Encode(data)
}

// PrintError outputs an error as YAML.
func (p *YAMLPrinter) PrintError(err error) error {
	return p.Print(map[string]interface{}{
		"error": err.Error(),
	})
}
