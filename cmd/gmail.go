package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"regexp"
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
	gmailListCmd.Flags().Int64("max", 10, "Maximum number of results")
	gmailListCmd.Flags().String("query", "", "Gmail search query (e.g., 'is:unread', 'from:someone@example.com')")

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

	// If replying, fetch the original message's Message-ID header
	var inReplyTo string
	if replyToMsgID != "" {
		origMsg, err := svc.Users.Messages.Get("me", replyToMsgID).Format("metadata").MetadataHeaders("Message-ID", "Message-Id").Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get original message for reply: %w", err))
		}
		for _, header := range origMsg.Payload.Headers {
			if header.Name == "Message-ID" || header.Name == "Message-Id" {
				inReplyTo = header.Value
				break
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
		msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", inReplyTo))
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
