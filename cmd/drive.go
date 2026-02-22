package cmd

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/drive/v3"
	driveactivity "google.golang.org/api/driveactivity/v2"
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

var driveCopyCmd = &cobra.Command{
	Use:   "copy <file-id>",
	Short: "Copy a file",
	Long: `Creates a copy of a file in Google Drive.

Useful for duplicating template files (Docs, Sheets, Slides, etc.).

Examples:
  gws drive copy 1abc123xyz
  gws drive copy 1abc123xyz --name "My Copy"
  gws drive copy 1abc123xyz --name "Project Deck" --folder 2def456uvw`,
	Args: cobra.ExactArgs(1),
	RunE: runDriveCopy,
}

// --- Permissions ---

var drivePermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "List permissions on a file",
	Long:  "Lists all permissions on a Google Drive file.",
	RunE:  runDrivePermissions,
}

var driveShareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share a file",
	Long: `Shares a file with a user, group, domain, or anyone.

Examples:
  gws drive share --file-id <id> --type user --role writer --email user@example.com
  gws drive share --file-id <id> --type domain --role reader --domain example.com
  gws drive share --file-id <id> --type anyone --role reader`,
	RunE: runDriveShare,
}

var driveUnshareCmd = &cobra.Command{
	Use:   "unshare",
	Short: "Remove a permission",
	Long:  "Removes a permission from a Google Drive file.",
	RunE:  runDriveUnshare,
}

var drivePermissionCmd = &cobra.Command{
	Use:   "permission",
	Short: "Get permission details",
	Long:  "Gets details of a specific permission on a file.",
	RunE:  runDrivePermission,
}

var driveUpdatePermissionCmd = &cobra.Command{
	Use:   "update-permission",
	Short: "Update a permission",
	Long:  "Updates the role of an existing permission on a file.",
	RunE:  runDriveUpdatePermission,
}

// --- Revisions ---

var driveRevisionsCmd = &cobra.Command{
	Use:   "revisions",
	Short: "List file revisions",
	Long:  "Lists all revisions of a Google Drive file.",
	RunE:  runDriveRevisions,
}

var driveRevisionCmd = &cobra.Command{
	Use:   "revision",
	Short: "Get revision details",
	Long:  "Gets details of a specific file revision.",
	RunE:  runDriveRevision,
}

var driveDeleteRevisionCmd = &cobra.Command{
	Use:   "delete-revision",
	Short: "Delete a revision",
	Long:  "Deletes a specific revision of a file.",
	RunE:  runDriveDeleteRevision,
}

// --- Replies ---

var driveRepliesCmd = &cobra.Command{
	Use:   "replies",
	Short: "List replies to a comment",
	Long:  "Lists all replies to a comment on a Google Drive file.",
	RunE:  runDriveReplies,
}

var driveReplyCmd = &cobra.Command{
	Use:   "reply",
	Short: "Reply to a comment",
	Long:  "Creates a reply to a comment on a Google Drive file.",
	RunE:  runDriveReply,
}

var driveGetReplyCmd = &cobra.Command{
	Use:   "get-reply",
	Short: "Get a reply",
	Long:  "Gets a specific reply to a comment.",
	RunE:  runDriveGetReply,
}

var driveDeleteReplyCmd = &cobra.Command{
	Use:   "delete-reply",
	Short: "Delete a reply",
	Long:  "Deletes a reply to a comment.",
	RunE:  runDriveDeleteReply,
}

// --- Comments (single) ---

var driveCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Get a comment",
	Long:  "Gets a specific comment on a Google Drive file.",
	RunE:  runDriveComment,
}

var driveAddCommentCmd = &cobra.Command{
	Use:   "add-comment",
	Short: "Add a comment",
	Long:  "Adds a comment to a Google Drive file.",
	RunE:  runDriveAddComment,
}

var driveDeleteCommentCmd = &cobra.Command{
	Use:   "delete-comment",
	Short: "Delete a comment",
	Long:  "Deletes a comment from a Google Drive file.",
	RunE:  runDriveDeleteComment,
}

// --- Files ---

var driveExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a Google Workspace file",
	Long: `Exports a Google Workspace file (Docs, Sheets, Slides) to a specified format.

Examples:
  gws drive export --file-id <id> --mime-type application/pdf --output report.pdf
  gws drive export --file-id <id> --mime-type text/csv --output data.csv`,
	RunE: runDriveExport,
}

var driveEmptyTrashCmd = &cobra.Command{
	Use:   "empty-trash",
	Short: "Empty trash",
	Long:  "Permanently deletes all files in the trash.",
	RunE:  runDriveEmptyTrash,
}

var driveUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update file metadata",
	Long: `Updates metadata of a file in Google Drive.

Examples:
  gws drive update --file-id <id> --name "New Name"
  gws drive update --file-id <id> --starred
  gws drive update --file-id <id> --description "Updated description"`,
	RunE: runDriveUpdate,
}

// --- Shared Drives ---

var driveSharedDrivesCmd = &cobra.Command{
	Use:   "shared-drives",
	Short: "List shared drives",
	Long:  "Lists all shared drives the user has access to.",
	RunE:  runDriveSharedDrives,
}

var driveSharedDriveCmd = &cobra.Command{
	Use:   "shared-drive",
	Short: "Get shared drive info",
	Long:  "Gets information about a shared drive.",
	RunE:  runDriveSharedDrive,
}

var driveCreateDriveCmd = &cobra.Command{
	Use:   "create-drive",
	Short: "Create a shared drive",
	Long:  "Creates a new shared drive.",
	RunE:  runDriveCreateDrive,
}

var driveDeleteDriveCmd = &cobra.Command{
	Use:   "delete-drive",
	Short: "Delete a shared drive",
	Long:  "Deletes a shared drive.",
	RunE:  runDriveDeleteDrive,
}

var driveUpdateDriveCmd = &cobra.Command{
	Use:   "update-drive",
	Short: "Update a shared drive",
	Long:  "Updates the name of a shared drive.",
	RunE:  runDriveUpdateDrive,
}

// --- Other ---

var driveAboutCmd = &cobra.Command{
	Use:   "about",
	Short: "Get drive storage and user info",
	Long:  "Gets information about the user's Drive storage quota and account.",
	RunE:  runDriveAbout,
}

var driveChangesCmd = &cobra.Command{
	Use:   "changes",
	Short: "List recent file changes",
	Long: `Lists recent changes to files in Google Drive.

If no --page-token is provided, the current start token is fetched automatically.
The first call will typically return zero results because the token represents "now";
save the returned new_start_page_token and pass it in subsequent calls to poll for
new changes (standard Drive changes polling pattern).`,
	RunE: runDriveChanges,
}

var driveActivityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Query Drive activity history",
	Long: `Queries the Google Drive Activity API v2 for file and folder activity.

Returns a detailed activity log including creates, edits, moves, renames, deletes,
restores, comments, permission changes, settings changes, and more.

Examples:
  gws drive activity --item-id 1abc123xyz
  gws drive activity --folder-id 0Bxyz789 --days 7
  gws drive activity --item-id 1abc123xyz --filter "detail.action_detail_case:EDIT"
  gws drive activity --item-id 1abc123xyz --no-consolidation --max 100`,
	RunE: runDriveActivity,
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
	driveCmd.AddCommand(driveCopyCmd)

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

	// Copy flags
	driveCopyCmd.Flags().String("name", "", "Name for the copy (default: 'Copy of <original>')")
	driveCopyCmd.Flags().String("folder", "", "Destination folder ID")

	// --- New commands ---
	driveCmd.AddCommand(drivePermissionsCmd)
	driveCmd.AddCommand(driveShareCmd)
	driveCmd.AddCommand(driveUnshareCmd)
	driveCmd.AddCommand(drivePermissionCmd)
	driveCmd.AddCommand(driveUpdatePermissionCmd)
	driveCmd.AddCommand(driveRevisionsCmd)
	driveCmd.AddCommand(driveRevisionCmd)
	driveCmd.AddCommand(driveDeleteRevisionCmd)
	driveCmd.AddCommand(driveRepliesCmd)
	driveCmd.AddCommand(driveReplyCmd)
	driveCmd.AddCommand(driveGetReplyCmd)
	driveCmd.AddCommand(driveDeleteReplyCmd)
	driveCmd.AddCommand(driveCommentCmd)
	driveCmd.AddCommand(driveAddCommentCmd)
	driveCmd.AddCommand(driveDeleteCommentCmd)
	driveCmd.AddCommand(driveExportCmd)
	driveCmd.AddCommand(driveEmptyTrashCmd)
	driveCmd.AddCommand(driveUpdateCmd)
	driveCmd.AddCommand(driveSharedDrivesCmd)
	driveCmd.AddCommand(driveSharedDriveCmd)
	driveCmd.AddCommand(driveCreateDriveCmd)
	driveCmd.AddCommand(driveDeleteDriveCmd)
	driveCmd.AddCommand(driveUpdateDriveCmd)
	driveCmd.AddCommand(driveAboutCmd)
	driveCmd.AddCommand(driveChangesCmd)

	// Permissions flags
	drivePermissionsCmd.Flags().String("file-id", "", "File ID (required)")
	drivePermissionsCmd.MarkFlagRequired("file-id")

	driveShareCmd.Flags().String("file-id", "", "File ID (required)")
	driveShareCmd.Flags().String("type", "", "Permission type: user, group, domain, anyone (required)")
	driveShareCmd.Flags().String("role", "", "Role: reader, commenter, writer, organizer, owner (required)")
	driveShareCmd.Flags().String("email", "", "Email address (for user/group type)")
	driveShareCmd.Flags().String("domain", "", "Domain (for domain type)")
	driveShareCmd.Flags().Bool("send-notification", true, "Send notification email")
	driveShareCmd.MarkFlagRequired("file-id")
	driveShareCmd.MarkFlagRequired("type")
	driveShareCmd.MarkFlagRequired("role")

	driveUnshareCmd.Flags().String("file-id", "", "File ID (required)")
	driveUnshareCmd.Flags().String("permission-id", "", "Permission ID (required)")
	driveUnshareCmd.MarkFlagRequired("file-id")
	driveUnshareCmd.MarkFlagRequired("permission-id")

	drivePermissionCmd.Flags().String("file-id", "", "File ID (required)")
	drivePermissionCmd.Flags().String("permission-id", "", "Permission ID (required)")
	drivePermissionCmd.MarkFlagRequired("file-id")
	drivePermissionCmd.MarkFlagRequired("permission-id")

	driveUpdatePermissionCmd.Flags().String("file-id", "", "File ID (required)")
	driveUpdatePermissionCmd.Flags().String("permission-id", "", "Permission ID (required)")
	driveUpdatePermissionCmd.Flags().String("role", "", "New role (required)")
	driveUpdatePermissionCmd.MarkFlagRequired("file-id")
	driveUpdatePermissionCmd.MarkFlagRequired("permission-id")
	driveUpdatePermissionCmd.MarkFlagRequired("role")

	// Revisions flags
	driveRevisionsCmd.Flags().String("file-id", "", "File ID (required)")
	driveRevisionsCmd.MarkFlagRequired("file-id")

	driveRevisionCmd.Flags().String("file-id", "", "File ID (required)")
	driveRevisionCmd.Flags().String("revision-id", "", "Revision ID (required)")
	driveRevisionCmd.MarkFlagRequired("file-id")
	driveRevisionCmd.MarkFlagRequired("revision-id")

	driveDeleteRevisionCmd.Flags().String("file-id", "", "File ID (required)")
	driveDeleteRevisionCmd.Flags().String("revision-id", "", "Revision ID (required)")
	driveDeleteRevisionCmd.MarkFlagRequired("file-id")
	driveDeleteRevisionCmd.MarkFlagRequired("revision-id")

	// Replies flags
	driveRepliesCmd.Flags().String("file-id", "", "File ID (required)")
	driveRepliesCmd.Flags().String("comment-id", "", "Comment ID (required)")
	driveRepliesCmd.MarkFlagRequired("file-id")
	driveRepliesCmd.MarkFlagRequired("comment-id")

	driveReplyCmd.Flags().String("file-id", "", "File ID (required)")
	driveReplyCmd.Flags().String("comment-id", "", "Comment ID (required)")
	driveReplyCmd.Flags().String("content", "", "Reply content (required)")
	driveReplyCmd.MarkFlagRequired("file-id")
	driveReplyCmd.MarkFlagRequired("comment-id")
	driveReplyCmd.MarkFlagRequired("content")

	driveGetReplyCmd.Flags().String("file-id", "", "File ID (required)")
	driveGetReplyCmd.Flags().String("comment-id", "", "Comment ID (required)")
	driveGetReplyCmd.Flags().String("reply-id", "", "Reply ID (required)")
	driveGetReplyCmd.MarkFlagRequired("file-id")
	driveGetReplyCmd.MarkFlagRequired("comment-id")
	driveGetReplyCmd.MarkFlagRequired("reply-id")

	driveDeleteReplyCmd.Flags().String("file-id", "", "File ID (required)")
	driveDeleteReplyCmd.Flags().String("comment-id", "", "Comment ID (required)")
	driveDeleteReplyCmd.Flags().String("reply-id", "", "Reply ID (required)")
	driveDeleteReplyCmd.MarkFlagRequired("file-id")
	driveDeleteReplyCmd.MarkFlagRequired("comment-id")
	driveDeleteReplyCmd.MarkFlagRequired("reply-id")

	// Comment flags
	driveCommentCmd.Flags().String("file-id", "", "File ID (required)")
	driveCommentCmd.Flags().String("comment-id", "", "Comment ID (required)")
	driveCommentCmd.MarkFlagRequired("file-id")
	driveCommentCmd.MarkFlagRequired("comment-id")

	driveAddCommentCmd.Flags().String("file-id", "", "File ID (required)")
	driveAddCommentCmd.Flags().String("content", "", "Comment content (required)")
	driveAddCommentCmd.MarkFlagRequired("file-id")
	driveAddCommentCmd.MarkFlagRequired("content")

	driveDeleteCommentCmd.Flags().String("file-id", "", "File ID (required)")
	driveDeleteCommentCmd.Flags().String("comment-id", "", "Comment ID (required)")
	driveDeleteCommentCmd.MarkFlagRequired("file-id")
	driveDeleteCommentCmd.MarkFlagRequired("comment-id")

	// Export flags
	driveExportCmd.Flags().String("file-id", "", "File ID (required)")
	driveExportCmd.Flags().String("mime-type", "", "Export MIME type (required, e.g. application/pdf)")
	driveExportCmd.Flags().String("output", "", "Output file path (required)")
	driveExportCmd.MarkFlagRequired("file-id")
	driveExportCmd.MarkFlagRequired("mime-type")
	driveExportCmd.MarkFlagRequired("output")

	// Update flags
	driveUpdateCmd.Flags().String("file-id", "", "File ID (required)")
	driveUpdateCmd.Flags().String("name", "", "New file name")
	driveUpdateCmd.Flags().String("description", "", "New description")
	driveUpdateCmd.Flags().Bool("starred", false, "Star or unstar the file")
	driveUpdateCmd.Flags().Bool("trashed", false, "Trash or untrash the file")
	driveUpdateCmd.MarkFlagRequired("file-id")

	// Shared Drives flags
	driveSharedDrivesCmd.Flags().Int64("max", 100, "Maximum number of shared drives")
	driveSharedDrivesCmd.Flags().String("query", "", "Search query")

	driveSharedDriveCmd.Flags().String("id", "", "Shared drive ID (required)")
	driveSharedDriveCmd.MarkFlagRequired("id")

	driveCreateDriveCmd.Flags().String("name", "", "Shared drive name (required)")
	driveCreateDriveCmd.MarkFlagRequired("name")

	driveDeleteDriveCmd.Flags().String("id", "", "Shared drive ID (required)")
	driveDeleteDriveCmd.MarkFlagRequired("id")

	driveUpdateDriveCmd.Flags().String("id", "", "Shared drive ID (required)")
	driveUpdateDriveCmd.Flags().String("name", "", "New name for the shared drive")
	driveUpdateDriveCmd.MarkFlagRequired("id")

	// Changes flags
	driveChangesCmd.Flags().Int64("max", 100, "Maximum number of changes")
	driveChangesCmd.Flags().String("page-token", "", "Page token (fetches start token if empty)")

	// Activity command
	driveCmd.AddCommand(driveActivityCmd)
	driveActivityCmd.Flags().String("item-id", "", "Filter by file/folder ID")
	driveActivityCmd.Flags().String("folder-id", "", "Filter by ancestor folder (all descendants)")
	driveActivityCmd.Flags().String("filter", "", "API filter string (e.g. \"detail.action_detail_case:EDIT\")")
	driveActivityCmd.Flags().Int("days", 0, "Last N days (auto-generates time filter)")
	driveActivityCmd.Flags().Int64("max", 50, "Page size")
	driveActivityCmd.Flags().String("page-token", "", "Pagination token")
	driveActivityCmd.Flags().Bool("no-consolidation", false, "Disable activity grouping")
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

func runDriveCopy(cmd *cobra.Command, args []string) error {
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
	name, _ := cmd.Flags().GetString("name")
	folderID, _ := cmd.Flags().GetString("folder")

	copyFile := &drive.File{}
	if name != "" {
		copyFile.Name = name
	}
	if folderID != "" {
		copyFile.Parents = []string{folderID}
	}

	copied, err := svc.Files.Copy(fileID, copyFile).
		SupportsAllDrives(true).
		Fields("id,name,mimeType,webViewLink").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to copy file: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":    "copied",
		"id":        copied.Id,
		"name":      copied.Name,
		"mime_type": copied.MimeType,
		"web_link":  copied.WebViewLink,
	})
}

// --- Permissions ---

func runDrivePermissions(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")

	resp, err := svc.Permissions.List(fileID).
		SupportsAllDrives(true).
		Fields("permissions(id,type,role,emailAddress,domain,displayName)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list permissions: %w", err))
	}

	permissions := make([]map[string]interface{}, 0, len(resp.Permissions))
	for _, perm := range resp.Permissions {
		permInfo := map[string]interface{}{
			"id":   perm.Id,
			"type": perm.Type,
			"role": perm.Role,
		}
		if perm.EmailAddress != "" {
			permInfo["email"] = perm.EmailAddress
		}
		if perm.Domain != "" {
			permInfo["domain"] = perm.Domain
		}
		if perm.DisplayName != "" {
			permInfo["display_name"] = perm.DisplayName
		}
		permissions = append(permissions, permInfo)
	}

	return p.Print(map[string]interface{}{
		"file_id":     fileID,
		"permissions": permissions,
		"count":       len(permissions),
	})
}

func runDriveShare(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	permType, _ := cmd.Flags().GetString("type")
	role, _ := cmd.Flags().GetString("role")
	email, _ := cmd.Flags().GetString("email")
	domain, _ := cmd.Flags().GetString("domain")
	sendNotification, _ := cmd.Flags().GetBool("send-notification")

	perm := &drive.Permission{
		Type: permType,
		Role: role,
	}
	if email != "" {
		perm.EmailAddress = email
	}
	if domain != "" {
		perm.Domain = domain
	}

	call := svc.Permissions.Create(fileID, perm).
		SupportsAllDrives(true).
		SendNotificationEmail(sendNotification).
		Fields("id,type,role,emailAddress,domain,displayName")

	if role == "owner" {
		call = call.TransferOwnership(true)
	}

	created, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to share file: %w", err))
	}

	result := map[string]interface{}{
		"status": "shared",
		"id":     created.Id,
		"type":   created.Type,
		"role":   created.Role,
	}
	if created.EmailAddress != "" {
		result["email"] = created.EmailAddress
	}
	if created.Domain != "" {
		result["domain"] = created.Domain
	}

	return p.Print(result)
}

func runDriveUnshare(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	permissionID, _ := cmd.Flags().GetString("permission-id")

	err = svc.Permissions.Delete(fileID, permissionID).
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to remove permission: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":        "removed",
		"file_id":       fileID,
		"permission_id": permissionID,
	})
}

func runDrivePermission(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	permissionID, _ := cmd.Flags().GetString("permission-id")

	perm, err := svc.Permissions.Get(fileID, permissionID).
		SupportsAllDrives(true).
		Fields("id,type,role,emailAddress,domain,displayName").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get permission: %w", err))
	}

	result := map[string]interface{}{
		"id":   perm.Id,
		"type": perm.Type,
		"role": perm.Role,
	}
	if perm.EmailAddress != "" {
		result["email"] = perm.EmailAddress
	}
	if perm.Domain != "" {
		result["domain"] = perm.Domain
	}
	if perm.DisplayName != "" {
		result["display_name"] = perm.DisplayName
	}

	return p.Print(result)
}

func runDriveUpdatePermission(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	permissionID, _ := cmd.Flags().GetString("permission-id")
	role, _ := cmd.Flags().GetString("role")

	perm := &drive.Permission{Role: role}

	call := svc.Permissions.Update(fileID, permissionID, perm).
		SupportsAllDrives(true).
		Fields("id,type,role,emailAddress,domain,displayName")

	if role == "owner" {
		call = call.TransferOwnership(true)
	}

	updated, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update permission: %w", err))
	}

	result := map[string]interface{}{
		"status": "updated",
		"id":     updated.Id,
		"type":   updated.Type,
		"role":   updated.Role,
	}
	if updated.EmailAddress != "" {
		result["email"] = updated.EmailAddress
	}

	return p.Print(result)
}

// --- Revisions ---

func runDriveRevisions(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")

	resp, err := svc.Revisions.List(fileID).
		Fields("revisions(id,mimeType,modifiedTime,size,lastModifyingUser(displayName,emailAddress),originalFilename,keepForever)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list revisions: %w", err))
	}

	revisions := make([]map[string]interface{}, 0, len(resp.Revisions))
	for _, rev := range resp.Revisions {
		revInfo := map[string]interface{}{
			"id": rev.Id,
		}
		if rev.MimeType != "" {
			revInfo["mime_type"] = rev.MimeType
		}
		if rev.ModifiedTime != "" {
			revInfo["modified"] = rev.ModifiedTime
		}
		if rev.Size > 0 {
			revInfo["size"] = rev.Size
		}
		if rev.OriginalFilename != "" {
			revInfo["original_filename"] = rev.OriginalFilename
		}
		if rev.LastModifyingUser != nil {
			revInfo["last_modifying_user"] = map[string]interface{}{
				"name":  rev.LastModifyingUser.DisplayName,
				"email": rev.LastModifyingUser.EmailAddress,
			}
		}
		revInfo["keep_forever"] = rev.KeepForever
		revisions = append(revisions, revInfo)
	}

	return p.Print(map[string]interface{}{
		"file_id":   fileID,
		"revisions": revisions,
		"count":     len(revisions),
	})
}

func runDriveRevision(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	revisionID, _ := cmd.Flags().GetString("revision-id")

	rev, err := svc.Revisions.Get(fileID, revisionID).
		Fields("id,mimeType,modifiedTime,size,lastModifyingUser(displayName,emailAddress),originalFilename,keepForever").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get revision: %w", err))
	}

	result := map[string]interface{}{
		"id":           rev.Id,
		"keep_forever": rev.KeepForever,
	}
	if rev.MimeType != "" {
		result["mime_type"] = rev.MimeType
	}
	if rev.ModifiedTime != "" {
		result["modified"] = rev.ModifiedTime
	}
	if rev.Size > 0 {
		result["size"] = rev.Size
	}
	if rev.OriginalFilename != "" {
		result["original_filename"] = rev.OriginalFilename
	}
	if rev.LastModifyingUser != nil {
		result["last_modifying_user"] = map[string]interface{}{
			"name":  rev.LastModifyingUser.DisplayName,
			"email": rev.LastModifyingUser.EmailAddress,
		}
	}

	return p.Print(result)
}

func runDriveDeleteRevision(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	revisionID, _ := cmd.Flags().GetString("revision-id")

	err = svc.Revisions.Delete(fileID, revisionID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete revision: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"file_id":     fileID,
		"revision_id": revisionID,
	})
}

// --- Replies ---

func runDriveReplies(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	commentID, _ := cmd.Flags().GetString("comment-id")

	resp, err := svc.Replies.List(fileID, commentID).
		Fields("replies(id,content,author(displayName,emailAddress),createdTime,modifiedTime,action)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list replies: %w", err))
	}

	replies := make([]map[string]interface{}, 0, len(resp.Replies))
	for _, reply := range resp.Replies {
		r := map[string]interface{}{
			"id":      reply.Id,
			"content": reply.Content,
			"created": reply.CreatedTime,
		}
		if reply.ModifiedTime != "" {
			r["modified"] = reply.ModifiedTime
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

	return p.Print(map[string]interface{}{
		"file_id":    fileID,
		"comment_id": commentID,
		"replies":    replies,
		"count":      len(replies),
	})
}

func runDriveReply(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	commentID, _ := cmd.Flags().GetString("comment-id")
	content, _ := cmd.Flags().GetString("content")

	reply := &drive.Reply{Content: content}

	created, err := svc.Replies.Create(fileID, commentID, reply).
		Fields("id,content,author(displayName,emailAddress),createdTime").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create reply: %w", err))
	}

	result := map[string]interface{}{
		"status":  "created",
		"id":      created.Id,
		"content": created.Content,
		"created": created.CreatedTime,
	}
	if created.Author != nil {
		result["author"] = map[string]interface{}{
			"name":  created.Author.DisplayName,
			"email": created.Author.EmailAddress,
		}
	}

	return p.Print(result)
}

func runDriveGetReply(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	commentID, _ := cmd.Flags().GetString("comment-id")
	replyID, _ := cmd.Flags().GetString("reply-id")

	reply, err := svc.Replies.Get(fileID, commentID, replyID).
		Fields("id,content,author(displayName,emailAddress),createdTime,modifiedTime,action").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get reply: %w", err))
	}

	result := map[string]interface{}{
		"id":      reply.Id,
		"content": reply.Content,
		"created": reply.CreatedTime,
	}
	if reply.ModifiedTime != "" {
		result["modified"] = reply.ModifiedTime
	}
	if reply.Author != nil {
		result["author"] = map[string]interface{}{
			"name":  reply.Author.DisplayName,
			"email": reply.Author.EmailAddress,
		}
	}
	if reply.Action != "" {
		result["action"] = reply.Action
	}

	return p.Print(result)
}

func runDriveDeleteReply(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	commentID, _ := cmd.Flags().GetString("comment-id")
	replyID, _ := cmd.Flags().GetString("reply-id")

	err = svc.Replies.Delete(fileID, commentID, replyID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete reply: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "deleted",
		"file_id":    fileID,
		"comment_id": commentID,
		"reply_id":   replyID,
	})
}

// --- Comments (single) ---

func runDriveComment(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	commentID, _ := cmd.Flags().GetString("comment-id")

	comment, err := svc.Comments.Get(fileID, commentID).
		Fields("id,content,author(displayName,emailAddress),createdTime,modifiedTime,resolved,quotedFileContent(value),replies(id,content,author(displayName,emailAddress),createdTime,action)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get comment: %w", err))
	}

	result := map[string]interface{}{
		"id":       comment.Id,
		"content":  comment.Content,
		"created":  comment.CreatedTime,
		"resolved": comment.Resolved,
	}
	if comment.ModifiedTime != "" {
		result["modified"] = comment.ModifiedTime
	}
	if comment.Author != nil {
		result["author"] = map[string]interface{}{
			"name":  comment.Author.DisplayName,
			"email": comment.Author.EmailAddress,
		}
	}
	if comment.QuotedFileContent != nil && comment.QuotedFileContent.Value != "" {
		result["quoted_text"] = comment.QuotedFileContent.Value
	}
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
		result["replies"] = replies
	}

	return p.Print(result)
}

func runDriveAddComment(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	content, _ := cmd.Flags().GetString("content")

	comment := &drive.Comment{Content: content}

	created, err := svc.Comments.Create(fileID, comment).
		Fields("id,content,author(displayName,emailAddress),createdTime").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add comment: %w", err))
	}

	result := map[string]interface{}{
		"status":  "created",
		"id":      created.Id,
		"content": created.Content,
		"created": created.CreatedTime,
	}
	if created.Author != nil {
		result["author"] = map[string]interface{}{
			"name":  created.Author.DisplayName,
			"email": created.Author.EmailAddress,
		}
	}

	return p.Print(result)
}

func runDriveDeleteComment(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	commentID, _ := cmd.Flags().GetString("comment-id")

	err = svc.Comments.Delete(fileID, commentID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete comment: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "deleted",
		"file_id":    fileID,
		"comment_id": commentID,
	})
}

// --- Files ---

func runDriveExport(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	mimeType, _ := cmd.Flags().GetString("mime-type")
	outputPath, _ := cmd.Flags().GetString("output")

	exportResp, err := svc.Files.Export(fileID, mimeType).Download()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to export file: %w", err))
	}
	defer exportResp.Body.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create output file: %w", err))
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, exportResp.Body)
	if err != nil {
		outFile.Close()
		os.Remove(outputPath)
		return p.PrintError(fmt.Errorf("failed to write file: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":    "exported",
		"file_id":   fileID,
		"mime_type": mimeType,
		"output":    outputPath,
		"size":      written,
	})
}

func runDriveEmptyTrash(cmd *cobra.Command, args []string) error {
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

	err = svc.Files.EmptyTrash().Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to empty trash: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "trash_emptied",
	})
}

func runDriveUpdate(cmd *cobra.Command, args []string) error {
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

	fileID, _ := cmd.Flags().GetString("file-id")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	starred, _ := cmd.Flags().GetBool("starred")
	trashed, _ := cmd.Flags().GetBool("trashed")

	file := &drive.File{}
	var forceSendFields []string

	if name != "" {
		file.Name = name
	}
	if description != "" {
		file.Description = description
	}
	if cmd.Flags().Changed("starred") {
		file.Starred = starred
		forceSendFields = append(forceSendFields, "Starred")
	}
	if cmd.Flags().Changed("trashed") {
		file.Trashed = trashed
		forceSendFields = append(forceSendFields, "Trashed")
	}
	if len(forceSendFields) > 0 {
		file.ForceSendFields = forceSendFields
	}

	updated, err := svc.Files.Update(fileID, file).
		SupportsAllDrives(true).
		Fields("id,name,mimeType,description,starred,trashed,modifiedTime,webViewLink").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update file: %w", err))
	}

	result := map[string]interface{}{
		"status":   "updated",
		"id":       updated.Id,
		"name":     updated.Name,
		"starred":  updated.Starred,
		"trashed":  updated.Trashed,
		"modified": updated.ModifiedTime,
	}
	if updated.Description != "" {
		result["description"] = updated.Description
	}
	if updated.WebViewLink != "" {
		result["web_link"] = updated.WebViewLink
	}

	return p.Print(result)
}

// --- Shared Drives ---

func runDriveSharedDrives(cmd *cobra.Command, args []string) error {
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

	maxResults, _ := cmd.Flags().GetInt64("max")
	query, _ := cmd.Flags().GetString("query")

	call := svc.Drives.List().
		PageSize(maxResults).
		Fields("drives(id,name,createdTime,hidden)")

	if query != "" {
		call = call.Q(query)
	}

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list shared drives: %w", err))
	}

	drives := make([]map[string]interface{}, 0, len(resp.Drives))
	for _, d := range resp.Drives {
		driveInfo := map[string]interface{}{
			"id":   d.Id,
			"name": d.Name,
		}
		if d.CreatedTime != "" {
			driveInfo["created"] = d.CreatedTime
		}
		drives = append(drives, driveInfo)
	}

	return p.Print(map[string]interface{}{
		"drives": drives,
		"count":  len(drives),
	})
}

func runDriveSharedDrive(cmd *cobra.Command, args []string) error {
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

	driveID, _ := cmd.Flags().GetString("id")

	d, err := svc.Drives.Get(driveID).
		Fields("id,name,createdTime,hidden,colorRgb,restrictions").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get shared drive: %w", err))
	}

	result := map[string]interface{}{
		"id":   d.Id,
		"name": d.Name,
	}
	if d.CreatedTime != "" {
		result["created"] = d.CreatedTime
	}
	if d.ColorRgb != "" {
		result["color"] = d.ColorRgb
	}

	return p.Print(result)
}

func runDriveCreateDrive(cmd *cobra.Command, args []string) error {
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
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())

	d := &drive.Drive{Name: name}

	created, err := svc.Drives.Create(requestID, d).
		Fields("id,name,createdTime").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create shared drive: %w", err))
	}

	result := map[string]interface{}{
		"status": "created",
		"id":     created.Id,
		"name":   created.Name,
	}
	if created.CreatedTime != "" {
		result["created"] = created.CreatedTime
	}

	return p.Print(result)
}

func runDriveDeleteDrive(cmd *cobra.Command, args []string) error {
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

	driveID, _ := cmd.Flags().GetString("id")

	err = svc.Drives.Delete(driveID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete shared drive: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"id":     driveID,
	})
}

func runDriveUpdateDrive(cmd *cobra.Command, args []string) error {
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

	driveID, _ := cmd.Flags().GetString("id")
	name, _ := cmd.Flags().GetString("name")

	d := &drive.Drive{}
	if name != "" {
		d.Name = name
	}

	updated, err := svc.Drives.Update(driveID, d).
		Fields("id,name").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update shared drive: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "updated",
		"id":     updated.Id,
		"name":   updated.Name,
	})
}

// --- Other ---

func runDriveAbout(cmd *cobra.Command, args []string) error {
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

	about, err := svc.About.Get().
		Fields("user(displayName,emailAddress),storageQuota(limit,usage,usageInDrive,usageInDriveTrash)").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get drive info: %w", err))
	}

	result := map[string]interface{}{}
	if about.User != nil {
		result["user"] = map[string]interface{}{
			"name":  about.User.DisplayName,
			"email": about.User.EmailAddress,
		}
	}
	if about.StorageQuota != nil {
		result["storage_quota"] = map[string]interface{}{
			"limit":          about.StorageQuota.Limit,
			"usage":          about.StorageQuota.Usage,
			"usage_in_drive": about.StorageQuota.UsageInDrive,
			"usage_in_trash": about.StorageQuota.UsageInDriveTrash,
		}
	}

	return p.Print(result)
}

func runDriveActivity(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.DriveActivity()
	if err != nil {
		return p.PrintError(err)
	}

	itemID, _ := cmd.Flags().GetString("item-id")
	folderID, _ := cmd.Flags().GetString("folder-id")
	filter, _ := cmd.Flags().GetString("filter")
	days, _ := cmd.Flags().GetInt("days")
	maxResults, _ := cmd.Flags().GetInt64("max")
	pageToken, _ := cmd.Flags().GetString("page-token")
	noConsolidation, _ := cmd.Flags().GetBool("no-consolidation")

	// Validate mutual exclusivity
	if itemID != "" && folderID != "" {
		return p.PrintError(fmt.Errorf("--item-id and --folder-id are mutually exclusive"))
	}

	// Validate --days
	if days < 0 {
		return p.PrintError(fmt.Errorf("--days must be a positive number"))
	}

	req := &driveactivity.QueryDriveActivityRequest{
		PageSize: maxResults,
	}

	if itemID != "" {
		req.ItemName = "items/" + itemID
	}
	if folderID != "" {
		req.AncestorName = "items/" + folderID
	}

	// Build filter: combine --days with --filter
	var filterParts []string
	if days > 0 {
		cutoff := time.Now().AddDate(0, 0, -days)
		filterParts = append(filterParts, fmt.Sprintf("time >= \"%s\"", cutoff.Format(time.RFC3339)))
	}
	if filter != "" {
		filterParts = append(filterParts, filter)
	}
	if len(filterParts) > 0 {
		combined := filterParts[0]
		for i := 1; i < len(filterParts); i++ {
			combined += " AND " + filterParts[i]
		}
		req.Filter = combined
	}

	if pageToken != "" {
		req.PageToken = pageToken
	}

	if noConsolidation {
		req.ConsolidationStrategy = &driveactivity.ConsolidationStrategy{
			None: &driveactivity.NoConsolidation{},
		}
	}

	resp, err := svc.Activity.Query(req).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to query drive activity: %w", err))
	}

	activities := make([]map[string]interface{}, 0, len(resp.Activities))
	for _, a := range resp.Activities {
		activity := formatDriveActivity(a)
		activities = append(activities, activity)
	}

	result := map[string]interface{}{
		"activities": activities,
		"count":      len(activities),
	}
	if resp.NextPageToken != "" {
		result["next_page_token"] = resp.NextPageToken
	}

	return p.Print(result)
}

func formatDriveActivity(a *driveactivity.DriveActivity) map[string]interface{} {
	activity := map[string]interface{}{}

	// Timestamp
	if a.Timestamp != "" {
		activity["timestamp"] = a.Timestamp
	}
	if a.TimeRange != nil {
		tr := map[string]interface{}{}
		if a.TimeRange.StartTime != "" {
			tr["start"] = a.TimeRange.StartTime
		}
		if a.TimeRange.EndTime != "" {
			tr["end"] = a.TimeRange.EndTime
		}
		activity["time_range"] = tr
	}

	// Primary action
	if a.PrimaryActionDetail != nil {
		activity["primary_action"] = formatActionDetail(a.PrimaryActionDetail)
	}

	// All actions (if consolidated)
	if len(a.Actions) > 1 {
		actions := make([]map[string]interface{}, 0, len(a.Actions))
		for _, action := range a.Actions {
			act := map[string]interface{}{}
			if action.Detail != nil {
				act["detail"] = formatActionDetail(action.Detail)
			}
			if action.Timestamp != "" {
				act["timestamp"] = action.Timestamp
			}
			if action.TimeRange != nil {
				tr := map[string]interface{}{}
				if action.TimeRange.StartTime != "" {
					tr["start"] = action.TimeRange.StartTime
				}
				if action.TimeRange.EndTime != "" {
					tr["end"] = action.TimeRange.EndTime
				}
				act["time_range"] = tr
			}
			if action.Actor != nil {
				act["actor"] = formatActor(action.Actor)
			}
			if action.Target != nil {
				act["target"] = formatTarget(action.Target)
			}
			actions = append(actions, act)
		}
		activity["actions"] = actions
	}

	// Actors
	if len(a.Actors) > 0 {
		actors := make([]map[string]interface{}, 0, len(a.Actors))
		for _, actor := range a.Actors {
			actors = append(actors, formatActor(actor))
		}
		activity["actors"] = actors
	}

	// Targets
	if len(a.Targets) > 0 {
		targets := make([]map[string]interface{}, 0, len(a.Targets))
		for _, target := range a.Targets {
			targets = append(targets, formatTarget(target))
		}
		activity["targets"] = targets
	}

	return activity
}

func formatActionDetail(d *driveactivity.ActionDetail) map[string]interface{} {
	result := map[string]interface{}{}

	if d.Create != nil {
		create := map[string]interface{}{}
		if d.Create.Copy != nil {
			create["method"] = "copy"
		} else if d.Create.Upload != nil {
			create["method"] = "upload"
		} else if d.Create.New != nil {
			create["method"] = "new"
		}
		result["type"] = "create"
		result["create"] = create
	} else if d.Edit != nil {
		result["type"] = "edit"
	} else if d.Move != nil {
		move := map[string]interface{}{}
		if len(d.Move.AddedParents) > 0 {
			added := make([]string, 0, len(d.Move.AddedParents))
			for _, p := range d.Move.AddedParents {
				if p.DriveItem != nil {
					added = append(added, p.DriveItem.Title)
				} else if p.Drive != nil {
					added = append(added, p.Drive.Title)
				}
			}
			move["added_parents"] = added
		}
		if len(d.Move.RemovedParents) > 0 {
			removed := make([]string, 0, len(d.Move.RemovedParents))
			for _, p := range d.Move.RemovedParents {
				if p.DriveItem != nil {
					removed = append(removed, p.DriveItem.Title)
				} else if p.Drive != nil {
					removed = append(removed, p.Drive.Title)
				}
			}
			move["removed_parents"] = removed
		}
		result["type"] = "move"
		result["move"] = move
	} else if d.Rename != nil {
		result["type"] = "rename"
		result["rename"] = map[string]interface{}{
			"old_title": d.Rename.OldTitle,
			"new_title": d.Rename.NewTitle,
		}
	} else if d.Delete != nil {
		result["type"] = "delete"
		result["delete"] = map[string]interface{}{
			"delete_type": d.Delete.Type,
		}
	} else if d.Restore != nil {
		result["type"] = "restore"
		result["restore"] = map[string]interface{}{
			"restore_type": d.Restore.Type,
		}
	} else if d.Comment != nil {
		comment := map[string]interface{}{}
		if d.Comment.Post != nil {
			comment["subtype"] = "post"
			if d.Comment.Post.Subtype != "" {
				comment["post_subtype"] = d.Comment.Post.Subtype
			}
		} else if d.Comment.Assignment != nil {
			comment["subtype"] = "assignment"
			if d.Comment.Assignment.Subtype != "" {
				comment["assignment_subtype"] = d.Comment.Assignment.Subtype
			}
			if d.Comment.Assignment.AssignedUser != nil {
				comment["assigned_user"] = formatUser(d.Comment.Assignment.AssignedUser)
			}
		} else if d.Comment.Suggestion != nil {
			comment["subtype"] = "suggestion"
			if d.Comment.Suggestion.Subtype != "" {
				comment["suggestion_subtype"] = d.Comment.Suggestion.Subtype
			}
		}
		if len(d.Comment.MentionedUsers) > 0 {
			mentioned := make([]map[string]interface{}, 0, len(d.Comment.MentionedUsers))
			for _, u := range d.Comment.MentionedUsers {
				mentioned = append(mentioned, formatUser(u))
			}
			comment["mentioned_users"] = mentioned
		}
		result["type"] = "comment"
		result["comment"] = comment
	} else if d.PermissionChange != nil {
		pc := map[string]interface{}{}
		if len(d.PermissionChange.AddedPermissions) > 0 {
			added := make([]map[string]interface{}, 0, len(d.PermissionChange.AddedPermissions))
			for _, p := range d.PermissionChange.AddedPermissions {
				perm := map[string]interface{}{}
				if p.Role != "" {
					perm["role"] = p.Role
				}
				if p.User != nil {
					perm["user"] = formatUser(p.User)
				}
				if p.Group != nil && p.Group.Email != "" {
					perm["group"] = p.Group.Email
				}
				if p.Domain != nil && p.Domain.Name != "" {
					perm["domain"] = p.Domain.Name
				}
				if p.Anyone != nil {
					perm["anyone"] = true
				}
				added = append(added, perm)
			}
			pc["added"] = added
		}
		if len(d.PermissionChange.RemovedPermissions) > 0 {
			removed := make([]map[string]interface{}, 0, len(d.PermissionChange.RemovedPermissions))
			for _, p := range d.PermissionChange.RemovedPermissions {
				perm := map[string]interface{}{}
				if p.Role != "" {
					perm["role"] = p.Role
				}
				if p.User != nil {
					perm["user"] = formatUser(p.User)
				}
				if p.Group != nil && p.Group.Email != "" {
					perm["group"] = p.Group.Email
				}
				if p.Domain != nil && p.Domain.Name != "" {
					perm["domain"] = p.Domain.Name
				}
				if p.Anyone != nil {
					perm["anyone"] = true
				}
				removed = append(removed, perm)
			}
			pc["removed"] = removed
		}
		result["type"] = "permission_change"
		result["permission_change"] = pc
	} else if d.SettingsChange != nil {
		sc := map[string]interface{}{}
		if len(d.SettingsChange.RestrictionChanges) > 0 {
			changes := make([]map[string]interface{}, 0, len(d.SettingsChange.RestrictionChanges))
			for _, rc := range d.SettingsChange.RestrictionChanges {
				changes = append(changes, map[string]interface{}{
					"feature":         rc.Feature,
					"new_restriction": rc.NewRestriction,
				})
			}
			sc["restriction_changes"] = changes
		}
		result["type"] = "settings_change"
		result["settings_change"] = sc
	} else if d.DlpChange != nil {
		result["type"] = "dlp_change"
		result["dlp_change"] = map[string]interface{}{
			"dlp_type": d.DlpChange.Type,
		}
	} else if d.Reference != nil {
		result["type"] = "reference"
		result["reference"] = map[string]interface{}{
			"reference_type": d.Reference.Type,
		}
	} else if d.AppliedLabelChange != nil {
		lc := map[string]interface{}{}
		if len(d.AppliedLabelChange.Changes) > 0 {
			changes := make([]map[string]interface{}, 0, len(d.AppliedLabelChange.Changes))
			for _, c := range d.AppliedLabelChange.Changes {
				change := map[string]interface{}{}
				if c.Label != "" {
					change["label"] = c.Label
				}
				if c.Title != "" {
					change["title"] = c.Title
				}
				if len(c.Types) > 0 {
					change["types"] = c.Types
				}
				changes = append(changes, change)
			}
			lc["changes"] = changes
		}
		result["type"] = "label_change"
		result["label_change"] = lc
	}

	return result
}

func formatActor(a *driveactivity.Actor) map[string]interface{} {
	result := map[string]interface{}{}

	if a.User != nil {
		result["type"] = "user"
		result["user"] = formatUser(a.User)
	} else if a.Administrator != nil {
		result["type"] = "administrator"
	} else if a.Anonymous != nil {
		result["type"] = "anonymous"
	} else if a.System != nil {
		result["type"] = "system"
		if a.System.Type != "" {
			result["system_type"] = a.System.Type
		}
	} else if a.Impersonation != nil {
		result["type"] = "impersonation"
		if a.Impersonation.ImpersonatedUser != nil {
			result["impersonated_user"] = formatUser(a.Impersonation.ImpersonatedUser)
		}
	}

	return result
}

func formatUser(u *driveactivity.User) map[string]interface{} {
	result := map[string]interface{}{}
	if u.KnownUser != nil {
		result["type"] = "known_user"
		if u.KnownUser.PersonName != "" {
			result["person_name"] = u.KnownUser.PersonName
		}
		if u.KnownUser.IsCurrentUser {
			result["is_current_user"] = true
		}
	} else if u.DeletedUser != nil {
		result["type"] = "deleted_user"
	} else if u.UnknownUser != nil {
		result["type"] = "unknown_user"
	}
	return result
}

func formatTarget(t *driveactivity.Target) map[string]interface{} {
	result := map[string]interface{}{}

	if t.DriveItem != nil {
		result["type"] = "drive_item"
		item := map[string]interface{}{}
		if t.DriveItem.Name != "" {
			item["name"] = t.DriveItem.Name
		}
		if t.DriveItem.Title != "" {
			item["title"] = t.DriveItem.Title
		}
		if t.DriveItem.MimeType != "" {
			item["mime_type"] = t.DriveItem.MimeType
		}
		if t.DriveItem.Owner != nil {
			owner := map[string]interface{}{}
			if t.DriveItem.Owner.User != nil {
				owner["user"] = formatUser(t.DriveItem.Owner.User)
			}
			if t.DriveItem.Owner.Drive != nil {
				owner["drive"] = map[string]interface{}{
					"name":  t.DriveItem.Owner.Drive.Name,
					"title": t.DriveItem.Owner.Drive.Title,
				}
			}
			if t.DriveItem.Owner.Domain != nil && t.DriveItem.Owner.Domain.Name != "" {
				owner["domain"] = t.DriveItem.Owner.Domain.Name
			}
			item["owner"] = owner
		}
		if t.DriveItem.DriveFolder != nil {
			item["item_type"] = "folder"
			if t.DriveItem.DriveFolder.Type != "" {
				item["folder_type"] = t.DriveItem.DriveFolder.Type
			}
		} else if t.DriveItem.DriveFile != nil {
			item["item_type"] = "file"
		}
		result["drive_item"] = item
	} else if t.Drive != nil {
		result["type"] = "shared_drive"
		sd := map[string]interface{}{}
		if t.Drive.Name != "" {
			sd["name"] = t.Drive.Name
		}
		if t.Drive.Title != "" {
			sd["title"] = t.Drive.Title
		}
		result["shared_drive"] = sd
	} else if t.FileComment != nil {
		result["type"] = "file_comment"
		fc := map[string]interface{}{}
		if t.FileComment.LegacyCommentId != "" {
			fc["comment_id"] = t.FileComment.LegacyCommentId
		}
		if t.FileComment.LinkToDiscussion != "" {
			fc["link"] = t.FileComment.LinkToDiscussion
		}
		if t.FileComment.Parent != nil {
			parent := map[string]interface{}{}
			if t.FileComment.Parent.Name != "" {
				parent["name"] = t.FileComment.Parent.Name
			}
			if t.FileComment.Parent.Title != "" {
				parent["title"] = t.FileComment.Parent.Title
			}
			fc["parent"] = parent
		}
		result["file_comment"] = fc
	}

	return result
}

func runDriveChanges(cmd *cobra.Command, args []string) error {
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

	maxResults, _ := cmd.Flags().GetInt64("max")
	pageToken, _ := cmd.Flags().GetString("page-token")

	// If no page token provided, get the start page token
	if pageToken == "" {
		startToken, err := svc.Changes.GetStartPageToken().
			SupportsAllDrives(true).
			Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get start page token: %w", err))
		}
		pageToken = startToken.StartPageToken
	}

	resp, err := svc.Changes.List(pageToken).
		PageSize(maxResults).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Fields("changes(fileId,file(id,name,mimeType,modifiedTime),removed,time,changeType),newStartPageToken,nextPageToken").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list changes: %w", err))
	}

	changes := make([]map[string]interface{}, 0, len(resp.Changes))
	for _, change := range resp.Changes {
		c := map[string]interface{}{
			"file_id": change.FileId,
			"removed": change.Removed,
		}
		if change.Time != "" {
			c["time"] = change.Time
		}
		if change.ChangeType != "" {
			c["change_type"] = change.ChangeType
		}
		if change.File != nil {
			c["file"] = map[string]interface{}{
				"id":        change.File.Id,
				"name":      change.File.Name,
				"mime_type": change.File.MimeType,
				"modified":  change.File.ModifiedTime,
			}
		}
		changes = append(changes, c)
	}

	result := map[string]interface{}{
		"changes": changes,
		"count":   len(changes),
	}
	if resp.NewStartPageToken != "" {
		result["new_start_page_token"] = resp.NewStartPageToken
	}
	if resp.NextPageToken != "" {
		result["next_page_token"] = resp.NextPageToken
	}

	return p.Print(result)
}
