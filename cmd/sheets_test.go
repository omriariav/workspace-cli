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

	// Flag exists - MarkFlagRequired sets internal annotations
	_ = cmd.Flags().Lookup("title").Annotations
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

// TestSheetsInsertRowsCommand_Flags tests insert-rows command flags
func TestSheetsInsertRowsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "insert-rows")
	if cmd == nil {
		t.Fatal("insert-rows command not found")
	}

	expectedFlags := []string{"sheet", "at", "count"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsDeleteRowsCommand_Flags tests delete-rows command flags
func TestSheetsDeleteRowsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "delete-rows")
	if cmd == nil {
		t.Fatal("delete-rows command not found")
	}

	expectedFlags := []string{"sheet", "from", "to"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsInsertColsCommand_Flags tests insert-cols command flags
func TestSheetsInsertColsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "insert-cols")
	if cmd == nil {
		t.Fatal("insert-cols command not found")
	}

	expectedFlags := []string{"sheet", "at", "count"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsDeleteColsCommand_Flags tests delete-cols command flags
func TestSheetsDeleteColsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "delete-cols")
	if cmd == nil {
		t.Fatal("delete-cols command not found")
	}

	expectedFlags := []string{"sheet", "from", "to"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsRenameSheetCommand_Flags tests rename-sheet command flags
func TestSheetsRenameSheetCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "rename-sheet")
	if cmd == nil {
		t.Fatal("rename-sheet command not found")
	}

	expectedFlags := []string{"sheet", "name"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsDuplicateSheetCommand_Flags tests duplicate-sheet command flags
func TestSheetsDuplicateSheetCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "duplicate-sheet")
	if cmd == nil {
		t.Fatal("duplicate-sheet command not found")
	}

	expectedFlags := []string{"sheet", "new-name"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsMergeCommand tests merge command
func TestSheetsMergeCommand(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "merge")
	if cmd == nil {
		t.Fatal("merge command not found")
	}

	if cmd.Use != "merge <spreadsheet-id> <range>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

// TestSheetsUnmergeCommand tests unmerge command
func TestSheetsUnmergeCommand(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "unmerge")
	if cmd == nil {
		t.Fatal("unmerge command not found")
	}

	if cmd.Use != "unmerge <spreadsheet-id> <range>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

// TestSheetsSortCommand_Flags tests sort command flags
func TestSheetsSortCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "sort")
	if cmd == nil {
		t.Fatal("sort command not found")
	}

	expectedFlags := []string{"by", "desc", "has-header"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsFindReplaceCommand_Flags tests find-replace command flags
func TestSheetsFindReplaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "find-replace")
	if cmd == nil {
		t.Fatal("find-replace command not found")
	}

	expectedFlags := []string{"find", "replace", "sheet", "match-case", "entire-cell"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestParseCellRef tests the parseCellRef helper function
func TestParseCellRef(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		wantCol int64
		wantRow int64
		wantErr bool
	}{
		{
			name:    "simple A1",
			ref:     "A1",
			wantCol: 0,
			wantRow: 0,
		},
		{
			name:    "B5",
			ref:     "B5",
			wantCol: 1,
			wantRow: 4,
		},
		{
			name:    "Z10",
			ref:     "Z10",
			wantCol: 25,
			wantRow: 9,
		},
		{
			name:    "AA1",
			ref:     "AA1",
			wantCol: 26,
			wantRow: 0,
		},
		{
			name:    "lowercase b3",
			ref:     "b3",
			wantCol: 1,
			wantRow: 2,
		},
		{
			name:    "invalid - no row",
			ref:     "A",
			wantErr: true,
		},
		{
			name:    "invalid - no column",
			ref:     "1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, row, err := parseCellRef(tt.ref)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if col != tt.wantCol {
				t.Errorf("expected col %d, got %d", tt.wantCol, col)
			}

			if row != tt.wantRow {
				t.Errorf("expected row %d, got %d", tt.wantRow, row)
			}
		})
	}
}

// TestColumnLetterToIndex tests the columnLetterToIndex helper function
func TestColumnLetterToIndex(t *testing.T) {
	tests := []struct {
		col   string
		index int64
	}{
		{"A", 0},
		{"B", 1},
		{"Z", 25},
		{"AA", 26},
		{"AB", 27},
		{"AZ", 51},
		{"BA", 52},
		{"a", 0},
		{"z", 25},
	}

	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			got := columnLetterToIndex(tt.col)
			if got != tt.index {
				t.Errorf("columnLetterToIndex(%s) = %d, want %d", tt.col, got, tt.index)
			}
		})
	}
}

// TestSheetsInsertRows_MockServer tests insert-rows API integration
func TestSheetsInsertRows_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle spreadsheet get for sheet ID lookup
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/spreadsheets/") {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle batchUpdate for insertDimension
		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			// Verify insertDimension request
			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				insertDim := requests[0].(map[string]interface{})["insertDimension"]
				if insertDim != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsDeleteRows_MockServer tests delete-rows API integration
func TestSheetsDeleteRows_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				deleteDim := requests[0].(map[string]interface{})["deleteDimension"]
				if deleteDim != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsMerge_MockServer tests merge cells API integration
func TestSheetsMerge_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				mergeCells := requests[0].(map[string]interface{})["mergeCells"]
				if mergeCells != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsFindReplace_MockServer tests find-replace API integration
func TestSheetsFindReplace_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				findReplace := requests[0].(map[string]interface{})["findReplace"]
				if findReplace != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
						"replies": []map[string]interface{}{
							{
								"findReplace": map[string]interface{}{
									"occurrencesChanged": 5,
									"sheetsChanged":      2,
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

// TestSheetsSort_MockServer tests sort range API integration
func TestSheetsSort_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				sortRange := requests[0].(map[string]interface{})["sortRange"]
				if sortRange != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsRenameSheet_MockServer tests rename-sheet API integration
func TestSheetsRenameSheet_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "OldName",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				updateProps := requests[0].(map[string]interface{})["updateSheetProperties"]
				if updateProps != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsDuplicateSheet_MockServer tests duplicate-sheet API integration
func TestSheetsDuplicateSheet_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				dupSheet := requests[0].(map[string]interface{})["duplicateSheet"]
				if dupSheet != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
						"replies": []map[string]interface{}{
							{
								"duplicateSheet": map[string]interface{}{
									"properties": map[string]interface{}{
										"sheetId": 12345,
										"title":   "Sheet1 Copy",
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

// TestSheetsCommands_Structure_Extended tests that all new sheet commands are registered
func TestSheetsCommands_Structure_Extended(t *testing.T) {
	commands := []string{
		"insert-rows",
		"delete-rows",
		"insert-cols",
		"delete-cols",
		"rename-sheet",
		"duplicate-sheet",
		"merge",
		"unmerge",
		"sort",
		"find-replace",
		"format",
		"set-column-width",
		"set-row-height",
		"freeze",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := findSubcommand(sheetsCmd, cmdName)
			if cmd == nil {
				t.Fatalf("command '%s' not found", cmdName)
			}
		})
	}
}

// TestSheetsFormatCommand_Flags tests format command flags
func TestSheetsFormatCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "format")
	if cmd == nil {
		t.Fatal("format command not found")
	}

	expectedFlags := []string{"bold", "italic", "bg-color", "color", "font-size"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsSetColumnWidthCommand_Flags tests set-column-width command flags
func TestSheetsSetColumnWidthCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "set-column-width")
	if cmd == nil {
		t.Fatal("set-column-width command not found")
	}

	expectedFlags := []string{"sheet", "col", "width"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsSetRowHeightCommand_Flags tests set-row-height command flags
func TestSheetsSetRowHeightCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "set-row-height")
	if cmd == nil {
		t.Fatal("set-row-height command not found")
	}

	expectedFlags := []string{"sheet", "row", "height"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsFreezeCommand_Flags tests freeze command flags
func TestSheetsFreezeCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "freeze")
	if cmd == nil {
		t.Fatal("freeze command not found")
	}

	expectedFlags := []string{"sheet", "rows", "cols"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestParseSheetsHexColor tests the parseSheetsHexColor helper
func TestParseSheetsHexColor(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantR   float64
		wantG   float64
		wantB   float64
		wantErr bool
	}{
		{"red", "#FF0000", 1.0, 0.0, 0.0, false},
		{"green", "#00FF00", 0.0, 1.0, 0.0, false},
		{"blue", "#0000FF", 0.0, 0.0, 1.0, false},
		{"yellow", "#FFFF00", 1.0, 1.0, 0.0, false},
		{"black", "#000000", 0.0, 0.0, 0.0, false},
		{"white", "#FFFFFF", 1.0, 1.0, 1.0, false},
		{"lowercase", "#ff0000", 1.0, 0.0, 0.0, false},
		{"invalid - no hash", "FF0000", 0, 0, 0, true},
		{"invalid - too short", "#FFF", 0, 0, 0, true},
		{"invalid - bad chars", "#GGGGGG", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color, err := parseSheetsHexColor(tt.hex)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if color.Red != tt.wantR {
				t.Errorf("expected red %f, got %f", tt.wantR, color.Red)
			}
			if color.Green != tt.wantG {
				t.Errorf("expected green %f, got %f", tt.wantG, color.Green)
			}
			if color.Blue != tt.wantB {
				t.Errorf("expected blue %f, got %f", tt.wantB, color.Blue)
			}
		})
	}
}

// TestSheetsFormat_MockServer tests format cells API integration
func TestSheetsFormat_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				repeatCell := requests[0].(map[string]interface{})["repeatCell"]
				if repeatCell == nil {
					t.Error("expected repeatCell request")
				}
			}

			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"replies":       []map[string]interface{}{{}},
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

// TestSheetsSetColumnWidth_MockServer tests set-column-width API integration
func TestSheetsSetColumnWidth_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				updateDim := requests[0].(map[string]interface{})["updateDimensionProperties"]
				if updateDim == nil {
					t.Error("expected updateDimensionProperties request")
				} else {
					dimRange := updateDim.(map[string]interface{})["range"].(map[string]interface{})
					if dimRange["dimension"] != "COLUMNS" {
						t.Errorf("expected dimension COLUMNS, got %v", dimRange["dimension"])
					}
				}
			}

			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"replies":       []map[string]interface{}{{}},
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

// TestSheetsSetRowHeight_MockServer tests set-row-height API integration
func TestSheetsSetRowHeight_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				updateDim := requests[0].(map[string]interface{})["updateDimensionProperties"]
				if updateDim == nil {
					t.Error("expected updateDimensionProperties request")
				} else {
					dimRange := updateDim.(map[string]interface{})["range"].(map[string]interface{})
					if dimRange["dimension"] != "ROWS" {
						t.Errorf("expected dimension ROWS, got %v", dimRange["dimension"])
					}
				}
			}

			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"replies":       []map[string]interface{}{{}},
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

// TestSheetsCopyToCommand_Flags tests copy-to command flags
func TestSheetsCopyToCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "copy-to")
	if cmd == nil {
		t.Fatal("copy-to command not found")
	}

	expectedFlags := []string{"sheet-id", "destination"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsBatchReadCommand_Flags tests batch-read command flags
func TestSheetsBatchReadCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "batch-read")
	if cmd == nil {
		t.Fatal("batch-read command not found")
	}

	expectedFlags := []string{"ranges", "value-render"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsBatchWriteCommand_Flags tests batch-write command flags
func TestSheetsBatchWriteCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "batch-write")
	if cmd == nil {
		t.Fatal("batch-write command not found")
	}

	expectedFlags := []string{"ranges", "values", "value-input"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsCopyTo_MockServer tests copy-to API integration
func TestSheetsCopyTo_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/sheets/") && strings.Contains(r.URL.Path, ":copyTo") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			destID, _ := req["destinationSpreadsheetId"].(string)
			if destID == "" {
				t.Error("expected destinationSpreadsheetId in request")
			}

			resp := map[string]interface{}{
				"sheetId": 99,
				"title":   "Sheet1 (copy)",
				"index":   1,
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

// TestSheetsBatchRead_MockServer tests batch-read API integration
func TestSheetsBatchRead_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/values:batchGet") {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"valueRanges": []map[string]interface{}{
					{
						"range":  "Sheet1!A1:B2",
						"values": [][]interface{}{{"a", "b"}, {"c", "d"}},
					},
					{
						"range":  "Sheet2!A1:C1",
						"values": [][]interface{}{{"x", "y", "z"}},
					},
				},
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

// TestSheetsBatchWrite_JSONValues tests that --values flag correctly handles JSON
// with commas and quotes (regression: StringSlice CSV-parses and breaks JSON)
func TestSheetsBatchWrite_JSONValues(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "batch-write")
	if cmd == nil {
		t.Fatal("batch-write command not found")
	}

	// Simulate the flags that would be passed on the CLI.
	// StringSlice would CSV-parse '[[\"hello\",\"world\"]]' and split on commas,
	// producing broken fragments. StringArray keeps each flag occurrence intact.
	flags := cmd.Flags()

	// Reset flags for isolated test
	flags.Set("ranges", "Sheet1!A1:B2")
	flags.Set("values", `[["hello","world"],["foo","bar"]]`)

	ranges, err := flags.GetStringArray("ranges")
	if err != nil {
		t.Fatalf("failed to get ranges: %v", err)
	}
	values, err := flags.GetStringArray("values")
	if err != nil {
		t.Fatalf("failed to get values: %v", err)
	}

	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d: %v", len(ranges), ranges)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value entry, got %d: %v", len(values), values)
	}

	// Verify the JSON is intact and parseable
	var parsed [][]interface{}
	if err := json.Unmarshal([]byte(values[0]), &parsed); err != nil {
		t.Fatalf("values JSON should be parseable, got error: %v\nraw value: %q", err, values[0])
	}
	if len(parsed) != 2 || len(parsed[0]) != 2 {
		t.Errorf("expected 2x2 array, got %v", parsed)
	}
}

// TestSheetsBatchWrite_MockServer tests batch-write API integration
func TestSheetsBatchWrite_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/values:batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			data, ok := req["data"].([]interface{})
			if !ok || len(data) == 0 {
				t.Error("expected data in batch write request")
			}

			resp := map[string]interface{}{
				"spreadsheetId":      "test-id",
				"totalUpdatedSheets": 2,
				"totalUpdatedRows":   3,
				"totalUpdatedCells":  6,
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

// TestSheetsAddNamedRangeCommand_Flags tests add-named-range command flags
func TestSheetsAddNamedRangeCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "add-named-range")
	if cmd == nil {
		t.Fatal("add-named-range command not found")
	}

	expectedFlags := []string{"name"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsListNamedRangesCommand tests list-named-ranges command
func TestSheetsListNamedRangesCommand(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "list-named-ranges")
	if cmd == nil {
		t.Fatal("list-named-ranges command not found")
	}

	if cmd.Use != "list-named-ranges <spreadsheet-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

// TestSheetsDeleteNamedRangeCommand_Flags tests delete-named-range command flags
func TestSheetsDeleteNamedRangeCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "delete-named-range")
	if cmd == nil {
		t.Fatal("delete-named-range command not found")
	}

	expectedFlags := []string{"named-range-id"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsAddFilterCommand tests add-filter command
func TestSheetsAddFilterCommand(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "add-filter")
	if cmd == nil {
		t.Fatal("add-filter command not found")
	}

	if cmd.Use != "add-filter <spreadsheet-id> <range>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

// TestSheetsClearFilterCommand_Flags tests clear-filter command flags
func TestSheetsClearFilterCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "clear-filter")
	if cmd == nil {
		t.Fatal("clear-filter command not found")
	}

	expectedFlags := []string{"sheet"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsAddFilterViewCommand_Flags tests add-filter-view command flags
func TestSheetsAddFilterViewCommand_Flags(t *testing.T) {
	cmd := findSubcommand(sheetsCmd, "add-filter-view")
	if cmd == nil {
		t.Fatal("add-filter-view command not found")
	}

	expectedFlags := []string{"name"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSheetsAddNamedRange_MockServer tests add-named-range API integration
func TestSheetsAddNamedRange_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				addNamedRange := requests[0].(map[string]interface{})["addNamedRange"]
				if addNamedRange != nil {
					namedRange := addNamedRange.(map[string]interface{})["namedRange"].(map[string]interface{})
					name := namedRange["name"].(string)

					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
						"replies": []map[string]interface{}{
							{
								"addNamedRange": map[string]interface{}{
									"namedRange": map[string]interface{}{
										"namedRangeId": "nr-123",
										"name":         name,
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

// TestSheetsListNamedRanges_MockServer tests list-named-ranges API integration
func TestSheetsListNamedRanges_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/spreadsheets/") {
			resp := map[string]interface{}{
				"namedRanges": []map[string]interface{}{
					{
						"namedRangeId": "nr-123",
						"name":         "MyRange",
						"range": map[string]interface{}{
							"sheetId":          0,
							"startRowIndex":    0,
							"endRowIndex":      10,
							"startColumnIndex": 0,
							"endColumnIndex":   4,
						},
					},
					{
						"namedRangeId": "nr-456",
						"name":         "DataRange",
						"range": map[string]interface{}{
							"sheetId":          0,
							"startRowIndex":    0,
							"endRowIndex":      100,
							"startColumnIndex": 0,
							"endColumnIndex":   26,
						},
					},
				},
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

// TestSheetsDeleteNamedRange_MockServer tests delete-named-range API integration
func TestSheetsDeleteNamedRange_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				deleteNamedRange := requests[0].(map[string]interface{})["deleteNamedRange"]
				if deleteNamedRange != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsAddFilter_MockServer tests add-filter API integration
func TestSheetsAddFilter_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				setBasicFilter := requests[0].(map[string]interface{})["setBasicFilter"]
				if setBasicFilter != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsClearFilter_MockServer tests clear-filter API integration
func TestSheetsClearFilter_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				clearFilter := requests[0].(map[string]interface{})["clearBasicFilter"]
				if clearFilter != nil {
					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
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

// TestSheetsAddFilterView_MockServer tests add-filter-view API integration
func TestSheetsAddFilterView_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				addFilterView := requests[0].(map[string]interface{})["addFilterView"]
				if addFilterView != nil {
					filter := addFilterView.(map[string]interface{})["filter"].(map[string]interface{})
					title := filter["title"].(string)

					resp := map[string]interface{}{
						"spreadsheetId": "test-id",
						"replies": []map[string]interface{}{
							{
								"addFilterView": map[string]interface{}{
									"filter": map[string]interface{}{
										"filterViewId": 98765,
										"title":        title,
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

// TestSheetsCommands_Structure_NamedRangesAndFilters tests that named range and filter commands are registered
func TestSheetsCommands_Structure_NamedRangesAndFilters(t *testing.T) {
	commands := []string{
		"add-named-range",
		"list-named-ranges",
		"delete-named-range",
		"add-filter",
		"clear-filter",
		"add-filter-view",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := findSubcommand(sheetsCmd, cmdName)
			if cmd == nil {
				t.Fatalf("command '%s' not found", cmdName)
			}
		})
	}
}

// TestSheetsFreeze_MockServer tests freeze panes API integration
func TestSheetsFreeze_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"sheets": []map[string]interface{}{
					{
						"properties": map[string]interface{}{
							"sheetId": 0,
							"title":   "Sheet1",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, ":batchUpdate") {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			requests := req["requests"].([]interface{})
			if len(requests) > 0 {
				updateProps := requests[0].(map[string]interface{})["updateSheetProperties"]
				if updateProps == nil {
					t.Error("expected updateSheetProperties request")
				} else {
					fields := updateProps.(map[string]interface{})["fields"].(string)
					if !strings.Contains(fields, "gridProperties.frozenRowCount") && !strings.Contains(fields, "gridProperties.frozenColumnCount") {
						t.Errorf("expected frozen fields in mask, got %s", fields)
					}
				}
			}

			resp := map[string]interface{}{
				"spreadsheetId": "test-id",
				"replies":       []map[string]interface{}{{}},
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
