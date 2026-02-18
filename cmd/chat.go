package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

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

var chatGetCmd = &cobra.Command{
	Use:   "get <message-name>",
	Short: "Get a single message",
	Long:  "Retrieves a single message by its resource name (e.g. spaces/AAAA/messages/msg1).",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatGet,
}

var chatUpdateCmd = &cobra.Command{
	Use:   "update <message-name>",
	Short: "Update a message",
	Long:  "Updates the text of an existing message.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatUpdate,
}

var chatDeleteCmd = &cobra.Command{
	Use:   "delete <message-name>",
	Short: "Delete a message",
	Long:  "Deletes a message by its resource name.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatDelete,
}

var chatReactionsCmd = &cobra.Command{
	Use:   "reactions <message-name>",
	Short: "List reactions on a message",
	Long:  "Lists all reactions on a message.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatReactions,
}

var chatReactCmd = &cobra.Command{
	Use:   "react <message-name>",
	Short: "Add a reaction to a message",
	Long:  "Adds an emoji reaction to a message.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatReact,
}

var chatUnreactCmd = &cobra.Command{
	Use:   "unreact <reaction-name>",
	Short: "Remove a reaction",
	Long:  "Removes a reaction by its resource name (e.g. spaces/AAAA/messages/msg1/reactions/rxn1).",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatUnreact,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.AddCommand(chatListCmd)
	chatCmd.AddCommand(chatMessagesCmd)
	chatCmd.AddCommand(chatSendCmd)
	chatCmd.AddCommand(chatMembersCmd)
	chatCmd.AddCommand(chatGetCmd)
	chatCmd.AddCommand(chatUpdateCmd)
	chatCmd.AddCommand(chatDeleteCmd)
	chatCmd.AddCommand(chatReactionsCmd)
	chatCmd.AddCommand(chatReactCmd)
	chatCmd.AddCommand(chatUnreactCmd)

	// List flags
	chatListCmd.Flags().String("filter", "", "Filter spaces (e.g. 'spaceType = \"SPACE\"')")
	chatListCmd.Flags().Int64("page-size", 100, "Number of spaces per page")

	// Members flags
	chatMembersCmd.Flags().Int64("max", 100, "Maximum number of members to return")
	chatMembersCmd.Flags().String("filter", "", "Filter members (e.g. 'member.type = \"HUMAN\"')")
	chatMembersCmd.Flags().Bool("show-groups", false, "Include Google Group memberships")
	chatMembersCmd.Flags().Bool("show-invited", false, "Include invited memberships")

	// Messages flags
	chatMessagesCmd.Flags().Int64("max", 25, "Maximum number of messages to return")
	chatMessagesCmd.Flags().String("filter", "", "Filter messages (e.g. 'createTime > \"2024-01-01T00:00:00Z\"')")
	chatMessagesCmd.Flags().String("order-by", "", "Order messages (e.g. 'createTime DESC')")
	chatMessagesCmd.Flags().Bool("show-deleted", false, "Include deleted messages in results")

	// Send flags
	chatSendCmd.Flags().String("space", "", "Space ID or name (required)")
	chatSendCmd.Flags().String("text", "", "Message text (required)")
	chatSendCmd.MarkFlagRequired("space")
	chatSendCmd.MarkFlagRequired("text")

	// Update flags
	chatUpdateCmd.Flags().String("text", "", "New message text (required)")
	chatUpdateCmd.MarkFlagRequired("text")

	// Delete flags
	chatDeleteCmd.Flags().Bool("force", false, "Force delete (even if message has replies)")

	// Reactions flags
	chatReactionsCmd.Flags().String("filter", "", "Filter reactions (e.g. 'emoji.unicode = \"😀\"')")
	chatReactionsCmd.Flags().Int64("page-size", 25, "Number of reactions per page")

	// React flags
	chatReactCmd.Flags().String("emoji", "", "Emoji unicode character (required, e.g. '😀')")
	chatReactCmd.MarkFlagRequired("emoji")
}

// ensureSpaceName normalizes a space identifier to its full resource name.
func ensureSpaceName(s string) string {
	if !strings.HasPrefix(s, "spaces/") {
		return "spaces/" + s
	}
	return s
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

	filter, _ := cmd.Flags().GetString("filter")
	pageSize, _ := cmd.Flags().GetInt64("page-size")

	var results []map[string]interface{}
	var pageToken string

	for {
		call := svc.Spaces.List().PageSize(pageSize)
		if filter != "" {
			call = call.Filter(filter)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list spaces: %w", err))
		}

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

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
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

	spaceName := ensureSpaceName(args[0])
	maxResults, _ := cmd.Flags().GetInt64("max")
	filter, _ := cmd.Flags().GetString("filter")
	orderBy, _ := cmd.Flags().GetString("order-by")
	showDeleted, _ := cmd.Flags().GetBool("show-deleted")

	var results []map[string]interface{}
	var pageToken string

	for {
		remaining := maxResults - int64(len(results))
		pageSize := remaining
		if pageSize > 1000 {
			pageSize = 1000
		}

		call := svc.Spaces.Messages.List(spaceName).PageSize(pageSize)
		if filter != "" {
			call = call.Filter(filter)
		}
		if orderBy != "" {
			call = call.OrderBy(orderBy)
		}
		if showDeleted {
			call = call.ShowDeleted(true)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list messages: %w", err))
		}

		for _, msg := range resp.Messages {
			if int64(len(results)) >= maxResults {
				break
			}
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
			if msg.DeleteTime != "" {
				msgInfo["delete_time"] = msg.DeleteTime
			}
			results = append(results, msgInfo)
		}

		if resp.NextPageToken == "" || int64(len(results)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
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

	spaceName := ensureSpaceName(args[0])
	maxResults, _ := cmd.Flags().GetInt64("max")
	filter, _ := cmd.Flags().GetString("filter")
	showGroups, _ := cmd.Flags().GetBool("show-groups")
	showInvited, _ := cmd.Flags().GetBool("show-invited")

	// Page size per request (Google caps at 100)
	pageSize := maxResults
	if pageSize > 100 {
		pageSize = 100
	}

	var results []map[string]interface{}
	errDone := errors.New("done")

	call := svc.Spaces.Members.List(spaceName).PageSize(pageSize)
	if filter != "" {
		call = call.Filter(filter)
	}
	if showGroups {
		call = call.ShowGroups(true)
	}
	if showInvited {
		call = call.ShowInvited(true)
	}

	err = call.Pages(ctx, func(resp *chat.ListMembershipsResponse) error {
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

	spaceName := ensureSpaceName(spaceID)

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

func runChatGet(cmd *cobra.Command, args []string) error {
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

	messageName := args[0]

	msg, err := svc.Spaces.Messages.Get(messageName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get message: %w", err))
	}

	result := map[string]interface{}{
		"name":        msg.Name,
		"text":        msg.Text,
		"create_time": msg.CreateTime,
	}
	if msg.Sender != nil {
		senderName := msg.Sender.DisplayName
		if senderName == "" {
			senderName = msg.Sender.Name
		}
		result["sender"] = senderName
		result["sender_type"] = msg.Sender.Type
	}
	if msg.Thread != nil {
		result["thread"] = msg.Thread.Name
	}

	return p.Print(result)
}

func runChatUpdate(cmd *cobra.Command, args []string) error {
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

	messageName := args[0]
	text, _ := cmd.Flags().GetString("text")

	msg := &chat.Message{
		Text: text,
	}

	updated, err := svc.Spaces.Messages.Patch(messageName, msg).UpdateMask("text").Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"name":        updated.Name,
		"text":        updated.Text,
		"create_time": updated.CreateTime,
	})
}

func runChatDelete(cmd *cobra.Command, args []string) error {
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

	messageName := args[0]
	force, _ := cmd.Flags().GetBool("force")

	call := svc.Spaces.Messages.Delete(messageName).Context(ctx)
	if force {
		call = call.Force(true)
	}

	_, err = call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete message: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"name":   messageName,
	})
}

func runChatReactions(cmd *cobra.Command, args []string) error {
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

	messageName := args[0]
	filter, _ := cmd.Flags().GetString("filter")
	pageSize, _ := cmd.Flags().GetInt64("page-size")

	var results []map[string]interface{}
	var pageToken string

	for {
		call := svc.Spaces.Messages.Reactions.List(messageName).PageSize(pageSize).Context(ctx)
		if filter != "" {
			call = call.Filter(filter)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list reactions: %w", err))
		}

		for _, r := range resp.Reactions {
			entry := map[string]interface{}{
				"name": r.Name,
			}
			if r.Emoji != nil {
				entry["emoji"] = r.Emoji.Unicode
			}
			if r.User != nil {
				userName := r.User.DisplayName
				if userName == "" {
					userName = r.User.Name
				}
				entry["user"] = userName
			}
			results = append(results, entry)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return p.Print(map[string]interface{}{
		"reactions": results,
		"count":     len(results),
	})
}

func runChatReact(cmd *cobra.Command, args []string) error {
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

	messageName := args[0]
	emoji, _ := cmd.Flags().GetString("emoji")

	reaction := &chat.Reaction{
		Emoji: &chat.Emoji{
			Unicode: emoji,
		},
	}

	created, err := svc.Spaces.Messages.Reactions.Create(messageName, reaction).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create reaction: %w", err))
	}

	result := map[string]interface{}{
		"status": "reacted",
		"name":   created.Name,
	}
	if created.Emoji != nil {
		result["emoji"] = created.Emoji.Unicode
	}

	return p.Print(result)
}

func runChatUnreact(cmd *cobra.Command, args []string) error {
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

	reactionName := args[0]

	_, err = svc.Spaces.Messages.Reactions.Delete(reactionName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete reaction: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "unreacted",
		"name":   reactionName,
	})
}
