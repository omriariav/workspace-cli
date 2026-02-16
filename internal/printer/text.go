package printer

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"
)

// TextPrinter outputs data as human-readable text.
type TextPrinter struct {
	w io.Writer
}

// NewTextPrinter creates a new text printer.
func NewTextPrinter(w io.Writer) *TextPrinter {
	return &TextPrinter{w: w}
}

// Print outputs data as human-readable text.
func (p *TextPrinter) Print(data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		return p.printMap(v)
	case []interface{}:
		return p.printSlice(v)
	case []map[string]interface{}:
		return p.printTable(v)
	default:
		// Handle slices of structs or maps via reflection
		rv := reflect.ValueOf(data)
		if rv.Kind() == reflect.Slice {
			return p.printReflectSlice(rv)
		}
		// Fall back to simple print
		_, err := fmt.Fprintf(p.w, "%v\n", data)
		return err
	}
}

// PrintError outputs an error as text.
func (p *TextPrinter) PrintError(err error) error {
	_, writeErr := fmt.Fprintf(p.w, "Error: %s\n", err.Error())
	return writeErr
}

func (p *TextPrinter) printMap(m map[string]interface{}) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	for _, k := range keys {
		fmt.Fprintf(tw, "%s:\t%v\n", k, m[k])
	}
	return tw.Flush()
}

func (p *TextPrinter) printSlice(s []interface{}) error {
	for i, item := range s {
		if i > 0 {
			fmt.Fprintln(p.w, "---")
		}
		if m, ok := item.(map[string]interface{}); ok {
			_ = p.printMap(m)
		} else {
			fmt.Fprintf(p.w, "%v\n", item)
		}
	}
	return nil
}

func (p *TextPrinter) printTable(rows []map[string]interface{}) error {
	if len(rows) == 0 {
		fmt.Fprintln(p.w, "(no results)")
		return nil
	}

	// Collect all unique keys
	keySet := make(map[string]bool)
	for _, row := range rows {
		for k := range row {
			keySet[k] = true
		}
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, strings.Join(keys, "\t"))

	// Separator
	seps := make([]string, len(keys))
	for i, k := range keys {
		seps[i] = strings.Repeat("-", len(k))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Rows
	for _, row := range rows {
		vals := make([]string, len(keys))
		for i, k := range keys {
			if v, ok := row[k]; ok {
				vals[i] = fmt.Sprintf("%v", v)
			}
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	return tw.Flush()
}

func (p *TextPrinter) printReflectSlice(rv reflect.Value) error {
	if rv.Len() == 0 {
		fmt.Fprintln(p.w, "(no results)")
		return nil
	}

	// Convert to []map[string]interface{} if possible
	rows := make([]map[string]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, m)
		} else {
			// Fall back to simple list
			for j := 0; j < rv.Len(); j++ {
				fmt.Fprintf(p.w, "%v\n", rv.Index(j).Interface())
			}
			return nil
		}
	}

	return p.printTable(rows)
}
