package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/docs/v1"
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

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsReadCmd)
	docsCmd.AddCommand(docsInfoCmd)
	docsCmd.AddCommand(docsCreateCmd)
	docsCmd.AddCommand(docsAppendCmd)
	docsCmd.AddCommand(docsInsertCmd)
	docsCmd.AddCommand(docsReplaceCmd)

	// Read flags
	docsReadCmd.Flags().Bool("include-formatting", false, "Include formatting information")

	// Create flags
	docsCreateCmd.Flags().String("title", "", "Document title (required)")
	docsCreateCmd.Flags().String("text", "", "Initial text content")
	docsCreateCmd.MarkFlagRequired("title")

	// Append flags
	docsAppendCmd.Flags().String("text", "", "Text to append (required)")
	docsAppendCmd.Flags().Bool("newline", true, "Add newline before appending")
	docsAppendCmd.MarkFlagRequired("text")

	// Insert flags
	docsInsertCmd.Flags().String("text", "", "Text to insert (required)")
	docsInsertCmd.Flags().Int64("at", 1, "Position to insert at (1-based index)")
	docsInsertCmd.MarkFlagRequired("text")

	// Replace flags
	docsReplaceCmd.Flags().String("find", "", "Text to find (required)")
	docsReplaceCmd.Flags().String("replace", "", "Replacement text (required)")
	docsReplaceCmd.Flags().Bool("match-case", true, "Case-sensitive matching")
	docsReplaceCmd.MarkFlagRequired("find")
	docsReplaceCmd.MarkFlagRequired("replace")
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

	// Create document with title
	doc, err := svc.Documents.Create(&docs.Document{
		Title: title,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create document: %w", err))
	}

	// If initial text provided, insert it
	if text != "" {
		requests := []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{
						Index: 1, // Insert at beginning (after document start)
					},
					Text: text,
				},
			},
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

	// Prepare text to insert
	insertText := text
	if addNewline {
		insertText = "\n" + text
	}

	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: endIndex,
				},
				Text: insertText,
			},
		},
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

	// Validate position
	if position < 1 {
		return p.PrintError(fmt.Errorf("position must be >= 1"))
	}

	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{
					Index: position,
				},
				Text: text,
			},
		},
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
