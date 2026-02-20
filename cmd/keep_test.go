package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/keep/v1"
	"google.golang.org/api/option"
)

// TestKeepCommands tests keep command structure
func TestKeepCommands(t *testing.T) {
	tests := []struct {
		name string
		use  string
	}{
		{"list", "list"},
		{"get", "get <note-id>"},
		{"create", "create"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(keepCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
			if cmd.Use != tt.use {
				t.Errorf("expected Use '%s', got '%s'", tt.use, cmd.Use)
			}
		})
	}
}

func TestKeepListCommand_Flags(t *testing.T) {
	cmd := findSubcommand(keepCmd, "list")
	if cmd == nil {
		t.Fatal("keep list command not found")
	}

	maxFlag := cmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Error("expected --max flag")
	}
	if maxFlag.DefValue != "20" {
		t.Errorf("expected --max default '20', got '%s'", maxFlag.DefValue)
	}
}

func TestKeepCreateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(keepCmd, "create")
	if cmd == nil {
		t.Fatal("keep create command not found")
	}

	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Error("expected --title flag")
	}

	textFlag := cmd.Flags().Lookup("text")
	if textFlag == nil {
		t.Error("expected --text flag")
	}
}

func TestKeepGetCommand_Help(t *testing.T) {
	cmd := keepGetCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestKeepCreateCommand_Help(t *testing.T) {
	cmd := keepCreateCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// mockKeepServer creates a test server that mocks Keep API responses
func mockKeepServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

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

func TestKeepList_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/notes": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			resp := &keep.ListNotesResponse{
				Notes: []*keep.Note{
					{
						Name:       "notes/abc123",
						Title:      "Shopping List",
						CreateTime: "2026-02-15T10:00:00Z",
						UpdateTime: "2026-02-15T10:30:00Z",
						Body: &keep.Section{
							Text: &keep.TextContent{
								Text: "Milk, eggs, bread",
							},
						},
					},
					{
						Name:       "notes/def456",
						Title:      "Meeting Notes",
						CreateTime: "2026-02-16T09:00:00Z",
						UpdateTime: "2026-02-16T09:15:00Z",
						Body: &keep.Section{
							Text: &keep.TextContent{
								Text: "Discuss Q1 goals",
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockKeepServer(t, handlers)
	defer server.Close()

	svc, err := keep.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create keep service: %v", err)
	}

	resp, err := svc.Notes.List().PageSize(20).Do()
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(resp.Notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(resp.Notes))
	}
	if resp.Notes[0].Name != "notes/abc123" {
		t.Errorf("expected first note name 'notes/abc123', got '%s'", resp.Notes[0].Name)
	}
	if resp.Notes[0].Title != "Shopping List" {
		t.Errorf("expected title 'Shopping List', got '%s'", resp.Notes[0].Title)
	}
	if resp.Notes[0].Body.Text.Text != "Milk, eggs, bread" {
		t.Errorf("expected text 'Milk, eggs, bread', got '%s'", resp.Notes[0].Body.Text.Text)
	}
	if resp.Notes[1].Name != "notes/def456" {
		t.Errorf("expected second note name 'notes/def456', got '%s'", resp.Notes[1].Name)
	}
}

func TestKeepGet_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/notes/abc123": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			resp := &keep.Note{
				Name:       "notes/abc123",
				Title:      "Shopping List",
				CreateTime: "2026-02-15T10:00:00Z",
				UpdateTime: "2026-02-15T10:30:00Z",
				Body: &keep.Section{
					Text: &keep.TextContent{
						Text: "Milk, eggs, bread",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockKeepServer(t, handlers)
	defer server.Close()

	svc, err := keep.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create keep service: %v", err)
	}

	note, err := svc.Notes.Get("notes/abc123").Do()
	if err != nil {
		t.Fatalf("failed to get note: %v", err)
	}

	if note.Name != "notes/abc123" {
		t.Errorf("expected name 'notes/abc123', got '%s'", note.Name)
	}
	if note.Title != "Shopping List" {
		t.Errorf("expected title 'Shopping List', got '%s'", note.Title)
	}
	if note.Body == nil || note.Body.Text == nil {
		t.Fatal("expected note body with text content")
	}
	if note.Body.Text.Text != "Milk, eggs, bread" {
		t.Errorf("expected text 'Milk, eggs, bread', got '%s'", note.Body.Text.Text)
	}
	if note.CreateTime != "2026-02-15T10:00:00Z" {
		t.Errorf("expected create time '2026-02-15T10:00:00Z', got '%s'", note.CreateTime)
	}
}

func TestKeepCreate_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/notes": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var req keep.Note
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			if req.Title != "New Note" {
				t.Errorf("expected title 'New Note', got '%s'", req.Title)
			}

			if req.Body == nil || req.Body.Text == nil || req.Body.Text.Text != "Some content" {
				t.Errorf("expected body text 'Some content'")
			}

			resp := &keep.Note{
				Name:       "notes/new789",
				Title:      "New Note",
				CreateTime: "2026-02-20T12:00:00Z",
				UpdateTime: "2026-02-20T12:00:00Z",
				Body: &keep.Section{
					Text: &keep.TextContent{
						Text: "Some content",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockKeepServer(t, handlers)
	defer server.Close()

	svc, err := keep.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create keep service: %v", err)
	}

	newNote := &keep.Note{
		Title: "New Note",
		Body: &keep.Section{
			Text: &keep.TextContent{
				Text: "Some content",
			},
		},
	}

	note, err := svc.Notes.Create(newNote).Do()
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if note.Name != "notes/new789" {
		t.Errorf("expected name 'notes/new789', got '%s'", note.Name)
	}
	if note.Title != "New Note" {
		t.Errorf("expected title 'New Note', got '%s'", note.Title)
	}
	if note.Body.Text.Text != "Some content" {
		t.Errorf("expected text 'Some content', got '%s'", note.Body.Text.Text)
	}
}

func TestKeepList_EmptyResponse(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/notes": func(w http.ResponseWriter, r *http.Request) {
			resp := &keep.ListNotesResponse{
				Notes: []*keep.Note{},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockKeepServer(t, handlers)
	defer server.Close()

	svc, err := keep.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create keep service: %v", err)
	}

	resp, err := svc.Notes.List().Do()
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(resp.Notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(resp.Notes))
	}
}

func TestKeepGet_TrashedNote(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/notes/trashed1": func(w http.ResponseWriter, r *http.Request) {
			resp := &keep.Note{
				Name:       "notes/trashed1",
				Title:      "Old Note",
				Trashed:    true,
				CreateTime: "2026-01-01T00:00:00Z",
				UpdateTime: "2026-02-01T00:00:00Z",
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockKeepServer(t, handlers)
	defer server.Close()

	svc, err := keep.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create keep service: %v", err)
	}

	note, err := svc.Notes.Get("notes/trashed1").Do()
	if err != nil {
		t.Fatalf("failed to get note: %v", err)
	}

	if !note.Trashed {
		t.Error("expected note to be trashed")
	}
	if note.Title != "Old Note" {
		t.Errorf("expected title 'Old Note', got '%s'", note.Title)
	}
}

// TestFormatNote tests the formatNote helper
func TestFormatNote(t *testing.T) {
	n := &keep.Note{
		Name:       "notes/abc123",
		Title:      "Test Note",
		CreateTime: "2026-02-15T10:00:00Z",
		UpdateTime: "2026-02-15T10:30:00Z",
		Body: &keep.Section{
			Text: &keep.TextContent{
				Text: "Hello world",
			},
		},
	}

	result := formatNote(n)

	if result["name"] != "notes/abc123" {
		t.Errorf("expected name 'notes/abc123', got '%v'", result["name"])
	}
	if result["title"] != "Test Note" {
		t.Errorf("expected title 'Test Note', got '%v'", result["title"])
	}
	if result["text"] != "Hello world" {
		t.Errorf("expected text 'Hello world', got '%v'", result["text"])
	}
	if result["create_time"] != "2026-02-15T10:00:00Z" {
		t.Errorf("expected create_time, got '%v'", result["create_time"])
	}
	if _, ok := result["trashed"]; ok {
		t.Error("expected no trashed field when not trashed")
	}
}

func TestFormatNote_Minimal(t *testing.T) {
	n := &keep.Note{
		Name:  "notes/min",
		Title: "Minimal",
	}

	result := formatNote(n)

	if result["name"] != "notes/min" {
		t.Errorf("expected name 'notes/min', got '%v'", result["name"])
	}
	if _, ok := result["text"]; ok {
		t.Error("expected no text field for note without body")
	}
	if _, ok := result["create_time"]; ok {
		t.Error("expected no create_time field when empty")
	}
}

func TestFormatNote_Trashed(t *testing.T) {
	n := &keep.Note{
		Name:    "notes/trash1",
		Title:   "Trashed",
		Trashed: true,
	}

	result := formatNote(n)

	if result["trashed"] != true {
		t.Error("expected trashed to be true")
	}
}
