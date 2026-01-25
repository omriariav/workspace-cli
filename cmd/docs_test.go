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
