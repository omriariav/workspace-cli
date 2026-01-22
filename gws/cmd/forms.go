package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/gws/internal/client"
	"github.com/omriariav/workspace-cli/gws/internal/printer"
	"github.com/spf13/cobra"
)

var formsCmd = &cobra.Command{
	Use:   "forms",
	Short: "Manage Google Forms",
	Long:  "Commands for interacting with Google Forms.",
}

var formsInfoCmd = &cobra.Command{
	Use:   "info <form-id>",
	Short: "Get form info",
	Long:  "Gets metadata about a Google Form.",
	Args:  cobra.ExactArgs(1),
	RunE:  runFormsInfo,
}

var formsResponsesCmd = &cobra.Command{
	Use:   "responses <form-id>",
	Short: "Get form responses",
	Long:  "Gets all responses submitted to a form.",
	Args:  cobra.ExactArgs(1),
	RunE:  runFormsResponses,
}

func init() {
	rootCmd.AddCommand(formsCmd)
	formsCmd.AddCommand(formsInfoCmd)
	formsCmd.AddCommand(formsResponsesCmd)
}

func runFormsInfo(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Forms()
	if err != nil {
		return p.PrintError(err)
	}

	formID := args[0]

	form, err := svc.Forms.Get(formID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get form: %w", err))
	}

	// Extract questions/items
	items := make([]map[string]interface{}, 0)
	if form.Items != nil {
		for _, item := range form.Items {
			itemInfo := map[string]interface{}{
				"id": item.ItemId,
			}

			if item.Title != "" {
				itemInfo["title"] = item.Title
			}
			if item.Description != "" {
				itemInfo["description"] = item.Description
			}

			// Determine question type
			if item.QuestionItem != nil {
				q := item.QuestionItem.Question
				if q != nil {
					itemInfo["required"] = q.Required

					if q.TextQuestion != nil {
						itemInfo["type"] = "text"
						itemInfo["paragraph"] = q.TextQuestion.Paragraph
					} else if q.ChoiceQuestion != nil {
						itemInfo["type"] = "choice"
						itemInfo["choice_type"] = q.ChoiceQuestion.Type
						options := make([]string, 0)
						for _, opt := range q.ChoiceQuestion.Options {
							options = append(options, opt.Value)
						}
						itemInfo["options"] = options
					} else if q.ScaleQuestion != nil {
						itemInfo["type"] = "scale"
						itemInfo["low"] = q.ScaleQuestion.Low
						itemInfo["high"] = q.ScaleQuestion.High
					} else if q.DateQuestion != nil {
						itemInfo["type"] = "date"
						itemInfo["include_time"] = q.DateQuestion.IncludeTime
					} else if q.TimeQuestion != nil {
						itemInfo["type"] = "time"
					} else if q.FileUploadQuestion != nil {
						itemInfo["type"] = "file_upload"
					}
				}
			} else if item.QuestionGroupItem != nil {
				itemInfo["type"] = "question_group"
			} else if item.PageBreakItem != nil {
				itemInfo["type"] = "page_break"
			} else if item.TextItem != nil {
				itemInfo["type"] = "text_block"
			} else if item.ImageItem != nil {
				itemInfo["type"] = "image"
			} else if item.VideoItem != nil {
				itemInfo["type"] = "video"
			}

			items = append(items, itemInfo)
		}
	}

	result := map[string]interface{}{
		"id":             form.FormId,
		"title":          form.Info.Title,
		"document_title": form.Info.DocumentTitle,
		"responder_uri":  form.ResponderUri,
		"items":          items,
		"item_count":     len(items),
	}

	if form.Info.Description != "" {
		result["description"] = form.Info.Description
	}

	return p.Print(result)
}

func runFormsResponses(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Forms()
	if err != nil {
		return p.PrintError(err)
	}

	formID := args[0]

	// First get form to map question IDs to titles
	form, err := svc.Forms.Get(formID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get form: %w", err))
	}

	// Build question ID to title map
	questionTitles := make(map[string]string)
	if form.Items != nil {
		for _, item := range form.Items {
			if item.QuestionItem != nil && item.QuestionItem.Question != nil {
				questionTitles[item.QuestionItem.Question.QuestionId] = item.Title
			}
		}
	}

	// Get responses
	resp, err := svc.Forms.Responses.List(formID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get responses: %w", err))
	}

	responses := make([]map[string]interface{}, 0)
	if resp.Responses != nil {
		for _, r := range resp.Responses {
			response := map[string]interface{}{
				"id":          r.ResponseId,
				"create_time": r.CreateTime,
				"last_submit": r.LastSubmittedTime,
			}

			if r.RespondentEmail != "" {
				response["email"] = r.RespondentEmail
			}

			// Extract answers
			answers := make(map[string]interface{})
			if r.Answers != nil {
				for qID, answer := range r.Answers {
					title := questionTitles[qID]
					if title == "" {
						title = qID
					}

					if answer.TextAnswers != nil && len(answer.TextAnswers.Answers) > 0 {
						// Text or choice answers
						if len(answer.TextAnswers.Answers) == 1 {
							answers[title] = answer.TextAnswers.Answers[0].Value
						} else {
							vals := make([]string, len(answer.TextAnswers.Answers))
							for i, a := range answer.TextAnswers.Answers {
								vals[i] = a.Value
							}
							answers[title] = vals
						}
					} else if answer.FileUploadAnswers != nil {
						// File uploads
						files := make([]map[string]string, 0)
						for _, f := range answer.FileUploadAnswers.Answers {
							files = append(files, map[string]string{
								"id":   f.FileId,
								"name": f.FileName,
							})
						}
						answers[title] = files
					}
				}
			}
			response["answers"] = answers

			responses = append(responses, response)
		}
	}

	return p.Print(map[string]interface{}{
		"form_id":        formID,
		"form_title":     form.Info.Title,
		"responses":      responses,
		"response_count": len(responses),
	})
}
