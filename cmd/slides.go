package cmd

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/slides/v1"
)

var slidesCmd = &cobra.Command{
	Use:   "slides",
	Short: "Manage Google Slides",
	Long:  "Commands for interacting with Google Slides presentations.",
}

var slidesInfoCmd = &cobra.Command{
	Use:   "info <presentation-id>",
	Short: "Get presentation info",
	Long:  "Gets metadata about a Google Slides presentation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesInfo,
}

var slidesListCmd = &cobra.Command{
	Use:   "list <presentation-id>",
	Short: "List slides",
	Long:  "Lists all slides in a presentation with their content.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesList,
}

var slidesReadCmd = &cobra.Command{
	Use:   "read <presentation-id> [slide-number]",
	Short: "Read slide content",
	Long:  "Reads the text content of a specific slide (1-indexed) or all slides.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runSlidesRead,
}

var slidesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new presentation",
	Long:  "Creates a new Google Slides presentation.",
	RunE:  runSlidesCreate,
}

var slidesAddSlideCmd = &cobra.Command{
	Use:   "add-slide <presentation-id>",
	Short: "Add a slide to a presentation",
	Long:  "Adds a new slide to an existing presentation with optional title and body text.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesAddSlide,
}

var slidesDeleteSlideCmd = &cobra.Command{
	Use:   "delete-slide <presentation-id>",
	Short: "Delete a slide",
	Long:  "Deletes a slide from a presentation by slide ID or number.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesDeleteSlide,
}

var slidesDuplicateSlideCmd = &cobra.Command{
	Use:   "duplicate-slide <presentation-id>",
	Short: "Duplicate a slide",
	Long:  "Creates a copy of an existing slide in the presentation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesDuplicateSlide,
}

var slidesAddShapeCmd = &cobra.Command{
	Use:   "add-shape <presentation-id>",
	Short: "Add a shape to a slide",
	Long: `Adds a shape to a slide at specified position.

Available shape types: RECTANGLE, ELLIPSE, TEXT_BOX, ROUND_RECTANGLE,
TRIANGLE, ARROW, etc.

Position and size are in points (PT).`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesAddShape,
}

var slidesAddImageCmd = &cobra.Command{
	Use:   "add-image <presentation-id>",
	Short: "Add an image to a slide",
	Long: `Adds an image to a slide from a URL.

Position and size are in points (PT). The image URL must be publicly accessible.`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesAddImage,
}

var slidesAddTextCmd = &cobra.Command{
	Use:   "add-text <presentation-id>",
	Short: "Add text to an object",
	Long: `Inserts text into an existing shape, text box, or table cell.

For shapes/text boxes, use --object-id.
For table cells, use --table-id with --row and --col (0-indexed).`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesAddText,
}

var slidesReplaceTextCmd = &cobra.Command{
	Use:   "replace-text <presentation-id>",
	Short: "Find and replace text",
	Long:  "Replaces all occurrences of text across all slides in the presentation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesReplaceText,
}

var slidesDeleteObjectCmd = &cobra.Command{
	Use:   "delete-object <presentation-id>",
	Short: "Delete any page element",
	Long:  "Deletes any page element (shape, image, table, etc.) from a presentation by object ID.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesDeleteObject,
}

var slidesDeleteTextCmd = &cobra.Command{
	Use:   "delete-text <presentation-id>",
	Short: "Clear text from a shape",
	Long:  "Deletes text from a shape, optionally within a specific range.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesDeleteText,
}

var slidesUpdateTextStyleCmd = &cobra.Command{
	Use:   "update-text-style <presentation-id>",
	Short: "Change font formatting",
	Long: `Updates text styling within a shape.

Supports bold, italic, underline, font size, font family, and text color.
Color should be specified as hex "#RRGGBB".`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesUpdateTextStyle,
}

var slidesUpdateTransformCmd = &cobra.Command{
	Use:   "update-transform <presentation-id>",
	Short: "Move or resize elements",
	Long: `Updates the position, size, or rotation of a page element.

Position and size are in points (PT). Rotation is in degrees.`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesUpdateTransform,
}

var slidesCreateTableCmd = &cobra.Command{
	Use:   "create-table <presentation-id>",
	Short: "Add a table to a slide",
	Long: `Creates a new table on a slide with specified rows and columns.

Position and size are in points (PT).`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesCreateTable,
}

var slidesInsertTableRowsCmd = &cobra.Command{
	Use:   "insert-table-rows <presentation-id>",
	Short: "Add rows to a table",
	Long:  "Inserts one or more rows into an existing table at a specified position.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesInsertTableRows,
}

var slidesDeleteTableRowCmd = &cobra.Command{
	Use:   "delete-table-row <presentation-id>",
	Short: "Remove row from table",
	Long:  "Deletes a row from an existing table.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesDeleteTableRow,
}

var slidesUpdateTableCellCmd = &cobra.Command{
	Use:   "update-table-cell <presentation-id>",
	Short: "Format table cell",
	Long: `Updates table cell properties like background color and padding.

Color should be specified as hex "#RRGGBB". Padding is in points (PT).`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesUpdateTableCell,
}

var slidesUpdateTableBorderCmd = &cobra.Command{
	Use:   "update-table-border <presentation-id>",
	Short: "Style table borders",
	Long: `Updates table border properties like color, width, and style.

Color should be specified as hex "#RRGGBB". Width is in points (PT).
Styles: solid, dashed, dotted.`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesUpdateTableBorder,
}

var slidesUpdateParagraphStyleCmd = &cobra.Command{
	Use:   "update-paragraph-style <presentation-id>",
	Short: "Paragraph formatting",
	Long: `Updates paragraph-level formatting within a shape.

Supports alignment, line spacing, and paragraph spacing.`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesUpdateParagraphStyle,
}

var slidesUpdateShapeCmd = &cobra.Command{
	Use:   "update-shape <presentation-id>",
	Short: "Modify shape properties",
	Long: `Updates shape properties like fill color and outline.

Colors should be specified as hex "#RRGGBB". Outline width is in points (PT).`,
	Args: cobra.ExactArgs(1),
	RunE: runSlidesUpdateShape,
}

var slidesReorderSlidesCmd = &cobra.Command{
	Use:   "reorder-slides <presentation-id>",
	Short: "Change slide order",
	Long:  "Moves slides to a new position within the presentation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesReorderSlides,
}

var slidesUpdateSlideBackgroundCmd = &cobra.Command{
	Use:   "update-slide-background <presentation-id>",
	Short: "Set slide background",
	Long:  "Sets the background of a slide to a solid color or an image URL.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesUpdateSlideBackground,
}

var slidesListLayoutsCmd = &cobra.Command{
	Use:   "list-layouts <presentation-id>",
	Short: "List slide layouts",
	Long:  "Lists all available slide layouts from the presentation's masters.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesListLayouts,
}

var slidesAddLineCmd = &cobra.Command{
	Use:   "add-line <presentation-id>",
	Short: "Add a line to a slide",
	Long:  "Creates a line or connector on a slide.\n\nLine types: STRAIGHT_CONNECTOR_1, BENT_CONNECTOR_2, CURVED_CONNECTOR_2, etc.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesAddLine,
}

var slidesGroupCmd = &cobra.Command{
	Use:   "group <presentation-id>",
	Short: "Group elements together",
	Long:  "Groups multiple page elements into a single group.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesGroup,
}

var slidesUngroupCmd = &cobra.Command{
	Use:   "ungroup <presentation-id>",
	Short: "Ungroup elements",
	Long:  "Ungroups a group element back into individual elements.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesUngroup,
}

func init() {
	rootCmd.AddCommand(slidesCmd)
	slidesCmd.AddCommand(slidesInfoCmd)
	slidesCmd.AddCommand(slidesListCmd)
	slidesCmd.AddCommand(slidesReadCmd)
	slidesCmd.AddCommand(slidesCreateCmd)
	slidesCmd.AddCommand(slidesAddSlideCmd)
	slidesCmd.AddCommand(slidesDeleteSlideCmd)
	slidesCmd.AddCommand(slidesDuplicateSlideCmd)
	slidesCmd.AddCommand(slidesAddShapeCmd)
	slidesCmd.AddCommand(slidesAddImageCmd)
	slidesCmd.AddCommand(slidesAddTextCmd)
	slidesCmd.AddCommand(slidesReplaceTextCmd)
	slidesCmd.AddCommand(slidesDeleteObjectCmd)
	slidesCmd.AddCommand(slidesDeleteTextCmd)
	slidesCmd.AddCommand(slidesUpdateTextStyleCmd)
	slidesCmd.AddCommand(slidesUpdateTransformCmd)
	slidesCmd.AddCommand(slidesCreateTableCmd)
	slidesCmd.AddCommand(slidesInsertTableRowsCmd)
	slidesCmd.AddCommand(slidesDeleteTableRowCmd)
	slidesCmd.AddCommand(slidesUpdateTableCellCmd)
	slidesCmd.AddCommand(slidesUpdateTableBorderCmd)
	slidesCmd.AddCommand(slidesUpdateParagraphStyleCmd)
	slidesCmd.AddCommand(slidesUpdateShapeCmd)
	slidesCmd.AddCommand(slidesReorderSlidesCmd)
	slidesCmd.AddCommand(slidesUpdateSlideBackgroundCmd)
	slidesCmd.AddCommand(slidesListLayoutsCmd)
	slidesCmd.AddCommand(slidesAddLineCmd)
	slidesCmd.AddCommand(slidesGroupCmd)
	slidesCmd.AddCommand(slidesUngroupCmd)

	// Notes flags for read commands
	slidesInfoCmd.Flags().Bool("notes", false, "Include speaker notes in output")
	slidesListCmd.Flags().Bool("notes", false, "Include speaker notes in output")
	slidesReadCmd.Flags().Bool("notes", false, "Include speaker notes in output")

	// Create flags
	slidesCreateCmd.Flags().String("title", "", "Presentation title (required)")
	slidesCreateCmd.MarkFlagRequired("title")

	// Add-slide flags
	slidesAddSlideCmd.Flags().String("title", "", "Slide title")
	slidesAddSlideCmd.Flags().String("body", "", "Slide body text")
	slidesAddSlideCmd.Flags().String("layout", "TITLE_AND_BODY", "Slide layout (TITLE_AND_BODY, TITLE_ONLY, BLANK, etc.)")
	slidesAddSlideCmd.Flags().String("layout-id", "", "Custom layout ID from presentation's masters (overrides --layout)")

	// Delete-slide flags
	slidesDeleteSlideCmd.Flags().String("slide-id", "", "Slide object ID to delete")
	slidesDeleteSlideCmd.Flags().Int("slide-number", 0, "Slide number to delete (1-indexed)")

	// Duplicate-slide flags
	slidesDuplicateSlideCmd.Flags().String("slide-id", "", "Slide object ID to duplicate")
	slidesDuplicateSlideCmd.Flags().Int("slide-number", 0, "Slide number to duplicate (1-indexed)")

	// Add-shape flags
	slidesAddShapeCmd.Flags().String("slide-id", "", "Slide object ID")
	slidesAddShapeCmd.Flags().Int("slide-number", 0, "Slide number (1-indexed)")
	slidesAddShapeCmd.Flags().String("type", "RECTANGLE", "Shape type (RECTANGLE, ELLIPSE, TEXT_BOX, etc.)")
	slidesAddShapeCmd.Flags().Float64("x", 100, "X position in points")
	slidesAddShapeCmd.Flags().Float64("y", 100, "Y position in points")
	slidesAddShapeCmd.Flags().Float64("width", 200, "Width in points")
	slidesAddShapeCmd.Flags().Float64("height", 100, "Height in points")

	// Add-image flags
	slidesAddImageCmd.Flags().String("slide-id", "", "Slide object ID")
	slidesAddImageCmd.Flags().Int("slide-number", 0, "Slide number (1-indexed)")
	slidesAddImageCmd.Flags().String("url", "", "Image URL (required, must be publicly accessible)")
	slidesAddImageCmd.Flags().Float64("x", 100, "X position in points")
	slidesAddImageCmd.Flags().Float64("y", 100, "Y position in points")
	slidesAddImageCmd.Flags().Float64("width", 400, "Width in points (height auto-calculated to maintain aspect ratio)")
	slidesAddImageCmd.MarkFlagRequired("url")

	// Add-text flags
	slidesAddTextCmd.Flags().String("object-id", "", "Object ID to insert text into (required for shapes/text boxes)")
	slidesAddTextCmd.Flags().String("table-id", "", "Table object ID (required for table cells, mutually exclusive with --object-id)")
	slidesAddTextCmd.Flags().Int("row", -1, "Row index, 0-based (required with --table-id)")
	slidesAddTextCmd.Flags().Int("col", -1, "Column index, 0-based (required with --table-id)")
	slidesAddTextCmd.Flags().String("text", "", "Text to insert (required)")
	slidesAddTextCmd.Flags().Int("at", 0, "Position to insert at (0 = beginning)")
	slidesAddTextCmd.Flags().Bool("notes", false, "Target speaker notes shape (mutually exclusive with --object-id and --table-id)")
	slidesAddTextCmd.Flags().String("slide-id", "", "Slide object ID (required with --notes)")
	slidesAddTextCmd.Flags().Int("slide-number", 0, "Slide number, 1-indexed (required with --notes)")
	slidesAddTextCmd.MarkFlagRequired("text")

	// Replace-text flags
	slidesReplaceTextCmd.Flags().String("find", "", "Text to find (required)")
	slidesReplaceTextCmd.Flags().String("replace", "", "Replacement text (required)")
	slidesReplaceTextCmd.Flags().Bool("match-case", true, "Case-sensitive matching")
	slidesReplaceTextCmd.MarkFlagRequired("find")
	slidesReplaceTextCmd.MarkFlagRequired("replace")

	// Delete-object flags
	slidesDeleteObjectCmd.Flags().String("object-id", "", "Object ID to delete (required)")
	slidesDeleteObjectCmd.MarkFlagRequired("object-id")

	// Delete-text flags
	slidesDeleteTextCmd.Flags().String("object-id", "", "Shape containing text (required unless --notes)")
	slidesDeleteTextCmd.Flags().Int("from", 0, "Start index (default 0)")
	slidesDeleteTextCmd.Flags().Int("to", -1, "End index (if omitted, deletes to end)")
	slidesDeleteTextCmd.Flags().Bool("notes", false, "Target speaker notes shape (alternative to --object-id)")
	slidesDeleteTextCmd.Flags().String("slide-id", "", "Slide object ID (required with --notes)")
	slidesDeleteTextCmd.Flags().Int("slide-number", 0, "Slide number, 1-indexed (required with --notes)")

	// Update-text-style flags
	slidesUpdateTextStyleCmd.Flags().String("object-id", "", "Shape containing text (required)")
	slidesUpdateTextStyleCmd.Flags().Int("from", 0, "Start index")
	slidesUpdateTextStyleCmd.Flags().Int("to", -1, "End index (if omitted, applies to all text)")
	slidesUpdateTextStyleCmd.Flags().Bool("bold", false, "Make text bold")
	slidesUpdateTextStyleCmd.Flags().Bool("italic", false, "Make text italic")
	slidesUpdateTextStyleCmd.Flags().Bool("underline", false, "Underline text")
	slidesUpdateTextStyleCmd.Flags().Float64("font-size", 0, "Font size in points")
	slidesUpdateTextStyleCmd.Flags().String("font-family", "", "Font family name")
	slidesUpdateTextStyleCmd.Flags().String("color", "", "Text color as hex #RRGGBB")
	slidesUpdateTextStyleCmd.MarkFlagRequired("object-id")

	// Update-transform flags
	slidesUpdateTransformCmd.Flags().String("object-id", "", "Element to transform (required)")
	slidesUpdateTransformCmd.Flags().Float64("x", 0, "X position in points")
	slidesUpdateTransformCmd.Flags().Float64("y", 0, "Y position in points")
	slidesUpdateTransformCmd.Flags().Float64("scale-x", 1, "Scale factor X")
	slidesUpdateTransformCmd.Flags().Float64("scale-y", 1, "Scale factor Y")
	slidesUpdateTransformCmd.Flags().Float64("rotate", 0, "Rotation in degrees")
	slidesUpdateTransformCmd.MarkFlagRequired("object-id")

	// Create-table flags
	slidesCreateTableCmd.Flags().String("slide-id", "", "Slide object ID")
	slidesCreateTableCmd.Flags().Int("slide-number", 0, "Slide number (1-indexed)")
	slidesCreateTableCmd.Flags().Int("rows", 0, "Number of rows (required)")
	slidesCreateTableCmd.Flags().Int("cols", 0, "Number of columns (required)")
	slidesCreateTableCmd.Flags().Float64("x", 100, "X position in points")
	slidesCreateTableCmd.Flags().Float64("y", 100, "Y position in points")
	slidesCreateTableCmd.Flags().Float64("width", 400, "Width in points")
	slidesCreateTableCmd.Flags().Float64("height", 200, "Height in points")
	slidesCreateTableCmd.MarkFlagRequired("rows")
	slidesCreateTableCmd.MarkFlagRequired("cols")

	// Insert-table-rows flags
	slidesInsertTableRowsCmd.Flags().String("table-id", "", "Table object ID (required)")
	slidesInsertTableRowsCmd.Flags().Int("at", 0, "Row index to insert at (required)")
	slidesInsertTableRowsCmd.Flags().Int("count", 1, "Number of rows to insert")
	slidesInsertTableRowsCmd.Flags().Bool("below", true, "Insert below the index")
	slidesInsertTableRowsCmd.MarkFlagRequired("table-id")
	slidesInsertTableRowsCmd.MarkFlagRequired("at")

	// Delete-table-row flags
	slidesDeleteTableRowCmd.Flags().String("table-id", "", "Table object ID (required)")
	slidesDeleteTableRowCmd.Flags().Int("row", 0, "Row index to delete (required)")
	slidesDeleteTableRowCmd.MarkFlagRequired("table-id")
	slidesDeleteTableRowCmd.MarkFlagRequired("row")

	// Update-table-cell flags
	slidesUpdateTableCellCmd.Flags().String("table-id", "", "Table object ID (required)")
	slidesUpdateTableCellCmd.Flags().Int("row", 0, "Row index (required)")
	slidesUpdateTableCellCmd.Flags().Int("col", 0, "Column index (required)")
	slidesUpdateTableCellCmd.Flags().String("background-color", "", "Background color as hex #RRGGBB")
	slidesUpdateTableCellCmd.MarkFlagRequired("table-id")
	slidesUpdateTableCellCmd.MarkFlagRequired("row")
	slidesUpdateTableCellCmd.MarkFlagRequired("col")

	// Update-table-border flags
	slidesUpdateTableBorderCmd.Flags().String("table-id", "", "Table object ID (required)")
	slidesUpdateTableBorderCmd.Flags().Int("row", 0, "Row index (required)")
	slidesUpdateTableBorderCmd.Flags().Int("col", 0, "Column index (required)")
	slidesUpdateTableBorderCmd.Flags().String("border", "all", "Border to style: top, bottom, left, right, all")
	slidesUpdateTableBorderCmd.Flags().String("color", "", "Border color as hex #RRGGBB")
	slidesUpdateTableBorderCmd.Flags().Float64("width", 1, "Border width in points")
	slidesUpdateTableBorderCmd.Flags().String("style", "solid", "Border style: solid, dashed, dotted")
	slidesUpdateTableBorderCmd.MarkFlagRequired("table-id")
	slidesUpdateTableBorderCmd.MarkFlagRequired("row")
	slidesUpdateTableBorderCmd.MarkFlagRequired("col")

	// Update-paragraph-style flags
	slidesUpdateParagraphStyleCmd.Flags().String("object-id", "", "Shape containing text (required)")
	slidesUpdateParagraphStyleCmd.Flags().Int("from", 0, "Start index")
	slidesUpdateParagraphStyleCmd.Flags().Int("to", -1, "End index (if omitted, applies to all text)")
	slidesUpdateParagraphStyleCmd.Flags().String("alignment", "", "Text alignment: START, CENTER, END, JUSTIFIED")
	slidesUpdateParagraphStyleCmd.Flags().Float64("line-spacing", 0, "Line spacing percentage (e.g., 100 for single, 200 for double)")
	slidesUpdateParagraphStyleCmd.Flags().Float64("space-above", 0, "Space above paragraph in points")
	slidesUpdateParagraphStyleCmd.Flags().Float64("space-below", 0, "Space below paragraph in points")
	slidesUpdateParagraphStyleCmd.MarkFlagRequired("object-id")

	// Update-shape flags
	slidesUpdateShapeCmd.Flags().String("object-id", "", "Shape to update (required)")
	slidesUpdateShapeCmd.Flags().String("background-color", "", "Fill color as hex #RRGGBB")
	slidesUpdateShapeCmd.Flags().String("outline-color", "", "Outline color as hex #RRGGBB")
	slidesUpdateShapeCmd.Flags().Float64("outline-width", 0, "Outline width in points")
	slidesUpdateShapeCmd.MarkFlagRequired("object-id")

	// Reorder-slides flags
	slidesReorderSlidesCmd.Flags().String("slide-ids", "", "Comma-separated slide IDs to move (required)")
	slidesReorderSlidesCmd.Flags().Int("to", 0, "Target position (0-indexed, required)")
	slidesReorderSlidesCmd.MarkFlagRequired("slide-ids")
	slidesReorderSlidesCmd.MarkFlagRequired("to")

	// Update-slide-background flags
	slidesUpdateSlideBackgroundCmd.Flags().String("slide-id", "", "Slide object ID")
	slidesUpdateSlideBackgroundCmd.Flags().Int("slide-number", 0, "Slide number (1-indexed)")
	slidesUpdateSlideBackgroundCmd.Flags().String("color", "", "Background color as hex #RRGGBB")
	slidesUpdateSlideBackgroundCmd.Flags().String("image-url", "", "Background image URL")

	// Add-line flags
	slidesAddLineCmd.Flags().String("slide-id", "", "Slide object ID")
	slidesAddLineCmd.Flags().Int("slide-number", 0, "Slide number (1-indexed)")
	slidesAddLineCmd.Flags().String("type", "STRAIGHT_CONNECTOR_1", "Line type (STRAIGHT_CONNECTOR_1, BENT_CONNECTOR_2, CURVED_CONNECTOR_2, etc.)")
	slidesAddLineCmd.Flags().Float64("start-x", 0, "Start X position in points")
	slidesAddLineCmd.Flags().Float64("start-y", 0, "Start Y position in points")
	slidesAddLineCmd.Flags().Float64("end-x", 200, "End X position in points")
	slidesAddLineCmd.Flags().Float64("end-y", 200, "End Y position in points")
	slidesAddLineCmd.Flags().String("color", "", "Line color as hex #RRGGBB")
	slidesAddLineCmd.Flags().Float64("weight", 1, "Line thickness in points")

	// Group flags
	slidesGroupCmd.Flags().String("object-ids", "", "Comma-separated element IDs to group (required)")
	slidesGroupCmd.MarkFlagRequired("object-ids")

	// Ungroup flags
	slidesUngroupCmd.Flags().String("group-id", "", "Object ID of the group to ungroup (required)")
	slidesUngroupCmd.MarkFlagRequired("group-id")
}

func runSlidesInfo(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]

	presentation, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
	}

	result := map[string]interface{}{
		"id":          presentation.PresentationId,
		"title":       presentation.Title,
		"slide_count": len(presentation.Slides),
		"locale":      presentation.Locale,
	}

	if presentation.PageSize != nil {
		result["page_size"] = map[string]interface{}{
			"width":  presentation.PageSize.Width,
			"height": presentation.PageSize.Height,
		}
	}

	includeNotes, _ := cmd.Flags().GetBool("notes")

	// List slide IDs and titles
	slideInfo := make([]map[string]interface{}, 0, len(presentation.Slides))
	for i, slide := range presentation.Slides {
		info := map[string]interface{}{
			"number": i + 1,
			"id":     slide.ObjectId,
		}

		// Try to get slide title from shape elements
		title := extractSlideTitle(slide)
		if title != "" {
			info["title"] = title
		}

		if includeNotes {
			notes := extractSpeakerNotes(slide)
			if notes != "" {
				info["notes"] = notes
			}
		}

		slideInfo = append(slideInfo, info)
	}
	result["slides"] = slideInfo

	return p.Print(result)
}

func runSlidesList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]

	presentation, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
	}

	includeNotes, _ := cmd.Flags().GetBool("notes")

	slidesList := make([]map[string]interface{}, 0, len(presentation.Slides))
	for i, slide := range presentation.Slides {
		slideData := map[string]interface{}{
			"number": i + 1,
			"id":     slide.ObjectId,
		}

		// Extract all text from slide
		text := extractSlideText(slide)
		if text != "" {
			slideData["text"] = text
		}

		// Count elements
		slideData["element_count"] = len(slide.PageElements)

		if includeNotes {
			notes := extractSpeakerNotes(slide)
			if notes != "" {
				slideData["notes"] = notes
			}
		}

		slidesList = append(slidesList, slideData)
	}

	return p.Print(map[string]interface{}{
		"presentation": presentation.Title,
		"slides":       slidesList,
		"count":        len(slidesList),
	})
}

func runSlidesRead(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]

	presentation, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
	}

	includeNotes, _ := cmd.Flags().GetBool("notes")

	// If slide number provided, read specific slide
	if len(args) > 1 {
		var slideNum int
		_, _ = fmt.Sscanf(args[1], "%d", &slideNum)

		if slideNum < 1 || slideNum > len(presentation.Slides) {
			return p.PrintError(fmt.Errorf("slide number %d out of range (1-%d)", slideNum, len(presentation.Slides)))
		}

		slide := presentation.Slides[slideNum-1]
		text := extractSlideText(slide)

		result := map[string]interface{}{
			"slide":  slideNum,
			"id":     slide.ObjectId,
			"text":   text,
			"title":  extractSlideTitle(slide),
			"layout": slide.SlideProperties.LayoutObjectId,
		}

		if includeNotes {
			notes := extractSpeakerNotes(slide)
			if notes != "" {
				result["notes"] = notes
			}
		}

		return p.Print(result)
	}

	// Read all slides
	slidesContent := make([]map[string]interface{}, 0, len(presentation.Slides))
	for i, slide := range presentation.Slides {
		slideData := map[string]interface{}{
			"slide": i + 1,
			"id":    slide.ObjectId,
			"text":  extractSlideText(slide),
		}

		title := extractSlideTitle(slide)
		if title != "" {
			slideData["title"] = title
		}

		if includeNotes {
			notes := extractSpeakerNotes(slide)
			if notes != "" {
				slideData["notes"] = notes
			}
		}

		slidesContent = append(slidesContent, slideData)
	}

	return p.Print(map[string]interface{}{
		"presentation": presentation.Title,
		"slides":       slidesContent,
		"count":        len(slidesContent),
	})
}

// extractSlideTitle extracts the title from a slide (typically from title placeholder).
func extractSlideTitle(slide *slides.Page) string {
	for _, element := range slide.PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil {
			if element.Shape.Placeholder.Type == "TITLE" || element.Shape.Placeholder.Type == "CENTERED_TITLE" {
				return extractShapeText(element.Shape)
			}
		}
	}
	return ""
}

// extractSlideText extracts all text content from a slide.
func extractSlideText(slide *slides.Page) string {
	var texts []string

	for _, element := range slide.PageElements {
		if element.Shape != nil {
			text := extractShapeText(element.Shape)
			if text != "" {
				texts = append(texts, text)
			}
		}
		if element.Table != nil {
			text := extractTableText(element.Table)
			if text != "" {
				texts = append(texts, text)
			}
		}
	}

	return strings.Join(texts, "\n\n")
}

// extractShapeText extracts text from a shape element.
func extractShapeText(shape *slides.Shape) string {
	if shape.Text == nil {
		return ""
	}

	var builder strings.Builder
	for _, elem := range shape.Text.TextElements {
		if elem.TextRun != nil {
			builder.WriteString(elem.TextRun.Content)
		}
	}

	return strings.TrimSpace(builder.String())
}

// extractTableText extracts text from a table element.
func extractTableText(table *slides.Table) string {
	var rows []string

	for _, row := range table.TableRows {
		var cells []string
		for _, cell := range row.TableCells {
			if cell.Text != nil {
				var cellText strings.Builder
				for _, elem := range cell.Text.TextElements {
					if elem.TextRun != nil {
						cellText.WriteString(elem.TextRun.Content)
					}
				}
				cells = append(cells, strings.TrimSpace(cellText.String()))
			}
		}
		rows = append(rows, strings.Join(cells, "\t"))
	}

	return strings.Join(rows, "\n")
}

// extractSpeakerNotes extracts speaker notes text from a slide's notes page.
func extractSpeakerNotes(slide *slides.Page) string {
	if slide.SlideProperties == nil {
		return ""
	}
	notesPage := slide.SlideProperties.NotesPage
	if notesPage == nil {
		return ""
	}
	if notesPage.NotesProperties == nil || notesPage.NotesProperties.SpeakerNotesObjectId == "" {
		return ""
	}

	notesObjectID := notesPage.NotesProperties.SpeakerNotesObjectId
	for _, element := range notesPage.PageElements {
		if element.ObjectId == notesObjectID && element.Shape != nil {
			return extractShapeText(element.Shape)
		}
	}
	return ""
}

// getSpeakerNotesObjectID returns the object ID of the speaker notes shape for a slide.
func getSpeakerNotesObjectID(slide *slides.Page) (string, error) {
	if slide.SlideProperties == nil || slide.SlideProperties.NotesPage == nil {
		return "", fmt.Errorf("slide has no notes page")
	}
	notesPage := slide.SlideProperties.NotesPage
	if notesPage.NotesProperties == nil || notesPage.NotesProperties.SpeakerNotesObjectId == "" {
		return "", fmt.Errorf("slide has no speaker notes shape")
	}
	return notesPage.NotesProperties.SpeakerNotesObjectId, nil
}

// findSlide resolves a slide from a presentation by --slide-id or --slide-number.
func findSlide(presentation *slides.Presentation, slideIDFlag string, slideNumber int) (*slides.Page, error) {
	if slideIDFlag != "" && slideNumber > 0 {
		return nil, fmt.Errorf("specify only one of --slide-id or --slide-number, not both")
	}
	if slideIDFlag != "" {
		for _, s := range presentation.Slides {
			if s.ObjectId == slideIDFlag {
				return s, nil
			}
		}
		return nil, fmt.Errorf("slide with ID '%s' not found", slideIDFlag)
	}
	if slideNumber < 1 || slideNumber > len(presentation.Slides) {
		return nil, fmt.Errorf("slide number %d out of range (1-%d)", slideNumber, len(presentation.Slides))
	}
	return presentation.Slides[slideNumber-1], nil
}

func runSlidesCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	title, _ := cmd.Flags().GetString("title")

	presentation, err := svc.Presentations.Create(&slides.Presentation{
		Title: title,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create presentation: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"id":          presentation.PresentationId,
		"title":       presentation.Title,
		"slide_count": len(presentation.Slides),
		"url":         fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit", presentation.PresentationId),
	})
}

func runSlidesAddSlide(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideTitle, _ := cmd.Flags().GetString("title")
	slideBody, _ := cmd.Flags().GetString("body")
	layout, _ := cmd.Flags().GetString("layout")
	layoutID, _ := cmd.Flags().GetString("layout-id")

	// Build layout reference
	var layoutRef *slides.LayoutReference
	if layoutID != "" {
		// Use custom layout ID from presentation's masters
		layoutRef = &slides.LayoutReference{
			LayoutId: layoutID,
		}
	} else {
		// Validate predefined layout
		validLayouts := map[string]bool{
			"BLANK":                         true,
			"CAPTION_ONLY":                  true,
			"TITLE":                         true,
			"TITLE_AND_BODY":                true,
			"TITLE_AND_TWO_COLUMNS":         true,
			"TITLE_ONLY":                    true,
			"SECTION_HEADER":                true,
			"SECTION_TITLE_AND_DESCRIPTION": true,
			"ONE_COLUMN_TEXT":               true,
			"MAIN_POINT":                    true,
			"BIG_NUMBER":                    true,
		}
		if !validLayouts[layout] {
			return p.PrintError(fmt.Errorf("invalid layout '%s'. Valid layouts: BLANK, TITLE, TITLE_AND_BODY, TITLE_AND_TWO_COLUMNS, TITLE_ONLY, SECTION_HEADER, CAPTION_ONLY, MAIN_POINT, BIG_NUMBER", layout))
		}
		layoutRef = &slides.LayoutReference{
			PredefinedLayout: layout,
		}
	}

	// Create requests for adding a slide (let API generate the ID)
	requests := []*slides.Request{
		{
			CreateSlide: &slides.CreateSlideRequest{
				SlideLayoutReference: layoutRef,
			},
		},
	}

	// Execute the create slide request
	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add slide: %w", err))
	}

	// Get the slide ID from the response
	var slideObjectID string
	if len(resp.Replies) > 0 && resp.Replies[0].CreateSlide != nil {
		slideObjectID = resp.Replies[0].CreateSlide.ObjectId
	}

	// Get the created slide to find placeholder IDs (for adding text)
	presentation, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
	}

	// Find the new slide by ID
	var newSlide *slides.Page
	if slideObjectID != "" {
		for _, slide := range presentation.Slides {
			if slide.ObjectId == slideObjectID {
				newSlide = slide
				break
			}
		}
	}

	if newSlide != nil && (slideTitle != "" || slideBody != "") {
		textRequests := []*slides.Request{}

		// Find title and body placeholders
		for _, element := range newSlide.PageElements {
			if element.Shape != nil && element.Shape.Placeholder != nil {
				placeholderType := element.Shape.Placeholder.Type

				if placeholderType == "TITLE" || placeholderType == "CENTERED_TITLE" {
					if slideTitle != "" {
						textRequests = append(textRequests, &slides.Request{
							InsertText: &slides.InsertTextRequest{
								ObjectId: element.ObjectId,
								Text:     slideTitle,
							},
						})
					}
				}

				if placeholderType == "BODY" || placeholderType == "SUBTITLE" {
					if slideBody != "" {
						textRequests = append(textRequests, &slides.Request{
							InsertText: &slides.InsertTextRequest{
								ObjectId: element.ObjectId,
								Text:     slideBody,
							},
						})
					}
				}
			}
		}

		if len(textRequests) > 0 {
			_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
				Requests: textRequests,
			}).Do()
			if err != nil {
				return p.PrintError(fmt.Errorf("failed to add text to slide: %w", err))
			}
		}
	}

	slideNumber := len(presentation.Slides)
	return p.Print(map[string]interface{}{
		"status":          "added",
		"presentation_id": presentationID,
		"slide_id":        slideObjectID,
		"slide_number":    slideNumber,
	})
}

func runSlidesDeleteSlide(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideID, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")

	// If slide number provided, look up slide ID
	if slideNumber > 0 {
		presentation, err := svc.Presentations.Get(presentationID).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
		}

		if slideNumber > len(presentation.Slides) {
			return p.PrintError(fmt.Errorf("slide number %d out of range (1-%d)", slideNumber, len(presentation.Slides)))
		}

		slideID = presentation.Slides[slideNumber-1].ObjectId
	} else if slideID == "" {
		return p.PrintError(fmt.Errorf("must specify --slide-id or --slide-number"))
	}

	requests := []*slides.Request{
		{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: slideID,
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete slide: %w", err))
	}

	result := map[string]interface{}{
		"status":          "deleted",
		"presentation_id": presentationID,
		"slide_id":        slideID,
	}
	if slideNumber > 0 {
		result["slide_number"] = slideNumber
	}

	return p.Print(result)
}

func runSlidesDuplicateSlide(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideID, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")

	// If slide number provided, look up slide ID
	if slideNumber > 0 {
		presentation, err := svc.Presentations.Get(presentationID).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
		}

		if slideNumber > len(presentation.Slides) {
			return p.PrintError(fmt.Errorf("slide number %d out of range (1-%d)", slideNumber, len(presentation.Slides)))
		}

		slideID = presentation.Slides[slideNumber-1].ObjectId
	} else if slideID == "" {
		return p.PrintError(fmt.Errorf("must specify --slide-id or --slide-number"))
	}

	requests := []*slides.Request{
		{
			DuplicateObject: &slides.DuplicateObjectRequest{
				ObjectId: slideID,
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to duplicate slide: %w", err))
	}

	// Get the new slide ID from response
	var newSlideID string
	if len(resp.Replies) > 0 && resp.Replies[0].DuplicateObject != nil {
		newSlideID = resp.Replies[0].DuplicateObject.ObjectId
	}

	result := map[string]interface{}{
		"status":          "duplicated",
		"presentation_id": presentationID,
		"source_slide_id": slideID,
		"new_slide_id":    newSlideID,
	}
	if slideNumber > 0 {
		result["source_slide_number"] = slideNumber
	}

	return p.Print(result)
}

// getSlideID resolves a slide ID from either --slide-id or --slide-number flags.
func getSlideID(svc *slides.Service, presentationID string, slideIDFlag string, slideNumber int) (string, error) {
	if slideIDFlag != "" {
		return slideIDFlag, nil
	}

	if slideNumber > 0 {
		presentation, err := svc.Presentations.Get(presentationID).Do()
		if err != nil {
			return "", fmt.Errorf("failed to get presentation: %w", err)
		}

		if slideNumber > len(presentation.Slides) {
			return "", fmt.Errorf("slide number %d out of range (1-%d)", slideNumber, len(presentation.Slides))
		}

		return presentation.Slides[slideNumber-1].ObjectId, nil
	}

	return "", fmt.Errorf("must specify --slide-id or --slide-number")
}

// validShapeTypes contains supported Google Slides shape types
var validShapeTypes = map[string]bool{
	"TEXT_BOX":                      true,
	"RECTANGLE":                     true,
	"ROUND_RECTANGLE":               true,
	"ELLIPSE":                       true,
	"ARC":                           true,
	"BENT_ARROW":                    true,
	"BENT_UP_ARROW":                 true,
	"BEVEL":                         true,
	"BLOCK_ARC":                     true,
	"BRACE_PAIR":                    true,
	"BRACKET_PAIR":                  true,
	"CAN":                           true,
	"CHEVRON":                       true,
	"CHORD":                         true,
	"CLOUD":                         true,
	"CORNER":                        true,
	"CUBE":                          true,
	"CURVED_DOWN_ARROW":             true,
	"CURVED_LEFT_ARROW":             true,
	"CURVED_RIGHT_ARROW":            true,
	"CURVED_UP_ARROW":               true,
	"DECAGON":                       true,
	"DIAGONAL_STRIPE":               true,
	"DIAMOND":                       true,
	"DODECAGON":                     true,
	"DONUT":                         true,
	"DOUBLE_WAVE":                   true,
	"DOWN_ARROW":                    true,
	"DOWN_ARROW_CALLOUT":            true,
	"FOLDED_CORNER":                 true,
	"FRAME":                         true,
	"HALF_FRAME":                    true,
	"HEART":                         true,
	"HEPTAGON":                      true,
	"HEXAGON":                       true,
	"HOME_PLATE":                    true,
	"HORIZONTAL_SCROLL":             true,
	"IRREGULAR_SEAL_1":              true,
	"IRREGULAR_SEAL_2":              true,
	"LEFT_ARROW":                    true,
	"LEFT_ARROW_CALLOUT":            true,
	"LEFT_BRACE":                    true,
	"LEFT_BRACKET":                  true,
	"LEFT_RIGHT_ARROW":              true,
	"LEFT_RIGHT_ARROW_CALLOUT":      true,
	"LEFT_RIGHT_UP_ARROW":           true,
	"LEFT_UP_ARROW":                 true,
	"LIGHTNING_BOLT":                true,
	"MATH_DIVIDE":                   true,
	"MATH_EQUAL":                    true,
	"MATH_MINUS":                    true,
	"MATH_MULTIPLY":                 true,
	"MATH_NOT_EQUAL":                true,
	"MATH_PLUS":                     true,
	"MOON":                          true,
	"NO_SMOKING":                    true,
	"NOTCHED_RIGHT_ARROW":           true,
	"OCTAGON":                       true,
	"PARALLELOGRAM":                 true,
	"PENTAGON":                      true,
	"PIE":                           true,
	"PLAQUE":                        true,
	"PLUS":                          true,
	"QUAD_ARROW":                    true,
	"QUAD_ARROW_CALLOUT":            true,
	"RIBBON":                        true,
	"RIBBON_2":                      true,
	"RIGHT_ARROW":                   true,
	"RIGHT_ARROW_CALLOUT":           true,
	"RIGHT_BRACE":                   true,
	"RIGHT_BRACKET":                 true,
	"RIGHT_TRIANGLE":                true,
	"ROUND_1_RECTANGLE":             true,
	"ROUND_2_DIAGONAL_RECTANGLE":    true,
	"ROUND_2_SAME_RECTANGLE":        true,
	"SNIP_1_RECTANGLE":              true,
	"SNIP_2_DIAGONAL_RECTANGLE":     true,
	"SNIP_2_SAME_RECTANGLE":         true,
	"SNIP_ROUND_RECTANGLE":          true,
	"STAR_10":                       true,
	"STAR_12":                       true,
	"STAR_16":                       true,
	"STAR_24":                       true,
	"STAR_32":                       true,
	"STAR_4":                        true,
	"STAR_5":                        true,
	"STAR_6":                        true,
	"STAR_7":                        true,
	"STAR_8":                        true,
	"STRIPED_RIGHT_ARROW":           true,
	"SUN":                           true,
	"TRAPEZOID":                     true,
	"TRIANGLE":                      true,
	"UP_ARROW":                      true,
	"UP_ARROW_CALLOUT":              true,
	"UP_DOWN_ARROW":                 true,
	"UTURN_ARROW":                   true,
	"VERTICAL_SCROLL":               true,
	"WAVE":                          true,
	"WEDGE_ELLIPSE_CALLOUT":         true,
	"WEDGE_RECTANGLE_CALLOUT":       true,
	"WEDGE_ROUND_RECTANGLE_CALLOUT": true,
	"FLOW_CHART_ALTERNATE_PROCESS":  true,
	"FLOW_CHART_COLLATE":            true,
	"FLOW_CHART_CONNECTOR":          true,
	"FLOW_CHART_DECISION":           true,
	"FLOW_CHART_DELAY":              true,
	"FLOW_CHART_DISPLAY":            true,
	"FLOW_CHART_DOCUMENT":           true,
	"FLOW_CHART_EXTRACT":            true,
	"FLOW_CHART_INPUT_OUTPUT":       true,
	"FLOW_CHART_INTERNAL_STORAGE":   true,
	"FLOW_CHART_MAGNETIC_DISK":      true,
	"FLOW_CHART_MAGNETIC_DRUM":      true,
	"FLOW_CHART_MAGNETIC_TAPE":      true,
	"FLOW_CHART_MANUAL_INPUT":       true,
	"FLOW_CHART_MANUAL_OPERATION":   true,
	"FLOW_CHART_MERGE":              true,
	"FLOW_CHART_MULTIDOCUMENT":      true,
	"FLOW_CHART_OFFLINE_STORAGE":    true,
	"FLOW_CHART_OFFPAGE_CONNECTOR":  true,
	"FLOW_CHART_ONLINE_STORAGE":     true,
	"FLOW_CHART_OR":                 true,
	"FLOW_CHART_PREDEFINED_PROCESS": true,
	"FLOW_CHART_PREPARATION":        true,
	"FLOW_CHART_PROCESS":            true,
	"FLOW_CHART_PUNCHED_CARD":       true,
	"FLOW_CHART_PUNCHED_TAPE":       true,
	"FLOW_CHART_SORT":               true,
	"FLOW_CHART_SUMMING_JUNCTION":   true,
	"FLOW_CHART_TERMINATOR":         true,
}

func runSlidesAddShape(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")
	shapeType, _ := cmd.Flags().GetString("type")
	x, _ := cmd.Flags().GetFloat64("x")
	y, _ := cmd.Flags().GetFloat64("y")
	width, _ := cmd.Flags().GetFloat64("width")
	height, _ := cmd.Flags().GetFloat64("height")

	// Validate shape type
	if !validShapeTypes[shapeType] {
		return p.PrintError(fmt.Errorf("invalid shape type '%s'. Common types: RECTANGLE, ELLIPSE, TEXT_BOX, TRIANGLE, ARROW, STAR_5", shapeType))
	}

	slideID, err := getSlideID(svc, presentationID, slideIDFlag, slideNumber)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*slides.Request{
		{
			CreateShape: &slides.CreateShapeRequest{
				ShapeType: shapeType,
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideID,
					Size: &slides.Size{
						Width:  &slides.Dimension{Magnitude: width, Unit: "PT"},
						Height: &slides.Dimension{Magnitude: height, Unit: "PT"},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1,
						ScaleY:     1,
						TranslateX: x,
						TranslateY: y,
						Unit:       "PT",
					},
				},
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add shape: %w", err))
	}

	// Get the shape object ID from response
	var shapeObjectID string
	if len(resp.Replies) > 0 && resp.Replies[0].CreateShape != nil {
		shapeObjectID = resp.Replies[0].CreateShape.ObjectId
	}

	result := map[string]interface{}{
		"status":          "created",
		"presentation_id": presentationID,
		"slide_id":        slideID,
		"shape_id":        shapeObjectID,
		"shape_type":      shapeType,
		"position":        map[string]float64{"x": x, "y": y},
		"size":            map[string]float64{"width": width, "height": height},
	}

	return p.Print(result)
}

func runSlidesAddImage(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")
	imageURL, _ := cmd.Flags().GetString("url")
	x, _ := cmd.Flags().GetFloat64("x")
	y, _ := cmd.Flags().GetFloat64("y")
	width, _ := cmd.Flags().GetFloat64("width")

	slideID, err := getSlideID(svc, presentationID, slideIDFlag, slideNumber)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*slides.Request{
		{
			CreateImage: &slides.CreateImageRequest{
				Url: imageURL,
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideID,
					Size: &slides.Size{
						Width: &slides.Dimension{Magnitude: width, Unit: "PT"},
						// Height not set - will maintain aspect ratio
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1,
						ScaleY:     1,
						TranslateX: x,
						TranslateY: y,
						Unit:       "PT",
					},
				},
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add image: %w", err))
	}

	// Get the image object ID from response
	var imageObjectID string
	if len(resp.Replies) > 0 && resp.Replies[0].CreateImage != nil {
		imageObjectID = resp.Replies[0].CreateImage.ObjectId
	}

	result := map[string]interface{}{
		"status":          "created",
		"presentation_id": presentationID,
		"slide_id":        slideID,
		"image_id":        imageObjectID,
		"url":             imageURL,
		"position":        map[string]float64{"x": x, "y": y},
		"width":           width,
	}

	return p.Print(result)
}

func runSlidesAddText(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	// Parse flags first (before client creation for early validation)
	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	tableID, _ := cmd.Flags().GetString("table-id")
	row, _ := cmd.Flags().GetInt("row")
	col, _ := cmd.Flags().GetInt("col")
	text, _ := cmd.Flags().GetString("text")
	insertionIndex, _ := cmd.Flags().GetInt("at")
	notesMode, _ := cmd.Flags().GetBool("notes")
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")

	// Validate mutually exclusive flags (fail fast before network calls)
	modeCount := 0
	if objectID != "" {
		modeCount++
	}
	if tableID != "" {
		modeCount++
	}
	if notesMode {
		modeCount++
	}
	if modeCount > 1 {
		return p.PrintError(fmt.Errorf("--object-id, --table-id, and --notes are mutually exclusive"))
	}
	if modeCount == 0 {
		return p.PrintError(fmt.Errorf("must specify --object-id, --table-id, or --notes"))
	}

	// Validate table cell mode requires row and col
	if tableID != "" {
		if row < 0 {
			return p.PrintError(fmt.Errorf("--row is required when using --table-id (valid values: 0 or greater)"))
		}
		if col < 0 {
			return p.PrintError(fmt.Errorf("--col is required when using --table-id (valid values: 0 or greater)"))
		}
	}

	// Validate notes mode requires slide targeting
	if notesMode && slideIDFlag == "" && slideNumber == 0 {
		return p.PrintError(fmt.Errorf("--notes requires --slide-id or --slide-number"))
	}

	// Now create the client after validation passes
	ctx := context.Background()
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	// Resolve notes mode to an object ID
	if notesMode {
		presentation, err := svc.Presentations.Get(presentationID).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
		}

		slide, err := findSlide(presentation, slideIDFlag, slideNumber)
		if err != nil {
			return p.PrintError(err)
		}

		notesObjID, err := getSpeakerNotesObjectID(slide)
		if err != nil {
			return p.PrintError(fmt.Errorf("cannot target speaker notes: %w", err))
		}
		objectID = notesObjID
	}

	// Build the InsertText request
	insertTextReq := &slides.InsertTextRequest{
		Text:           text,
		InsertionIndex: int64(insertionIndex),
	}

	result := map[string]interface{}{
		"status":          "inserted",
		"presentation_id": presentationID,
		"text_length":     len(text),
		"position":        insertionIndex,
	}

	if tableID != "" {
		// Table cell mode
		insertTextReq.ObjectId = tableID
		insertTextReq.CellLocation = &slides.TableCellLocation{
			RowIndex:    int64(row),
			ColumnIndex: int64(col),
		}
		result["table_id"] = tableID
		result["row"] = row
		result["col"] = col
	} else {
		// Shape/text box mode (including resolved notes mode)
		insertTextReq.ObjectId = objectID
		result["object_id"] = objectID
		if notesMode {
			result["target"] = "speaker_notes"
		}
	}

	requests := []*slides.Request{
		{
			InsertText: insertTextReq,
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add text: %w", err))
	}

	return p.Print(result)
}

func runSlidesReplaceText(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	findText, _ := cmd.Flags().GetString("find")
	replaceText, _ := cmd.Flags().GetString("replace")
	matchCase, _ := cmd.Flags().GetBool("match-case")

	requests := []*slides.Request{
		{
			ReplaceAllText: &slides.ReplaceAllTextRequest{
				ContainsText: &slides.SubstringMatchCriteria{
					Text:      findText,
					MatchCase: matchCase,
				},
				ReplaceText: replaceText,
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to replace text: %w", err))
	}

	// Get replacement count from response
	var occurrences int64
	if len(resp.Replies) > 0 && resp.Replies[0].ReplaceAllText != nil {
		occurrences = resp.Replies[0].ReplaceAllText.OccurrencesChanged
	}

	return p.Print(map[string]interface{}{
		"status":              "replaced",
		"presentation_id":     presentationID,
		"find":                findText,
		"replace":             replaceText,
		"occurrences_changed": occurrences,
	})
}

// parseHexColor converts "#RRGGBB" to slides.RgbColor
func parseHexColor(hex string) (*slides.RgbColor, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return nil, fmt.Errorf("invalid hex color format: %s (expected #RRGGBB)", hex)
	}

	var r, g, b int64
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return nil, fmt.Errorf("invalid hex color: %s", hex)
	}

	return &slides.RgbColor{
		Red:   float64(r) / 255.0,
		Green: float64(g) / 255.0,
		Blue:  float64(b) / 255.0,
	}, nil
}

func runSlidesDeleteObject(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")

	requests := []*slides.Request{
		{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: objectID,
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete object: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "deleted",
		"presentation_id": presentationID,
		"object_id":       objectID,
	})
}

func runSlidesDeleteText(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	fromIndex, _ := cmd.Flags().GetInt("from")
	toIndex, _ := cmd.Flags().GetInt("to")
	notesMode, _ := cmd.Flags().GetBool("notes")
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")

	// Validate: need either --object-id or --notes
	if objectID == "" && !notesMode {
		return p.PrintError(fmt.Errorf("must specify --object-id or --notes"))
	}
	if objectID != "" && notesMode {
		return p.PrintError(fmt.Errorf("--object-id and --notes are mutually exclusive"))
	}

	// Validate notes mode requires slide targeting
	if notesMode && slideIDFlag == "" && slideNumber == 0 {
		return p.PrintError(fmt.Errorf("--notes requires --slide-id or --slide-number"))
	}

	ctx := context.Background()
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	// Resolve notes mode to an object ID
	if notesMode {
		presentation, err := svc.Presentations.Get(presentationID).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
		}

		slide, err := findSlide(presentation, slideIDFlag, slideNumber)
		if err != nil {
			return p.PrintError(err)
		}

		notesObjID, err := getSpeakerNotesObjectID(slide)
		if err != nil {
			return p.PrintError(fmt.Errorf("cannot target speaker notes: %w", err))
		}
		objectID = notesObjID
	}

	startIdx := int64(fromIndex)
	textRange := &slides.Range{
		StartIndex: &startIdx,
		Type:       "FIXED_RANGE",
	}

	// If toIndex is -1, delete to end (use ALL type)
	if toIndex < 0 {
		textRange.Type = "FROM_START_INDEX"
	} else {
		endIdx := int64(toIndex)
		textRange.EndIndex = &endIdx
	}

	requests := []*slides.Request{
		{
			DeleteText: &slides.DeleteTextRequest{
				ObjectId:  objectID,
				TextRange: textRange,
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete text: %w", err))
	}

	result := map[string]interface{}{
		"status":          "deleted",
		"presentation_id": presentationID,
		"object_id":       objectID,
		"from":            fromIndex,
	}
	if toIndex >= 0 {
		result["to"] = toIndex
	}

	return p.Print(result)
}

func runSlidesUpdateTextStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	fromIndex, _ := cmd.Flags().GetInt("from")
	toIndex, _ := cmd.Flags().GetInt("to")
	bold, _ := cmd.Flags().GetBool("bold")
	italic, _ := cmd.Flags().GetBool("italic")
	underline, _ := cmd.Flags().GetBool("underline")
	fontSize, _ := cmd.Flags().GetFloat64("font-size")
	fontFamily, _ := cmd.Flags().GetString("font-family")
	colorHex, _ := cmd.Flags().GetString("color")

	// Build text style and fields mask
	style := &slides.TextStyle{}
	var fields []string

	if cmd.Flags().Changed("bold") {
		style.Bold = bold
		fields = append(fields, "bold")
	}
	if cmd.Flags().Changed("italic") {
		style.Italic = italic
		fields = append(fields, "italic")
	}
	if cmd.Flags().Changed("underline") {
		style.Underline = underline
		fields = append(fields, "underline")
	}
	if fontSize > 0 {
		style.FontSize = &slides.Dimension{
			Magnitude: fontSize,
			Unit:      "PT",
		}
		fields = append(fields, "fontSize")
	}
	if fontFamily != "" {
		style.FontFamily = fontFamily
		fields = append(fields, "fontFamily")
	}
	if colorHex != "" {
		color, err := parseHexColor(colorHex)
		if err != nil {
			return p.PrintError(err)
		}
		style.ForegroundColor = &slides.OptionalColor{
			OpaqueColor: &slides.OpaqueColor{
				RgbColor: color,
			},
		}
		fields = append(fields, "foregroundColor")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no style changes specified"))
	}

	var textRange *slides.Range
	if toIndex < 0 {
		textRange = &slides.Range{
			Type: "ALL",
		}
	} else {
		startIdx := int64(fromIndex)
		endIdx := int64(toIndex)
		textRange = &slides.Range{
			StartIndex: &startIdx,
			EndIndex:   &endIdx,
			Type:       "FIXED_RANGE",
		}
	}

	requests := []*slides.Request{
		{
			UpdateTextStyle: &slides.UpdateTextStyleRequest{
				ObjectId:  objectID,
				TextRange: textRange,
				Style:     style,
				Fields:    strings.Join(fields, ","),
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update text style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "updated",
		"presentation_id": presentationID,
		"object_id":       objectID,
		"fields_updated":  fields,
	})
}

func runSlidesUpdateTransform(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	x, _ := cmd.Flags().GetFloat64("x")
	y, _ := cmd.Flags().GetFloat64("y")
	scaleX, _ := cmd.Flags().GetFloat64("scale-x")
	scaleY, _ := cmd.Flags().GetFloat64("scale-y")
	rotate, _ := cmd.Flags().GetFloat64("rotate")

	// Convert rotation from degrees to radians
	radians := rotate * math.Pi / 180.0
	cosR := math.Cos(radians)
	sinR := math.Sin(radians)

	transform := &slides.AffineTransform{
		ScaleX:     scaleX * cosR,
		ScaleY:     scaleY * cosR,
		ShearX:     -scaleX * sinR,
		ShearY:     scaleY * sinR,
		TranslateX: x,
		TranslateY: y,
		Unit:       "PT",
	}

	requests := []*slides.Request{
		{
			UpdatePageElementTransform: &slides.UpdatePageElementTransformRequest{
				ObjectId:  objectID,
				Transform: transform,
				ApplyMode: "ABSOLUTE",
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update transform: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "updated",
		"presentation_id": presentationID,
		"object_id":       objectID,
		"position":        map[string]float64{"x": x, "y": y},
		"scale":           map[string]float64{"x": scaleX, "y": scaleY},
		"rotation":        rotate,
	})
}

func runSlidesCreateTable(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")
	rows, _ := cmd.Flags().GetInt("rows")
	cols, _ := cmd.Flags().GetInt("cols")
	x, _ := cmd.Flags().GetFloat64("x")
	y, _ := cmd.Flags().GetFloat64("y")
	width, _ := cmd.Flags().GetFloat64("width")
	height, _ := cmd.Flags().GetFloat64("height")

	slideID, err := getSlideID(svc, presentationID, slideIDFlag, slideNumber)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*slides.Request{
		{
			CreateTable: &slides.CreateTableRequest{
				Rows:    int64(rows),
				Columns: int64(cols),
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideID,
					Size: &slides.Size{
						Width:  &slides.Dimension{Magnitude: width, Unit: "PT"},
						Height: &slides.Dimension{Magnitude: height, Unit: "PT"},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1,
						ScaleY:     1,
						TranslateX: x,
						TranslateY: y,
						Unit:       "PT",
					},
				},
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create table: %w", err))
	}

	var tableID string
	if len(resp.Replies) > 0 && resp.Replies[0].CreateTable != nil {
		tableID = resp.Replies[0].CreateTable.ObjectId
	}

	return p.Print(map[string]interface{}{
		"status":          "created",
		"presentation_id": presentationID,
		"slide_id":        slideID,
		"table_id":        tableID,
		"rows":            rows,
		"cols":            cols,
		"position":        map[string]float64{"x": x, "y": y},
		"size":            map[string]float64{"width": width, "height": height},
	})
}

func runSlidesInsertTableRows(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	tableID, _ := cmd.Flags().GetString("table-id")
	atIndex, _ := cmd.Flags().GetInt("at")
	count, _ := cmd.Flags().GetInt("count")
	below, _ := cmd.Flags().GetBool("below")

	requests := []*slides.Request{
		{
			InsertTableRows: &slides.InsertTableRowsRequest{
				TableObjectId: tableID,
				CellLocation: &slides.TableCellLocation{
					RowIndex: int64(atIndex),
				},
				InsertBelow: below,
				Number:      int64(count),
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to insert table rows: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "inserted",
		"presentation_id": presentationID,
		"table_id":        tableID,
		"at_row":          atIndex,
		"count":           count,
		"below":           below,
	})
}

func runSlidesDeleteTableRow(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	tableID, _ := cmd.Flags().GetString("table-id")
	rowIndex, _ := cmd.Flags().GetInt("row")

	requests := []*slides.Request{
		{
			DeleteTableRow: &slides.DeleteTableRowRequest{
				TableObjectId: tableID,
				CellLocation: &slides.TableCellLocation{
					RowIndex: int64(rowIndex),
				},
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete table row: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "deleted",
		"presentation_id": presentationID,
		"table_id":        tableID,
		"row":             rowIndex,
	})
}

func runSlidesUpdateTableCell(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	tableID, _ := cmd.Flags().GetString("table-id")
	rowIndex, _ := cmd.Flags().GetInt("row")
	colIndex, _ := cmd.Flags().GetInt("col")
	bgColor, _ := cmd.Flags().GetString("background-color")

	if bgColor == "" {
		return p.PrintError(fmt.Errorf("--background-color is required"))
	}

	color, err := parseHexColor(bgColor)
	if err != nil {
		return p.PrintError(err)
	}

	cellProps := &slides.TableCellProperties{
		TableCellBackgroundFill: &slides.TableCellBackgroundFill{
			SolidFill: &slides.SolidFill{
				Color: &slides.OpaqueColor{
					RgbColor: color,
				},
			},
		},
	}

	requests := []*slides.Request{
		{
			UpdateTableCellProperties: &slides.UpdateTableCellPropertiesRequest{
				ObjectId: tableID,
				TableRange: &slides.TableRange{
					Location: &slides.TableCellLocation{
						RowIndex:    int64(rowIndex),
						ColumnIndex: int64(colIndex),
					},
					RowSpan:    1,
					ColumnSpan: 1,
				},
				TableCellProperties: cellProps,
				Fields:              "tableCellBackgroundFill",
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update table cell: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":           "updated",
		"presentation_id":  presentationID,
		"table_id":         tableID,
		"row":              rowIndex,
		"col":              colIndex,
		"background_color": bgColor,
	})
}

func runSlidesUpdateTableBorder(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	tableID, _ := cmd.Flags().GetString("table-id")
	rowIndex, _ := cmd.Flags().GetInt("row")
	colIndex, _ := cmd.Flags().GetInt("col")
	border, _ := cmd.Flags().GetString("border")
	colorHex, _ := cmd.Flags().GetString("color")
	width, _ := cmd.Flags().GetFloat64("width")
	style, _ := cmd.Flags().GetString("style")

	// Build border properties
	borderProps := &slides.TableBorderProperties{
		Weight: &slides.Dimension{
			Magnitude: width,
			Unit:      "PT",
		},
	}

	var fields []string
	fields = append(fields, "weight")

	if colorHex != "" {
		color, err := parseHexColor(colorHex)
		if err != nil {
			return p.PrintError(err)
		}
		borderProps.TableBorderFill = &slides.TableBorderFill{
			SolidFill: &slides.SolidFill{
				Color: &slides.OpaqueColor{
					RgbColor: color,
				},
			},
		}
		fields = append(fields, "tableBorderFill")
	}

	// Map style to dash style
	dashStyle := "SOLID"
	switch style {
	case "dashed":
		dashStyle = "DASH"
	case "dotted":
		dashStyle = "DOT"
	}
	borderProps.DashStyle = dashStyle
	fields = append(fields, "dashStyle")

	// Determine which borders to update
	var borders []string
	switch border {
	case "all":
		borders = []string{"INNER_HORIZONTAL", "INNER_VERTICAL", "TOP", "BOTTOM", "LEFT", "RIGHT"}
	case "top":
		borders = []string{"TOP"}
	case "bottom":
		borders = []string{"BOTTOM"}
	case "left":
		borders = []string{"LEFT"}
	case "right":
		borders = []string{"RIGHT"}
	default:
		return p.PrintError(fmt.Errorf("invalid border: %s (use top, bottom, left, right, or all)", border))
	}

	requests := make([]*slides.Request, 0, len(borders))
	for _, borderPos := range borders {
		requests = append(requests, &slides.Request{
			UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
				ObjectId: tableID,
				TableRange: &slides.TableRange{
					Location: &slides.TableCellLocation{
						RowIndex:    int64(rowIndex),
						ColumnIndex: int64(colIndex),
					},
					RowSpan:    1,
					ColumnSpan: 1,
				},
				BorderPosition:        borderPos,
				TableBorderProperties: borderProps,
				Fields:                strings.Join(fields, ","),
			},
		})
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update table border: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "updated",
		"presentation_id": presentationID,
		"table_id":        tableID,
		"row":             rowIndex,
		"col":             colIndex,
		"border":          border,
		"style":           style,
		"width":           width,
	})
}

func runSlidesUpdateParagraphStyle(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	fromIndex, _ := cmd.Flags().GetInt("from")
	toIndex, _ := cmd.Flags().GetInt("to")
	alignment, _ := cmd.Flags().GetString("alignment")
	lineSpacing, _ := cmd.Flags().GetFloat64("line-spacing")
	spaceAbove, _ := cmd.Flags().GetFloat64("space-above")
	spaceBelow, _ := cmd.Flags().GetFloat64("space-below")

	paragraphStyle := &slides.ParagraphStyle{}
	var fields []string

	if alignment != "" {
		paragraphStyle.Alignment = alignment
		fields = append(fields, "alignment")
	}
	if lineSpacing > 0 {
		paragraphStyle.LineSpacing = lineSpacing
		fields = append(fields, "lineSpacing")
	}
	if spaceAbove > 0 {
		paragraphStyle.SpaceAbove = &slides.Dimension{
			Magnitude: spaceAbove,
			Unit:      "PT",
		}
		fields = append(fields, "spaceAbove")
	}
	if spaceBelow > 0 {
		paragraphStyle.SpaceBelow = &slides.Dimension{
			Magnitude: spaceBelow,
			Unit:      "PT",
		}
		fields = append(fields, "spaceBelow")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no paragraph style changes specified"))
	}

	var textRange *slides.Range
	if toIndex < 0 {
		textRange = &slides.Range{
			Type: "ALL",
		}
	} else {
		startIdx := int64(fromIndex)
		endIdx := int64(toIndex)
		textRange = &slides.Range{
			StartIndex: &startIdx,
			EndIndex:   &endIdx,
			Type:       "FIXED_RANGE",
		}
	}

	requests := []*slides.Request{
		{
			UpdateParagraphStyle: &slides.UpdateParagraphStyleRequest{
				ObjectId:  objectID,
				TextRange: textRange,
				Style:     paragraphStyle,
				Fields:    strings.Join(fields, ","),
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update paragraph style: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "updated",
		"presentation_id": presentationID,
		"object_id":       objectID,
		"fields_updated":  fields,
	})
}

func runSlidesUpdateShape(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	objectID, _ := cmd.Flags().GetString("object-id")
	bgColor, _ := cmd.Flags().GetString("background-color")
	outlineColor, _ := cmd.Flags().GetString("outline-color")
	outlineWidth, _ := cmd.Flags().GetFloat64("outline-width")

	shapeProps := &slides.ShapeProperties{}
	var fields []string

	if bgColor != "" {
		color, err := parseHexColor(bgColor)
		if err != nil {
			return p.PrintError(err)
		}
		shapeProps.ShapeBackgroundFill = &slides.ShapeBackgroundFill{
			SolidFill: &slides.SolidFill{
				Color: &slides.OpaqueColor{
					RgbColor: color,
				},
			},
		}
		fields = append(fields, "shapeBackgroundFill")
	}

	if outlineColor != "" || outlineWidth > 0 {
		outline := &slides.Outline{}
		if outlineColor != "" {
			color, err := parseHexColor(outlineColor)
			if err != nil {
				return p.PrintError(err)
			}
			outline.OutlineFill = &slides.OutlineFill{
				SolidFill: &slides.SolidFill{
					Color: &slides.OpaqueColor{
						RgbColor: color,
					},
				},
			}
		}
		if outlineWidth > 0 {
			outline.Weight = &slides.Dimension{
				Magnitude: outlineWidth,
				Unit:      "PT",
			}
		}
		shapeProps.Outline = outline
		fields = append(fields, "outline")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no shape properties specified"))
	}

	requests := []*slides.Request{
		{
			UpdateShapeProperties: &slides.UpdateShapePropertiesRequest{
				ObjectId:        objectID,
				ShapeProperties: shapeProps,
				Fields:          strings.Join(fields, ","),
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update shape: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "updated",
		"presentation_id": presentationID,
		"object_id":       objectID,
		"fields_updated":  fields,
	})
}

func runSlidesReorderSlides(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideIDsStr, _ := cmd.Flags().GetString("slide-ids")
	toPosition, _ := cmd.Flags().GetInt("to")

	slideIDs := strings.Split(slideIDsStr, ",")
	for i, id := range slideIDs {
		slideIDs[i] = strings.TrimSpace(id)
	}

	requests := []*slides.Request{
		{
			UpdateSlidesPosition: &slides.UpdateSlidesPositionRequest{
				SlideObjectIds: slideIDs,
				InsertionIndex: int64(toPosition),
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to reorder slides: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "reordered",
		"presentation_id": presentationID,
		"slide_ids":       slideIDs,
		"new_position":    toPosition,
	})
}

func runSlidesUpdateSlideBackground(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	presentationID := args[0]
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")
	colorHex, _ := cmd.Flags().GetString("color")
	imageURL, _ := cmd.Flags().GetString("image-url")

	// Validate: must specify exactly one of --color or --image-url
	if colorHex == "" && imageURL == "" {
		return p.PrintError(fmt.Errorf("must specify --color or --image-url"))
	}
	if colorHex != "" && imageURL != "" {
		return p.PrintError(fmt.Errorf("--color and --image-url are mutually exclusive"))
	}

	ctx := context.Background()
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	slideID, err := getSlideID(svc, presentationID, slideIDFlag, slideNumber)
	if err != nil {
		return p.PrintError(err)
	}

	pageProps := &slides.PageProperties{
		PageBackgroundFill: &slides.PageBackgroundFill{},
	}
	fields := "pageBackgroundFill"

	if colorHex != "" {
		color, err := parseHexColor(colorHex)
		if err != nil {
			return p.PrintError(err)
		}
		pageProps.PageBackgroundFill.SolidFill = &slides.SolidFill{
			Color: &slides.OpaqueColor{
				RgbColor: color,
			},
		}
	} else {
		pageProps.PageBackgroundFill.StretchedPictureFill = &slides.StretchedPictureFill{
			ContentUrl: imageURL,
		}
	}

	requests := []*slides.Request{
		{
			UpdatePageProperties: &slides.UpdatePagePropertiesRequest{
				ObjectId:       slideID,
				PageProperties: pageProps,
				Fields:         fields,
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update slide background: %w", err))
	}

	result := map[string]interface{}{
		"status":          "updated",
		"presentation_id": presentationID,
		"slide_id":        slideID,
	}
	if colorHex != "" {
		result["background_color"] = colorHex
	} else {
		result["background_image"] = imageURL
	}

	return p.Print(result)
}

func runSlidesListLayouts(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]

	presentation, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
	}

	layouts := make([]map[string]interface{}, 0, len(presentation.Layouts))
	for _, layout := range presentation.Layouts {
		info := map[string]interface{}{
			"id": layout.ObjectId,
		}
		if layout.LayoutProperties != nil {
			if layout.LayoutProperties.Name != "" {
				info["name"] = layout.LayoutProperties.Name
			}
			if layout.LayoutProperties.DisplayName != "" {
				info["display_name"] = layout.LayoutProperties.DisplayName
			}
			if layout.LayoutProperties.MasterObjectId != "" {
				info["master_id"] = layout.LayoutProperties.MasterObjectId
			}
		}
		layouts = append(layouts, info)
	}

	return p.Print(map[string]interface{}{
		"presentation_id": presentationID,
		"layouts":         layouts,
		"count":           len(layouts),
	})
}

func runSlidesAddLine(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	slideIDFlag, _ := cmd.Flags().GetString("slide-id")
	slideNumber, _ := cmd.Flags().GetInt("slide-number")
	lineType, _ := cmd.Flags().GetString("type")
	startX, _ := cmd.Flags().GetFloat64("start-x")
	startY, _ := cmd.Flags().GetFloat64("start-y")
	endX, _ := cmd.Flags().GetFloat64("end-x")
	endY, _ := cmd.Flags().GetFloat64("end-y")
	colorHex, _ := cmd.Flags().GetString("color")
	weight, _ := cmd.Flags().GetFloat64("weight")

	slideID, err := getSlideID(svc, presentationID, slideIDFlag, slideNumber)
	if err != nil {
		return p.PrintError(err)
	}

	// Map line type to category
	category := "STRAIGHT"
	if strings.HasPrefix(lineType, "BENT") {
		category = "BENT"
	} else if strings.HasPrefix(lineType, "CURVED") {
		category = "CURVED"
	}

	// Calculate size and position from start/end coordinates
	width := endX - startX
	height := endY - startY

	// Handle negative dimensions by adjusting position
	translateX := startX
	translateY := startY
	scaleX := 1.0
	scaleY := 1.0
	if width < 0 {
		width = -width
		translateX = endX
		scaleX = -1.0
	}
	if height < 0 {
		height = -height
		translateY = endY
		scaleY = -1.0
	}

	requests := []*slides.Request{
		{
			CreateLine: &slides.CreateLineRequest{
				Category: category,
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideID,
					Size: &slides.Size{
						Width:  &slides.Dimension{Magnitude: width, Unit: "PT"},
						Height: &slides.Dimension{Magnitude: height, Unit: "PT"},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     scaleX,
						ScaleY:     scaleY,
						TranslateX: translateX,
						TranslateY: translateY,
						Unit:       "PT",
					},
				},
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add line: %w", err))
	}

	var lineObjectID string
	if len(resp.Replies) > 0 && resp.Replies[0].CreateLine != nil {
		lineObjectID = resp.Replies[0].CreateLine.ObjectId
	}

	// Apply line styling if color or weight specified
	if lineObjectID != "" && (colorHex != "" || cmd.Flags().Changed("weight")) {
		lineProps := &slides.LineProperties{}
		var fields []string

		if colorHex != "" {
			color, err := parseHexColor(colorHex)
			if err != nil {
				return p.PrintError(err)
			}
			lineProps.LineFill = &slides.LineFill{
				SolidFill: &slides.SolidFill{
					Color: &slides.OpaqueColor{
						RgbColor: color,
					},
				},
			}
			fields = append(fields, "lineFill")
		}

		if cmd.Flags().Changed("weight") {
			lineProps.Weight = &slides.Dimension{
				Magnitude: weight,
				Unit:      "PT",
			}
			fields = append(fields, "weight")
		}

		if len(fields) > 0 {
			styleRequests := []*slides.Request{
				{
					UpdateLineProperties: &slides.UpdateLinePropertiesRequest{
						ObjectId:       lineObjectID,
						LineProperties: lineProps,
						Fields:         strings.Join(fields, ","),
					},
				},
			}

			_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
				Requests: styleRequests,
			}).Do()
			if err != nil {
				return p.PrintError(fmt.Errorf("line created but failed to apply styling: %w", err))
			}
		}
	}

	result := map[string]interface{}{
		"status":          "created",
		"presentation_id": presentationID,
		"slide_id":        slideID,
		"line_id":         lineObjectID,
		"line_type":       lineType,
		"start":           map[string]float64{"x": startX, "y": startY},
		"end":             map[string]float64{"x": endX, "y": endY},
	}

	return p.Print(result)
}

func runSlidesGroup(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	objectIDsStr, _ := cmd.Flags().GetString("object-ids")

	objectIDs := strings.Split(objectIDsStr, ",")
	for i, id := range objectIDs {
		objectIDs[i] = strings.TrimSpace(id)
	}

	if len(objectIDs) < 2 {
		return p.PrintError(fmt.Errorf("at least 2 element IDs are required for grouping"))
	}

	requests := []*slides.Request{
		{
			GroupObjects: &slides.GroupObjectsRequest{
				ChildrenObjectIds: objectIDs,
			},
		},
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to group objects: %w", err))
	}

	var groupID string
	if len(resp.Replies) > 0 && resp.Replies[0].GroupObjects != nil {
		groupID = resp.Replies[0].GroupObjects.ObjectId
	}

	return p.Print(map[string]interface{}{
		"status":          "grouped",
		"presentation_id": presentationID,
		"group_id":        groupID,
		"children":        objectIDs,
	})
}

func runSlidesUngroup(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Slides()
	if err != nil {
		return p.PrintError(err)
	}

	presentationID := args[0]
	groupID, _ := cmd.Flags().GetString("group-id")

	requests := []*slides.Request{
		{
			UngroupObjects: &slides.UngroupObjectsRequest{
				ObjectIds: []string{groupID},
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to ungroup objects: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "ungrouped",
		"presentation_id": presentationID,
		"group_id":        groupID,
	})
}
