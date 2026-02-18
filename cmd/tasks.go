package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/tasks/v1"
)

// normalizeDueDate converts YYYY-MM-DD to RFC3339 format required by Google Tasks API.
// If already RFC3339, returns as-is.
func normalizeDueDate(due string) (string, error) {
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, due); matched {
		t, err := time.Parse("2006-01-02", due)
		if err != nil {
			return "", fmt.Errorf("invalid date %q: %w", due, err)
		}
		return t.UTC().Format(time.RFC3339), nil
	}
	return due, nil
}

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

var tasksUpdateCmd = &cobra.Command{
	Use:   "update <tasklist-id> <task-id>",
	Short: "Update a task",
	Long: `Updates an existing task's title, notes, or due date.

At least one of --title, --notes, or --due is required.

Examples:
  gws tasks update @default dGFzay0x --title "Updated title"
  gws tasks update @default dGFzay0x --notes "New notes" --due "2024-03-01"`,
	Args: cobra.ExactArgs(2),
	RunE: runTasksUpdate,
}

var tasksCompleteCmd = &cobra.Command{
	Use:   "complete <tasklist-id> <task-id>",
	Short: "Mark a task as completed",
	Long:  "Marks a specific task as completed.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTasksComplete,
}

var tasksListInfoCmd = &cobra.Command{
	Use:   "list-info <tasklist-id>",
	Short: "Get task list details",
	Long:  "Gets details for a specific task list.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTasksListInfo,
}

var tasksCreateListCmd = &cobra.Command{
	Use:   "create-list",
	Short: "Create a task list",
	Long:  "Creates a new task list.",
	RunE:  runTasksCreateList,
}

var tasksUpdateListCmd = &cobra.Command{
	Use:   "update-list <tasklist-id>",
	Short: "Update a task list",
	Long:  "Updates a task list's title.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTasksUpdateList,
}

var tasksDeleteListCmd = &cobra.Command{
	Use:   "delete-list <tasklist-id>",
	Short: "Delete a task list",
	Long:  "Deletes a task list and all its tasks.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTasksDeleteList,
}

var tasksGetCmd = &cobra.Command{
	Use:   "get <tasklist-id> <task-id>",
	Short: "Get a task",
	Long:  "Gets details for a specific task.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTasksGet,
}

var tasksDeleteCmd = &cobra.Command{
	Use:   "delete <tasklist-id> <task-id>",
	Short: "Delete a task",
	Long:  "Deletes a specific task.",
	Args:  cobra.ExactArgs(2),
	RunE:  runTasksDelete,
}

var tasksMoveCmd = &cobra.Command{
	Use:   "move <tasklist-id> <task-id>",
	Short: "Move a task",
	Long: `Moves a task to a different position, parent, or task list.

Examples:
  gws tasks move @default task-1 --previous task-2
  gws tasks move @default task-1 --parent parent-task
  gws tasks move @default task-1 --destination-list other-list-id`,
	Args: cobra.ExactArgs(2),
	RunE: runTasksMove,
}

var tasksClearCmd = &cobra.Command{
	Use:   "clear <tasklist-id>",
	Short: "Clear completed tasks",
	Long:  "Clears all completed tasks from a task list. Completed tasks are marked as hidden and no longer returned by default.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTasksClear,
}

func init() {
	rootCmd.AddCommand(tasksCmd)
	tasksCmd.AddCommand(tasksListsCmd)
	tasksCmd.AddCommand(tasksListCmd)
	tasksCmd.AddCommand(tasksCreateCmd)
	tasksCmd.AddCommand(tasksUpdateCmd)
	tasksCmd.AddCommand(tasksCompleteCmd)
	tasksCmd.AddCommand(tasksListInfoCmd)
	tasksCmd.AddCommand(tasksCreateListCmd)
	tasksCmd.AddCommand(tasksUpdateListCmd)
	tasksCmd.AddCommand(tasksDeleteListCmd)
	tasksCmd.AddCommand(tasksGetCmd)
	tasksCmd.AddCommand(tasksDeleteCmd)
	tasksCmd.AddCommand(tasksMoveCmd)
	tasksCmd.AddCommand(tasksClearCmd)

	// Update flags
	tasksUpdateCmd.Flags().String("title", "", "New task title")
	tasksUpdateCmd.Flags().String("notes", "", "New task notes/description")
	tasksUpdateCmd.Flags().String("due", "", "New due date in RFC3339 or YYYY-MM-DD format")

	// List flags
	tasksListCmd.Flags().Bool("show-completed", false, "Include completed tasks")
	tasksListCmd.Flags().Int64("max", 100, "Maximum number of tasks")

	// Create flags
	tasksCreateCmd.Flags().String("tasklist", "@default", "Task list ID (default: @default)")
	tasksCreateCmd.Flags().String("title", "", "Task title (required)")
	tasksCreateCmd.Flags().String("notes", "", "Task notes/description")
	tasksCreateCmd.Flags().String("due", "", "Due date in RFC3339 or YYYY-MM-DD format")
	tasksCreateCmd.MarkFlagRequired("title")

	// Create-list flags
	tasksCreateListCmd.Flags().String("title", "", "Task list title (required)")
	tasksCreateListCmd.MarkFlagRequired("title")

	// Update-list flags
	tasksUpdateListCmd.Flags().String("title", "", "New task list title (required)")
	tasksUpdateListCmd.MarkFlagRequired("title")

	// Move flags
	tasksMoveCmd.Flags().String("parent", "", "Parent task ID (makes this a subtask)")
	tasksMoveCmd.Flags().String("previous", "", "Previous sibling task ID (positions after this task)")
	tasksMoveCmd.Flags().String("destination-list", "", "Destination task list ID (moves to another list)")
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
		due, err = normalizeDueDate(due)
		if err != nil {
			return p.PrintError(err)
		}
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

func runTasksUpdate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	// Validate flags before creating client
	titleChanged := cmd.Flags().Changed("title")
	notesChanged := cmd.Flags().Changed("notes")
	dueChanged := cmd.Flags().Changed("due")

	if !titleChanged && !notesChanged && !dueChanged {
		return p.PrintError(fmt.Errorf("at least one of --title, --notes, or --due is required"))
	}

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

	// Get existing task
	task, err := svc.Tasks.Get(tasklistID, taskID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get task: %w", err))
	}

	// Apply updates (only for flags that were explicitly set)
	if titleChanged {
		title, _ := cmd.Flags().GetString("title")
		task.Title = title
	}
	if notesChanged {
		notes, _ := cmd.Flags().GetString("notes")
		task.Notes = notes
	}
	if dueChanged {
		due, _ := cmd.Flags().GetString("due")
		due, err = normalizeDueDate(due)
		if err != nil {
			return p.PrintError(err)
		}
		task.Due = due
	}

	updated, err := svc.Tasks.Update(tasklistID, taskID, task).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update task: %w", err))
	}

	result := map[string]interface{}{
		"status": "updated",
		"id":     updated.Id,
		"title":  updated.Title,
	}
	if updated.Notes != "" {
		result["notes"] = updated.Notes
	}
	if updated.Due != "" {
		result["due"] = updated.Due
	}

	return p.Print(result)
}

func runTasksListInfo(cmd *cobra.Command, args []string) error {
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

	list, err := svc.Tasklists.Get(tasklistID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get task list: %w", err))
	}

	return p.Print(map[string]interface{}{
		"id":       list.Id,
		"title":    list.Title,
		"updated":  list.Updated,
		"selfLink": list.SelfLink,
	})
}

func runTasksCreateList(cmd *cobra.Command, args []string) error {
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

	title, _ := cmd.Flags().GetString("title")

	list := &tasks.TaskList{
		Title: title,
	}

	created, err := svc.Tasklists.Insert(list).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create task list: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "created",
		"id":     created.Id,
		"title":  created.Title,
	})
}

func runTasksUpdateList(cmd *cobra.Command, args []string) error {
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
	title, _ := cmd.Flags().GetString("title")

	list := &tasks.TaskList{
		Title: title,
	}

	updated, err := svc.Tasklists.Patch(tasklistID, list).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update task list: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "updated",
		"id":     updated.Id,
		"title":  updated.Title,
	})
}

func runTasksDeleteList(cmd *cobra.Command, args []string) error {
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

	err = svc.Tasklists.Delete(tasklistID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete task list: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"id":     tasklistID,
	})
}

func runTasksGet(cmd *cobra.Command, args []string) error {
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

	task, err := svc.Tasks.Get(tasklistID, taskID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get task: %w", err))
	}

	result := map[string]interface{}{
		"id":     task.Id,
		"title":  task.Title,
		"status": task.Status,
	}
	if task.Notes != "" {
		result["notes"] = task.Notes
	}
	if task.Due != "" {
		result["due"] = task.Due
	}
	if task.Parent != "" {
		result["parent"] = task.Parent
	}
	if task.Completed != nil && *task.Completed != "" {
		result["completed"] = *task.Completed
	}
	if task.Updated != "" {
		result["updated"] = task.Updated
	}

	return p.Print(result)
}

func runTasksDelete(cmd *cobra.Command, args []string) error {
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

	err = svc.Tasks.Delete(tasklistID, taskID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete task: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"id":     taskID,
	})
}

func runTasksMove(cmd *cobra.Command, args []string) error {
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

	parent, _ := cmd.Flags().GetString("parent")
	previous, _ := cmd.Flags().GetString("previous")
	destinationList, _ := cmd.Flags().GetString("destination-list")

	call := svc.Tasks.Move(tasklistID, taskID)
	if parent != "" {
		call = call.Parent(parent)
	}
	if previous != "" {
		call = call.Previous(previous)
	}
	if destinationList != "" {
		call = call.DestinationTasklist(destinationList)
	}

	moved, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to move task: %w", err))
	}

	result := map[string]interface{}{
		"status": "moved",
		"id":     moved.Id,
		"title":  moved.Title,
	}
	if moved.Parent != "" {
		result["parent"] = moved.Parent
	}

	return p.Print(result)
}

func runTasksClear(cmd *cobra.Command, args []string) error {
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

	err = svc.Tasks.Clear(tasklistID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to clear completed tasks: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "cleared",
		"id":     tasklistID,
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
