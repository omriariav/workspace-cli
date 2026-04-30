package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/spacecache"
	"github.com/omriariav/workspace-cli/internal/usercache"
	"github.com/spf13/cobra"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/people/v1"
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

var chatRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Recap recent messages across active spaces",
	Long: `Lists messages across all spaces active within --since.

Uses spaces.list lastActiveTime as a cheap prefilter, then queries
messages.list per active space with createTime > since (orderBy
createTime DESC). Results are flattened and sorted globally by newest
first.

--since accepts a Go duration ("2h", "12h", "7d") or an RFC3339
timestamp ("2026-04-30T09:00:00Z").

Examples:
  gws chat recent --since 2h
  gws chat recent --since 7d --max 1000
  gws chat recent --since 12h --resolve-senders --exclude-self`,
	RunE: runChatRecent,
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
	Long: `Finds a direct message space with a specific user.

Use --user with a resource name or --email with an email address.

Examples:
  gws chat find-dm --user users/123456789
  gws chat find-dm --email user@example.com`,
	RunE: runChatFindDm,
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

var chatUnreadCmd = &cobra.Command{
	Use:   "unread <space>",
	Short: "List unread messages in a space",
	Long: `Lists messages received after the last read time for a Chat space.

Combines read-state and messages list to show only unread content.

Examples:
  gws chat unread spaces/AAAA
  gws chat unread spaces/AAAA --max 10
  gws chat unread spaces/AAAA --mark-read`,
	Args: cobra.ExactArgs(1),
	RunE: runChatUnread,
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

var chatFindSpaceCmd = &cobra.Command{
	Use:   "find-space",
	Short: "Find spaces by display name",
	Long: `Searches the local space cache for spaces whose display_name contains the
given query (case-insensitive substring match).

The cache must already cover the space type you want to search. The default
'gws chat build-cache' run only caches GROUP_CHAT, so to search SPACE-type
rooms or all types either:
  - prebuild with 'gws chat build-cache --type SPACE' (or '--type all'), or
  - pass --refresh to rebuild the cache from spaces.list before searching.

When --refresh is set together with --type, the cache is rebuilt scoped to that
type only; otherwise --refresh rebuilds for all space types.`,
	RunE: runChatFindSpace,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.AddCommand(chatListCmd)
	chatCmd.AddCommand(chatMessagesCmd)
	chatCmd.AddCommand(chatRecentCmd)
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
	chatCmd.AddCommand(chatUnreadCmd)
	chatCmd.AddCommand(chatAttachmentCmd)
	chatCmd.AddCommand(chatUploadCmd)
	chatCmd.AddCommand(chatDownloadCmd)
	chatCmd.AddCommand(chatEventsCmd)
	chatCmd.AddCommand(chatEventCmd)
	chatCmd.AddCommand(chatBuildCacheCmd)
	chatCmd.AddCommand(chatFindGroupCmd)
	chatCmd.AddCommand(chatFindSpaceCmd)

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
	chatMessagesCmd.Flags().Bool("resolve-senders", false, "Resolve sender display names by listing the space membership (one extra API call per space)")

	// Recent flags
	chatRecentCmd.Flags().String("since", "2h", "Time window: duration (e.g. 2h, 12h, 7d) or RFC3339 timestamp")
	chatRecentCmd.Flags().Int64("max", 500, "Maximum total messages to return (0 = all)")
	chatRecentCmd.Flags().Int64("max-per-space", 100, "Maximum messages per active space (0 = all)")
	chatRecentCmd.Flags().Int64("max-spaces", 0, "Maximum active spaces to query, after sorting by lastActiveTime DESC (0 = all)")
	chatRecentCmd.Flags().Bool("resolve-senders", false, "Resolve sender display names by listing each active space's membership (one extra API call per space)")
	chatRecentCmd.Flags().Bool("exclude-self", false, "Omit messages sent by the authenticated user (requires self detection)")

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
	chatFindDmCmd.Flags().String("user", "", "User resource name (e.g. users/123)")
	chatFindDmCmd.Flags().String("email", "", "User email address (e.g. user@example.com)")

	// Unread flags
	chatUnreadCmd.Flags().Int64("max", 25, "Maximum number of unread messages")
	chatUnreadCmd.Flags().Bool("mark-read", false, "Mark space as read after listing")
	chatUnreadCmd.Flags().Bool("resolve-senders", false, "Resolve sender display names by listing the space membership (one extra API call per space)")

	// Get flags
	chatGetCmd.Flags().Bool("resolve-senders", false, "Resolve sender display name by listing the space membership (one extra API call)")

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

	// Find space flags
	chatFindSpaceCmd.Flags().String("name", "", "Display name substring to search for (case-insensitive, required)")
	chatFindSpaceCmd.Flags().String("type", "", "Filter by space type: SPACE, GROUP_CHAT, or DIRECT_MESSAGE")
	chatFindSpaceCmd.Flags().Bool("refresh", false, "Rebuild cache before searching")
	chatFindSpaceCmd.MarkFlagRequired("name")
}

// ensureSpaceName normalizes a space identifier to its full resource name.
func ensureSpaceName(s string) string {
	if !strings.HasPrefix(s, "spaces/") {
		return "spaces/" + s
	}
	return s
}

// serializeChatAttachments converts a Message.Attachment slice into a JSON-friendly
// shape, exposing the resource name needed by `gws chat attachment` and
// `gws chat download`. Returns nil when there are no attachments so the field
// is omitted from output by callers that check for nil.
func serializeChatAttachments(atts []*chat.Attachment) []map[string]interface{} {
	if len(atts) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(atts))
	for _, a := range atts {
		if a == nil {
			continue
		}
		entry := map[string]interface{}{}
		if a.Name != "" {
			entry["name"] = a.Name
		}
		if a.ContentName != "" {
			entry["content_name"] = a.ContentName
		}
		if a.ContentType != "" {
			entry["content_type"] = a.ContentType
		}
		if a.Source != "" {
			entry["source"] = a.Source
		}
		if a.DownloadUri != "" {
			entry["download_uri"] = a.DownloadUri
		}
		if a.ThumbnailUri != "" {
			entry["thumbnail_uri"] = a.ThumbnailUri
		}
		if a.AttachmentDataRef != nil && a.AttachmentDataRef.ResourceName != "" {
			entry["attachment_data_ref"] = map[string]interface{}{
				"resource_name": a.AttachmentDataRef.ResourceName,
			}
		}
		if a.DriveDataRef != nil && a.DriveDataRef.DriveFileId != "" {
			entry["drive_data_ref"] = map[string]interface{}{
				"drive_file_id": a.DriveDataRef.DriveFileId,
			}
		}
		if len(entry) == 0 {
			continue
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runChatList(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	p := GetPrinter()
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
	resolveSenders, _ := cmd.Flags().GetBool("resolve-senders")

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

	senderCtx := nilSenderContext()
	if resolveSenders {
		peopleSvc, _ := factory.PeopleProfile() // best-effort; nil-tolerant inside resolver
		senderCtx = resolveSendersForSpace(ctx, svc, peopleSvc, spaceName)
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
					if resolved, ok := senderCtx.displayNames[msg.Sender.Name]; ok && resolved != "" {
						senderName = resolved
					} else {
						senderName = msg.Sender.Name
					}
				}
				msgInfo["sender"] = senderName
			}
			senderCtx.annotate(msg, msgInfo)
			if msg.Thread != nil {
				msgInfo["thread"] = msg.Thread.Name
			}
			if msg.LastUpdateTime != "" {
				msgInfo["last_update_time"] = msg.LastUpdateTime
			}
			if msg.DeleteTime != "" {
				msgInfo["delete_time"] = msg.DeleteTime
			}
			if atts := serializeChatAttachments(msg.Attachment); atts != nil {
				msgInfo["attachment"] = atts
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

// parseSinceWindow parses --since as either a Go duration ("2h", "12h",
// "7d") or an RFC3339 timestamp ("2026-04-30T09:00:00Z"). Negative or
// zero durations are rejected. Days are accepted via the "d" suffix
// because time.ParseDuration does not.
func parseSinceWindow(value string, now time.Time) (time.Time, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return time.Time{}, fmt.Errorf("--since is required")
	}

	// "Nd" → N*24h, since time.ParseDuration only knows up to "h".
	if strings.HasSuffix(v, "d") {
		base := strings.TrimSuffix(v, "d")
		if base != "" {
			if dur, err := time.ParseDuration(base + "h"); err == nil {
				if dur <= 0 {
					return time.Time{}, fmt.Errorf("--since must be a positive duration, got %q", value)
				}
				return now.Add(-dur * 24), nil
			}
		}
	}

	if dur, err := time.ParseDuration(v); err == nil {
		if dur <= 0 {
			return time.Time{}, fmt.Errorf("--since must be a positive duration, got %q", value)
		}
		return now.Add(-dur), nil
	}

	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("--since must be a duration (e.g. 2h, 12h, 7d) or RFC3339 timestamp, got %q", value)
}

// detectSelfResource returns the canonical "users/{id}" for the
// authenticated user, or "" when detection fails. Best-effort.
func detectSelfResource(ctx context.Context, peopleSvc *people.Service) string {
	if peopleSvc == nil {
		return ""
	}
	me, err := peopleSvc.People.Get("people/me").PersonFields("metadata").Context(ctx).Do()
	if err != nil || me == nil {
		return ""
	}
	if !strings.HasPrefix(me.ResourceName, "people/") {
		return ""
	}
	return "users/" + strings.TrimPrefix(me.ResourceName, "people/")
}

func runChatRecent(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Chat()
	if err != nil {
		return p.PrintError(err)
	}

	since, _ := cmd.Flags().GetString("since")
	maxResults, _ := cmd.Flags().GetInt64("max")
	maxPerSpace, _ := cmd.Flags().GetInt64("max-per-space")
	maxSpaces, _ := cmd.Flags().GetInt64("max-spaces")
	resolveSenders, _ := cmd.Flags().GetBool("resolve-senders")
	excludeSelf, _ := cmd.Flags().GetBool("exclude-self")

	sinceTime, err := parseSinceWindow(since, time.Now())
	if err != nil {
		return p.PrintError(err)
	}
	sinceRFC := sinceTime.UTC().Format(time.RFC3339)

	// Self detection — needed for --exclude-self regardless of --resolve-senders.
	var selfResource string
	if excludeSelf || resolveSenders {
		peopleSvc, _ := factory.PeopleProfile()
		selfResource = detectSelfResource(ctx, peopleSvc)
	}

	// Step 1: list all spaces and keep the ones active within the window.
	type activeSpace struct {
		space      *chat.Space
		activeTime time.Time
	}
	var (
		active        []activeSpace
		spacesScanned int
		pageToken     string
	)
	for {
		call := svc.Spaces.List().PageSize(1000).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list spaces: %w", err))
		}
		for _, s := range resp.Spaces {
			if s == nil {
				continue
			}
			spacesScanned++
			if s.LastActiveTime == "" {
				continue
			}
			lat, err := time.Parse(time.RFC3339, s.LastActiveTime)
			if err != nil {
				continue
			}
			if lat.Before(sinceTime) {
				continue
			}
			active = append(active, activeSpace{space: s, activeTime: lat})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Sort active spaces by lastActiveTime DESC and apply --max-spaces.
	sort.Slice(active, func(i, j int) bool {
		return active[i].activeTime.After(active[j].activeTime)
	})
	if maxSpaces > 0 && int64(len(active)) > maxSpaces {
		active = active[:maxSpaces]
	}

	// Step 2: per active space, fetch messages with createTime > since.
	var messages []map[string]interface{}
	for _, as := range active {
		spaceName := as.space.Name

		senderCtx := nilSenderContext()
		if resolveSenders {
			peopleSvc, _ := factory.PeopleProfile()
			senderCtx = resolveSendersForSpace(ctx, svc, peopleSvc, spaceName)
		}

		filter := fmt.Sprintf(`createTime > "%s"`, sinceRFC)
		var spaceCount int64
		var msgPageToken string
		for {
			pageSize := int64(1000)
			if maxPerSpace > 0 {
				remaining := maxPerSpace - spaceCount
				if remaining <= 0 {
					break
				}
				if remaining < pageSize {
					pageSize = remaining
				}
			}

			call := svc.Spaces.Messages.List(spaceName).
				PageSize(pageSize).
				Filter(filter).
				OrderBy("createTime DESC").
				Context(ctx)
			if msgPageToken != "" {
				call = call.PageToken(msgPageToken)
			}
			resp, err := call.Do()
			if err != nil {
				return p.PrintError(fmt.Errorf("failed to list messages in %s: %w", spaceName, err))
			}

			for _, msg := range resp.Messages {
				if maxPerSpace > 0 && spaceCount >= maxPerSpace {
					break
				}
				if excludeSelf && selfResource != "" && msg.Sender != nil && msg.Sender.Name == selfResource {
					continue
				}
				row := map[string]interface{}{
					"space":                  as.space.Name,
					"space_display_name":     as.space.DisplayName,
					"space_type":             as.space.SpaceType,
					"space_last_active_time": as.space.LastActiveTime,
					"name":                   msg.Name,
					"text":                   msg.Text,
					"create_time":            msg.CreateTime,
				}
				if msg.Sender != nil {
					senderName := msg.Sender.DisplayName
					if senderName == "" {
						if resolved, ok := senderCtx.displayNames[msg.Sender.Name]; ok && resolved != "" {
							senderName = resolved
						} else {
							senderName = msg.Sender.Name
						}
					}
					row["sender"] = senderName
				}
				senderCtx.annotate(msg, row)
				if msg.Thread != nil {
					row["thread"] = msg.Thread.Name
				}
				if atts := serializeChatAttachments(msg.Attachment); atts != nil {
					row["attachment"] = atts
				}
				messages = append(messages, row)
				spaceCount++
			}

			if resp.NextPageToken == "" || (maxPerSpace > 0 && spaceCount >= maxPerSpace) {
				break
			}
			msgPageToken = resp.NextPageToken
		}
	}

	// Step 3: global sort by create_time DESC and apply --max.
	sort.SliceStable(messages, func(i, j int) bool {
		ai, _ := messages[i]["create_time"].(string)
		aj, _ := messages[j]["create_time"].(string)
		return ai > aj // RFC3339 strings sort lexicographically by time.
	})
	if maxResults > 0 && int64(len(messages)) > maxResults {
		messages = messages[:maxResults]
	}

	return p.Print(map[string]interface{}{
		"since":          sinceRFC,
		"spaces_scanned": spacesScanned,
		"active_spaces":  len(active),
		"count":          len(messages),
		"messages":       messages,
	})
}

func runChatMembers(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	resolveSenders, _ := cmd.Flags().GetBool("resolve-senders")

	msg, err := svc.Spaces.Messages.Get(messageName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get message: %w", err))
	}

	result := map[string]interface{}{
		"name":        msg.Name,
		"text":        msg.Text,
		"create_time": msg.CreateTime,
	}
	senderCtx := nilSenderContext()
	if resolveSenders {
		peopleSvc, _ := factory.PeopleProfile()
		senderCtx = resolveSendersForSpace(ctx, svc, peopleSvc, spaceFromMessageName(msg.Name))
	}
	if msg.Sender != nil {
		senderName := msg.Sender.DisplayName
		if senderName == "" {
			if resolved, ok := senderCtx.displayNames[msg.Sender.Name]; ok && resolved != "" {
				senderName = resolved
			} else {
				senderName = msg.Sender.Name
			}
		}
		result["sender"] = senderName
	}
	senderCtx.annotate(msg, result)
	if msg.Thread != nil {
		result["thread"] = msg.Thread.Name
	}
	if msg.LastUpdateTime != "" {
		result["last_update_time"] = msg.LastUpdateTime
	}
	if msg.DeleteTime != "" {
		result["delete_time"] = msg.DeleteTime
	}
	if atts := serializeChatAttachments(msg.Attachment); atts != nil {
		result["attachment"] = atts
	}

	return p.Print(result)
}

func runChatUpdate(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	email, _ := cmd.Flags().GetString("email")

	if user == "" && email == "" {
		return p.PrintError(fmt.Errorf("provide either --user or --email"))
	}
	if user != "" && email != "" {
		return p.PrintError(fmt.Errorf("--user and --email are mutually exclusive"))
	}
	if email != "" {
		user = "users/" + email
	}

	space, err := svc.Spaces.FindDirectMessage().Name(user).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to find DM: %w", err))
	}

	return p.Print(mapSpaceToOutput(space))
}

func runChatSetupSpace(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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

func runChatUnread(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	markRead, _ := cmd.Flags().GetBool("mark-read")
	resolveSenders, _ := cmd.Flags().GetBool("resolve-senders")

	// Get the space read state to find last read time
	readStateName := ensureReadStateName(args[0])
	state, err := svc.Users.Spaces.GetSpaceReadState(readStateName).Context(ctx).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get read state: %w", err))
	}

	lastReadTime := state.LastReadTime

	// Build filter for unread messages
	var filter string
	if lastReadTime != "" {
		filter = fmt.Sprintf("createTime > \"%s\"", lastReadTime)
	}

	senderCtx := nilSenderContext()
	if resolveSenders {
		peopleSvc, _ := factory.PeopleProfile()
		senderCtx = resolveSendersForSpace(ctx, svc, peopleSvc, spaceName)
	}

	// List messages after last read time
	var messages []map[string]interface{}
	pageToken := ""
	for {
		call := svc.Spaces.Messages.List(spaceName).
			PageSize(maxResults).
			OrderBy("createTime ASC").
			Context(ctx)
		if filter != "" {
			call = call.Filter(filter)
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list messages: %w", err))
		}

		for _, msg := range resp.Messages {
			if int64(len(messages)) >= maxResults {
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
					if resolved, ok := senderCtx.displayNames[msg.Sender.Name]; ok && resolved != "" {
						senderName = resolved
					} else {
						senderName = msg.Sender.Name
					}
				}
				msgInfo["sender"] = senderName
			}
			senderCtx.annotate(msg, msgInfo)
			if msg.Thread != nil {
				msgInfo["thread"] = msg.Thread.Name
			}
			if atts := serializeChatAttachments(msg.Attachment); atts != nil {
				msgInfo["attachment"] = atts
			}
			messages = append(messages, msgInfo)
		}

		if resp.NextPageToken == "" || int64(len(messages)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Optionally mark as read
	markedRead := false
	if markRead && len(messages) > 0 {
		now := time.Now().UTC().Format(time.RFC3339)
		_, err := svc.Users.Spaces.UpdateSpaceReadState(readStateName, &chat.SpaceReadState{
			LastReadTime: now,
		}).UpdateMask("last_read_time").Context(ctx).Do()
		if err == nil {
			markedRead = true
		}
	}

	result := map[string]interface{}{
		"space":          spaceName,
		"last_read_time": lastReadTime,
		"count":          len(messages),
		"messages":       messages,
	}
	if markRead {
		result["marked_read"] = markedRead
	}

	return p.Print(result)
}

func runChatAttachment(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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
	p := GetPrinter()
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

// chatServiceForTest and peopleServiceForTest, when non-nil, replace the
// factory-built services in runChatFindSpace's --refresh path. Tests set these
// to point at httptest endpoints; production paths leave both nil so the
// factory is used.
var (
	chatServiceForTest   *chat.Service
	peopleServiceForTest *people.Service
)

func runChatFindSpace(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()

	rawName, _ := cmd.Flags().GetString("name")
	spaceType, _ := cmd.Flags().GetString("type")
	refresh, _ := cmd.Flags().GetBool("refresh")

	name := strings.TrimSpace(rawName)
	if name == "" {
		return p.PrintError(fmt.Errorf("--name must not be empty"))
	}

	if spaceType != "" {
		switch strings.ToUpper(spaceType) {
		case "SPACE", "GROUP_CHAT", "DIRECT_MESSAGE":
		default:
			return p.PrintError(fmt.Errorf("invalid --type %q: must be SPACE, GROUP_CHAT, or DIRECT_MESSAGE", spaceType))
		}
	}

	cachePath := spacecache.DefaultPath()

	if refresh {
		var chatSvc *chat.Service
		var peopleSvc *people.Service
		if chatServiceForTest != nil {
			chatSvc = chatServiceForTest
			peopleSvc = peopleServiceForTest
		} else {
			factory, err := client.NewFactory(ctx)
			if err != nil {
				return p.PrintError(err)
			}
			chatSvc, err = factory.Chat()
			if err != nil {
				return p.PrintError(err)
			}
			peopleSvc, err = factory.People()
			if err != nil {
				return p.PrintError(err)
			}
		}

		buildType := "all"
		if spaceType != "" {
			buildType = strings.ToUpper(spaceType)
		}
		cache, err := spacecache.Build(ctx, chatSvc, peopleSvc, buildType, func(current, total int) {
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
		return p.PrintError(fmt.Errorf("no cache found — run 'gws chat build-cache' first or pass --refresh"))
	}

	matches := spacecache.FindByDisplayName(cache, name, spaceType)

	type matchRow struct {
		space, displayName string
		entry              spacecache.SpaceEntry
	}
	rows := make([]matchRow, 0, len(matches))
	for spaceName, entry := range matches {
		rows = append(rows, matchRow{space: spaceName, displayName: entry.DisplayName, entry: entry})
	}
	// Stable order across runs: by display_name, then by space resource name.
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].displayName != rows[j].displayName {
			return rows[i].displayName < rows[j].displayName
		}
		return rows[i].space < rows[j].space
	})

	results := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		row := map[string]interface{}{
			"space":        r.space,
			"type":         r.entry.Type,
			"display_name": r.entry.DisplayName,
			"member_count": r.entry.MemberCount,
		}
		if r.entry.MembersUnresolved {
			row["members_unresolved"] = true
		}
		results = append(results, row)
	}

	out := map[string]interface{}{
		"matches": results,
		"count":   len(results),
		"query":   name,
	}
	if spaceType != "" {
		out["type"] = strings.ToUpper(spaceType)
	}
	return p.Print(out)
}

// senderContext resolves sender display names and self markers for a single
// space within one command invocation. Resolution is best-effort: failures
// degrade to "no resolution" rather than failing the whole command. When
// constructed via nilSenderContext (the default-path object), it makes no
// API calls and only annotates fields available directly on the message.
type senderContext struct {
	space        string
	selfResource string            // canonical "users/{id}" for the authenticated user, or "".
	displayNames map[string]string // users/{id} -> display name (only populated when --resolve-senders).
}

// nilSenderContext returns a no-op resolver suitable for the default path:
// no API calls, no self detection, no display-name resolution. annotate still
// adds sender_type and sender_resource purely from the message payload.
func nilSenderContext() *senderContext {
	return &senderContext{}
}

// resolveSendersForSpace builds a fully-populated resolver for one space.
// Self detection uses the People API people/me (cached implicitly per
// invocation by the caller), then maps people/{id} to the canonical
// users/{id} that Chat returns for sender resources. Display names come from
// listing space membership. Both calls are best-effort: a failure in either
// one leaves the corresponding fields empty, never aborts the command.
func resolveSendersForSpace(ctx context.Context, chatSvc *chat.Service, peopleSvc *people.Service, space string) *senderContext {
	sc := &senderContext{space: space, displayNames: map[string]string{}}

	if peopleSvc != nil {
		if me, err := peopleSvc.People.Get("people/me").PersonFields("metadata").Context(ctx).Do(); err == nil {
			if me != nil && strings.HasPrefix(me.ResourceName, "people/") {
				sc.selfResource = "users/" + strings.TrimPrefix(me.ResourceName, "people/")
			}
		}
	}

	if chatSvc == nil || space == "" {
		return sc
	}

	pageToken := ""
	for {
		call := chatSvc.Spaces.Members.List(space).PageSize(1000).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return sc // Best-effort: leave unresolved senders as-is.
		}
		for _, m := range resp.Memberships {
			if m == nil || m.Member == nil || m.Member.Name == "" {
				continue
			}
			if m.Member.DisplayName != "" {
				sc.displayNames[m.Member.Name] = m.Member.DisplayName
			}
		}
		if resp.NextPageToken == "" {
			return sc
		}
		pageToken = resp.NextPageToken
	}
}

// annotate adds additive sender attribution fields to the given output map
// without disturbing the existing "sender" field set by callers. Safe to call
// when ctx is nil (no-op aside from the existing fields).
func (sc *senderContext) annotate(msg *chat.Message, info map[string]interface{}) {
	if msg == nil || msg.Sender == nil || info == nil {
		return
	}
	s := msg.Sender
	if s.Type != "" {
		info["sender_type"] = s.Type
	}
	if s.Name != "" {
		info["sender_resource"] = s.Name
	}
	if sc != nil {
		display := s.DisplayName
		if display == "" {
			if name, ok := sc.displayNames[s.Name]; ok {
				display = name
			}
		}
		if display != "" {
			info["sender_display_name"] = display
		}
		if sc.selfResource != "" && s.Name != "" {
			info["self"] = s.Name == sc.selfResource
		}
	}
}

// spaceFromMessageName returns "spaces/{space}" derived from a Chat message
// resource name like "spaces/AAAA/messages/msg1". Returns "" when the input
// does not match the expected shape so callers can keep output usable.
func spaceFromMessageName(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) < 4 || parts[0] != "spaces" || parts[2] != "messages" {
		return ""
	}
	return parts[0] + "/" + parts[1]
}
