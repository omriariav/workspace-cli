package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/omriariav/workspace-cli/gws/internal/client"
	"github.com/omriariav/workspace-cli/gws/internal/printer"
	"github.com/spf13/cobra"
)

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Manage Google Drive",
	Long:  "Commands for interacting with Google Drive files and folders.",
}

var driveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files",
	Long:  "Lists files and folders in Google Drive.",
	RunE:  runDriveList,
}

var driveSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for files",
	Long:  "Searches for files in Google Drive using a query string.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveSearch,
}

var driveDownloadCmd = &cobra.Command{
	Use:   "download <file-id>",
	Short: "Download a file",
	Long:  "Downloads a file from Google Drive.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveDownload,
}

var driveInfoCmd = &cobra.Command{
	Use:   "info <file-id>",
	Short: "Get file info",
	Long:  "Gets detailed information about a file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDriveInfo,
}

func init() {
	rootCmd.AddCommand(driveCmd)
	driveCmd.AddCommand(driveListCmd)
	driveCmd.AddCommand(driveSearchCmd)
	driveCmd.AddCommand(driveDownloadCmd)
	driveCmd.AddCommand(driveInfoCmd)

	// List flags
	driveListCmd.Flags().String("folder", "root", "Folder ID to list (default: root)")
	driveListCmd.Flags().Int64("max", 50, "Maximum number of files")
	driveListCmd.Flags().String("order", "modifiedTime desc", "Sort order (e.g., 'name', 'modifiedTime desc')")

	// Search flags
	driveSearchCmd.Flags().Int64("max", 50, "Maximum number of results")

	// Download flags
	driveDownloadCmd.Flags().String("output", "", "Output file path (default: original filename)")
}

func runDriveList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Drive()
	if err != nil {
		return p.PrintError(err)
	}

	folderID, _ := cmd.Flags().GetString("folder")
	maxResults, _ := cmd.Flags().GetInt64("max")
	orderBy, _ := cmd.Flags().GetString("order")

	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)

	resp, err := svc.Files.List().
		Q(query).
		PageSize(maxResults).
		OrderBy(orderBy).
		Fields("files(id, name, mimeType, size, modifiedTime, webViewLink)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list files: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Files))
	for _, file := range resp.Files {
		fileInfo := map[string]interface{}{
			"id":        file.Id,
			"name":      file.Name,
			"mime_type": file.MimeType,
		}
		if file.Size > 0 {
			fileInfo["size"] = file.Size
		}
		if file.ModifiedTime != "" {
			fileInfo["modified"] = file.ModifiedTime
		}
		if file.WebViewLink != "" {
			fileInfo["web_link"] = file.WebViewLink
		}
		results = append(results, fileInfo)
	}

	return p.Print(map[string]interface{}{
		"files": results,
		"count": len(results),
	})
}

func runDriveSearch(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Drive()
	if err != nil {
		return p.PrintError(err)
	}

	searchQuery := args[0]
	maxResults, _ := cmd.Flags().GetInt64("max")

	// Build query - search in name and full text
	query := fmt.Sprintf("(name contains '%s' or fullText contains '%s') and trashed = false", searchQuery, searchQuery)

	resp, err := svc.Files.List().
		Q(query).
		PageSize(maxResults).
		Fields("files(id, name, mimeType, size, modifiedTime, webViewLink)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to search files: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Files))
	for _, file := range resp.Files {
		fileInfo := map[string]interface{}{
			"id":        file.Id,
			"name":      file.Name,
			"mime_type": file.MimeType,
		}
		if file.Size > 0 {
			fileInfo["size"] = file.Size
		}
		if file.ModifiedTime != "" {
			fileInfo["modified"] = file.ModifiedTime
		}
		if file.WebViewLink != "" {
			fileInfo["web_link"] = file.WebViewLink
		}
		results = append(results, fileInfo)
	}

	return p.Print(map[string]interface{}{
		"files": results,
		"count": len(results),
		"query": searchQuery,
	})
}

func runDriveDownload(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Drive()
	if err != nil {
		return p.PrintError(err)
	}

	fileID := args[0]
	outputPath, _ := cmd.Flags().GetString("output")

	// Get file metadata first
	file, err := svc.Files.Get(fileID).Fields("name, mimeType, size").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get file info: %w", err))
	}

	// Determine output filename
	if outputPath == "" {
		outputPath = file.Name
	}

	// Check if it's a Google Workspace file (needs export)
	var resp *io.ReadCloser
	switch file.MimeType {
	case "application/vnd.google-apps.document":
		// Export as PDF
		exportResp, err := svc.Files.Export(fileID, "application/pdf").Download()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to export document: %w", err))
		}
		resp = &exportResp.Body
		if outputPath == file.Name {
			outputPath += ".pdf"
		}
	case "application/vnd.google-apps.spreadsheet":
		// Export as Excel
		exportResp, err := svc.Files.Export(fileID, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet").Download()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to export spreadsheet: %w", err))
		}
		resp = &exportResp.Body
		if outputPath == file.Name {
			outputPath += ".xlsx"
		}
	case "application/vnd.google-apps.presentation":
		// Export as PDF
		exportResp, err := svc.Files.Export(fileID, "application/pdf").Download()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to export presentation: %w", err))
		}
		resp = &exportResp.Body
		if outputPath == file.Name {
			outputPath += ".pdf"
		}
	default:
		// Regular file download
		downloadResp, err := svc.Files.Get(fileID).Download()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to download file: %w", err))
		}
		resp = &downloadResp.Body
	}
	defer (*resp).Close()

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create output file: %w", err))
	}
	defer outFile.Close()

	// Copy data
	written, err := io.Copy(outFile, *resp)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to write file: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":   "downloaded",
		"file":     outputPath,
		"size":     written,
		"original": file.Name,
	})
}

func runDriveInfo(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Drive()
	if err != nil {
		return p.PrintError(err)
	}

	fileID := args[0]

	file, err := svc.Files.Get(fileID).
		Fields("id, name, mimeType, size, createdTime, modifiedTime, webViewLink, webContentLink, owners, parents, shared").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get file info: %w", err))
	}

	result := map[string]interface{}{
		"id":        file.Id,
		"name":      file.Name,
		"mime_type": file.MimeType,
		"created":   file.CreatedTime,
		"modified":  file.ModifiedTime,
		"shared":    file.Shared,
	}

	if file.Size > 0 {
		result["size"] = file.Size
	}
	if file.WebViewLink != "" {
		result["web_link"] = file.WebViewLink
	}
	if file.WebContentLink != "" {
		result["download_link"] = file.WebContentLink
	}
	if len(file.Owners) > 0 {
		owners := make([]string, len(file.Owners))
		for i, owner := range file.Owners {
			owners[i] = owner.EmailAddress
		}
		result["owners"] = owners
	}
	if len(file.Parents) > 0 {
		result["parents"] = file.Parents
	}

	return p.Print(result)
}
