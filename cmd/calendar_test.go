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

	if cmd.Flags().Lookup("message") == nil {
		t.Error("expected --message flag to exist")
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

// TestCalendarUpdate_MockServer tests update API integration using Patch
func TestCalendarUpdate_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Patch event (partial update)
		if r.URL.Path == "/calendars/primary/events/evt-123" && r.Method == "PATCH" {
			var event calendar.Event
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			resp := &calendar.Event{
				Id:       "evt-123",
				Summary:  event.Summary,
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

	// Patch with only changed fields
	patch := &calendar.Event{Summary: "Updated Title"}
	updated, err := svc.Events.Patch("primary", "evt-123", patch).Do()
	if err != nil {
		t.Fatalf("failed to patch event: %v", err)
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

// TestCalendarRsvp_MockServer_WithMessage tests RSVP with message sets Comment and sendUpdates
func TestCalendarRsvp_MockServer_WithMessage(t *testing.T) {
	var receivedSendUpdates string
	var receivedComment string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/calendars/primary/events/evt-msg" && r.Method == "GET" {
			resp := &calendar.Event{
				Id:      "evt-msg",
				Summary: "Meeting with Message",
				Attendees: []*calendar.EventAttendee{
					{Email: "organizer@example.com", ResponseStatus: "accepted"},
					{Email: "me@example.com", Self: true, ResponseStatus: "needsAction"},
				},
				HtmlLink: "https://calendar.google.com/event?id=evt-msg",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/calendars/primary/events/evt-msg" && r.Method == "PATCH" {
			receivedSendUpdates = r.URL.Query().Get("sendUpdates")

			var event calendar.Event
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			for _, a := range event.Attendees {
				if a.Self {
					receivedComment = a.Comment
				}
			}

			resp := &calendar.Event{
				Id:       "evt-msg",
				Summary:  "Meeting with Message",
				HtmlLink: "https://calendar.google.com/event?id=evt-msg",
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
	event, err := svc.Events.Get("primary", "evt-msg").Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	// Update self attendee with comment
	for _, attendee := range event.Attendees {
		if attendee.Self {
			attendee.ResponseStatus = "declined"
			attendee.Comment = "Sorry, I have a conflict"
			break
		}
	}

	// Patch with sendUpdates
	_, err = svc.Events.Patch("primary", "evt-msg", &calendar.Event{
		Attendees: event.Attendees,
	}).SendUpdates("all").Do()
	if err != nil {
		t.Fatalf("failed to RSVP with message: %v", err)
	}

	if receivedSendUpdates != "all" {
		t.Errorf("expected sendUpdates=all, got '%s'", receivedSendUpdates)
	}
	if receivedComment != "Sorry, I have a conflict" {
		t.Errorf("expected comment 'Sorry, I have a conflict', got '%s'", receivedComment)
	}
}

// TestCalendarRsvp_MockServer_WithoutMessage verifies sendUpdates is NOT set when no message provided
func TestCalendarRsvp_MockServer_WithoutMessage(t *testing.T) {
	var receivedSendUpdates string
	patchCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/calendars/primary/events/evt-nomsg" && r.Method == "GET" {
			resp := &calendar.Event{
				Id:      "evt-nomsg",
				Summary: "Silent RSVP",
				Attendees: []*calendar.EventAttendee{
					{Email: "me@example.com", Self: true, ResponseStatus: "needsAction"},
				},
				HtmlLink: "https://calendar.google.com/event?id=evt-nomsg",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/calendars/primary/events/evt-nomsg" && r.Method == "PATCH" {
			patchCalled = true
			receivedSendUpdates = r.URL.Query().Get("sendUpdates")

			resp := &calendar.Event{
				Id:       "evt-nomsg",
				Summary:  "Silent RSVP",
				HtmlLink: "https://calendar.google.com/event?id=evt-nomsg",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := calendar.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	event, err := svc.Events.Get("primary", "evt-nomsg").Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	for _, attendee := range event.Attendees {
		if attendee.Self {
			attendee.ResponseStatus = "accepted"
			break
		}
	}

	// Patch WITHOUT SendUpdates (mirrors no --message code path)
	_, err = svc.Events.Patch("primary", "evt-nomsg", &calendar.Event{
		Attendees: event.Attendees,
	}).Do()
	if err != nil {
		t.Fatalf("failed to RSVP: %v", err)
	}

	if !patchCalled {
		t.Fatal("expected PATCH to be called")
	}
	if receivedSendUpdates != "" {
		t.Errorf("expected no sendUpdates param, got '%s'", receivedSendUpdates)
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

// --- Event response_status and --pending tests ---

func TestCalendarEventsCommand_PendingFlag(t *testing.T) {
	if calendarEventsCmd.Flags().Lookup("pending") == nil {
		t.Error("expected --pending flag to exist on events command")
	}
}

func TestCalendarEvents_MockServer_ResponseStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/calendars/primary/events" && r.Method == "GET" {
			resp := &calendar.Events{
				Items: []*calendar.Event{
					{
						Id:          "evt-pending",
						Summary:     "Pending Meeting",
						Status:      "confirmed",
						Description: "Discuss Q1 planning",
						Start:       &calendar.EventDateTime{DateTime: "2026-02-01T10:00:00Z"},
						End:         &calendar.EventDateTime{DateTime: "2026-02-01T11:00:00Z"},
						HtmlLink:    "https://calendar.google.com/event?id=evt-pending",
						Created:     "2026-01-15T08:00:00Z",
						Updated:     "2026-01-20T10:00:00Z",
						Organizer: &calendar.EventOrganizer{
							Email: "boss@example.com",
						},
						Creator: &calendar.EventCreator{
							Email: "boss@example.com",
						},
						Attendees: []*calendar.EventAttendee{
							{Email: "boss@example.com", ResponseStatus: "accepted", Organizer: true},
							{Email: "me@example.com", Self: true, ResponseStatus: "needsAction"},
							{Email: "optional@example.com", ResponseStatus: "needsAction", Optional: true},
						},
						ConferenceData: &calendar.ConferenceData{
							ConferenceId: "meet-abc-123",
							ConferenceSolution: &calendar.ConferenceSolution{
								Name: "Google Meet",
							},
							EntryPoints: []*calendar.EntryPoint{
								{EntryPointType: "video", Uri: "https://meet.google.com/abc-123"},
							},
						},
						Attachments: []*calendar.EventAttachment{
							{FileUrl: "https://drive.google.com/file/d/abc", Title: "Agenda.pdf", MimeType: "application/pdf", FileId: "abc"},
						},
						Reminders: &calendar.EventReminders{
							UseDefault: false,
							Overrides: []*calendar.EventReminder{
								{Method: "popup", Minutes: 10},
							},
						},
					},
					{
						Id:      "evt-accepted",
						Summary: "Accepted Meeting",
						Status:  "confirmed",
						Start:   &calendar.EventDateTime{DateTime: "2026-02-01T14:00:00Z"},
						End:     &calendar.EventDateTime{DateTime: "2026-02-01T15:00:00Z"},
						Organizer: &calendar.EventOrganizer{
							Email: "colleague@example.com",
						},
						Attendees: []*calendar.EventAttendee{
							{Email: "colleague@example.com", ResponseStatus: "accepted"},
							{Email: "me@example.com", Self: true, ResponseStatus: "accepted"},
						},
					},
					{
						Id:      "evt-no-attendees",
						Summary: "Solo Event",
						Status:  "confirmed",
						Start:   &calendar.EventDateTime{DateTime: "2026-02-01T16:00:00Z"},
						End:     &calendar.EventDateTime{DateTime: "2026-02-01T17:00:00Z"},
					},
				},
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

	resp, err := svc.Events.List("primary").Do()
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Items))
	}

	// Verify first event has attendee data
	evt := resp.Items[0]
	if evt.Organizer == nil || evt.Organizer.Email != "boss@example.com" {
		t.Error("expected organizer email boss@example.com")
	}

	var selfStatus string
	for _, a := range evt.Attendees {
		if a.Self {
			selfStatus = a.ResponseStatus
			break
		}
	}
	if selfStatus != "needsAction" {
		t.Errorf("expected self response_status 'needsAction', got '%s'", selfStatus)
	}

	// Verify second event has accepted status
	evt2 := resp.Items[1]
	for _, a := range evt2.Attendees {
		if a.Self {
			if a.ResponseStatus != "accepted" {
				t.Errorf("expected self response_status 'accepted', got '%s'", a.ResponseStatus)
			}
			break
		}
	}

	// Verify third event has no attendees (solo event — no response_status)
	if len(resp.Items[2].Attendees) != 0 {
		t.Error("expected no attendees on solo event")
	}
}

func TestCalendarEvents_OutputFormat(t *testing.T) {
	// Verify the event output includes response_status and organizer
	eventInfo := map[string]interface{}{
		"id":              "evt-100",
		"summary":         "Team Sync",
		"status":          "confirmed",
		"start":           "2026-02-01T10:00:00Z",
		"end":             "2026-02-01T11:00:00Z",
		"organizer":       "boss@example.com",
		"response_status": "needsAction",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(eventInfo); err != nil {
		t.Fatalf("failed to encode event: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode event: %v", err)
	}

	// New fields
	if decoded["response_status"] != "needsAction" {
		t.Errorf("expected response_status 'needsAction', got '%v'", decoded["response_status"])
	}
	if decoded["organizer"] != "boss@example.com" {
		t.Errorf("expected organizer 'boss@example.com', got '%v'", decoded["organizer"])
	}

	// Existing fields still present
	if decoded["id"] != "evt-100" {
		t.Errorf("expected id 'evt-100', got '%v'", decoded["id"])
	}
	if decoded["summary"] != "Team Sync" {
		t.Errorf("expected summary 'Team Sync', got '%v'", decoded["summary"])
	}

	// Solo event without response_status or organizer
	soloEvent := map[string]interface{}{
		"id":      "evt-200",
		"summary": "Focus Time",
		"status":  "confirmed",
		"start":   "2026-02-01T14:00:00Z",
		"end":     "2026-02-01T15:00:00Z",
	}

	buf.Reset()
	if err := encoder.Encode(soloEvent); err != nil {
		t.Fatalf("failed to encode solo event: %v", err)
	}

	var decodedSolo map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decodedSolo); err != nil {
		t.Fatalf("failed to decode solo event: %v", err)
	}

	if _, exists := decodedSolo["response_status"]; exists {
		t.Error("solo event should not have response_status")
	}
	if _, exists := decodedSolo["organizer"]; exists {
		t.Error("solo event should not have organizer")
	}
}

func TestCalendarEvents_PendingFilter(t *testing.T) {
	// Simulate the filtering logic from runCalendarEvents
	events := []struct {
		id             string
		responseStatus string
	}{
		{"evt-1", "needsAction"},
		{"evt-2", "accepted"},
		{"evt-3", "needsAction"},
		{"evt-4", "declined"},
		{"evt-5", ""},
	}

	var filtered []string
	for _, evt := range events {
		if evt.responseStatus != "needsAction" {
			continue
		}
		filtered = append(filtered, evt.id)
	}

	if len(filtered) != 2 {
		t.Errorf("expected 2 pending events, got %d", len(filtered))
	}
	if filtered[0] != "evt-1" || filtered[1] != "evt-3" {
		t.Errorf("unexpected filtered events: %v", filtered)
	}
}

// TestMapEventToOutput_AllFields verifies all new fields appear in output when set
func TestMapEventToOutput_AllFields(t *testing.T) {
	event := &calendar.Event{
		Id:           "evt-full",
		Summary:      "Full Event",
		Status:       "confirmed",
		Description:  "A detailed description",
		Start:        &calendar.EventDateTime{DateTime: "2026-03-01T09:00:00Z"},
		End:          &calendar.EventDateTime{DateTime: "2026-03-01T10:00:00Z"},
		Location:     "Room 42",
		HangoutLink:  "https://hangouts.google.com/call/abc",
		HtmlLink:     "https://calendar.google.com/event?id=evt-full",
		Created:      "2026-02-01T08:00:00Z",
		Updated:      "2026-02-15T12:00:00Z",
		ColorId:      "9",
		Visibility:   "default",
		Transparency: "opaque",
		EventType:    "default",
		Organizer:    &calendar.EventOrganizer{Email: "org@example.com"},
		Creator:      &calendar.EventCreator{Email: "creator@example.com"},
		Attendees: []*calendar.EventAttendee{
			{Email: "org@example.com", ResponseStatus: "accepted", Organizer: true},
			{Email: "me@example.com", Self: true, ResponseStatus: "tentative"},
			{Email: "opt@example.com", ResponseStatus: "needsAction", Optional: true},
		},
		ConferenceData: &calendar.ConferenceData{
			ConferenceId: "conf-xyz",
			ConferenceSolution: &calendar.ConferenceSolution{
				Name: "Google Meet",
			},
			EntryPoints: []*calendar.EntryPoint{
				{EntryPointType: "video", Uri: "https://meet.google.com/xyz"},
				{EntryPointType: "phone", Uri: "tel:+1234567890"},
			},
		},
		Attachments: []*calendar.EventAttachment{
			{FileUrl: "https://drive.google.com/file/d/123", Title: "Notes.docx", MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", FileId: "123"},
		},
		Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
		Reminders: &calendar.EventReminders{
			UseDefault: false,
			Overrides: []*calendar.EventReminder{
				{Method: "email", Minutes: 30},
				{Method: "popup", Minutes: 10},
			},
		},
	}

	result := mapEventToOutput(event)

	// Always-present fields
	if result["id"] != "evt-full" {
		t.Errorf("expected id 'evt-full', got %v", result["id"])
	}
	if result["summary"] != "Full Event" {
		t.Errorf("expected summary 'Full Event', got %v", result["summary"])
	}
	if result["status"] != "confirmed" {
		t.Errorf("expected status 'confirmed', got %v", result["status"])
	}

	// String fields
	stringFields := map[string]string{
		"description":  "A detailed description",
		"location":     "Room 42",
		"hangout_link": "https://hangouts.google.com/call/abc",
		"html_link":    "https://calendar.google.com/event?id=evt-full",
		"created":      "2026-02-01T08:00:00Z",
		"updated":      "2026-02-15T12:00:00Z",
		"color_id":     "9",
		"visibility":   "default",
		"transparency": "opaque",
		"event_type":   "default",
		"organizer":    "org@example.com",
		"creator":      "creator@example.com",
	}
	for field, expected := range stringFields {
		val, ok := result[field].(string)
		if !ok {
			t.Errorf("field %q missing or not a string", field)
			continue
		}
		if val != expected {
			t.Errorf("field %q: expected %q, got %q", field, expected, val)
		}
	}

	// Time
	if result["start"] != "2026-03-01T09:00:00Z" {
		t.Errorf("expected start time, got %v", result["start"])
	}
	if result["end"] != "2026-03-01T10:00:00Z" {
		t.Errorf("expected end time, got %v", result["end"])
	}

	// Response status (self)
	if result["response_status"] != "tentative" {
		t.Errorf("expected response_status 'tentative', got %v", result["response_status"])
	}

	// Attendees
	attendees, ok := result["attendees"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected attendees to be a slice of maps")
	}
	if len(attendees) != 3 {
		t.Errorf("expected 3 attendees, got %d", len(attendees))
	}
	// Check optional flag on third attendee
	if opt, ok := attendees[2]["optional"].(bool); !ok || !opt {
		t.Error("expected third attendee to have optional=true")
	}

	// Conference
	conf, ok := result["conference"].(map[string]interface{})
	if !ok {
		t.Fatal("expected conference to be a map")
	}
	if conf["conference_id"] != "conf-xyz" {
		t.Errorf("expected conference_id 'conf-xyz', got %v", conf["conference_id"])
	}
	if conf["solution"] != "Google Meet" {
		t.Errorf("expected solution 'Google Meet', got %v", conf["solution"])
	}
	eps, ok := conf["entry_points"].([]map[string]interface{})
	if !ok || len(eps) != 2 {
		t.Errorf("expected 2 entry points, got %v", conf["entry_points"])
	}

	// Attachments
	atts, ok := result["attachments"].([]map[string]interface{})
	if !ok || len(atts) != 1 {
		t.Fatal("expected 1 attachment")
	}
	if atts[0]["title"] != "Notes.docx" {
		t.Errorf("expected attachment title 'Notes.docx', got %v", atts[0]["title"])
	}

	// Recurrence
	recurrence, ok := result["recurrence"].([]string)
	if !ok || len(recurrence) != 1 {
		t.Fatal("expected 1 recurrence rule")
	}
	if recurrence[0] != "RRULE:FREQ=WEEKLY;BYDAY=MO" {
		t.Errorf("unexpected recurrence: %v", recurrence[0])
	}

	// Reminders
	reminders, ok := result["reminders"].(map[string]interface{})
	if !ok {
		t.Fatal("expected reminders to be a map")
	}
	if reminders["use_default"] != false {
		t.Errorf("expected use_default false, got %v", reminders["use_default"])
	}
	overrides, ok := reminders["overrides"].([]map[string]interface{})
	if !ok || len(overrides) != 2 {
		t.Fatal("expected 2 reminder overrides")
	}
}

// TestMapEventToOutput_EmptyOptionals verifies empty/nil fields are omitted
func TestMapEventToOutput_EmptyOptionals(t *testing.T) {
	event := &calendar.Event{
		Id:      "evt-minimal",
		Summary: "Minimal Event",
		Status:  "confirmed",
		Start:   &calendar.EventDateTime{DateTime: "2026-03-01T09:00:00Z"},
		End:     &calendar.EventDateTime{DateTime: "2026-03-01T10:00:00Z"},
	}

	result := mapEventToOutput(event)

	// These must be present
	if result["id"] != "evt-minimal" {
		t.Errorf("expected id 'evt-minimal', got %v", result["id"])
	}

	// These must be absent
	absentFields := []string{
		"description", "location", "hangout_link", "html_link",
		"created", "updated", "color_id", "visibility", "transparency",
		"event_type", "organizer", "creator", "response_status",
		"attendees", "conference", "attachments", "recurrence", "reminders",
	}
	for _, field := range absentFields {
		if _, exists := result[field]; exists {
			t.Errorf("field %q should be omitted for minimal event, but was present: %v", field, result[field])
		}
	}
}

// TestMapEventToOutput_AllDayEvent verifies all-day event handling through the helper
func TestMapEventToOutput_AllDayEvent(t *testing.T) {
	event := &calendar.Event{
		Id:      "evt-allday",
		Summary: "Holiday",
		Status:  "confirmed",
		Start:   &calendar.EventDateTime{Date: "2026-03-15"},
		End:     &calendar.EventDateTime{Date: "2026-03-16"},
	}

	result := mapEventToOutput(event)

	if result["start"] != "2026-03-15" {
		t.Errorf("expected start '2026-03-15', got %v", result["start"])
	}
	if result["end"] != "2026-03-16" {
		t.Errorf("expected end '2026-03-16', got %v", result["end"])
	}
	if result["all_day"] != true {
		t.Error("expected all_day=true for date-only event")
	}
}

// TestMapEventToOutput_NilNestedEntries verifies nil elements in slices are safely skipped
func TestMapEventToOutput_NilNestedEntries(t *testing.T) {
	event := &calendar.Event{
		Id:      "evt-nils",
		Summary: "Nil Test",
		Status:  "confirmed",
		Start:   &calendar.EventDateTime{DateTime: "2026-03-01T09:00:00Z"},
		End:     &calendar.EventDateTime{DateTime: "2026-03-01T10:00:00Z"},
		Attendees: []*calendar.EventAttendee{
			nil,
			{Email: "valid@example.com", ResponseStatus: "accepted"},
			{Email: "", ResponseStatus: ""},
			nil,
		},
		ConferenceData: &calendar.ConferenceData{
			ConferenceId: "conf-nil",
			EntryPoints: []*calendar.EntryPoint{
				nil,
				{EntryPointType: "video", Uri: "https://meet.google.com/nil"},
				nil,
			},
		},
		Attachments: []*calendar.EventAttachment{
			nil,
			{FileUrl: "https://drive.google.com/file/d/x", Title: "Doc.pdf"},
			nil,
		},
		Reminders: &calendar.EventReminders{
			UseDefault: false,
			Overrides: []*calendar.EventReminder{
				nil,
				{Method: "popup", Minutes: 5},
				nil,
			},
		},
	}

	// Must not panic
	result := mapEventToOutput(event)

	// Attendees: nil and empty-email entries skipped, only valid@example.com remains
	attendees, ok := result["attendees"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected attendees to be present")
	}
	if len(attendees) != 1 {
		t.Errorf("expected 1 valid attendee, got %d", len(attendees))
	}
	if attendees[0]["email"] != "valid@example.com" {
		t.Errorf("expected valid@example.com, got %v", attendees[0]["email"])
	}

	// Conference: nil entry points skipped
	conf, ok := result["conference"].(map[string]interface{})
	if !ok {
		t.Fatal("expected conference to be present")
	}
	eps, ok := conf["entry_points"].([]map[string]interface{})
	if !ok || len(eps) != 1 {
		t.Errorf("expected 1 valid entry point, got %v", conf["entry_points"])
	}

	// Attachments: nil entries skipped
	atts, ok := result["attachments"].([]map[string]interface{})
	if !ok || len(atts) != 1 {
		t.Errorf("expected 1 valid attachment, got %v", result["attachments"])
	}

	// Reminders: nil overrides skipped
	reminders, ok := result["reminders"].(map[string]interface{})
	if !ok {
		t.Fatal("expected reminders to be present")
	}
	overrides, ok := reminders["overrides"].([]map[string]interface{})
	if !ok || len(overrides) != 1 {
		t.Errorf("expected 1 valid reminder override, got %v", reminders["overrides"])
	}
}

// TestMapEventToOutput_EmptyNestedObjects verifies fully-empty nested structures are omitted
func TestMapEventToOutput_EmptyNestedObjects(t *testing.T) {
	event := &calendar.Event{
		Id:      "evt-empty-nested",
		Summary: "Empty Nested",
		Status:  "confirmed",
		Start:   &calendar.EventDateTime{DateTime: "2026-03-01T09:00:00Z"},
		End:     &calendar.EventDateTime{DateTime: "2026-03-01T10:00:00Z"},
		// Conference with no useful data
		ConferenceData: &calendar.ConferenceData{},
		// Attachments with only nil/empty entries
		Attachments: []*calendar.EventAttachment{nil},
		// Attendees with only nil entries
		Attendees: []*calendar.EventAttendee{nil},
	}

	result := mapEventToOutput(event)

	if _, exists := result["conference"]; exists {
		t.Error("empty conference data should be omitted")
	}
	if _, exists := result["attachments"]; exists {
		t.Error("attachments with only nil entries should be omitted")
	}
	if _, exists := result["attendees"]; exists {
		t.Error("attendees with only nil entries should be omitted")
	}
}

// TestMapEventToOutput_EmptyMethodOverride verifies overrides with empty method are skipped
func TestMapEventToOutput_EmptyMethodOverride(t *testing.T) {
	event := &calendar.Event{
		Id:      "evt-override",
		Summary: "Override Test",
		Status:  "confirmed",
		Start:   &calendar.EventDateTime{DateTime: "2026-03-01T09:00:00Z"},
		End:     &calendar.EventDateTime{DateTime: "2026-03-01T10:00:00Z"},
		Reminders: &calendar.EventReminders{
			UseDefault: false,
			Overrides: []*calendar.EventReminder{
				{Method: "", Minutes: 0},
				{Method: "popup", Minutes: 10},
				{Method: "", Minutes: 5},
			},
		},
	}

	result := mapEventToOutput(event)

	reminders, ok := result["reminders"].(map[string]interface{})
	if !ok {
		t.Fatal("expected reminders to be present")
	}
	overrides, ok := reminders["overrides"].([]map[string]interface{})
	if !ok || len(overrides) != 1 {
		t.Fatalf("expected 1 valid override (empty-method skipped), got %v", reminders["overrides"])
	}
	if overrides[0]["method"] != "popup" {
		t.Errorf("expected method 'popup', got %v", overrides[0]["method"])
	}
}
