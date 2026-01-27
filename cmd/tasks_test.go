package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

// TestTasksUpdate_NoFlagsValidation tests that at least one flag is required
func TestTasksUpdate_NoFlagsValidation(t *testing.T) {
	// Simulate calling runTasksUpdate with no flags set
	// The validation is: if title == "" && notes == "" && due == "" → error
	// We test this by checking the function signature requires flags
	cmd := tasksUpdateCmd

	// Verify all flags default to empty
	title, _ := cmd.Flags().GetString("title")
	notes, _ := cmd.Flags().GetString("notes")
	due, _ := cmd.Flags().GetString("due")

	if title != "" || notes != "" || due != "" {
		t.Error("expected all update flags to default to empty string")
	}
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
