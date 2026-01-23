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

func init() {
	rootCmd.AddCommand(calendarCmd)
	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarEventsCmd)
	calendarCmd.AddCommand(calendarCreateCmd)

	// Events flags
	calendarEventsCmd.Flags().Int("days", 7, "Number of days to look ahead")
	calendarEventsCmd.Flags().String("calendar-id", "primary", "Calendar ID (default: primary)")
	calendarEventsCmd.Flags().Int64("max", 50, "Maximum number of events")

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
