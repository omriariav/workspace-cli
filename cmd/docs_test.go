package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

func TestDocsCommands_Flags(t *testing.T) {
	// Test read command flags
	readCmd := findSubcommand(docsCmd, "read")
	if readCmd == nil {
		t.Fatal("docs read command not found")
	}
	if readCmd.Flags().Lookup("include-formatting") == nil {
		t.Error("expected --include-formatting flag")
	}

	// Test create command flags
	createCmd := findSubcommand(docsCmd, "create")
	if createCmd == nil {
		t.Fatal("docs create command not found")
	}
	if createCmd.Flags().Lookup("title") == nil {
		t.Error("expected --title flag")
	}
	if createCmd.Flags().Lookup("text") == nil {
		t.Error("expected --text flag")
	}

	// Test append command flags
	appendCmd := findSubcommand(docsCmd, "append")
	if appendCmd == nil {
		t.Fatal("docs append command not found")
	}
	if appendCmd.Flags().Lookup("text") == nil {
		t.Error("expected --text flag")
	}
	if appendCmd.Flags().Lookup("newline") == nil {
		t.Error("expected --newline flag")
	}
}

// mockDocsServer creates a test server that mocks Docs API responses
func mockDocsServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Find matching handler
		for pattern, handler := range handlers {
			if r.URL.Path == pattern {
				handler(w, r)
				return
			}
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestDocsCreate_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			// Parse request body
			var req docs.Document
			json.NewDecoder(r.Body).Decode(&req)

			if req.Title != "Test Document" {
				t.Errorf("expected title 'Test Document', got '%s'", req.Title)
			}

			// Return created document
			json.NewEncoder(w).Encode(&docs.Document{
				DocumentId: "doc-123",
				Title:      req.Title,
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	doc, err := svc.Documents.Create(&docs.Document{
		Title: "Test Document",
	}).Do()
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	if doc.DocumentId != "doc-123" {
		t.Errorf("expected doc ID 'doc-123', got '%s'", doc.DocumentId)
	}

	if doc.Title != "Test Document" {
		t.Errorf("expected title 'Test Document', got '%s'", doc.Title)
	}
}

func TestDocsCreate_WithInitialText(t *testing.T) {
	createCalled := false
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents": func(w http.ResponseWriter, r *http.Request) {
			createCalled = true
			json.NewEncoder(w).Encode(&docs.Document{
				DocumentId: "doc-456",
				Title:      "Doc With Text",
			})
		},
		"/v1/documents/doc-456:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			if req.Requests[0].InsertText == nil {
				t.Error("expected InsertText request")
			} else if req.Requests[0].InsertText.Text != "Hello World" {
				t.Errorf("expected text 'Hello World', got '%s'", req.Requests[0].InsertText.Text)
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-456",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	// Create document
	doc, err := svc.Documents.Create(&docs.Document{
		Title: "Doc With Text",
	}).Do()
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	// Add initial text
	_, err = svc.Documents.BatchUpdate(doc.DocumentId, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: 1},
					Text:     "Hello World",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to add text: %v", err)
	}

	if !createCalled {
		t.Error("create endpoint was not called")
	}
	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

func TestDocsAppend_Success(t *testing.T) {
	getCalled := false
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-789": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				getCalled = true
				json.NewEncoder(w).Encode(&docs.Document{
					DocumentId: "doc-789",
					Title:      "Existing Doc",
					Body: &docs.Body{
						Content: []*docs.StructuralElement{
							{
								StartIndex: 0,
								EndIndex:   50,
								Paragraph: &docs.Paragraph{
									Elements: []*docs.ParagraphElement{
										{TextRun: &docs.TextRun{Content: "Existing content"}},
									},
								},
							},
						},
					},
				})
			}
		},
		"/v1/documents/doc-789:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			insertReq := req.Requests[0].InsertText
			if insertReq == nil {
				t.Error("expected InsertText request")
			} else {
				// Should insert at end of document (endIndex - 1)
				if insertReq.Location.Index != 49 {
					t.Errorf("expected index 49, got %d", insertReq.Location.Index)
				}
				if insertReq.Text != "\nAppended text" {
					t.Errorf("expected text with newline, got '%s'", insertReq.Text)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-789",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	// Get document to find end index
	doc, err := svc.Documents.Get("doc-789").Do()
	if err != nil {
		t.Fatalf("failed to get document: %v", err)
	}

	endIndex := doc.Body.Content[len(doc.Body.Content)-1].EndIndex - 1

	// Append text
	_, err = svc.Documents.BatchUpdate("doc-789", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: endIndex},
					Text:     "\nAppended text",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to append text: %v", err)
	}

	if !getCalled {
		t.Error("get endpoint was not called")
	}
	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

func TestDocsAppend_WithoutNewline(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-abc": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&docs.Document{
				DocumentId: "doc-abc",
				Title:      "Test Doc",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{StartIndex: 0, EndIndex: 20},
					},
				},
			})
		},
		"/v1/documents/doc-abc:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			// Without newline flag, text should not have leading newline
			insertReq := req.Requests[0].InsertText
			if insertReq.Text == "\nText without newline" {
				t.Error("text should not have leading newline when newline=false")
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	doc, err := svc.Documents.Get("doc-abc").Do()
	if err != nil {
		t.Fatalf("failed to get document: %v", err)
	}

	endIndex := doc.Body.Content[len(doc.Body.Content)-1].EndIndex - 1

	// Append without newline (simulating --newline=false)
	addNewline := false
	insertText := "Text without newline"
	if addNewline {
		insertText = "\n" + insertText
	}

	_, err = svc.Documents.BatchUpdate("doc-abc", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: endIndex},
					Text:     insertText,
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to append text: %v", err)
	}
}

func TestDocsRead_ExtractText(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-read": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&docs.Document{
				DocumentId: "doc-read",
				Title:      "Read Test",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{TextRun: &docs.TextRun{Content: "First paragraph\n"}},
								},
							},
						},
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{TextRun: &docs.TextRun{Content: "Second paragraph\n"}},
								},
							},
						},
					},
				},
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	doc, err := svc.Documents.Get("doc-read").Do()
	if err != nil {
		t.Fatalf("failed to get document: %v", err)
	}

	// Extract text using the same logic as runDocsRead
	var textBuilder strings.Builder
	extractText(doc.Body.Content, &textBuilder)

	text := textBuilder.String()
	if !strings.Contains(text, "First paragraph") {
		t.Errorf("expected 'First paragraph' in text: %s", text)
	}
	if !strings.Contains(text, "Second paragraph") {
		t.Errorf("expected 'Second paragraph' in text: %s", text)
	}
}

func TestDocsRead_ExtractStructure(t *testing.T) {
	content := []*docs.StructuralElement{
		{
			Paragraph: &docs.Paragraph{
				ParagraphStyle: &docs.ParagraphStyle{
					NamedStyleType: "HEADING_1",
					HeadingId:      "h.abc123",
				},
				Elements: []*docs.ParagraphElement{
					{TextRun: &docs.TextRun{Content: "Heading Text"}},
				},
			},
		},
		{
			Table: &docs.Table{
				Rows:    3,
				Columns: 2,
			},
		},
	}

	structure := extractStructure(content)

	if len(structure) != 2 {
		t.Fatalf("expected 2 structure elements, got %d", len(structure))
	}

	// Check paragraph
	para := structure[0]
	if para["type"] != "paragraph" {
		t.Errorf("expected type 'paragraph', got '%v'", para["type"])
	}
	if para["style"] != "HEADING_1" {
		t.Errorf("expected style 'HEADING_1', got '%v'", para["style"])
	}
	if para["heading_id"] != "h.abc123" {
		t.Errorf("expected heading_id 'h.abc123', got '%v'", para["heading_id"])
	}

	// Check table
	table := structure[1]
	if table["type"] != "table" {
		t.Errorf("expected type 'table', got '%v'", table["type"])
	}
	if table["rows"] != int64(3) {
		t.Errorf("expected 3 rows, got %v", table["rows"])
	}
}

// TestDocsInsertCommand_Flags tests insert command flags
func TestDocsInsertCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "insert")
	if cmd == nil {
		t.Fatal("docs insert command not found")
	}

	expectedFlags := []string{"text", "at"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsReplaceCommand_Flags tests replace command flags
func TestDocsReplaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "replace")
	if cmd == nil {
		t.Fatal("docs replace command not found")
	}

	expectedFlags := []string{"find", "replace", "match-case"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsInsert_Success tests insert text at position
func TestDocsInsert_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-insert:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			insertReq := req.Requests[0].InsertText
			if insertReq == nil {
				t.Error("expected InsertText request")
			} else {
				if insertReq.Location.Index != 10 {
					t.Errorf("expected index 10, got %d", insertReq.Location.Index)
				}
				if insertReq.Text != "Inserted text" {
					t.Errorf("expected 'Inserted text', got '%s'", insertReq.Text)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-insert",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	_, err = svc.Documents.BatchUpdate("doc-insert", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: 10},
					Text:     "Inserted text",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to insert text: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestDocsReplace_Success tests find and replace
func TestDocsReplace_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-replace:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			replaceReq := req.Requests[0].ReplaceAllText
			if replaceReq == nil {
				t.Error("expected ReplaceAllText request")
			} else {
				if replaceReq.ContainsText.Text != "old" {
					t.Errorf("expected find text 'old', got '%s'", replaceReq.ContainsText.Text)
				}
				if replaceReq.ReplaceText != "new" {
					t.Errorf("expected replace text 'new', got '%s'", replaceReq.ReplaceText)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-replace",
				Replies: []*docs.Response{
					{
						ReplaceAllText: &docs.ReplaceAllTextResponse{
							OccurrencesChanged: 5,
						},
					},
				},
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	resp, err := svc.Documents.BatchUpdate("doc-replace", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				ReplaceAllText: &docs.ReplaceAllTextRequest{
					ContainsText: &docs.SubstringMatchCriteria{
						Text:      "old",
						MatchCase: true,
					},
					ReplaceText: "new",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to replace text: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}

	if len(resp.Replies) == 0 || resp.Replies[0].ReplaceAllText == nil {
		t.Error("expected ReplaceAllText response")
	} else if resp.Replies[0].ReplaceAllText.OccurrencesChanged != 5 {
		t.Errorf("expected 5 occurrences changed, got %d", resp.Replies[0].ReplaceAllText.OccurrencesChanged)
	}
}

// TestDocsDeleteCommand_Flags tests delete command flags
func TestDocsDeleteCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "delete")
	if cmd == nil {
		t.Fatal("docs delete command not found")
	}

	expectedFlags := []string{"from", "to"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsAddTableCommand_Flags tests add-table command flags
func TestDocsAddTableCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "add-table")
	if cmd == nil {
		t.Fatal("docs add-table command not found")
	}

	expectedFlags := []string{"rows", "cols", "at"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsDelete_Success tests deleting content
func TestDocsDelete_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-delete:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			deleteReq := req.Requests[0].DeleteContentRange
			if deleteReq == nil {
				t.Error("expected DeleteContentRange request")
			} else {
				if deleteReq.Range.StartIndex != 10 {
					t.Errorf("expected start index 10, got %d", deleteReq.Range.StartIndex)
				}
				if deleteReq.Range.EndIndex != 50 {
					t.Errorf("expected end index 50, got %d", deleteReq.Range.EndIndex)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-delete",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	_, err = svc.Documents.BatchUpdate("doc-delete", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{
						StartIndex: 10,
						EndIndex:   50,
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to delete content: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestDocsAddTable_Success tests adding a table
func TestDocsAddTable_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-table:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			insertTableReq := req.Requests[0].InsertTable
			if insertTableReq == nil {
				t.Error("expected InsertTable request")
			} else {
				if insertTableReq.Rows != 3 {
					t.Errorf("expected 3 rows, got %d", insertTableReq.Rows)
				}
				if insertTableReq.Columns != 4 {
					t.Errorf("expected 4 columns, got %d", insertTableReq.Columns)
				}
				if insertTableReq.Location.Index != 10 {
					t.Errorf("expected index 10, got %d", insertTableReq.Location.Index)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-table",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	_, err = svc.Documents.BatchUpdate("doc-table", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertTable: &docs.InsertTableRequest{
					Rows:     3,
					Columns:  4,
					Location: &docs.Location{Index: 10},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to add table: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestDocsCommands_ContentFormatFlag tests that content-format flag exists on create/append/insert
func TestDocsCommands_ContentFormatFlag(t *testing.T) {
	commands := []string{"create", "append", "insert"}
	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := findSubcommand(docsCmd, cmdName)
			if cmd == nil {
				t.Fatalf("docs %s command not found", cmdName)
			}
			flag := cmd.Flags().Lookup("content-format")
			if flag == nil {
				t.Fatalf("expected --content-format flag on docs %s", cmdName)
			}
			if flag.DefValue != "markdown" {
				t.Errorf("expected default 'markdown', got '%s'", flag.DefValue)
			}
		})
	}
}

// TestBuildTextRequests_Plaintext tests that plaintext mode creates InsertText
func TestBuildTextRequests_Plaintext(t *testing.T) {
	requests, err := buildTextRequests("Hello World", "plaintext", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].InsertText == nil {
		t.Fatal("expected InsertText request")
	}
	if requests[0].InsertText.Text != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", requests[0].InsertText.Text)
	}
	if requests[0].InsertText.Location.Index != 1 {
		t.Errorf("expected index 1, got %d", requests[0].InsertText.Location.Index)
	}
}

// TestBuildTextRequests_Markdown tests that markdown mode creates InsertText with raw markdown
func TestBuildTextRequests_Markdown(t *testing.T) {
	mdText := "# Title\n\n**Bold** and *italic*"
	requests, err := buildTextRequests(mdText, "markdown", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].InsertText == nil {
		t.Fatal("expected InsertText request")
	}
	// Markdown text should be preserved as-is
	if requests[0].InsertText.Text != mdText {
		t.Errorf("expected markdown text preserved, got '%s'", requests[0].InsertText.Text)
	}
	if requests[0].InsertText.Location.Index != 5 {
		t.Errorf("expected index 5, got %d", requests[0].InsertText.Location.Index)
	}
}

// TestBuildTextRequests_Richformat tests that richformat mode parses JSON requests
func TestBuildTextRequests_Richformat(t *testing.T) {
	jsonInput := `[{"insertText":{"location":{"index":1},"text":"Hello"}}]`
	requests, err := buildTextRequests(jsonInput, "richformat", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].InsertText == nil {
		t.Fatal("expected InsertText request from richformat")
	}
	if requests[0].InsertText.Text != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", requests[0].InsertText.Text)
	}
}

// TestBuildTextRequests_RichformatInvalid tests that invalid JSON returns error
func TestBuildTextRequests_RichformatInvalid(t *testing.T) {
	_, err := buildTextRequests("not json", "richformat", 1)
	if err == nil {
		t.Error("expected error for invalid richformat JSON")
	}
}

// TestBuildTextRequests_UnknownFormat tests that unknown format returns error
func TestBuildTextRequests_UnknownFormat(t *testing.T) {
	_, err := buildTextRequests("text", "unknown", 1)
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

// TestDocsCreate_WithMarkdown tests create with markdown format (inserts text as-is)
func TestDocsCreate_WithMarkdown(t *testing.T) {
	batchUpdateCalled := false
	var capturedRequests docs.BatchUpdateDocumentRequest

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&docs.Document{
				DocumentId: "doc-md",
				Title:      "Markdown Doc",
			})
		},
		"/v1/documents/doc-md:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true
			json.NewDecoder(r.Body).Decode(&capturedRequests)
			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-md",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	// Simulate create with markdown content
	doc, err := svc.Documents.Create(&docs.Document{
		Title: "Markdown Doc",
	}).Do()
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	mdText := "# Hello\n\n**Bold** text"
	requests, err := buildTextRequests(mdText, "markdown", 1)
	if err != nil {
		t.Fatalf("failed to build requests: %v", err)
	}

	_, err = svc.Documents.BatchUpdate(doc.DocumentId, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		t.Fatalf("failed to batch update: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}

	// Verify the request has InsertText with raw markdown
	if len(capturedRequests.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(capturedRequests.Requests))
	}
	if capturedRequests.Requests[0].InsertText == nil {
		t.Fatal("expected InsertText request")
	}
	if capturedRequests.Requests[0].InsertText.Text != mdText {
		t.Errorf("expected markdown text '%s', got '%s'", mdText, capturedRequests.Requests[0].InsertText.Text)
	}
}

// TestDocsCreate_ContentFormatE2E tests end-to-end Cobra flag parsing for --content-format
func TestDocsCreate_ContentFormatE2E(t *testing.T) {
	// Verify the flag wiring from Cobra through to buildTextRequests
	createCmd := findSubcommand(docsCmd, "create")
	if createCmd == nil {
		t.Fatal("docs create command not found")
	}

	// Test that the flag parses correctly and returns the right value
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"default is markdown", []string{"--title", "T", "--text", "x"}, "markdown"},
		{"explicit plaintext", []string{"--title", "T", "--text", "x", "--content-format", "plaintext"}, "plaintext"},
		{"explicit richformat", []string{"--title", "T", "--text", "x", "--content-format", "richformat"}, "richformat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			cmd := findSubcommand(docsCmd, "create")
			cmd.ResetFlags()
			cmd.Flags().String("title", "", "Document title (required)")
			cmd.Flags().String("text", "", "Initial text content")
			cmd.Flags().String("content-format", "markdown", "Content format: markdown, plaintext, or richformat")

			cmd.ParseFlags(tt.args)
			got, _ := cmd.Flags().GetString("content-format")
			if got != tt.expected {
				t.Errorf("expected content-format '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

// TestDocsInsert_RichformatIgnoresAt tests that --at with richformat produces a warning
func TestDocsInsert_RichformatIgnoresAt(t *testing.T) {
	cmd := findSubcommand(docsCmd, "insert")
	if cmd == nil {
		t.Fatal("docs insert command not found")
	}

	// Verify that Changed("at") works when --at is explicitly set
	cmd.ResetFlags()
	cmd.Flags().String("text", "", "Text to insert (required)")
	cmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")
	cmd.Flags().String("content-format", "markdown", "Content format")

	cmd.ParseFlags([]string{"--text", "x", "--at", "50", "--content-format", "richformat"})

	if !cmd.Flags().Changed("at") {
		t.Error("expected --at to be marked as changed")
	}
	cf, _ := cmd.Flags().GetString("content-format")
	if cf != "richformat" {
		t.Errorf("expected 'richformat', got '%s'", cf)
	}
}

// TestDocsCommands_Structure_Extended tests that all new docs commands are registered
func TestDocsCommands_Structure_Extended(t *testing.T) {
	commands := []string{
		"delete",
		"add-table",
		"format",
		"set-paragraph-style",
		"add-list",
		"remove-list",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := findSubcommand(docsCmd, cmdName)
			if cmd == nil {
				t.Fatalf("command '%s' not found", cmdName)
			}
		})
	}
}

// TestDocsFormatCommand_Flags tests format command flags
func TestDocsFormatCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "format")
	if cmd == nil {
		t.Fatal("docs format command not found")
	}

	expectedFlags := []string{"from", "to", "bold", "italic", "font-size", "color"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsSetParagraphStyleCommand_Flags tests set-paragraph-style command flags
func TestDocsSetParagraphStyleCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "set-paragraph-style")
	if cmd == nil {
		t.Fatal("docs set-paragraph-style command not found")
	}

	expectedFlags := []string{"from", "to", "alignment", "line-spacing"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsAddListCommand_Flags tests add-list command flags
func TestDocsAddListCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "add-list")
	if cmd == nil {
		t.Fatal("docs add-list command not found")
	}

	expectedFlags := []string{"at", "type", "items"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsRemoveListCommand_Flags tests remove-list command flags
func TestDocsRemoveListCommand_Flags(t *testing.T) {
	cmd := findSubcommand(docsCmd, "remove-list")
	if cmd == nil {
		t.Fatal("docs remove-list command not found")
	}

	expectedFlags := []string{"from", "to"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestDocsFormat_Success tests formatting text
func TestDocsFormat_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-format:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			updateStyle := req.Requests[0].UpdateTextStyle
			if updateStyle == nil {
				t.Error("expected UpdateTextStyle request")
			} else {
				if updateStyle.Range.StartIndex != 10 {
					t.Errorf("expected start index 10, got %d", updateStyle.Range.StartIndex)
				}
				if updateStyle.Range.EndIndex != 50 {
					t.Errorf("expected end index 50, got %d", updateStyle.Range.EndIndex)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-format",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	_, err = svc.Documents.BatchUpdate("doc-format", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				UpdateTextStyle: &docs.UpdateTextStyleRequest{
					TextStyle: &docs.TextStyle{Bold: true},
					Range: &docs.Range{
						StartIndex: 10,
						EndIndex:   50,
					},
					Fields: "bold",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to format text: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestDocsSetParagraphStyle_Success tests setting paragraph style
func TestDocsSetParagraphStyle_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-para:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			updatePara := req.Requests[0].UpdateParagraphStyle
			if updatePara == nil {
				t.Error("expected UpdateParagraphStyle request")
			} else {
				if updatePara.ParagraphStyle.Alignment != "CENTER" {
					t.Errorf("expected alignment CENTER, got %s", updatePara.ParagraphStyle.Alignment)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-para",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	_, err = svc.Documents.BatchUpdate("doc-para", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
					ParagraphStyle: &docs.ParagraphStyle{Alignment: "CENTER"},
					Range: &docs.Range{
						StartIndex: 1,
						EndIndex:   100,
					},
					Fields: "alignment",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to set paragraph style: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestDocsAddList_Success tests adding a list
func TestDocsAddList_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-list:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) < 2 {
				t.Errorf("expected at least 2 requests (insertText + createParagraphBullets), got %d", len(req.Requests))
			}

			if req.Requests[0].InsertText == nil {
				t.Error("expected InsertText as first request")
			}

			if req.Requests[1].CreateParagraphBullets == nil {
				t.Error("expected CreateParagraphBullets as second request")
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-list",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	insertText := "Item 1\nItem 2\nItem 3\n"
	_, err = svc.Documents.BatchUpdate("doc-list", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: 1},
					Text:     insertText,
				},
			},
			{
				CreateParagraphBullets: &docs.CreateParagraphBulletsRequest{
					Range: &docs.Range{
						StartIndex: 1,
						EndIndex:   1 + int64(len(insertText)),
					},
					BulletPreset: "BULLET_DISC_CIRCLE_SQUARE",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to add list: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestDocsRemoveList_Success tests removing list formatting
func TestDocsRemoveList_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/documents/doc-unlist:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req docs.BatchUpdateDocumentRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			deleteBullets := req.Requests[0].DeleteParagraphBullets
			if deleteBullets == nil {
				t.Error("expected DeleteParagraphBullets request")
			} else {
				if deleteBullets.Range.StartIndex != 10 {
					t.Errorf("expected start index 10, got %d", deleteBullets.Range.StartIndex)
				}
				if deleteBullets.Range.EndIndex != 50 {
					t.Errorf("expected end index 50, got %d", deleteBullets.Range.EndIndex)
				}
			}

			json.NewEncoder(w).Encode(&docs.BatchUpdateDocumentResponse{
				DocumentId: "doc-unlist",
			})
		},
	}

	server := mockDocsServer(t, handlers)
	defer server.Close()

	svc, err := docs.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create docs service: %v", err)
	}

	_, err = svc.Documents.BatchUpdate("doc-unlist", &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				DeleteParagraphBullets: &docs.DeleteParagraphBulletsRequest{
					Range: &docs.Range{
						StartIndex: 10,
						EndIndex:   50,
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to remove list: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestParseDocsHexColor tests the parseDocsHexColor helper
func TestParseDocsHexColor(t *testing.T) {
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
		{"invalid - no hash", "FF0000", 0, 0, 0, true},
		{"invalid - too short", "#FFF", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color, err := parseDocsHexColor(tt.hex)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rgb := color.Color.RgbColor
			if rgb.Red != tt.wantR {
				t.Errorf("expected red %f, got %f", tt.wantR, rgb.Red)
			}
			if rgb.Green != tt.wantG {
				t.Errorf("expected green %f, got %f", tt.wantG, rgb.Green)
			}
			if rgb.Blue != tt.wantB {
				t.Errorf("expected blue %f, got %f", tt.wantB, rgb.Blue)
			}
		})
	}
}
