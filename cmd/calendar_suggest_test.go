package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad test time %q: %v", s, err)
	}
	return v
}

func TestCalendarSuggestTimeCommand_Flags(t *testing.T) {
	cmd := findSubcommand(calendarCmd, "suggest-time")
	if cmd == nil {
		t.Fatal("calendar suggest-time command not found")
	}
	for _, flag := range []string{"from", "to", "duration", "calendars", "attendees", "working-hours", "timezone", "max", "include-weekends"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestParseSuggestDuration(t *testing.T) {
	good := map[string]time.Duration{"30m": 30 * time.Minute, "1h30m": 90 * time.Minute, "45": 45 * time.Minute, " 1h ": time.Hour}
	for in, want := range good {
		if got, err := parseSuggestDuration(in); err != nil || got != want {
			t.Errorf("parseSuggestDuration(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if got, err := parseSuggestDuration("1440"); err != nil || got != 24*time.Hour {
		t.Errorf("24h in minutes = %v, %v", got, err)
	}
	// Huge values must be rejected, not overflow into a negative duration.
	for _, in := range []string{"", "0", "-5", "-30m", "soon", "1441", "25h", "9223372036854775807", "153722867281", "500ms", "90s", "1m30s"} {
		if _, err := parseSuggestDuration(in); err == nil {
			t.Errorf("parseSuggestDuration(%q) should fail", in)
		}
	}
}

func TestParseWorkingHours(t *testing.T) {
	s, e, err := parseWorkingHours("09:00-17:30")
	if err != nil || s != 9*time.Hour || e != 17*time.Hour+30*time.Minute {
		t.Errorf("got %v-%v, %v", s, e, err)
	}
	for _, in := range []string{"any", "", "ANY", "00:00-24:00"} {
		if s, e, err := parseWorkingHours(in); err != nil || s != 0 || e != 24*time.Hour {
			t.Errorf("parseWorkingHours(%q) = %v-%v, %v; want whole day", in, s, e, err)
		}
	}
	for _, in := range []string{"9-5pm", "17:00-09:00", "09:00", "09:00-09:00", "25:00-26:00"} {
		if _, _, err := parseWorkingHours(in); err == nil {
			t.Errorf("parseWorkingHours(%q) should fail", in)
		}
	}
}

func TestParseTimeIn(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	got, dateOnly, err := parseTimeIn("2026-12-01", loc)
	if err != nil || !dateOnly || !got.Equal(mustTime(t, "2026-12-01T05:00:00Z")) {
		t.Errorf("date-only = %v, %v, %v", got, dateOnly, err)
	}
	got, dateOnly, err = parseTimeIn("2026-12-01 13:30", loc)
	if err != nil || dateOnly || !got.Equal(mustTime(t, "2026-12-01T18:30:00Z")) {
		t.Errorf("date-time = %v, %v, %v", got, dateOnly, err)
	}
	got, _, err = parseTimeIn("2026-12-01T10:00:00+02:00", loc)
	if err != nil || !got.Equal(mustTime(t, "2026-12-01T08:00:00Z")) {
		t.Errorf("rfc3339 = %v, %v", got, err)
	}
	if _, _, err := parseTimeIn("tomorrow", loc); err == nil {
		t.Error("expected error for unrecognized format")
	}
}

func TestMergeSpans(t *testing.T) {
	spans := []timeSpan{
		{mustTime(t, "2026-12-01T14:00:00Z"), mustTime(t, "2026-12-01T15:00:00Z")},
		{mustTime(t, "2026-12-01T10:00:00Z"), mustTime(t, "2026-12-01T11:00:00Z")},
		{mustTime(t, "2026-12-01T10:30:00Z"), mustTime(t, "2026-12-01T12:00:00Z")},
		{mustTime(t, "2026-12-01T12:00:00Z"), mustTime(t, "2026-12-01T12:30:00Z")}, // touches
		{mustTime(t, "2026-12-01T10:15:00Z"), mustTime(t, "2026-12-01T10:20:00Z")}, // contained
	}
	got := mergeSpans(spans)
	if len(got) != 2 {
		t.Fatalf("expected 2 merged spans, got %d: %v", len(got), got)
	}
	if !got[0].Start.Equal(mustTime(t, "2026-12-01T10:00:00Z")) || !got[0].End.Equal(mustTime(t, "2026-12-01T12:30:00Z")) {
		t.Errorf("first merged span = %v", got[0])
	}
	if mergeSpans(nil) != nil {
		t.Error("mergeSpans(nil) should be nil")
	}
}

func TestSuggestSlots(t *testing.T) {
	// Tuesday 2026-12-01 through Monday 2026-12-07, UTC.
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-08T00:00:00Z")
	busy := []timeSpan{
		{mustTime(t, "2026-12-01T08:30:00Z"), mustTime(t, "2026-12-01T10:00:00Z")}, // overlaps start of day
		{mustTime(t, "2026-12-01T10:20:00Z"), mustTime(t, "2026-12-01T12:00:00Z")}, // leaves a 20m gap
		{mustTime(t, "2026-12-01T13:00:00Z"), mustTime(t, "2026-12-01T18:00:00Z")}, // runs past end of day
	}

	got := suggestSlots(busy, from, to, 30*time.Minute, 9*time.Hour, 17*time.Hour, time.UTC, false, 10)

	// Tue: 12:00-13:00 only (the 20m gap is too short). Wed-Fri and the
	// next Monday are fully free. Sat/Sun are skipped.
	wantStarts := []string{
		"2026-12-01T12:00:00Z",
		"2026-12-02T09:00:00Z",
		"2026-12-03T09:00:00Z",
		"2026-12-04T09:00:00Z",
		"2026-12-07T09:00:00Z",
	}
	if len(got) != len(wantStarts) {
		t.Fatalf("got %d suggestions, want %d: %v", len(got), len(wantStarts), got)
	}
	for i, want := range wantStarts {
		if !got[i].Start.Equal(mustTime(t, want)) {
			t.Errorf("suggestion %d start = %v, want %s", i, got[i].Start, want)
		}
		if got[i].End.Sub(got[i].Start) != 30*time.Minute {
			t.Errorf("suggestion %d is not 30m long", i)
		}
	}
	if !got[0].FreeUntil.Equal(mustTime(t, "2026-12-01T13:00:00Z")) {
		t.Errorf("first free_until = %v", got[0].FreeUntil)
	}
	if !got[1].FreeUntil.Equal(mustTime(t, "2026-12-02T17:00:00Z")) {
		t.Errorf("second free_until = %v", got[1].FreeUntil)
	}

	// A 20-minute meeting fits the short gap.
	short := suggestSlots(busy, from, to, 20*time.Minute, 9*time.Hour, 17*time.Hour, time.UTC, false, 1)
	if len(short) != 1 || !short[0].Start.Equal(mustTime(t, "2026-12-01T10:00:00Z")) {
		t.Errorf("20m suggestion = %v", short)
	}

	// --max caps the result; weekends appear when asked for.
	weekend := suggestSlots(nil, mustTime(t, "2026-12-05T00:00:00Z"), to, time.Hour, 9*time.Hour, 17*time.Hour, time.UTC, true, 2)
	if len(weekend) != 2 || weekend[0].Start.Weekday() != time.Saturday || weekend[1].Start.Weekday() != time.Sunday {
		t.Errorf("weekend suggestions = %v", weekend)
	}
}

func TestSuggestSlots_ClipsToRangeAndHonoursTimezone(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	// Range starts mid-morning New York time and ends early afternoon.
	from := mustTime(t, "2026-12-01T15:30:00Z") // 10:30 EST
	to := mustTime(t, "2026-12-01T18:00:00Z")   // 13:00 EST

	got := suggestSlots(nil, from, to, time.Hour, 9*time.Hour, 17*time.Hour, loc, false, 5)
	if len(got) != 1 || !got[0].Start.Equal(from) || !got[0].FreeUntil.Equal(to) {
		t.Fatalf("suggestions = %v, want one window clipped to the range", got)
	}

	// Working hours are evaluated in loc: 09:00 EST is 14:00 UTC.
	day := suggestSlots(nil, mustTime(t, "2026-12-01T05:00:00Z"), mustTime(t, "2026-12-02T05:00:00Z"), time.Hour, 9*time.Hour, 17*time.Hour, loc, false, 5)
	if len(day) != 1 || !day[0].Start.Equal(mustTime(t, "2026-12-01T14:00:00Z")) {
		t.Errorf("timezone-aware start = %v", day)
	}

	// A range too short for the meeting yields nothing.
	if none := suggestSlots(nil, from, from.Add(20*time.Minute), time.Hour, 0, 24*time.Hour, loc, true, 5); len(none) != 0 {
		t.Errorf("expected no suggestions, got %v", none)
	}

	// A fully busy day yields nothing.
	full := []timeSpan{{mustTime(t, "2026-12-01T00:00:00Z"), mustTime(t, "2026-12-02T12:00:00Z")}}
	if none := suggestSlots(full, from, to, 30*time.Minute, 9*time.Hour, 17*time.Hour, loc, false, 5); len(none) != 0 {
		t.Errorf("expected no suggestions on a busy day, got %v", none)
	}
}

func TestSuggestSlots_SpansMidnightWhenWorkingHoursAreUnrestricted(t *testing.T) {
	// Monday 23:00 to Tuesday 01:00, empty calendar: a 90-minute meeting
	// fits only if the two whole-day windows are treated as one.
	from := mustTime(t, "2026-11-30T23:00:00Z")
	to := mustTime(t, "2026-12-01T01:00:00Z")
	got := suggestSlots(nil, from, to, 90*time.Minute, 0, 24*time.Hour, time.UTC, false, 5)
	if len(got) != 1 || !got[0].Start.Equal(from) || !got[0].FreeUntil.Equal(to) {
		t.Fatalf("suggestions = %v, want one window across midnight", got)
	}

	// Friday night into Saturday: the weekend day is not eligible, so the
	// window stops at midnight and the meeting does not fit.
	fri := mustTime(t, "2026-12-04T23:00:00Z")
	if none := suggestSlots(nil, fri, fri.Add(2*time.Hour), 90*time.Minute, 0, 24*time.Hour, time.UTC, false, 5); len(none) != 0 {
		t.Errorf("expected no suggestion across an excluded weekend day, got %v", none)
	}
	// Restricted working hours never merge across the night.
	if none := suggestSlots(nil, from, to, 30*time.Minute, 9*time.Hour, 17*time.Hour, time.UTC, false, 5); len(none) != 0 {
		t.Errorf("expected nothing outside working hours, got %v", none)
	}
}

func TestSuggestSlots_MidnightDSTTransition(t *testing.T) {
	// In America/Sao_Paulo, DST began at midnight on Sunday 2018-11-04:
	// 00:00 did not exist that day. Every weekday around it must still
	// get its 09:00-17:00 window, on the right date.
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	from := time.Date(2018, 11, 2, 0, 0, 0, 0, loc) // Friday
	to := time.Date(2018, 11, 8, 0, 0, 0, 0, loc)   // Thursday

	got := suggestSlots(nil, from, to, time.Hour, 9*time.Hour, 17*time.Hour, loc, true, 10)

	wantDays := []int{2, 3, 4, 5, 6, 7}
	if len(got) != len(wantDays) {
		t.Fatalf("got %d windows, want %d: %v", len(got), len(wantDays), got)
	}
	for i, day := range wantDays {
		s := got[i].Start.In(loc)
		e := got[i].FreeUntil.In(loc)
		if s.Day() != day || s.Hour() != 9 || s.Minute() != 0 || e.Day() != day || e.Hour() != 17 {
			t.Errorf("window %d = %v - %v, want Nov %d 09:00-17:00", i, s, e, day)
		}
	}

	// Whole-day windows across the skipped midnight stay contiguous.
	whole := suggestSlots(nil, from, to, time.Hour, 0, 24*time.Hour, loc, true, 10)
	if len(whole) != 1 || !whole[0].Start.Equal(from) || !whole[0].FreeUntil.Equal(to) {
		t.Errorf("whole-day windows = %v, want one continuous window", whole)
	}
}

func TestCivilDateBoundaries_MidnightDSTTransition(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// 2018-11-04 00:00 did not exist: the day began at 01:00 (-02:00),
	// which is 03:00 UTC.
	firstInstant := mustTime(t, "2018-11-04T03:00:00Z")

	if got := startOfCivilDate(2018, 11, 4, loc); !got.Equal(firstInstant) {
		t.Errorf("startOfCivilDate = %v, want %v", got.UTC(), firstInstant)
	}
	// A date-only --from on the transition date starts at that instant.
	from, dateOnly, err := parseTimeIn("2018-11-04", loc)
	if err != nil || !dateOnly || !from.Equal(firstInstant) {
		t.Errorf("parseTimeIn = %v, %v, %v", from.UTC(), dateOnly, err)
	}
	// A date-only --to of the day before includes all of that day: the
	// exclusive cutoff is the first instant of the transition date.
	to, _, _ := parseTimeIn("2018-11-03", loc)
	if got := endOfCivilDate(to, loc); !got.Equal(firstInstant) {
		t.Errorf("endOfCivilDate(2018-11-03) = %v, want %v", got.UTC(), firstInstant)
	}
	// ...and a date-only --to on the transition date ends a civil day later.
	if got := endOfCivilDate(from, loc); !got.Equal(time.Date(2018, 11, 5, 0, 0, 0, 0, loc)) {
		t.Errorf("endOfCivilDate(2018-11-04) = %v", got)
	}
	// Ordinary dates and month overflow are unaffected.
	if got := startOfCivilDate(2026, 1, 32, time.UTC); !got.Equal(mustTime(t, "2026-02-01T00:00:00Z")) {
		t.Errorf("day overflow = %v", got)
	}
}

func newSuggestTestService(t *testing.T, handler http.HandlerFunc) *calendar.Service {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	svc, err := calendar.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}
	return svc
}

// TestQueryBusySpans_MockServer drives the free/busy request and response
// handling behind suggest-time: the request body, per-calendar errors, and
// the suggestions computed from the returned busy periods.
func TestQueryBusySpans_MockServer(t *testing.T) {
	var got calendar.FreeBusyRequest
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/freeBusy" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Calendars: map[string]calendar.FreeBusyCalendar{
				"primary": {Busy: []*calendar.TimePeriod{
					{Start: "2026-12-01T09:00:00Z", End: "2026-12-01T11:00:00Z"},
				}},
				"a@example.com": {Busy: []*calendar.TimePeriod{
					{Start: "2026-12-01T10:30:00Z", End: "2026-12-01T15:00:00Z"},
				}},
				"b@example.com": {Errors: []*calendar.Error{{Domain: "global", Reason: "notFound"}}},
				// "c@example.com" is absent from the response.
			},
		})
	})

	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "primary"}, {Id: "a@example.com"}, {Id: "b@example.com"}, {Id: "c@example.com"}}

	busy, checked, unavailable, err := queryBusySpans(svc, items, from, to)
	if err != nil {
		t.Fatalf("queryBusySpans: %v", err)
	}

	if got.TimeMin != "2026-12-01T00:00:00Z" || got.TimeMax != "2026-12-02T00:00:00Z" || len(got.Items) != 4 || got.Items[1].Id != "a@example.com" {
		t.Errorf("unexpected free/busy request: %+v", got)
	}
	if strings.Join(checked, ",") != "primary,a@example.com" {
		t.Errorf("checked = %v", checked)
	}
	if len(unavailable) != 2 ||
		unavailable[0]["calendar"] != "b@example.com" || unavailable[0]["reason"] != "notFound" ||
		unavailable[1]["calendar"] != "c@example.com" || unavailable[1]["reason"] != "missing from response" {
		t.Errorf("unavailable = %v", unavailable)
	}
	if len(busy) != 2 {
		t.Fatalf("busy spans = %v", busy)
	}

	// The two calendars are jointly busy 09:00-15:00, so the only free
	// window in working hours is 15:00-17:00.
	slots := suggestSlots(busy, from, to, time.Hour, 9*time.Hour, 17*time.Hour, time.UTC, false, 5)
	if len(slots) != 1 || !slots[0].Start.Equal(mustTime(t, "2026-12-01T15:00:00Z")) || !slots[0].FreeUntil.Equal(mustTime(t, "2026-12-01T17:00:00Z")) {
		t.Errorf("suggestions = %v", slots)
	}
}

func TestQueryBusySpans_Failures(t *testing.T) {
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "a@example.com"}}

	// No readable calendar: refuse to suggest anything.
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Calendars: map[string]calendar.FreeBusyCalendar{
				"a@example.com": {Errors: []*calendar.Error{{Reason: "notFound"}}},
			},
		})
	})
	if _, _, _, err := queryBusySpans(svc, items, from, to); err == nil || !strings.Contains(err.Error(), "none of the requested calendars") {
		t.Errorf("error = %v, want unreadable-calendars error", err)
	}

	// API errors keep their HTTP status for exit-code mapping.
	svc = newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"code":403,"message":"Request had insufficient authentication scopes."}}`))
	})
	_, _, _, err := queryBusySpans(svc, items, from, to)
	if err == nil {
		t.Fatal("expected an API error")
	}
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) || apiErr.Code != http.StatusForbidden {
		t.Errorf("error = %v, want wrapped 403", err)
	}
	if exitCodeForError(err) != ExitAuth {
		t.Errorf("exit code = %d, want %d", exitCodeForError(err), ExitAuth)
	}

	// A malformed busy period is an error, not a silently ignored slot.
	svc = newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Calendars: map[string]calendar.FreeBusyCalendar{
				"a@example.com": {Busy: []*calendar.TimePeriod{{Start: "not-a-time", End: "2026-12-01T10:00:00Z"}}},
			},
		})
	})
	if _, _, _, err := queryBusySpans(svc, items, from, to); err == nil || !strings.Contains(err.Error(), "unexpected busy period") {
		t.Errorf("error = %v, want malformed-period error", err)
	}
}
