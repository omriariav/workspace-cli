package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

var gmailCmd = &cobra.Command{
	Use:   "gmail",
	Short: "Manage Gmail",
	Long:  "Commands for interacting with Gmail messages and threads.",
}

var gmailListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent messages/threads",
	Long:  "Lists recent email threads from your inbox.",
	RunE:  runGmailList,
}

var gmailReadCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a message",
	Long:  "Reads and displays the content of a specific email message.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailRead,
}

var gmailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an email",
	Long:  "Sends a new email message.",
	RunE:  runGmailSend,
}

var gmailLabelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "List all labels",
	Long:  "Lists all Gmail labels in the account.",
	RunE:  runGmailLabels,
}

var gmailLabelCmd = &cobra.Command{
	Use:   "label <message-id>",
	Short: "Add or remove labels",
	Long: `Adds or removes labels from a Gmail message.

Use --add and --remove to specify label names (comma-separated).
Use "gws gmail labels" to see available labels.

Examples:
  gws gmail label 18abc123 --add "STARRED"
  gws gmail label 18abc123 --add "ActionNeeded,IMPORTANT" --remove "INBOX"
  gws gmail label 18abc123 --remove "UNREAD"`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailLabel,
}

var gmailArchiveCmd = &cobra.Command{
	Use:   "archive <message-id>",
	Short: "Archive a message",
	Long: `Archives a Gmail message by removing the INBOX label.

Examples:
  gws gmail archive 18abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailArchive,
}

var gmailTrashCmd = &cobra.Command{
	Use:   "trash <message-id>",
	Short: "Trash a message",
	Long: `Moves a Gmail message to the trash.

Examples:
  gws gmail trash 18abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailTrash,
}

func init() {
	rootCmd.AddCommand(gmailCmd)
	gmailCmd.AddCommand(gmailListCmd)
	gmailCmd.AddCommand(gmailReadCmd)
	gmailCmd.AddCommand(gmailSendCmd)
	gmailCmd.AddCommand(gmailLabelsCmd)
	gmailCmd.AddCommand(gmailLabelCmd)
	gmailCmd.AddCommand(gmailArchiveCmd)
	gmailCmd.AddCommand(gmailTrashCmd)

	// List flags
	gmailListCmd.Flags().Int64("max", 10, "Maximum number of results")
	gmailListCmd.Flags().String("query", "", "Gmail search query (e.g., 'is:unread', 'from:someone@example.com')")

	// Send flags
	gmailSendCmd.Flags().String("to", "", "Recipient email address (required)")
	gmailSendCmd.Flags().String("subject", "", "Email subject (required)")
	gmailSendCmd.Flags().String("body", "", "Email body (required)")
	gmailSendCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	gmailSendCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	gmailSendCmd.MarkFlagRequired("to")
	gmailSendCmd.MarkFlagRequired("subject")
	gmailSendCmd.MarkFlagRequired("body")

	// Label flags
	gmailLabelCmd.Flags().String("add", "", "Label names to add (comma-separated)")
	gmailLabelCmd.Flags().String("remove", "", "Label names to remove (comma-separated)")
}

func runGmailList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	maxResults, _ := cmd.Flags().GetInt64("max")
	query, _ := cmd.Flags().GetString("query")

	// List threads (more useful than individual messages)
	call := svc.Users.Threads.List("me").MaxResults(maxResults)
	if query != "" {
		call = call.Q(query)
	}

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list threads: %w", err))
	}

	// Format results
	results := make([]map[string]interface{}, 0, len(resp.Threads))
	for _, thread := range resp.Threads {
		// Get thread details for snippet and subject
		threadDetail, err := svc.Users.Threads.Get("me", thread.Id).Format("metadata").MetadataHeaders("Subject", "From", "Date").Do()
		if err != nil {
			continue
		}

		threadInfo := map[string]interface{}{
			"id":      thread.Id,
			"snippet": thread.Snippet,
		}

		// Extract headers from first message
		if len(threadDetail.Messages) > 0 {
			msg := threadDetail.Messages[0]
			for _, header := range msg.Payload.Headers {
				switch header.Name {
				case "Subject":
					threadInfo["subject"] = header.Value
				case "From":
					threadInfo["from"] = header.Value
				case "Date":
					threadInfo["date"] = header.Value
				}
			}
		}

		results = append(results, threadInfo)
	}

	return p.Print(map[string]interface{}{
		"threads": results,
		"count":   len(results),
	})
}

func runGmailRead(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	messageID := args[0]

	msg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get message: %w", err))
	}

	result := map[string]interface{}{
		"id": msg.Id,
	}

	// Extract headers
	headers := make(map[string]string)
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Subject", "From", "To", "Date", "Cc", "Bcc":
			headers[strings.ToLower(header.Name)] = header.Value
		}
	}
	result["headers"] = headers

	// Extract body
	body := extractBody(msg.Payload)
	result["body"] = body

	// Labels
	result["labels"] = msg.LabelIds

	return p.Print(result)
}

func runGmailSend(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	to, _ := cmd.Flags().GetString("to")
	subject, _ := cmd.Flags().GetString("subject")
	body, _ := cmd.Flags().GetString("body")
	cc, _ := cmd.Flags().GetString("cc")
	bcc, _ := cmd.Flags().GetString("bcc")

	// Build RFC 2822 message
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(body)

	// Encode as base64url
	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))

	msg := &gmail.Message{
		Raw: raw,
	}

	sent, err := svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to send message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "sent",
		"message_id": sent.Id,
		"thread_id":  sent.ThreadId,
	})
}

func runGmailLabels(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list labels: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Labels))
	for _, label := range resp.Labels {
		l := map[string]interface{}{
			"id":   label.Id,
			"name": label.Name,
			"type": label.Type,
		}
		results = append(results, l)
	}

	return p.Print(map[string]interface{}{
		"labels": results,
		"count":  len(results),
	})
}

// fetchLabelMap fetches all Gmail labels and returns a case-insensitive name-to-ID map.
func fetchLabelMap(svc *gmail.Service) (map[string]string, error) {
	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}

	nameToID := make(map[string]string, len(resp.Labels))
	for _, label := range resp.Labels {
		nameToID[strings.ToUpper(label.Name)] = label.Id
	}

	return nameToID, nil
}

// resolveFromMap converts label names to IDs using a pre-fetched label map.
func resolveFromMap(labelMap map[string]string, names []string) ([]string, error) {
	ids := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id, ok := labelMap[strings.ToUpper(name)]
		if !ok {
			return nil, fmt.Errorf("label not found: %s", name)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// resolveLabelNames converts label names to label IDs.
// Gmail API requires IDs for modify, but users think in names.
func resolveLabelNames(svc *gmail.Service, names []string) ([]string, error) {
	labelMap, err := fetchLabelMap(svc)
	if err != nil {
		return nil, err
	}
	return resolveFromMap(labelMap, names)
}

func runGmailLabel(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	messageID := args[0]
	addStr, _ := cmd.Flags().GetString("add")
	removeStr, _ := cmd.Flags().GetString("remove")

	if addStr == "" && removeStr == "" {
		return p.PrintError(fmt.Errorf("at least one of --add or --remove is required"))
	}

	// Fetch label map once for both add and remove
	labelMap, err := fetchLabelMap(svc)
	if err != nil {
		return p.PrintError(err)
	}

	req := &gmail.ModifyMessageRequest{}

	if addStr != "" {
		ids, err := resolveFromMap(labelMap, strings.Split(addStr, ","))
		if err != nil {
			return p.PrintError(err)
		}
		req.AddLabelIds = ids
	}

	if removeStr != "" {
		ids, err := resolveFromMap(labelMap, strings.Split(removeStr, ","))
		if err != nil {
			return p.PrintError(err)
		}
		req.RemoveLabelIds = ids
	}

	msg, err := svc.Users.Messages.Modify("me", messageID, req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to modify labels: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "modified",
		"message_id": msg.Id,
		"labels":     msg.LabelIds,
	})
}

func runGmailArchive(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	messageID := args[0]

	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}

	msg, err := svc.Users.Messages.Modify("me", messageID, req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to archive message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "archived",
		"message_id": msg.Id,
		"labels":     msg.LabelIds,
	})
}

func runGmailTrash(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	messageID := args[0]

	msg, err := svc.Users.Messages.Trash("me", messageID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to trash message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "trashed",
		"message_id": msg.Id,
	})
}

// extractBody extracts the plain text body from a message payload.
func extractBody(payload *gmail.MessagePart) string {
	// Check if this part has data
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Check parts recursively, prefer text/plain
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" {
			if part.Body != nil && part.Body.Data != "" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return string(data)
				}
			}
		}
	}

	// Fall back to text/html if no plain text
	for _, part := range payload.Parts {
		if part.MimeType == "text/html" {
			if part.Body != nil && part.Body.Data != "" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return string(data)
				}
			}
		}
	}

	// Check nested multipart
	for _, part := range payload.Parts {
		if strings.HasPrefix(part.MimeType, "multipart/") {
			if body := extractBody(part); body != "" {
				return body
			}
		}
	}

	return ""
}
