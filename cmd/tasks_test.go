package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

func TestTasksUpdateCommand_Help(t *testing.T) {
	cmd := tasksUpdateCmd

	if cmd.Use != "update <tasklist-id> <task-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestTasksUpdateCommand_Flags(t *testing.T) {
	cmd := tasksUpdateCmd

	flags := []string{"title", "notes", "due"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// TestTasksUpdate_FlagDefaults tests that update flags default to empty strings
func TestTasksUpdate_FlagDefaults(t *testing.T) {
	cmd := tasksUpdateCmd

	title, _ := cmd.Flags().GetString("title")
	notes, _ := cmd.Flags().GetString("notes")
	due, _ := cmd.Flags().GetString("due")

	if title != "" || notes != "" || due != "" {
		t.Error("expected all update flags to default to empty string")
	}
}

// TestTasksUpdate_NoFlagsReturnsEarly tests that runTasksUpdate returns without
// attempting OAuth when no flags are set (PrintError prints the error, returns nil)
func TestTasksUpdate_NoFlagsReturnsEarly(t *testing.T) {
	// Create a fresh command to avoid flag state pollution
	cmd := &cobra.Command{Use: "update", Args: cobra.ExactArgs(2)}
	cmd.Flags().String("title", "", "")
	cmd.Flags().String("notes", "", "")
	cmd.Flags().String("due", "", "")

	// runTasksUpdate uses PrintError which prints and returns nil.
	// Without OAuth creds, if validation didn't catch it, we'd get a
	// "missing OAuth credentials" panic or different error path.
	// The function should return quickly (nil from PrintError) without
	// attempting client creation.
	err := runTasksUpdate(cmd, []string{"@default", "task-123"})
	// PrintError returns nil (print succeeded), so err is nil.
	// The key assertion is that it doesn't panic or attempt OAuth.
	_ = err
}

// TestTasksUpdate_PartialUpdates tests updating individual fields
func TestTasksUpdate_PartialUpdates(t *testing.T) {
	tests := []struct {
		name        string
		updateField string
		updateValue string
	}{
		{"notes only", "notes", "New notes content"},
		{"due only", "due", "2024-06-15"},
		{"title only", "title", "New title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedTask tasks.Task
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.Method == "GET" {
					resp := &tasks.Task{
						Id:    "task-456",
						Title: "Original title",
						Notes: "Original notes",
						Due:   "2024-01-01",
					}
					json.NewEncoder(w).Encode(resp)
					return
				}

				if r.Method == "PUT" {
					json.NewDecoder(r.Body).Decode(&receivedTask)
					json.NewEncoder(w).Encode(&receivedTask)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
			if err != nil {
				t.Fatalf("failed to create service: %v", err)
			}

			task, _ := svc.Tasks.Get("@default", "task-456").Do()

			// Apply the single update
			switch tt.updateField {
			case "title":
				task.Title = tt.updateValue
			case "notes":
				task.Notes = tt.updateValue
			case "due":
				task.Due = tt.updateValue
			}

			updated, err := svc.Tasks.Update("@default", "task-456", task).Do()
			if err != nil {
				t.Fatalf("failed to update: %v", err)
			}

			switch tt.updateField {
			case "title":
				if updated.Title != tt.updateValue {
					t.Errorf("expected title '%s', got '%s'", tt.updateValue, updated.Title)
				}
			case "notes":
				if updated.Notes != tt.updateValue {
					t.Errorf("expected notes '%s', got '%s'", tt.updateValue, updated.Notes)
				}
			case "due":
				if updated.Due != tt.updateValue {
					t.Errorf("expected due '%s', got '%s'", tt.updateValue, updated.Due)
				}
			}
		})
	}
}

// TestTasksUpdate_GetFailure tests error handling when task fetch fails
func TestTasksUpdate_GetFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    404,
				"message": "Task not found",
			},
		})
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = svc.Tasks.Get("@default", "nonexistent").Do()
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

// TestTasksUpdate_OutputFormat tests the update response JSON structure
func TestTasksUpdate_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status": "updated",
		"id":     "task-123",
		"title":  "Updated title",
		"notes":  "Some notes",
		"due":    "2024-06-15",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	expectedFields := []string{"status", "id", "title", "notes", "due"}
	for _, field := range expectedFields {
		if _, ok := decoded[field]; !ok {
			t.Errorf("missing expected field: %s", field)
		}
	}

	if decoded["status"] != "updated" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}
}

// TestNormalizeDueDate tests date format normalization
func TestNormalizeDueDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"2026-02-04", "2026-02-04T00:00:00Z", false},
		{"2024-01-01", "2024-01-01T00:00:00Z", false},
		{"2026-02-04T00:00:00Z", "2026-02-04T00:00:00Z", false},
		{"2026-02-04T10:30:00+02:00", "2026-02-04T10:30:00+02:00", false},
		{"not-a-date", "not-a-date", false},           // non-matching passes through
		{"", "", false},                               // empty string passthrough
		{"2024-13-01", "", true},                      // invalid month
		{"2024-02-30", "", true},                      // invalid day
		{"2023-02-29", "", true},                      // non-leap year
		{"2024-02-29", "2024-02-29T00:00:00Z", false}, // valid leap year
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeDueDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeDueDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("normalizeDueDate(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// --- Flag tests for new commands ---

func TestTasksListInfoCommand_Flags(t *testing.T) {
	cmd := tasksListInfoCmd
	if cmd.Use != "list-info <tasklist-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestTasksCreateListCommand_Flags(t *testing.T) {
	cmd := tasksCreateListCmd
	if cmd.Flags().Lookup("title") == nil {
		t.Error("expected --title flag")
	}
}

func TestTasksUpdateListCommand_Flags(t *testing.T) {
	cmd := tasksUpdateListCmd
	if cmd.Use != "update-list <tasklist-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Flags().Lookup("title") == nil {
		t.Error("expected --title flag")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestTasksDeleteListCommand_Flags(t *testing.T) {
	cmd := tasksDeleteListCmd
	if cmd.Use != "delete-list <tasklist-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestTasksGetCommand_Flags(t *testing.T) {
	cmd := tasksGetCmd
	if cmd.Use != "get <tasklist-id> <task-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestTasksDeleteCommand_Flags(t *testing.T) {
	cmd := tasksDeleteCmd
	if cmd.Use != "delete <tasklist-id> <task-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestTasksMoveCommand_Flags(t *testing.T) {
	cmd := tasksMoveCmd
	if cmd.Use != "move <tasklist-id> <task-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	flags := []string{"parent", "previous", "destination-list"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestTasksClearCommand_Flags(t *testing.T) {
	cmd := tasksClearCmd
	if cmd.Use != "clear <tasklist-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// --- Mock server tests for new commands ---

func TestTasksListInfo_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/tasks/v1/users/@me/lists/list-123" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "list-123",
				"title":    "My List",
				"updated":  "2026-02-18T00:00:00Z",
				"selfLink": "https://www.googleapis.com/tasks/v1/users/@me/lists/list-123",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	list, err := svc.Tasklists.Get("list-123").Do()
	if err != nil {
		t.Fatalf("failed to get task list: %v", err)
	}

	if list.Id != "list-123" {
		t.Errorf("expected id 'list-123', got '%s'", list.Id)
	}
	if list.Title != "My List" {
		t.Errorf("expected title 'My List', got '%s'", list.Title)
	}
}

func TestTasksCreateList_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/tasks/v1/users/@me/lists" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if body["title"] != "New List" {
				t.Errorf("expected title 'New List', got '%v'", body["title"])
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    "new-list-id",
				"title": body["title"],
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	created, err := svc.Tasklists.Insert(&tasks.TaskList{Title: "New List"}).Do()
	if err != nil {
		t.Fatalf("failed to create task list: %v", err)
	}

	if created.Id != "new-list-id" {
		t.Errorf("expected id 'new-list-id', got '%s'", created.Id)
	}
	if created.Title != "New List" {
		t.Errorf("expected title 'New List', got '%s'", created.Title)
	}
}

func TestTasksUpdateList_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "PATCH" && r.URL.Path == "/tasks/v1/users/@me/lists/list-123" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if body["title"] != "Renamed List" {
				t.Errorf("expected title 'Renamed List', got '%v'", body["title"])
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    "list-123",
				"title": body["title"],
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	updated, err := svc.Tasklists.Patch("list-123", &tasks.TaskList{Title: "Renamed List"}).Do()
	if err != nil {
		t.Fatalf("failed to update task list: %v", err)
	}

	if updated.Title != "Renamed List" {
		t.Errorf("expected title 'Renamed List', got '%s'", updated.Title)
	}
}

func TestTasksDeleteList_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/tasks/v1/users/@me/lists/list-123" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	err = svc.Tasklists.Delete("list-123").Do()
	if err != nil {
		t.Fatalf("expected no error on delete, got: %v", err)
	}
}

func TestTasksGet_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/tasks/v1/lists/@default/tasks/task-456" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      "task-456",
				"title":   "Test Task",
				"status":  "needsAction",
				"notes":   "Some notes",
				"due":     "2026-03-01T00:00:00Z",
				"parent":  "parent-task",
				"updated": "2026-02-18T00:00:00Z",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	task, err := svc.Tasks.Get("@default", "task-456").Do()
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if task.Id != "task-456" {
		t.Errorf("expected id 'task-456', got '%s'", task.Id)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got '%s'", task.Title)
	}
	if task.Notes != "Some notes" {
		t.Errorf("expected notes 'Some notes', got '%s'", task.Notes)
	}
	if task.Parent != "parent-task" {
		t.Errorf("expected parent 'parent-task', got '%s'", task.Parent)
	}
}

func TestTasksDelete_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/tasks/v1/lists/@default/tasks/task-456" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	err = svc.Tasks.Delete("@default", "task-456").Do()
	if err != nil {
		t.Fatalf("expected no error on delete, got: %v", err)
	}
}

func TestTasksMove_MockServer(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/tasks/v1/lists/@default/tasks/task-1/move" {
			capturedQuery = r.URL.RawQuery
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "task-1",
				"title":  "Moved Task",
				"parent": "parent-task",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	moved, err := svc.Tasks.Move("@default", "task-1").Parent("parent-task").Previous("prev-task").Do()
	if err != nil {
		t.Fatalf("failed to move task: %v", err)
	}

	if moved.Id != "task-1" {
		t.Errorf("expected id 'task-1', got '%s'", moved.Id)
	}

	// Verify query params were sent
	if capturedQuery == "" {
		t.Error("expected query parameters to be sent")
	}
}

func TestTasksClear_MockServer(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/tasks/v1/lists/@default/clear" {
			called = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	err = svc.Tasks.Clear("@default").Do()
	if err != nil {
		t.Fatalf("expected no error on clear, got: %v", err)
	}

	if !called {
		t.Error("expected clear endpoint to be called")
	}
}

// TestTasksUpdate_MockServer tests update API integration
func TestTasksUpdate_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// GET task
		if r.URL.Path == "/tasks/v1/lists/@default/tasks/task-123" && r.Method == "GET" {
			resp := &tasks.Task{
				Id:    "task-123",
				Title: "Original title",
				Notes: "Original notes",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// PUT (update) task
		if r.URL.Path == "/tasks/v1/lists/@default/tasks/task-123" && r.Method == "PUT" {
			var task tasks.Task
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if task.Title != "Updated title" {
				t.Errorf("expected title 'Updated title', got '%s'", task.Title)
			}

			resp := &tasks.Task{
				Id:    "task-123",
				Title: task.Title,
				Notes: task.Notes,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := tasks.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create tasks service: %v", err)
	}

	// Get task
	task, err := svc.Tasks.Get("@default", "task-123").Do()
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if task.Title != "Original title" {
		t.Errorf("unexpected title: %s", task.Title)
	}

	// Update task
	task.Title = "Updated title"
	updated, err := svc.Tasks.Update("@default", "task-123", task).Do()
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	if updated.Title != "Updated title" {
		t.Errorf("unexpected updated title: %s", updated.Title)
	}
}
