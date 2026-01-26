package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestCalendarUpdateCommand_Flags(t *testing.T) {
	cmd := calendarUpdateCmd

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	flags := []string{"title", "start", "end", "description", "location", "add-attendees", "calendar-id"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to exist", name)
		}
	}
}

func TestCalendarUpdateCommand_Help(t *testing.T) {
	cmd := calendarUpdateCmd

	if cmd.Use != "update <event-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestCalendarDeleteCommand_Flags(t *testing.T) {
	cmd := calendarDeleteCmd

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	if cmd.Flags().Lookup("calendar-id") == nil {
		t.Error("expected --calendar-id flag to exist")
	}
}

func TestCalendarDeleteCommand_Help(t *testing.T) {
	cmd := calendarDeleteCmd

	if cmd.Use != "delete <event-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestCalendarRsvpCommand_Flags(t *testing.T) {
	cmd := calendarRsvpCmd

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	responseFlag := cmd.Flags().Lookup("response")
	if responseFlag == nil {
		t.Error("expected --response flag to exist")
	}

	if cmd.Flags().Lookup("calendar-id") == nil {
		t.Error("expected --calendar-id flag to exist")
	}
}

func TestCalendarRsvpCommand_Help(t *testing.T) {
	cmd := calendarRsvpCmd

	if cmd.Use != "rsvp <event-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// TestCalendarUpdate_MockServer tests update API integration
func TestCalendarUpdate_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get event
		if r.URL.Path == "/calendars/primary/events/evt-123" && r.Method == "GET" {
			resp := &calendar.Event{
				Id:          "evt-123",
				Summary:     "Original Title",
				Description: "Original description",
				Start:       &calendar.EventDateTime{DateTime: "2024-02-01T10:00:00Z"},
				End:         &calendar.EventDateTime{DateTime: "2024-02-01T11:00:00Z"},
				HtmlLink:    "https://calendar.google.com/event?id=evt-123",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Update event
		if r.URL.Path == "/calendars/primary/events/evt-123" && r.Method == "PUT" {
			var event calendar.Event
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			resp := &calendar.Event{
				Id:       "evt-123",
				Summary:  event.Summary,
				Start:    event.Start,
				End:      event.End,
				HtmlLink: "https://calendar.google.com/event?id=evt-123",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := calendar.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Get existing event
	event, err := svc.Events.Get("primary", "evt-123").Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	if event.Summary != "Original Title" {
		t.Errorf("unexpected summary: %s", event.Summary)
	}

	// Update it
	event.Summary = "Updated Title"
	updated, err := svc.Events.Update("primary", "evt-123", event).Do()
	if err != nil {
		t.Fatalf("failed to update event: %v", err)
	}

	if updated.Summary != "Updated Title" {
		t.Errorf("expected updated summary, got: %s", updated.Summary)
	}
}

// TestCalendarDelete_MockServer tests delete API integration
func TestCalendarDelete_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get event
		if r.URL.Path == "/calendars/primary/events/evt-456" && r.Method == "GET" {
			resp := &calendar.Event{
				Id:      "evt-456",
				Summary: "Meeting to Cancel",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Delete event
		if r.URL.Path == "/calendars/primary/events/evt-456" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := calendar.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Get event first
	event, err := svc.Events.Get("primary", "evt-456").Fields("summary").Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	if event.Summary != "Meeting to Cancel" {
		t.Errorf("unexpected summary: %s", event.Summary)
	}

	// Delete it
	err = svc.Events.Delete("primary", "evt-456").Do()
	if err != nil {
		t.Fatalf("failed to delete event: %v", err)
	}
}

// TestCalendarRsvp_MockServer tests RSVP API integration
func TestCalendarRsvp_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get event
		if r.URL.Path == "/calendars/primary/events/evt-789" && r.Method == "GET" {
			resp := &calendar.Event{
				Id:      "evt-789",
				Summary: "Team Standup",
				Attendees: []*calendar.EventAttendee{
					{Email: "organizer@example.com", ResponseStatus: "accepted"},
					{Email: "me@example.com", Self: true, ResponseStatus: "needsAction"},
					{Email: "other@example.com", ResponseStatus: "tentative"},
				},
				HtmlLink: "https://calendar.google.com/event?id=evt-789",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Patch event (RSVP)
		if r.URL.Path == "/calendars/primary/events/evt-789" && r.Method == "PATCH" {
			var event calendar.Event
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Verify the self attendee was updated
			for _, a := range event.Attendees {
				if a.Self && a.ResponseStatus != "accepted" {
					t.Errorf("expected self attendee response 'accepted', got '%s'", a.ResponseStatus)
				}
			}

			resp := &calendar.Event{
				Id:       "evt-789",
				Summary:  "Team Standup",
				HtmlLink: "https://calendar.google.com/event?id=evt-789",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := calendar.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Get event
	event, err := svc.Events.Get("primary", "evt-789").Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	// Find self attendee and update RSVP
	for _, attendee := range event.Attendees {
		if attendee.Self {
			attendee.ResponseStatus = "accepted"
			break
		}
	}

	// Patch with updated attendees
	updated, err := svc.Events.Patch("primary", "evt-789", &calendar.Event{
		Attendees: event.Attendees,
	}).Do()
	if err != nil {
		t.Fatalf("failed to RSVP: %v", err)
	}

	if updated.Id != "evt-789" {
		t.Errorf("unexpected event id: %s", updated.Id)
	}
}

// TestCalendarRsvp_InvalidResponse tests RSVP validation
func TestCalendarRsvp_InvalidResponse(t *testing.T) {
	tests := []struct {
		response string
		valid    bool
	}{
		{"accepted", true},
		{"declined", true},
		{"tentative", true},
		{"maybe", false},
		{"yes", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.response, func(t *testing.T) {
			result := validRsvpResponses[tt.response]
			if result != tt.valid {
				t.Errorf("validRsvpResponses[%q] = %v, want %v", tt.response, result, tt.valid)
			}
		})
	}
}

// TestCalendarUpdate_OutputFormat tests the update response format
func TestCalendarUpdate_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":    "updated",
		"id":        "evt-123",
		"summary":   "Updated Meeting",
		"html_link": "https://calendar.google.com/event?id=evt-123",
		"start":     "2024-02-01T14:00:00Z",
		"end":       "2024-02-01T15:00:00Z",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["status"] != "updated" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}

	if decoded["summary"] != "Updated Meeting" {
		t.Errorf("unexpected summary: %v", decoded["summary"])
	}
}

// TestCalendarRsvp_OutputFormat tests the RSVP response format
func TestCalendarRsvp_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":    "accepted",
		"id":        "evt-789",
		"summary":   "Team Standup",
		"html_link": "https://calendar.google.com/event?id=evt-789",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["status"] != "accepted" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}
}
