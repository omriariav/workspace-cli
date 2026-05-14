package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stderr during a function call and returns the output.
// (PrintError writes structured errors to stderr per #190.)
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// extractError returns the error message from either format the CLI may
// produce on stderr:
//   - structured JSON: {"error": "..."} (runtime/API failures)
//   - plain text:      "Error: ..."     (CLI usage errors)
//
// Returns "" if no recognizable error shape is found.
func extractError(output string) string {
	trim := strings.TrimSpace(output)
	if trim == "" {
		return ""
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(trim), &result); err == nil {
		if errMsg, ok := result["error"].(string); ok {
			return errMsg
		}
	}
	for _, line := range strings.Split(trim, "\n") {
		if strings.HasPrefix(line, "Error: ") {
			return strings.TrimPrefix(line, "Error: ")
		}
	}
	return trim
}

// --- Forms validation tests ---

func TestFormsUpdate_Validation(t *testing.T) {
	// Test no-flags case first (before any flags are mutated on the shared command)
	t.Run("no_flags", func(t *testing.T) {
		cmd := findSubcommand(formsCmd, "update")
		if cmd == nil {
			t.Fatal("forms update command not found")
		}

		output := captureOutput(t, func() {
			cmd.RunE(cmd, []string{"form-id"})
		})

		errMsg := extractError(output)
		if !strings.Contains(errMsg, "provide --file, --title, or --description") {
			t.Errorf("expected missing-flags error, got output: %s", output)
		}
	})

	t.Run("file_and_title_conflict", func(t *testing.T) {
		cmd := findSubcommand(formsCmd, "update")
		if cmd == nil {
			t.Fatal("forms update command not found")
		}

		cmd.Flags().Set("file", "test.json")
		cmd.Flags().Set("title", "New Title")

		output := captureOutput(t, func() {
			cmd.RunE(cmd, []string{"form-id"})
		})

		errMsg := extractError(output)
		if !strings.Contains(errMsg, "cannot be combined") {
			t.Errorf("expected conflict error, got output: %s", output)
		}
	})
}

// --- Sheets validation tests ---

func TestSheetsAddConditionalFormat_ValueRequired(t *testing.T) {
	rulesNeedingValue := []string{">", "<", "=", "!=", "contains", "not-contains", "formula"}
	for _, rule := range rulesNeedingValue {
		t.Run(rule, func(t *testing.T) {
			cmd := findSubcommand(sheetsCmd, "add-conditional-format")
			if cmd == nil {
				t.Fatal("sheets add-conditional-format command not found")
			}

			cmd.Flags().Set("rule", rule)
			defer cmd.Flags().Set("rule", "")

			output := captureOutput(t, func() {
				cmd.RunE(cmd, []string{"spreadsheet-id", "Sheet1!A1:A10"})
			})

			errMsg := extractError(output)
			if !strings.Contains(errMsg, "--value is required") {
				t.Errorf("expected --value required error for rule %q, got output: %s", rule, output)
			}
		})
	}
}

func TestSheetsDeleteConditionalFormat_NegativeIndex(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "delete-conditional-format")
	if cmd == nil {
		t.Fatal("sheets delete-conditional-format command not found")
	}

	cmd.Flags().Set("sheet", "Sheet1")
	cmd.Flags().Set("index", "-1")
	defer func() {
		cmd.Flags().Set("sheet", "")
		cmd.Flags().Set("index", "0")
	}()

	output := captureOutput(t, func() {
		cmd.RunE(cmd, []string{"spreadsheet-id"})
	})

	errMsg := extractError(output)
	if !strings.Contains(errMsg, "--index must be >= 0") {
		t.Errorf("expected negative index error, got output: %s", output)
	}
}

// --- Slides validation tests ---

func TestSlidesThumbnail_InvalidSize(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "thumbnail")
	if cmd == nil {
		t.Fatal("slides thumbnail command not found")
	}

	cmd.Flags().Set("slide", "1")
	cmd.Flags().Set("size", "XLARGE")
	defer func() {
		cmd.Flags().Set("slide", "")
		cmd.Flags().Set("size", "MEDIUM")
	}()

	output := captureOutput(t, func() {
		cmd.RunE(cmd, []string{"presentation-id"})
	})

	errMsg := extractError(output)
	if !strings.Contains(errMsg, "invalid size") {
		t.Errorf("expected invalid size error, got output: %s", output)
	}
}

// --- Contacts validation tests ---

func TestContactsUpdate_NoFlags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "update")
	if cmd == nil {
		t.Fatal("contacts update command not found")
	}

	output := captureOutput(t, func() {
		cmd.RunE(cmd, []string{"people/c123"})
	})

	errMsg := extractError(output)
	if !strings.Contains(errMsg, "at least one field") {
		t.Errorf("expected missing-flags error, got output: %s", output)
	}
}
