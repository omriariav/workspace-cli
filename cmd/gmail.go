package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
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

var gmailArchiveThreadCmd = &cobra.Command{
	Use:   "archive-thread <thread-id>",
	Short: "Archive all messages in a thread",
	Long: `Archives all messages in a Gmail thread by removing the INBOX label
and marking them as read.

Examples:
  gws gmail archive-thread 18abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailArchiveThread,
}

var gmailEventIDCmd = &cobra.Command{
	Use:   "event-id <message-id>",
	Short: "Extract calendar event ID from an invite email",
	Long: `Extracts the Google Calendar event ID from a calendar invite email.

Parses the eid parameter from Google Calendar URLs in the email body
and base64 decodes it to extract the event ID.

Examples:
  gws gmail event-id 19c041be3fcd1b79
  gws gmail event-id 19c041be3fcd1b79 | jq -r '.event_id' | xargs -I{} gws calendar rsvp {} --response accepted`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailEventID,
}

var gmailReplyCmd = &cobra.Command{
	Use:   "reply <message-id>",
	Short: "Reply to a message",
	Long: `Replies to an existing email message within its thread.

Automatically populates the thread ID, subject (with Re: prefix),
and In-Reply-To/References headers from the original message.

Examples:
  gws gmail reply 18abc123 --body "Thanks, got it!"
  gws gmail reply 18abc123 --body "Adding someone" --cc extra@example.com
  gws gmail reply 18abc123 --body "Sounds good" --all`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailReply,
}

var gmailThreadCmd = &cobra.Command{
	Use:   "thread <thread-id>",
	Short: "Read a full thread",
	Long: `Reads and displays all messages in a Gmail thread (conversation).

Use the thread_id from "gws gmail list" to view the full conversation.

Examples:
  gws gmail thread 18abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailThread,
}

var gmailUntrashCmd = &cobra.Command{
	Use:   "untrash <message-id>",
	Short: "Remove a message from trash",
	Long:  "Removes a Gmail message from the trash, restoring it to its previous location.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailUntrash,
}

var gmailDeleteCmd = &cobra.Command{
	Use:   "delete <message-id>",
	Short: "Permanently delete a message",
	Long:  "Permanently deletes a Gmail message. This action cannot be undone.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailDelete,
}

var gmailBatchModifyCmd = &cobra.Command{
	Use:   "batch-modify",
	Short: "Modify labels on multiple messages",
	Long: `Modifies labels on multiple Gmail messages at once.

Examples:
  gws gmail batch-modify --ids "msg1,msg2,msg3" --add-labels "STARRED"
  gws gmail batch-modify --ids "msg1,msg2" --remove-labels "INBOX,UNREAD"`,
	RunE: runGmailBatchModify,
}

var gmailBatchDeleteCmd = &cobra.Command{
	Use:   "batch-delete",
	Short: "Permanently delete multiple messages",
	Long:  "Permanently deletes multiple Gmail messages at once. This action cannot be undone.",
	RunE:  runGmailBatchDelete,
}

var gmailTrashThreadCmd = &cobra.Command{
	Use:   "trash-thread <thread-id>",
	Short: "Move a thread to trash",
	Long:  "Moves all messages in a Gmail thread to the trash.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailTrashThread,
}

var gmailUntrashThreadCmd = &cobra.Command{
	Use:   "untrash-thread <thread-id>",
	Short: "Remove a thread from trash",
	Long:  "Removes all messages in a Gmail thread from the trash.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailUntrashThread,
}

var gmailDeleteThreadCmd = &cobra.Command{
	Use:   "delete-thread <thread-id>",
	Short: "Permanently delete a thread",
	Long:  "Permanently deletes all messages in a Gmail thread. This action cannot be undone.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailDeleteThread,
}

var gmailLabelInfoCmd = &cobra.Command{
	Use:   "label-info",
	Short: "Get label details",
	Long:  "Gets detailed information about a specific Gmail label.",
	RunE:  runGmailLabelInfo,
}

var gmailCreateLabelCmd = &cobra.Command{
	Use:   "create-label",
	Short: "Create a new label",
	Long:  "Creates a new Gmail label.",
	RunE:  runGmailCreateLabel,
}

var gmailUpdateLabelCmd = &cobra.Command{
	Use:   "update-label",
	Short: "Update a label",
	Long:  "Updates an existing Gmail label's name or visibility settings.",
	RunE:  runGmailUpdateLabel,
}

var gmailDeleteLabelCmd = &cobra.Command{
	Use:   "delete-label",
	Short: "Delete a label",
	Long:  "Permanently deletes a Gmail label. Messages with this label are not deleted.",
	RunE:  runGmailDeleteLabel,
}

var gmailDraftsCmd = &cobra.Command{
	Use:   "drafts",
	Short: "List drafts",
	Long:  "Lists Gmail drafts.",
	RunE:  runGmailDrafts,
}

var gmailDraftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Get a draft by ID",
	Long:  "Gets the content of a specific Gmail draft.",
	RunE:  runGmailDraft,
}

var gmailCreateDraftCmd = &cobra.Command{
	Use:   "create-draft",
	Short: "Create a draft",
	Long:  "Creates a new Gmail draft message.",
	RunE:  runGmailCreateDraft,
}

var gmailUpdateDraftCmd = &cobra.Command{
	Use:   "update-draft",
	Short: "Update a draft",
	Long:  "Replaces the content of an existing Gmail draft.",
	RunE:  runGmailUpdateDraft,
}

var gmailSendDraftCmd = &cobra.Command{
	Use:   "send-draft",
	Short: "Send an existing draft",
	Long:  "Sends an existing Gmail draft.",
	RunE:  runGmailSendDraft,
}

var gmailDeleteDraftCmd = &cobra.Command{
	Use:   "delete-draft",
	Short: "Delete a draft",
	Long:  "Permanently deletes a Gmail draft.",
	RunE:  runGmailDeleteDraft,
}

var gmailAttachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Download an attachment",
	Long:  "Downloads a Gmail message attachment to a local file.",
	RunE:  runGmailAttachment,
}

func init() {
	rootCmd.AddCommand(gmailCmd)
	gmailCmd.AddCommand(gmailListCmd)
	gmailCmd.AddCommand(gmailReadCmd)
	gmailCmd.AddCommand(gmailSendCmd)
	gmailCmd.AddCommand(gmailLabelsCmd)
	gmailCmd.AddCommand(gmailLabelCmd)
	gmailCmd.AddCommand(gmailArchiveCmd)
	gmailCmd.AddCommand(gmailArchiveThreadCmd)
	gmailCmd.AddCommand(gmailTrashCmd)
	gmailCmd.AddCommand(gmailThreadCmd)
	gmailCmd.AddCommand(gmailEventIDCmd)
	gmailCmd.AddCommand(gmailReplyCmd)

	// List flags
	gmailListCmd.Flags().Int64("max", 10, "Maximum number of results (use --all for unlimited)")
	gmailListCmd.Flags().String("query", "", "Gmail search query (e.g., 'is:unread', 'from:someone@example.com')")
	gmailListCmd.Flags().Bool("all", false, "Fetch all matching results (may take time for large result sets)")
	gmailListCmd.Flags().Bool("include-labels", false, "Include Gmail label IDs in output")

	// Send flags
	gmailSendCmd.Flags().String("to", "", "Recipient email address (required)")
	gmailSendCmd.Flags().String("subject", "", "Email subject (required)")
	gmailSendCmd.Flags().String("body", "", "Email body (required)")
	gmailSendCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	gmailSendCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	gmailSendCmd.Flags().String("thread-id", "", "Thread ID to reply in")
	gmailSendCmd.Flags().String("reply-to-message-id", "", "Message ID to reply to (sets In-Reply-To/References headers)")
	gmailSendCmd.MarkFlagRequired("to")
	gmailSendCmd.MarkFlagRequired("subject")
	gmailSendCmd.MarkFlagRequired("body")

	// Reply flags
	gmailReplyCmd.Flags().String("body", "", "Reply body (required)")
	gmailReplyCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	gmailReplyCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	gmailReplyCmd.Flags().Bool("all", false, "Reply to all recipients")
	gmailReplyCmd.MarkFlagRequired("body")

	// Label flags
	gmailLabelCmd.Flags().String("add", "", "Label names to add (comma-separated)")
	gmailLabelCmd.Flags().String("remove", "", "Label names to remove (comma-separated)")

	// New commands
	gmailCmd.AddCommand(gmailUntrashCmd)
	gmailCmd.AddCommand(gmailDeleteCmd)
	gmailCmd.AddCommand(gmailBatchModifyCmd)
	gmailCmd.AddCommand(gmailBatchDeleteCmd)
	gmailCmd.AddCommand(gmailTrashThreadCmd)
	gmailCmd.AddCommand(gmailUntrashThreadCmd)
	gmailCmd.AddCommand(gmailDeleteThreadCmd)
	gmailCmd.AddCommand(gmailLabelInfoCmd)
	gmailCmd.AddCommand(gmailCreateLabelCmd)
	gmailCmd.AddCommand(gmailUpdateLabelCmd)
	gmailCmd.AddCommand(gmailDeleteLabelCmd)
	gmailCmd.AddCommand(gmailDraftsCmd)
	gmailCmd.AddCommand(gmailDraftCmd)
	gmailCmd.AddCommand(gmailCreateDraftCmd)
	gmailCmd.AddCommand(gmailUpdateDraftCmd)
	gmailCmd.AddCommand(gmailSendDraftCmd)
	gmailCmd.AddCommand(gmailDeleteDraftCmd)
	gmailCmd.AddCommand(gmailAttachmentCmd)

	// Batch modify flags
	gmailBatchModifyCmd.Flags().String("ids", "", "Comma-separated message IDs (required)")
	gmailBatchModifyCmd.Flags().String("add-labels", "", "Label names to add (comma-separated)")
	gmailBatchModifyCmd.Flags().String("remove-labels", "", "Label names to remove (comma-separated)")
	gmailBatchModifyCmd.MarkFlagRequired("ids")

	// Batch delete flags
	gmailBatchDeleteCmd.Flags().String("ids", "", "Comma-separated message IDs (required)")
	gmailBatchDeleteCmd.MarkFlagRequired("ids")

	// Label info flags
	gmailLabelInfoCmd.Flags().String("id", "", "Label ID (required)")
	gmailLabelInfoCmd.MarkFlagRequired("id")

	// Create label flags
	gmailCreateLabelCmd.Flags().String("name", "", "Label name (required)")
	gmailCreateLabelCmd.Flags().String("visibility", "", "Message visibility: labelShow, labelShowIfUnread, labelHide")
	gmailCreateLabelCmd.Flags().String("list-visibility", "", "Label list visibility: labelShow, labelHide")
	gmailCreateLabelCmd.MarkFlagRequired("name")

	// Update label flags
	gmailUpdateLabelCmd.Flags().String("id", "", "Label ID (required)")
	gmailUpdateLabelCmd.Flags().String("name", "", "New label name")
	gmailUpdateLabelCmd.Flags().String("visibility", "", "Message visibility: labelShow, labelShowIfUnread, labelHide")
	gmailUpdateLabelCmd.Flags().String("list-visibility", "", "Label list visibility: labelShow, labelHide")
	gmailUpdateLabelCmd.MarkFlagRequired("id")

	// Delete label flags
	gmailDeleteLabelCmd.Flags().String("id", "", "Label ID (required)")
	gmailDeleteLabelCmd.MarkFlagRequired("id")

	// Drafts list flags
	gmailDraftsCmd.Flags().Int64("max", 10, "Maximum number of results")
	gmailDraftsCmd.Flags().String("query", "", "Gmail search query")

	// Draft get flags
	gmailDraftCmd.Flags().String("id", "", "Draft ID (required)")
	gmailDraftCmd.MarkFlagRequired("id")

	// Create draft flags
	gmailCreateDraftCmd.Flags().String("to", "", "Recipient email address (required)")
	gmailCreateDraftCmd.Flags().String("subject", "", "Email subject")
	gmailCreateDraftCmd.Flags().String("body", "", "Email body")
	gmailCreateDraftCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	gmailCreateDraftCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	gmailCreateDraftCmd.Flags().String("thread-id", "", "Thread ID for reply draft")
	gmailCreateDraftCmd.MarkFlagRequired("to")

	// Update draft flags
	gmailUpdateDraftCmd.Flags().String("id", "", "Draft ID (required)")
	gmailUpdateDraftCmd.Flags().String("to", "", "Recipient email address")
	gmailUpdateDraftCmd.Flags().String("subject", "", "Email subject")
	gmailUpdateDraftCmd.Flags().String("body", "", "Email body")
	gmailUpdateDraftCmd.Flags().String("cc", "", "CC recipients (comma-separated)")
	gmailUpdateDraftCmd.Flags().String("bcc", "", "BCC recipients (comma-separated)")
	gmailUpdateDraftCmd.MarkFlagRequired("id")

	// Send draft flags
	gmailSendDraftCmd.Flags().String("id", "", "Draft ID (required)")
	gmailSendDraftCmd.MarkFlagRequired("id")

	// Delete draft flags
	gmailDeleteDraftCmd.Flags().String("id", "", "Draft ID (required)")
	gmailDeleteDraftCmd.MarkFlagRequired("id")

	// Attachment flags
	gmailAttachmentCmd.Flags().String("message-id", "", "Message ID (required)")
	gmailAttachmentCmd.Flags().String("id", "", "Attachment ID (required)")
	gmailAttachmentCmd.Flags().String("output", "", "Output file path (required)")
	gmailAttachmentCmd.MarkFlagRequired("message-id")
	gmailAttachmentCmd.MarkFlagRequired("id")
	gmailAttachmentCmd.MarkFlagRequired("output")
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
	fetchAll, _ := cmd.Flags().GetBool("all")
	includeLabels, _ := cmd.Flags().GetBool("include-labels")

	// Gmail API has a hard limit of 500 results per request
	const apiMaxPerPage int64 = 500

	// Collect all threads using pagination
	var allThreads []*gmail.Thread
	var pageToken string
	pageNum := 1

	for {
		// Determine how many to fetch in this request
		perPage := apiMaxPerPage
		if !fetchAll && maxResults > 0 {
			remaining := maxResults - int64(len(allThreads))
			if remaining <= 0 {
				break
			}
			if remaining < perPage {
				perPage = remaining
			}
		}

		call := svc.Users.Threads.List("me").MaxResults(perPage)
		if query != "" {
			call = call.Q(query)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list threads: %w", err))
		}

		allThreads = append(allThreads, resp.Threads...)

		// Progress indicator for multi-page fetches (to stderr)
		if resp.NextPageToken != "" && (fetchAll || maxResults > apiMaxPerPage) {
			fmt.Fprintf(os.Stderr, "Fetched page %d (%d threads so far)...\n", pageNum, len(allThreads))
		}

		// Check if we should continue
		if resp.NextPageToken == "" {
			break
		}
		if !fetchAll && int64(len(allThreads)) >= maxResults {
			break
		}

		pageToken = resp.NextPageToken
		pageNum++
	}

	// Trim to max if we fetched more (can happen due to page boundaries)
	if !fetchAll && maxResults > 0 && int64(len(allThreads)) > maxResults {
		allThreads = allThreads[:maxResults]
	}

	// Format results
	results := make([]map[string]interface{}, 0, len(allThreads))
	for _, thread := range allThreads {
		// Get thread details for snippet and subject
		threadDetail, err := svc.Users.Threads.Get("me", thread.Id).Format("metadata").MetadataHeaders("Subject", "From", "Date").Do()
		if err != nil {
			continue
		}

		threadInfo := map[string]interface{}{
			"thread_id":     thread.Id,
			"snippet":       thread.Snippet,
			"message_count": len(threadDetail.Messages),
		}

		// Extract latest message ID and headers from first message
		if len(threadDetail.Messages) > 0 {
			// Latest message ID (for use with read, label, archive, trash)
			latestMsg := threadDetail.Messages[len(threadDetail.Messages)-1]
			threadInfo["message_id"] = latestMsg.Id

			// Headers from first message (thread subject/sender)
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

			if includeLabels {
				labelSet := make(map[string]bool)
				for _, m := range threadDetail.Messages {
					for _, lbl := range m.LabelIds {
						labelSet[lbl] = true
					}
				}
				labels := make([]string, 0, len(labelSet))
				for lbl := range labelSet {
					labels = append(labels, lbl)
				}
				sort.Strings(labels)
				threadInfo["labels"] = labels
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
	threadID, _ := cmd.Flags().GetString("thread-id")
	replyToMsgID, _ := cmd.Flags().GetString("reply-to-message-id")

	// If replying, fetch the original message's Message-ID and References headers
	var inReplyTo, origReferences string
	if replyToMsgID != "" {
		origMsg, err := svc.Users.Messages.Get("me", replyToMsgID).Format("metadata").MetadataHeaders("Message-ID", "Message-Id", "References").Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get original message for reply: %w", err))
		}
		for _, header := range origMsg.Payload.Headers {
			switch header.Name {
			case "Message-ID", "Message-Id":
				inReplyTo = header.Value
			case "References":
				origReferences = header.Value
			}
		}
		// Default thread ID from original message if not specified
		if threadID == "" {
			threadID = origMsg.ThreadId
		}
	}

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
	if inReplyTo != "" {
		msgBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", inReplyTo))
		references := inReplyTo
		if origReferences != "" {
			references = origReferences + " " + inReplyTo
		}
		msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", references))
	}
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(body)

	// Encode as base64url
	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))

	msg := &gmail.Message{
		Raw: raw,
	}
	if threadID != "" {
		msg.ThreadId = threadID
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
	p := GetPrinter()
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
	p := GetPrinter()
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

func runGmailArchiveThread(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	threadID := args[0]

	// Fetch thread with minimal format to get message IDs
	thread, err := svc.Users.Threads.Get("me", threadID).Format("minimal").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get thread: %w", err))
	}

	archived := 0
	failed := 0
	for _, msg := range thread.Messages {
		// Remove INBOX and UNREAD labels
		req := &gmail.ModifyMessageRequest{
			RemoveLabelIds: []string{"INBOX", "UNREAD"},
		}
		_, err := svc.Users.Messages.Modify("me", msg.Id, req).Do()
		if err != nil {
			failed++
			continue
		}
		archived++
	}

	return p.Print(map[string]interface{}{
		"status":    "archived",
		"thread_id": threadID,
		"archived":  archived,
		"failed":    failed,
		"total":     len(thread.Messages),
	})
}

func runGmailTrash(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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

func runGmailThread(cmd *cobra.Command, args []string) error {
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

	threadID := args[0]

	thread, err := svc.Users.Threads.Get("me", threadID).Format("full").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get thread: %w", err))
	}

	messages := make([]map[string]interface{}, 0, len(thread.Messages))
	for _, msg := range thread.Messages {
		msgInfo := map[string]interface{}{
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
		msgInfo["headers"] = headers

		// Extract body
		msgInfo["body"] = extractBody(msg.Payload)

		// Labels
		msgInfo["labels"] = msg.LabelIds

		messages = append(messages, msgInfo)
	}

	return p.Print(map[string]interface{}{
		"thread_id":     threadID,
		"message_count": len(messages),
		"messages":      messages,
	})
}

func runGmailEventID(cmd *cobra.Command, args []string) error {
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

	body := extractBody(msg.Payload)

	// Try to extract event ID from Google Calendar URL eid parameter
	eventID, err := extractEventIDFromBody(body)
	if err != nil {
		return p.PrintError(fmt.Errorf("no calendar event ID found in message: %w", err))
	}

	result := map[string]interface{}{
		"message_id": messageID,
		"event_id":   eventID,
	}

	// Extract subject for context
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			result["subject"] = header.Value
			break
		}
	}

	return p.Print(result)
}

// extractEventIDFromBody parses a Google Calendar eid from the email body.
func extractEventIDFromBody(body string) (string, error) {
	// Pattern: look for eid= parameter in Google Calendar URLs
	// Match broadly (any non-whitespace, non-& chars) to capture URL-encoded values
	re := regexp.MustCompile(`[?&]eid=([^\s&"<>]+)`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("no eid parameter found")
	}

	eidEncoded := matches[1]

	// URL-decode first (in case of URL encoding)
	eidEncoded, _ = url.QueryUnescape(eidEncoded)

	// Base64 decode (standard encoding, pad if needed)
	// Google uses standard base64, but we try both
	var decoded []byte
	var err error
	decoded, err = base64.StdEncoding.DecodeString(eidEncoded)
	if err != nil {
		// Try URL-safe encoding
		decoded, err = base64.URLEncoding.DecodeString(eidEncoded)
		if err != nil {
			// Try without padding
			decoded, err = base64.RawStdEncoding.DecodeString(eidEncoded)
			if err != nil {
				return "", fmt.Errorf("failed to decode eid: %w", err)
			}
		}
	}

	// The decoded value is "eventID email@domain.com" — take the first part
	parts := strings.SplitN(string(decoded), " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("decoded eid is empty")
	}

	return parts[0], nil
}

func runGmailReply(cmd *cobra.Command, args []string) error {
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
	body, _ := cmd.Flags().GetString("body")
	cc, _ := cmd.Flags().GetString("cc")
	bcc, _ := cmd.Flags().GetString("bcc")
	replyAll, _ := cmd.Flags().GetBool("all")

	// Fetch the original message
	origMsg, err := svc.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get original message: %w", err))
	}

	// Extract headers from original
	var origSubject, origFrom, origTo, origCc, origMessageID, origReferences string
	for _, header := range origMsg.Payload.Headers {
		switch header.Name {
		case "Subject":
			origSubject = header.Value
		case "From":
			origFrom = header.Value
		case "To":
			origTo = header.Value
		case "Cc":
			origCc = header.Value
		case "Message-ID", "Message-Id":
			origMessageID = header.Value
		case "References":
			origReferences = header.Value
		}
	}

	// Build reply To: reply to sender
	replyTo := origFrom

	// For reply-all, add original To and Cc (excluding self)
	if replyAll {
		// Get user's email to exclude from recipients
		profile, err := svc.Users.GetProfile("me").Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get profile for reply-all: %w", err))
		}
		myEmail := strings.ToLower(profile.EmailAddress)
		var additionalRecipients []string

		// Add original To recipients (minus self)
		for _, addr := range strings.Split(origTo, ",") {
			addr = strings.TrimSpace(addr)
			if addr != "" && !emailMatchesSelf(addr, myEmail) {
				additionalRecipients = append(additionalRecipients, addr)
			}
		}

		if len(additionalRecipients) > 0 {
			replyTo = replyTo + ", " + strings.Join(additionalRecipients, ", ")
		}

		// Add original Cc to cc
		if origCc != "" {
			var ccRecipients []string
			for _, addr := range strings.Split(origCc, ",") {
				addr = strings.TrimSpace(addr)
				if addr != "" && !emailMatchesSelf(addr, myEmail) {
					ccRecipients = append(ccRecipients, addr)
				}
			}
			if len(ccRecipients) > 0 {
				if cc != "" {
					cc = cc + ", " + strings.Join(ccRecipients, ", ")
				} else {
					cc = strings.Join(ccRecipients, ", ")
				}
			}
		}
	}

	// Build subject with Re: prefix
	replySubject := origSubject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}

	// Build RFC 2822 message with threading headers
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", replyTo))
	if cc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", replySubject))
	if origMessageID != "" {
		msgBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", origMessageID))
		// Chain References: original References + original Message-ID
		references := origMessageID
		if origReferences != "" {
			references = origReferences + " " + origMessageID
		}
		msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", references))
	}
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))

	msg := &gmail.Message{
		Raw:      raw,
		ThreadId: origMsg.ThreadId,
	}

	sent, err := svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to send reply: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "sent",
		"message_id":  sent.Id,
		"thread_id":   sent.ThreadId,
		"in_reply_to": messageID,
	})
}

// emailMatchesSelf checks if an RFC 5322 address matches the user's email.
// Handles both "user@domain.com" and "Name <user@domain.com>" formats.
func emailMatchesSelf(addr string, myEmail string) bool {
	addr = strings.ToLower(strings.TrimSpace(addr))
	// Extract email from angle brackets if present: "Name <email>"
	if idx := strings.LastIndex(addr, "<"); idx >= 0 {
		end := strings.Index(addr[idx:], ">")
		if end > 0 {
			addr = addr[idx+1 : idx+end]
		}
	}
	return addr == myEmail
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

func runGmailUntrash(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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

	msg, err := svc.Users.Messages.Untrash("me", messageID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to untrash message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "untrashed",
		"message_id": msg.Id,
	})
}

func runGmailDelete(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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

	err = svc.Users.Messages.Delete("me", messageID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "deleted",
		"message_id": messageID,
	})
}

func runGmailBatchModify(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	idsStr, _ := cmd.Flags().GetString("ids")
	addLabelsStr, _ := cmd.Flags().GetString("add-labels")
	removeLabelsStr, _ := cmd.Flags().GetString("remove-labels")

	if addLabelsStr == "" && removeLabelsStr == "" {
		return p.PrintError(fmt.Errorf("at least one of --add-labels or --remove-labels is required"))
	}

	ids := strings.Split(idsStr, ",")
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
	}

	// Fetch label map once for both add and remove
	labelMap, err := fetchLabelMap(svc)
	if err != nil {
		return p.PrintError(err)
	}

	req := &gmail.BatchModifyMessagesRequest{
		Ids: ids,
	}

	if addLabelsStr != "" {
		addIDs, err := resolveFromMap(labelMap, strings.Split(addLabelsStr, ","))
		if err != nil {
			return p.PrintError(err)
		}
		req.AddLabelIds = addIDs
	}

	if removeLabelsStr != "" {
		removeIDs, err := resolveFromMap(labelMap, strings.Split(removeLabelsStr, ","))
		if err != nil {
			return p.PrintError(err)
		}
		req.RemoveLabelIds = removeIDs
	}

	err = svc.Users.Messages.BatchModify("me", req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch modify messages: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "modified",
		"count":  len(ids),
	})
}

func runGmailBatchDelete(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	idsStr, _ := cmd.Flags().GetString("ids")
	ids := strings.Split(idsStr, ",")
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
	}

	req := &gmail.BatchDeleteMessagesRequest{
		Ids: ids,
	}

	err = svc.Users.Messages.BatchDelete("me", req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch delete messages: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"count":  len(ids),
	})
}

func runGmailTrashThread(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	threadID := args[0]

	thread, err := svc.Users.Threads.Trash("me", threadID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to trash thread: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":    "trashed",
		"thread_id": thread.Id,
	})
}

func runGmailUntrashThread(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	threadID := args[0]

	thread, err := svc.Users.Threads.Untrash("me", threadID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to untrash thread: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":    "untrashed",
		"thread_id": thread.Id,
	})
}

func runGmailDeleteThread(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	threadID := args[0]

	err = svc.Users.Threads.Delete("me", threadID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete thread: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":    "deleted",
		"thread_id": threadID,
	})
}

func runGmailLabelInfo(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	labelID, _ := cmd.Flags().GetString("id")

	label, err := svc.Users.Labels.Get("me", labelID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get label: %w", err))
	}

	return p.Print(map[string]interface{}{
		"id":                      label.Id,
		"name":                    label.Name,
		"type":                    label.Type,
		"message_list_visibility": label.MessageListVisibility,
		"label_list_visibility":   label.LabelListVisibility,
		"messages_total":          label.MessagesTotal,
		"messages_unread":         label.MessagesUnread,
		"threads_total":           label.ThreadsTotal,
		"threads_unread":          label.ThreadsUnread,
	})
}

func runGmailCreateLabel(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	name, _ := cmd.Flags().GetString("name")
	visibility, _ := cmd.Flags().GetString("visibility")
	listVisibility, _ := cmd.Flags().GetString("list-visibility")

	label := &gmail.Label{
		Name: name,
	}
	if visibility != "" {
		label.MessageListVisibility = visibility
	}
	if listVisibility != "" {
		label.LabelListVisibility = listVisibility
	}

	created, err := svc.Users.Labels.Create("me", label).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create label: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "created",
		"id":     created.Id,
		"name":   created.Name,
	})
}

func runGmailUpdateLabel(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	labelID, _ := cmd.Flags().GetString("id")
	name, _ := cmd.Flags().GetString("name")
	visibility, _ := cmd.Flags().GetString("visibility")
	listVisibility, _ := cmd.Flags().GetString("list-visibility")

	current, err := svc.Users.Labels.Get("me", labelID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get label: %w", err))
	}

	if name != "" {
		current.Name = name
	}
	if visibility != "" {
		current.MessageListVisibility = visibility
	}
	if listVisibility != "" {
		current.LabelListVisibility = listVisibility
	}

	updated, err := svc.Users.Labels.Update("me", labelID, current).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update label: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "updated",
		"id":     updated.Id,
		"name":   updated.Name,
	})
}

func runGmailDeleteLabel(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	labelID, _ := cmd.Flags().GetString("id")

	err = svc.Users.Labels.Delete("me", labelID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete label: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"id":     labelID,
	})
}

func runGmailDrafts(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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

	call := svc.Users.Drafts.List("me").MaxResults(maxResults)
	if query != "" {
		call = call.Q(query)
	}

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list drafts: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Drafts))
	for _, draft := range resp.Drafts {
		d := map[string]interface{}{
			"id": draft.Id,
		}
		if draft.Message != nil {
			d["message_id"] = draft.Message.Id
		}
		results = append(results, d)
	}

	return p.Print(map[string]interface{}{
		"drafts": results,
		"count":  len(results),
	})
}

func runGmailDraft(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	draftID, _ := cmd.Flags().GetString("id")

	draft, err := svc.Users.Drafts.Get("me", draftID).Format("full").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get draft: %w", err))
	}

	result := map[string]interface{}{
		"id": draft.Id,
	}

	if draft.Message != nil {
		result["message_id"] = draft.Message.Id

		if draft.Message.Payload != nil {
			headers := make(map[string]string)
			for _, header := range draft.Message.Payload.Headers {
				switch header.Name {
				case "Subject", "From", "To", "Date", "Cc", "Bcc":
					headers[strings.ToLower(header.Name)] = header.Value
				}
			}
			result["headers"] = headers
			result["body"] = extractBody(draft.Message.Payload)
		}
	}

	return p.Print(result)
}

func runGmailCreateDraft(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	threadID, _ := cmd.Flags().GetString("thread-id")

	// Build RFC 2822 message
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}
	if subject != "" {
		msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	}
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	if body != "" {
		msgBuilder.WriteString(body)
	}

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))

	msg := &gmail.Message{Raw: raw}
	if threadID != "" {
		msg.ThreadId = threadID
	}

	draft := &gmail.Draft{
		Message: msg,
	}

	created, err := svc.Users.Drafts.Create("me", draft).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create draft: %w", err))
	}

	result := map[string]interface{}{
		"status":   "created",
		"draft_id": created.Id,
	}
	if created.Message != nil {
		result["message_id"] = created.Message.Id
	}

	return p.Print(result)
}

func runGmailUpdateDraft(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	draftID, _ := cmd.Flags().GetString("id")
	to, _ := cmd.Flags().GetString("to")
	subject, _ := cmd.Flags().GetString("subject")
	body, _ := cmd.Flags().GetString("body")
	cc, _ := cmd.Flags().GetString("cc")
	bcc, _ := cmd.Flags().GetString("bcc")

	// Build RFC 2822 message
	var msgBuilder strings.Builder
	if to != "" {
		msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	}
	if cc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		msgBuilder.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}
	if subject != "" {
		msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	}
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	if body != "" {
		msgBuilder.WriteString(body)
	}

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))

	draft := &gmail.Draft{
		Message: &gmail.Message{Raw: raw},
	}

	updated, err := svc.Users.Drafts.Update("me", draftID, draft).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update draft: %w", err))
	}

	result := map[string]interface{}{
		"status":   "updated",
		"draft_id": updated.Id,
	}
	if updated.Message != nil {
		result["message_id"] = updated.Message.Id
	}

	return p.Print(result)
}

func runGmailSendDraft(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	draftID, _ := cmd.Flags().GetString("id")

	draft := &gmail.Draft{
		Id: draftID,
	}

	sent, err := svc.Users.Drafts.Send("me", draft).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to send draft: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":     "sent",
		"message_id": sent.Id,
		"thread_id":  sent.ThreadId,
	})
}

func runGmailDeleteDraft(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	draftID, _ := cmd.Flags().GetString("id")

	err = svc.Users.Drafts.Delete("me", draftID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete draft: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":   "deleted",
		"draft_id": draftID,
	})
}

func runGmailAttachment(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Gmail()
	if err != nil {
		return p.PrintError(err)
	}

	messageID, _ := cmd.Flags().GetString("message-id")
	attachmentID, _ := cmd.Flags().GetString("id")
	output, _ := cmd.Flags().GetString("output")

	attachment, err := svc.Users.Messages.Attachments.Get("me", messageID, attachmentID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get attachment: %w", err))
	}

	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to decode attachment: %w", err))
	}

	err = os.WriteFile(output, data, 0644)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to write file: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "downloaded",
		"file":   output,
		"size":   len(data),
	})
}
