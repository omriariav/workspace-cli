package cmd

// `--raw` + `--params` paths for the Chat list endpoints documented in #188:
//   * spaces.list           (gws chat spaces list / gws chat list --raw)
//   * spaces.members.list   (gws chat members list)
//   * spaces.messages.list  (gws chat messages list)
//
// Each runner emits the SDK response struct as JSON — Google's API shape:
// `{"spaces":[...],"nextPageToken":"..."}`, etc. With --all, the list field
// is concatenated across pages and nextPageToken is dropped.

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/api/chat/v1"
)

// chatSpacesCmd is a noun-style parent so callers can write
// `gws chat spaces list` per the API reference. `gws chat list` keeps
// working with its existing ergonomic output.
var chatSpacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "Chat spaces (API-shape friendly subcommands)",
}

var chatSpacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Chat spaces",
	Long: `List Chat spaces. Supports --raw and --params for programmatic use.

--raw emits the unmodified spaces.list response JSON.
--params overrides equivalent flags; for example:
  gws chat spaces list --params '{"pageSize":50,"filter":"spaceType = \"DIRECT_MESSAGE\""}' --raw --all`,
	RunE: runChatList,
}

var chatMessagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List messages in a Chat space",
	Long: `List messages in a Chat space.

Either pass <space-id> as a positional argument or set "parent" via --params.
Supports --raw and --params for programmatic use. Example:
  gws chat messages list --params '{"parent":"spaces/AAA","pageSize":50,"filter":"createTime > \"2025-01-01T00:00:00Z\""}' --raw --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runChatMessages,
}

var chatMembersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List members of a Chat space",
	Long: `List members of a Chat space.

Either pass <space-id> as a positional argument or set "parent" via --params.
Supports --raw and --params for programmatic use.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runChatMembers,
}

func init() {
	chatCmd.AddCommand(chatSpacesCmd)
	chatSpacesCmd.AddCommand(chatSpacesListCmd)
	chatMessagesCmd.AddCommand(chatMessagesListCmd)
	chatMembersCmd.AddCommand(chatMembersListCmd)

	// Mirror the underlying flag surface for the new paths.
	chatSpacesListCmd.Flags().String("filter", "", "Filter spaces (e.g. 'spaceType = \"DIRECT_MESSAGE\"')")
	chatSpacesListCmd.Flags().Int64("page-size", 100, "Number of spaces per page")
	chatSpacesListCmd.Flags().Int64("max", 0, "Maximum number of spaces to return (0 = all)")
	chatSpacesListCmd.Flags().Bool("all", false, "Fetch all matching results across pages (raw mode aggregates list field)")
	addRawParamsFlags(chatSpacesListCmd)

	chatMessagesListCmd.Flags().Int64("max", 25, "Maximum number of messages to return")
	chatMessagesListCmd.Flags().String("filter", "", "Filter messages (e.g. 'createTime > \"2024-01-01T00:00:00Z\"')")
	chatMessagesListCmd.Flags().String("order-by", "", "Order messages (e.g. 'createTime DESC')")
	chatMessagesListCmd.Flags().Bool("show-deleted", false, "Include deleted messages")
	chatMessagesListCmd.Flags().String("after", "", "Show messages after this time (RFC3339)")
	chatMessagesListCmd.Flags().String("before", "", "Show messages before this time (RFC3339)")
	chatMessagesListCmd.Flags().Bool("resolve-senders", false, "Resolve sender display names (non-raw mode only)")
	chatMessagesListCmd.Flags().Bool("all", false, "Fetch all matching results across pages (raw mode aggregates list field)")
	addRawParamsFlags(chatMessagesListCmd)

	chatMembersListCmd.Flags().Int64("max", 100, "Maximum number of members to return")
	chatMembersListCmd.Flags().String("filter", "", "Filter members (e.g. 'member.type = \"HUMAN\"')")
	chatMembersListCmd.Flags().Bool("show-groups", false, "Include Google Group memberships")
	chatMembersListCmd.Flags().Bool("show-invited", false, "Include invited memberships")
	chatMembersListCmd.Flags().Bool("all", false, "Fetch all matching results across pages (raw mode aggregates list field)")
	addRawParamsFlags(chatMembersListCmd)

	// Also expose --raw + --params on the existing leaf commands so
	// scripts that already use them have a smooth upgrade path.
	addRawParamsFlags(chatListCmd)
	addRawParamsFlags(chatMessagesCmd)
	addRawParamsFlags(chatMembersCmd)
	// --all is wired here for symmetry with the new list commands, but
	// the ergonomic chat messages/members runners do not honor it
	// (their default ergonomic shape is per-space and intentionally
	// bounded). Help text reflects that it only takes effect under
	// --raw on these commands.
	chatListCmd.Flags().Bool("all", false, "Fetch all matching results across pages (raw mode aggregates list field; ergonomic mode already returns all when --max=0)")
	chatMessagesCmd.Flags().Bool("all", false, "Raw mode only: fetch all matching pages and concatenate (no effect without --raw)")
	chatMembersCmd.Flags().Bool("all", false, "Raw mode only: fetch all matching pages and concatenate (no effect without --raw)")
}

// runChatListRaw implements `gws chat spaces list --raw` (and `gws chat list --raw`).
func runChatListRaw(cmd *cobra.Command, svc *chat.Service, filter string, pageSize, maxResults int64, fetchAll, maxExplicit bool) error {
	p := GetPrinter()
	params, perr := parseParams(cmd)
	if perr != nil {
		return p.PrintError(perr)
	}

	if v, ok := paramString(params, "filter"); ok {
		filter = v
	}
	if v, ok := paramInt64(params, "pageSize"); ok {
		pageSize = v
	}
	pageToken, _ := paramString(params, "pageToken")

	// Raw mode is verbatim: only honor --max when the caller set it.
	if !maxExplicit {
		maxResults = 0
	}
	// --all means "fetch every page" — drop --max even if it was set
	// (already 0 from above for the default case).
	if fetchAll {
		maxResults = 0
	}

	if pageSize <= 0 {
		pageSize = 100
	}
	// If the caller capped results, ask the server for at most that many
	// per page. Otherwise the server's nextPageToken points past the
	// full page (e.g. item 100) and clients that continue pagination
	// would skip the items we silently sliced off.
	if maxResults > 0 && !fetchAll && pageSize > maxResults {
		pageSize = maxResults
	}

	var aggregated *chat.ListSpacesResponse
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

		if aggregated == nil {
			aggregated = resp
		} else {
			aggregated.Spaces = append(aggregated.Spaces, resp.Spaces...)
			aggregated.NextPageToken = resp.NextPageToken
		}

		// Stop conditions.
		if resp.NextPageToken == "" {
			break
		}
		if !fetchAll {
			if maxResults > 0 && int64(len(aggregated.Spaces)) >= maxResults {
				break
			}
			// Without --all we only want one page unless caller paged via --params.
			break
		}
		if maxResults > 0 && int64(len(aggregated.Spaces)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	if aggregated == nil {
		aggregated = &chat.ListSpacesResponse{}
	}
	if maxResults > 0 && int64(len(aggregated.Spaces)) > maxResults {
		aggregated.Spaces = aggregated.Spaces[:maxResults]
	}
	if fetchAll {
		aggregated.NextPageToken = ""
	}
	return printRaw(aggregated)
}

// runChatMessagesRaw implements `gws chat messages list --raw`.
func runChatMessagesRaw(cmd *cobra.Command, svc *chat.Service, spaceName string, maxResults int64, filter, orderBy string, showDeleted, fetchAll, maxExplicit bool) error {
	p := GetPrinter()
	params, perr := parseParams(cmd)
	if perr != nil {
		return p.PrintError(perr)
	}

	if v, ok := paramString(params, "parent"); ok && v != "" {
		spaceName = v
	}
	if spaceName == "" {
		return p.PrintError(errors.New("chat messages list: a space name is required (positional arg or --params parent)"))
	}
	pageSize := int64(0)
	if v, ok := paramInt64(params, "pageSize"); ok {
		pageSize = v
	}
	if v, ok := paramString(params, "filter"); ok {
		filter = v
	}
	if v, ok := paramString(params, "orderBy"); ok {
		orderBy = v
	}
	if v, ok := paramBool(params, "showDeleted"); ok {
		showDeleted = v
	}
	pageToken, _ := paramString(params, "pageToken")

	// Raw mode is verbatim: drop the CLI default --max unless the
	// caller explicitly set it. --all also disables --max for symmetry.
	if !maxExplicit || fetchAll {
		maxResults = 0
	}

	var aggregated *chat.ListMessagesResponse
	for {
		thisPage := pageSize
		if thisPage <= 0 {
			thisPage = maxResults
			if thisPage <= 0 || thisPage > 1000 {
				thisPage = 1000
			}
		}
		// Clamp page size to the remaining budget so the server's
		// nextPageToken stays aligned with the items we hand back.
		if maxResults > 0 && !fetchAll {
			collected := int64(0)
			if aggregated != nil {
				collected = int64(len(aggregated.Messages))
			}
			remaining := maxResults - collected
			if remaining > 0 && thisPage > remaining {
				thisPage = remaining
			}
		}

		call := svc.Spaces.Messages.List(spaceName).PageSize(thisPage)
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

		if aggregated == nil {
			aggregated = resp
		} else {
			aggregated.Messages = append(aggregated.Messages, resp.Messages...)
			aggregated.NextPageToken = resp.NextPageToken
		}

		if resp.NextPageToken == "" {
			break
		}
		if !fetchAll {
			break
		}
		if maxResults > 0 && int64(len(aggregated.Messages)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	if aggregated == nil {
		aggregated = &chat.ListMessagesResponse{}
	}
	if maxResults > 0 && int64(len(aggregated.Messages)) > maxResults {
		aggregated.Messages = aggregated.Messages[:maxResults]
	}
	if fetchAll {
		aggregated.NextPageToken = ""
	}
	return printRaw(aggregated)
}

// runChatMembersRaw implements `gws chat members list --raw`.
func runChatMembersRaw(cmd *cobra.Command, svc *chat.Service, spaceName string, maxResults int64, filter string, showGroups, showInvited, fetchAll, maxExplicit bool) error {
	p := GetPrinter()
	params, perr := parseParams(cmd)
	if perr != nil {
		return p.PrintError(perr)
	}

	if v, ok := paramString(params, "parent"); ok && v != "" {
		spaceName = v
	}
	if spaceName == "" {
		return p.PrintError(errors.New("chat members list: a space name is required (positional arg or --params parent)"))
	}
	pageSize := int64(0)
	if v, ok := paramInt64(params, "pageSize"); ok {
		pageSize = v
	}
	if v, ok := paramString(params, "filter"); ok {
		filter = v
	}
	if v, ok := paramBool(params, "showGroups"); ok {
		showGroups = v
	}
	if v, ok := paramBool(params, "showInvited"); ok {
		showInvited = v
	}
	pageToken, _ := paramString(params, "pageToken")

	// Raw mode is verbatim: drop the CLI default --max unless the
	// caller explicitly set it. --all also disables --max.
	if !maxExplicit || fetchAll {
		maxResults = 0
	}

	// Clamp page size so the server's nextPageToken aligns with the
	// items we return — otherwise clients continuing pagination skip
	// items we sliced off locally.
	if maxResults > 0 && !fetchAll && (pageSize <= 0 || pageSize > maxResults) {
		pageSize = maxResults
	}

	if pageSize <= 0 {
		pageSize = maxResults
		if pageSize <= 0 || pageSize > 100 {
			pageSize = 100
		}
	}

	var aggregated *chat.ListMembershipsResponse
	for {
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
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list members: %w", err))
		}

		if aggregated == nil {
			aggregated = resp
		} else {
			aggregated.Memberships = append(aggregated.Memberships, resp.Memberships...)
			aggregated.NextPageToken = resp.NextPageToken
		}

		if resp.NextPageToken == "" {
			break
		}
		if !fetchAll {
			break
		}
		if maxResults > 0 && int64(len(aggregated.Memberships)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	if aggregated == nil {
		aggregated = &chat.ListMembershipsResponse{}
	}
	if maxResults > 0 && int64(len(aggregated.Memberships)) > maxResults {
		aggregated.Memberships = aggregated.Memberships[:maxResults]
	}
	if fetchAll {
		aggregated.NextPageToken = ""
	}
	return printRaw(aggregated)
}
