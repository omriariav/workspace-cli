package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	for _, flag := range []string{"from", "to", "duration", "calendars", "attendees", "working-hours", "timezone", "max", "include-weekends", "allow-partial"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestCalendarSuggestTime_RejectsTooManyCalendars(t *testing.T) {
	cmd := calendarSuggestTimeCmd
	set := func(name, value string) {
		t.Helper()
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, name := range []string{"from", "to", "attendees"} {
			_ = cmd.Flags().Set(name, "")
			cmd.Flags().Lookup(name).Changed = false
		}
	})

	attendees := make([]string, 0, maxFreeBusyCalendars)
	for i := 0; i < maxFreeBusyCalendars; i++ {
		attendees = append(attendees, fmt.Sprintf("user%d@example.com", i))
	}
	set("from", "2026-12-01")
	set("to", "2026-12-02")
	// 50 attendees plus the default "primary" calendar is one too many;
	// the command fails before it needs credentials or the network.
	set("attendees", strings.Join(attendees, ","))
	err := runCalendarSuggestTime(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "too many calendars: 51") {
		t.Fatalf("error = %v, want too-many-calendars usage error", err)
	}
	if resolveExitError(err, io.Discard) != ExitUsage {
		t.Errorf("exit code = %d, want %d", resolveExitError(err, io.Discard), ExitUsage)
	}
}

func TestCalendarSuggestTime_RejectsOverlongRange(t *testing.T) {
	cmd := calendarSuggestTimeCmd
	t.Cleanup(func() {
		for _, name := range []string{"from", "to"} {
			_ = cmd.Flags().Set(name, "")
			cmd.Flags().Lookup(name).Changed = false
		}
	})
	_ = cmd.Flags().Set("from", "2026-01-01")
	_ = cmd.Flags().Set("to", "2027-01-01")
	// Rejected before any client is created or request sent.
	err := runCalendarSuggestTime(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "search range is too long") {
		t.Fatalf("error = %v, want a too-long-range usage error", err)
	}
	if resolveExitError(err, io.Discard) != ExitUsage {
		t.Errorf("exit code = %d, want %d", resolveExitError(err, io.Discard), ExitUsage)
	}
}

func TestCalendarSuggestTime_RejectsPositionalArgs(t *testing.T) {
	if err := calendarSuggestTimeCmd.Args(calendarSuggestTimeCmd, []string{"a@example.com"}); err == nil {
		t.Error("a stray positional argument should be rejected")
	}
	if err := calendarSuggestTimeCmd.Args(calendarSuggestTimeCmd, nil); err != nil {
		t.Errorf("no arguments should be accepted: %v", err)
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

// End to end through the command handler against a mock Calendar server:
// flag parsing, the free/busy request, and the printed result.
func TestCalendarSuggestTime_CommandAgainstMockServer(t *testing.T) {
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
				"primary":       {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T09:00:00Z", End: "2026-12-01T12:00:00Z"}}},
				"a@example.com": {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T13:00:00Z", End: "2026-12-01T16:00:00Z"}}},
				"b@example.com": {Errors: []*calendar.Error{{Reason: "notFound"}}},
			},
		})
	})
	original := suggestTimeCalendarService
	suggestTimeCalendarService = func(context.Context) (*calendar.Service, error) { return svc, nil }
	t.Cleanup(func() { suggestTimeCalendarService = original })

	cmd := calendarSuggestTimeCmd
	flags := map[string]string{
		"from": "2026-12-01", "to": "2026-12-01", "timezone": "UTC", "duration": "45",
		"attendees": "a@example.com, b@example.com", "max": "3",
	}
	set := func(name, value string) {
		t.Helper()
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	for name, value := range flags {
		set(name, value)
	}
	t.Cleanup(func() {
		for name := range flags {
			f := cmd.Flags().Lookup(name)
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
		f := cmd.Flags().Lookup("allow-partial")
		_ = f.Value.Set("false")
		f.Changed = false
	})

	// One attendee is unreadable: by default the command refuses.
	if _, err := captureStdout(t, func() error { return runCalendarSuggestTime(cmd, nil) }); err == nil || !strings.Contains(err.Error(), "b@example.com (notFound)") {
		t.Fatalf("error = %v, want the unreadable calendar named", err)
	}
	// The request covered the whole date (a date-only --to is inclusive)
	// and every calendar, de-duplicated and in order.
	if got.TimeMin != "2026-12-01T00:00:00Z" || got.TimeMax != "2026-12-02T00:00:00Z" {
		t.Errorf("request range = %s - %s", got.TimeMin, got.TimeMax)
	}
	var ids []string
	for _, item := range got.Items {
		ids = append(ids, item.Id)
	}
	if strings.Join(ids, ",") != "primary,a@example.com,b@example.com" {
		t.Errorf("request calendars = %v", ids)
	}

	// With --allow-partial it proceeds and says the result is partial.
	set("allow-partial", "true")
	out, err := captureStdout(t, func() error { return runCalendarSuggestTime(cmd, nil) })
	if err != nil {
		t.Fatalf("runCalendarSuggestTime: %v", err)
	}
	var result struct {
		Timezone         string   `json:"timezone"`
		DurationMinutes  int      `json:"duration_minutes"`
		CalendarsChecked []string `json:"calendars_checked"`
		Partial          bool     `json:"partial"`
		Count            int      `json:"count"`
		Unavailable      []struct {
			Calendar string `json:"calendar"`
			Reason   string `json:"reason"`
		} `json:"unavailable"`
		Suggestions []struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			FreeUntil   string `json:"free_until"`
			FreeMinutes int    `json:"free_minutes"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if !result.Partial || len(result.Unavailable) != 1 || result.Unavailable[0].Calendar != "b@example.com" || result.Unavailable[0].Reason != "notFound" {
		t.Errorf("partial = %v, unavailable = %+v", result.Partial, result.Unavailable)
	}
	if strings.Join(result.CalendarsChecked, ",") != "primary,a@example.com" || result.DurationMinutes != 45 || result.Timezone != "UTC" {
		t.Errorf("checked = %v, duration = %d, timezone = %s", result.CalendarsChecked, result.DurationMinutes, result.Timezone)
	}
	// Working hours 09:00-17:00, busy 09:00-12:00 and 13:00-16:00: the
	// free windows are 12:00-13:00 and 16:00-17:00.
	if result.Count != 2 || len(result.Suggestions) != 2 {
		t.Fatalf("suggestions = %+v", result.Suggestions)
	}
	first, second := result.Suggestions[0], result.Suggestions[1]
	if first.Start != "2026-12-01T12:00:00Z" || first.End != "2026-12-01T12:45:00Z" || first.FreeUntil != "2026-12-01T13:00:00Z" || first.FreeMinutes != 60 {
		t.Errorf("first suggestion = %+v", first)
	}
	if second.Start != "2026-12-01T16:00:00Z" || second.FreeUntil != "2026-12-01T17:00:00Z" {
		t.Errorf("second suggestion = %+v", second)
	}
}

func TestParseTimeIn_NonexistentLocalTimeMovesForward(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// 02:30 on 2026-03-08 does not exist in New York. As a --from it must
	// become 03:00 EDT (07:00 UTC), never 01:30 EST, which is earlier
	// than what was asked for.
	got, dateOnly, err := parseTimeIn("2026-03-08 02:30", loc)
	if err != nil || dateOnly || !got.Equal(mustTime(t, "2026-03-08T07:00:00Z")) {
		t.Errorf("parseTimeIn = %v, %v, %v; want 2026-03-08T07:00:00Z", got.UTC(), dateOnly, err)
	}
	slots := suggestSlots(nil, got, got.Add(6*time.Hour), time.Hour, 0, 24*time.Hour, loc, true, 1)
	if len(slots) != 1 || slots[0].Start.Before(mustTime(t, "2026-03-08T07:00:00Z")) {
		t.Errorf("suggestion %v starts before the requested --from", slots)
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

// Liberia moved its clocks by 44 minutes 30 seconds on 1972-01-07. The
// working-hours cutoff that day must still fall exactly on 17:00 local
// time, not half a minute later.
func TestDayWindows_SubMinuteOffsetChange(t *testing.T) {
	loc, err := time.LoadLocation("Africa/Monrovia")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	civil := time.Date(1972, 1, 7, 12, 0, 0, 0, time.UTC)
	before := time.Date(1972, 1, 6, 12, 0, 0, 0, loc)
	after := time.Date(1972, 1, 8, 12, 0, 0, 0, loc)
	_, offBefore := before.Zone()
	_, offAfter := after.Zone()
	if offBefore == offAfter || (offAfter-offBefore)%60 == 0 {
		t.Skipf("tzdata does not model the sub-minute change (offsets %d -> %d)", offBefore, offAfter)
	}

	windows := dayWindows(civil, 9*time.Hour, 17*time.Hour, loc)
	if len(windows) != 1 {
		t.Fatalf("windows = %v, want one", windows)
	}
	start, end := windows[0].Start.In(loc), windows[0].End.In(loc)
	if start.Hour() != 9 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("window starts at %v, want 09:00:00 local", start)
	}
	if end.Hour() != 17 || end.Minute() != 0 || end.Second() != 0 {
		t.Errorf("window ends at %v, want exactly 17:00:00 local", end)
	}
	if got := windows[0].End.Sub(windows[0].Start); got != 8*time.Hour {
		t.Errorf("window lasts %v, want 8h", got)
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
	if got, err := endOfDateOnly("2018-11-03", loc); err != nil || !got.Equal(firstInstant) {
		t.Errorf("endOfDateOnly(2018-11-03) = %v, %v; want %v", got.UTC(), err, firstInstant)
	}
	// ...and a date-only --to on the transition date ends a civil day later.
	if got, err := endOfDateOnly("2018-11-04", loc); err != nil || !got.Equal(time.Date(2018, 11, 5, 0, 0, 0, 0, loc)) {
		t.Errorf("endOfDateOnly(2018-11-04) = %v, %v", got, err)
	}
	if _, err := endOfDateOnly("not-a-date", loc); err == nil {
		t.Error("endOfDateOnly should reject a malformed date")
	}
	// Ordinary dates and month overflow are unaffected.
	if got := startOfCivilDate(2026, 1, 32, time.UTC); !got.Equal(mustTime(t, "2026-02-01T00:00:00Z")) {
		t.Errorf("day overflow = %v", got)
	}
}

func TestSuggestSlots_WorkingHoursBoundaryInSpringForwardGap(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// On Sunday 2026-03-08 clocks jump from 02:00 to 03:00, so 02:30 does
	// not exist. A window that starts at 02:30 must begin at 03:00 EDT
	// (07:00 UTC), never at 01:30 EST.
	gapStart := mustTime(t, "2026-03-08T07:00:00Z")
	if got := civilInstant(2026, 3, 8, 2, 30, loc); !got.Equal(gapStart) {
		t.Errorf("civilInstant(02:30) = %v, want %v", got.UTC(), gapStart)
	}
	// Existing times are untouched, on both sides of the change.
	if got := civilInstant(2026, 3, 8, 1, 30, loc); !got.Equal(mustTime(t, "2026-03-08T06:30:00Z")) {
		t.Errorf("civilInstant(01:30) = %v", got.UTC())
	}
	if got := civilInstant(2026, 3, 8, 9, 0, loc); !got.Equal(mustTime(t, "2026-03-08T13:00:00Z")) {
		t.Errorf("civilInstant(09:00) = %v", got.UTC())
	}
	// An ambiguous autumn time resolves to its first occurrence (EDT).
	if got := civilInstant(2026, 11, 1, 1, 30, loc); !got.Equal(mustTime(t, "2026-11-01T05:30:00Z")) {
		t.Errorf("civilInstant(ambiguous 01:30) = %v", got.UTC())
	}

	// Fall back, Sunday 2026-11-01: 01:00-02:00 happens twice. A window
	// ending at 01:30 must stop at the first 01:30 (EDT, 05:30 UTC), not
	// run on for another hour to the second one.
	fallFrom := time.Date(2026, 11, 1, 0, 0, 0, 0, loc)
	fallTo := time.Date(2026, 11, 2, 0, 0, 0, 0, loc)
	// Working hours 00:00-01:30 are met twice: 00:00-01:30 EDT, and
	// again when 01:00-01:30 repeats in EST. The half hour between them
	// (01:30-02:00 EDT) is outside working hours, so these are two windows.
	fall := suggestSlots(nil, fallFrom, fallTo, 30*time.Minute, 0, time.Hour+30*time.Minute, loc, true, 5)
	if len(fall) != 2 ||
		!fall[0].Start.Equal(mustTime(t, "2026-11-01T04:00:00Z")) || !fall[0].FreeUntil.Equal(mustTime(t, "2026-11-01T05:30:00Z")) ||
		!fall[1].Start.Equal(mustTime(t, "2026-11-01T06:00:00Z")) || !fall[1].FreeUntil.Equal(mustTime(t, "2026-11-01T06:30:00Z")) {
		t.Errorf("fall-back windows = %v, want 04:00-05:30 and 06:00-06:30 UTC", fall)
	}

	// Working hours 01:30-02:00 also occur twice, an hour apart. A
	// 60-minute meeting would have to run through 01:00-01:30 EST, which
	// is outside the requested hours, so nothing fits...
	if none := suggestSlots(nil, fallFrom, fallTo, 60*time.Minute, time.Hour+30*time.Minute, 2*time.Hour, loc, true, 5); len(none) != 0 {
		t.Errorf("a 60m meeting must not span excluded local times, got %v", none)
	}
	// ...while a 30-minute one fits each occurrence separately.
	twice := suggestSlots(nil, fallFrom, fallTo, 30*time.Minute, time.Hour+30*time.Minute, 2*time.Hour, loc, true, 5)
	if len(twice) != 2 || !twice[0].Start.Equal(mustTime(t, "2026-11-01T05:30:00Z")) || !twice[1].Start.Equal(mustTime(t, "2026-11-01T06:30:00Z")) {
		t.Errorf("repeated working hours = %v, want 05:30 and 06:30 UTC", twice)
	}
	for _, s := range twice {
		for c := s.Start; c.Before(s.End); c = c.Add(time.Minute) {
			l := c.In(loc)
			if l.Hour() != 1 || l.Minute() < 30 {
				t.Errorf("slot %v includes %v, outside 01:30-02:00", s, l)
			}
		}
	}

	for _, m := range []int{0, 15, 30, 59} {
		got := civilInstant(2026, 11, 1, 1, m, loc)
		if name, _ := got.In(loc).Zone(); name != "EDT" {
			t.Errorf("civilInstant(01:%02d) resolved to %s, want the first occurrence (EDT)", m, name)
		}
	}

	from := time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	to := time.Date(2026, 3, 9, 0, 0, 0, 0, loc)
	got := suggestSlots(nil, from, to, 30*time.Minute, 2*time.Hour+30*time.Minute, 17*time.Hour, loc, true, 5)
	if len(got) != 1 || !got[0].Start.Equal(gapStart) {
		t.Fatalf("suggestions = %v, want one window starting at 03:00 EDT", got)
	}
	if h := got[0].Start.In(loc).Hour(); h < 2 {
		t.Errorf("suggestion starts at %v, before the working-hours lower bound", got[0].Start.In(loc))
	}
}

func TestCivilDates_WholeDaySkipped(t *testing.T) {
	loc, err := time.LoadLocation("Pacific/Apia")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// Samoa went from Thursday 2011-12-29 straight to Saturday 2011-12-31:
	// December 30 never happened there.
	dec31 := time.Date(2011, 12, 31, 0, 0, 0, 0, loc)
	if l := dec31.In(loc); l.Day() != 31 {
		t.Skipf("tzdata does not model the skipped day (got %v)", l)
	}

	// The skipped date starts where the next real day starts...
	if got := startOfCivilDate(2011, 12, 30, loc); !got.Equal(dec31) {
		t.Errorf("startOfCivilDate(skipped day) = %v, want %v", got.In(loc), dec31)
	}
	// ...a date-only --to of the 29th ends at that same instant, having
	// covered the whole of the 29th and nothing of the 31st...
	end29, err := endOfDateOnly("2011-12-29", loc)
	if err != nil || !end29.Equal(dec31) {
		t.Errorf("endOfDateOnly(2011-12-29) = %v, %v; want %v", end29.In(loc), err, dec31)
	}
	// ...and a date-only --to of the skipped day adds no time at all: it
	// does not spill onto the 31st.
	end30, err := endOfDateOnly("2011-12-30", loc)
	if err != nil || !end30.Equal(dec31) {
		t.Errorf("endOfDateOnly(2011-12-30) = %v, %v; want %v", end30.In(loc), err, dec31)
	}
	// The 31st itself is a full, ordinary day.
	end31, _ := endOfDateOnly("2011-12-31", loc)
	if end31.Sub(dec31) != 24*time.Hour {
		t.Errorf("2011-12-31 lasts %v, want 24h", end31.Sub(dec31))
	}
}

func TestQueryBusySpans_ExpandsGroups(t *testing.T) {
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Groups: map[string]calendar.FreeBusyGroup{
				"team@example.com":   {Calendars: []string{"a@example.com", "b@example.com"}},
				"broken@example.com": {Errors: []*calendar.Error{{Reason: "groupTooBig"}}},
				// The unreadable member comes first: the readable one after
				// it must still be processed.
				"half@example.com":  {Calendars: []string{"hidden@example.com", "c@example.com"}},
				"empty@example.com": {},
			},
			Calendars: map[string]calendar.FreeBusyCalendar{
				"a@example.com":      {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T09:00:00Z", End: "2026-12-01T10:00:00Z"}}},
				"b@example.com":      {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T14:00:00Z", End: "2026-12-01T15:00:00Z"}}},
				"hidden@example.com": {Errors: []*calendar.Error{{Reason: "notFound"}}},
				"c@example.com":      {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T16:00:00Z", End: "2026-12-01T17:00:00Z"}}},
			},
		})
	})
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "team@example.com"}, {Id: "broken@example.com"}, {Id: "half@example.com"}, {Id: "empty@example.com"}}

	busy, checked, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
	if err != nil {
		t.Fatal(err)
	}
	// The expanded group is readable, and its members' busy time counts.
	if strings.Join(checked, ",") != "team@example.com" {
		t.Errorf("checked = %v, want only the fully readable group", checked)
	}
	merged := mergeSpans(busy)
	// ...including the readable member of the partly unreadable group:
	// c@example.com is known to be busy 16:00-17:00, so that hour must not
	// be offered even though its group as a whole is unavailable.
	if len(merged) != 3 || !merged[0].Start.Equal(mustTime(t, "2026-12-01T09:00:00Z")) || !merged[1].Start.Equal(mustTime(t, "2026-12-01T14:00:00Z")) || !merged[2].Start.Equal(mustTime(t, "2026-12-01T16:00:00Z")) {
		t.Errorf("busy = %v, want every readable member's periods", merged)
	}
	for _, s := range suggestSlots(busy, from, to, time.Hour, 9*time.Hour, 17*time.Hour, time.UTC, true, 10) {
		if s.Start.Before(mustTime(t, "2026-12-01T17:00:00Z")) && s.FreeUntil.After(mustTime(t, "2026-12-01T16:00:00Z")) {
			t.Errorf("suggestion %v overlaps a known busy period of a group member", s)
		}
	}
	reasons := map[string]string{}
	for _, u := range unavailable {
		reasons[u["calendar"].(string)] = u["reason"].(string)
	}
	if reasons["broken@example.com"] != "groupTooBig" ||
		!strings.Contains(reasons["half@example.com"], "hidden@example.com") ||
		reasons["empty@example.com"] == "" || len(reasons) != 3 {
		t.Errorf("unavailable = %v", reasons)
	}
}

// A partly unreadable group must give the same answer whatever order
// Google lists its members in.
func TestQueryBusySpans_GroupMemberOrderDoesNotMatter(t *testing.T) {
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "primary"}, {Id: "team@example.com"}}

	run := func(members []string) (string, []suggestion) {
		svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
				Groups: map[string]calendar.FreeBusyGroup{"team@example.com": {Calendars: members}},
				Calendars: map[string]calendar.FreeBusyCalendar{
					"primary":            {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T00:00:00Z", End: "2026-12-01T01:00:00Z"}}},
					"busy@example.com":   {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T09:00:00Z", End: "2026-12-01T13:00:00Z"}}},
					"hidden@example.com": {Errors: []*calendar.Error{{Reason: "notFound"}}},
				},
			})
		})
		busy, checked, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(checked, ",") != "primary" || len(unavailable) != 1 || unavailable[0]["calendar"] != "team@example.com" {
			t.Errorf("members %v: checked = %v, unavailable = %v", members, checked, unavailable)
		}
		slots := suggestSlots(busy, from, to, time.Hour, 9*time.Hour, 17*time.Hour, time.UTC, true, 5)
		var starts []string
		for _, s := range slots {
			starts = append(starts, s.Start.Format(time.RFC3339))
		}
		return strings.Join(starts, ","), slots
	}

	first, slots := run([]string{"busy@example.com", "hidden@example.com"})
	second, _ := run([]string{"hidden@example.com", "busy@example.com"})
	if first != second {
		t.Errorf("suggestions depend on member order: %q vs %q", first, second)
	}
	// Either way, the readable member's morning is blocked.
	if len(slots) != 1 || !slots[0].Start.Equal(mustTime(t, "2026-12-01T13:00:00Z")) {
		t.Errorf("suggestions = %v, want a single window from 13:00", slots)
	}
}

// A single group with one readable and one unreadable member: something
// was read, so --allow-partial can proceed; nothing read at all cannot.
func TestQueryBusySpans_SingleGroupPartlyReadable(t *testing.T) {
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "team@example.com"}}
	respond := func(calendars map[string]calendar.FreeBusyCalendar) *calendar.Service {
		return newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
				Groups:    map[string]calendar.FreeBusyGroup{"team@example.com": {Calendars: []string{"busy@example.com", "hidden@example.com"}}},
				Calendars: calendars,
			})
		})
	}

	svc := respond(map[string]calendar.FreeBusyCalendar{
		"busy@example.com":   {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T09:00:00Z", End: "2026-12-01T12:00:00Z"}}},
		"hidden@example.com": {Errors: []*calendar.Error{{Reason: "notFound"}}},
	})
	busy, checked, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
	if err != nil {
		t.Fatalf("a partly readable group should not be a hard failure: %v", err)
	}
	if len(checked) != 0 || len(unavailable) != 1 || len(busy) != 1 {
		t.Errorf("checked = %v, unavailable = %v, busy = %v", checked, unavailable, busy)
	}
	// Without --allow-partial the command still refuses; with it, it goes on.
	if unreadableCalendarsError(unavailable, false) == nil {
		t.Error("the group is unavailable, so the default must still refuse")
	}
	if err := unreadableCalendarsError(unavailable, true); err != nil {
		t.Errorf("--allow-partial should proceed: %v", err)
	}

	// No member readable at all: an error regardless of --allow-partial.
	svc = respond(map[string]calendar.FreeBusyCalendar{
		"busy@example.com":   {Errors: []*calendar.Error{{Reason: "notFound"}}},
		"hidden@example.com": {Errors: []*calendar.Error{{Reason: "notFound"}}},
	})
	if _, _, _, err := queryBusySpans(context.Background(), svc, items, from, to); err == nil || !strings.Contains(err.Error(), "none of the requested calendars") {
		t.Errorf("error = %v, want none-readable error", err)
	}
}

// Null entries in the response must not crash the command. The Go client
// models calendars and groups as values, so those decode to empty entries;
// busy periods are pointers, so a null one arrives as nil.
func TestQueryBusySpans_NullEntriesDoNotPanic(t *testing.T) {
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"calendars":{"primary":null,"a@example.com":{"busy":[{"start":"2026-12-01T09:00:00Z","end":"2026-12-01T10:00:00Z"}]},"n@example.com":{"busy":[null]}},"groups":{"team@example.com":null}}`))
	})
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "primary"}, {Id: "a@example.com"}, {Id: "team@example.com"}, {Id: "n@example.com"}}

	busy, _, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
	if err != nil {
		t.Fatalf("queryBusySpans: %v", err)
	}
	// A null busy period — a nil pointer in the client's list — makes that
	// calendar unavailable: strict mode refuses, --allow-partial goes on.
	nullPeriodReported := false
	for _, u := range unavailable {
		if u["calendar"] == "n@example.com" && strings.Contains(u["reason"].(string), "invalid busy period") {
			nullPeriodReported = true
		}
	}
	if !nullPeriodReported {
		t.Errorf("a calendar with a null busy period should be unavailable: %v", unavailable)
	}
	if unreadableCalendarsError(unavailable, false) == nil {
		t.Error("strict mode should refuse when a calendar is unavailable")
	}
	if err := unreadableCalendarsError(unavailable, true); err != nil {
		t.Errorf("--allow-partial should proceed: %v", err)
	}
	if len(busy) != 1 {
		t.Errorf("busy = %v, want the one real period", busy)
	}
	// A null calendar entry reports nothing, so it is unknown, not free.
	nullCalendarReported := false
	for _, u := range unavailable {
		if u["calendar"] == "primary" && u["reason"] == "no availability data returned" {
			nullCalendarReported = true
		}
	}
	if !nullCalendarReported {
		t.Errorf("a null calendar entry should be unavailable, not free: %v", unavailable)
	}
	// The null group has no members to read, so it is unavailable.
	found := false
	for _, u := range unavailable {
		if u["calendar"] == "team@example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("a null group should be reported as unavailable: %v", unavailable)
	}
}

// An explicitly empty busy list means free; a missing one means unknown.
func TestQueryBusySpans_EmptyBusyListIsFreeMissingIsUnknown(t *testing.T) {
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"calendars":{"free@example.com":{"busy":[]},"blank@example.com":{},"null@example.com":null}}`))
	})
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "free@example.com"}, {Id: "blank@example.com"}, {Id: "null@example.com"}}

	busy, checked, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(checked, ",") != "free@example.com" || len(busy) != 0 {
		t.Errorf("checked = %v, busy = %v; want only the calendar with an explicit empty list", checked, busy)
	}
	if len(unavailable) != 2 || unavailable[0]["calendar"] != "blank@example.com" || unavailable[1]["calendar"] != "null@example.com" {
		t.Errorf("unavailable = %v", unavailable)
	}
	// Strict mode refuses rather than treat the unknown calendars as free.
	if unreadableCalendarsError(unavailable, false) == nil {
		t.Error("strict mode must reject a calendar with no availability data")
	}
}

// The free/busy request carries its bounds in UTC, so the instant sent is
// the instant searched even where the local offset has a seconds part.
// Cancelling the command abandons the free/busy request.
func TestQueryBusySpans_HonoursCancellation(t *testing.T) {
	release := make(chan struct{})
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, _, err := queryBusySpans(ctx, svc, []*calendar.FreeBusyRequestItem{{Id: "primary"}},
			mustTime(t, "2026-12-01T00:00:00Z"), mustTime(t, "2026-12-02T00:00:00Z"))
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v, want the cancellation", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the request kept waiting after the command was cancelled")
	}
}

func TestQueryBusySpans_RequestBoundsAreUTC(t *testing.T) {
	var got calendar.FreeBusyRequest
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"calendars":{"primary":{"busy":[]}}}`))
	})
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// Paris in 1900 was on local mean time, +00:09:21.
	from := time.Date(1900, 6, 1, 9, 0, 0, 0, loc)
	to := from.Add(8 * time.Hour)
	if _, _, _, err := queryBusySpans(context.Background(), svc, []*calendar.FreeBusyRequestItem{{Id: "primary"}}, from, to); err != nil {
		t.Fatal(err)
	}
	sentFrom, err1 := time.Parse(time.RFC3339, got.TimeMin)
	sentTo, err2 := time.Parse(time.RFC3339, got.TimeMax)
	if err1 != nil || err2 != nil || !sentFrom.Equal(from) || !sentTo.Equal(to) {
		t.Errorf("request bounds %s - %s do not match the searched instants %s - %s", got.TimeMin, got.TimeMax, from.UTC(), to.UTC())
	}
	if !strings.HasSuffix(got.TimeMin, "Z") || !strings.HasSuffix(got.TimeMax, "Z") {
		t.Errorf("request bounds should be in UTC: %s - %s", got.TimeMin, got.TimeMax)
	}
}

// A group that reports an error but still lists members: the group is
// unavailable, yet a readable member's busy period must block that slot.
func TestQueryBusySpans_GroupErrorStillUsesReturnedMembers(t *testing.T) {
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"groups": {"team@example.com": {"errors": [{"reason": "groupTooBig"}], "calendars": ["busy@example.com"]}},
			"calendars": {
				"primary": {"busy": []},
				"busy@example.com": {"busy": [{"start": "2026-12-01T09:00:00Z", "end": "2026-12-01T16:00:00Z"}]}
			}
		}`))
	})
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	busy, checked, unavailable, err := queryBusySpans(context.Background(), svc,
		[]*calendar.FreeBusyRequestItem{{Id: "primary"}, {Id: "team@example.com"}}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(checked, ",") != "primary" || len(unavailable) != 1 || unavailable[0]["calendar"] != "team@example.com" || unavailable[0]["reason"] != "groupTooBig" {
		t.Errorf("checked = %v, unavailable = %v", checked, unavailable)
	}
	if len(busy) != 1 {
		t.Fatalf("busy = %v, want the readable member's period", busy)
	}
	// With --allow-partial, only 16:00-17:00 is offered: the member is
	// known to be busy until 16:00.
	slots := suggestSlots(busy, from, to, time.Hour, 9*time.Hour, 17*time.Hour, time.UTC, true, 5)
	if len(slots) != 1 || !slots[0].Start.Equal(mustTime(t, "2026-12-01T16:00:00Z")) {
		t.Errorf("suggestions = %v, want a single slot at 16:00", slots)
	}
}

func TestQueryBusySpans_NullErrorEntriesDoNotPanic(t *testing.T) {
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"calendars":{"a@example.com":{"errors":[null]},"b@example.com":{"errors":[null,{"reason":"notFound"}]},"primary":{"busy":[]}},"groups":{"team@example.com":{"errors":[null]}}}`))
	})
	from := mustTime(t, "2026-12-01T00:00:00Z")
	to := mustTime(t, "2026-12-02T00:00:00Z")
	items := []*calendar.FreeBusyRequestItem{{Id: "a@example.com"}, {Id: "b@example.com"}, {Id: "team@example.com"}, {Id: "primary"}}

	_, checked, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]string{}
	for _, u := range unavailable {
		reasons[u["calendar"].(string)] = u["reason"].(string)
	}
	if strings.Join(checked, ",") != "primary" || len(reasons) != 3 ||
		reasons["a@example.com"] != "unspecified error" || reasons["b@example.com"] != "notFound" || reasons["team@example.com"] != "unspecified error" {
		t.Errorf("checked = %v, unavailable = %v", checked, reasons)
	}
	if got := firstErrorReason(nil); got != "unspecified error" {
		t.Errorf("firstErrorReason(nil) = %q", got)
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

	busy, checked, unavailable, err := queryBusySpans(context.Background(), svc, items, from, to)
	if err != nil {
		t.Fatalf("queryBusySpans: %v", err)
	}

	if got.TimeMin != "2026-12-01T00:00:00Z" || got.TimeMax != "2026-12-02T00:00:00Z" || len(got.Items) != 4 || got.Items[1].Id != "a@example.com" {
		t.Errorf("unexpected free/busy request: %+v", got)
	}
	// Expansion limits are stated explicitly, not left to server defaults.
	if got.CalendarExpansionMax != maxFreeBusyCalendars || got.GroupExpansionMax != maxGroupExpansion {
		t.Errorf("expansion limits = %d calendars, %d group members", got.CalendarExpansionMax, got.GroupExpansionMax)
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

	// Unknown availability is an error unless partial results are asked for.
	if err := unreadableCalendarsError(unavailable, false); err == nil ||
		!strings.Contains(err.Error(), "b@example.com (notFound)") || !strings.Contains(err.Error(), "c@example.com") || !strings.Contains(err.Error(), "--allow-partial") {
		t.Errorf("error = %v, want both unreadable calendars named", err)
	}
	if err := unreadableCalendarsError(unavailable, true); err != nil {
		t.Errorf("--allow-partial should proceed, got %v", err)
	}
	if err := unreadableCalendarsError(nil, false); err != nil {
		t.Errorf("no unreadable calendars should proceed, got %v", err)
	}

	// The two calendars are jointly busy 09:00-15:00, so the only free
	// window in working hours is 15:00-17:00.
	slots := suggestSlots(busy, from, to, time.Hour, 9*time.Hour, 17*time.Hour, time.UTC, false, 5)
	if len(slots) != 1 || !slots[0].Start.Equal(mustTime(t, "2026-12-01T15:00:00Z")) || !slots[0].FreeUntil.Equal(mustTime(t, "2026-12-01T17:00:00Z")) {
		t.Errorf("suggestions = %v", slots)
	}
}

func TestFormatInstant_AlwaysNamesTheSameInstant(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// Modern offset: local time, as usual.
	modern := mustTime(t, "2026-12-01T12:00:00Z")
	if got := formatInstant(modern, paris); got != "2026-12-01T13:00:00+01:00" {
		t.Errorf("modern = %s", got)
	}
	// 1900: +00:09:21, which RFC3339 cannot express. Whatever is printed
	// must parse back to exactly the same instant.
	old := time.Date(1900, 6, 1, 9, 0, 0, 0, paris)
	got := formatInstant(old, paris)
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil || !parsed.Equal(old) {
		t.Errorf("formatInstant = %s, which parses to %v, not %v", got, parsed.UTC(), old.UTC())
	}
	if !strings.HasSuffix(got, "Z") {
		t.Errorf("an inexpressible offset should fall back to UTC, got %s", got)
	}
}

func TestCeilSecond(t *testing.T) {
	whole := mustTime(t, "2026-12-01T10:30:00Z")
	if got := ceilSecond(whole); !got.Equal(whole) {
		t.Errorf("whole second changed: %v", got)
	}
	if got := ceilSecond(whole.Add(900 * time.Millisecond)); !got.Equal(whole.Add(time.Second)) {
		t.Errorf("fraction not rounded up: %v", got)
	}
	if got := ceilSecond(whole.Add(time.Nanosecond)); !got.Equal(whole.Add(time.Second)) {
		t.Errorf("nanosecond not rounded up: %v", got)
	}
}

// Busy periods with fractional seconds are widened to whole seconds, so a
// suggestion printed without fractions never overlaps one.
func TestQueryBusySpans_FractionalSecondsAreWidened(t *testing.T) {
	svc := newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Calendars: map[string]calendar.FreeBusyCalendar{
				"primary": {Busy: []*calendar.TimePeriod{
					{Start: "2026-12-01T10:00:00.900Z", End: "2026-12-01T10:30:00.100Z"},
				}},
			},
		})
	})
	from := mustTime(t, "2026-12-01T09:00:00Z")
	to := mustTime(t, "2026-12-01T12:00:00Z")
	busy, _, _, err := queryBusySpans(context.Background(), svc, []*calendar.FreeBusyRequestItem{{Id: "primary"}}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(busy) != 1 || !busy[0].Start.Equal(mustTime(t, "2026-12-01T10:00:00Z")) || !busy[0].End.Equal(mustTime(t, "2026-12-01T10:30:01Z")) {
		t.Fatalf("busy = %v, want widened to 10:00:00 - 10:30:01", busy)
	}

	slots := suggestSlots(busy, from, to, 30*time.Minute, 0, 24*time.Hour, time.UTC, true, 5)
	if len(slots) != 2 {
		t.Fatalf("slots = %v", slots)
	}
	for _, s := range slots {
		for _, moment := range []time.Time{s.Start, s.End, s.FreeUntil} {
			if moment.Nanosecond() != 0 {
				t.Errorf("slot boundary %v has a fractional second", moment)
			}
		}
	}
	// The second slot starts after the real end of the busy period.
	if !slots[1].Start.After(mustTime(t, "2026-12-01T10:30:00.100Z")) {
		t.Errorf("slot starts at %v, inside the busy period", slots[1].Start)
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
	if _, _, _, err := queryBusySpans(context.Background(), svc, items, from, to); err == nil ||
		!strings.Contains(err.Error(), "none of the requested calendars") || !strings.Contains(err.Error(), "a@example.com (notFound)") {
		t.Errorf("error = %v, want unreadable-calendars error naming the calendar and reason", err)
	}

	// API errors keep their HTTP status for exit-code mapping.
	svc = newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"code":403,"message":"Request had insufficient authentication scopes."}}`))
	})
	_, _, _, err := queryBusySpans(context.Background(), svc, items, from, to)
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

	// A busy period that ends at or before its start makes that calendar's
	// availability unknown; none of its other periods are used either.
	svc = newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Calendars: map[string]calendar.FreeBusyCalendar{
				"a@example.com": {Busy: []*calendar.TimePeriod{
					{Start: "2026-12-01T09:00:00Z", End: "2026-12-01T10:00:00Z"},
					{Start: "2026-12-01T15:00:00Z", End: "2026-12-01T11:00:00Z"}, // reversed
				}},
				"b@example.com": {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T12:00:00Z", End: "2026-12-01T12:00:00Z"}}}, // empty
				"primary":       {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T13:00:00Z", End: "2026-12-01T14:00:00Z"}}},
			},
		})
	})
	busy, checked, unavailable, err := queryBusySpans(context.Background(), svc, []*calendar.FreeBusyRequestItem{{Id: "a@example.com"}, {Id: "b@example.com"}, {Id: "primary"}}, from, to)
	if err != nil {
		t.Fatalf("an invalid interval should mark the calendar unavailable, not fail: %v", err)
	}
	if strings.Join(checked, ",") != "primary" || len(unavailable) != 2 || !strings.Contains(unavailable[0]["reason"].(string), "invalid busy period") {
		t.Errorf("checked = %v, unavailable = %v", checked, unavailable)
	}
	if len(busy) != 1 || !busy[0].Start.Equal(mustTime(t, "2026-12-01T13:00:00Z")) {
		t.Errorf("busy = %v, want only the valid calendar's period", busy)
	}
	for _, span := range busy {
		if !span.End.After(span.Start) {
			t.Errorf("a non-positive busy span reached the scheduler: %v", span)
		}
	}

	// A malformed busy timestamp is handled like any unreadable calendar:
	// alone it leaves nothing to suggest from, and the error names it...
	svc = newSuggestTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&calendar.FreeBusyResponse{
			Calendars: map[string]calendar.FreeBusyCalendar{
				"a@example.com": {Busy: []*calendar.TimePeriod{{Start: "not-a-time", End: "2026-12-01T10:00:00Z"}}},
				"primary":       {Busy: []*calendar.TimePeriod{{Start: "2026-12-01T13:00:00Z", End: "2026-12-01T14:00:00Z"}}},
			},
		})
	})
	if _, _, _, err := queryBusySpans(context.Background(), svc, items, from, to); err == nil || !strings.Contains(err.Error(), "a@example.com (invalid busy period") {
		t.Errorf("error = %v, want the malformed calendar named", err)
	}
	// ...and next to a readable calendar it is reported as unavailable,
	// so --allow-partial can still return results.
	busy, checked, unavailable, err = queryBusySpans(context.Background(), svc, []*calendar.FreeBusyRequestItem{{Id: "a@example.com"}, {Id: "primary"}}, from, to)
	if err != nil || strings.Join(checked, ",") != "primary" || len(unavailable) != 1 || unavailable[0]["calendar"] != "a@example.com" || len(busy) != 1 {
		t.Errorf("checked = %v, unavailable = %v, busy = %v, err = %v", checked, unavailable, busy, err)
	}
}

// Monrovia skipped from 23:59:59 on 1972-01-06 straight to 00:44:30 on
// the 7th. A date-only --to 1972-01-06 must end at that exact instant: a
// boundary rounded up to the minute would include 30 seconds of the 7th.
func TestEndOfDateOnly_SubMinuteGap(t *testing.T) {
	loc, err := time.LoadLocation("Africa/Monrovia")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	_, offBefore := time.Date(1972, 1, 6, 12, 0, 0, 0, loc).Zone()
	_, offAfter := time.Date(1972, 1, 8, 12, 0, 0, 0, loc).Zone()
	if offBefore == offAfter || (offAfter-offBefore)%60 == 0 {
		t.Skipf("tzdata does not model the sub-minute change (offsets %d -> %d)", offBefore, offAfter)
	}

	end, err := endOfDateOnly("1972-01-06", loc)
	if err != nil {
		t.Fatal(err)
	}
	if last := end.Add(-time.Second).In(loc); last.Day() != 6 {
		t.Errorf("one second before the boundary reads %v, want still January 6", last)
	}
	if first := end.In(loc); first.Day() != 7 {
		t.Errorf("the boundary reads %v, want the first instant of January 7", first)
	}
}

// On 2010-11-07 St. John's clocks went from Sunday 00:01 back to Saturday
// 23:01, so part of Saturday evening repeats after Sunday has begun. A
// window for Sunday must contain only instants that read Sunday.
func TestDayWindows_TransitionAcrossMidnight(t *testing.T) {
	loc, err := time.LoadLocation("America/St_Johns")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	// Skip unless tzdata models the 00:01 rollback.
	before := time.Date(2010, 11, 7, 2, 30, 0, 0, time.UTC).In(loc) // Sunday 00:00 NDT
	after := before.Add(2 * time.Minute)
	if before.Day() != 7 || after.Day() != 6 {
		t.Skipf("tzdata does not model the midnight-crossing change (%v -> %v)", before, after)
	}

	// Saturday's 23:00-24:00 happens twice: 23:00-24:00 before the change
	// and 23:01-24:00 again after Sunday's first minute. Both belong to
	// Saturday's working hours, so no valid slot is lost.
	saturday := time.Date(2010, 11, 6, 12, 0, 0, 0, time.UTC)
	var total time.Duration
	satWindows := dayWindows(saturday, 23*time.Hour, 24*time.Hour, loc)
	for _, w := range satWindows {
		total += w.End.Sub(w.Start)
		for _, instant := range []time.Time{w.Start, w.End.Add(-time.Second)} {
			if l := instant.In(loc); l.Day() != 6 || l.Hour() != 23 {
				t.Errorf("Saturday window %v..%v contains %v", w.Start.In(loc), w.End.In(loc), l)
			}
		}
	}
	if len(satWindows) != 2 || total != time.Hour+59*time.Minute {
		t.Errorf("Saturday 23:00-24:00 windows = %v (total %v), want two windows totalling 1h59m", satWindows, total)
	}

	sunday := time.Date(2010, 11, 7, 12, 0, 0, 0, time.UTC)
	windows := dayWindows(sunday, 23*time.Hour, 24*time.Hour, loc)
	if len(windows) == 0 {
		t.Fatal("no window for Sunday 23:00-24:00")
	}
	for _, w := range windows {
		for _, instant := range []time.Time{w.Start, w.End.Add(-time.Second)} {
			if l := instant.In(loc); l.Day() != 7 || l.Hour() != 23 {
				t.Errorf("window %v..%v contains %v, want only Sunday 23:xx", w.Start.In(loc), w.End.In(loc), l)
			}
		}
	}
}

// --attendees "$UNSET" must not fall back to checking the default
// calendar alone and reporting that as everyone's availability.
func TestCalendarSuggestTime_RejectsAnExplicitlyEmptyList(t *testing.T) {
	flags := calendarSuggestTimeCmd.Flags()
	reset := func() {
		for name, def := range map[string]string{"from": "", "to": "", "attendees": "", "calendars": "primary"} {
			_ = flags.Set(name, def)
			flags.Lookup(name).Changed = false
		}
	}
	t.Cleanup(reset)

	for _, name := range []string{"attendees", "calendars"} {
		reset()
		_ = flags.Set("from", "2026-03-02")
		_ = flags.Set("to", "2026-03-03")
		_ = flags.Set(name, " , ")
		var err error
		out := captureStderrStr(t, func() { err = runCalendarSuggestTime(calendarSuggestTimeCmd, nil) })
		if err == nil || !strings.Contains(out+err.Error(), "--"+name+" was given but names no calendar") {
			t.Errorf("--%s with no IDs: err = %v, output = %q", name, err, out)
		}
	}
}

// A date-only --to includes the whole day, including the part of it that
// repeats after the next date has briefly begun (St. John's, 2010-11-06:
// Sunday 00:01 became Saturday 23:01).
func TestEndOfDateOnly_DateResumesAfterMidnight(t *testing.T) {
	loc, err := time.LoadLocation("America/St_Johns")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	before := time.Date(2010, 11, 7, 2, 30, 0, 0, time.UTC).In(loc)
	if before.Day() != 7 || before.Add(2*time.Minute).Day() != 6 {
		t.Skipf("tzdata does not model the midnight-crossing change")
	}

	end, err := endOfDateOnly("2010-11-06", loc)
	if err != nil {
		t.Fatal(err)
	}
	if last := end.Add(-time.Second).In(loc); last.Day() != 6 || last.Hour() != 23 || last.Minute() != 59 {
		t.Errorf("the search ends just after %v, want Saturday 23:59:59 (second occurrence)", last)
	}
	if first := end.In(loc); first.Day() != 7 {
		t.Errorf("the boundary reads %v, want Sunday", first)
	}

	// Through the slot search: the repeated Saturday evening is offered.
	from := startOfCivilDate(2010, 11, 6, loc)
	slots := suggestSlots(nil, from, end, 30*time.Minute, 23*time.Hour, 24*time.Hour, loc, true, 10)
	repeated := false
	for _, slot := range slots {
		if slot.Start.After(before) && slot.Start.In(loc).Day() == 6 {
			repeated = true
		}
	}
	if !repeated {
		t.Errorf("no slot in the repeated Saturday evening: %v", slots)
	}

	// An ordinary day still ends at the next midnight.
	plain, _ := endOfDateOnly("2010-06-10", loc)
	if l := plain.In(loc); l.Day() != 11 || l.Hour() != 0 || l.Minute() != 0 {
		t.Errorf("ordinary day ends at %v, want the next midnight", l)
	}
}

// Kwajalein moved from UTC+11 to UTC-12 on 1969-09-30, so 23 hours of
// that date happened twice. A local time inside the repeat resolves to
// its first occurrence, as documented, not to the one 23 hours later.
func TestCivilInstant_LongRepeatedInterval(t *testing.T) {
	loc, err := time.LoadLocation("Pacific/Kwajalein")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	_, before := time.Date(1969, 9, 29, 12, 0, 0, 0, loc).Zone()
	_, after := time.Date(1969, 10, 2, 12, 0, 0, 0, loc).Zone()
	if before-after != 23*3600 {
		t.Skipf("tzdata does not model the 23-hour rollback (offsets %d -> %d)", before, after)
	}

	got := civilInstant(1969, 9, 30, 14, 0, loc)
	if l := got.In(loc); l.Day() != 30 || l.Hour() != 14 || l.Minute() != 0 {
		t.Fatalf("resolved to %v, want a 14:00 on September 30", l)
	}
	if _, offset := got.In(loc).Zone(); offset != before {
		t.Errorf("resolved to the later occurrence (offset %d), want the first (offset %d)", offset, before)
	}
	// The other occurrence exists and is 23 hours later.
	if later := got.Add(23 * time.Hour).In(loc); later.Day() != 30 || later.Hour() != 14 {
		t.Errorf("expected the same wall time 23h later, got %v", later)
	}
}
