package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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

func init() {
	rootCmd.AddCommand(slidesCmd)
	slidesCmd.AddCommand(slidesInfoCmd)
	slidesCmd.AddCommand(slidesListCmd)
	slidesCmd.AddCommand(slidesReadCmd)
	slidesCmd.AddCommand(slidesCreateCmd)
	slidesCmd.AddCommand(slidesAddSlideCmd)

	// Create flags
	slidesCreateCmd.Flags().String("title", "", "Presentation title (required)")
	slidesCreateCmd.MarkFlagRequired("title")

	// Add-slide flags
	slidesAddSlideCmd.Flags().String("title", "", "Slide title")
	slidesAddSlideCmd.Flags().String("body", "", "Slide body text")
	slidesAddSlideCmd.Flags().String("layout", "TITLE_AND_BODY", "Slide layout (TITLE_AND_BODY, TITLE_ONLY, BLANK, etc.)")
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

	// Generate a unique object ID for the new slide
	slideObjectID := fmt.Sprintf("slide_%d", time.Now().UnixNano())

	// Create requests for adding a slide
	requests := []*slides.Request{
		{
			CreateSlide: &slides.CreateSlideRequest{
				ObjectId: slideObjectID,
				SlideLayoutReference: &slides.LayoutReference{
					PredefinedLayout: layout,
				},
			},
		},
	}

	// Execute the create slide request first
	resp, err := svc.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add slide: %w", err))
	}

	// Get the created slide to find placeholder IDs
	presentation, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get presentation: %w", err))
	}

	// Find the new slide
	var newSlide *slides.Page
	for _, slide := range presentation.Slides {
		if slide.ObjectId == slideObjectID {
			newSlide = slide
			break
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
	result := map[string]interface{}{
		"status":          "added",
		"presentation_id": presentationID,
		"slide_id":        slideObjectID,
		"slide_number":    slideNumber,
	}

	if len(resp.Replies) > 0 && resp.Replies[0].CreateSlide != nil {
		result["slide_id"] = resp.Replies[0].CreateSlide.ObjectId
	}

	return p.Print(result)
}
