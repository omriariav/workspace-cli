package cmd

import (
	"context"
	"fmt"
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
	Long:  "Inserts text into an existing shape or text box on a slide.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesAddText,
}

var slidesReplaceTextCmd = &cobra.Command{
	Use:   "replace-text <presentation-id>",
	Short: "Find and replace text",
	Long:  "Replaces all occurrences of text across all slides in the presentation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSlidesReplaceText,
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

	// Create flags
	slidesCreateCmd.Flags().String("title", "", "Presentation title (required)")
	slidesCreateCmd.MarkFlagRequired("title")

	// Add-slide flags
	slidesAddSlideCmd.Flags().String("title", "", "Slide title")
	slidesAddSlideCmd.Flags().String("body", "", "Slide body text")
	slidesAddSlideCmd.Flags().String("layout", "TITLE_AND_BODY", "Slide layout (TITLE_AND_BODY, TITLE_ONLY, BLANK, etc.)")

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
	slidesAddTextCmd.Flags().String("object-id", "", "Object ID to insert text into (required)")
	slidesAddTextCmd.Flags().String("text", "", "Text to insert (required)")
	slidesAddTextCmd.Flags().Int("at", 0, "Position to insert at (0 = beginning)")
	slidesAddTextCmd.MarkFlagRequired("object-id")
	slidesAddTextCmd.MarkFlagRequired("text")

	// Replace-text flags
	slidesReplaceTextCmd.Flags().String("find", "", "Text to find (required)")
	slidesReplaceTextCmd.Flags().String("replace", "", "Replacement text (required)")
	slidesReplaceTextCmd.Flags().Bool("match-case", true, "Case-sensitive matching")
	slidesReplaceTextCmd.MarkFlagRequired("find")
	slidesReplaceTextCmd.MarkFlagRequired("replace")
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

	// If slide number provided, read specific slide
	if len(args) > 1 {
		var slideNum int
		fmt.Sscanf(args[1], "%d", &slideNum)

		if slideNum < 1 || slideNum > len(presentation.Slides) {
			return p.PrintError(fmt.Errorf("slide number %d out of range (1-%d)", slideNum, len(presentation.Slides)))
		}

		slide := presentation.Slides[slideNum-1]
		text := extractSlideText(slide)

		return p.Print(map[string]interface{}{
			"slide":  slideNum,
			"id":     slide.ObjectId,
			"text":   text,
			"title":  extractSlideTitle(slide),
			"layout": slide.SlideProperties.LayoutObjectId,
		})
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

	// Validate layout
	validLayouts := map[string]bool{
		"BLANK":                      true,
		"CAPTION_ONLY":               true,
		"TITLE":                      true,
		"TITLE_AND_BODY":             true,
		"TITLE_AND_TWO_COLUMNS":      true,
		"TITLE_ONLY":                 true,
		"SECTION_HEADER":             true,
		"SECTION_TITLE_AND_DESCRIPTION": true,
		"ONE_COLUMN_TEXT":            true,
		"MAIN_POINT":                 true,
		"BIG_NUMBER":                 true,
	}
	if !validLayouts[layout] {
		return p.PrintError(fmt.Errorf("invalid layout '%s'. Valid layouts: BLANK, TITLE, TITLE_AND_BODY, TITLE_AND_TWO_COLUMNS, TITLE_ONLY, SECTION_HEADER, CAPTION_ONLY, MAIN_POINT, BIG_NUMBER", layout))
	}

	// Create requests for adding a slide (let API generate the ID)
	requests := []*slides.Request{
		{
			CreateSlide: &slides.CreateSlideRequest{
				SlideLayoutReference: &slides.LayoutReference{
					PredefinedLayout: layout,
				},
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
	text, _ := cmd.Flags().GetString("text")
	insertionIndex, _ := cmd.Flags().GetInt("at")

	requests := []*slides.Request{
		{
			InsertText: &slides.InsertTextRequest{
				ObjectId:       objectID,
				Text:           text,
				InsertionIndex: int64(insertionIndex),
			},
		},
	}

	_, err = svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add text: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":          "inserted",
		"presentation_id": presentationID,
		"object_id":       objectID,
		"text_length":     len(text),
		"position":        insertionIndex,
	})
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
