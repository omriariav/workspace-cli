package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/gws/internal/client"
	"github.com/omriariav/workspace-cli/gws/internal/printer"
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

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsReadCmd)
	docsCmd.AddCommand(docsInfoCmd)

	// Read flags
	docsReadCmd.Flags().Bool("include-formatting", false, "Include formatting information")
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
