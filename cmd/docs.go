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

func init() {
	rootCmd.AddCommand(docsCmd)
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

	doc, err := svc.Documents.Get(docID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get document: %w", err))
	}

	// Extract text content
	var textBuilder strings.Builder
	extractText(doc.Body.Content, &textBuilder)

	result := map[string]interface{}{
		"id":    doc.DocumentId,
		"title": doc.Title,
		"text":  textBuilder.String(),
	}

	if includeFormatting {
		// Include structural information
		result["structure"] = extractStructure(doc.Body.Content)
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

	doc, err := svc.Documents.Get(docID).Do()
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
func buildTextRequests(text, contentFormat string, insertIndex int64) ([]*docs.Request, error) {
	switch contentFormat {
	case "richformat":
		return markdown.ParseRichFormat(text)
	case "markdown", "plaintext":
		return []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{
						Index: insertIndex,
					},
					Text: text,
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
		requests, err := buildTextRequests(text, contentFormat, 1)
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
	doc, err := svc.Documents.Get(docID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get document: %w", err))
	}

	// Guard against empty document
	if doc.Body == nil || len(doc.Body.Content) == 0 {
		return p.PrintError(fmt.Errorf("document has no content"))
	}

	// Find the end index of the document body
	endIndex := doc.Body.Content[len(doc.Body.Content)-1].EndIndex - 1

	// Prepare text: add newline prefix unless richformat (which provides its own requests)
	insertText := text
	if contentFormat != "richformat" && addNewline {
		insertText = "\n" + text
	}
	if contentFormat == "richformat" && cmd.Flags().Changed("newline") {
		fmt.Fprintln(os.Stderr, "warning: --newline is ignored when --content-format is richformat")
	}

	requests, err := buildTextRequests(insertText, contentFormat, endIndex)
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

	// Validate position (skip for richformat which provides its own positions)
	if contentFormat != "richformat" && position < 1 {
		return p.PrintError(fmt.Errorf("position must be >= 1"))
	}
	if contentFormat == "richformat" && cmd.Flags().Changed("at") {
		fmt.Fprintln(os.Stderr, "warning: --at is ignored when --content-format is richformat")
	}

	requests, err := buildTextRequests(text, contentFormat, position)
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

	requests := []*docs.Request{
		{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					Text:      findText,
					MatchCase: matchCase,
				},
				ReplaceText: replaceText,
			},
		},
	}

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

	// Validate positions
	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	requests := []*docs.Request{
		{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{
					StartIndex: from,
					EndIndex:   to,
				},
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

	requests := []*docs.Request{
		{
			InsertTable: &docs.InsertTableRequest{
				Rows:    rows,
				Columns: cols,
				Location: &docs.Location{
					Index: position,
				},
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

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
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

	requests := []*docs.Request{
		{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				TextStyle: textStyle,
				Range: &docs.Range{
					StartIndex: from,
					EndIndex:   to,
				},
				Fields: strings.Join(fields, ","),
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

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
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

	requests := []*docs.Request{
		{
			UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
				ParagraphStyle: paraStyle,
				Range: &docs.Range{
					StartIndex: from,
					EndIndex:   to,
				},
				Fields: strings.Join(fields, ","),
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

	if position < 1 {
		return p.PrintError(fmt.Errorf("--at must be >= 1"))
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

	// Two-step batchUpdate: insert text, then apply bullets
	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: position},
				Text:     insertText,
			},
		},
		{
			CreateParagraphBullets: &docs.CreateParagraphBulletsRequest{
				Range: &docs.Range{
					StartIndex: position,
					EndIndex:   position + int64(len(insertText)),
				},
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

	if from < 1 {
		return p.PrintError(fmt.Errorf("--from must be >= 1"))
	}
	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	requests := []*docs.Request{
		{
			DeleteParagraphBullets: &docs.DeleteParagraphBulletsRequest{
				Range: &docs.Range{
					StartIndex: from,
					EndIndex:   to,
				},
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
