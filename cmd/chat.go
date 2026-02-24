package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/omriariav/workspace-cli/internal/spacecache"
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

// --- Spaces CRUD ---

var chatGetSpaceCmd = &cobra.Command{
	Use:   "get-space <space>",
	Short: "Get a space",
	Long:  "Retrieves details about a Chat space by its resource name.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatGetSpace,
}

var chatCreateSpaceCmd = &cobra.Command{
	Use:   "create-space",
	Short: "Create a space",
	Long:  "Creates a new Chat space.",
	RunE:  runChatCreateSpace,
}

var chatDeleteSpaceCmd = &cobra.Command{
	Use:   "delete-space <space>",
	Short: "Delete a space",
	Long:  "Deletes a Chat space by its resource name.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatDeleteSpace,
}

var chatUpdateSpaceCmd = &cobra.Command{
	Use:   "update-space <space>",
	Short: "Update a space",
	Long:  "Updates a Chat space's display name or description.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatUpdateSpace,
}

var chatSearchSpacesCmd = &cobra.Command{
	Use:   "search-spaces",
	Short: "Search spaces (admin only)",
	Long:  "Searches for Chat spaces using a query. Requires Workspace admin privileges and chat.admin.spaces scope.",
	RunE:  runChatSearchSpaces,
}

var chatFindDmCmd = &cobra.Command{
	Use:   "find-dm",
	Short: "Find a direct message space",
	Long:  "Finds a direct message space with a specific user.",
	RunE:  runChatFindDm,
}

var chatSetupSpaceCmd = &cobra.Command{
	Use:   "setup-space",
	Short: "Set up a space with members",
	Long:  "Creates a space and adds initial members in one call.",
	RunE:  runChatSetupSpace,
}

// --- Member Management ---

var chatGetMemberCmd = &cobra.Command{
	Use:   "get-member <member-name>",
	Short: "Get a member",
	Long:  "Retrieves details about a space member.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatGetMember,
}

var chatAddMemberCmd = &cobra.Command{
	Use:   "add-member <space>",
	Short: "Add a member to a space",
	Long:  "Adds a user as a member of a Chat space.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatAddMember,
}

var chatRemoveMemberCmd = &cobra.Command{
	Use:   "remove-member <member-name>",
	Short: "Remove a member",
	Long:  "Removes a member from a Chat space.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatRemoveMember,
}

var chatUpdateMemberCmd = &cobra.Command{
	Use:   "update-member <member-name>",
	Short: "Update a member's role",
	Long:  "Updates a member's role in a Chat space.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatUpdateMember,
}

// --- Read State ---

var chatReadStateCmd = &cobra.Command{
	Use:   "read-state <space>",
	Short: "Get space read state",
	Long:  "Gets the read state for a space (when you last read it).",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatReadState,
}

var chatMarkReadCmd = &cobra.Command{
	Use:   "mark-read <space>",
	Short: "Mark a space as read",
	Long:  "Updates the read state for a space to mark it as read.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatMarkRead,
}

var chatThreadReadStateCmd = &cobra.Command{
	Use:   "thread-read-state <thread>",
	Short: "Get thread read state",
	Long:  "Gets the read state for a thread.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatThreadReadState,
}

// --- Attachments ---

var chatAttachmentCmd = &cobra.Command{
	Use:   "attachment <attachment-name>",
	Short: "Get attachment metadata",
	Long:  "Retrieves metadata for a message attachment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatAttachment,
}

// --- Media ---

var chatUploadCmd = &cobra.Command{
	Use:   "upload <space>",
	Short: "Upload a file to a space",
	Long:  "Uploads a file as an attachment to a Chat space.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatUpload,
}

var chatDownloadCmd = &cobra.Command{
	Use:   "download <resource-name>",
	Short: "Download a media attachment",
	Long:  "Downloads a media attachment to a local file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatDownload,
}

// --- Space Events ---

var chatEventsCmd = &cobra.Command{
	Use:   "events <space>",
	Short: "List space events",
	Long:  "Lists events in a Chat space (requires filter with event types).",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatEvents,
}

var chatEventCmd = &cobra.Command{
	Use:   "event <event-name>",
	Short: "Get a space event",
	Long:  "Retrieves details about a single space event.",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatEvent,
}

// --- Space member cache ---

var chatBuildCacheCmd = &cobra.Command{
	Use:   "build-cache",
	Short: "Build space-members cache",
	Long:  "Iterates spaces, fetches members, resolves emails, and builds a local cache for fast lookup.",
	RunE:  runChatBuildCache,
}

var chatFindGroupCmd = &cobra.Command{
	Use:   "find-group",
	Short: "Find group chats by members",
	Long:  "Searches the local space-members cache for spaces containing all specified members.",
	RunE:  runChatFindGroup,
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
	chatCmd.AddCommand(chatGetSpaceCmd)
	chatCmd.AddCommand(chatCreateSpaceCmd)
	chatCmd.AddCommand(chatDeleteSpaceCmd)
	chatCmd.AddCommand(chatUpdateSpaceCmd)
	chatCmd.AddCommand(chatSearchSpacesCmd)
	chatCmd.AddCommand(chatFindDmCmd)
	chatCmd.AddCommand(chatSetupSpaceCmd)
	chatCmd.AddCommand(chatGetMemberCmd)
	chatCmd.AddCommand(chatAddMemberCmd)
	chatCmd.AddCommand(chatRemoveMemberCmd)
	chatCmd.AddCommand(chatUpdateMemberCmd)
	chatCmd.AddCommand(chatReadStateCmd)
	chatCmd.AddCommand(chatMarkReadCmd)
	chatCmd.AddCommand(chatThreadReadStateCmd)
	chatCmd.AddCommand(chatAttachmentCmd)
	chatCmd.AddCommand(chatUploadCmd)
	chatCmd.AddCommand(chatDownloadCmd)
	chatCmd.AddCommand(chatEventsCmd)
	chatCmd.AddCommand(chatEventCmd)
	chatCmd.AddCommand(chatBuildCacheCmd)
	chatCmd.AddCommand(chatFindGroupCmd)

	// List flags
	chatListCmd.Flags().String("filter", "", "Filter spaces (e.g. 'spaceType = \"SPACE\"')")
	chatListCmd.Flags().Int64("page-size", 100, "Number of spaces per page")
	chatListCmd.Flags().Int64("max", 0, "Maximum number of spaces to return (0 = all)")

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
	chatMessagesCmd.Flags().String("after", "", "Show messages after this time (RFC3339, e.g. 2026-02-17T00:00:00Z)")
	chatMessagesCmd.Flags().String("before", "", "Show messages before this time (RFC3339, e.g. 2026-02-20T00:00:00Z)")

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
	chatReactionsCmd.Flags().Int64("max", 0, "Maximum number of reactions to return (0 = all)")

	// React flags
	chatReactCmd.Flags().String("emoji", "", "Emoji unicode character (required, e.g. '😀')")
	chatReactCmd.MarkFlagRequired("emoji")

	// Create space flags
	chatCreateSpaceCmd.Flags().String("display-name", "", "Space display name (required)")
	chatCreateSpaceCmd.Flags().String("type", "SPACE", "Space type: SPACE or GROUP_CHAT")
	chatCreateSpaceCmd.Flags().String("description", "", "Space description")
	chatCreateSpaceCmd.MarkFlagRequired("display-name")

	// Update space flags
	chatUpdateSpaceCmd.Flags().String("display-name", "", "New display name")
	chatUpdateSpaceCmd.Flags().String("description", "", "New description")

	// Search spaces flags
	chatSearchSpacesCmd.Flags().String("query", "", "Search query (required)")
	chatSearchSpacesCmd.Flags().Int64("page-size", 100, "Number of results per page")
	chatSearchSpacesCmd.Flags().Int64("max", 0, "Maximum number of spaces to return (0 = all)")
	chatSearchSpacesCmd.MarkFlagRequired("query")

	// Find DM flags
	chatFindDmCmd.Flags().String("user", "", "User resource name (required, e.g. users/123)")
	chatFindDmCmd.MarkFlagRequired("user")

	// Setup space flags
	chatSetupSpaceCmd.Flags().String("display-name", "", "Space display name (required for SPACE type)")
	chatSetupSpaceCmd.Flags().String("type", "SPACE", "Space type: SPACE, GROUP_CHAT, or DIRECT_MESSAGE")
	chatSetupSpaceCmd.Flags().String("members", "", "Comma-separated user resource names")

	// Add member flags
	chatAddMemberCmd.Flags().String("user", "", "User resource name (required, e.g. users/123)")
	chatAddMemberCmd.Flags().String("role", "ROLE_MEMBER", "Member role: ROLE_MEMBER or ROLE_MANAGER")
	chatAddMemberCmd.MarkFlagRequired("user")

	// Update member flags
	chatUpdateMemberCmd.Flags().String("role", "", "New role: ROLE_MEMBER or ROLE_MANAGER (required)")
	chatUpdateMemberCmd.MarkFlagRequired("role")

	// Mark read flags
	chatMarkReadCmd.Flags().String("time", "", "Read time (RFC-3339, defaults to now)")

	// Upload flags
	chatUploadCmd.Flags().String("file", "", "Path to file to upload (required)")
	chatUploadCmd.MarkFlagRequired("file")

	// Download flags
	chatDownloadCmd.Flags().String("output", "", "Output file path (required)")
	chatDownloadCmd.MarkFlagRequired("output")

	// Events flags
	chatEventsCmd.Flags().String("filter", "", "Event type filter (required)")
	chatEventsCmd.Flags().Int64("page-size", 100, "Number of events per page")
	chatEventsCmd.Flags().Int64("max", 0, "Maximum number of events to return (0 = all)")
	chatEventsCmd.MarkFlagRequired("filter")

	// Build cache flags
	chatBuildCacheCmd.Flags().String("type", "GROUP_CHAT", "Space type to cache: GROUP_CHAT, SPACE, DIRECT_MESSAGE, or all")

	// Find group flags
	chatFindGroupCmd.Flags().String("members", "", "Comma-separated email addresses to search for (required)")
	chatFindGroupCmd.Flags().Bool("refresh", false, "Rebuild cache before searching")
	chatFindGroupCmd.MarkFlagRequired("members")
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
	maxResults, _ := cmd.Flags().GetInt64("max")

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
			results = append(results, mapSpaceToOutput(space))
			if maxResults > 0 && int64(len(results)) >= maxResults {
				break
			}
		}

		if resp.NextPageToken == "" || (maxResults > 0 && int64(len(results)) >= maxResults) {
			break
		}
		pageToken = resp.NextPageToken
	}

	if maxResults > 0 && int64(len(results)) > maxResults {
		results = results[:maxResults]
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
	after, _ := cmd.Flags().GetString("after")
	before, _ := cmd.Flags().GetString("before")

	// Build filter from --after/--before flags, combining with --filter
	var filterParts []string
	if after != "" {
		filterParts = append(filterParts, fmt.Sprintf(`createTime > "%s"`, after))
	}
	if before != "" {
		filterParts = append(filterParts, fmt.Sprintf(`createTime < "%s"`, before))
	}
	if filter != "" {
		filterParts = append(filterParts, filter)
	}
	if len(filterParts) > 0 {
		filter = strings.Join(filterParts, " AND ")
	}

	if maxResults <= 0 {
		return p.Print(map[string]interface{}{
			"messages": []map[string]interface{}{},
			"count":    0,
		})
	}

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
			if msg.Thread != nil {
				msgInfo["thread"] = msg.Thread.Name
			}
			if msg.LastUpdateTime != "" {
				msgInfo["last_update_time"] = msg.LastUpdateTime
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
	if m.DeleteTime != "" {
		entry["delete_time"] = m.DeleteTime
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
	if msg.LastUpdateTime != "" {
		result["last_update_time"] = msg.LastUpdateTime
	}
	if msg.DeleteTime != "" {
		result["delete_time"] = msg.DeleteTime
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
	maxResults, _ := cmd.Flags().GetInt64("max")

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
			if maxResults > 0 && int64(len(results)) >= maxResults {
				break
			}
		}

		if resp.NextPageToken == "" || (maxResults > 0 && int64(len(results)) >= maxResults) {
			break
		}
		pageToken = resp.NextPageToken
	}

	if maxResults > 0 && int64(len(results)) > maxResults {
		results = results[:maxResults]
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

// ensureReadStateName normalizes a space identifier to a read state resource name.
func ensureReadStateName(spaceID string) string {
	if strings.HasPrefix(spaceID, "users/") {
		return spaceID
	}
	s := ensureSpaceName(spaceID)
	return "users/me/" + s + "/spaceReadState"
}

// mapSpaceToOutput converts a Chat space into a map for JSON output.
func mapSpaceToOutput(space *chat.Space) map[string]interface{} {
	result := map[string]interface{}{
		"name":         space.Name,
		"display_name": space.DisplayName,
		"type":         space.SpaceType,
	}
	if space.SpaceDetails != nil && space.SpaceDetails.Description != "" {
		result["description"] = space.SpaceDetails.Description
	}
	if space.CreateTime != "" {
		result["create_time"] = space.CreateTime
	}
	if space.LastActiveTime != "" {
		result["last_active_time"] = space.LastActiveTime
	}
	if space.SpaceThreadingState != "" {
		result["threading_state"] = space.SpaceThreadingState
	}
	return result
}

func runChatGetSpace(cmd *cobra.Command, args []string) error {
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

	space, err := svc.Spaces.Get(spaceName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get space: %w", err))
	}

	return p.Print(mapSpaceToOutput(space))
}

func runChatCreateSpace(cmd *cobra.Command, args []string) error {
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

	displayName, _ := cmd.Flags().GetString("display-name")
	spaceType, _ := cmd.Flags().GetString("type")
	description, _ := cmd.Flags().GetString("description")

	space := &chat.Space{
		DisplayName: displayName,
		SpaceType:   spaceType,
	}
	if description != "" {
		space.SpaceDetails = &chat.SpaceDetails{Description: description}
	}

	created, err := svc.Spaces.Create(space).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create space: %w", err))
	}

	result := mapSpaceToOutput(created)
	result["status"] = "created"
	return p.Print(result)
}

func runChatDeleteSpace(cmd *cobra.Command, args []string) error {
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

	_, err = svc.Spaces.Delete(spaceName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete space: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"name":   spaceName,
	})
}

func runChatUpdateSpace(cmd *cobra.Command, args []string) error {
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
	displayName, _ := cmd.Flags().GetString("display-name")
	description, _ := cmd.Flags().GetString("description")

	space := &chat.Space{}
	var masks []string

	if displayName != "" {
		space.DisplayName = displayName
		masks = append(masks, "display_name")
	}
	if description != "" {
		space.SpaceDetails = &chat.SpaceDetails{Description: description}
		masks = append(masks, "space_details")
	}

	if len(masks) == 0 {
		return p.PrintError(fmt.Errorf("at least one of --display-name or --description is required"))
	}

	updated, err := svc.Spaces.Patch(spaceName, space).UpdateMask(strings.Join(masks, ",")).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update space: %w", err))
	}

	result := mapSpaceToOutput(updated)
	result["status"] = "updated"
	return p.Print(result)
}

func runChatSearchSpaces(cmd *cobra.Command, args []string) error {
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

	query, _ := cmd.Flags().GetString("query")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	maxResults, _ := cmd.Flags().GetInt64("max")

	var results []map[string]interface{}
	var pageToken string

	for {
		call := svc.Spaces.Search().Query(query).PageSize(pageSize).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to search spaces: %w", err))
		}

		for _, space := range resp.Spaces {
			results = append(results, mapSpaceToOutput(space))
			if maxResults > 0 && int64(len(results)) >= maxResults {
				break
			}
		}

		if resp.NextPageToken == "" || (maxResults > 0 && int64(len(results)) >= maxResults) {
			break
		}
		pageToken = resp.NextPageToken
	}

	if maxResults > 0 && int64(len(results)) > maxResults {
		results = results[:maxResults]
	}

	return p.Print(map[string]interface{}{
		"spaces": results,
		"count":  len(results),
	})
}

func runChatFindDm(cmd *cobra.Command, args []string) error {
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

	user, _ := cmd.Flags().GetString("user")

	space, err := svc.Spaces.FindDirectMessage().Name(user).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to find DM: %w", err))
	}

	return p.Print(mapSpaceToOutput(space))
}

func runChatSetupSpace(cmd *cobra.Command, args []string) error {
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

	displayName, _ := cmd.Flags().GetString("display-name")
	spaceType, _ := cmd.Flags().GetString("type")
	membersStr, _ := cmd.Flags().GetString("members")

	// Validate type
	switch spaceType {
	case "SPACE", "DIRECT_MESSAGE", "GROUP_CHAT":
	default:
		return p.PrintError(fmt.Errorf("invalid --type %q: must be SPACE, GROUP_CHAT, or DIRECT_MESSAGE", spaceType))
	}

	// Validate flags based on space type
	switch spaceType {
	case "DIRECT_MESSAGE", "GROUP_CHAT":
		if membersStr == "" {
			return p.PrintError(fmt.Errorf("--members is required for %s type", spaceType))
		}
	default: // SPACE
		if displayName == "" {
			return p.PrintError(fmt.Errorf("--display-name is required for %s type", spaceType))
		}
	}

	req := &chat.SetUpSpaceRequest{}

	// API rejects displayName for DM and GROUP_CHAT types
	switch spaceType {
	case "DIRECT_MESSAGE", "GROUP_CHAT":
		req.Space = &chat.Space{SpaceType: spaceType}
	default:
		req.Space = &chat.Space{DisplayName: displayName, SpaceType: spaceType}
	}

	if membersStr != "" {
		for _, m := range strings.Split(membersStr, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				req.Memberships = append(req.Memberships, &chat.Membership{
					Member: &chat.User{
						Name: m,
						Type: "HUMAN",
					},
				})
			}
		}
	}

	space, err := svc.Spaces.Setup(req).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to setup space: %w", err))
	}

	result := mapSpaceToOutput(space)
	result["status"] = "created"
	return p.Print(result)
}

func runChatGetMember(cmd *cobra.Command, args []string) error {
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

	memberName := args[0]

	member, err := svc.Spaces.Members.Get(memberName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get member: %w", err))
	}

	return p.Print(mapMemberToOutput(member))
}

func runChatAddMember(cmd *cobra.Command, args []string) error {
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
	user, _ := cmd.Flags().GetString("user")
	role, _ := cmd.Flags().GetString("role")

	membership := &chat.Membership{
		Member: &chat.User{
			Name: user,
			Type: "HUMAN",
		},
		Role: role,
	}

	created, err := svc.Spaces.Members.Create(spaceName, membership).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add member: %w", err))
	}

	result := mapMemberToOutput(created)
	result["status"] = "added"
	return p.Print(result)
}

func runChatRemoveMember(cmd *cobra.Command, args []string) error {
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

	memberName := args[0]

	_, err = svc.Spaces.Members.Delete(memberName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to remove member: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "removed",
		"name":   memberName,
	})
}

func runChatUpdateMember(cmd *cobra.Command, args []string) error {
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

	memberName := args[0]
	role, _ := cmd.Flags().GetString("role")

	membership := &chat.Membership{
		Role: role,
	}

	updated, err := svc.Spaces.Members.Patch(memberName, membership).UpdateMask("role").Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update member: %w", err))
	}

	result := mapMemberToOutput(updated)
	result["status"] = "updated"
	return p.Print(result)
}

func runChatReadState(cmd *cobra.Command, args []string) error {
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

	name := ensureReadStateName(args[0])

	state, err := svc.Users.Spaces.GetSpaceReadState(name).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get read state: %w", err))
	}

	return p.Print(map[string]interface{}{
		"name":           state.Name,
		"last_read_time": state.LastReadTime,
	})
}

func runChatMarkRead(cmd *cobra.Command, args []string) error {
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

	name := ensureReadStateName(args[0])
	readTime, _ := cmd.Flags().GetString("time")
	if readTime == "" {
		readTime = time.Now().UTC().Format(time.RFC3339)
	}

	state := &chat.SpaceReadState{
		LastReadTime: readTime,
	}

	updated, err := svc.Users.Spaces.UpdateSpaceReadState(name, state).UpdateMask("last_read_time").Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to mark read: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":         "marked_read",
		"name":           updated.Name,
		"last_read_time": updated.LastReadTime,
	})
}

func runChatThreadReadState(cmd *cobra.Command, args []string) error {
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

	threadName := args[0]

	state, err := svc.Users.Spaces.Threads.GetThreadReadState(threadName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get thread read state: %w", err))
	}

	return p.Print(map[string]interface{}{
		"name":           state.Name,
		"last_read_time": state.LastReadTime,
	})
}

func runChatAttachment(cmd *cobra.Command, args []string) error {
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

	attachmentName := args[0]

	att, err := svc.Spaces.Messages.Attachments.Get(attachmentName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get attachment: %w", err))
	}

	result := map[string]interface{}{
		"name":         att.Name,
		"content_name": att.ContentName,
		"content_type": att.ContentType,
		"source":       att.Source,
	}
	if att.DownloadUri != "" {
		result["download_uri"] = att.DownloadUri
	}
	if att.ThumbnailUri != "" {
		result["thumbnail_uri"] = att.ThumbnailUri
	}

	return p.Print(result)
}

func runChatUpload(cmd *cobra.Command, args []string) error {
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
	filePath, _ := cmd.Flags().GetString("file")

	file, err := os.Open(filePath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to open file: %w", err))
	}
	defer file.Close()

	filename := filepath.Base(filePath)

	req := &chat.UploadAttachmentRequest{
		Filename: filename,
	}

	resp, err := svc.Media.Upload(spaceName, req).Media(file).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to upload file: %w", err))
	}

	result := map[string]interface{}{
		"status":   "uploaded",
		"filename": filename,
	}
	if resp.AttachmentDataRef != nil {
		result["attachment_data_ref"] = resp.AttachmentDataRef.ResourceName
	}

	return p.Print(result)
}

func runChatDownload(cmd *cobra.Command, args []string) error {
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

	resourceName := args[0]
	outputPath, _ := cmd.Flags().GetString("output")

	resp, err := svc.Media.Download(resourceName).Context(ctx).Download()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to download media: %w", err))
	}
	defer resp.Body.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create output file: %w", err))
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to write file: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "downloaded",
		"output": outputPath,
		"bytes":  written,
	})
}

func runChatEvents(cmd *cobra.Command, args []string) error {
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
	filter, _ := cmd.Flags().GetString("filter")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	maxResults, _ := cmd.Flags().GetInt64("max")

	var results []map[string]interface{}
	var pageToken string

	for {
		call := svc.Spaces.SpaceEvents.List(spaceName).Filter(filter).PageSize(pageSize).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list events: %w", err))
		}

		for _, event := range resp.SpaceEvents {
			results = append(results, mapSpaceEventToOutput(event))
			if maxResults > 0 && int64(len(results)) >= maxResults {
				break
			}
		}

		if resp.NextPageToken == "" || (maxResults > 0 && int64(len(results)) >= maxResults) {
			break
		}
		pageToken = resp.NextPageToken
	}

	if maxResults > 0 && int64(len(results)) > maxResults {
		results = results[:maxResults]
	}

	return p.Print(map[string]interface{}{
		"events": results,
		"count":  len(results),
	})
}

func runChatEvent(cmd *cobra.Command, args []string) error {
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

	eventName := args[0]

	event, err := svc.Spaces.SpaceEvents.Get(eventName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get event: %w", err))
	}

	return p.Print(mapSpaceEventToOutput(event))
}

// mapSpaceEventToOutput converts a Chat space event into a map for JSON output.
func mapSpaceEventToOutput(event *chat.SpaceEvent) map[string]interface{} {
	return map[string]interface{}{
		"name":       event.Name,
		"event_type": event.EventType,
		"event_time": event.EventTime,
	}
}

func runChatBuildCache(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	chatSvc, err := factory.Chat()
	if err != nil {
		return p.PrintError(err)
	}

	peopleSvc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}

	spaceType, _ := cmd.Flags().GetString("type")

	switch spaceType {
	case "GROUP_CHAT", "SPACE", "DIRECT_MESSAGE", "all":
	default:
		return p.PrintError(fmt.Errorf("invalid --type %q: must be GROUP_CHAT, SPACE, DIRECT_MESSAGE, or all", spaceType))
	}

	start := time.Now()
	cache, err := spacecache.Build(ctx, chatSvc, peopleSvc, spaceType, func(current, total int) {
		fmt.Fprintf(os.Stderr, "\rScanning spaces... %d/%d", current, total)
	})
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to build cache: %w", err))
	}
	fmt.Fprintln(os.Stderr) // newline after progress

	cachePath := spacecache.DefaultPath()
	if err := spacecache.Save(cachePath, cache); err != nil {
		return p.PrintError(fmt.Errorf("failed to save cache: %w", err))
	}

	return p.Print(map[string]interface{}{
		"spaces_cached": len(cache.Spaces),
		"cache_path":    cachePath,
		"duration":      time.Since(start).Round(time.Second).String(),
	})
}

func runChatFindGroup(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	membersStr, _ := cmd.Flags().GetString("members")
	refresh, _ := cmd.Flags().GetBool("refresh")

	cachePath := spacecache.DefaultPath()

	if refresh {
		factory, err := client.NewFactory(ctx)
		if err != nil {
			return p.PrintError(err)
		}

		chatSvc, err := factory.Chat()
		if err != nil {
			return p.PrintError(err)
		}

		peopleSvc, err := factory.People()
		if err != nil {
			return p.PrintError(err)
		}

		cache, err := spacecache.Build(ctx, chatSvc, peopleSvc, "GROUP_CHAT", func(current, total int) {
			fmt.Fprintf(os.Stderr, "\rScanning spaces... %d/%d", current, total)
		})
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to build cache: %w", err))
		}
		fmt.Fprintln(os.Stderr)

		if err := spacecache.Save(cachePath, cache); err != nil {
			return p.PrintError(fmt.Errorf("failed to save cache: %w", err))
		}
	}

	cache, err := spacecache.Load(cachePath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to load cache: %w", err))
	}

	if len(cache.Spaces) == 0 {
		return p.PrintError(fmt.Errorf("no cache found — run 'gws chat build-cache' first"))
	}

	var emails []string
	for _, e := range strings.Split(membersStr, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			emails = append(emails, e)
		}
	}
	if len(emails) == 0 {
		return p.PrintError(fmt.Errorf("--members must contain at least one email address"))
	}

	matches := spacecache.FindByMembers(cache, emails)

	var results []map[string]interface{}
	for name, entry := range matches {
		results = append(results, map[string]interface{}{
			"space":        name,
			"type":         entry.Type,
			"display_name": entry.DisplayName,
			"members":      entry.Members,
			"member_count": entry.MemberCount,
		})
	}

	return p.Print(map[string]interface{}{
		"matches": results,
		"count":   len(results),
		"query":   emails,
	})
}
