package cmd

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
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

var driveCommentsCmd = &cobra.Command{
	Use:   "comments <file-id>",
	Short: "List comments on a file",
	Long: `Lists all comments and replies on a Google Drive file (Docs, Sheets, Slides, etc.).

By default, resolved comments are excluded. Use --include-resolved to see them.
Note: When filtering resolved comments, the actual result count may be less than --max
since filtering happens after fetching from the API.`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveComments,
}

var driveUploadCmd = &cobra.Command{
	Use:   "upload <local-file>",
	Short: "Upload a file to Drive",
	Long: `Uploads a local file to Google Drive.

Examples:
  gws drive upload report.pdf
  gws drive upload data.xlsx --folder 1abc123xyz
  gws drive upload document.docx --name "My Report"`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveUpload,
}

var driveCreateFolderCmd = &cobra.Command{
	Use:   "create-folder",
	Short: "Create a new folder",
	Long: `Creates a new folder in Google Drive.

Examples:
  gws drive create-folder --name "Project Files"
  gws drive create-folder --name "Subproject" --parent 1abc123xyz`,
	RunE: runDriveCreateFolder,
}

var driveMoveCmd = &cobra.Command{
	Use:   "move <file-id>",
	Short: "Move a file to another folder",
	Long: `Moves a file to a different folder in Google Drive.

Examples:
  gws drive move 1abc123xyz --to 2def456uvw
  gws drive move 1abc123xyz --to root`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveMove,
}

var driveDeleteCmd = &cobra.Command{
	Use:   "delete <file-id>",
	Short: "Delete a file",
	Long: `Deletes a file from Google Drive.

By default, moves the file to trash. Use --permanent to permanently delete.

Warning: --permanent bypasses trash and cannot be undone.

Examples:
  gws drive delete 1abc123xyz
  gws drive delete 1abc123xyz --permanent`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveDelete,
}

func init() {
	rootCmd.AddCommand(driveCmd)
	driveCmd.AddCommand(driveListCmd)
	driveCmd.AddCommand(driveSearchCmd)
	driveCmd.AddCommand(driveDownloadCmd)
	driveCmd.AddCommand(driveInfoCmd)
	driveCmd.AddCommand(driveCommentsCmd)
	driveCmd.AddCommand(driveUploadCmd)
	driveCmd.AddCommand(driveCreateFolderCmd)
	driveCmd.AddCommand(driveMoveCmd)
	driveCmd.AddCommand(driveDeleteCmd)

	// List flags
	driveListCmd.Flags().String("folder", "root", "Folder ID to list (default: root)")
	driveListCmd.Flags().Int64("max", 50, "Maximum number of files")
	driveListCmd.Flags().String("order", "modifiedTime desc", "Sort order (e.g., 'name', 'modifiedTime desc')")

	// Search flags
	driveSearchCmd.Flags().Int64("max", 50, "Maximum number of results")

	// Download flags
	driveDownloadCmd.Flags().String("output", "", "Output file path (default: original filename)")

	// Comments flags
	driveCommentsCmd.Flags().Int64("max", 100, "Maximum number of comments")
	driveCommentsCmd.Flags().Bool("include-resolved", false, "Include resolved comments")
	driveCommentsCmd.Flags().Bool("include-deleted", false, "Include deleted comments")

	// Upload flags
	driveUploadCmd.Flags().String("folder", "", "Parent folder ID (default: root)")
	driveUploadCmd.Flags().String("name", "", "File name in Drive (default: local filename)")
	driveUploadCmd.Flags().String("mime-type", "", "MIME type (auto-detected if not specified)")

	// Create folder flags
	driveCreateFolderCmd.Flags().String("name", "", "Folder name (required)")
	driveCreateFolderCmd.Flags().String("parent", "", "Parent folder ID (default: root)")
	driveCreateFolderCmd.MarkFlagRequired("name")

	// Move flags
	driveMoveCmd.Flags().String("to", "", "Destination folder ID (required)")
	driveMoveCmd.MarkFlagRequired("to")

	// Delete flags
	driveDeleteCmd.Flags().Bool("permanent", false, "Permanently delete (skip trash)")
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
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
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
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
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
	file, err := svc.Files.Get(fileID).SupportsAllDrives(true).Fields("name, mimeType, size").Do()
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
		downloadResp, err := svc.Files.Get(fileID).SupportsAllDrives(true).Download()
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
		SupportsAllDrives(true).
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

func runDriveComments(cmd *cobra.Command, args []string) error {
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
	maxResults, _ := cmd.Flags().GetInt64("max")
	includeResolved, _ := cmd.Flags().GetBool("include-resolved")
	includeDeleted, _ := cmd.Flags().GetBool("include-deleted")

	// Get file info first for context
	file, err := svc.Files.Get(fileID).SupportsAllDrives(true).Fields("name, mimeType").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get file info: %w", err))
	}

	// List comments with full field specification including anchor and reply subfields
	fields := "nextPageToken,comments(id,content,anchor,author(displayName,emailAddress),createdTime,modifiedTime,resolved,quotedFileContent(mimeType,value),replies(id,content,author(displayName,emailAddress),createdTime,modifiedTime,action))"

	var allComments []*drive.Comment
	var pageToken string
	for {
		call := svc.Comments.List(fileID).
			PageSize(maxResults).
			Fields(googleapi.Field(fields))
		if includeDeleted {
			call = call.IncludeDeleted(true)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list comments: %w", err))
		}
		allComments = append(allComments, resp.Comments...)
		if resp.NextPageToken == "" || int64(len(allComments)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}
	// Trim to max if pagination overshot
	if int64(len(allComments)) > maxResults {
		allComments = allComments[:maxResults]
	}

	comments := make([]map[string]interface{}, 0)
	for _, comment := range allComments {
		// Skip resolved comments unless explicitly requested
		if comment.Resolved && !includeResolved {
			continue
		}

		// Construct direct link to comment based on file type
		var directLink string
		switch file.MimeType {
		case "application/vnd.google-apps.document":
			directLink = fmt.Sprintf("https://docs.google.com/document/d/%s/edit?disco=%s", fileID, comment.Id)
		case "application/vnd.google-apps.spreadsheet":
			directLink = fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit?disco=%s", fileID, comment.Id)
		case "application/vnd.google-apps.presentation":
			directLink = fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit?disco=%s", fileID, comment.Id)
		default:
			directLink = fmt.Sprintf("https://drive.google.com/file/d/%s/view?disco=%s", fileID, comment.Id)
		}

		c := map[string]interface{}{
			"id":          comment.Id,
			"content":     comment.Content,
			"created":     comment.CreatedTime,
			"resolved":    comment.Resolved,
			"direct_link": directLink,
		}

		if comment.ModifiedTime != "" {
			c["modified"] = comment.ModifiedTime
		}

		// Anchor info (e.g., slide or element location for presentations)
		if comment.Anchor != "" {
			c["anchor"] = comment.Anchor
		}

		// Author info
		if comment.Author != nil {
			c["author"] = map[string]interface{}{
				"name":  comment.Author.DisplayName,
				"email": comment.Author.EmailAddress,
			}
		}

		// Quoted text (the text the comment is anchored to)
		if comment.QuotedFileContent != nil && comment.QuotedFileContent.Value != "" {
			c["quoted_text"] = comment.QuotedFileContent.Value
		}

		// Replies
		if len(comment.Replies) > 0 {
			replies := make([]map[string]interface{}, 0, len(comment.Replies))
			for _, reply := range comment.Replies {
				r := map[string]interface{}{
					"id":      reply.Id,
					"content": reply.Content,
					"created": reply.CreatedTime,
				}
				if reply.Author != nil {
					r["author"] = map[string]interface{}{
						"name":  reply.Author.DisplayName,
						"email": reply.Author.EmailAddress,
					}
				}
				if reply.Action != "" {
					r["action"] = reply.Action
				}
				replies = append(replies, r)
			}
			c["replies"] = replies
		}

		comments = append(comments, c)
	}

	result := map[string]interface{}{
		"file_id":   fileID,
		"file_name": file.Name,
		"mime_type": file.MimeType,
		"comments":  comments,
		"count":     len(comments),
	}

	// Add note for presentations when no comments found
	if len(comments) == 0 && file.MimeType == "application/vnd.google-apps.presentation" {
		result["note"] = "No comments returned. Google Slides comments made via 'Insert > Comment' are stored in the Drive comments API. If you see comments in the UI but not here, they may be resolved (use --include-resolved) or may be suggestion-type annotations not exposed via this API."
	}

	return p.Print(result)
}

func runDriveUpload(cmd *cobra.Command, args []string) error {
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

	localPath := args[0]
	folderID, _ := cmd.Flags().GetString("folder")
	fileName, _ := cmd.Flags().GetString("name")
	mimeType, _ := cmd.Flags().GetString("mime-type")

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to open file: %w", err))
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to stat file: %w", err))
	}

	// Use local filename if name not specified
	if fileName == "" {
		fileName = filepath.Base(localPath)
	}

	// Auto-detect MIME type if not specified
	if mimeType == "" {
		ext := filepath.Ext(localPath)
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// Build file metadata
	driveFile := &drive.File{
		Name:     fileName,
		MimeType: mimeType,
	}

	// Set parent folder if specified
	if folderID != "" {
		driveFile.Parents = []string{folderID}
	}

	// Upload file
	created, err := svc.Files.Create(driveFile).
		SupportsAllDrives(true).
		Media(file).
		Fields("id, name, mimeType, size, webViewLink").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to upload file: %w", err))
	}

	result := map[string]interface{}{
		"status":    "uploaded",
		"id":        created.Id,
		"name":      created.Name,
		"mime_type": created.MimeType,
		"size":      fileInfo.Size(),
	}

	if created.WebViewLink != "" {
		result["web_link"] = created.WebViewLink
	}

	return p.Print(result)
}

func runDriveCreateFolder(cmd *cobra.Command, args []string) error {
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

	name, _ := cmd.Flags().GetString("name")
	parentID, _ := cmd.Flags().GetString("parent")

	// Build folder metadata
	folder := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}

	// Set parent folder if specified
	if parentID != "" {
		folder.Parents = []string{parentID}
	}

	// Create folder
	created, err := svc.Files.Create(folder).
		SupportsAllDrives(true).
		Fields("id, name, webViewLink").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create folder: %w", err))
	}

	result := map[string]interface{}{
		"status": "created",
		"id":     created.Id,
		"name":   created.Name,
	}

	if created.WebViewLink != "" {
		result["web_link"] = created.WebViewLink
	}

	return p.Print(result)
}

func runDriveMove(cmd *cobra.Command, args []string) error {
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
	toFolderID, _ := cmd.Flags().GetString("to")

	// Get current file info to find existing parents
	file, err := svc.Files.Get(fileID).SupportsAllDrives(true).Fields("name, parents").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get file info: %w", err))
	}

	// Build removeParents string (comma-separated list of current parents)
	var removeParents string
	if len(file.Parents) > 0 {
		for i, parent := range file.Parents {
			if i > 0 {
				removeParents += ","
			}
			removeParents += parent
		}
	}

	// Move file by adding new parent and removing old parents
	updated, err := svc.Files.Update(fileID, nil).
		SupportsAllDrives(true).
		AddParents(toFolderID).
		RemoveParents(removeParents).
		Fields("id, name, parents, webViewLink").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to move file: %w", err))
	}

	result := map[string]interface{}{
		"status":  "moved",
		"id":      updated.Id,
		"name":    updated.Name,
		"parents": updated.Parents,
	}

	if updated.WebViewLink != "" {
		result["web_link"] = updated.WebViewLink
	}

	return p.Print(result)
}

func runDriveDelete(cmd *cobra.Command, args []string) error {
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
	permanent, _ := cmd.Flags().GetBool("permanent")

	// Get file info first for the response
	file, err := svc.Files.Get(fileID).SupportsAllDrives(true).Fields("name").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get file info: %w", err))
	}

	if permanent {
		// Permanently delete
		err = svc.Files.Delete(fileID).SupportsAllDrives(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to delete file: %w", err))
		}

		return p.Print(map[string]interface{}{
			"status": "deleted",
			"id":     fileID,
			"name":   file.Name,
		})
	}

	// Move to trash
	_, err = svc.Files.Update(fileID, &drive.File{Trashed: true}).SupportsAllDrives(true).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to trash file: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "trashed",
		"id":     fileID,
		"name":   file.Name,
	})
}
