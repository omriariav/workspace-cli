package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/markdown"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// flattenTabs recursively flattens a tab tree into a flat list.
func flattenTabs(tabs []*docs.Tab) []*docs.Tab {
	var result []*docs.Tab
	for _, tab := range tabs {
		result = append(result, tab)
		if len(tab.ChildTabs) > 0 {
			result = append(result, flattenTabs(tab.ChildTabs)...)
		}
	}
	return result
}

// resolveTabID resolves a --tab query (ID or title) to a tab ID.
// Matches by exact ID first, then case-insensitive title.
func resolveTabID(tabs []*docs.Tab, query string) (string, error) {
	flat := flattenTabs(tabs)

	// Match by exact ID
	for _, tab := range flat {
		if tab.TabProperties != nil && tab.TabProperties.TabId == query {
			return tab.TabProperties.TabId, nil
		}
	}

	// Match by case-insensitive title
	var matches []*docs.Tab
	queryLower := strings.ToLower(query)
	for _, tab := range flat {
		if tab.TabProperties != nil && strings.ToLower(tab.TabProperties.Title) == queryLower {
			matches = append(matches, tab)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no tab found matching %q", query)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple tabs match title %q; use tab ID instead", query)
	}
	return matches[0].TabProperties.TabId, nil
}

// getTabBody returns the Body for a given tabID. If tabID is "", returns the
// first tab's body from doc.Tabs (or doc.Body for backward compat).
func getTabBody(doc *docs.Document, tabID string) (*docs.Body, error) {
	if len(doc.Tabs) > 0 {
		flat := flattenTabs(doc.Tabs)
		if tabID == "" {
			// Return first tab
			if flat[0].DocumentTab != nil {
				return flat[0].DocumentTab.Body, nil
			}
			return nil, fmt.Errorf("first tab has no content")
		}
		for _, tab := range flat {
			if tab.TabProperties != nil && tab.TabProperties.TabId == tabID {
				if tab.DocumentTab != nil {
					return tab.DocumentTab.Body, nil
				}
				return nil, fmt.Errorf("tab %q has no content", tabID)
			}
		}
		return nil, fmt.Errorf("tab %q not found", tabID)
	}
	// Fallback: no tabs populated, use doc.Body
	if tabID != "" {
		return nil, fmt.Errorf("tab data not available; re-fetch with IncludeTabsContent")
	}
	return doc.Body, nil
}

// resolveTabFromFlags reads --tab and optionally --tab-index flags and returns the resolved tab ID.
func resolveTabFromFlags(cmd *cobra.Command, doc *docs.Document) (string, error) {
	tabQuery, _ := cmd.Flags().GetString("tab")

	// Check for --tab-index (only on read command)
	tabIndex := int64(-1)
	if f := cmd.Flags().Lookup("tab-index"); f != nil {
		tabIndex, _ = cmd.Flags().GetInt64("tab-index")
	}

	if tabQuery != "" && tabIndex >= 0 {
		return "", fmt.Errorf("cannot use both --tab and --tab-index")
	}

	if tabIndex >= 0 {
		if len(doc.Tabs) == 0 {
			return "", fmt.Errorf("document has no tabs data")
		}
		flat := flattenTabs(doc.Tabs)
		if tabIndex >= int64(len(flat)) {
			return "", fmt.Errorf("tab index %d out of range (document has %d tabs)", tabIndex, len(flat))
		}
		return flat[tabIndex].TabProperties.TabId, nil
	}

	if tabQuery != "" {
		if len(doc.Tabs) == 0 {
			return "", fmt.Errorf("document has no tabs data")
		}
		return resolveTabID(doc.Tabs, tabQuery)
	}

	return "", nil
}

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Manage Google Docs",
	Long:  "Commands for interacting with Google Docs documents.",
}

var docsReadCmd = &cobra.Command{
	Use:   "read <document-id>",
	Short: "Read document content",
	Long:  "Reads and displays the text content of a Google Doc.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsRead,
}

var docsInfoCmd = &cobra.Command{
	Use:   "info <document-id>",
	Short: "Get document info",
	Long:  "Gets metadata about a Google Doc.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsInfo,
}

var docsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new document",
	Long:  "Creates a new Google Doc with optional initial content.",
	RunE:  runDocsCreate,
}

var docsAppendCmd = &cobra.Command{
	Use:   "append <document-id>",
	Short: "Append text to a document",
	Long:  "Appends text to the end of an existing Google Doc.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAppend,
}

var docsInsertCmd = &cobra.Command{
	Use:   "insert <document-id>",
	Short: "Insert text at a position",
	Long: `Inserts text at a specific position in the document.

Position is a 1-based index (1 = start of document content).
Use 'gws docs read <id> --include-formatting' to see element positions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsInsert,
}

var docsReplaceCmd = &cobra.Command{
	Use:   "replace <document-id>",
	Short: "Find and replace text",
	Long:  "Replaces all occurrences of a text string in the document.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsReplace,
}

var docsDeleteCmd = &cobra.Command{
	Use:   "delete <document-id>",
	Short: "Delete content from a document",
	Long: `Deletes content from a range of positions in the document.

Positions are 1-based indices. Use 'gws docs read <id> --include-formatting' to see positions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsDelete,
}

var docsAddTableCmd = &cobra.Command{
	Use:   "add-table <document-id>",
	Short: "Add a table to the document",
	Long: `Adds a table at a specified position in the document.

Position is a 1-based index. Use 'gws docs read <id> --include-formatting' to see positions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsAddTable,
}

var docsFormatCmd = &cobra.Command{
	Use:   "format <document-id>",
	Short: "Format text style",
	Long: `Applies text formatting to a range of positions in the document.

Positions are 1-based indices. Use 'gws docs read <id> --include-formatting' to see positions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsFormat,
}

var docsSetParagraphStyleCmd = &cobra.Command{
	Use:   "set-paragraph-style <document-id>",
	Short: "Set paragraph style",
	Long: `Sets paragraph style properties for a range of positions in the document.

Positions are 1-based indices. Use 'gws docs read <id> --include-formatting' to see positions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsSetParagraphStyle,
}

var docsAddListCmd = &cobra.Command{
	Use:   "add-list <document-id>",
	Short: "Add a bullet or numbered list",
	Long: `Inserts text items as a bullet or numbered list at a specified position.

Items are separated by semicolons. Position is a 1-based index.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsAddList,
}

var docsRemoveListCmd = &cobra.Command{
	Use:   "remove-list <document-id>",
	Short: "Remove list formatting",
	Long: `Removes bullet or numbered list formatting from a range of positions.

Positions are 1-based indices. Use 'gws docs read <id> --include-formatting' to see positions.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsRemoveList,
}

var docsTrashCmd = &cobra.Command{
	Use:   "trash <document-id>",
	Short: "Trash or permanently delete a document",
	Long: `Moves a Google Doc to the trash via the Drive API.

By default, moves the document to trash. Use --permanent to permanently delete.

Warning: --permanent bypasses trash and cannot be undone.

Examples:
  gws docs trash 1abc123xyz
  gws docs trash 1abc123xyz --permanent`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsTrash,
}

// Tab management commands
var docsAddTabCmd = &cobra.Command{
	Use:   "add-tab <document-id>",
	Short: "Add a new tab to the document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAddTab,
}

var docsDeleteTabCmd = &cobra.Command{
	Use:   "delete-tab <document-id>",
	Short: "Delete a tab from the document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsDeleteTab,
}

var docsRenameTabCmd = &cobra.Command{
	Use:   "rename-tab <document-id>",
	Short: "Rename a tab in the document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsRenameTab,
}

// Image command
var docsAddImageCmd = &cobra.Command{
	Use:   "add-image <document-id>",
	Short: "Insert an image into the document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAddImage,
}

// Table operation commands
var docsInsertTableRowCmd = &cobra.Command{
	Use:   "insert-table-row <document-id>",
	Short: "Insert a row into a table",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsInsertTableRow,
}

var docsDeleteTableRowCmd = &cobra.Command{
	Use:   "delete-table-row <document-id>",
	Short: "Delete a row from a table",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsDeleteTableRow,
}

var docsInsertTableColCmd = &cobra.Command{
	Use:   "insert-table-col <document-id>",
	Short: "Insert a column into a table",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsInsertTableCol,
}

var docsDeleteTableColCmd = &cobra.Command{
	Use:   "delete-table-col <document-id>",
	Short: "Delete a column from a table",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsDeleteTableCol,
}

var docsMergeCellsCmd = &cobra.Command{
	Use:   "merge-cells <document-id>",
	Short: "Merge table cells",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsMergeCells,
}

var docsUnmergeCellsCmd = &cobra.Command{
	Use:   "unmerge-cells <document-id>",
	Short: "Unmerge table cells",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsUnmergeCells,
}

var docsPinRowsCmd = &cobra.Command{
	Use:   "pin-rows <document-id>",
	Short: "Pin header rows in a table",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsPinRows,
}

// Page/section break commands
var docsPageBreakCmd = &cobra.Command{
	Use:   "page-break <document-id>",
	Short: "Insert a page break",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsPageBreak,
}

var docsSectionBreakCmd = &cobra.Command{
	Use:   "section-break <document-id>",
	Short: "Insert a section break",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsSectionBreak,
}

// Header & footer commands
var docsAddHeaderCmd = &cobra.Command{
	Use:   "add-header <document-id>",
	Short: "Add a header to the document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAddHeader,
}

var docsDeleteHeaderCmd = &cobra.Command{
	Use:   "delete-header <document-id> <header-id>",
	Short: "Delete a header from the document",
	Args:  cobra.ExactArgs(2),
	RunE:  runDocsDeleteHeader,
}

var docsAddFooterCmd = &cobra.Command{
	Use:   "add-footer <document-id>",
	Short: "Add a footer to the document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAddFooter,
}

var docsDeleteFooterCmd = &cobra.Command{
	Use:   "delete-footer <document-id> <footer-id>",
	Short: "Delete a footer from the document",
	Args:  cobra.ExactArgs(2),
	RunE:  runDocsDeleteFooter,
}

// Named range commands
var docsAddNamedRangeCmd = &cobra.Command{
	Use:   "add-named-range <document-id>",
	Short: "Create a named range",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAddNamedRange,
}

var docsDeleteNamedRangeCmd = &cobra.Command{
	Use:   "delete-named-range <document-id>",
	Short: "Delete a named range",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsDeleteNamedRange,
}

// Footnote & misc commands
var docsAddFootnoteCmd = &cobra.Command{
	Use:   "add-footnote <document-id>",
	Short: "Insert a footnote",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsAddFootnote,
}

var docsDeleteObjectCmd = &cobra.Command{
	Use:   "delete-object <document-id> <object-id>",
	Short: "Delete a positioned object",
	Args:  cobra.ExactArgs(2),
	RunE:  runDocsDeleteObject,
}

var docsReplaceImageCmd = &cobra.Command{
	Use:   "replace-image <document-id>",
	Short: "Replace an inline image",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsReplaceImage,
}

var docsReplaceNamedRangeCmd = &cobra.Command{
	Use:   "replace-named-range <document-id>",
	Short: "Replace text in a named range",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsReplaceNamedRange,
}

var docsUpdateStyleCmd = &cobra.Command{
	Use:   "update-style <document-id>",
	Short: "Update document style (margins)",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsUpdateStyle,
}

var docsUpdateSectionStyleCmd = &cobra.Command{
	Use:   "update-section-style <document-id>",
	Short: "Update section style (columns, direction)",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsUpdateSectionStyle,
}

var docsUpdateTableCellStyleCmd = &cobra.Command{
	Use:   "update-table-cell-style <document-id>",
	Short: "Update table cell style",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsUpdateTableCellStyle,
}

var docsUpdateTableColPropertiesCmd = &cobra.Command{
	Use:   "update-table-col-properties <document-id>",
	Short: "Update table column width",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsUpdateTableColProperties,
}

var docsUpdateTableRowStyleCmd = &cobra.Command{
	Use:   "update-table-row-style <document-id>",
	Short: "Update table row style",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsUpdateTableRowStyle,
}

func init() {
	rootCmd.AddCommand(docsCmd)

	// Persistent --tab flag for all subcommands
	docsCmd.PersistentFlags().String("tab", "", "Tab ID or title to target (omit for first tab)")

	// Read-only convenience flag
	docsReadCmd.Flags().Int64("tab-index", -1, "Zero-based tab index (alternative to --tab)")

	// Existing commands
	docsCmd.AddCommand(docsReadCmd)
	docsCmd.AddCommand(docsInfoCmd)
	docsCmd.AddCommand(docsCreateCmd)
	docsCmd.AddCommand(docsAppendCmd)
	docsCmd.AddCommand(docsInsertCmd)
	docsCmd.AddCommand(docsReplaceCmd)
	docsCmd.AddCommand(docsDeleteCmd)
	docsCmd.AddCommand(docsAddTableCmd)
	docsCmd.AddCommand(docsFormatCmd)
	docsCmd.AddCommand(docsSetParagraphStyleCmd)
	docsCmd.AddCommand(docsAddListCmd)
	docsCmd.AddCommand(docsRemoveListCmd)
	docsCmd.AddCommand(docsTrashCmd)

	// Tab management commands
	docsCmd.AddCommand(docsAddTabCmd)
	docsCmd.AddCommand(docsDeleteTabCmd)
	docsCmd.AddCommand(docsRenameTabCmd)

	// Image command
	docsCmd.AddCommand(docsAddImageCmd)

	// Table operation commands
	docsCmd.AddCommand(docsInsertTableRowCmd)
	docsCmd.AddCommand(docsDeleteTableRowCmd)
	docsCmd.AddCommand(docsInsertTableColCmd)
	docsCmd.AddCommand(docsDeleteTableColCmd)
	docsCmd.AddCommand(docsMergeCellsCmd)
	docsCmd.AddCommand(docsUnmergeCellsCmd)
	docsCmd.AddCommand(docsPinRowsCmd)

	// Page/section break commands
	docsCmd.AddCommand(docsPageBreakCmd)
	docsCmd.AddCommand(docsSectionBreakCmd)

	// Header & footer commands
	docsCmd.AddCommand(docsAddHeaderCmd)
	docsCmd.AddCommand(docsDeleteHeaderCmd)
	docsCmd.AddCommand(docsAddFooterCmd)
	docsCmd.AddCommand(docsDeleteFooterCmd)

	// Named range commands
	docsCmd.AddCommand(docsAddNamedRangeCmd)
	docsCmd.AddCommand(docsDeleteNamedRangeCmd)

	// Footnote & misc commands
	docsCmd.AddCommand(docsAddFootnoteCmd)
	docsCmd.AddCommand(docsDeleteObjectCmd)
	docsCmd.AddCommand(docsReplaceImageCmd)
	docsCmd.AddCommand(docsReplaceNamedRangeCmd)
	docsCmd.AddCommand(docsUpdateStyleCmd)
	docsCmd.AddCommand(docsUpdateSectionStyleCmd)
	docsCmd.AddCommand(docsUpdateTableCellStyleCmd)
	docsCmd.AddCommand(docsUpdateTableColPropertiesCmd)
	docsCmd.AddCommand(docsUpdateTableRowStyleCmd)

	// Trash flags
	docsTrashCmd.Flags().Bool("permanent", false, "Permanently delete (skip trash)")

	// Format flags
	docsFormatCmd.Flags().Int64("from", 0, "Start position (1-based index, required)")
	docsFormatCmd.Flags().Int64("to", 0, "End position (1-based index, required)")
	docsFormatCmd.Flags().Bool("bold", false, "Make text bold")
	docsFormatCmd.Flags().Bool("italic", false, "Make text italic")
	docsFormatCmd.Flags().Int64("font-size", 0, "Font size in points")
	docsFormatCmd.Flags().String("color", "", "Text color (hex, e.g., #FF0000)")
	docsFormatCmd.MarkFlagRequired("from")
	docsFormatCmd.MarkFlagRequired("to")

	// Set-paragraph-style flags
	docsSetParagraphStyleCmd.Flags().Int64("from", 0, "Start position (1-based index, required)")
	docsSetParagraphStyleCmd.Flags().Int64("to", 0, "End position (1-based index, required)")
	docsSetParagraphStyleCmd.Flags().String("alignment", "", "Paragraph alignment: START, CENTER, END, JUSTIFIED")
	docsSetParagraphStyleCmd.Flags().Float64("line-spacing", 0, "Line spacing multiplier (e.g., 1.15, 1.5, 2.0)")
	docsSetParagraphStyleCmd.MarkFlagRequired("from")
	docsSetParagraphStyleCmd.MarkFlagRequired("to")

	// Add-list flags
	docsAddListCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")
	docsAddListCmd.Flags().String("type", "bullet", "List type: bullet or numbered")
	docsAddListCmd.Flags().String("items", "", "List items separated by semicolons (required)")
	docsAddListCmd.MarkFlagRequired("items")

	// Remove-list flags
	docsRemoveListCmd.Flags().Int64("from", 0, "Start position (1-based index, required)")
	docsRemoveListCmd.Flags().Int64("to", 0, "End position (1-based index, required)")
	docsRemoveListCmd.MarkFlagRequired("from")
	docsRemoveListCmd.MarkFlagRequired("to")

	// Read flags
	docsReadCmd.Flags().Bool("include-formatting", false, "Include formatting information")

	// Create flags
	docsCreateCmd.Flags().String("title", "", "Document title (required)")
	docsCreateCmd.Flags().String("text", "", "Initial text content")
	docsCreateCmd.Flags().String("content-format", "markdown", "Content format: markdown, plaintext, or richformat")
	docsCreateCmd.MarkFlagRequired("title")

	// Append flags
	docsAppendCmd.Flags().String("text", "", "Text to append (required)")
	docsAppendCmd.Flags().Bool("newline", true, "Add newline before appending")
	docsAppendCmd.Flags().String("content-format", "markdown", "Content format: markdown, plaintext, or richformat (--newline is ignored for richformat)")
	docsAppendCmd.MarkFlagRequired("text")

	// Insert flags
	docsInsertCmd.Flags().String("text", "", "Text to insert (required)")
	docsInsertCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")
	docsInsertCmd.Flags().String("content-format", "markdown", "Content format: markdown, plaintext, or richformat (--at is ignored for richformat)")
	docsInsertCmd.MarkFlagRequired("text")

	// Replace flags
	docsReplaceCmd.Flags().String("find", "", "Text to find (required)")
	docsReplaceCmd.Flags().String("replace", "", "Replacement text (required)")
	docsReplaceCmd.Flags().Bool("match-case", true, "Case-sensitive matching")
	docsReplaceCmd.MarkFlagRequired("find")
	docsReplaceCmd.MarkFlagRequired("replace")

	// Delete flags
	docsDeleteCmd.Flags().Int64("from", 0, "Start position (1-based index, required)")
	docsDeleteCmd.Flags().Int64("to", 0, "End position (1-based index, required)")
	docsDeleteCmd.MarkFlagRequired("from")
	docsDeleteCmd.MarkFlagRequired("to")

	// Add-table flags
	docsAddTableCmd.Flags().Int64("rows", 3, "Number of rows")
	docsAddTableCmd.Flags().Int64("cols", 3, "Number of columns")
	docsAddTableCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")

	// Add-tab flags
	docsAddTabCmd.Flags().String("title", "", "Tab title (required)")
	docsAddTabCmd.Flags().Int64("index", -1, "Position index for the new tab")
	docsAddTabCmd.MarkFlagRequired("title")

	// Delete-tab flags
	docsDeleteTabCmd.Flags().String("tab-id", "", "Tab ID to delete (required)")
	docsDeleteTabCmd.MarkFlagRequired("tab-id")

	// Rename-tab flags
	docsRenameTabCmd.Flags().String("tab-id", "", "Tab ID to rename (required)")
	docsRenameTabCmd.Flags().String("title", "", "New tab title (required)")
	docsRenameTabCmd.MarkFlagRequired("tab-id")
	docsRenameTabCmd.MarkFlagRequired("title")

	// Add-image flags
	docsAddImageCmd.Flags().String("uri", "", "Image URI (required)")
	docsAddImageCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")
	docsAddImageCmd.Flags().Float64("width", 0, "Image width in points")
	docsAddImageCmd.Flags().Float64("height", 0, "Image height in points")
	docsAddImageCmd.MarkFlagRequired("uri")

	// Table cell location flags (shared pattern)
	for _, cmd := range []*cobra.Command{docsInsertTableRowCmd, docsDeleteTableRowCmd,
		docsInsertTableColCmd, docsDeleteTableColCmd, docsMergeCellsCmd, docsUnmergeCellsCmd} {
		cmd.Flags().Int64("table-start", 0, "Table start index (required)")
		cmd.Flags().Int64("row", 0, "Zero-based row index (required)")
		cmd.Flags().Int64("col", 0, "Zero-based column index (required)")
		cmd.MarkFlagRequired("table-start")
		cmd.MarkFlagRequired("row")
		cmd.MarkFlagRequired("col")
	}
	docsInsertTableRowCmd.Flags().Bool("below", true, "Insert below the reference cell")
	docsInsertTableColCmd.Flags().Bool("right", true, "Insert to the right of the reference cell")
	docsMergeCellsCmd.Flags().Int64("row-span", 1, "Number of rows to merge")
	docsMergeCellsCmd.Flags().Int64("col-span", 1, "Number of columns to merge")
	docsUnmergeCellsCmd.Flags().Int64("row-span", 1, "Number of rows to unmerge")
	docsUnmergeCellsCmd.Flags().Int64("col-span", 1, "Number of columns to unmerge")

	// Pin-rows flags
	docsPinRowsCmd.Flags().Int64("table-start", 0, "Table start index (required)")
	docsPinRowsCmd.Flags().Int64("count", 0, "Number of rows to pin (required)")
	docsPinRowsCmd.MarkFlagRequired("table-start")
	docsPinRowsCmd.MarkFlagRequired("count")

	// Page-break flags
	docsPageBreakCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")

	// Section-break flags
	docsSectionBreakCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")
	docsSectionBreakCmd.Flags().String("type", "NEXT_PAGE", "Section break type: NEXT_PAGE or CONTINUOUS")

	// Header/footer flags
	docsAddHeaderCmd.Flags().String("type", "DEFAULT", "Header type: DEFAULT")
	docsAddFooterCmd.Flags().String("type", "DEFAULT", "Footer type: DEFAULT")

	// Named range flags
	docsAddNamedRangeCmd.Flags().String("name", "", "Range name (required)")
	docsAddNamedRangeCmd.Flags().Int64("from", 0, "Start position (required)")
	docsAddNamedRangeCmd.Flags().Int64("to", 0, "End position (required)")
	docsAddNamedRangeCmd.MarkFlagRequired("name")
	docsAddNamedRangeCmd.MarkFlagRequired("from")
	docsAddNamedRangeCmd.MarkFlagRequired("to")

	docsDeleteNamedRangeCmd.Flags().String("name", "", "Named range name")
	docsDeleteNamedRangeCmd.Flags().String("id", "", "Named range ID")

	// Footnote flags
	docsAddFootnoteCmd.Flags().Int64("at", 0, "Insertion index (required)")
	docsAddFootnoteCmd.MarkFlagRequired("at")

	// Replace-image flags
	docsReplaceImageCmd.Flags().String("object-id", "", "Inline object ID (required)")
	docsReplaceImageCmd.Flags().String("uri", "", "New image URI (required)")
	docsReplaceImageCmd.MarkFlagRequired("object-id")
	docsReplaceImageCmd.MarkFlagRequired("uri")

	// Replace-named-range flags
	docsReplaceNamedRangeCmd.Flags().String("name", "", "Named range name")
	docsReplaceNamedRangeCmd.Flags().String("id", "", "Named range ID")
	docsReplaceNamedRangeCmd.Flags().String("text", "", "Replacement text (required)")
	docsReplaceNamedRangeCmd.MarkFlagRequired("text")

	// Update-style flags (document margins)
	docsUpdateStyleCmd.Flags().Float64("margin-top", -1, "Top margin in points")
	docsUpdateStyleCmd.Flags().Float64("margin-bottom", -1, "Bottom margin in points")
	docsUpdateStyleCmd.Flags().Float64("margin-left", -1, "Left margin in points")
	docsUpdateStyleCmd.Flags().Float64("margin-right", -1, "Right margin in points")

	// Update-section-style flags
	docsUpdateSectionStyleCmd.Flags().Int64("from", 0, "Start position (required)")
	docsUpdateSectionStyleCmd.Flags().Int64("to", 0, "End position (required)")
	docsUpdateSectionStyleCmd.Flags().Int64("column-count", 0, "Number of columns")
	docsUpdateSectionStyleCmd.Flags().String("content-direction", "", "Content direction: LEFT_TO_RIGHT or RIGHT_TO_LEFT")
	docsUpdateSectionStyleCmd.MarkFlagRequired("from")
	docsUpdateSectionStyleCmd.MarkFlagRequired("to")

	// Update-table-cell-style flags
	docsUpdateTableCellStyleCmd.Flags().Int64("table-start", 0, "Table start index (required)")
	docsUpdateTableCellStyleCmd.Flags().Int64("row", 0, "Zero-based row index (required)")
	docsUpdateTableCellStyleCmd.Flags().Int64("col", 0, "Zero-based column index (required)")
	docsUpdateTableCellStyleCmd.Flags().Int64("row-span", 1, "Number of rows")
	docsUpdateTableCellStyleCmd.Flags().Int64("col-span", 1, "Number of columns")
	docsUpdateTableCellStyleCmd.Flags().String("bg-color", "", "Background color (#RRGGBB)")
	docsUpdateTableCellStyleCmd.Flags().Float64("padding", -1, "Cell padding in points")
	docsUpdateTableCellStyleCmd.MarkFlagRequired("table-start")
	docsUpdateTableCellStyleCmd.MarkFlagRequired("row")
	docsUpdateTableCellStyleCmd.MarkFlagRequired("col")

	// Update-table-col-properties flags
	docsUpdateTableColPropertiesCmd.Flags().Int64("table-start", 0, "Table start index (required)")
	docsUpdateTableColPropertiesCmd.Flags().Int64("col-index", 0, "Column index (required)")
	docsUpdateTableColPropertiesCmd.Flags().Float64("width", 0, "Column width in points (required)")
	docsUpdateTableColPropertiesCmd.MarkFlagRequired("table-start")
	docsUpdateTableColPropertiesCmd.MarkFlagRequired("col-index")
	docsUpdateTableColPropertiesCmd.MarkFlagRequired("width")

	// Update-table-row-style flags
	docsUpdateTableRowStyleCmd.Flags().Int64("table-start", 0, "Table start index (required)")
	docsUpdateTableRowStyleCmd.Flags().Int64("row", 0, "Zero-based row index (required)")
	docsUpdateTableRowStyleCmd.Flags().Float64("min-height", 0, "Minimum row height in points (required)")
	docsUpdateTableRowStyleCmd.MarkFlagRequired("table-start")
	docsUpdateTableRowStyleCmd.MarkFlagRequired("row")
	docsUpdateTableRowStyleCmd.MarkFlagRequired("min-height")
}

func runDocsRead(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	includeFormatting, _ := cmd.Flags().GetBool("include-formatting")

	doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get document: %w", err))
	}

	// Resolve tab
	tabID, err := resolveTabFromFlags(cmd, doc)
	if err != nil {
		return p.PrintError(err)
	}

	body, err := getTabBody(doc, tabID)
	if err != nil {
		return p.PrintError(err)
	}

	// Extract text content
	var textBuilder strings.Builder
	if body != nil {
		extractText(body.Content, &textBuilder)
	}

	result := map[string]interface{}{
		"id":    doc.DocumentId,
		"title": doc.Title,
		"text":  textBuilder.String(),
	}

	// Add tab info when targeting a specific tab
	if tabID != "" {
		result["tab_id"] = tabID
		flat := flattenTabs(doc.Tabs)
		for _, tab := range flat {
			if tab.TabProperties != nil && tab.TabProperties.TabId == tabID {
				result["tab_title"] = tab.TabProperties.Title
				break
			}
		}
	}

	if includeFormatting {
		if body != nil {
			result["structure"] = extractStructure(body.Content)
		}
	}

	return p.Print(result)
}

func runDocsInfo(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]

	doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get document: %w", err))
	}

	result := map[string]interface{}{
		"id":    doc.DocumentId,
		"title": doc.Title,
	}

	// Document style info
	if doc.DocumentStyle != nil {
		style := map[string]interface{}{}
		if doc.DocumentStyle.PageSize != nil {
			style["page_width"] = doc.DocumentStyle.PageSize.Width
			style["page_height"] = doc.DocumentStyle.PageSize.Height
		}
		result["style"] = style
	}

	// Named styles
	if doc.NamedStyles != nil && len(doc.NamedStyles.Styles) > 0 {
		styles := make([]string, 0)
		for _, s := range doc.NamedStyles.Styles {
			styles = append(styles, s.NamedStyleType)
		}
		result["named_styles"] = styles
	}

	// Revision ID
	result["revision_id"] = doc.RevisionId

	// Tabs info
	if len(doc.Tabs) > 0 {
		flat := flattenTabs(doc.Tabs)
		tabsInfo := make([]map[string]interface{}, 0, len(flat))
		for _, tab := range flat {
			if tab.TabProperties != nil {
				tabInfo := map[string]interface{}{
					"tab_id":        tab.TabProperties.TabId,
					"title":         tab.TabProperties.Title,
					"index":         tab.TabProperties.Index,
					"nesting_level": tab.TabProperties.NestingLevel,
				}
				tabsInfo = append(tabsInfo, tabInfo)
			}
		}
		result["tabs"] = tabsInfo
	}

	return p.Print(result)
}

// extractText recursively extracts plain text from document content.
func extractText(content []*docs.StructuralElement, builder *strings.Builder) {
	for _, elem := range content {
		if elem.Paragraph != nil {
			for _, pe := range elem.Paragraph.Elements {
				if pe.TextRun != nil {
					builder.WriteString(pe.TextRun.Content)
				}
			}
		}
		if elem.Table != nil {
			for _, row := range elem.Table.TableRows {
				for _, cell := range row.TableCells {
					extractText(cell.Content, builder)
					builder.WriteString("\t")
				}
				builder.WriteString("\n")
			}
		}
		if elem.SectionBreak != nil {
			builder.WriteString("\n---\n")
		}
	}
}

// extractStructure extracts document structure information.
func extractStructure(content []*docs.StructuralElement) []map[string]interface{} {
	structure := make([]map[string]interface{}, 0)

	for _, elem := range content {
		item := map[string]interface{}{}

		if elem.Paragraph != nil {
			item["type"] = "paragraph"
			if elem.Paragraph.ParagraphStyle != nil {
				item["style"] = elem.Paragraph.ParagraphStyle.NamedStyleType
				if elem.Paragraph.ParagraphStyle.HeadingId != "" {
					item["heading_id"] = elem.Paragraph.ParagraphStyle.HeadingId
				}
			}

			// Get text content
			var text strings.Builder
			for _, pe := range elem.Paragraph.Elements {
				if pe.TextRun != nil {
					text.WriteString(pe.TextRun.Content)
				}
			}
			item["text"] = strings.TrimSpace(text.String())
		}

		if elem.Table != nil {
			item["type"] = "table"
			item["rows"] = elem.Table.Rows
			item["columns"] = elem.Table.Columns
		}

		if elem.SectionBreak != nil {
			item["type"] = "section_break"
		}

		if len(item) > 0 {
			structure = append(structure, item)
		}
	}

	return structure
}

// buildTextRequests builds Google Docs API requests based on the content format.
// For markdown and plaintext, it creates an InsertText request.
// For richformat, it parses the text as JSON Google Docs API requests.
func buildTextRequests(text, contentFormat string, insertIndex int64, tabID string) ([]*docs.Request, error) {
	switch contentFormat {
	case "richformat":
		if tabID != "" {
			return nil, fmt.Errorf("--tab not supported with richformat; include tabId in your JSON requests")
		}
		return markdown.ParseRichFormat(text)
	case "markdown", "plaintext":
		loc := &docs.Location{
			Index: insertIndex,
		}
		if tabID != "" {
			loc.TabId = tabID
		}
		return []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: loc,
					Text:     text,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown content format: %s (use markdown, plaintext, or richformat)", contentFormat)
	}
}

func runDocsCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	title, _ := cmd.Flags().GetString("title")
	text, _ := cmd.Flags().GetString("text")
	contentFormat, _ := cmd.Flags().GetString("content-format")

	// Create document with title
	doc, err := svc.Documents.Create(&docs.Document{
		Title: title,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create document: %w", err))
	}

	// If initial text provided, insert it
	if text != "" {
		requests, err := buildTextRequests(text, contentFormat, 1, "")
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to build text requests: %w", err))
		}

		_, err = svc.Documents.BatchUpdate(doc.DocumentId, &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to add initial text: %w", err))
		}
	}

	return p.Print(map[string]interface{}{
		"status": "created",
		"id":     doc.DocumentId,
		"title":  doc.Title,
		"url":    fmt.Sprintf("https://docs.google.com/document/d/%s/edit", doc.DocumentId),
	})
}

func runDocsAppend(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	text, _ := cmd.Flags().GetString("text")
	addNewline, _ := cmd.Flags().GetBool("newline")
	contentFormat, _ := cmd.Flags().GetString("content-format")

	// Get current document to find end index
	doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get document: %w", err))
	}

	// Resolve tab
	tabID, err := resolveTabFromFlags(cmd, doc)
	if err != nil {
		return p.PrintError(err)
	}

	body, err := getTabBody(doc, tabID)
	if err != nil {
		return p.PrintError(err)
	}

	// Guard against empty document
	if body == nil || len(body.Content) == 0 {
		return p.PrintError(fmt.Errorf("document has no content"))
	}

	// Find the end index of the document body
	endIndex := body.Content[len(body.Content)-1].EndIndex - 1

	// Prepare text: add newline prefix unless richformat (which provides its own requests)
	insertText := text
	if contentFormat != "richformat" && addNewline {
		insertText = "\n" + text
	}
	if contentFormat == "richformat" && cmd.Flags().Changed("newline") {
		fmt.Fprintln(os.Stderr, "warning: --newline is ignored when --content-format is richformat")
	}

	requests, err := buildTextRequests(insertText, contentFormat, endIndex, tabID)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to build text requests: %w", err))
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to append text: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "appended",
		"document_id": docID,
		"title":       doc.Title,
		"text_length": len(text),
	})
}

func runDocsInsert(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	text, _ := cmd.Flags().GetString("text")
	position, _ := cmd.Flags().GetInt64("at")
	contentFormat, _ := cmd.Flags().GetString("content-format")
	tabQuery, _ := cmd.Flags().GetString("tab")

	// Validate position (skip for richformat which provides its own positions)
	if contentFormat != "richformat" && position < 1 {
		return p.PrintError(fmt.Errorf("position must be >= 1"))
	}
	if contentFormat == "richformat" && cmd.Flags().Changed("at") {
		fmt.Fprintln(os.Stderr, "warning: --at is ignored when --content-format is richformat")
	}

	// Resolve tab ID if --tab provided
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	requests, err := buildTextRequests(text, contentFormat, position, tabID)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to build text requests: %w", err))
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to insert text: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"document_id": docID,
		"position":    position,
		"text_length": len(text),
	})
}

func runDocsReplace(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	findText, _ := cmd.Flags().GetString("find")
	replaceText, _ := cmd.Flags().GetString("replace")
	matchCase, _ := cmd.Flags().GetBool("match-case")
	tabQuery, _ := cmd.Flags().GetString("tab")

	replaceReq := &docs.ReplaceAllTextRequest{
		ContainsText: &docs.SubstringMatchCriteria{
			Text:      findText,
			MatchCase: matchCase,
		},
		ReplaceText: replaceText,
	}

	// Set TabsCriteria when --tab is provided
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err := resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
		replaceReq.TabsCriteria = &docs.TabsCriteria{
			TabIds: []string{tabID},
		}
	}

	requests := []*docs.Request{{ReplaceAllText: replaceReq}}

	resp, err := svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to replace text: %w", err))
	}

	// Get replacement count from response
	var replacements int64
	if len(resp.Replies) > 0 && resp.Replies[0].ReplaceAllText != nil {
		replacements = resp.Replies[0].ReplaceAllText.OccurrencesChanged
	}

	return p.Print(map[string]interface{}{
		"status":       "replaced",
		"document_id":  docID,
		"find":         findText,
		"replace":      replaceText,
		"replacements": replacements,
	})
}

func runDocsDelete(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	tabQuery, _ := cmd.Flags().GetString("tab")

	// Validate positions
	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	rng := &docs.Range{
		StartIndex: from,
		EndIndex:   to,
	}
	if tabID != "" {
		rng.TabId = tabID
	}

	requests := []*docs.Request{
		{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: rng,
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete content: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"from":        from,
		"to":          to,
		"characters":  to - from,
	})
}

func runDocsAddTable(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	rows, _ := cmd.Flags().GetInt64("rows")
	cols, _ := cmd.Flags().GetInt64("cols")
	position, _ := cmd.Flags().GetInt64("at")
	tabQuery, _ := cmd.Flags().GetString("tab")

	// Validate
	if position < 1 {
		return p.PrintError(fmt.Errorf("--at must be >= 1"))
	}
	if rows < 1 {
		return p.PrintError(fmt.Errorf("--rows must be >= 1"))
	}
	if cols < 1 {
		return p.PrintError(fmt.Errorf("--cols must be >= 1"))
	}

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	loc := &docs.Location{Index: position}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			InsertTable: &docs.InsertTableRequest{
				Rows:     rows,
				Columns:  cols,
				Location: loc,
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add table: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"rows":        rows,
		"columns":     cols,
		"position":    position,
	})
}

// parseDocsHexColor parses a hex color string (#RRGGBB) into a Docs OptionalColor.
func parseDocsHexColor(hex string) (*docs.OptionalColor, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return nil, fmt.Errorf("invalid hex color format: %s (expected #RRGGBB)", hex)
	}

	var r, g, b int64
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return nil, fmt.Errorf("invalid hex color: %s", hex)
	}

	return &docs.OptionalColor{
		Color: &docs.Color{
			RgbColor: &docs.RgbColor{
				Red:   float64(r) / 255.0,
				Green: float64(g) / 255.0,
				Blue:  float64(b) / 255.0,
			},
		},
	}, nil
}

func runDocsFormat(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	bold, _ := cmd.Flags().GetBool("bold")
	italic, _ := cmd.Flags().GetBool("italic")
	fontSize, _ := cmd.Flags().GetInt64("font-size")
	textColor, _ := cmd.Flags().GetString("color")
	tabQuery, _ := cmd.Flags().GetString("tab")

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	textStyle := &docs.TextStyle{}
	var fields []string

	if cmd.Flags().Changed("bold") {
		textStyle.Bold = bold
		if !bold {
			textStyle.ForceSendFields = append(textStyle.ForceSendFields, "Bold")
		}
		fields = append(fields, "bold")
	}

	if cmd.Flags().Changed("italic") {
		textStyle.Italic = italic
		if !italic {
			textStyle.ForceSendFields = append(textStyle.ForceSendFields, "Italic")
		}
		fields = append(fields, "italic")
	}

	if fontSize > 0 {
		textStyle.FontSize = &docs.Dimension{
			Magnitude: float64(fontSize),
			Unit:      "PT",
		}
		fields = append(fields, "fontSize")
	}

	if textColor != "" {
		color, err := parseDocsHexColor(textColor)
		if err != nil {
			return p.PrintError(err)
		}
		textStyle.ForegroundColor = color
		fields = append(fields, "foregroundColor")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no formatting options specified; use --bold, --italic, --font-size, or --color"))
	}

	rng := &docs.Range{
		StartIndex: from,
		EndIndex:   to,
	}
	if tabID != "" {
		rng.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				TextStyle: textStyle,
				Range:     rng,
				Fields:    strings.Join(fields, ","),
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to format text: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "formatted",
		"document_id": docID,
		"from":        from,
		"to":          to,
	})
}

func runDocsSetParagraphStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	alignment, _ := cmd.Flags().GetString("alignment")
	lineSpacing, _ := cmd.Flags().GetFloat64("line-spacing")
	tabQuery, _ := cmd.Flags().GetString("tab")

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	paraStyle := &docs.ParagraphStyle{}
	var fields []string

	if alignment != "" {
		paraStyle.Alignment = alignment
		fields = append(fields, "alignment")
	}

	if lineSpacing > 0 {
		paraStyle.LineSpacing = lineSpacing * 100 // API uses percentage (e.g., 115 for 1.15)
		fields = append(fields, "lineSpacing")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no style options specified; use --alignment or --line-spacing"))
	}

	rng := &docs.Range{
		StartIndex: from,
		EndIndex:   to,
	}
	if tabID != "" {
		rng.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
				ParagraphStyle: paraStyle,
				Range:          rng,
				Fields:         strings.Join(fields, ","),
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to set paragraph style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "styled",
		"document_id": docID,
		"from":        from,
		"to":          to,
	})
}

func runDocsAddList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	position, _ := cmd.Flags().GetInt64("at")
	listType, _ := cmd.Flags().GetString("type")
	itemsStr, _ := cmd.Flags().GetString("items")
	tabQuery, _ := cmd.Flags().GetString("tab")

	if position < 1 {
		return p.PrintError(fmt.Errorf("--at must be >= 1"))
	}

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	// Parse items
	items := strings.Split(itemsStr, ";")
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}

	// Build text to insert: each item on its own line
	insertText := strings.Join(items, "\n") + "\n"

	// Determine bullet preset
	var bulletPreset string
	switch listType {
	case "bullet":
		bulletPreset = "BULLET_DISC_CIRCLE_SQUARE"
	case "numbered":
		bulletPreset = "NUMBERED_DECIMAL_NESTED"
	default:
		return p.PrintError(fmt.Errorf("invalid list type: %s (use 'bullet' or 'numbered')", listType))
	}

	loc := &docs.Location{Index: position}
	rng := &docs.Range{
		StartIndex: position,
		EndIndex:   position + int64(len(insertText)),
	}
	if tabID != "" {
		loc.TabId = tabID
		rng.TabId = tabID
	}

	// Two-step batchUpdate: insert text, then apply bullets
	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: loc,
				Text:     insertText,
			},
		},
		{
			CreateParagraphBullets: &docs.CreateParagraphBulletsRequest{
				Range:        rng,
				BulletPreset: bulletPreset,
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add list: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"type":        listType,
		"items":       len(items),
		"position":    position,
	})
}

func runDocsRemoveList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	tabQuery, _ := cmd.Flags().GetString("tab")

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	rng := &docs.Range{
		StartIndex: from,
		EndIndex:   to,
	}
	if tabID != "" {
		rng.TabId = tabID
	}

	requests := []*docs.Request{
		{
			DeleteParagraphBullets: &docs.DeleteParagraphBulletsRequest{
				Range: rng,
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to remove list: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "removed",
		"document_id": docID,
		"from":        from,
		"to":          to,
	})
}

func runDocsTrash(cmd *cobra.Command, args []string) error {
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

	docID := args[0]
	permanent, _ := cmd.Flags().GetBool("permanent")

	// Get file info first for the response
	file, err := svc.Files.Get(docID).SupportsAllDrives(true).Fields("name").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get document info: %w", err))
	}

	if permanent {
		err = svc.Files.Delete(docID).SupportsAllDrives(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to delete document: %w", err))
		}

		return p.Print(map[string]interface{}{
			"status":      "deleted",
			"document_id": docID,
			"name":        file.Name,
		})
	}

	// Move to trash
	_, err = svc.Files.Update(docID, &drive.File{Trashed: true}).SupportsAllDrives(true).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to trash document: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "trashed",
		"document_id": docID,
		"name":        file.Name,
	})
}

// docsBatchUpdate is a helper that creates a client and executes a batch update.
func docsBatchUpdate(docID string, requests []*docs.Request) error {
	ctx := context.Background()
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return err
	}
	svc, err := factory.Docs()
	if err != nil {
		return err
	}
	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	return err
}

func runDocsAddTab(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	title, _ := cmd.Flags().GetString("title")
	index, _ := cmd.Flags().GetInt64("index")

	tabProps := &docs.TabProperties{
		Title: title,
	}
	if index >= 0 {
		tabProps.Index = index
		tabProps.ForceSendFields = append(tabProps.ForceSendFields, "Index")
	}

	requests := []*docs.Request{
		{AddDocumentTab: &docs.AddDocumentTabRequest{TabProperties: tabProps}},
	}

	resp, err := svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add tab: %w", err))
	}

	result := map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"title":       title,
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddDocumentTab != nil {
		tp := resp.Replies[0].AddDocumentTab.TabProperties
		if tp != nil {
			result["tab_id"] = tp.TabId
		}
	}

	return p.Print(result)
}

func runDocsDeleteTab(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	tabID, _ := cmd.Flags().GetString("tab-id")

	requests := []*docs.Request{
		{DeleteTab: &docs.DeleteTabRequest{TabId: tabID}},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete tab: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"tab_id":      tabID,
	})
}

func runDocsRenameTab(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	tabID, _ := cmd.Flags().GetString("tab-id")
	title, _ := cmd.Flags().GetString("title")

	requests := []*docs.Request{
		{
			UpdateDocumentTabProperties: &docs.UpdateDocumentTabPropertiesRequest{
				TabProperties: &docs.TabProperties{
					TabId: tabID,
					Title: title,
				},
				Fields: "title",
			},
		},
	}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to rename tab: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "renamed",
		"document_id": docID,
		"tab_id":      tabID,
		"title":       title,
	})
}

func runDocsAddImage(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Docs()
	if err != nil {
		return p.PrintError(err)
	}

	docID := args[0]
	uri, _ := cmd.Flags().GetString("uri")
	position, _ := cmd.Flags().GetInt64("at")
	width, _ := cmd.Flags().GetFloat64("width")
	height, _ := cmd.Flags().GetFloat64("height")
	tabQuery, _ := cmd.Flags().GetString("tab")

	// Resolve tab ID
	tabID := ""
	if tabQuery != "" {
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err = resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
	}

	loc := &docs.Location{Index: position}
	if tabID != "" {
		loc.TabId = tabID
	}

	insertReq := &docs.InsertInlineImageRequest{
		Uri:      uri,
		Location: loc,
	}

	if width > 0 {
		insertReq.ObjectSize = &docs.Size{
			Width: &docs.Dimension{Magnitude: width, Unit: "PT"},
		}
	}
	if height > 0 {
		if insertReq.ObjectSize == nil {
			insertReq.ObjectSize = &docs.Size{}
		}
		insertReq.ObjectSize.Height = &docs.Dimension{Magnitude: height, Unit: "PT"}
	}

	requests := []*docs.Request{{InsertInlineImage: insertReq}}

	_, err = svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add image: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"document_id": docID,
		"uri":         uri,
		"position":    position,
	})
}

// tableCellLocation builds a TableCellLocation from common flags.
// resolveTabQueryToID resolves --tab flag to a tab ID by fetching the document.
// Returns empty string if --tab is not set. Used by commands that don't already
// fetch the document for other reasons.
func resolveTabQueryToID(cmd *cobra.Command, docID string) (string, error) {
	tabQuery, _ := cmd.Flags().GetString("tab")
	if tabQuery == "" {
		return "", nil
	}
	ctx := context.Background()
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return "", err
	}
	svc, err := factory.Docs()
	if err != nil {
		return "", err
	}
	doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}
	return resolveTabID(doc.Tabs, tabQuery)
}

func tableCellLocation(cmd *cobra.Command, tabID string) *docs.TableCellLocation {
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	row, _ := cmd.Flags().GetInt64("row")
	col, _ := cmd.Flags().GetInt64("col")
	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}
	return &docs.TableCellLocation{
		TableStartLocation: loc,
		RowIndex:           row,
		ColumnIndex:        col,
	}
}

func runDocsInsertTableRow(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	below, _ := cmd.Flags().GetBool("below")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*docs.Request{
		{
			InsertTableRow: &docs.InsertTableRowRequest{
				TableCellLocation: tableCellLocation(cmd, tabID),
				InsertBelow:       below,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to insert table row: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"document_id": docID,
		"type":        "row",
	})
}

func runDocsDeleteTableRow(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*docs.Request{
		{
			DeleteTableRow: &docs.DeleteTableRowRequest{
				TableCellLocation: tableCellLocation(cmd, tabID),
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to delete table row: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"type":        "row",
	})
}

func runDocsInsertTableCol(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	right, _ := cmd.Flags().GetBool("right")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*docs.Request{
		{
			InsertTableColumn: &docs.InsertTableColumnRequest{
				TableCellLocation: tableCellLocation(cmd, tabID),
				InsertRight:       right,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to insert table column: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"document_id": docID,
		"type":        "column",
	})
}

func runDocsDeleteTableCol(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*docs.Request{
		{
			DeleteTableColumn: &docs.DeleteTableColumnRequest{
				TableCellLocation: tableCellLocation(cmd, tabID),
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to delete table column: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"type":        "column",
	})
}

func runDocsMergeCells(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	row, _ := cmd.Flags().GetInt64("row")
	col, _ := cmd.Flags().GetInt64("col")
	rowSpan, _ := cmd.Flags().GetInt64("row-span")
	colSpan, _ := cmd.Flags().GetInt64("col-span")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			MergeTableCells: &docs.MergeTableCellsRequest{
				TableRange: &docs.TableRange{
					TableCellLocation: &docs.TableCellLocation{
						TableStartLocation: loc,
						RowIndex:           row,
						ColumnIndex:        col,
					},
					RowSpan:    rowSpan,
					ColumnSpan: colSpan,
				},
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to merge cells: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "merged",
		"document_id": docID,
	})
}

func runDocsUnmergeCells(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	row, _ := cmd.Flags().GetInt64("row")
	col, _ := cmd.Flags().GetInt64("col")
	rowSpan, _ := cmd.Flags().GetInt64("row-span")
	colSpan, _ := cmd.Flags().GetInt64("col-span")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UnmergeTableCells: &docs.UnmergeTableCellsRequest{
				TableRange: &docs.TableRange{
					TableCellLocation: &docs.TableCellLocation{
						TableStartLocation: loc,
						RowIndex:           row,
						ColumnIndex:        col,
					},
					RowSpan:    rowSpan,
					ColumnSpan: colSpan,
				},
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to unmerge cells: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "unmerged",
		"document_id": docID,
	})
}

func runDocsPinRows(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	count, _ := cmd.Flags().GetInt64("count")

	if count < 0 {
		return p.PrintError(fmt.Errorf("--count must be >= 0"))
	}

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			PinTableHeaderRows: &docs.PinTableHeaderRowsRequest{
				TableStartLocation:    loc,
				PinnedHeaderRowsCount: count,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to pin rows: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "pinned",
		"document_id": docID,
		"count":       count,
	})
}

func runDocsPageBreak(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	position, _ := cmd.Flags().GetInt64("at")
	tabQuery, _ := cmd.Flags().GetString("tab")

	loc := &docs.Location{Index: position}
	if tabQuery != "" {
		ctx := context.Background()
		factory, err := client.NewFactory(ctx)
		if err != nil {
			return p.PrintError(err)
		}
		svc, err := factory.Docs()
		if err != nil {
			return p.PrintError(err)
		}
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err := resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			InsertPageBreak: &docs.InsertPageBreakRequest{
				Location: loc,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to insert page break: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"document_id": docID,
		"type":        "page_break",
		"position":    position,
	})
}

func runDocsSectionBreak(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	position, _ := cmd.Flags().GetInt64("at")
	breakType, _ := cmd.Flags().GetString("type")
	tabQuery, _ := cmd.Flags().GetString("tab")

	loc := &docs.Location{Index: position}
	if tabQuery != "" {
		ctx := context.Background()
		factory, err := client.NewFactory(ctx)
		if err != nil {
			return p.PrintError(err)
		}
		svc, err := factory.Docs()
		if err != nil {
			return p.PrintError(err)
		}
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err := resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			InsertSectionBreak: &docs.InsertSectionBreakRequest{
				Location:    loc,
				SectionType: breakType,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to insert section break: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"document_id": docID,
		"type":        breakType,
		"position":    position,
	})
}

func runDocsAddHeader(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	headerType, _ := cmd.Flags().GetString("type")

	requests := []*docs.Request{
		{
			CreateHeader: &docs.CreateHeaderRequest{
				Type:                 headerType,
				SectionBreakLocation: &docs.Location{Index: 0},
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to add header: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"type":        "header",
	})
}

func runDocsDeleteHeader(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	headerID := args[1]

	requests := []*docs.Request{
		{DeleteHeader: &docs.DeleteHeaderRequest{HeaderId: headerID}},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to delete header: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"header_id":   headerID,
	})
}

func runDocsAddFooter(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	footerType, _ := cmd.Flags().GetString("type")

	requests := []*docs.Request{
		{
			CreateFooter: &docs.CreateFooterRequest{
				Type:                 footerType,
				SectionBreakLocation: &docs.Location{Index: 0},
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to add footer: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"type":        "footer",
	})
}

func runDocsDeleteFooter(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	footerID := args[1]

	requests := []*docs.Request{
		{DeleteFooter: &docs.DeleteFooterRequest{FooterId: footerID}},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to delete footer: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"footer_id":   footerID,
	})
}

func runDocsAddNamedRange(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	name, _ := cmd.Flags().GetString("name")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	rng := &docs.Range{
		StartIndex: from,
		EndIndex:   to,
	}
	if tabID != "" {
		rng.TabId = tabID
	}

	requests := []*docs.Request{
		{
			CreateNamedRange: &docs.CreateNamedRangeRequest{
				Name:  name,
				Range: rng,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to add named range: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"name":        name,
		"from":        from,
		"to":          to,
	})
}

func runDocsDeleteNamedRange(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	name, _ := cmd.Flags().GetString("name")
	id, _ := cmd.Flags().GetString("id")

	if name == "" && id == "" {
		return p.PrintError(fmt.Errorf("either --name or --id is required"))
	}
	if name != "" && id != "" {
		return p.PrintError(fmt.Errorf("use either --name or --id, not both"))
	}

	req := &docs.DeleteNamedRangeRequest{}
	if name != "" {
		req.Name = name
	} else {
		req.NamedRangeId = id
	}

	requests := []*docs.Request{{DeleteNamedRange: req}}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to delete named range: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
	})
}

func runDocsAddFootnote(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	position, _ := cmd.Flags().GetInt64("at")
	tabQuery, _ := cmd.Flags().GetString("tab")

	if position < 1 {
		return p.PrintError(fmt.Errorf("--at must be >= 1"))
	}

	loc := &docs.Location{Index: position}
	if tabQuery != "" {
		ctx := context.Background()
		factory, err := client.NewFactory(ctx)
		if err != nil {
			return p.PrintError(err)
		}
		svc, err := factory.Docs()
		if err != nil {
			return p.PrintError(err)
		}
		doc, err := svc.Documents.Get(docID).IncludeTabsContent(true).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get document: %w", err))
		}
		tabID, err := resolveTabID(doc.Tabs, tabQuery)
		if err != nil {
			return p.PrintError(err)
		}
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			CreateFootnote: &docs.CreateFootnoteRequest{
				Location: loc,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to add footnote: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"document_id": docID,
		"position":    position,
	})
}

func runDocsDeleteObject(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	objectID := args[1]

	requests := []*docs.Request{
		{
			DeletePositionedObject: &docs.DeletePositionedObjectRequest{
				ObjectId: objectID,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to delete object: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"document_id": docID,
		"object_id":   objectID,
	})
}

func runDocsReplaceImage(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	uri, _ := cmd.Flags().GetString("uri")

	requests := []*docs.Request{
		{
			ReplaceImage: &docs.ReplaceImageRequest{
				ImageObjectId: objectID,
				Uri:           uri,
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to replace image: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "replaced",
		"document_id": docID,
		"object_id":   objectID,
	})
}

func runDocsReplaceNamedRange(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	name, _ := cmd.Flags().GetString("name")
	id, _ := cmd.Flags().GetString("id")
	text, _ := cmd.Flags().GetString("text")

	if name == "" && id == "" {
		return p.PrintError(fmt.Errorf("either --name or --id is required"))
	}
	if name != "" && id != "" {
		return p.PrintError(fmt.Errorf("use either --name or --id, not both"))
	}

	req := &docs.ReplaceNamedRangeContentRequest{
		Text: text,
	}
	if name != "" {
		req.NamedRangeName = name
	} else {
		req.NamedRangeId = id
	}

	requests := []*docs.Request{{ReplaceNamedRangeContent: req}}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to replace named range: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "replaced",
		"document_id": docID,
	})
}

func runDocsUpdateStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	marginTop, _ := cmd.Flags().GetFloat64("margin-top")
	marginBottom, _ := cmd.Flags().GetFloat64("margin-bottom")
	marginLeft, _ := cmd.Flags().GetFloat64("margin-left")
	marginRight, _ := cmd.Flags().GetFloat64("margin-right")

	docStyle := &docs.DocumentStyle{}
	var fields []string

	if cmd.Flags().Changed("margin-top") {
		docStyle.MarginTop = &docs.Dimension{Magnitude: marginTop, Unit: "PT"}
		fields = append(fields, "marginTop")
	}
	if cmd.Flags().Changed("margin-bottom") {
		docStyle.MarginBottom = &docs.Dimension{Magnitude: marginBottom, Unit: "PT"}
		fields = append(fields, "marginBottom")
	}
	if cmd.Flags().Changed("margin-left") {
		docStyle.MarginLeft = &docs.Dimension{Magnitude: marginLeft, Unit: "PT"}
		fields = append(fields, "marginLeft")
	}
	if cmd.Flags().Changed("margin-right") {
		docStyle.MarginRight = &docs.Dimension{Magnitude: marginRight, Unit: "PT"}
		fields = append(fields, "marginRight")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no margin options specified; use --margin-top, --margin-bottom, --margin-left, or --margin-right"))
	}

	requests := []*docs.Request{
		{
			UpdateDocumentStyle: &docs.UpdateDocumentStyleRequest{
				DocumentStyle: docStyle,
				Fields:        strings.Join(fields, ","),
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to update style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"document_id": docID,
	})
}

func runDocsUpdateSectionStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	columnCount, _ := cmd.Flags().GetInt64("column-count")
	contentDirection, _ := cmd.Flags().GetString("content-direction")

	sectionStyle := &docs.SectionStyle{}
	var fields []string

	if columnCount > 0 {
		cols := make([]*docs.SectionColumnProperties, columnCount)
		for i := range cols {
			cols[i] = &docs.SectionColumnProperties{}
		}
		sectionStyle.ColumnProperties = cols
		fields = append(fields, "columnProperties")
	}

	if contentDirection != "" {
		sectionStyle.ContentDirection = contentDirection
		fields = append(fields, "contentDirection")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no section style options specified"))
	}

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	rng := &docs.Range{
		StartIndex: from,
		EndIndex:   to,
	}
	if tabID != "" {
		rng.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UpdateSectionStyle: &docs.UpdateSectionStyleRequest{
				SectionStyle: sectionStyle,
				Range:        rng,
				Fields:       strings.Join(fields, ","),
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to update section style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"document_id": docID,
		"from":        from,
		"to":          to,
	})
}

func runDocsUpdateTableCellStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	row, _ := cmd.Flags().GetInt64("row")
	col, _ := cmd.Flags().GetInt64("col")
	rowSpan, _ := cmd.Flags().GetInt64("row-span")
	colSpan, _ := cmd.Flags().GetInt64("col-span")
	bgColor, _ := cmd.Flags().GetString("bg-color")
	padding, _ := cmd.Flags().GetFloat64("padding")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	cellStyle := &docs.TableCellStyle{}
	var fields []string

	if bgColor != "" {
		color, err := parseDocsHexColor(bgColor)
		if err != nil {
			return p.PrintError(err)
		}
		cellStyle.BackgroundColor = color
		fields = append(fields, "backgroundColor")
	}

	if cmd.Flags().Changed("padding") {
		dim := &docs.Dimension{Magnitude: padding, Unit: "PT"}
		cellStyle.PaddingTop = dim
		cellStyle.PaddingBottom = dim
		cellStyle.PaddingLeft = dim
		cellStyle.PaddingRight = dim
		fields = append(fields, "paddingTop", "paddingBottom", "paddingLeft", "paddingRight")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no cell style options specified; use --bg-color or --padding"))
	}

	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UpdateTableCellStyle: &docs.UpdateTableCellStyleRequest{
				TableCellStyle: cellStyle,
				TableRange: &docs.TableRange{
					TableCellLocation: &docs.TableCellLocation{
						TableStartLocation: loc,
						RowIndex:           row,
						ColumnIndex:        col,
					},
					RowSpan:    rowSpan,
					ColumnSpan: colSpan,
				},
				Fields: strings.Join(fields, ","),
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to update table cell style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"document_id": docID,
	})
}

func runDocsUpdateTableColProperties(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	colIndex, _ := cmd.Flags().GetInt64("col-index")
	width, _ := cmd.Flags().GetFloat64("width")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UpdateTableColumnProperties: &docs.UpdateTableColumnPropertiesRequest{
				TableStartLocation: loc,
				ColumnIndices:      []int64{colIndex},
				TableColumnProperties: &docs.TableColumnProperties{
					Width:     &docs.Dimension{Magnitude: width, Unit: "PT"},
					WidthType: "FIXED_WIDTH",
				},
				Fields: "width,widthType",
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to update table column properties: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"document_id": docID,
		"col_index":   colIndex,
		"width":       width,
	})
}

func runDocsUpdateTableRowStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	docID := args[0]
	tableStart, _ := cmd.Flags().GetInt64("table-start")
	row, _ := cmd.Flags().GetInt64("row")
	minHeight, _ := cmd.Flags().GetFloat64("min-height")

	tabID, err := resolveTabQueryToID(cmd, docID)
	if err != nil {
		return p.PrintError(err)
	}

	loc := &docs.Location{Index: tableStart}
	if tabID != "" {
		loc.TabId = tabID
	}

	requests := []*docs.Request{
		{
			UpdateTableRowStyle: &docs.UpdateTableRowStyleRequest{
				TableStartLocation: loc,
				RowIndices:         []int64{row},
				TableRowStyle: &docs.TableRowStyle{
					MinRowHeight: &docs.Dimension{Magnitude: minHeight, Unit: "PT"},
				},
				Fields: "minRowHeight",
			},
		},
	}

	if err := docsBatchUpdate(docID, requests); err != nil {
		return p.PrintError(fmt.Errorf("failed to update table row style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"document_id": docID,
		"row":         row,
		"min_height":  minHeight,
	})
}
