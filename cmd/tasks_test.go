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
