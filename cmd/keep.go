package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/keep/v1"
)

var keepCmd = &cobra.Command{
	Use:   "keep",
	Short: "Manage Google Keep notes",
	Long:  "Commands for interacting with Google Keep.",
}

var keepListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	Long:  "Lists notes from Google Keep.",
	Args:  cobra.NoArgs,
	RunE:  runKeepList,
}

var keepGetCmd = &cobra.Command{
	Use:   "get <note-id>",
	Short: "Get a note",
	Long:  "Gets a specific note from Google Keep by its ID (e.g. notes/abc123).",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeepGet,
}

var keepCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a note",
	Long: `Creates a new note in Google Keep.

Examples:
  gws keep create --title "Shopping List" --text "Milk, eggs, bread"
  gws keep create --title "Meeting Notes" --text "Discuss Q1 goals"`,
	Args: cobra.NoArgs,
	RunE: runKeepCreate,
}

func init() {
	rootCmd.AddCommand(keepCmd)
	keepCmd.AddCommand(keepListCmd)
	keepCmd.AddCommand(keepGetCmd)
	keepCmd.AddCommand(keepCreateCmd)

	// list flags
	keepListCmd.Flags().Int("max", 20, "Maximum number of notes to return")

	// create flags
	keepCreateCmd.Flags().String("title", "", "Note title (required)")
	keepCreateCmd.Flags().String("text", "", "Note text content (required)")
	keepCreateCmd.MarkFlagRequired("title")
	keepCreateCmd.MarkFlagRequired("text")
}

func runKeepList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Keep()
	if err != nil {
		return p.PrintError(err)
	}

	max, _ := cmd.Flags().GetInt("max")

	resp, err := svc.Notes.List().PageSize(int64(max)).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list notes: %w", err))
	}

	notes := make([]map[string]interface{}, 0, len(resp.Notes))
	for _, n := range resp.Notes {
		note := formatNote(n)
		notes = append(notes, note)
	}

	return p.Print(map[string]interface{}{
		"notes": notes,
		"count": len(notes),
	})
}

func runKeepGet(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Keep()
	if err != nil {
		return p.PrintError(err)
	}

	noteID := args[0]

	n, err := svc.Notes.Get(noteID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get note: %w", err))
	}

	return p.Print(formatNote(n))
}

func runKeepCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Keep()
	if err != nil {
		return p.PrintError(err)
	}

	title, _ := cmd.Flags().GetString("title")
	text, _ := cmd.Flags().GetString("text")

	newNote := &keep.Note{
		Title: title,
		Body: &keep.Section{
			Text: &keep.TextContent{
				Text: text,
			},
		},
	}

	n, err := svc.Notes.Create(newNote).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create note: %w", err))
	}

	return p.Print(formatNote(n))
}

func formatNote(n *keep.Note) map[string]interface{} {
	result := map[string]interface{}{
		"name":  n.Name,
		"title": n.Title,
	}

	if n.Body != nil && n.Body.Text != nil {
		result["text"] = n.Body.Text.Text
	}

	if n.CreateTime != "" {
		result["create_time"] = n.CreateTime
	}
	if n.UpdateTime != "" {
		result["update_time"] = n.UpdateTime
	}
	if n.Trashed {
		result["trashed"] = true
	}

	return result
}
