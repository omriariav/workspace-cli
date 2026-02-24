package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
)

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Manage Google Calendar",
	Long:  "Commands for interacting with Google Calendar.",
}

var calendarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendars",
	Long:  "Lists all calendars you have access to.",
	RunE:  runCalendarList,
}

var calendarEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List events",
	Long:  "Lists upcoming events from a calendar.",
	RunE:  runCalendarEvents,
}

var calendarCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an event",
	Long:  "Creates a new calendar event.",
	RunE:  runCalendarCreate,
}

var calendarUpdateCmd = &cobra.Command{
	Use:   "update <event-id>",
	Short: "Update an event",
	Long: `Updates an existing calendar event. Only specified fields are changed.

Examples:
  gws calendar update abc123 --title "New Title"
  gws calendar update abc123 --start "2024-02-01 14:00" --end "2024-02-01 15:00"
  gws calendar update abc123 --add-attendees user@example.com
  gws calendar update abc123 --location "Room 42" --description "Updated agenda"`,
	Args: cobra.ExactArgs(1),
	RunE: runCalendarUpdate,
}

var calendarDeleteCmd = &cobra.Command{
	Use:   "delete <event-id>",
	Short: "Delete an event",
	Long: `Deletes a calendar event.

Examples:
  gws calendar delete abc123
  gws calendar delete abc123 --calendar-id work@group.calendar.google.com`,
	Args: cobra.ExactArgs(1),
	RunE: runCalendarDelete,
}

var calendarRsvpCmd = &cobra.Command{
	Use:   "rsvp <event-id>",
	Short: "Respond to an event invitation",
	Long: `Sets your RSVP status for a calendar event.

Valid responses: accepted, declined, tentative

Examples:
  gws calendar rsvp abc123 --response accepted
  gws calendar rsvp abc123 --response declined
  gws calendar rsvp abc123 --response tentative`,
	Args: cobra.ExactArgs(1),
	RunE: runCalendarRsvp,
}

// --- New event commands ---

var calendarGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get event by ID",
	Long: `Gets a single calendar event by its ID.

Examples:
  gws calendar get --id abc123
  gws calendar get --id abc123 --calendar-id work@group.calendar.google.com`,
	RunE: runCalendarGet,
}

var calendarQuickAddCmd = &cobra.Command{
	Use:   "quick-add",
	Short: "Quick add event from text",
	Long: `Creates an event from a text string using Google's natural language processing.

Examples:
  gws calendar quick-add --text "Lunch with John tomorrow at noon"
  gws calendar quick-add --text "Team meeting Friday 3pm-4pm"`,
	RunE: runCalendarQuickAdd,
}

var calendarInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "List instances of recurring event",
	Long: `Lists all instances of a recurring calendar event.

Examples:
  gws calendar instances --id abc123
  gws calendar instances --id abc123 --max 10 --from "2024-03-01" --to "2024-06-01"`,
	RunE: runCalendarInstances,
}

var calendarMoveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move event to another calendar",
	Long: `Moves an event from one calendar to another.

Examples:
  gws calendar move --id abc123 --destination work@group.calendar.google.com`,
	RunE: runCalendarMove,
}

// --- Calendar CRUD commands ---

var calendarGetCalendarCmd = &cobra.Command{
	Use:   "get-calendar",
	Short: "Get calendar metadata",
	Long: `Gets metadata for a calendar by its ID.

Examples:
  gws calendar get-calendar --id primary
  gws calendar get-calendar --id work@group.calendar.google.com`,
	RunE: runCalendarGetCalendar,
}

var calendarCreateCalendarCmd = &cobra.Command{
	Use:   "create-calendar",
	Short: "Create a secondary calendar",
	Long: `Creates a new secondary calendar.

Examples:
  gws calendar create-calendar --summary "Work Projects"
  gws calendar create-calendar --summary "Gym" --description "Workout schedule" --timezone "America/New_York"`,
	RunE: runCalendarCreateCalendar,
}

var calendarUpdateCalendarCmd = &cobra.Command{
	Use:   "update-calendar",
	Short: "Update a calendar",
	Long: `Updates an existing calendar's metadata.

Examples:
  gws calendar update-calendar --id cal123 --summary "New Name"
  gws calendar update-calendar --id cal123 --description "Updated description" --timezone "Europe/London"`,
	RunE: runCalendarUpdateCalendar,
}

var calendarDeleteCalendarCmd = &cobra.Command{
	Use:   "delete-calendar",
	Short: "Delete a secondary calendar",
	Long: `Deletes a secondary calendar. Cannot delete the primary calendar.

Examples:
  gws calendar delete-calendar --id cal123@group.calendar.google.com`,
	RunE: runCalendarDeleteCalendar,
}

var calendarClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all events from a calendar",
	Long: `Clears all events from a calendar. Use with caution.

Examples:
  gws calendar clear
  gws calendar clear --calendar-id work@group.calendar.google.com`,
	RunE: runCalendarClear,
}

// --- Subscription commands ---

var calendarSubscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to a public calendar",
	Long: `Adds a public calendar to your calendar list.

Examples:
  gws calendar subscribe --id en.usa#holiday@group.v.calendar.google.com`,
	RunE: runCalendarSubscribe,
}

var calendarUnsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe",
	Short: "Unsubscribe from a calendar",
	Long: `Removes a calendar from your calendar list (unsubscribe).

Examples:
  gws calendar unsubscribe --id en.usa#holiday@group.v.calendar.google.com`,
	RunE: runCalendarUnsubscribe,
}

var calendarCalendarInfoCmd = &cobra.Command{
	Use:   "calendar-info",
	Short: "Get calendar list entry (subscription info)",
	Long: `Gets the calendar list entry for a calendar (subscription settings, color, visibility).

Examples:
  gws calendar calendar-info --id primary
  gws calendar calendar-info --id work@group.calendar.google.com`,
	RunE: runCalendarCalendarInfo,
}

var calendarUpdateSubscriptionCmd = &cobra.Command{
	Use:   "update-subscription",
	Short: "Update subscription settings",
	Long: `Updates subscription settings for a calendar in your list (color, hidden, summary override).

Examples:
  gws calendar update-subscription --id cal123 --color-id 7
  gws calendar update-subscription --id cal123 --hidden
  gws calendar update-subscription --id cal123 --summary-override "My Custom Name"`,
	RunE: runCalendarUpdateSubscription,
}

// --- ACL commands ---

var calendarAclCmd = &cobra.Command{
	Use:   "acl",
	Short: "List access control rules",
	Long: `Lists access control rules for a calendar.

Examples:
  gws calendar acl
  gws calendar acl --calendar-id work@group.calendar.google.com`,
	RunE: runCalendarAcl,
}

var calendarShareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share calendar with a user",
	Long: `Shares a calendar with a user by creating an ACL rule.

Valid roles: reader, writer, owner, freeBusyReader

Examples:
  gws calendar share --email user@example.com --role reader
  gws calendar share --email user@example.com --role writer --calendar-id work@group.calendar.google.com`,
	RunE: runCalendarShare,
}

var calendarUnshareCmd = &cobra.Command{
	Use:   "unshare",
	Short: "Remove calendar access",
	Long: `Removes an access control rule from a calendar.

Examples:
  gws calendar unshare --rule-id "user:user@example.com"`,
	RunE: runCalendarUnshare,
}

var calendarUpdateAclCmd = &cobra.Command{
	Use:   "update-acl",
	Short: "Update access control rule",
	Long: `Updates an existing access control rule for a calendar.

Valid roles: reader, writer, owner, freeBusyReader

Examples:
  gws calendar update-acl --rule-id "user:user@example.com" --role writer`,
	RunE: runCalendarUpdateAcl,
}

// --- Other commands ---

var calendarFreebusyCmd = &cobra.Command{
	Use:   "freebusy",
	Short: "Query free/busy information",
	Long: `Queries free/busy information for one or more calendars.

Examples:
  gws calendar freebusy --from "2024-03-01 09:00" --to "2024-03-01 17:00"
  gws calendar freebusy --from "2024-03-01 09:00" --to "2024-03-01 17:00" --calendars "primary,user@example.com"`,
	RunE: runCalendarFreebusy,
}

var calendarColorsCmd = &cobra.Command{
	Use:   "colors",
	Short: "List available calendar colors",
	Long:  "Lists all available calendar and event colors.",
	RunE:  runCalendarColors,
}

var calendarSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "List user calendar settings",
	Long:  "Lists all user calendar settings.",
	RunE:  runCalendarSettings,
}

var validRsvpResponses = map[string]bool{
	"accepted":  true,
	"declined":  true,
	"tentative": true,
}

var validAclRoles = map[string]bool{
	"reader":         true,
	"writer":         true,
	"owner":          true,
	"freeBusyReader": true,
}

func init() {
	rootCmd.AddCommand(calendarCmd)
	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarEventsCmd)
	calendarCmd.AddCommand(calendarCreateCmd)
	calendarCmd.AddCommand(calendarUpdateCmd)
	calendarCmd.AddCommand(calendarDeleteCmd)
	calendarCmd.AddCommand(calendarRsvpCmd)
	calendarCmd.AddCommand(calendarGetCmd)
	calendarCmd.AddCommand(calendarQuickAddCmd)
	calendarCmd.AddCommand(calendarInstancesCmd)
	calendarCmd.AddCommand(calendarMoveCmd)
	calendarCmd.AddCommand(calendarGetCalendarCmd)
	calendarCmd.AddCommand(calendarCreateCalendarCmd)
	calendarCmd.AddCommand(calendarUpdateCalendarCmd)
	calendarCmd.AddCommand(calendarDeleteCalendarCmd)
	calendarCmd.AddCommand(calendarClearCmd)
	calendarCmd.AddCommand(calendarSubscribeCmd)
	calendarCmd.AddCommand(calendarUnsubscribeCmd)
	calendarCmd.AddCommand(calendarCalendarInfoCmd)
	calendarCmd.AddCommand(calendarUpdateSubscriptionCmd)
	calendarCmd.AddCommand(calendarAclCmd)
	calendarCmd.AddCommand(calendarShareCmd)
	calendarCmd.AddCommand(calendarUnshareCmd)
	calendarCmd.AddCommand(calendarUpdateAclCmd)
	calendarCmd.AddCommand(calendarFreebusyCmd)
	calendarCmd.AddCommand(calendarColorsCmd)
	calendarCmd.AddCommand(calendarSettingsCmd)

	// Events flags
	calendarEventsCmd.Flags().Int("days", 7, "Number of days to look ahead")
	calendarEventsCmd.Flags().String("calendar-id", "primary", "Calendar ID (default: primary)")
	calendarEventsCmd.Flags().Int64("max", 50, "Maximum number of events")
	calendarEventsCmd.Flags().Bool("pending", false, "Only show events with pending RSVP (needsAction)")

	// Create flags
	calendarCreateCmd.Flags().String("title", "", "Event title (required)")
	calendarCreateCmd.Flags().String("start", "", "Start time in RFC3339 format or 'YYYY-MM-DD HH:MM' (required)")
	calendarCreateCmd.Flags().String("end", "", "End time in RFC3339 format or 'YYYY-MM-DD HH:MM' (required)")
	calendarCreateCmd.Flags().String("calendar-id", "primary", "Calendar ID (default: primary)")
	calendarCreateCmd.Flags().String("description", "", "Event description")
	calendarCreateCmd.Flags().String("location", "", "Event location")
	calendarCreateCmd.Flags().StringSlice("attendees", nil, "Attendee email addresses")
	calendarCreateCmd.MarkFlagRequired("title")
	calendarCreateCmd.MarkFlagRequired("start")
	calendarCreateCmd.MarkFlagRequired("end")

	// Update flags
	calendarUpdateCmd.Flags().String("title", "", "New event title")
	calendarUpdateCmd.Flags().String("start", "", "New start time")
	calendarUpdateCmd.Flags().String("end", "", "New end time")
	calendarUpdateCmd.Flags().String("description", "", "New event description")
	calendarUpdateCmd.Flags().String("location", "", "New event location")
	calendarUpdateCmd.Flags().StringSlice("add-attendees", nil, "Attendee emails to add")
	calendarUpdateCmd.Flags().String("calendar-id", "primary", "Calendar ID")

	// Delete flags
	calendarDeleteCmd.Flags().String("calendar-id", "primary", "Calendar ID")

	// RSVP flags
	calendarRsvpCmd.Flags().String("response", "", "Response: accepted, declined, tentative (required)")
	calendarRsvpCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarRsvpCmd.Flags().String("message", "", "Optional message to include with your RSVP (notifies all attendees)")
	calendarRsvpCmd.MarkFlagRequired("response")

	// Get event flags
	calendarGetCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarGetCmd.Flags().String("id", "", "Event ID (required)")
	calendarGetCmd.MarkFlagRequired("id")

	// Quick-add flags
	calendarQuickAddCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarQuickAddCmd.Flags().String("text", "", "Text describing the event (required)")
	calendarQuickAddCmd.MarkFlagRequired("text")

	// Instances flags
	calendarInstancesCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarInstancesCmd.Flags().String("id", "", "Recurring event ID (required)")
	calendarInstancesCmd.Flags().Int64("max", 50, "Maximum number of instances")
	calendarInstancesCmd.Flags().String("from", "", "Start of time range (RFC3339 or YYYY-MM-DD)")
	calendarInstancesCmd.Flags().String("to", "", "End of time range (RFC3339 or YYYY-MM-DD)")
	calendarInstancesCmd.MarkFlagRequired("id")

	// Move flags
	calendarMoveCmd.Flags().String("calendar-id", "primary", "Source calendar ID")
	calendarMoveCmd.Flags().String("id", "", "Event ID (required)")
	calendarMoveCmd.Flags().String("destination", "", "Destination calendar ID (required)")
	calendarMoveCmd.MarkFlagRequired("id")
	calendarMoveCmd.MarkFlagRequired("destination")

	// Get-calendar flags
	calendarGetCalendarCmd.Flags().String("id", "", "Calendar ID (required)")
	calendarGetCalendarCmd.MarkFlagRequired("id")

	// Create-calendar flags
	calendarCreateCalendarCmd.Flags().String("summary", "", "Calendar name (required)")
	calendarCreateCalendarCmd.Flags().String("description", "", "Calendar description")
	calendarCreateCalendarCmd.Flags().String("timezone", "", "Calendar timezone (e.g. America/New_York)")
	calendarCreateCalendarCmd.MarkFlagRequired("summary")

	// Update-calendar flags
	calendarUpdateCalendarCmd.Flags().String("id", "", "Calendar ID (required)")
	calendarUpdateCalendarCmd.Flags().String("summary", "", "New calendar name")
	calendarUpdateCalendarCmd.Flags().String("description", "", "New calendar description")
	calendarUpdateCalendarCmd.Flags().String("timezone", "", "New calendar timezone")
	calendarUpdateCalendarCmd.MarkFlagRequired("id")

	// Delete-calendar flags
	calendarDeleteCalendarCmd.Flags().String("id", "", "Calendar ID (required)")
	calendarDeleteCalendarCmd.MarkFlagRequired("id")

	// Clear flags
	calendarClearCmd.Flags().String("calendar-id", "primary", "Calendar ID")

	// Subscribe flags
	calendarSubscribeCmd.Flags().String("id", "", "Calendar ID to subscribe to (required)")
	calendarSubscribeCmd.MarkFlagRequired("id")

	// Unsubscribe flags
	calendarUnsubscribeCmd.Flags().String("id", "", "Calendar ID to unsubscribe from (required)")
	calendarUnsubscribeCmd.MarkFlagRequired("id")

	// Calendar-info flags
	calendarCalendarInfoCmd.Flags().String("id", "", "Calendar ID (required)")
	calendarCalendarInfoCmd.MarkFlagRequired("id")

	// Update-subscription flags
	calendarUpdateSubscriptionCmd.Flags().String("id", "", "Calendar ID (required)")
	calendarUpdateSubscriptionCmd.Flags().String("color-id", "", "Color ID (use 'gws calendar colors' to list valid IDs)")
	calendarUpdateSubscriptionCmd.Flags().Bool("hidden", false, "Hide calendar from the list")
	calendarUpdateSubscriptionCmd.Flags().String("summary-override", "", "Custom display name")
	calendarUpdateSubscriptionCmd.MarkFlagRequired("id")

	// ACL flags
	calendarAclCmd.Flags().String("calendar-id", "primary", "Calendar ID")

	// Share flags
	calendarShareCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarShareCmd.Flags().String("email", "", "Email address to share with (required)")
	calendarShareCmd.Flags().String("role", "", "Access role: reader, writer, owner, freeBusyReader (required)")
	calendarShareCmd.MarkFlagRequired("email")
	calendarShareCmd.MarkFlagRequired("role")

	// Unshare flags
	calendarUnshareCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarUnshareCmd.Flags().String("rule-id", "", "ACL rule ID (required)")
	calendarUnshareCmd.MarkFlagRequired("rule-id")

	// Update-acl flags
	calendarUpdateAclCmd.Flags().String("calendar-id", "primary", "Calendar ID")
	calendarUpdateAclCmd.Flags().String("rule-id", "", "ACL rule ID (required)")
	calendarUpdateAclCmd.Flags().String("role", "", "New access role: reader, writer, owner, freeBusyReader (required)")
	calendarUpdateAclCmd.MarkFlagRequired("rule-id")
	calendarUpdateAclCmd.MarkFlagRequired("role")

	// Freebusy flags
	calendarFreebusyCmd.Flags().String("from", "", "Start of time range (required)")
	calendarFreebusyCmd.Flags().String("to", "", "End of time range (required)")
	calendarFreebusyCmd.Flags().String("calendars", "primary", "Comma-separated calendar IDs")
	calendarFreebusyCmd.MarkFlagRequired("from")
	calendarFreebusyCmd.MarkFlagRequired("to")
}

func runCalendarList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	resp, err := svc.CalendarList.List().Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list calendars: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Items))
	for _, cal := range resp.Items {
		calInfo := map[string]interface{}{
			"id":      cal.Id,
			"summary": cal.Summary,
			"primary": cal.Primary,
		}
		if cal.Description != "" {
			calInfo["description"] = cal.Description
		}
		results = append(results, calInfo)
	}

	return p.Print(map[string]interface{}{
		"calendars": results,
		"count":     len(results),
	})
}

func runCalendarEvents(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	days, _ := cmd.Flags().GetInt("days")
	calendarID, _ := cmd.Flags().GetString("calendar-id")
	maxResults, _ := cmd.Flags().GetInt64("max")
	pending, _ := cmd.Flags().GetBool("pending")

	now := time.Now()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)

	resp, err := svc.Events.List(calendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		MaxResults(maxResults).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list events: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Items))
	for _, event := range resp.Items {
		if event == nil {
			continue
		}
		eventInfo := mapEventToOutput(event)

		if pending {
			rs, _ := eventInfo["response_status"].(string)
			if rs != "needsAction" {
				continue
			}
		}

		results = append(results, eventInfo)
	}

	return p.Print(map[string]interface{}{
		"events": results,
		"count":  len(results),
	})
}

func runCalendarCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	title, _ := cmd.Flags().GetString("title")
	startStr, _ := cmd.Flags().GetString("start")
	endStr, _ := cmd.Flags().GetString("end")
	calendarID, _ := cmd.Flags().GetString("calendar-id")
	description, _ := cmd.Flags().GetString("description")
	location, _ := cmd.Flags().GetString("location")
	attendees, _ := cmd.Flags().GetStringSlice("attendees")

	// Parse times
	startTime, err := parseTime(startStr)
	if err != nil {
		return p.PrintError(fmt.Errorf("invalid start time: %w", err))
	}
	endTime, err := parseTime(endStr)
	if err != nil {
		return p.PrintError(fmt.Errorf("invalid end time: %w", err))
	}

	startDT := &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)}
	if tz := resolveIANA(startTime); tz != "" {
		startDT.TimeZone = tz
	}
	endDT := &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)}
	if tz := resolveIANA(endTime); tz != "" {
		endDT.TimeZone = tz
	}

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Location:    location,
		Start:       startDT,
		End:         endDT,
	}

	// Add attendees
	if len(attendees) > 0 {
		eventAttendees := make([]*calendar.EventAttendee, len(attendees))
		for i, email := range attendees {
			eventAttendees[i] = &calendar.EventAttendee{Email: email}
		}
		event.Attendees = eventAttendees
	}

	created, err := svc.Events.Insert(calendarID, event).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create event: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":       "created",
		"id":           created.Id,
		"html_link":    created.HtmlLink,
		"hangout_link": created.HangoutLink,
	})
}

func runCalendarUpdate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	updateFlags := []string{"title", "start", "end", "description", "location", "add-attendees"}
	hasChanges := false
	for _, flag := range updateFlags {
		if cmd.Flags().Changed(flag) {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		return p.PrintError(fmt.Errorf("at least one update flag is required (--title, --start, --end, --description, --location, --add-attendees)"))
	}

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	eventID := args[0]
	calendarID, _ := cmd.Flags().GetString("calendar-id")

	// Build patch with only changed fields to avoid triggering
	// unnecessary notifications or overwriting server-side fields
	patch := &calendar.Event{}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		patch.Summary = title
	}
	if cmd.Flags().Changed("description") {
		description, _ := cmd.Flags().GetString("description")
		patch.Description = description
	}
	if cmd.Flags().Changed("location") {
		location, _ := cmd.Flags().GetString("location")
		patch.Location = location
	}
	if cmd.Flags().Changed("start") {
		startStr, _ := cmd.Flags().GetString("start")
		startTime, err := parseTime(startStr)
		if err != nil {
			return p.PrintError(fmt.Errorf("invalid start time: %w", err))
		}
		patchStart := &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)}
		if tz := resolveIANA(startTime); tz != "" {
			patchStart.TimeZone = tz
		}
		patch.Start = patchStart
	}
	if cmd.Flags().Changed("end") {
		endStr, _ := cmd.Flags().GetString("end")
		endTime, err := parseTime(endStr)
		if err != nil {
			return p.PrintError(fmt.Errorf("invalid end time: %w", err))
		}
		patchEnd := &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)}
		if tz := resolveIANA(endTime); tz != "" {
			patchEnd.TimeZone = tz
		}
		patch.End = patchEnd
	}
	if cmd.Flags().Changed("add-attendees") {
		// For attendees we need the existing list, so fetch the event
		event, err := svc.Events.Get(calendarID, eventID).Fields("attendees").Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get event: %w", err))
		}
		newAttendees, _ := cmd.Flags().GetStringSlice("add-attendees")
		patch.Attendees = event.Attendees
		for _, email := range newAttendees {
			patch.Attendees = append(patch.Attendees, &calendar.EventAttendee{Email: email})
		}
	}

	updated, err := svc.Events.Patch(calendarID, eventID, patch).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update event: %w", err))
	}

	result := map[string]interface{}{
		"status":    "updated",
		"id":        updated.Id,
		"summary":   updated.Summary,
		"html_link": updated.HtmlLink,
	}

	if updated.Start != nil {
		if updated.Start.DateTime != "" {
			result["start"] = updated.Start.DateTime
		} else {
			result["start"] = updated.Start.Date
		}
	}
	if updated.End != nil {
		if updated.End.DateTime != "" {
			result["end"] = updated.End.DateTime
		} else {
			result["end"] = updated.End.Date
		}
	}

	return p.Print(result)
}

func runCalendarDelete(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	eventID := args[0]
	calendarID, _ := cmd.Flags().GetString("calendar-id")

	// Get event info first for the response
	event, err := svc.Events.Get(calendarID, eventID).Fields("summary").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get event: %w", err))
	}

	err = svc.Events.Delete(calendarID, eventID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete event: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":  "deleted",
		"id":      eventID,
		"summary": event.Summary,
	})
}

func runCalendarRsvp(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	eventID := args[0]
	calendarID, _ := cmd.Flags().GetString("calendar-id")
	response, _ := cmd.Flags().GetString("response")
	message, _ := cmd.Flags().GetString("message")

	if !validRsvpResponses[response] {
		return p.PrintError(fmt.Errorf("invalid response '%s': must be accepted, declined, or tentative", response))
	}

	// Get the event to find our attendee entry
	event, err := svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get event: %w", err))
	}

	// Find and update our RSVP status
	// The "self" field indicates the current user's attendee entry
	found := false
	for _, attendee := range event.Attendees {
		if attendee.Self {
			attendee.ResponseStatus = response
			if message != "" {
				attendee.Comment = message
			}
			found = true
			break
		}
	}

	if !found {
		return p.PrintError(fmt.Errorf("you are not an attendee of this event"))
	}

	patchCall := svc.Events.Patch(calendarID, eventID, &calendar.Event{
		Attendees: event.Attendees,
	})
	if message != "" {
		patchCall = patchCall.SendUpdates("all")
	}

	updated, err := patchCall.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update RSVP: %w", err))
	}

	result := map[string]interface{}{
		"status":    response,
		"id":        updated.Id,
		"summary":   updated.Summary,
		"html_link": updated.HtmlLink,
	}
	if message != "" {
		result["message"] = message
	}

	return p.Print(result)
}

// --- New event command implementations ---

func runCalendarGet(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	eventID, _ := cmd.Flags().GetString("id")

	event, err := svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get event: %w", err))
	}

	return p.Print(mapEventToOutput(event))
}

func runCalendarQuickAdd(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	text, _ := cmd.Flags().GetString("text")

	event, err := svc.Events.QuickAdd(calendarID, text).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to quick-add event: %w", err))
	}

	return p.Print(mapEventToOutput(event))
}

func runCalendarInstances(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	eventID, _ := cmd.Flags().GetString("id")
	maxResults, _ := cmd.Flags().GetInt64("max")

	call := svc.Events.Instances(calendarID, eventID).MaxResults(maxResults)

	if cmd.Flags().Changed("from") {
		fromStr, _ := cmd.Flags().GetString("from")
		fromTime, err := parseTime(fromStr)
		if err != nil {
			return p.PrintError(fmt.Errorf("invalid --from time: %w", err))
		}
		call = call.TimeMin(fromTime.Format(time.RFC3339))
	}
	if cmd.Flags().Changed("to") {
		toStr, _ := cmd.Flags().GetString("to")
		toTime, err := parseTime(toStr)
		if err != nil {
			return p.PrintError(fmt.Errorf("invalid --to time: %w", err))
		}
		call = call.TimeMax(toTime.Format(time.RFC3339))
	}

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list instances: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Items))
	for _, event := range resp.Items {
		if event == nil {
			continue
		}
		results = append(results, mapEventToOutput(event))
	}

	return p.Print(map[string]interface{}{
		"instances": results,
		"count":     len(results),
	})
}

func runCalendarMove(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	eventID, _ := cmd.Flags().GetString("id")
	destination, _ := cmd.Flags().GetString("destination")

	event, err := svc.Events.Move(calendarID, eventID, destination).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to move event: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "moved",
		"id":          event.Id,
		"summary":     event.Summary,
		"destination": destination,
		"html_link":   event.HtmlLink,
	})
}

// --- Calendar CRUD implementations ---

func runCalendarGetCalendar(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	cal, err := svc.Calendars.Get(calID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get calendar: %w", err))
	}

	result := map[string]interface{}{
		"id":      cal.Id,
		"summary": cal.Summary,
	}
	if cal.Description != "" {
		result["description"] = cal.Description
	}
	if cal.TimeZone != "" {
		result["timezone"] = cal.TimeZone
	}
	if cal.Location != "" {
		result["location"] = cal.Location
	}
	if cal.Etag != "" {
		result["etag"] = cal.Etag
	}

	return p.Print(result)
}

func runCalendarCreateCalendar(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	summary, _ := cmd.Flags().GetString("summary")
	description, _ := cmd.Flags().GetString("description")
	timezone, _ := cmd.Flags().GetString("timezone")

	cal := &calendar.Calendar{
		Summary: summary,
	}
	if description != "" {
		cal.Description = description
	}
	if timezone != "" {
		cal.TimeZone = timezone
	}

	created, err := svc.Calendars.Insert(cal).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create calendar: %w", err))
	}

	result := map[string]interface{}{
		"status":  "created",
		"id":      created.Id,
		"summary": created.Summary,
	}
	if created.Description != "" {
		result["description"] = created.Description
	}
	if created.TimeZone != "" {
		result["timezone"] = created.TimeZone
	}

	return p.Print(result)
}

func runCalendarUpdateCalendar(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	// Fetch existing calendar to patch
	cal, err := svc.Calendars.Get(calID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get calendar: %w", err))
	}

	if cmd.Flags().Changed("summary") {
		summary, _ := cmd.Flags().GetString("summary")
		cal.Summary = summary
	}
	if cmd.Flags().Changed("description") {
		description, _ := cmd.Flags().GetString("description")
		cal.Description = description
	}
	if cmd.Flags().Changed("timezone") {
		timezone, _ := cmd.Flags().GetString("timezone")
		cal.TimeZone = timezone
	}

	updated, err := svc.Calendars.Update(calID, cal).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update calendar: %w", err))
	}

	result := map[string]interface{}{
		"status":  "updated",
		"id":      updated.Id,
		"summary": updated.Summary,
	}
	if updated.Description != "" {
		result["description"] = updated.Description
	}
	if updated.TimeZone != "" {
		result["timezone"] = updated.TimeZone
	}

	return p.Print(result)
}

func runCalendarDeleteCalendar(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	err = svc.Calendars.Delete(calID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete calendar: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "deleted",
		"id":     calID,
	})
}

func runCalendarClear(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")

	err = svc.Calendars.Clear(calendarID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to clear calendar: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "cleared",
		"calendar_id": calendarID,
	})
}

// --- Subscription implementations ---

func runCalendarSubscribe(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	entry, err := svc.CalendarList.Insert(&calendar.CalendarListEntry{
		Id: calID,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to subscribe: %w", err))
	}

	result := map[string]interface{}{
		"status":  "subscribed",
		"id":      entry.Id,
		"summary": entry.Summary,
	}
	if entry.Description != "" {
		result["description"] = entry.Description
	}

	return p.Print(result)
}

func runCalendarUnsubscribe(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	err = svc.CalendarList.Delete(calID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to unsubscribe: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "unsubscribed",
		"id":     calID,
	})
}

func runCalendarCalendarInfo(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	entry, err := svc.CalendarList.Get(calID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get calendar info: %w", err))
	}

	result := map[string]interface{}{
		"id":      entry.Id,
		"summary": entry.Summary,
		"primary": entry.Primary,
	}
	if entry.Description != "" {
		result["description"] = entry.Description
	}
	if entry.TimeZone != "" {
		result["timezone"] = entry.TimeZone
	}
	if entry.ColorId != "" {
		result["color_id"] = entry.ColorId
	}
	if entry.BackgroundColor != "" {
		result["background_color"] = entry.BackgroundColor
	}
	if entry.ForegroundColor != "" {
		result["foreground_color"] = entry.ForegroundColor
	}
	if entry.SummaryOverride != "" {
		result["summary_override"] = entry.SummaryOverride
	}
	if entry.Hidden {
		result["hidden"] = true
	}
	if entry.Selected {
		result["selected"] = true
	}
	if entry.AccessRole != "" {
		result["access_role"] = entry.AccessRole
	}

	return p.Print(result)
}

func runCalendarUpdateSubscription(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calID, _ := cmd.Flags().GetString("id")

	patch := &calendar.CalendarListEntry{}
	if cmd.Flags().Changed("color-id") {
		colorID, _ := cmd.Flags().GetString("color-id")
		patch.ColorId = colorID
	}
	if cmd.Flags().Changed("hidden") {
		hidden, _ := cmd.Flags().GetBool("hidden")
		patch.Hidden = hidden
		if !hidden {
			patch.ForceSendFields = append(patch.ForceSendFields, "Hidden")
		}
	}
	if cmd.Flags().Changed("summary-override") {
		override, _ := cmd.Flags().GetString("summary-override")
		patch.SummaryOverride = override
	}

	updated, err := svc.CalendarList.Patch(calID, patch).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update subscription: %w", err))
	}

	result := map[string]interface{}{
		"status":  "updated",
		"id":      updated.Id,
		"summary": updated.Summary,
	}
	if updated.SummaryOverride != "" {
		result["summary_override"] = updated.SummaryOverride
	}
	if updated.ColorId != "" {
		result["color_id"] = updated.ColorId
	}
	if updated.Hidden {
		result["hidden"] = true
	}

	return p.Print(result)
}

// --- ACL implementations ---

func runCalendarAcl(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")

	resp, err := svc.Acl.List(calendarID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list ACL rules: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Items))
	for _, rule := range resp.Items {
		entry := map[string]interface{}{
			"id":   rule.Id,
			"role": rule.Role,
		}
		if rule.Scope != nil {
			entry["scope_type"] = rule.Scope.Type
			if rule.Scope.Value != "" {
				entry["scope_value"] = rule.Scope.Value
			}
		}
		results = append(results, entry)
	}

	return p.Print(map[string]interface{}{
		"rules": results,
		"count": len(results),
	})
}

func runCalendarShare(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	email, _ := cmd.Flags().GetString("email")
	role, _ := cmd.Flags().GetString("role")

	if !validAclRoles[role] {
		return p.PrintError(fmt.Errorf("invalid role '%s': must be reader, writer, owner, or freeBusyReader", role))
	}

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	rule := &calendar.AclRule{
		Role: role,
		Scope: &calendar.AclRuleScope{
			Type:  "user",
			Value: email,
		},
	}

	created, err := svc.Acl.Insert(calendarID, rule).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to share calendar: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status": "shared",
		"id":     created.Id,
		"role":   created.Role,
		"email":  email,
	})
}

func runCalendarUnshare(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	ruleID, _ := cmd.Flags().GetString("rule-id")

	err = svc.Acl.Delete(calendarID, ruleID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to remove access: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":  "unshared",
		"rule_id": ruleID,
	})
}

func runCalendarUpdateAcl(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	calendarID, _ := cmd.Flags().GetString("calendar-id")
	ruleID, _ := cmd.Flags().GetString("rule-id")
	role, _ := cmd.Flags().GetString("role")

	if !validAclRoles[role] {
		return p.PrintError(fmt.Errorf("invalid role '%s': must be reader, writer, owner, or freeBusyReader", role))
	}

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	// Get existing rule to preserve scope
	existing, err := svc.Acl.Get(calendarID, ruleID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get ACL rule: %w", err))
	}

	existing.Role = role

	updated, err := svc.Acl.Update(calendarID, ruleID, existing).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update ACL rule: %w", err))
	}

	result := map[string]interface{}{
		"status": "updated",
		"id":     updated.Id,
		"role":   updated.Role,
	}
	if updated.Scope != nil && updated.Scope.Value != "" {
		result["scope_value"] = updated.Scope.Value
	}

	return p.Print(result)
}

// --- Other implementations ---

func runCalendarFreebusy(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")
	calendarsStr, _ := cmd.Flags().GetString("calendars")

	fromTime, err := parseTime(fromStr)
	if err != nil {
		return p.PrintError(fmt.Errorf("invalid --from time: %w", err))
	}
	toTime, err := parseTime(toStr)
	if err != nil {
		return p.PrintError(fmt.Errorf("invalid --to time: %w", err))
	}

	calIDs := strings.Split(calendarsStr, ",")
	items := make([]*calendar.FreeBusyRequestItem, len(calIDs))
	for i, id := range calIDs {
		items[i] = &calendar.FreeBusyRequestItem{Id: strings.TrimSpace(id)}
	}

	req := &calendar.FreeBusyRequest{
		TimeMin: fromTime.Format(time.RFC3339),
		TimeMax: toTime.Format(time.RFC3339),
		Items:   items,
	}

	resp, err := svc.Freebusy.Query(req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to query free/busy: %w", err))
	}

	calendars := map[string]interface{}{}
	for calID, fb := range resp.Calendars {
		busy := make([]map[string]interface{}, 0, len(fb.Busy))
		for _, period := range fb.Busy {
			busy = append(busy, map[string]interface{}{
				"start": period.Start,
				"end":   period.End,
			})
		}
		entry := map[string]interface{}{
			"busy": busy,
		}
		if len(fb.Errors) > 0 {
			errors := make([]string, 0, len(fb.Errors))
			for _, e := range fb.Errors {
				errors = append(errors, e.Reason)
			}
			entry["errors"] = errors
		}
		calendars[calID] = entry
	}

	return p.Print(map[string]interface{}{
		"time_min":  resp.TimeMin,
		"time_max":  resp.TimeMax,
		"calendars": calendars,
	})
}

func runCalendarColors(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	colors, err := svc.Colors.Get().Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get colors: %w", err))
	}

	calendarColors := map[string]interface{}{}
	for id, c := range colors.Calendar {
		calendarColors[id] = map[string]interface{}{
			"background": c.Background,
			"foreground": c.Foreground,
		}
	}

	eventColors := map[string]interface{}{}
	for id, c := range colors.Event {
		eventColors[id] = map[string]interface{}{
			"background": c.Background,
			"foreground": c.Foreground,
		}
	}

	return p.Print(map[string]interface{}{
		"calendar_colors": calendarColors,
		"event_colors":    eventColors,
	})
}

func runCalendarSettings(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Calendar()
	if err != nil {
		return p.PrintError(err)
	}

	resp, err := svc.Settings.List().Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list settings: %w", err))
	}

	settings := map[string]interface{}{}
	for _, s := range resp.Items {
		settings[s.Id] = s.Value
	}

	return p.Print(map[string]interface{}{
		"settings": settings,
		"count":    len(settings),
	})
}

// mapEventToOutput converts a Google Calendar event into a map for JSON output.
// Fields are omitted when empty/nil to keep output clean.
func mapEventToOutput(event *calendar.Event) map[string]interface{} {
	eventInfo := map[string]interface{}{
		"id":      event.Id,
		"summary": event.Summary,
		"status":  event.Status,
	}

	// Time fields
	if event.Start != nil {
		if event.Start.DateTime != "" {
			eventInfo["start"] = event.Start.DateTime
		} else {
			eventInfo["start"] = event.Start.Date
			eventInfo["all_day"] = true
		}
	}
	if event.End != nil {
		if event.End.DateTime != "" {
			eventInfo["end"] = event.End.DateTime
		} else {
			eventInfo["end"] = event.End.Date
		}
	}

	// String fields (omit if empty)
	if event.Description != "" {
		eventInfo["description"] = event.Description
	}
	if event.Location != "" {
		eventInfo["location"] = event.Location
	}
	if event.HangoutLink != "" {
		eventInfo["hangout_link"] = event.HangoutLink
	}
	if event.HtmlLink != "" {
		eventInfo["html_link"] = event.HtmlLink
	}
	if event.Created != "" {
		eventInfo["created"] = event.Created
	}
	if event.Updated != "" {
		eventInfo["updated"] = event.Updated
	}
	if event.ColorId != "" {
		eventInfo["color_id"] = event.ColorId
	}
	if event.Visibility != "" {
		eventInfo["visibility"] = event.Visibility
	}
	if event.Transparency != "" {
		eventInfo["transparency"] = event.Transparency
	}
	if event.EventType != "" {
		eventInfo["event_type"] = event.EventType
	}

	// People
	if event.Organizer != nil && event.Organizer.Email != "" {
		eventInfo["organizer"] = event.Organizer.Email
	}
	if event.Creator != nil && event.Creator.Email != "" {
		eventInfo["creator"] = event.Creator.Email
	}

	// Self RSVP status
	for _, attendee := range event.Attendees {
		if attendee != nil && attendee.Self {
			eventInfo["response_status"] = attendee.ResponseStatus
			break
		}
	}

	// Full attendee list
	if len(event.Attendees) > 0 {
		attendees := make([]map[string]interface{}, 0, len(event.Attendees))
		for _, a := range event.Attendees {
			if a == nil || a.Email == "" {
				continue
			}
			entry := map[string]interface{}{
				"email":           a.Email,
				"response_status": a.ResponseStatus,
			}
			if a.Optional {
				entry["optional"] = true
			}
			if a.Organizer {
				entry["organizer"] = true
			}
			if a.Self {
				entry["self"] = true
			}
			attendees = append(attendees, entry)
		}
		if len(attendees) > 0 {
			eventInfo["attendees"] = attendees
		}
	}

	// Conference data
	if event.ConferenceData != nil {
		conf := map[string]interface{}{}
		if event.ConferenceData.ConferenceId != "" {
			conf["conference_id"] = event.ConferenceData.ConferenceId
		}
		if event.ConferenceData.ConferenceSolution != nil && event.ConferenceData.ConferenceSolution.Name != "" {
			conf["solution"] = event.ConferenceData.ConferenceSolution.Name
		}
		if len(event.ConferenceData.EntryPoints) > 0 {
			eps := make([]map[string]interface{}, 0, len(event.ConferenceData.EntryPoints))
			for _, ep := range event.ConferenceData.EntryPoints {
				if ep == nil {
					continue
				}
				entry := map[string]interface{}{}
				if ep.EntryPointType != "" {
					entry["type"] = ep.EntryPointType
				}
				if ep.Uri != "" {
					entry["uri"] = ep.Uri
				}
				if len(entry) > 0 {
					eps = append(eps, entry)
				}
			}
			if len(eps) > 0 {
				conf["entry_points"] = eps
			}
		}
		if len(conf) > 0 {
			eventInfo["conference"] = conf
		}
	}

	// Attachments
	if len(event.Attachments) > 0 {
		attachments := make([]map[string]interface{}, 0, len(event.Attachments))
		for _, att := range event.Attachments {
			if att == nil {
				continue
			}
			entry := map[string]interface{}{}
			if att.FileUrl != "" {
				entry["file_url"] = att.FileUrl
			}
			if att.Title != "" {
				entry["title"] = att.Title
			}
			if att.MimeType != "" {
				entry["mime_type"] = att.MimeType
			}
			if att.FileId != "" {
				entry["file_id"] = att.FileId
			}
			if len(entry) > 0 {
				attachments = append(attachments, entry)
			}
		}
		if len(attachments) > 0 {
			eventInfo["attachments"] = attachments
		}
	}

	// Recurrence
	if len(event.Recurrence) > 0 {
		eventInfo["recurrence"] = event.Recurrence
	}

	// Reminders
	if event.Reminders != nil {
		reminders := map[string]interface{}{
			"use_default": event.Reminders.UseDefault,
		}
		if len(event.Reminders.Overrides) > 0 {
			overrides := make([]map[string]interface{}, 0, len(event.Reminders.Overrides))
			for _, o := range event.Reminders.Overrides {
				if o == nil || o.Method == "" {
					continue
				}
				overrides = append(overrides, map[string]interface{}{
					"method":  o.Method,
					"minutes": o.Minutes,
				})
			}
			if len(overrides) > 0 {
				reminders["overrides"] = overrides
			}
		}
		eventInfo["reminders"] = reminders
	}

	return eventInfo
}

// parseTime parses a time string in RFC3339 or "YYYY-MM-DD HH:MM" format.
func parseTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try common format
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local); err == nil {
		return t, nil
	}

	// Try date only (all-day)
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %s (use RFC3339 or 'YYYY-MM-DD HH:MM')", s)
}

// resolveIANA returns the IANA timezone name for a time.Time.
// If the location is "Local", it attempts to resolve the real IANA name
// from the TZ env var or /etc/localtime symlink. Returns "" as fallback,
// which omits the TimeZone field and lets the RFC3339 offset suffice.
func resolveIANA(t time.Time) string {
	name := t.Location().String()
	if name != "Local" {
		return name
	}
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}
	if runtime.GOOS != "windows" {
		if target, err := os.Readlink("/etc/localtime"); err == nil {
			if idx := strings.Index(target, "zoneinfo/"); idx != -1 {
				return target[idx+len("zoneinfo/"):]
			}
		}
	}
	return ""
}
