package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/chat/v1"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Manage Google Chat",
	Long: `Commands for interacting with Google Chat spaces and messages.

Note: Google Chat API requires additional setup. You may need to:
1. Enable the Chat API in your Google Cloud project
2. Configure the OAuth consent screen for Chat scopes
3. For some operations, you may need a service account with domain-wide delegation`,
}

var chatListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Chat spaces",
	Long:  "Lists all Chat spaces (rooms, DMs, group chats) you have access to.",
	RunE:  runChatList,
}

var chatMessagesCmd = &cobra.Command{
	Use:   "messages <space-id>",
	Short: "List messages in a space",
	Long:  "Lists recent messages in a Chat space.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatMessages,
}

var chatSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message to a space",
	Long:  "Sends a text message to a Chat space.",
	RunE:  runChatSend,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.AddCommand(chatListCmd)
	chatCmd.AddCommand(chatMessagesCmd)
	chatCmd.AddCommand(chatSendCmd)

	// Messages flags
	chatMessagesCmd.Flags().Int64("max", 25, "Maximum number of messages to return")

	// Send flags
	chatSendCmd.Flags().String("space", "", "Space ID or name (required)")
	chatSendCmd.Flags().String("text", "", "Message text (required)")
	chatSendCmd.MarkFlagRequired("space")
	chatSendCmd.MarkFlagRequired("text")
}

func runChatList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Chat()
	if err != nil {
		return p.PrintError(err)
	}

	resp, err := svc.Spaces.List().Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list spaces: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Spaces))
	for _, space := range resp.Spaces {
		spaceInfo := map[string]interface{}{
			"name":         space.Name,
			"display_name": space.DisplayName,
			"type":         space.Type,
		}
		if space.SpaceDetails != nil {
			spaceInfo["description"] = space.SpaceDetails.Description
		}
		results = append(results, spaceInfo)
	}

	return p.Print(map[string]interface{}{
		"spaces": results,
		"count":  len(results),
	})
}

func runChatMessages(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Chat()
	if err != nil {
		return p.PrintError(err)
	}

	spaceID := args[0]
	maxResults, _ := cmd.Flags().GetInt64("max")

	// Ensure space name has correct format
	spaceName := spaceID
	if spaceName[:7] != "spaces/" {
		spaceName = "spaces/" + spaceName
	}

	resp, err := svc.Spaces.Messages.List(spaceName).PageSize(int64(maxResults)).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list messages: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		msgInfo := map[string]interface{}{
			"name":        msg.Name,
			"text":        msg.Text,
			"create_time": msg.CreateTime,
		}
		if msg.Sender != nil {
			msgInfo["sender"] = msg.Sender.DisplayName
		}
		results = append(results, msgInfo)
	}

	return p.Print(map[string]interface{}{
		"messages": results,
		"count":    len(results),
	})
}

func runChatSend(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Chat()
	if err != nil {
		return p.PrintError(err)
	}

	spaceID, _ := cmd.Flags().GetString("space")
	text, _ := cmd.Flags().GetString("text")

	// Ensure space name has correct format
	spaceName := spaceID
	if len(spaceName) < 7 || spaceName[:7] != "spaces/" {
		spaceName = "spaces/" + spaceName
	}

	msg := &chat.Message{
		Text: text,
	}

	sent, err := svc.Spaces.Messages.Create(spaceName, msg).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to send message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "sent",
		"name":        sent.Name,
		"create_time": sent.CreateTime,
	})
}
