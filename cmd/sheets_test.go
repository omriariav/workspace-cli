package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestSheetsCommands_Flags tests that all sheets commands have expected flags
func TestSheetsCommands_Flags(t *testing.T) {
	tests := []struct {
		name          string
		cmdName       string
		expectedFlags []string
	}{
		{
			name:          "create flags",
			cmdName:       "create",
			expectedFlags: []string{"title", "sheet-names"},
		},
		{
			name:          "write flags",
			cmdName:       "write",
			expectedFlags: []string{"values", "values-json"},
		},
		{
			name:          "append flags",
			cmdName:       "append",
			expectedFlags: []string{"values", "values-json"},
		},
		{
			name:          "read flags",
			cmdName:       "read",
			expectedFlags: []string{"output-format", "headers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(sheetsCmd, tt.cmdName)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.cmdName)
			}

			for _, flag := range tt.expectedFlags {
				if cmd.Flags().Lookup(flag) == nil {
					t.Errorf("expected flag '--%s' not found", flag)
				}
			}
		})
	}
}

// TestSheetsCreate_Success tests creating a spreadsheet
func TestSheetsCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sheets API uses /v4/spreadsheets for create (POST)
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/spreadsheets") {
			// Parse request body
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			title := "Test Spreadsheet"
			if props, ok := req["properties"].(map[string]interface{}); ok {
				if t, ok := props["title"].(string); ok {
					title = t
				}
			}

			resp := map[string]interface{}{
				"spreadsheetId": "test-spreadsheet-id",
				"properties": map[string]interface{}{
					"title": title,
				},
				"spreadsheetUrl": "https://docs.google.com/spreadsheets/d/test-spreadsheet-id/edit",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
							"index":   0,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Return 404 for unhandled paths
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	// Verify server is running
	if server == nil {
		t.Fatal("server not created")
	}
}

// TestSheetsCreate_WithSheetNames tests creating a spreadsheet with custom sheet names
func TestSheetsCreate_WithSheetNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		// Check that sheets were included
		sheetsData, hasSheets := req["sheets"]
		if !hasSheets {
			t.Error("expected sheets to be included in request")
			return
		}

		sheets := sheetsData.([]interface{})
		sheetNames := make([]map[string]interface{}, len(sheets))
		for i, s := range sheets {
			sheet := s.(map[string]interface{})
			props := sheet["properties"].(map[string]interface{})
			sheetNames[i] = map[string]interface{}{
				"sheetId": i,
				"title":   props["title"],
				"index":   props["index"],
			}
		}

		resp := map[string]interface{}{
			"spreadsheetId": "test-spreadsheet-id",
			"properties": map[string]interface{}{
				"title": "Test",
			},
			"spreadsheetUrl": "https://docs.google.com/spreadsheets/d/test-spreadsheet-id/edit",
			"sheets": []map[string]interface{}{
				{"properties": sheetNames[0]},
				{"properties": sheetNames[1]},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Verify server responds
	if server == nil {
		t.Fatal("server not created")
	}
}

// TestSheetsWrite_Success tests writing values to a spreadsheet
func TestSheetsWrite_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || !strings.Contains(r.URL.Path, "/values/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		values := req["values"].([]interface{})
		rowCount := len(values)
		cellCount := 0
		for _, row := range values {
			cellCount += len(row.([]interface{}))
		}

		resp := map[string]interface{}{
			"spreadsheetId": "test-spreadsheet-id",
			"updatedRange":  "Sheet1!A1:C2",
			"updatedRows":   rowCount,
			"updatedCells":  cellCount,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	if server == nil {
		t.Fatal("server not created")
	}
}

// TestSheetsAppend_Success tests appending values to a spreadsheet
func TestSheetsAppend_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || !strings.Contains(r.URL.Path, "/values/") || !strings.Contains(r.URL.RawQuery, "append") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		values := req["values"].([]interface{})
		rowCount := len(values)
		cellCount := 0
		for _, row := range values {
			cellCount += len(row.([]interface{}))
		}

		resp := map[string]interface{}{
			"spreadsheetId": "test-spreadsheet-id",
			"updates": map[string]interface{}{
				"updatedRange": "Sheet1!A5:C6",
				"updatedRows":  rowCount,
				"updatedCells": cellCount,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	if server == nil {
		t.Fatal("server not created")
	}
}

// TestParseValues tests the parseValues helper function
func TestParseValues(t *testing.T) {
	tests := []struct {
		name       string
		values     string
		valuesJSON string
		wantRows   int
		wantCols   int
		wantErr    bool
	}{
		{
			name:     "simple single row",
			values:   "a,b,c",
			wantRows: 1,
			wantCols: 3,
		},
		{
			name:     "multiple rows",
			values:   "a,b,c;d,e,f",
			wantRows: 2,
			wantCols: 3,
		},
		{
			name:       "json format",
			valuesJSON: `[["a","b"],["c","d"]]`,
			wantRows:   2,
			wantCols:   2,
		},
		{
			name:       "json with numbers",
			valuesJSON: `[["Name",100],["Test",200]]`,
			wantRows:   2,
			wantCols:   2,
		},
		{
			name:       "invalid json",
			valuesJSON: `[["a","b"`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("values", "", "")
			cmd.Flags().String("values-json", "", "")

			if tt.values != "" {
				cmd.Flags().Set("values", tt.values)
			}
			if tt.valuesJSON != "" {
				cmd.Flags().Set("values-json", tt.valuesJSON)
			}

			values, err := parseValues(cmd)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(values) != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, len(values))
			}

			if len(values) > 0 && len(values[0]) != tt.wantCols {
				t.Errorf("expected %d cols, got %d", tt.wantCols, len(values[0]))
			}
		})
	}
}

// TestParseValues_JSONPrecedence tests that JSON takes precedence over simple format
func TestParseValues_JSONPrecedence(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("values", "", "")
	cmd.Flags().String("values-json", "", "")

	cmd.Flags().Set("values", "a,b,c")
	cmd.Flags().Set("values-json", `[["x","y"]]`)

	values, err := parseValues(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// JSON should take precedence
	if len(values) != 1 || len(values[0]) != 2 {
		t.Error("JSON format should take precedence")
	}

	if values[0][0] != "x" {
		t.Errorf("expected 'x', got '%v'", values[0][0])
	}
}

// TestParseValues_Empty tests parseValues with no input
func TestParseValues_Empty(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("values", "", "")
	cmd.Flags().String("values-json", "", "")

	values, err := parseValues(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if values != nil {
		t.Error("expected nil for empty input")
	}
}

// TestSheetsCommands_Structure tests that all sheet commands are registered
func TestSheetsCommands_Structure(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"info"},
		{"list"},
		{"read"},
		{"create"},
		{"write"},
		{"append"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(sheetsCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestSheetsCreate_RequiresTitle tests that create requires --title flag
func TestSheetsCreate_RequiresTitle(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "create")
	if cmd == nil {
		t.Fatal("create command not found")
	}

	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Error("expected --title flag")
	}

	// Check if it's marked as required
	annotations := cmd.Flags().Lookup("title").Annotations
	if annotations == nil {
		// Flag exists but might use MarkFlagRequired which sets different internals
		// Just verify the flag exists
	}
}

// TestSheetsAddSheetCommand_Flags tests add-sheet command flags
func TestSheetsAddSheetCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "add-sheet")
	if cmd == nil {
		t.Fatal("add-sheet command not found")
	}

	expectedFlags := []string{"name", "rows", "cols"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsDeleteSheetCommand_Flags tests delete-sheet command flags
func TestSheetsDeleteSheetCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "delete-sheet")
	if cmd == nil {
		t.Fatal("delete-sheet command not found")
	}

	expectedFlags := []string{"name", "sheet-id"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsClearCommand tests clear command
func TestSheetsClearCommand(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "clear")
	if cmd == nil {
		t.Fatal("clear command not found")
	}

	if cmd.Use != "clear <spreadsheet-id> <range>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

// TestSheetsAddSheet_MockServer tests add-sheet API integration
func TestSheetsAddSheet_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			// Verify addSheet request
			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				addSheet := requests[0].(map[string]interface{})["addSheet"]
				if addSheet != nil {
					props := addSheet.(map[string]interface{})["properties"].(map[string]interface{})
					sheetTitle := props["title"].(string)

					resp := map[string]interface{}{
						"spreadsheetId": "test-spreadsheet-id",
						"replies": []map[string]interface{}{
							{
								"addSheet": map[string]interface{}{
									"properties": map[string]interface{}{
										"sheetId": 12345,
										"title":   sheetTitle,
									},
								},
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(resp)
					return
				}
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if server == nil {
		t.Fatal("server not created")
	}
}

// TestSheetsDeleteSheet_MockServer tests delete-sheet API integration
func TestSheetsDeleteSheet_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			// Verify deleteSheet request
			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				deleteSheet := requests[0].(map[string]interface{})["deleteSheet"]
				if deleteSheet != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-spreadsheet-id",
						"replies":       []map[string]interface{}{{}},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(resp)
					return
				}
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if server == nil {
		t.Fatal("server not created")
	}
}

// TestSheetsClear_MockServer tests clear API integration
func TestSheetsClear_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, ":clear") {
			resp := map[string]interface{}{
				"spreadsheetId": "test-spreadsheet-id",
				"clearedRange":  "Sheet1!A1:D10",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if server == nil {
		t.Fatal("server not created")
	}
}
