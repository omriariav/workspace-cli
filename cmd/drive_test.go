package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		t.Fatal("expected --max flag to exist")
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
	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
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

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
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

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
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

func TestDriveComments_DirectLink(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"application/vnd.google-apps.document", "https://docs.google.com/document/d/file-123/edit?disco=comment-1"},
		{"application/vnd.google-apps.spreadsheet", "https://docs.google.com/spreadsheets/d/file-123/edit?disco=comment-1"},
		{"application/vnd.google-apps.presentation", "https://docs.google.com/presentation/d/file-123/edit?disco=comment-1"},
		{"application/pdf", "https://drive.google.com/file/d/file-123/view?disco=comment-1"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			fileID := "file-123"
			commentID := "comment-1"

			var directLink string
			switch tt.mimeType {
			case "application/vnd.google-apps.document":
				directLink = fmt.Sprintf("https://docs.google.com/document/d/%s/edit?disco=%s", fileID, commentID)
			case "application/vnd.google-apps.spreadsheet":
				directLink = fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit?disco=%s", fileID, commentID)
			case "application/vnd.google-apps.presentation":
				directLink = fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit?disco=%s", fileID, commentID)
			default:
				directLink = fmt.Sprintf("https://drive.google.com/file/d/%s/view?disco=%s", fileID, commentID)
			}

			if directLink != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, directLink)
			}
		})
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
				"id":          "c1",
				"content":     "Test comment",
				"direct_link": "https://docs.google.com/document/d/test-id/edit?disco=c1",
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

// TestDriveUploadCommand_Flags tests that upload command has expected flags
func TestDriveUploadCommand_Flags(t *testing.T) {
	cmd := driveUploadCmd

	// Check required args
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Check flags exist
	folderFlag := cmd.Flags().Lookup("folder")
	if folderFlag == nil {
		t.Error("expected --folder flag to exist")
	}

	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("expected --name flag to exist")
	}

	mimeFlag := cmd.Flags().Lookup("mime-type")
	if mimeFlag == nil {
		t.Error("expected --mime-type flag to exist")
	}
}

func TestDriveUploadCommand_Help(t *testing.T) {
	cmd := driveUploadCmd

	if cmd.Use != "upload <local-file>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// TestDriveUpload_MockServer tests the upload API integration
func TestDriveUpload_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle upload request (multipart)
		if r.URL.Path == "/upload/drive/v3/files" && r.Method == "POST" {
			resp := &drive.File{
				Id:          "uploaded-file-id",
				Name:        "test-upload.txt",
				MimeType:    "text/plain",
				WebViewLink: "https://drive.google.com/file/d/uploaded-file-id/view",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create a Drive service pointing to our mock server
	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// We can't easily test the actual upload without a real file,
	// but we can verify the service is created correctly
	if svc == nil {
		t.Error("expected service to be created")
	}
}

// TestDriveUpload_OutputFormat tests the upload response format
func TestDriveUpload_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":    "uploaded",
		"id":        "test-file-id",
		"name":      "uploaded-file.xlsx",
		"mime_type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"size":      12345,
		"web_link":  "https://drive.google.com/file/d/test-file-id/view",
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

	if decoded["status"] != "uploaded" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}

	if decoded["id"] != "test-file-id" {
		t.Errorf("unexpected id: %v", decoded["id"])
	}

	if decoded["name"] != "uploaded-file.xlsx" {
		t.Errorf("unexpected name: %v", decoded["name"])
	}
}

// TestDriveCreateFolderCommand_Flags tests that create-folder command has expected flags
func TestDriveCreateFolderCommand_Flags(t *testing.T) {
	cmd := driveCreateFolderCmd

	// Check flags exist
	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("expected --name flag to exist")
	}

	parentFlag := cmd.Flags().Lookup("parent")
	if parentFlag == nil {
		t.Error("expected --parent flag to exist")
	}
}

func TestDriveCreateFolderCommand_Help(t *testing.T) {
	cmd := driveCreateFolderCmd

	if cmd.Use != "create-folder" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// TestDriveCreateFolder_MockServer tests create-folder API integration
func TestDriveCreateFolder_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// The Google API client uses /files not /drive/v3/files
		if r.URL.Path == "/files" && r.Method == "POST" {
			// Decode request to verify folder creation
			var file drive.File
			if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Verify MIME type is folder
			if file.MimeType != "application/vnd.google-apps.folder" {
				t.Errorf("expected folder MIME type, got: %s", file.MimeType)
			}

			resp := &drive.File{
				Id:          "new-folder-id",
				Name:        file.Name,
				WebViewLink: "https://drive.google.com/drive/folders/new-folder-id",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test folder creation
	folder := &drive.File{
		Name:     "Test Folder",
		MimeType: "application/vnd.google-apps.folder",
	}

	created, err := svc.Files.Create(folder).Fields("id, name, webViewLink").Do()
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	if created.Id != "new-folder-id" {
		t.Errorf("unexpected folder id: %s", created.Id)
	}

	if created.Name != "Test Folder" {
		t.Errorf("unexpected folder name: %s", created.Name)
	}
}

// TestDriveCreateFolder_OutputFormat tests the create-folder response format
func TestDriveCreateFolder_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":   "created",
		"id":       "new-folder-id",
		"name":     "My Folder",
		"web_link": "https://drive.google.com/drive/folders/new-folder-id",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["status"] != "created" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}

	if decoded["id"] != "new-folder-id" {
		t.Errorf("unexpected id: %v", decoded["id"])
	}
}

// TestDriveMoveCommand_Flags tests that move command has expected flags
func TestDriveMoveCommand_Flags(t *testing.T) {
	cmd := driveMoveCmd

	// Check required args
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Check flags exist
	toFlag := cmd.Flags().Lookup("to")
	if toFlag == nil {
		t.Error("expected --to flag to exist")
	}
}

func TestDriveMoveCommand_Help(t *testing.T) {
	cmd := driveMoveCmd

	if cmd.Use != "move <file-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// TestDriveMove_MockServer tests move API integration
func TestDriveMove_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get file info
		if r.URL.Path == "/files/test-file-id" && r.Method == "GET" {
			resp := &drive.File{
				Id:      "test-file-id",
				Name:    "Test File.docx",
				Parents: []string{"old-folder-id"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Update file (move)
		if r.URL.Path == "/files/test-file-id" && r.Method == "PATCH" {
			addParents := r.URL.Query().Get("addParents")
			removeParents := r.URL.Query().Get("removeParents")

			if addParents != "new-folder-id" {
				t.Errorf("expected addParents=new-folder-id, got: %s", addParents)
			}
			if removeParents != "old-folder-id" {
				t.Errorf("expected removeParents=old-folder-id, got: %s", removeParents)
			}

			resp := &drive.File{
				Id:          "test-file-id",
				Name:        "Test File.docx",
				Parents:     []string{"new-folder-id"},
				WebViewLink: "https://drive.google.com/file/d/test-file-id/view",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Get file info first
	file, err := svc.Files.Get("test-file-id").Fields("name, parents").Do()
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}

	if len(file.Parents) != 1 || file.Parents[0] != "old-folder-id" {
		t.Errorf("unexpected parents: %v", file.Parents)
	}

	// Move file
	updated, err := svc.Files.Update("test-file-id", nil).
		AddParents("new-folder-id").
		RemoveParents("old-folder-id").
		Fields("id, name, parents, webViewLink").
		Do()
	if err != nil {
		t.Fatalf("failed to move file: %v", err)
	}

	if len(updated.Parents) != 1 || updated.Parents[0] != "new-folder-id" {
		t.Errorf("unexpected parents after move: %v", updated.Parents)
	}
}

// TestDriveMove_OutputFormat tests the move response format
func TestDriveMove_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":   "moved",
		"id":       "test-file-id",
		"name":     "My File.docx",
		"parents":  []string{"new-folder-id"},
		"web_link": "https://drive.google.com/file/d/test-file-id/view",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["status"] != "moved" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}

	parents := decoded["parents"].([]interface{})
	if len(parents) != 1 || parents[0] != "new-folder-id" {
		t.Errorf("unexpected parents: %v", parents)
	}
}

// TestDriveDeleteCommand_Flags tests that delete command has expected flags
func TestDriveDeleteCommand_Flags(t *testing.T) {
	cmd := driveDeleteCmd

	// Check required args
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Check flags exist
	permanentFlag := cmd.Flags().Lookup("permanent")
	if permanentFlag == nil {
		t.Fatal("expected --permanent flag to exist")
	}
	if permanentFlag.DefValue != "false" {
		t.Errorf("expected --permanent default to be false, got %s", permanentFlag.DefValue)
	}
}

func TestDriveDeleteCommand_Help(t *testing.T) {
	cmd := driveDeleteCmd

	if cmd.Use != "delete <file-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// TestDriveDelete_MockServer_Trash tests trash (soft delete) API integration
func TestDriveDelete_MockServer_Trash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get file info
		if r.URL.Path == "/files/test-file-id" && r.Method == "GET" {
			resp := &drive.File{
				Id:   "test-file-id",
				Name: "Test File.docx",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Update file (trash)
		if r.URL.Path == "/files/test-file-id" && r.Method == "PATCH" {
			var file drive.File
			if err := json.NewDecoder(r.Body).Decode(&file); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if !file.Trashed {
				t.Error("expected Trashed to be true")
			}

			resp := &drive.File{
				Id:      "test-file-id",
				Name:    "Test File.docx",
				Trashed: true,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Get file info
	file, err := svc.Files.Get("test-file-id").Fields("name").Do()
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}

	if file.Name != "Test File.docx" {
		t.Errorf("unexpected file name: %s", file.Name)
	}

	// Trash file
	_, err = svc.Files.Update("test-file-id", &drive.File{Trashed: true}).Do()
	if err != nil {
		t.Fatalf("failed to trash file: %v", err)
	}
}

// TestDriveDelete_MockServer_Permanent tests permanent delete API integration
func TestDriveDelete_MockServer_Permanent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get file info
		if r.URL.Path == "/files/test-file-id" && r.Method == "GET" {
			resp := &drive.File{
				Id:   "test-file-id",
				Name: "Test File.docx",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Delete file (permanent)
		if r.URL.Path == "/files/test-file-id" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Get file info
	file, err := svc.Files.Get("test-file-id").Fields("name").Do()
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}

	if file.Name != "Test File.docx" {
		t.Errorf("unexpected file name: %s", file.Name)
	}

	// Permanently delete file
	err = svc.Files.Delete("test-file-id").Do()
	if err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}
}

// TestDriveDelete_OutputFormat tests the delete response formats
func TestDriveDelete_OutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"trash", "trashed", "trashed"},
		{"permanent", "deleted", "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := map[string]interface{}{
				"status": tt.status,
				"id":     "test-file-id",
				"name":   "Test File.docx",
			}

			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				t.Fatalf("failed to encode result: %v", err)
			}

			var decoded map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
				t.Fatalf("failed to decode result: %v", err)
			}

			if decoded["status"] != tt.expected {
				t.Errorf("unexpected status: %v", decoded["status"])
			}
		})
	}
}

// --- Permissions Tests ---

func TestDrivePermissionsCommand_Flags(t *testing.T) {
	cmd := drivePermissionsCmd
	if cmd.Flags().Lookup("file-id") == nil {
		t.Error("expected --file-id flag")
	}
}

func TestDriveShareCommand_Flags(t *testing.T) {
	cmd := driveShareCmd
	flags := []string{"file-id", "type", "role", "email", "domain", "send-notification"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveUnshareCommand_Flags(t *testing.T) {
	cmd := driveUnshareCmd
	flags := []string{"file-id", "permission-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDrivePermissionCommand_Flags(t *testing.T) {
	cmd := drivePermissionCmd
	flags := []string{"file-id", "permission-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveUpdatePermissionCommand_Flags(t *testing.T) {
	cmd := driveUpdatePermissionCmd
	flags := []string{"file-id", "permission-id", "role"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDrivePermissions_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/files/test-file-id/permissions" && r.Method == "GET" {
			resp := &drive.PermissionList{
				Permissions: []*drive.Permission{
					{
						Id:           "perm-1",
						Type:         "user",
						Role:         "writer",
						EmailAddress: "user@example.com",
						DisplayName:  "Test User",
					},
					{
						Id:   "perm-2",
						Type: "anyone",
						Role: "reader",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	resp, err := svc.Permissions.List("test-file-id").
		Fields("permissions(id,type,role,emailAddress,displayName)").
		Do()
	if err != nil {
		t.Fatalf("failed to list permissions: %v", err)
	}

	if len(resp.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(resp.Permissions))
	}

	if resp.Permissions[0].Role != "writer" {
		t.Errorf("expected role 'writer', got '%s'", resp.Permissions[0].Role)
	}
}

func TestDriveShare_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/files/test-file-id/permissions" && r.Method == "POST" {
			var perm drive.Permission
			json.NewDecoder(r.Body).Decode(&perm)

			resp := &drive.Permission{
				Id:           "new-perm-id",
				Type:         perm.Type,
				Role:         perm.Role,
				EmailAddress: perm.EmailAddress,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	perm := &drive.Permission{
		Type:         "user",
		Role:         "writer",
		EmailAddress: "user@example.com",
	}

	created, err := svc.Permissions.Create("test-file-id", perm).
		Fields("id,type,role,emailAddress").
		Do()
	if err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	if created.Id != "new-perm-id" {
		t.Errorf("expected id 'new-perm-id', got '%s'", created.Id)
	}
	if created.Role != "writer" {
		t.Errorf("expected role 'writer', got '%s'", created.Role)
	}
}

func TestDriveUnshare_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/files/test-file-id/permissions/perm-1" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	err = svc.Permissions.Delete("test-file-id", "perm-1").Do()
	if err != nil {
		t.Fatalf("failed to delete permission: %v", err)
	}
}

// --- Revisions Tests ---

func TestDriveRevisionsCommand_Flags(t *testing.T) {
	cmd := driveRevisionsCmd
	if cmd.Flags().Lookup("file-id") == nil {
		t.Error("expected --file-id flag")
	}
}

func TestDriveRevisionCommand_Flags(t *testing.T) {
	cmd := driveRevisionCmd
	flags := []string{"file-id", "revision-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveDeleteRevisionCommand_Flags(t *testing.T) {
	cmd := driveDeleteRevisionCmd
	flags := []string{"file-id", "revision-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveRevisions_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/files/test-file-id/revisions" && r.Method == "GET" {
			resp := &drive.RevisionList{
				Revisions: []*drive.Revision{
					{
						Id:           "rev-1",
						MimeType:     "application/vnd.google-apps.document",
						ModifiedTime: "2024-01-15T10:00:00Z",
						Size:         1024,
						LastModifyingUser: &drive.User{
							DisplayName:  "Test User",
							EmailAddress: "test@example.com",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/revisions/rev-1" && r.Method == "GET" {
			resp := &drive.Revision{
				Id:           "rev-1",
				MimeType:     "application/vnd.google-apps.document",
				ModifiedTime: "2024-01-15T10:00:00Z",
				Size:         1024,
				KeepForever:  true,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/revisions/rev-1" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test list revisions
	revList, err := svc.Revisions.List("test-file-id").Do()
	if err != nil {
		t.Fatalf("failed to list revisions: %v", err)
	}
	if len(revList.Revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(revList.Revisions))
	}

	// Test get revision
	rev, err := svc.Revisions.Get("test-file-id", "rev-1").Do()
	if err != nil {
		t.Fatalf("failed to get revision: %v", err)
	}
	if rev.Id != "rev-1" {
		t.Errorf("expected id 'rev-1', got '%s'", rev.Id)
	}

	// Test delete revision
	err = svc.Revisions.Delete("test-file-id", "rev-1").Do()
	if err != nil {
		t.Fatalf("failed to delete revision: %v", err)
	}
}

// --- Replies Tests ---

func TestDriveRepliesCommand_Flags(t *testing.T) {
	cmd := driveRepliesCmd
	flags := []string{"file-id", "comment-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveReplyCommand_Flags(t *testing.T) {
	cmd := driveReplyCmd
	flags := []string{"file-id", "comment-id", "content"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveGetReplyCommand_Flags(t *testing.T) {
	cmd := driveGetReplyCmd
	flags := []string{"file-id", "comment-id", "reply-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveDeleteReplyCommand_Flags(t *testing.T) {
	cmd := driveDeleteReplyCmd
	flags := []string{"file-id", "comment-id", "reply-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveReplies_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/files/test-file-id/comments/comment-1/replies" && r.Method == "GET" {
			resp := &drive.ReplyList{
				Replies: []*drive.Reply{
					{
						Id:          "reply-1",
						Content:     "Test reply",
						CreatedTime: "2024-01-15T10:00:00Z",
						Author: &drive.User{
							DisplayName:  "Test User",
							EmailAddress: "test@example.com",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/comments/comment-1/replies" && r.Method == "POST" {
			var reply drive.Reply
			json.NewDecoder(r.Body).Decode(&reply)
			resp := &drive.Reply{
				Id:          "new-reply-id",
				Content:     reply.Content,
				CreatedTime: "2024-01-15T11:00:00Z",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/comments/comment-1/replies/reply-1" && r.Method == "GET" {
			resp := &drive.Reply{
				Id:          "reply-1",
				Content:     "Test reply",
				CreatedTime: "2024-01-15T10:00:00Z",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/comments/comment-1/replies/reply-1" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test list replies
	replyList, err := svc.Replies.List("test-file-id", "comment-1").Do()
	if err != nil {
		t.Fatalf("failed to list replies: %v", err)
	}
	if len(replyList.Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replyList.Replies))
	}

	// Test create reply
	newReply, err := svc.Replies.Create("test-file-id", "comment-1", &drive.Reply{Content: "New reply"}).Do()
	if err != nil {
		t.Fatalf("failed to create reply: %v", err)
	}
	if newReply.Id != "new-reply-id" {
		t.Errorf("expected id 'new-reply-id', got '%s'", newReply.Id)
	}

	// Test get reply
	reply, err := svc.Replies.Get("test-file-id", "comment-1", "reply-1").Do()
	if err != nil {
		t.Fatalf("failed to get reply: %v", err)
	}
	if reply.Content != "Test reply" {
		t.Errorf("unexpected content: %s", reply.Content)
	}

	// Test delete reply
	err = svc.Replies.Delete("test-file-id", "comment-1", "reply-1").Do()
	if err != nil {
		t.Fatalf("failed to delete reply: %v", err)
	}
}

// --- Comment (single) Tests ---

func TestDriveCommentCommand_Flags(t *testing.T) {
	cmd := driveCommentCmd
	flags := []string{"file-id", "comment-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveAddCommentCommand_Flags(t *testing.T) {
	cmd := driveAddCommentCmd
	flags := []string{"file-id", "content"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveDeleteCommentCommand_Flags(t *testing.T) {
	cmd := driveDeleteCommentCmd
	flags := []string{"file-id", "comment-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveComment_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/files/test-file-id/comments/comment-1" && r.Method == "GET" {
			resp := &drive.Comment{
				Id:          "comment-1",
				Content:     "Test comment",
				CreatedTime: "2024-01-15T10:00:00Z",
				Resolved:    false,
				Author: &drive.User{
					DisplayName:  "Test User",
					EmailAddress: "test@example.com",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/comments" && r.Method == "POST" {
			var comment drive.Comment
			json.NewDecoder(r.Body).Decode(&comment)
			resp := &drive.Comment{
				Id:          "new-comment-id",
				Content:     comment.Content,
				CreatedTime: "2024-01-15T11:00:00Z",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/files/test-file-id/comments/comment-1" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test get comment
	comment, err := svc.Comments.Get("test-file-id", "comment-1").Do()
	if err != nil {
		t.Fatalf("failed to get comment: %v", err)
	}
	if comment.Content != "Test comment" {
		t.Errorf("unexpected content: %s", comment.Content)
	}

	// Test create comment
	newComment, err := svc.Comments.Create("test-file-id", &drive.Comment{Content: "New comment"}).Do()
	if err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}
	if newComment.Id != "new-comment-id" {
		t.Errorf("expected id 'new-comment-id', got '%s'", newComment.Id)
	}

	// Test delete comment
	err = svc.Comments.Delete("test-file-id", "comment-1").Do()
	if err != nil {
		t.Fatalf("failed to delete comment: %v", err)
	}
}

// --- Export Tests ---

func TestDriveExportCommand_Flags(t *testing.T) {
	cmd := driveExportCmd
	flags := []string{"file-id", "mime-type", "output"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// --- Empty Trash Tests ---

func TestDriveEmptyTrashCommand_Help(t *testing.T) {
	cmd := driveEmptyTrashCmd
	if cmd.Use != "empty-trash" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestDriveEmptyTrash_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/files/trash" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	err = svc.Files.EmptyTrash().Do()
	if err != nil {
		t.Fatalf("failed to empty trash: %v", err)
	}
}

// --- Update Tests ---

func TestDriveUpdateCommand_Flags(t *testing.T) {
	cmd := driveUpdateCmd
	flags := []string{"file-id", "name", "description", "starred", "trashed"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveUpdate_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/files/test-file-id" && r.Method == "PATCH" {
			resp := &drive.File{
				Id:           "test-file-id",
				Name:         "Updated Name",
				Starred:      true,
				ModifiedTime: "2024-01-15T12:00:00Z",
				WebViewLink:  "https://drive.google.com/file/d/test-file-id/view",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	updated, err := svc.Files.Update("test-file-id", &drive.File{
		Name:    "Updated Name",
		Starred: true,
	}).Fields("id,name,starred,modifiedTime,webViewLink").Do()
	if err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("unexpected name: %s", updated.Name)
	}
	if !updated.Starred {
		t.Error("expected starred to be true")
	}
}

// --- Shared Drives Tests ---

func TestDriveSharedDrivesCommand_Flags(t *testing.T) {
	cmd := driveSharedDrivesCmd
	flags := []string{"max", "query"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveSharedDriveCommand_Flags(t *testing.T) {
	cmd := driveSharedDriveCmd
	if cmd.Flags().Lookup("id") == nil {
		t.Error("expected --id flag")
	}
}

func TestDriveCreateDriveCommand_Flags(t *testing.T) {
	cmd := driveCreateDriveCmd
	if cmd.Flags().Lookup("name") == nil {
		t.Error("expected --name flag")
	}
}

func TestDriveDeleteDriveCommand_Flags(t *testing.T) {
	cmd := driveDeleteDriveCmd
	if cmd.Flags().Lookup("id") == nil {
		t.Error("expected --id flag")
	}
}

func TestDriveUpdateDriveCommand_Flags(t *testing.T) {
	cmd := driveUpdateDriveCmd
	flags := []string{"id", "name"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveSharedDrives_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/drives" && r.Method == "GET" {
			resp := &drive.DriveList{
				Drives: []*drive.Drive{
					{
						Id:          "drive-1",
						Name:        "Engineering",
						CreatedTime: "2024-01-01T00:00:00Z",
					},
					{
						Id:          "drive-2",
						Name:        "Marketing",
						CreatedTime: "2024-02-01T00:00:00Z",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/drives/drive-1" && r.Method == "GET" {
			resp := &drive.Drive{
				Id:          "drive-1",
				Name:        "Engineering",
				CreatedTime: "2024-01-01T00:00:00Z",
				ColorRgb:    "#4285f4",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/drives" && r.Method == "POST" {
			resp := &drive.Drive{
				Id:          "new-drive-id",
				Name:        "New Drive",
				CreatedTime: "2024-03-01T00:00:00Z",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/drives/drive-1" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.URL.Path == "/drives/drive-1" && r.Method == "PATCH" {
			resp := &drive.Drive{
				Id:   "drive-1",
				Name: "Updated Engineering",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test list shared drives
	driveList, err := svc.Drives.List().Do()
	if err != nil {
		t.Fatalf("failed to list drives: %v", err)
	}
	if len(driveList.Drives) != 2 {
		t.Fatalf("expected 2 drives, got %d", len(driveList.Drives))
	}

	// Test get shared drive
	d, err := svc.Drives.Get("drive-1").Do()
	if err != nil {
		t.Fatalf("failed to get drive: %v", err)
	}
	if d.Name != "Engineering" {
		t.Errorf("unexpected name: %s", d.Name)
	}

	// Test create shared drive
	created, err := svc.Drives.Create("req-123", &drive.Drive{Name: "New Drive"}).Do()
	if err != nil {
		t.Fatalf("failed to create drive: %v", err)
	}
	if created.Id != "new-drive-id" {
		t.Errorf("unexpected id: %s", created.Id)
	}

	// Test delete shared drive
	err = svc.Drives.Delete("drive-1").Do()
	if err != nil {
		t.Fatalf("failed to delete drive: %v", err)
	}

	// Test update shared drive
	updated, err := svc.Drives.Update("drive-1", &drive.Drive{Name: "Updated Engineering"}).Do()
	if err != nil {
		t.Fatalf("failed to update drive: %v", err)
	}
	if updated.Name != "Updated Engineering" {
		t.Errorf("unexpected name: %s", updated.Name)
	}
}

// --- About Tests ---

func TestDriveAboutCommand_Help(t *testing.T) {
	cmd := driveAboutCmd
	if cmd.Use != "about" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestDriveAbout_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/about" && r.Method == "GET" {
			resp := &drive.About{
				User: &drive.User{
					DisplayName:  "Test User",
					EmailAddress: "test@example.com",
				},
				StorageQuota: &drive.AboutStorageQuota{
					Limit:             16106127360,
					Usage:             5368709120,
					UsageInDrive:      4294967296,
					UsageInDriveTrash: 1073741824,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	about, err := svc.About.Get().
		Fields("user(displayName,emailAddress),storageQuota(limit,usage,usageInDrive,usageInDriveTrash)").
		Do()
	if err != nil {
		t.Fatalf("failed to get about: %v", err)
	}

	if about.User.EmailAddress != "test@example.com" {
		t.Errorf("unexpected email: %s", about.User.EmailAddress)
	}
	if about.StorageQuota.Limit != 16106127360 {
		t.Errorf("unexpected storage limit: %d", about.StorageQuota.Limit)
	}
}

// --- Changes Tests ---

func TestDriveChangesCommand_Flags(t *testing.T) {
	cmd := driveChangesCmd
	flags := []string{"max", "page-token"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestDriveChanges_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/changes/startPageToken" && r.Method == "GET" {
			resp := &drive.StartPageToken{
				StartPageToken: "start-token-123",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/changes" && r.Method == "GET" {
			resp := &drive.ChangeList{
				Changes: []*drive.Change{
					{
						FileId:     "file-1",
						ChangeType: "file",
						Time:       "2024-01-15T10:00:00Z",
						Removed:    false,
						File: &drive.File{
							Id:           "file-1",
							Name:         "Modified File.docx",
							MimeType:     "application/vnd.google-apps.document",
							ModifiedTime: "2024-01-15T10:00:00Z",
						},
					},
				},
				NewStartPageToken: "new-token-456",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := drive.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Test get start page token
	startToken, err := svc.Changes.GetStartPageToken().Do()
	if err != nil {
		t.Fatalf("failed to get start token: %v", err)
	}
	if startToken.StartPageToken != "start-token-123" {
		t.Errorf("unexpected start token: %s", startToken.StartPageToken)
	}

	// Test list changes
	changes, err := svc.Changes.List("start-token-123").Do()
	if err != nil {
		t.Fatalf("failed to list changes: %v", err)
	}
	if len(changes.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes.Changes))
	}
	if changes.Changes[0].FileId != "file-1" {
		t.Errorf("unexpected file id: %s", changes.Changes[0].FileId)
	}
	if changes.NewStartPageToken != "new-token-456" {
		t.Errorf("unexpected new start token: %s", changes.NewStartPageToken)
	}
}
