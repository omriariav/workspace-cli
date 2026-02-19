package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/forms/v1"
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

var formsGetCmd = &cobra.Command{
	Use:   "get <form-id>",
	Short: "Get form details",
	Long:  "Gets metadata about a Google Form. Alias for 'forms info'.",
	Args:  cobra.ExactArgs(1),
	RunE:  runFormsInfo, // Same handler as info
}

var formsResponsesCmd = &cobra.Command{
	Use:   "responses <form-id>",
	Short: "Get form responses",
	Long:  "Gets all responses submitted to a form.",
	Args:  cobra.ExactArgs(1),
	RunE:  runFormsResponses,
}

var formsResponseCmd = &cobra.Command{
	Use:   "response <form-id>",
	Short: "Get a single form response",
	Long:  "Gets a specific response by ID from a form.",
	Args:  cobra.ExactArgs(1),
	RunE:  runFormsResponse,
}

var formsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new form",
	Long: `Creates a new blank Google Form with a title and optional description.

Examples:
  gws forms create --title "Feedback Survey"
  gws forms create --title "Team Poll" --description "Weekly team feedback"`,
	Args: cobra.NoArgs,
	RunE: runFormsCreate,
}

var formsUpdateCmd = &cobra.Command{
	Use:   "update <form-id>",
	Short: "Update a form",
	Long: `Updates a Google Form using a batch update request.

For simple updates (title/description), use flags:
  gws forms update <form-id> --title "New Title"
  gws forms update <form-id> --description "New description"

For advanced updates (adding questions, etc.), provide a JSON file:
  gws forms update <form-id> --file batch-update.json

The --file flag should contain a JSON file with a batchUpdate request body.
See https://developers.google.com/forms/api/reference/rest/v1/forms/batchUpdate`,
	Args: cobra.ExactArgs(1),
	RunE: runFormsUpdate,
}

func init() {
	rootCmd.AddCommand(formsCmd)
	formsCmd.AddCommand(formsInfoCmd)
	formsCmd.AddCommand(formsGetCmd)
	formsCmd.AddCommand(formsResponsesCmd)
	formsCmd.AddCommand(formsResponseCmd)
	formsCmd.AddCommand(formsCreateCmd)
	formsCmd.AddCommand(formsUpdateCmd)

	// response flags
	formsResponseCmd.Flags().String("response-id", "", "Response ID to retrieve (required)")
	formsResponseCmd.MarkFlagRequired("response-id")

	// create flags
	formsCreateCmd.Flags().String("title", "", "Form title (required)")
	formsCreateCmd.Flags().String("description", "", "Form description")
	formsCreateCmd.MarkFlagRequired("title")

	// update flags
	formsUpdateCmd.Flags().String("title", "", "New form title")
	formsUpdateCmd.Flags().String("description", "", "New form description")
	formsUpdateCmd.Flags().String("file", "", "Path to JSON file with batchUpdate request body")
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

func runFormsResponse(cmd *cobra.Command, args []string) error {
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
	responseID, _ := cmd.Flags().GetString("response-id")

	// Get the form to map question IDs to titles
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

	// Get specific response
	r, err := svc.Forms.Responses.Get(formID, responseID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get response: %w", err))
	}

	result := map[string]interface{}{
		"id":          r.ResponseId,
		"form_id":     formID,
		"form_title":  form.Info.Title,
		"create_time": r.CreateTime,
		"last_submit": r.LastSubmittedTime,
	}

	if r.RespondentEmail != "" {
		result["email"] = r.RespondentEmail
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
	result["answers"] = answers

	return p.Print(result)
}

func runFormsCreate(cmd *cobra.Command, args []string) error {
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

	title, _ := cmd.Flags().GetString("title")
	description, _ := cmd.Flags().GetString("description")

	newForm := &forms.Form{
		Info: &forms.Info{
			Title:         title,
			DocumentTitle: title,
		},
	}

	if description != "" {
		newForm.Info.Description = description
	}

	form, err := svc.Forms.Create(newForm).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create form: %w", err))
	}

	result := map[string]interface{}{
		"id":             form.FormId,
		"title":          form.Info.Title,
		"document_title": form.Info.DocumentTitle,
		"responder_uri":  form.ResponderUri,
	}

	if form.Info.Description != "" {
		result["description"] = form.Info.Description
	}

	return p.Print(result)
}

func runFormsUpdate(cmd *cobra.Command, args []string) error {
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
	filePath, _ := cmd.Flags().GetString("file")
	title, _ := cmd.Flags().GetString("title")
	description, _ := cmd.Flags().GetString("description")

	hasFile := cmd.Flags().Changed("file")
	hasTitle := cmd.Flags().Changed("title")
	hasDescription := cmd.Flags().Changed("description")

	if hasFile && (hasTitle || hasDescription) {
		return p.PrintError(fmt.Errorf("--file cannot be combined with --title or --description"))
	}

	var batchReq forms.BatchUpdateFormRequest

	if hasFile {
		// Read batch update request from file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to read file: %w", err))
		}

		if err := json.Unmarshal(data, &batchReq); err != nil {
			return p.PrintError(fmt.Errorf("failed to parse JSON: %w", err))
		}
	} else if hasTitle || hasDescription {
		// Build simple update request for title/description
		var requests []*forms.Request

		if hasTitle {
			requests = append(requests, &forms.Request{
				UpdateFormInfo: &forms.UpdateFormInfoRequest{
					Info: &forms.Info{
						Title: title,
					},
					UpdateMask: "title",
				},
			})
		}

		if hasDescription {
			requests = append(requests, &forms.Request{
				UpdateFormInfo: &forms.UpdateFormInfoRequest{
					Info: &forms.Info{
						Description: description,
					},
					UpdateMask: "description",
				},
			})
		}

		batchReq.Requests = requests
	} else {
		return p.PrintError(fmt.Errorf("provide --file, --title, or --description"))
	}

	resp, err := svc.Forms.BatchUpdate(formID, &batchReq).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update form: %w", err))
	}

	// After batch update, get the updated form for full details
	form, err := svc.Forms.Get(formID).Do()
	if err != nil {
		// If we can't fetch the updated form, return what we have
		return p.Print(map[string]interface{}{
			"id":            formID,
			"replies_count": len(resp.Replies),
			"status":        "updated",
		})
	}

	result := map[string]interface{}{
		"id":             form.FormId,
		"title":          form.Info.Title,
		"document_title": form.Info.DocumentTitle,
		"responder_uri":  form.ResponderUri,
		"status":         "updated",
		"replies_count":  len(resp.Replies),
	}

	if form.Info.Description != "" {
		result["description"] = form.Info.Description
	}

	return p.Print(result)
}
