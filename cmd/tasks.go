package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/tasks/v1"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Manage Google Tasks",
	Long:  "Commands for interacting with Google Tasks.",
}

var tasksListsCmd = &cobra.Command{
	Use:   "lists",
	Short: "List task lists",
	Long:  "Lists all your task lists.",
	RunE:  runTasksLists,
}

var tasksListCmd = &cobra.Command{
	Use:   "list <tasklist-id>",
	Short: "List tasks in a task list",
	Long:  "Lists all tasks in a specific task list.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTasksList,
}

var tasksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a task",
	Long:  "Creates a new task in a task list.",
	RunE:  runTasksCreate,
}

var tasksCompleteCmd = &cobra.Command{
	Use:   "complete <tasklist-id> <task-id>",
	Short: "Mark a task as completed",
	Long:  "Marks a specific task as completed.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTasksComplete,
}

func init() {
	rootCmd.AddCommand(tasksCmd)
	tasksCmd.AddCommand(tasksListsCmd)
	tasksCmd.AddCommand(tasksListCmd)
	tasksCmd.AddCommand(tasksCreateCmd)
	tasksCmd.AddCommand(tasksCompleteCmd)

	// List flags
	tasksListCmd.Flags().Bool("show-completed", false, "Include completed tasks")
	tasksListCmd.Flags().Int64("max", 100, "Maximum number of tasks")

	// Create flags
	tasksCreateCmd.Flags().String("tasklist", "@default", "Task list ID (default: @default)")
	tasksCreateCmd.Flags().String("title", "", "Task title (required)")
	tasksCreateCmd.Flags().String("notes", "", "Task notes/description")
	tasksCreateCmd.Flags().String("due", "", "Due date in RFC3339 or YYYY-MM-DD format")
	tasksCreateCmd.MarkFlagRequired("title")
}

func runTasksLists(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Tasks()
	if err != nil {
		return p.PrintError(err)
	}

	resp, err := svc.Tasklists.List().Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list task lists: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Items))
	for _, list := range resp.Items {
		listInfo := map[string]interface{}{
			"id":    list.Id,
			"title": list.Title,
		}
		results = append(results, listInfo)
	}

	return p.Print(map[string]interface{}{
		"tasklists": results,
		"count":     len(results),
	})
}

func runTasksList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Tasks()
	if err != nil {
		return p.PrintError(err)
	}

	tasklistID := args[0]
	showCompleted, _ := cmd.Flags().GetBool("show-completed")
	maxResults, _ := cmd.Flags().GetInt64("max")

	call := svc.Tasks.List(tasklistID).MaxResults(maxResults)
	if !showCompleted {
		call = call.ShowCompleted(false).ShowHidden(false)
	}

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list tasks: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Items))
	for _, task := range resp.Items {
		taskInfo := map[string]interface{}{
			"id":     task.Id,
			"title":  task.Title,
			"status": task.Status,
		}
		if task.Due != "" {
			taskInfo["due"] = task.Due
		}
		if task.Notes != "" {
			taskInfo["notes"] = task.Notes
		}
		if task.Parent != "" {
			taskInfo["parent"] = task.Parent
		}
		results = append(results, taskInfo)
	}

	return p.Print(map[string]interface{}{
		"tasks": results,
		"count": len(results),
	})
}

func runTasksCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Tasks()
	if err != nil {
		return p.PrintError(err)
	}

	tasklistID, _ := cmd.Flags().GetString("tasklist")
	title, _ := cmd.Flags().GetString("title")
	notes, _ := cmd.Flags().GetString("notes")
	due, _ := cmd.Flags().GetString("due")

	task := &tasks.Task{
		Title: title,
		Notes: notes,
	}

	if due != "" {
		task.Due = due
	}

	created, err := svc.Tasks.Insert(tasklistID, task).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create task: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "created",
		"id":     created.Id,
		"title":  created.Title,
	})
}

func runTasksComplete(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Tasks()
	if err != nil {
		return p.PrintError(err)
	}

	tasklistID := args[0]
	taskID := args[1]

	// Get the task first
	task, err := svc.Tasks.Get(tasklistID, taskID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get task: %w", err))
	}

	// Mark as completed
	task.Status = "completed"

	updated, err := svc.Tasks.Update(tasklistID, taskID, task).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to complete task: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "completed",
		"id":     updated.Id,
		"title":  updated.Title,
	})
}
