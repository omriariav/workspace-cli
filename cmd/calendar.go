package cmd

import (
	"context"
	"fmt"
	"os"
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

var validRsvpResponses = map[string]bool{
	"accepted":  true,
	"declined":  true,
	"tentative": true,
}

func init() {
	rootCmd.AddCommand(calendarCmd)
	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarEventsCmd)
	calendarCmd.AddCommand(calendarCreateCmd)
	calendarCmd.AddCommand(calendarUpdateCmd)
	calendarCmd.AddCommand(calendarDeleteCmd)
	calendarCmd.AddCommand(calendarRsvpCmd)

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
	calendarRsvpCmd.Flags().String("message", "", "Optional message to include with your RSVP")
	calendarRsvpCmd.MarkFlagRequired("response")
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
		eventInfo := map[string]interface{}{
			"id":      event.Id,
			"summary": event.Summary,
			"status":  event.Status,
		}

		// Handle all-day events vs timed events
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

		if event.Location != "" {
			eventInfo["location"] = event.Location
		}
		if event.HangoutLink != "" {
			eventInfo["hangout_link"] = event.HangoutLink
		}

		if event.Organizer != nil && event.Organizer.Email != "" {
			eventInfo["organizer"] = event.Organizer.Email
		}

		for _, attendee := range event.Attendees {
			if attendee.Self {
				eventInfo["response_status"] = attendee.ResponseStatus
				break
			}
		}

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

	event := &calendar.Event{
		Summary:     title,
		Description: description,
		Location:    location,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: startTime.Location().String(),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: endTime.Location().String(),
		},
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
		patch.Start = &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: startTime.Location().String(),
		}
	}
	if cmd.Flags().Changed("end") {
		endStr, _ := cmd.Flags().GetString("end")
		endTime, err := parseTime(endStr)
		if err != nil {
			return p.PrintError(fmt.Errorf("invalid end time: %w", err))
		}
		patch.End = &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: endTime.Location().String(),
		}
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
