package printer

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNew_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, "json")

	if _, ok := p.(*JSONPrinter); !ok {
		t.Error("expected JSONPrinter for 'json' format")
	}
}

func TestNew_Text(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, "text")

	if _, ok := p.(*TextPrinter); !ok {
		t.Error("expected TextPrinter for 'text' format")
	}
}

func TestNew_Default(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, "")

	if _, ok := p.(*JSONPrinter); !ok {
		t.Error("expected JSONPrinter for empty format (default)")
	}
}

func TestNullPrinter_Print(t *testing.T) {
	p := NewNullPrinter()

	data := map[string]interface{}{"key": "value"}
	if err := p.Print(data); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNullPrinter_PrintError(t *testing.T) {
	p := NewNullPrinter()

	err := errors.New("test error")
	if printErr := p.PrintError(err); printErr != nil {
		t.Errorf("unexpected error: %v", printErr)
	}
}

func TestNullPrinter_ImplementsPrinter(t *testing.T) {
	var p Printer = NewNullPrinter()
	_ = p // Verify interface compliance
}

func TestNew_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, "xml")

	// Unknown formats should default to JSON
	if _, ok := p.(*JSONPrinter); !ok {
		t.Error("expected JSONPrinter for unknown format")
	}
}

func TestJSONPrinter_Print_Map(t *testing.T) {
	var buf bytes.Buffer
	p := NewJSONPrinter(&buf)

	data := map[string]interface{}{
		"name": "test",
		"count": 42,
	}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name": "test"`) {
		t.Errorf("expected name in output: %s", output)
	}
	if !strings.Contains(output, `"count": 42`) {
		t.Errorf("expected count in output: %s", output)
	}
}

func TestJSONPrinter_Print_Slice(t *testing.T) {
	var buf bytes.Buffer
	p := NewJSONPrinter(&buf)

	data := []string{"a", "b", "c"}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"a"`) {
		t.Errorf("expected 'a' in output: %s", output)
	}
}

func TestJSONPrinter_Print_Nested(t *testing.T) {
	var buf bytes.Buffer
	p := NewJSONPrinter(&buf)

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		},
		"items": []int{1, 2, 3},
	}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name": "John"`) {
		t.Errorf("expected nested name in output: %s", output)
	}
}

func TestJSONPrinter_PrintError(t *testing.T) {
	var buf bytes.Buffer
	p := NewJSONPrinter(&buf)

	err := errors.New("something went wrong")
	if printErr := p.PrintError(err); printErr != nil {
		t.Fatalf("unexpected error: %v", printErr)
	}

	output := buf.String()
	if !strings.Contains(output, `"error": "something went wrong"`) {
		t.Errorf("expected error message in output: %s", output)
	}
}

func TestJSONPrinter_Indentation(t *testing.T) {
	var buf bytes.Buffer
	p := NewJSONPrinter(&buf)

	data := map[string]interface{}{"key": "value"}
	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Check for 2-space indentation
	if !strings.Contains(output, "  ") {
		t.Errorf("expected indented output: %s", output)
	}
}

func TestTextPrinter_Print_Map(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	data := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name:") {
		t.Errorf("expected name in output: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("expected value 'test' in output: %s", output)
	}
	if !strings.Contains(output, "count:") {
		t.Errorf("expected count in output: %s", output)
	}
}

func TestTextPrinter_Print_Map_SortedKeys(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	data := map[string]interface{}{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
	}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Keys should be sorted alphabetically
	appleIdx := strings.Index(output, "apple")
	mangoIdx := strings.Index(output, "mango")
	zebraIdx := strings.Index(output, "zebra")

	if appleIdx > mangoIdx || mangoIdx > zebraIdx {
		t.Errorf("expected keys in alphabetical order: %s", output)
	}
}

func TestTextPrinter_Print_Slice(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	data := []interface{}{
		map[string]interface{}{"id": 1, "name": "first"},
		map[string]interface{}{"id": 2, "name": "second"},
	}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "first") {
		t.Errorf("expected 'first' in output: %s", output)
	}
	if !strings.Contains(output, "second") {
		t.Errorf("expected 'second' in output: %s", output)
	}
	if !strings.Contains(output, "---") {
		t.Errorf("expected separator in output: %s", output)
	}
}

func TestTextPrinter_Print_Table(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	data := []map[string]interface{}{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
	}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should have header row
	if !strings.Contains(output, "id") || !strings.Contains(output, "name") {
		t.Errorf("expected headers in output: %s", output)
	}
	// Should have separator
	if !strings.Contains(output, "--") {
		t.Errorf("expected separator in output: %s", output)
	}
	// Should have data
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") {
		t.Errorf("expected data in output: %s", output)
	}
}

func TestTextPrinter_Print_EmptyTable(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	data := []map[string]interface{}{}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(no results)") {
		t.Errorf("expected '(no results)' for empty table: %s", output)
	}
}

func TestTextPrinter_Print_SimpleValue(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	if err := p.Print("hello world"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected simple value in output: %s", output)
	}
}

func TestTextPrinter_PrintError(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	err := errors.New("something went wrong")
	if printErr := p.PrintError(err); printErr != nil {
		t.Fatalf("unexpected error: %v", printErr)
	}

	output := buf.String()
	if !strings.Contains(output, "Error: something went wrong") {
		t.Errorf("expected error message in output: %s", output)
	}
}

func TestTextPrinter_Print_SliceOfStrings(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	// Test with a simple slice (not interface{})
	data := []interface{}{"one", "two", "three"}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "one") {
		t.Errorf("expected 'one' in output: %s", output)
	}
}

func TestTextPrinter_Print_ReflectSlice(t *testing.T) {
	var buf bytes.Buffer
	p := NewTextPrinter(&buf)

	// Using a concrete slice type to trigger reflection path
	type item struct {
		Name string
	}
	data := []item{{Name: "test"}}

	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}

	// Verify the actual content is present
	if !strings.Contains(output, "test") {
		t.Errorf("expected 'test' in output: %s", output)
	}
}
