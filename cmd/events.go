package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/omriariav/workspace-cli/internal/events"
	"github.com/spf13/cobra"
	workspaceevents "google.golang.org/api/workspaceevents/v1"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage Workspace Events subscriptions",
	Long: `Commands for the Google Workspace Events API: subscription lifecycle
(list, get, create, update/renew, delete, reactivate) and CloudEvent
notification decoding.

Only GA subscription targets and event types are supported: Chat space and
user events (including read-state), baseline Drive file and permission
events, and Meet conference, participant, recording, transcript, and
smart-note events. Notifications are delivered to an existing Pub/Sub topic
in the subscription's Cloud project; gws never creates or modifies topics.`,
}

var eventsSubscriptionsCmd = &cobra.Command{
	Use:   "subscriptions",
	Short: "Manage event subscriptions",
	Long:  "Lifecycle operations for Workspace Events API subscriptions.",
}

var eventsSubsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscriptions",
	Long: `Lists Workspace Events subscriptions.

The API requires filtering by at least one event type; --target is an
optional additional clause.

Examples:
  gws events subscriptions list --event-type google.workspace.chat.message.v1.created
  gws events subscriptions list --event-type google.workspace.chat.message.v1.created --target //chat.googleapis.com/spaces/AAA
  gws events subscriptions list --filter 'event_types:"google.workspace.chat.message.v1.created"'`,
	Args: cobra.NoArgs,
	RunE: runEventsSubsList,
}

var eventsSubsGetCmd = &cobra.Command{
	Use:   "get <subscription>",
	Short: "Get a subscription",
	Long:  "Gets a Workspace Events subscription by resource name (subscriptions/abc123 or the bare ID).",
	Args:  cobra.ExactArgs(1),
	RunE:  runEventsSubsGet,
}

var eventsSubsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a subscription",
	Long: `Creates a Workspace Events subscription delivering notifications to an
existing Pub/Sub topic in the same Cloud project.

Supported GA targets:
  //chat.googleapis.com/spaces/SPACE          Chat space (user auth, or admin-approved app auth)
  //chat.googleapis.com/spaces/-              all Chat spaces of the user (user auth)
  //cloudidentity.googleapis.com/users/USER   Chat user events incl. read-state, or Meet user events
  //drive.googleapis.com/files/FILE           Drive file (file + permission events)
  //drive.googleapis.com/drives/DRIVE         shared drive (file + permission events)
  //meet.googleapis.com/spaces/SPACE          Meet space

Credentials are minted with the least-privilege documented scopes for the
subscription's event classes. Subscriptions with --include-resource-data
have shorter maximum expirations; renew explicitly with
gws events subscriptions renew.

Examples:
  gws events subscriptions create \
    --target //chat.googleapis.com/spaces/AAA \
    --event-types google.workspace.chat.message.v1.created \
    --topic projects/my-project/topics/chat-events

  gws events subscriptions create \
    --target //drive.googleapis.com/files/FILE_ID \
    --event-types google.workspace.drive.file.v3.contentChanged,google.workspace.drive.permission.v3.created \
    --topic projects/my-project/topics/drive-events --include-resource-data`,
	Args: cobra.NoArgs,
	RunE: runEventsSubsCreate,
}

var eventsSubsUpdateCmd = &cobra.Command{
	Use:   "update <subscription>",
	Short: "Update a subscription (event types and/or TTL)",
	Long: `Updates a Workspace Events subscription. Pass --ttl to extend the
expiration (renew) and/or --event-types to replace the event-type list.
--ttl 0s requests the maximum allowed expiration.`,
	Args: cobra.ExactArgs(1),
	RunE: runEventsSubsUpdate,
}

var eventsSubsRenewCmd = &cobra.Command{
	Use:   "renew <subscription>",
	Short: "Renew a subscription (extend its expiration)",
	Long: `Renews a Workspace Events subscription by updating its TTL.
--ttl 0s (the default) requests the maximum allowed expiration. Subscriptions
that include resource data have shorter maximums and need renewing sooner.`,
	Args: cobra.ExactArgs(1),
	RunE: runEventsSubsRenew,
}

var eventsSubsDeleteCmd = &cobra.Command{
	Use:   "delete <subscription>",
	Short: "Delete a subscription",
	Long:  "Deletes a Workspace Events subscription.",
	Args:  cobra.ExactArgs(1),
	RunE:  runEventsSubsDelete,
}

var eventsSubsReactivateCmd = &cobra.Command{
	Use:   "reactivate <subscription>",
	Short: "Reactivate a suspended subscription",
	Long: `Reactivates a suspended Workspace Events subscription after the
suspension reason (for example a Pub/Sub permission problem) is fixed.`,
	Args: cobra.ExactArgs(1),
	RunE: runEventsSubsReactivate,
}

var eventsDecodeCmd = &cobra.Command{
	Use:   "decode",
	Short: "Decode a Pub/Sub CloudEvent notification from stdin",
	Long: `Reads a Pub/Sub message JSON (push envelope or bare message) from stdin
and decodes the CloudEvent attributes plus the base64 event payload.

Examples:
  gcloud pubsub subscriptions pull mysub --format=json | jq '.[0].message' | gws events decode
  cat push-body.json | gws events decode`,
	Args: cobra.NoArgs,
	RunE: runEventsDecode,
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(eventsSubscriptionsCmd)
	eventsCmd.AddCommand(eventsDecodeCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsListCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsGetCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsCreateCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsUpdateCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsRenewCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsDeleteCmd)
	eventsSubscriptionsCmd.AddCommand(eventsSubsReactivateCmd)

	eventsSubsListCmd.Flags().String("target", "", "Filter by target resource")
	eventsSubsListCmd.Flags().String("event-type", "", "Filter by event type")
	eventsSubsListCmd.Flags().String("filter", "", "Raw filter expression (overrides --target/--event-type)")
	eventsSubsListCmd.Flags().Int("max", 50, "Maximum subscriptions to return")

	eventsSubsCreateCmd.Flags().String("target", "", "Target resource (required)")
	eventsSubsCreateCmd.Flags().String("event-types", "", "Comma-separated GA event types (required)")
	eventsSubsCreateCmd.Flags().String("topic", "", "Existing Pub/Sub topic: projects/PROJECT/topics/TOPIC (required)")
	eventsSubsCreateCmd.Flags().Bool("include-resource-data", false, "Include the changed resource in payloads (shorter max expiration)")
	eventsSubsCreateCmd.Flags().String("ttl", "", "Subscription TTL (e.g. 86400s); 0s or empty = maximum allowed")
	eventsSubsCreateCmd.Flags().Bool("meet-created-scope", false, "Use the narrower meetings.space.created scope (Meet spaces you created)")
	eventsSubsCreateCmd.MarkFlagRequired("target")
	eventsSubsCreateCmd.MarkFlagRequired("event-types")
	eventsSubsCreateCmd.MarkFlagRequired("topic")

	eventsSubsUpdateCmd.Flags().String("event-types", "", "Comma-separated GA event types to replace the current set")
	eventsSubsUpdateCmd.Flags().String("ttl", "", "New TTL (e.g. 86400s); 0s = maximum allowed")
	eventsSubsUpdateCmd.Flags().Bool("meet-created-scope", false, "Use the narrower meetings.space.created scope (Meet spaces you created)")

	eventsSubsRenewCmd.Flags().String("ttl", "0s", "New TTL (e.g. 86400s); 0s = maximum allowed")
}

// subscriptionName normalizes a subscription argument to the
// subscriptions/{id} resource form.
func subscriptionName(arg string) string {
	if strings.HasPrefix(arg, "subscriptions/") {
		return arg
	}
	return "subscriptions/" + arg
}

// splitEventTypes parses a comma-separated event-type list.
func splitEventTypes(s string) []string {
	var out []string
	for _, et := range strings.Split(s, ",") {
		if et = strings.TrimSpace(et); et != "" {
			out = append(out, et)
		}
	}
	return out
}

// buildListFilter assembles the required list filter from flags. The
// Workspace Events API requires at least one event_types clause; a target
// clause alone is rejected server-side, so it is rejected locally too.
func buildListFilter(raw, target, eventType string) (string, error) {
	if raw != "" {
		if !strings.Contains(raw, "event_types") {
			return "", fmt.Errorf("the Workspace Events API requires an event_types clause in the list filter (e.g. --filter 'event_types:\"google.workspace.chat.message.v1.created\"')")
		}
		return raw, nil
	}
	if eventType == "" {
		return "", fmt.Errorf("the Workspace Events API requires filtering by event type: pass --event-type (optionally with --target), or a raw --filter with an event_types clause")
	}
	parts := []string{fmt.Sprintf("event_types:%q", eventType)}
	if target != "" {
		parts = append(parts, fmt.Sprintf("target_resource=%q", target))
	}
	return strings.Join(parts, " AND "), nil
}

// currentAuthMode resolves the configured auth mode for events scope
// decisions (validated again by the factory).
func currentAuthMode() string {
	modeCfg, err := auth.ResolveAuthMode(
		config.GetAuthMode(),
		config.GetServiceAccountKeyFile(),
		config.GetImpersonateUser(),
	)
	if err != nil {
		return config.GetAuthMode()
	}
	return modeCfg.Mode
}

// eventsReadService returns a client for read/delete lifecycle operations.
func eventsReadService(ctx context.Context) (*workspaceevents.Service, error) {
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return nil, err
	}
	return factory.WorkspaceEvents(events.ReadLifecycleScopes(currentAuthMode()))
}

// eventsMutationService returns a client for a mutating lifecycle operation
// on a subscription with the given target and event types, minting only the
// matching least-privilege scopes in service-account modes.
func eventsMutationService(ctx context.Context, target string, eventTypes []string, meetCreated bool) (*workspaceevents.Service, *events.Plan, error) {
	plan, err := events.PlanSubscription(target, eventTypes, currentAuthMode(), meetCreated)
	if err != nil {
		return nil, nil, err
	}
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return nil, nil, err
	}
	svc, err := factory.WorkspaceEvents(plan.Scopes)
	if err != nil {
		return nil, nil, err
	}
	return svc, plan, nil
}

// lookupSubscriptionPlanInput fetches an existing subscription with
// read-lifecycle credentials (chat.bot in app mode) and returns its target
// and event types, so mutations can mint exactly the matching granular
// scopes.
func lookupSubscriptionPlanInput(ctx context.Context, name string) (target string, eventTypes []string, err error) {
	svc, err := eventsReadService(ctx)
	if err != nil {
		return "", nil, err
	}
	sub, err := svc.Subscriptions.Get(name).Do()
	if err != nil {
		return "", nil, fmt.Errorf("failed to look up subscription %s to derive its scopes: %w", name, err)
	}
	return sub.TargetResource, sub.EventTypes, nil
}

func runEventsSubsList(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	raw, _ := cmd.Flags().GetString("filter")
	target, _ := cmd.Flags().GetString("target")
	eventType, _ := cmd.Flags().GetString("event-type")
	max, _ := cmd.Flags().GetInt("max")

	filter, err := buildListFilter(raw, target, eventType)
	if err != nil {
		return usageErrorf("%s", err.Error())
	}

	svc, err := eventsReadService(context.Background())
	if err != nil {
		return p.PrintError(err)
	}

	var subs []*workspaceevents.Subscription
	pageToken := ""
	for {
		call := svc.Subscriptions.List().Filter(filter).PageSize(int64(max))
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return p.PrintError(err)
		}
		subs = append(subs, resp.Subscriptions...)
		if resp.NextPageToken == "" || len(subs) >= max {
			break
		}
		pageToken = resp.NextPageToken
	}
	if len(subs) > max {
		subs = subs[:max]
	}

	return p.Print(map[string]interface{}{
		"count":         len(subs),
		"subscriptions": subs,
	})
}

func runEventsSubsGet(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	svc, err := eventsReadService(context.Background())
	if err != nil {
		return p.PrintError(err)
	}

	sub, err := svc.Subscriptions.Get(subscriptionName(args[0])).Do()
	if err != nil {
		return p.PrintError(err)
	}
	return p.Print(sub)
}

func runEventsSubsCreate(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	target, _ := cmd.Flags().GetString("target")
	eventTypesFlag, _ := cmd.Flags().GetString("event-types")
	topic, _ := cmd.Flags().GetString("topic")
	includeResourceData, _ := cmd.Flags().GetBool("include-resource-data")
	ttl, _ := cmd.Flags().GetString("ttl")
	meetCreated, _ := cmd.Flags().GetBool("meet-created-scope")

	if err := events.ValidateTopic(topic); err != nil {
		return usageErrorf("%s", err.Error())
	}
	eventTypes := splitEventTypes(eventTypesFlag)

	svc, plan, err := eventsMutationService(context.Background(), target, eventTypes, meetCreated)
	if err != nil {
		if _, planErr := events.PlanSubscription(target, eventTypes, currentAuthMode(), meetCreated); planErr != nil {
			return usageErrorf("%s", planErr.Error())
		}
		return p.PrintError(err)
	}

	sub := &workspaceevents.Subscription{
		TargetResource: target,
		EventTypes:     plan.EventTypes,
		NotificationEndpoint: &workspaceevents.NotificationEndpoint{
			PubsubTopic: topic,
		},
		PayloadOptions: &workspaceevents.PayloadOptions{
			IncludeResource: includeResourceData,
		},
	}
	if ttl != "" {
		sub.Ttl = ttl
	}

	op, err := svc.Subscriptions.Create(sub).Do()
	if err != nil {
		return p.PrintError(err)
	}

	return p.Print(map[string]interface{}{
		"status":    "created",
		"operation": op,
		"scopes":    plan.Scopes,
	})
}

func runEventsSubsUpdate(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	name := subscriptionName(args[0])

	eventTypesFlag, _ := cmd.Flags().GetString("event-types")
	ttl, _ := cmd.Flags().GetString("ttl")
	meetCreated, _ := cmd.Flags().GetBool("meet-created-scope")

	if eventTypesFlag == "" && ttl == "" {
		return usageErrorf("nothing to update: pass --ttl and/or --event-types")
	}

	eventTypes := splitEventTypes(eventTypesFlag)
	if eventTypesFlag != "" && len(eventTypes) == 0 {
		return usageErrorf("--event-types %q contains no event types: pass a comma-separated list of GA event types", eventTypesFlag)
	}

	return patchSubscription(p, name, eventTypes, ttl, meetCreated)
}

func runEventsSubsRenew(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ttl, _ := cmd.Flags().GetString("ttl")
	return patchSubscription(p, subscriptionName(args[0]), nil, ttl, false)
}

// patchSubscription updates a subscription's event types and/or TTL. The
// mutation credentials are minted from the subscription's exact event
// classes: from --event-types when replacing them, otherwise from the
// existing subscription (looked up with read-lifecycle credentials).
func patchSubscription(p interface {
	Print(interface{}) error
	PrintError(error) error
}, name string, eventTypes []string, ttl string, meetCreated bool) error {
	ctx := context.Background()

	target, existingTypes, err := lookupSubscriptionPlanInput(ctx, name)
	if err != nil {
		return p.PrintError(err)
	}

	var updateMask []string
	sub := &workspaceevents.Subscription{}
	planTypes := existingTypes
	if len(eventTypes) > 0 {
		sub.EventTypes = eventTypes
		planTypes = eventTypes
		updateMask = append(updateMask, "event_types")
	}
	if ttl != "" {
		sub.Ttl = ttl
		updateMask = append(updateMask, "ttl")
	}

	svc, plan, err := eventsMutationService(ctx, target, planTypes, meetCreated)
	if err != nil {
		if _, planErr := events.PlanSubscription(target, planTypes, currentAuthMode(), meetCreated); planErr != nil {
			return usageErrorf("%s", planErr.Error())
		}
		return p.PrintError(err)
	}

	op, err := svc.Subscriptions.Patch(name, sub).UpdateMask(strings.Join(updateMask, ",")).Do()
	if err != nil {
		return p.PrintError(err)
	}

	return p.Print(map[string]interface{}{
		"status":    "updated",
		"operation": op,
		"scopes":    plan.Scopes,
	})
}

func runEventsSubsDelete(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	svc, err := eventsReadService(context.Background())
	if err != nil {
		return p.PrintError(err)
	}

	op, err := svc.Subscriptions.Delete(subscriptionName(args[0])).Do()
	if err != nil {
		return p.PrintError(err)
	}

	return p.Print(map[string]interface{}{
		"status":    "deleted",
		"operation": op,
	})
}

func runEventsSubsReactivate(cmd *cobra.Command, args []string) error {
	p := GetPrinter()
	ctx := context.Background()
	name := subscriptionName(args[0])

	target, eventTypes, err := lookupSubscriptionPlanInput(ctx, name)
	if err != nil {
		return p.PrintError(err)
	}

	svc, plan, err := eventsMutationService(ctx, target, eventTypes, false)
	if err != nil {
		if _, planErr := events.PlanSubscription(target, eventTypes, currentAuthMode(), false); planErr != nil {
			return usageErrorf("%s", planErr.Error())
		}
		return p.PrintError(err)
	}

	op, err := svc.Subscriptions.Reactivate(name, &workspaceevents.ReactivateSubscriptionRequest{}).Do()
	if err != nil {
		return p.PrintError(err)
	}

	return p.Print(map[string]interface{}{
		"status":    "reactivated",
		"operation": op,
		"scopes":    plan.Scopes,
	})
}

func runEventsDecode(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	ev, err := events.DecodeCloudEvent(os.Stdin)
	if err != nil {
		return usageErrorf("%s", err.Error())
	}
	return p.Print(ev)
}
