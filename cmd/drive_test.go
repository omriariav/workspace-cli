package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestDriveCommentsCommand_Flags(t *testing.T) {
	// Test that the command has the expected flags
	cmd := driveCommentsCmd

	// Check required args
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Check flags exist
	maxFlag := cmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Error("expected --max flag to exist")
	}
	if maxFlag.DefValue != "100" {
		t.Errorf("expected --max default to be 100, got %s", maxFlag.DefValue)
	}

	resolvedFlag := cmd.Flags().Lookup("include-resolved")
	if resolvedFlag == nil {
		t.Error("expected --include-resolved flag to exist")
	}

	deletedFlag := cmd.Flags().Lookup("include-deleted")
	if deletedFlag == nil {
		t.Error("expected --include-deleted flag to exist")
	}
}

func TestDriveCommentsCommand_Help(t *testing.T) {
	cmd := driveCommentsCmd

	if cmd.Use != "comments <file-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// mockDriveServer creates a test server that mocks Drive API responses
func mockDriveServer(t *testing.T, fileResp *drive.File, commentsResp *drive.CommentList) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// The Google API client uses paths like /files/ID, not /drive/v3/files/ID
		switch {
		case r.URL.Path == "/files/test-file-id" && r.Method == "GET":
			// File metadata request
			if fileResp != nil {
				json.NewEncoder(w).Encode(fileResp)
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    404,
						"message": "File not found",
					},
				})
			}
		case r.URL.Path == "/files/test-file-id/comments" && r.Method == "GET":
			// Comments list request
			if commentsResp != nil {
				json.NewEncoder(w).Encode(commentsResp)
			} else {
				json.NewEncoder(w).Encode(&drive.CommentList{Comments: []*drive.Comment{}})
			}
		default:
			t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestDriveComments_ParseResponse(t *testing.T) {
	// Test that we can create a Drive service with a mock server
	server := mockDriveServer(t,
		&drive.File{
			Id:       "test-file-id",
			Name:     "Test Document",
			MimeType: "application/vnd.google-apps.document",
		},
		&drive.CommentList{
			Comments: []*drive.Comment{
				{
					Id:           "comment-1",
					Content:      "This is a test comment",
					CreatedTime:  "2024-01-15T10:00:00Z",
					ModifiedTime: "2024-01-15T10:00:00Z",
					Resolved:     false,
					Author: &drive.User{
						DisplayName:  "Test User",
						EmailAddress: "test@example.com",
					},
					QuotedFileContent: &drive.CommentQuotedFileContent{
						Value: "quoted text",
					},
					Replies: []*drive.Reply{
						{
							Id:          "reply-1",
							Content:     "This is a reply",
							CreatedTime: "2024-01-15T11:00:00Z",
							Author: &drive.User{
								DisplayName:  "Reply User",
								EmailAddress: "reply@example.com",
							},
						},
					},
				},
			},
		},
	)
	defer server.Close()

	// Create a Drive service pointing to our mock server
	svc, err := drive.NewService(nil, option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test fetching file info
	file, err := svc.Files.Get("test-file-id").Fields("name, mimeType").Do()
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}

	if file.Name != "Test Document" {
		t.Errorf("expected file name 'Test Document', got '%s'", file.Name)
	}

	// Test fetching comments
	comments, err := svc.Comments.List("test-file-id").
		Fields("comments(id, content, author, createdTime, modifiedTime, resolved, quotedFileContent, replies)").
		Do()
	if err != nil {
		t.Fatalf("failed to list comments: %v", err)
	}

	if len(comments.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments.Comments))
	}

	comment := comments.Comments[0]
	if comment.Content != "This is a test comment" {
		t.Errorf("unexpected comment content: %s", comment.Content)
	}

	if comment.Author.EmailAddress != "test@example.com" {
		t.Errorf("unexpected author email: %s", comment.Author.EmailAddress)
	}

	if comment.QuotedFileContent.Value != "quoted text" {
		t.Errorf("unexpected quoted text: %s", comment.QuotedFileContent.Value)
	}

	if len(comment.Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(comment.Replies))
	}

	if comment.Replies[0].Content != "This is a reply" {
		t.Errorf("unexpected reply content: %s", comment.Replies[0].Content)
	}
}

func TestDriveComments_FilterResolved(t *testing.T) {
	// Test that resolved comments are filtered by default
	server := mockDriveServer(t,
		&drive.File{
			Id:       "test-file-id",
			Name:     "Test Document",
			MimeType: "application/vnd.google-apps.document",
		},
		&drive.CommentList{
			Comments: []*drive.Comment{
				{
					Id:       "comment-1",
					Content:  "Open comment",
					Resolved: false,
				},
				{
					Id:       "comment-2",
					Content:  "Resolved comment",
					Resolved: true,
				},
			},
		},
	)
	defer server.Close()

	svc, err := drive.NewService(nil, option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	comments, err := svc.Comments.List("test-file-id").Do()
	if err != nil {
		t.Fatalf("failed to list comments: %v", err)
	}

	// Simulate the filtering logic from runDriveComments
	includeResolved := false
	filteredCount := 0
	for _, comment := range comments.Comments {
		if comment.Resolved && !includeResolved {
			continue
		}
		filteredCount++
	}

	if filteredCount != 1 {
		t.Errorf("expected 1 comment after filtering, got %d", filteredCount)
	}

	// Now test with includeResolved = true
	includeResolved = true
	filteredCount = 0
	for _, comment := range comments.Comments {
		if comment.Resolved && !includeResolved {
			continue
		}
		filteredCount++
	}

	if filteredCount != 2 {
		t.Errorf("expected 2 comments with includeResolved, got %d", filteredCount)
	}
}

func TestDriveComments_EmptyResponse(t *testing.T) {
	server := mockDriveServer(t,
		&drive.File{
			Id:       "test-file-id",
			Name:     "Empty Doc",
			MimeType: "application/vnd.google-apps.document",
		},
		&drive.CommentList{
			Comments: []*drive.Comment{},
		},
	)
	defer server.Close()

	svc, err := drive.NewService(nil, option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	comments, err := svc.Comments.List("test-file-id").Do()
	if err != nil {
		t.Fatalf("failed to list comments: %v", err)
	}

	if len(comments.Comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments.Comments))
	}
}

func TestDriveComments_OutputFormat(t *testing.T) {
	// Test that output is properly structured JSON
	result := map[string]interface{}{
		"file_id":   "test-id",
		"file_name": "Test Doc",
		"mime_type": "application/vnd.google-apps.document",
		"comments": []map[string]interface{}{
			{
				"id":      "c1",
				"content": "Test comment",
				"author": map[string]interface{}{
					"name":  "Test User",
					"email": "test@example.com",
				},
				"resolved": false,
			},
		},
		"count": 1,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	// Verify we can decode it back
	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["file_id"] != "test-id" {
		t.Errorf("unexpected file_id: %v", decoded["file_id"])
	}

	if decoded["count"].(float64) != 1 {
		t.Errorf("unexpected count: %v", decoded["count"])
	}

	comments := decoded["comments"].([]interface{})
	if len(comments) != 1 {
		t.Errorf("unexpected comments count: %d", len(comments))
	}
}
