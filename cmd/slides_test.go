package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

func TestSlidesCommands_Flags(t *testing.T) {
	// Test create command flags
	createCmd := findSubcommand(slidesCmd, "create")
	if createCmd == nil {
		t.Fatal("slides create command not found")
	}
	if createCmd.Flags().Lookup("title") == nil {
		t.Error("expected --title flag")
	}

	// Test add-slide command flags
	addSlideCmd := findSubcommand(slidesCmd, "add-slide")
	if addSlideCmd == nil {
		t.Fatal("slides add-slide command not found")
	}
	if addSlideCmd.Flags().Lookup("title") == nil {
		t.Error("expected --title flag")
	}
	if addSlideCmd.Flags().Lookup("body") == nil {
		t.Error("expected --body flag")
	}
	if addSlideCmd.Flags().Lookup("layout") == nil {
		t.Error("expected --layout flag")
	}
}

// mockSlidesServer creates a test server that mocks Slides API responses
func mockSlidesServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
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

func TestSlidesCreate_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var req slides.Presentation
			json.NewDecoder(r.Body).Decode(&req)

			if req.Title != "Test Presentation" {
				t.Errorf("expected title 'Test Presentation', got '%s'", req.Title)
			}

			// Return created presentation with default slide
			json.NewEncoder(w).Encode(&slides.Presentation{
				PresentationId: "pres-123",
				Title:          req.Title,
				Slides: []*slides.Page{
					{ObjectId: "slide-default"},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	pres, err := svc.Presentations.Create(&slides.Presentation{
		Title: "Test Presentation",
	}).Do()
	if err != nil {
		t.Fatalf("failed to create presentation: %v", err)
	}

	if pres.PresentationId != "pres-123" {
		t.Errorf("expected presentation ID 'pres-123', got '%s'", pres.PresentationId)
	}

	if pres.Title != "Test Presentation" {
		t.Errorf("expected title 'Test Presentation', got '%s'", pres.Title)
	}

	if len(pres.Slides) != 1 {
		t.Errorf("expected 1 default slide, got %d", len(pres.Slides))
	}
}

func TestSlidesAddSlide_Success(t *testing.T) {
	batchUpdateCalled := false
	getCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-456:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			createSlide := req.Requests[0].CreateSlide
			if createSlide == nil {
				t.Error("expected CreateSlide request")
			} else {
				if createSlide.SlideLayoutReference.PredefinedLayout != "TITLE_AND_BODY" {
					t.Errorf("expected layout 'TITLE_AND_BODY', got '%s'", createSlide.SlideLayoutReference.PredefinedLayout)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-456",
				Replies: []*slides.Response{
					{
						CreateSlide: &slides.CreateSlideResponse{
							ObjectId: "new-slide-id",
						},
					},
				},
			})
		},
		"/v1/presentations/pres-456": func(w http.ResponseWriter, r *http.Request) {
			getCalled = true
			json.NewEncoder(w).Encode(&slides.Presentation{
				PresentationId: "pres-456",
				Title:          "Test Pres",
				Slides: []*slides.Page{
					{ObjectId: "slide-1"},
					{
						ObjectId: "new-slide-id",
						PageElements: []*slides.PageElement{
							{
								ObjectId: "title-element",
								Shape: &slides.Shape{
									Placeholder: &slides.Placeholder{
										Type: "TITLE",
									},
								},
							},
							{
								ObjectId: "body-element",
								Shape: &slides.Shape{
									Placeholder: &slides.Placeholder{
										Type: "BODY",
									},
								},
							},
						},
					},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	// Create slide
	resp, err := svc.Presentations.BatchUpdate("pres-456", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateSlide: &slides.CreateSlideRequest{
					ObjectId: "new-slide-id",
					SlideLayoutReference: &slides.LayoutReference{
						PredefinedLayout: "TITLE_AND_BODY",
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to add slide: %v", err)
	}

	if len(resp.Replies) == 0 || resp.Replies[0].CreateSlide == nil {
		t.Fatal("expected CreateSlide reply")
	}

	if resp.Replies[0].CreateSlide.ObjectId != "new-slide-id" {
		t.Errorf("expected slide ID 'new-slide-id', got '%s'", resp.Replies[0].CreateSlide.ObjectId)
	}

	// Get presentation to find placeholders
	_, err = svc.Presentations.Get("pres-456").Do()
	if err != nil {
		t.Fatalf("failed to get presentation: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
	if !getCalled {
		t.Error("get endpoint was not called")
	}
}

func TestSlidesAddSlide_WithText(t *testing.T) {
	textInsertCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-789:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			// Check if this is the text insert request
			for _, r := range req.Requests {
				if r.InsertText != nil {
					textInsertCalled = true
					if r.InsertText.Text != "Slide Title" && r.InsertText.Text != "Slide body content" {
						t.Errorf("unexpected text: %s", r.InsertText.Text)
					}
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-789",
				Replies:        []*slides.Response{{}},
			})
		},
		"/v1/presentations/pres-789": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&slides.Presentation{
				PresentationId: "pres-789",
				Slides: []*slides.Page{
					{
						ObjectId: "slide-with-text",
						PageElements: []*slides.PageElement{
							{
								ObjectId: "title-placeholder",
								Shape: &slides.Shape{
									Placeholder: &slides.Placeholder{Type: "TITLE"},
								},
							},
							{
								ObjectId: "body-placeholder",
								Shape: &slides.Shape{
									Placeholder: &slides.Placeholder{Type: "BODY"},
								},
							},
						},
					},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	// Get presentation to find placeholders
	pres, err := svc.Presentations.Get("pres-789").Do()
	if err != nil {
		t.Fatalf("failed to get presentation: %v", err)
	}

	// Find placeholders and insert text
	textRequests := []*slides.Request{}
	for _, element := range pres.Slides[0].PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil {
			if element.Shape.Placeholder.Type == "TITLE" {
				textRequests = append(textRequests, &slides.Request{
					InsertText: &slides.InsertTextRequest{
						ObjectId: element.ObjectId,
						Text:     "Slide Title",
					},
				})
			}
			if element.Shape.Placeholder.Type == "BODY" {
				textRequests = append(textRequests, &slides.Request{
					InsertText: &slides.InsertTextRequest{
						ObjectId: element.ObjectId,
						Text:     "Slide body content",
					},
				})
			}
		}
	}

	if len(textRequests) > 0 {
		_, err = svc.Presentations.BatchUpdate("pres-789", &slides.BatchUpdatePresentationRequest{
			Requests: textRequests,
		}).Do()
		if err != nil {
			t.Fatalf("failed to insert text: %v", err)
		}
	}

	if !textInsertCalled {
		t.Error("text insert was not called")
	}
}

func TestSlidesRead_ExtractText(t *testing.T) {
	slide := &slides.Page{
		ObjectId: "test-slide",
		PageElements: []*slides.PageElement{
			{
				Shape: &slides.Shape{
					Text: &slides.TextContent{
						TextElements: []*slides.TextElement{
							{TextRun: &slides.TextRun{Content: "Title text"}},
						},
					},
				},
			},
			{
				Shape: &slides.Shape{
					Text: &slides.TextContent{
						TextElements: []*slides.TextElement{
							{TextRun: &slides.TextRun{Content: "Body text"}},
						},
					},
				},
			},
		},
	}

	text := extractSlideText(slide)

	if !strings.Contains(text, "Title text") {
		t.Errorf("expected 'Title text' in output: %s", text)
	}
	if !strings.Contains(text, "Body text") {
		t.Errorf("expected 'Body text' in output: %s", text)
	}
}

func TestSlidesRead_ExtractTitle(t *testing.T) {
	slide := &slides.Page{
		PageElements: []*slides.PageElement{
			{
				Shape: &slides.Shape{
					Placeholder: &slides.Placeholder{Type: "TITLE"},
					Text: &slides.TextContent{
						TextElements: []*slides.TextElement{
							{TextRun: &slides.TextRun{Content: "My Slide Title"}},
						},
					},
				},
			},
			{
				Shape: &slides.Shape{
					Placeholder: &slides.Placeholder{Type: "BODY"},
					Text: &slides.TextContent{
						TextElements: []*slides.TextElement{
							{TextRun: &slides.TextRun{Content: "Body content"}},
						},
					},
				},
			},
		},
	}

	title := extractSlideTitle(slide)

	if title != "My Slide Title" {
		t.Errorf("expected title 'My Slide Title', got '%s'", title)
	}
}

func TestSlidesRead_ExtractTableText(t *testing.T) {
	table := &slides.Table{
		TableRows: []*slides.TableRow{
			{
				TableCells: []*slides.TableCell{
					{
						Text: &slides.TextContent{
							TextElements: []*slides.TextElement{
								{TextRun: &slides.TextRun{Content: "Cell 1"}},
							},
						},
					},
					{
						Text: &slides.TextContent{
							TextElements: []*slides.TextElement{
								{TextRun: &slides.TextRun{Content: "Cell 2"}},
							},
						},
					},
				},
			},
		},
	}

	text := extractTableText(table)

	if !strings.Contains(text, "Cell 1") {
		t.Errorf("expected 'Cell 1' in output: %s", text)
	}
	if !strings.Contains(text, "Cell 2") {
		t.Errorf("expected 'Cell 2' in output: %s", text)
	}
}

func TestSlidesLayouts(t *testing.T) {
	// Test that various layout types work
	layouts := []string{
		"TITLE_AND_BODY",
		"TITLE_ONLY",
		"BLANK",
		"SECTION_HEADER",
		"TITLE_AND_TWO_COLUMNS",
	}

	for _, layout := range layouts {
		t.Run(layout, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/v1/presentations/layout-test:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
					var req slides.BatchUpdatePresentationRequest
					json.NewDecoder(r.Body).Decode(&req)

					if req.Requests[0].CreateSlide.SlideLayoutReference.PredefinedLayout != layout {
						t.Errorf("expected layout '%s', got '%s'", layout, req.Requests[0].CreateSlide.SlideLayoutReference.PredefinedLayout)
					}

					json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
						Replies: []*slides.Response{{CreateSlide: &slides.CreateSlideResponse{ObjectId: "new-slide"}}},
					})
				},
			}

			server := mockSlidesServer(t, handlers)
			defer server.Close()

			svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
			if err != nil {
				t.Fatalf("failed to create slides service: %v", err)
			}

			_, err = svc.Presentations.BatchUpdate("layout-test", &slides.BatchUpdatePresentationRequest{
				Requests: []*slides.Request{
					{
						CreateSlide: &slides.CreateSlideRequest{
							SlideLayoutReference: &slides.LayoutReference{
								PredefinedLayout: layout,
							},
						},
					},
				},
			}).Do()
			if err != nil {
				t.Fatalf("failed to create slide with layout %s: %v", layout, err)
			}
		})
	}
}

// TestSlidesDeleteSlideCommand_Flags tests delete-slide command flags
func TestSlidesDeleteSlideCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "delete-slide")
	if cmd == nil {
		t.Fatal("slides delete-slide command not found")
	}

	expectedFlags := []string{"slide-id", "slide-number"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesDuplicateSlideCommand_Flags tests duplicate-slide command flags
func TestSlidesDuplicateSlideCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "duplicate-slide")
	if cmd == nil {
		t.Fatal("slides duplicate-slide command not found")
	}

	expectedFlags := []string{"slide-id", "slide-number"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesDeleteSlide_Success tests deleting a slide
func TestSlidesDeleteSlide_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-delete:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			deleteReq := req.Requests[0].DeleteObject
			if deleteReq == nil {
				t.Error("expected DeleteObject request")
			} else if deleteReq.ObjectId != "slide-to-delete" {
				t.Errorf("expected object ID 'slide-to-delete', got '%s'", deleteReq.ObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-delete",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-delete", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: "slide-to-delete",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to delete slide: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesDuplicateSlide_Success tests duplicating a slide
func TestSlidesDuplicateSlide_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-dup:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			dupReq := req.Requests[0].DuplicateObject
			if dupReq == nil {
				t.Error("expected DuplicateObject request")
			} else if dupReq.ObjectId != "slide-to-dup" {
				t.Errorf("expected object ID 'slide-to-dup', got '%s'", dupReq.ObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-dup",
				Replies: []*slides.Response{
					{
						DuplicateObject: &slides.DuplicateObjectResponse{
							ObjectId: "new-duplicated-slide",
						},
					},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	resp, err := svc.Presentations.BatchUpdate("pres-dup", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DuplicateObject: &slides.DuplicateObjectRequest{
					ObjectId: "slide-to-dup",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to duplicate slide: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}

	if len(resp.Replies) == 0 || resp.Replies[0].DuplicateObject == nil {
		t.Error("expected DuplicateObject response")
	} else if resp.Replies[0].DuplicateObject.ObjectId != "new-duplicated-slide" {
		t.Errorf("expected new slide ID 'new-duplicated-slide', got '%s'", resp.Replies[0].DuplicateObject.ObjectId)
	}
}

// TestSlidesAddShapeCommand_Flags tests add-shape command flags
func TestSlidesAddShapeCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "add-shape")
	if cmd == nil {
		t.Fatal("slides add-shape command not found")
	}

	expectedFlags := []string{"slide-id", "slide-number", "type", "x", "y", "width", "height"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesAddImageCommand_Flags tests add-image command flags
func TestSlidesAddImageCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "add-image")
	if cmd == nil {
		t.Fatal("slides add-image command not found")
	}

	expectedFlags := []string{"slide-id", "slide-number", "url", "x", "y", "width"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesAddTextCommand_Flags tests add-text command flags
func TestSlidesAddTextCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "add-text")
	if cmd == nil {
		t.Fatal("slides add-text command not found")
	}

	expectedFlags := []string{"object-id", "table-id", "row", "col", "text", "at"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesAddTextCommand_FlagDefaults tests add-text command flag defaults
func TestSlidesAddTextCommand_FlagDefaults(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "add-text")
	if cmd == nil {
		t.Fatal("slides add-text command not found")
	}

	// Row and col should default to -1 (sentinel for "not provided")
	rowFlag := cmd.Flags().Lookup("row")
	if rowFlag.DefValue != "-1" {
		t.Errorf("expected --row default '-1', got '%s'", rowFlag.DefValue)
	}

	colFlag := cmd.Flags().Lookup("col")
	if colFlag.DefValue != "-1" {
		t.Errorf("expected --col default '-1', got '%s'", colFlag.DefValue)
	}

	// at should default to 0
	atFlag := cmd.Flags().Lookup("at")
	if atFlag.DefValue != "0" {
		t.Errorf("expected --at default '0', got '%s'", atFlag.DefValue)
	}
}

// TestSlidesAddTextCommand_TextRequired tests that --text is required
func TestSlidesAddTextCommand_TextRequired(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "add-text")
	if cmd == nil {
		t.Fatal("slides add-text command not found")
	}

	textFlag := cmd.Flags().Lookup("text")
	if textFlag == nil {
		t.Fatal("expected --text flag to exist")
	}
}

// TestSlidesAddTextCommand_ValidationErrors tests add-text flag validation via CLI
func TestSlidesAddTextCommand_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "mutual exclusivity - both object-id and table-id",
			flags: map[string]string{
				"object-id": "shape-123",
				"table-id":  "table-456",
				"text":      "test",
			},
			expectedError: "--object-id, --table-id, and --notes are mutually exclusive",
		},
		{
			name: "missing both object-id and table-id",
			flags: map[string]string{
				"text": "test",
			},
			expectedError: "must specify --object-id, --table-id, or --notes",
		},
		{
			name: "missing row with table-id",
			flags: map[string]string{
				"table-id": "table-456",
				"col":      "0",
				"text":     "test",
			},
			expectedError: "--row is required when using --table-id (valid values: 0 or greater)",
		},
		{
			name: "missing col with table-id",
			flags: map[string]string{
				"table-id": "table-456",
				"row":      "0",
				"text":     "test",
			},
			expectedError: "--col is required when using --table-id (valid values: 0 or greater)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(slidesCmd, "add-text")
			if cmd == nil {
				t.Fatal("slides add-text command not found")
			}

			// Reset flags to defaults before each test
			cmd.Flags().Set("object-id", "")
			cmd.Flags().Set("table-id", "")
			cmd.Flags().Set("row", "-1")
			cmd.Flags().Set("col", "-1")
			cmd.Flags().Set("text", "")
			cmd.Flags().Set("at", "0")
			cmd.Flags().Set("notes", "false")
			cmd.Flags().Set("slide-id", "")
			cmd.Flags().Set("slide-number", "0")

			// Set test flags
			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("failed to set flag --%s: %v", flag, err)
				}
			}

			// Capture os.Stdout since the printer writes directly to it
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute the command with a dummy presentation ID
			_ = cmd.RunE(cmd, []string{"test-presentation-id"})

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			captured, _ := io.ReadAll(r)
			output := string(captured)

			if !strings.Contains(output, tt.expectedError) {
				t.Errorf("expected output containing %q, got %q", tt.expectedError, output)
			}
		})
	}
}

// TestExtractSpeakerNotes tests extracting text from speaker notes
func TestExtractSpeakerNotes(t *testing.T) {
	slide := &slides.Page{
		ObjectId: "slide-1",
		SlideProperties: &slides.SlideProperties{
			NotesPage: &slides.Page{
				NotesProperties: &slides.NotesProperties{
					SpeakerNotesObjectId: "notes-shape-1",
				},
				PageElements: []*slides.PageElement{
					{
						ObjectId: "notes-shape-1",
						Shape: &slides.Shape{
							Text: &slides.TextContent{
								TextElements: []*slides.TextElement{
									{TextRun: &slides.TextRun{Content: "These are my speaker notes"}},
								},
							},
						},
					},
				},
			},
		},
	}

	notes := extractSpeakerNotes(slide)
	if notes != "These are my speaker notes" {
		t.Errorf("expected 'These are my speaker notes', got '%s'", notes)
	}
}

// TestExtractSpeakerNotes_Empty tests nil/missing notes page
func TestExtractSpeakerNotes_Empty(t *testing.T) {
	tests := []struct {
		name  string
		slide *slides.Page
	}{
		{"nil SlideProperties", &slides.Page{ObjectId: "s1"}},
		{"nil NotesPage", &slides.Page{
			ObjectId:        "s2",
			SlideProperties: &slides.SlideProperties{},
		}},
		{"nil NotesProperties", &slides.Page{
			ObjectId: "s3",
			SlideProperties: &slides.SlideProperties{
				NotesPage: &slides.Page{},
			},
		}},
		{"empty SpeakerNotesObjectId", &slides.Page{
			ObjectId: "s4",
			SlideProperties: &slides.SlideProperties{
				NotesPage: &slides.Page{
					NotesProperties: &slides.NotesProperties{
						SpeakerNotesObjectId: "",
					},
				},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := extractSpeakerNotes(tt.slide)
			if notes != "" {
				t.Errorf("expected empty string, got '%s'", notes)
			}
		})
	}
}

// TestGetSpeakerNotesObjectID tests successful retrieval
func TestGetSpeakerNotesObjectID(t *testing.T) {
	slide := &slides.Page{
		SlideProperties: &slides.SlideProperties{
			NotesPage: &slides.Page{
				NotesProperties: &slides.NotesProperties{
					SpeakerNotesObjectId: "notes-shape-abc",
				},
			},
		},
	}

	id, err := getSpeakerNotesObjectID(slide)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "notes-shape-abc" {
		t.Errorf("expected 'notes-shape-abc', got '%s'", id)
	}
}

// TestGetSpeakerNotesObjectID_NoNotesPage tests error case
func TestGetSpeakerNotesObjectID_NoNotesPage(t *testing.T) {
	tests := []struct {
		name  string
		slide *slides.Page
	}{
		{"nil SlideProperties", &slides.Page{}},
		{"nil NotesPage", &slides.Page{
			SlideProperties: &slides.SlideProperties{},
		}},
		{"nil NotesProperties", &slides.Page{
			SlideProperties: &slides.SlideProperties{
				NotesPage: &slides.Page{},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getSpeakerNotesObjectID(tt.slide)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// TestSlidesReadCommand_NotesFlag tests that --notes flag exists on read
func TestSlidesReadCommand_NotesFlag(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "read")
	if cmd == nil {
		t.Fatal("slides read command not found")
	}
	if cmd.Flags().Lookup("notes") == nil {
		t.Error("expected --notes flag on read command")
	}
}

// TestSlidesListCommand_NotesFlag tests that --notes flag exists on list
func TestSlidesListCommand_NotesFlag(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "list")
	if cmd == nil {
		t.Fatal("slides list command not found")
	}
	if cmd.Flags().Lookup("notes") == nil {
		t.Error("expected --notes flag on list command")
	}
}

// TestSlidesInfoCommand_NotesFlag tests that --notes flag exists on info
func TestSlidesInfoCommand_NotesFlag(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "info")
	if cmd == nil {
		t.Fatal("slides info command not found")
	}
	if cmd.Flags().Lookup("notes") == nil {
		t.Error("expected --notes flag on info command")
	}
}

// TestSlidesAddTextCommand_NotesValidation tests notes mode mutual exclusivity
func TestSlidesAddTextCommand_NotesValidation(t *testing.T) {
	tests := []struct {
		name          string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "notes with object-id",
			flags: map[string]string{
				"object-id": "shape-123",
				"notes":     "true",
				"text":      "test",
			},
			expectedError: "--object-id, --table-id, and --notes are mutually exclusive",
		},
		{
			name: "notes with table-id",
			flags: map[string]string{
				"table-id": "table-456",
				"notes":    "true",
				"text":     "test",
			},
			expectedError: "--object-id, --table-id, and --notes are mutually exclusive",
		},
		{
			name: "notes without slide targeting",
			flags: map[string]string{
				"notes": "true",
				"text":  "test",
			},
			expectedError: "--notes requires --slide-id or --slide-number",
		},
		{
			name: "no mode specified",
			flags: map[string]string{
				"text": "test",
			},
			expectedError: "must specify --object-id, --table-id, or --notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(slidesCmd, "add-text")
			if cmd == nil {
				t.Fatal("slides add-text command not found")
			}

			// Reset flags to defaults
			cmd.Flags().Set("object-id", "")
			cmd.Flags().Set("table-id", "")
			cmd.Flags().Set("row", "-1")
			cmd.Flags().Set("col", "-1")
			cmd.Flags().Set("text", "")
			cmd.Flags().Set("at", "0")
			cmd.Flags().Set("notes", "false")
			cmd.Flags().Set("slide-id", "")
			cmd.Flags().Set("slide-number", "0")

			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("failed to set flag --%s: %v", flag, err)
				}
			}

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_ = cmd.RunE(cmd, []string{"test-presentation-id"})

			w.Close()
			os.Stdout = oldStdout
			captured, _ := io.ReadAll(r)
			output := string(captured)

			if !strings.Contains(output, tt.expectedError) {
				t.Errorf("expected output containing %q, got %q", tt.expectedError, output)
			}
		})
	}
}

// TestSlidesDeleteTextCommand_NotesValidation tests delete-text notes validation
func TestSlidesDeleteTextCommand_NotesValidation(t *testing.T) {
	tests := []struct {
		name          string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "notes with object-id",
			flags: map[string]string{
				"object-id": "shape-123",
				"notes":     "true",
			},
			expectedError: "--object-id and --notes are mutually exclusive",
		},
		{
			name: "notes without slide targeting",
			flags: map[string]string{
				"notes": "true",
			},
			expectedError: "--notes requires --slide-id or --slide-number",
		},
		{
			name:          "no mode specified",
			flags:         map[string]string{},
			expectedError: "must specify --object-id or --notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(slidesCmd, "delete-text")
			if cmd == nil {
				t.Fatal("slides delete-text command not found")
			}

			// Reset flags to defaults
			cmd.Flags().Set("object-id", "")
			cmd.Flags().Set("from", "0")
			cmd.Flags().Set("to", "-1")
			cmd.Flags().Set("notes", "false")
			cmd.Flags().Set("slide-id", "")
			cmd.Flags().Set("slide-number", "0")

			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("failed to set flag --%s: %v", flag, err)
				}
			}

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_ = cmd.RunE(cmd, []string{"test-presentation-id"})

			w.Close()
			os.Stdout = oldStdout
			captured, _ := io.ReadAll(r)
			output := string(captured)

			if !strings.Contains(output, tt.expectedError) {
				t.Errorf("expected output containing %q, got %q", tt.expectedError, output)
			}
		})
	}
}

// TestFindSlide tests the findSlide helper
func TestFindSlide(t *testing.T) {
	pres := &slides.Presentation{
		Slides: []*slides.Page{
			{ObjectId: "slide-a"},
			{ObjectId: "slide-b"},
		},
	}

	// By slide ID
	s, err := findSlide(pres, "slide-b", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ObjectId != "slide-b" {
		t.Errorf("expected slide-b, got %s", s.ObjectId)
	}

	// By slide number
	s, err = findSlide(pres, "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ObjectId != "slide-a" {
		t.Errorf("expected slide-a, got %s", s.ObjectId)
	}

	// Both specified
	_, err = findSlide(pres, "slide-a", 1)
	if err == nil {
		t.Error("expected error when both slide-id and slide-number specified")
	}

	// Not found
	_, err = findSlide(pres, "nonexistent", 0)
	if err == nil {
		t.Error("expected error for nonexistent slide ID")
	}

	// Out of range
	_, err = findSlide(pres, "", 99)
	if err == nil {
		t.Error("expected error for out-of-range slide number")
	}
}

// TestSlidesReplaceTextCommand_Flags tests replace-text command flags
func TestSlidesReplaceTextCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "replace-text")
	if cmd == nil {
		t.Fatal("slides replace-text command not found")
	}

	expectedFlags := []string{"find", "replace", "match-case"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesAddShape_Success tests creating a shape
func TestSlidesAddShape_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-shape:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			createShape := req.Requests[0].CreateShape
			if createShape == nil {
				t.Error("expected CreateShape request")
			} else {
				if createShape.ShapeType != "RECTANGLE" {
					t.Errorf("expected shape type 'RECTANGLE', got '%s'", createShape.ShapeType)
				}
				if createShape.ElementProperties.PageObjectId != "slide-1" {
					t.Errorf("expected page object ID 'slide-1', got '%s'", createShape.ElementProperties.PageObjectId)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-shape",
				Replies: []*slides.Response{
					{
						CreateShape: &slides.CreateShapeResponse{
							ObjectId: "new-shape-id",
						},
					},
				},
			})
		},
		"/v1/presentations/pres-shape": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&slides.Presentation{
				PresentationId: "pres-shape",
				Slides: []*slides.Page{
					{ObjectId: "slide-1"},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	resp, err := svc.Presentations.BatchUpdate("pres-shape", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateShape: &slides.CreateShapeRequest{
					ShapeType: "RECTANGLE",
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: "slide-1",
						Size: &slides.Size{
							Width:  &slides.Dimension{Magnitude: 200, Unit: "PT"},
							Height: &slides.Dimension{Magnitude: 100, Unit: "PT"},
						},
						Transform: &slides.AffineTransform{
							ScaleX:     1,
							ScaleY:     1,
							TranslateX: 100,
							TranslateY: 100,
							Unit:       "PT",
						},
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to create shape: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}

	if len(resp.Replies) == 0 || resp.Replies[0].CreateShape == nil {
		t.Error("expected CreateShape response")
	} else if resp.Replies[0].CreateShape.ObjectId != "new-shape-id" {
		t.Errorf("expected shape ID 'new-shape-id', got '%s'", resp.Replies[0].CreateShape.ObjectId)
	}
}

// TestSlidesAddImage_Success tests adding an image
func TestSlidesAddImage_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-image:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			createImage := req.Requests[0].CreateImage
			if createImage == nil {
				t.Error("expected CreateImage request")
			} else {
				if createImage.Url != "https://example.com/image.png" {
					t.Errorf("expected URL 'https://example.com/image.png', got '%s'", createImage.Url)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-image",
				Replies: []*slides.Response{
					{
						CreateImage: &slides.CreateImageResponse{
							ObjectId: "new-image-id",
						},
					},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	resp, err := svc.Presentations.BatchUpdate("pres-image", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateImage: &slides.CreateImageRequest{
					Url: "https://example.com/image.png",
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: "slide-1",
						Size: &slides.Size{
							Width: &slides.Dimension{Magnitude: 400, Unit: "PT"},
						},
						Transform: &slides.AffineTransform{
							ScaleX:     1,
							ScaleY:     1,
							TranslateX: 100,
							TranslateY: 100,
							Unit:       "PT",
						},
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to create image: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}

	if len(resp.Replies) == 0 || resp.Replies[0].CreateImage == nil {
		t.Error("expected CreateImage response")
	} else if resp.Replies[0].CreateImage.ObjectId != "new-image-id" {
		t.Errorf("expected image ID 'new-image-id', got '%s'", resp.Replies[0].CreateImage.ObjectId)
	}
}

// TestSlidesAddText_Success tests inserting text
func TestSlidesAddText_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-text:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			insertText := req.Requests[0].InsertText
			if insertText == nil {
				t.Error("expected InsertText request")
			} else {
				if insertText.ObjectId != "text-box-1" {
					t.Errorf("expected object ID 'text-box-1', got '%s'", insertText.ObjectId)
				}
				if insertText.Text != "Hello World" {
					t.Errorf("expected text 'Hello World', got '%s'", insertText.Text)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-text",
				Replies:        []*slides.Response{{}},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-text", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				InsertText: &slides.InsertTextRequest{
					ObjectId:       "text-box-1",
					Text:           "Hello World",
					InsertionIndex: 0,
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

// TestSlidesAddTextToTableCell_Success tests inserting text into a table cell
func TestSlidesAddTextToTableCell_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-table-text:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			insertText := req.Requests[0].InsertText
			if insertText == nil {
				t.Error("expected InsertText request")
			} else {
				if insertText.ObjectId != "table-123" {
					t.Errorf("expected object ID 'table-123', got '%s'", insertText.ObjectId)
				}
				if insertText.Text != "Cell Content" {
					t.Errorf("expected text 'Cell Content', got '%s'", insertText.Text)
				}
				if insertText.CellLocation == nil {
					t.Error("expected CellLocation to be set for table cell")
				} else {
					if insertText.CellLocation.RowIndex != 1 {
						t.Errorf("expected row index 1, got %d", insertText.CellLocation.RowIndex)
					}
					if insertText.CellLocation.ColumnIndex != 2 {
						t.Errorf("expected column index 2, got %d", insertText.CellLocation.ColumnIndex)
					}
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-table-text",
				Replies:        []*slides.Response{{}},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-table-text", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				InsertText: &slides.InsertTextRequest{
					ObjectId: "table-123",
					Text:     "Cell Content",
					CellLocation: &slides.TableCellLocation{
						RowIndex:    1,
						ColumnIndex: 2,
					},
					InsertionIndex: 0,
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to insert text into table cell: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesAddText_ShapeMode_NoCellLocation verifies shape mode doesn't set CellLocation
func TestSlidesAddText_ShapeMode_NoCellLocation(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-shape-text:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			insertText := req.Requests[0].InsertText
			if insertText == nil {
				t.Error("expected InsertText request")
			} else {
				if insertText.ObjectId != "shape-456" {
					t.Errorf("expected object ID 'shape-456', got '%s'", insertText.ObjectId)
				}
				// CellLocation should NOT be set for shape mode
				if insertText.CellLocation != nil {
					t.Error("CellLocation should be nil for shape mode (backward compatibility)")
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-shape-text",
				Replies:        []*slides.Response{{}},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	// Shape mode: only ObjectId, no CellLocation
	_, err = svc.Presentations.BatchUpdate("pres-shape-text", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				InsertText: &slides.InsertTextRequest{
					ObjectId:       "shape-456",
					Text:           "Shape text",
					InsertionIndex: 0,
					// CellLocation intentionally nil
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to insert text into shape: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesReplaceText_Success tests find and replace
func TestSlidesReplaceText_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-replace:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			replaceText := req.Requests[0].ReplaceAllText
			if replaceText == nil {
				t.Error("expected ReplaceAllText request")
			} else {
				if replaceText.ContainsText.Text != "old text" {
					t.Errorf("expected find text 'old text', got '%s'", replaceText.ContainsText.Text)
				}
				if replaceText.ReplaceText != "new text" {
					t.Errorf("expected replace text 'new text', got '%s'", replaceText.ReplaceText)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-replace",
				Replies: []*slides.Response{
					{
						ReplaceAllText: &slides.ReplaceAllTextResponse{
							OccurrencesChanged: 5,
						},
					},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	resp, err := svc.Presentations.BatchUpdate("pres-replace", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				ReplaceAllText: &slides.ReplaceAllTextRequest{
					ContainsText: &slides.SubstringMatchCriteria{
						Text:      "old text",
						MatchCase: true,
					},
					ReplaceText: "new text",
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

// TestSlidesCommands_Structure_Extended tests that all new slides commands are registered
func TestSlidesCommands_Structure_Extended(t *testing.T) {
	commands := []string{
		"add-shape",
		"add-image",
		"add-text",
		"replace-text",
		"delete-object",
		"delete-text",
		"update-text-style",
		"update-transform",
		"create-table",
		"insert-table-rows",
		"delete-table-row",
		"update-table-cell",
		"update-table-border",
		"update-paragraph-style",
		"update-shape",
		"reorder-slides",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := findSubcommand(slidesCmd, cmdName)
			if cmd == nil {
				t.Fatalf("command '%s' not found", cmdName)
			}
		})
	}
}

// TestSlidesDeleteObjectCommand_Flags tests delete-object command flags
func TestSlidesDeleteObjectCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "delete-object")
	if cmd == nil {
		t.Fatal("slides delete-object command not found")
	}

	expectedFlags := []string{"object-id"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesDeleteTextCommand_Flags tests delete-text command flags
func TestSlidesDeleteTextCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "delete-text")
	if cmd == nil {
		t.Fatal("slides delete-text command not found")
	}

	expectedFlags := []string{"object-id", "from", "to"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesUpdateTextStyleCommand_Flags tests update-text-style command flags
func TestSlidesUpdateTextStyleCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "update-text-style")
	if cmd == nil {
		t.Fatal("slides update-text-style command not found")
	}

	expectedFlags := []string{"object-id", "from", "to", "bold", "italic", "underline", "font-size", "font-family", "color"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesUpdateTransformCommand_Flags tests update-transform command flags
func TestSlidesUpdateTransformCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "update-transform")
	if cmd == nil {
		t.Fatal("slides update-transform command not found")
	}

	expectedFlags := []string{"object-id", "x", "y", "scale-x", "scale-y", "rotate"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesCreateTableCommand_Flags tests create-table command flags
func TestSlidesCreateTableCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "create-table")
	if cmd == nil {
		t.Fatal("slides create-table command not found")
	}

	expectedFlags := []string{"slide-id", "slide-number", "rows", "cols", "x", "y", "width", "height"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesInsertTableRowsCommand_Flags tests insert-table-rows command flags
func TestSlidesInsertTableRowsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "insert-table-rows")
	if cmd == nil {
		t.Fatal("slides insert-table-rows command not found")
	}

	expectedFlags := []string{"table-id", "at", "count", "below"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesDeleteTableRowCommand_Flags tests delete-table-row command flags
func TestSlidesDeleteTableRowCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "delete-table-row")
	if cmd == nil {
		t.Fatal("slides delete-table-row command not found")
	}

	expectedFlags := []string{"table-id", "row"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesUpdateTableCellCommand_Flags tests update-table-cell command flags
func TestSlidesUpdateTableCellCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "update-table-cell")
	if cmd == nil {
		t.Fatal("slides update-table-cell command not found")
	}

	expectedFlags := []string{"table-id", "row", "col", "background-color"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesUpdateTableBorderCommand_Flags tests update-table-border command flags
func TestSlidesUpdateTableBorderCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "update-table-border")
	if cmd == nil {
		t.Fatal("slides update-table-border command not found")
	}

	expectedFlags := []string{"table-id", "row", "col", "border", "color", "width", "style"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesUpdateParagraphStyleCommand_Flags tests update-paragraph-style command flags
func TestSlidesUpdateParagraphStyleCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "update-paragraph-style")
	if cmd == nil {
		t.Fatal("slides update-paragraph-style command not found")
	}

	expectedFlags := []string{"object-id", "from", "to", "alignment", "line-spacing", "space-above", "space-below"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesUpdateShapeCommand_Flags tests update-shape command flags
func TestSlidesUpdateShapeCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "update-shape")
	if cmd == nil {
		t.Fatal("slides update-shape command not found")
	}

	expectedFlags := []string{"object-id", "background-color", "outline-color", "outline-width"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestSlidesReorderSlidesCommand_Flags tests reorder-slides command flags
func TestSlidesReorderSlidesCommand_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "reorder-slides")
	if cmd == nil {
		t.Fatal("slides reorder-slides command not found")
	}

	expectedFlags := []string{"slide-ids", "to"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
	}
}

// TestParseHexColor tests the parseHexColor helper function
func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantRed   float64
		wantGreen float64
		wantBlue  float64
		wantErr   bool
	}{
		{"red", "#FF0000", 1.0, 0.0, 0.0, false},
		{"green", "#00FF00", 0.0, 1.0, 0.0, false},
		{"blue", "#0000FF", 0.0, 0.0, 1.0, false},
		{"white", "#FFFFFF", 1.0, 1.0, 1.0, false},
		{"black", "#000000", 0.0, 0.0, 0.0, false},
		{"lowercase", "#ff00ff", 1.0, 0.0, 1.0, false},
		{"missing hash", "FF0000", 0, 0, 0, true},
		{"too short", "#FFF", 0, 0, 0, true},
		{"invalid chars", "#GGGGGG", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color, err := parseHexColor(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %s", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if color.Red != tt.wantRed {
				t.Errorf("red: expected %f, got %f", tt.wantRed, color.Red)
			}
			if color.Green != tt.wantGreen {
				t.Errorf("green: expected %f, got %f", tt.wantGreen, color.Green)
			}
			if color.Blue != tt.wantBlue {
				t.Errorf("blue: expected %f, got %f", tt.wantBlue, color.Blue)
			}
		})
	}
}

// TestSlidesDeleteObject_Success tests deleting an object
func TestSlidesDeleteObject_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-del-obj:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Requests) == 0 {
				t.Error("expected at least one request")
			}

			deleteReq := req.Requests[0].DeleteObject
			if deleteReq == nil {
				t.Error("expected DeleteObject request")
			} else if deleteReq.ObjectId != "shape-to-delete" {
				t.Errorf("expected object ID 'shape-to-delete', got '%s'", deleteReq.ObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-del-obj",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-del-obj", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: "shape-to-delete",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to delete object: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesDeleteText_Success tests deleting text from a shape
func TestSlidesDeleteText_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-del-text:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			deleteTextReq := req.Requests[0].DeleteText
			if deleteTextReq == nil {
				t.Error("expected DeleteText request")
			} else if deleteTextReq.ObjectId != "text-shape" {
				t.Errorf("expected object ID 'text-shape', got '%s'", deleteTextReq.ObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-del-text",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	startIdx := int64(0)
	_, err = svc.Presentations.BatchUpdate("pres-del-text", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteText: &slides.DeleteTextRequest{
					ObjectId: "text-shape",
					TextRange: &slides.Range{
						StartIndex: &startIdx,
						Type:       "FROM_START_INDEX",
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to delete text: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesUpdateTextStyle_Success tests updating text style
func TestSlidesUpdateTextStyle_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-style:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			updateReq := req.Requests[0].UpdateTextStyle
			if updateReq == nil {
				t.Error("expected UpdateTextStyle request")
			} else {
				if updateReq.ObjectId != "styled-shape" {
					t.Errorf("expected object ID 'styled-shape', got '%s'", updateReq.ObjectId)
				}
				if updateReq.Style.Bold != true {
					t.Error("expected bold to be true")
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-style",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-style", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateTextStyle: &slides.UpdateTextStyleRequest{
					ObjectId: "styled-shape",
					TextRange: &slides.Range{
						Type: "ALL",
					},
					Style: &slides.TextStyle{
						Bold: true,
					},
					Fields: "bold",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to update text style: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesCreateTable_Success tests creating a table
func TestSlidesCreateTable_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-table:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			createReq := req.Requests[0].CreateTable
			if createReq == nil {
				t.Error("expected CreateTable request")
			} else {
				if createReq.Rows != 3 {
					t.Errorf("expected 3 rows, got %d", createReq.Rows)
				}
				if createReq.Columns != 4 {
					t.Errorf("expected 4 columns, got %d", createReq.Columns)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-table",
				Replies: []*slides.Response{
					{
						CreateTable: &slides.CreateTableResponse{
							ObjectId: "new-table-id",
						},
					},
				},
			})
		},
		"/v1/presentations/pres-table": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&slides.Presentation{
				PresentationId: "pres-table",
				Slides: []*slides.Page{
					{ObjectId: "slide-1"},
				},
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	resp, err := svc.Presentations.BatchUpdate("pres-table", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateTable: &slides.CreateTableRequest{
					Rows:    3,
					Columns: 4,
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: "slide-1",
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}

	if len(resp.Replies) == 0 || resp.Replies[0].CreateTable == nil {
		t.Error("expected CreateTable response")
	} else if resp.Replies[0].CreateTable.ObjectId != "new-table-id" {
		t.Errorf("expected table ID 'new-table-id', got '%s'", resp.Replies[0].CreateTable.ObjectId)
	}
}

// TestSlidesInsertTableRows_Success tests inserting table rows
func TestSlidesInsertTableRows_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-insert-rows:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			insertReq := req.Requests[0].InsertTableRows
			if insertReq == nil {
				t.Error("expected InsertTableRows request")
			} else {
				if insertReq.TableObjectId != "table-1" {
					t.Errorf("expected table ID 'table-1', got '%s'", insertReq.TableObjectId)
				}
				if insertReq.Number != 2 {
					t.Errorf("expected 2 rows, got %d", insertReq.Number)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-insert-rows",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-insert-rows", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				InsertTableRows: &slides.InsertTableRowsRequest{
					TableObjectId: "table-1",
					CellLocation: &slides.TableCellLocation{
						RowIndex: 0,
					},
					InsertBelow: true,
					Number:      2,
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to insert table rows: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesDeleteTableRow_Success tests deleting a table row
func TestSlidesDeleteTableRow_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-del-row:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			deleteReq := req.Requests[0].DeleteTableRow
			if deleteReq == nil {
				t.Error("expected DeleteTableRow request")
			} else if deleteReq.TableObjectId != "table-1" {
				t.Errorf("expected table ID 'table-1', got '%s'", deleteReq.TableObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-del-row",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-del-row", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteTableRow: &slides.DeleteTableRowRequest{
					TableObjectId: "table-1",
					CellLocation: &slides.TableCellLocation{
						RowIndex: 1,
					},
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to delete table row: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesUpdateTableCell_Success tests updating table cell properties
func TestSlidesUpdateTableCell_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-cell:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			updateReq := req.Requests[0].UpdateTableCellProperties
			if updateReq == nil {
				t.Error("expected UpdateTableCellProperties request")
			} else if updateReq.ObjectId != "table-1" {
				t.Errorf("expected table ID 'table-1', got '%s'", updateReq.ObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-cell",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-cell", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateTableCellProperties: &slides.UpdateTableCellPropertiesRequest{
					ObjectId: "table-1",
					TableRange: &slides.TableRange{
						Location: &slides.TableCellLocation{
							RowIndex:    0,
							ColumnIndex: 0,
						},
						RowSpan:    1,
						ColumnSpan: 1,
					},
					TableCellProperties: &slides.TableCellProperties{
						TableCellBackgroundFill: &slides.TableCellBackgroundFill{
							SolidFill: &slides.SolidFill{
								Color: &slides.OpaqueColor{
									RgbColor: &slides.RgbColor{
										Red: 1.0, Green: 0, Blue: 0,
									},
								},
							},
						},
					},
					Fields: "tableCellBackgroundFill",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to update table cell: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesUpdateShape_Success tests updating shape properties
func TestSlidesUpdateShape_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-shape-props:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			updateReq := req.Requests[0].UpdateShapeProperties
			if updateReq == nil {
				t.Error("expected UpdateShapeProperties request")
			} else if updateReq.ObjectId != "shape-1" {
				t.Errorf("expected object ID 'shape-1', got '%s'", updateReq.ObjectId)
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-shape-props",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-shape-props", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateShapeProperties: &slides.UpdateShapePropertiesRequest{
					ObjectId: "shape-1",
					ShapeProperties: &slides.ShapeProperties{
						ShapeBackgroundFill: &slides.ShapeBackgroundFill{
							SolidFill: &slides.SolidFill{
								Color: &slides.OpaqueColor{
									RgbColor: &slides.RgbColor{
										Red: 0, Green: 0, Blue: 1.0,
									},
								},
							},
						},
					},
					Fields: "shapeBackgroundFill",
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to update shape: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

// TestSlidesReorderSlides_Success tests reordering slides
func TestSlidesReorderSlides_Success(t *testing.T) {
	batchUpdateCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-reorder:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			batchUpdateCalled = true

			var req slides.BatchUpdatePresentationRequest
			json.NewDecoder(r.Body).Decode(&req)

			reorderReq := req.Requests[0].UpdateSlidesPosition
			if reorderReq == nil {
				t.Error("expected UpdateSlidesPosition request")
			} else {
				if len(reorderReq.SlideObjectIds) != 2 {
					t.Errorf("expected 2 slide IDs, got %d", len(reorderReq.SlideObjectIds))
				}
				if reorderReq.InsertionIndex != 0 {
					t.Errorf("expected insertion index 0, got %d", reorderReq.InsertionIndex)
				}
			}

			json.NewEncoder(w).Encode(&slides.BatchUpdatePresentationResponse{
				PresentationId: "pres-reorder",
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	_, err = svc.Presentations.BatchUpdate("pres-reorder", &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateSlidesPosition: &slides.UpdateSlidesPositionRequest{
					SlideObjectIds: []string{"slide-3", "slide-4"},
					InsertionIndex: 0,
				},
			},
		},
	}).Do()
	if err != nil {
		t.Fatalf("failed to reorder slides: %v", err)
	}

	if !batchUpdateCalled {
		t.Error("batchUpdate endpoint was not called")
	}
}

func TestSlidesThumbnail_Flags(t *testing.T) {
	cmd := findSubcommand(slidesCmd, "thumbnail")
	if cmd == nil {
		t.Fatal("slides thumbnail command not found")
	}

	if cmd.Use != "thumbnail <presentation-id>" {
		t.Errorf("expected Use 'thumbnail <presentation-id>', got '%s'", cmd.Use)
	}

	slideFlag := cmd.Flags().Lookup("slide")
	if slideFlag == nil {
		t.Fatal("expected --slide flag")
	}

	sizeFlag := cmd.Flags().Lookup("size")
	if sizeFlag == nil {
		t.Fatal("expected --size flag")
	}
	if sizeFlag.DefValue != "MEDIUM" {
		t.Errorf("expected --size default 'MEDIUM', got '%s'", sizeFlag.DefValue)
	}

	downloadFlag := cmd.Flags().Lookup("download")
	if downloadFlag == nil {
		t.Fatal("expected --download flag")
	}
}

func TestSlidesThumbnail_GetByObjectID(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-thumb/pages/slide-abc/thumbnail": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			sizeParam := r.URL.Query().Get("thumbnailProperties.thumbnailSize")
			if sizeParam != "LARGE" {
				t.Errorf("expected size LARGE, got '%s'", sizeParam)
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"contentUrl": "https://example.com/thumb.png",
				"width":      1600,
				"height":     900,
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	thumbnail, err := svc.Presentations.Pages.GetThumbnail("pres-thumb", "slide-abc").
		ThumbnailPropertiesThumbnailSize("LARGE").
		Do()
	if err != nil {
		t.Fatalf("failed to get thumbnail: %v", err)
	}

	if thumbnail.ContentUrl != "https://example.com/thumb.png" {
		t.Errorf("expected contentUrl 'https://example.com/thumb.png', got '%s'", thumbnail.ContentUrl)
	}
	if thumbnail.Width != 1600 {
		t.Errorf("expected width 1600, got %d", thumbnail.Width)
	}
	if thumbnail.Height != 900 {
		t.Errorf("expected height 900, got %d", thumbnail.Height)
	}
}

func TestSlidesThumbnail_GetBySlideNumber(t *testing.T) {
	getCalled := false
	thumbnailCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-num": func(w http.ResponseWriter, r *http.Request) {
			getCalled = true
			json.NewEncoder(w).Encode(&slides.Presentation{
				PresentationId: "pres-num",
				Slides: []*slides.Page{
					{ObjectId: "page-one"},
					{ObjectId: "page-two"},
					{ObjectId: "page-three"},
				},
			})
		},
		"/v1/presentations/pres-num/pages/page-two/thumbnail": func(w http.ResponseWriter, r *http.Request) {
			thumbnailCalled = true
			json.NewEncoder(w).Encode(map[string]interface{}{
				"contentUrl": "https://example.com/page-two.png",
				"width":      800,
				"height":     450,
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	// Simulate slide number resolution: fetch presentation, then use slide object ID
	pres, err := svc.Presentations.Get("pres-num").Do()
	if err != nil {
		t.Fatalf("failed to get presentation: %v", err)
	}

	slideNumber := 2
	if slideNumber > len(pres.Slides) {
		t.Fatalf("slide number %d out of range", slideNumber)
	}
	pageObjectID := pres.Slides[slideNumber-1].ObjectId

	thumbnail, err := svc.Presentations.Pages.GetThumbnail("pres-num", pageObjectID).
		ThumbnailPropertiesThumbnailSize("MEDIUM").
		Do()
	if err != nil {
		t.Fatalf("failed to get thumbnail: %v", err)
	}

	if !getCalled {
		t.Error("presentation get endpoint was not called")
	}
	if !thumbnailCalled {
		t.Error("thumbnail endpoint was not called")
	}
	if thumbnail.ContentUrl != "https://example.com/page-two.png" {
		t.Errorf("expected contentUrl for page-two, got '%s'", thumbnail.ContentUrl)
	}
}

func TestSlidesThumbnail_Download(t *testing.T) {
	// Serve a thumbnail image
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-png-data"))
	}))
	defer imageServer.Close()

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/presentations/pres-dl/pages/slide-dl/thumbnail": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"contentUrl": imageServer.URL + "/thumb.png",
				"width":      800,
				"height":     450,
			})
		},
	}

	server := mockSlidesServer(t, handlers)
	defer server.Close()

	svc, err := slides.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create slides service: %v", err)
	}

	thumbnail, err := svc.Presentations.Pages.GetThumbnail("pres-dl", "slide-dl").
		ThumbnailPropertiesThumbnailSize("MEDIUM").
		Do()
	if err != nil {
		t.Fatalf("failed to get thumbnail: %v", err)
	}

	// Download the image
	resp, err := http.Get(thumbnail.ContentUrl)
	if err != nil {
		t.Fatalf("failed to download thumbnail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "thumbnail-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		t.Fatalf("failed to write thumbnail: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "fake-png-data" {
		t.Errorf("expected 'fake-png-data', got '%s'", string(data))
	}
}
