package cmd

import (
	"testing"
	"time"
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
	for _, in := range []string{"", "0", "-5", "-30m", "soon"} {
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
