package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/omriariav/workspace-cli/internal/usercache"
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

var chatMembersCmd = &cobra.Command{
	Use:   "members <space-id>",
	Short: "List members of a space",
	Long:  "Lists all members (users and bots) in a Chat space with display names.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatMembers,
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
	chatCmd.AddCommand(chatMembersCmd)

	// Members flags
	chatMembersCmd.Flags().Int64("max", 100, "Maximum number of members to return")

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
			senderName := msg.Sender.DisplayName
			if senderName == "" {
				senderName = msg.Sender.Name
			}
			msgInfo["sender"] = senderName
			msgInfo["sender_type"] = msg.Sender.Type
		}
		results = append(results, msgInfo)
	}

	return p.Print(map[string]interface{}{
		"messages": results,
		"count":    len(results),
	})
}

func runChatMembers(cmd *cobra.Command, args []string) error {
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

	spaceName := spaceID
	if len(spaceName) < 7 || spaceName[:7] != "spaces/" {
		spaceName = "spaces/" + spaceName
	}

	// Page size per request (Google caps at 100)
	pageSize := maxResults
	if pageSize > 100 {
		pageSize = 100
	}

	var results []map[string]interface{}
	errDone := errors.New("done")
	err = svc.Spaces.Members.List(spaceName).PageSize(pageSize).Pages(ctx, func(resp *chat.ListMembershipsResponse) error {
		for _, m := range resp.Memberships {
			if m == nil {
				continue
			}
			if int64(len(results)) >= maxResults {
				return errDone
			}
			results = append(results, mapMemberToOutput(m))
		}
		if int64(len(results)) >= maxResults {
			return errDone
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDone) {
		return p.PrintError(fmt.Errorf("failed to list members: %w", err))
	}

	// Resolve display names via People API + local cache
	cache, cacheErr := usercache.New()
	if cacheErr == nil {
		// Collect human user IDs missing display_name (skip bots)
		var toResolve []string
		for _, r := range results {
			if _, hasName := r["display_name"]; !hasName {
				if memberType, _ := r["type"].(string); memberType == "BOT" {
					continue
				}
				if uid, ok := r["user"].(string); ok {
					toResolve = append(toResolve, uid)
				}
			}
		}

		if len(toResolve) > 0 {
			// Try People API resolution
			peopleSvc, pErr := factory.People()
			if pErr == nil {
				cache.ResolveMany(peopleSvc, toResolve)
			}
		}

		// Enrich results from cache
		for _, r := range results {
			if uid, ok := r["user"].(string); ok {
				if info, found := cache.Get(uid); found {
					r["display_name"] = info.DisplayName
					if info.Email != "" {
						r["email"] = info.Email
					}
				}
			}
		}
	}

	return p.Print(map[string]interface{}{
		"members": results,
		"count":   len(results),
		"space":   spaceName,
	})
}

// mapMemberToOutput converts a Chat membership into a map for JSON output.
func mapMemberToOutput(m *chat.Membership) map[string]interface{} {
	entry := map[string]interface{}{
		"name": m.Name,
		"role": m.Role,
	}
	if m.Member != nil {
		if m.Member.DisplayName != "" {
			entry["display_name"] = m.Member.DisplayName
		}
		if m.Member.Name != "" {
			entry["user"] = m.Member.Name
		}
		if m.Member.Type != "" {
			entry["type"] = m.Member.Type
		}
	}
	if m.CreateTime != "" {
		entry["joined"] = m.CreateTime
	}
	return entry
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
