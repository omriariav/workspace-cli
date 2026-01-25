package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	expectedFlags := []string{"object-id", "text", "at"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' not found", flag)
		}
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
