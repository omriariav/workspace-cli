package cmd

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
)

var calendarSuggestTimeCmd = &cobra.Command{
	Use:   "suggest-time",
	Short: "Suggest free meeting slots",
	Long: `Finds times when every given calendar is free.

Queries free/busy for the calendars, merges their busy periods, and returns
the free windows inside working hours that fit the requested duration. Each
result gives the earliest slot in a window and how long the window stays free.

A date-only --to includes that whole day. If any requested calendar cannot
be read the command fails, because a slot is only useful when everyone's
availability is known. With --allow-partial it suggests times from the
calendars it could read and lists the others under "unavailable".

Examples:
  gws calendar suggest-time --from 2026-12-01 --to 2026-12-05
  gws calendar suggest-time --from 2026-12-01 --to 2026-12-05 --attendees a@example.com,b@example.com --duration 45m
  gws calendar suggest-time --from "2026-12-01 13:00" --to "2026-12-01 18:00" --working-hours any --timezone Europe/London`,
	Args: cobra.NoArgs,
	RunE: runCalendarSuggestTime,
}

func init() {
	calendarCmd.AddCommand(calendarSuggestTimeCmd)

	calendarSuggestTimeCmd.Flags().String("from", "", "Start of the search range (required)")
	calendarSuggestTimeCmd.Flags().String("to", "", "End of the search range; a date-only value includes that day (required)")
	calendarSuggestTimeCmd.Flags().String("duration", "30m", "Meeting length, e.g. 30m, 1h, 1h30m, or minutes as a number")
	calendarSuggestTimeCmd.Flags().String("calendars", "primary", "Comma-separated calendar IDs that must be free")
	calendarSuggestTimeCmd.Flags().String("attendees", "", "Comma-separated attendee emails that must also be free")
	calendarSuggestTimeCmd.Flags().String("working-hours", "09:00-17:00", "Daily window to search, HH:MM-HH:MM, or 'any' for the whole day")
	calendarSuggestTimeCmd.Flags().String("timezone", "", "IANA timezone for working hours and output (default: local)")
	calendarSuggestTimeCmd.Flags().Int("max", 5, "Maximum number of suggestions")
	calendarSuggestTimeCmd.Flags().Bool("include-weekends", false, "Also search Saturdays and Sundays")
	calendarSuggestTimeCmd.Flags().Bool("allow-partial", false, "Suggest times even if some calendars could not be read (they are listed under \"unavailable\")")
	calendarSuggestTimeCmd.MarkFlagRequired("from")
	calendarSuggestTimeCmd.MarkFlagRequired("to")
}

// suggestTimeCalendarService returns the Calendar client suggest-time
// uses. A variable so tests can point the whole command at a mock server.
var suggestTimeCalendarService = func(ctx context.Context) (*calendar.Service, error) {
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return nil, err
	}
	return factory.Calendar()
}

// timeSpan is a half-open interval [Start, End).
type timeSpan struct {
	Start, End time.Time
}

// suggestion is one free window that fits the requested duration.
type suggestion struct {
	Start     time.Time // earliest slot start
	End       time.Time // Start + duration
	FreeUntil time.Time // end of the free window
}

// parseSuggestDuration accepts a Go duration ("45m", "1h30m") or a bare
// number of minutes ("45").
func parseSuggestDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		if n <= 0 {
			return 0, fmt.Errorf("must be positive")
		}
		// Bound the minute count before multiplying so a huge value
		// cannot overflow into a negative duration.
		if n > int(maxSuggestDuration/time.Minute) {
			return 0, fmt.Errorf("must be at most 24h")
		}
		return time.Duration(n) * time.Minute, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("use a duration like 30m or 1h30m")
	}
	if d <= 0 {
		return 0, fmt.Errorf("must be positive")
	}
	if d > maxSuggestDuration {
		return 0, fmt.Errorf("must be at most 24h")
	}
	// Slots are reported in whole minutes and RFC3339 seconds.
	if d%time.Minute != 0 {
		return 0, fmt.Errorf("must be a whole number of minutes")
	}
	return d, nil
}

// maxSuggestRange is the longest span suggest-time will search.
const maxSuggestRange = 92 * 24 * time.Hour

// maxFreeBusyCalendars is the most calendars a single Calendar free/busy
// query will report on, counting the members that groups expand into.
const maxFreeBusyCalendars = 50

// maxGroupExpansion is the most members one group may expand into.
const maxGroupExpansion = 100

// maxSuggestDuration is the longest meeting suggest-time will look for.
const maxSuggestDuration = 24 * time.Hour

// parseWorkingHours parses "HH:MM-HH:MM" into offsets from midnight.
// "any" (or an empty value) means the whole day.
func parseWorkingHours(s string) (start, end time.Duration, err error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "any" {
		return 0, 24 * time.Hour, nil
	}
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("use HH:MM-HH:MM or 'any'")
	}
	parse := func(v string) (time.Duration, error) {
		v = strings.TrimSpace(v)
		if v == "24:00" {
			return 24 * time.Hour, nil
		}
		t, err := time.Parse("15:04", v)
		if err != nil {
			return 0, fmt.Errorf("invalid time %q, use HH:MM", v)
		}
		return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
	}
	if start, err = parse(parts[0]); err != nil {
		return 0, 0, err
	}
	if end, err = parse(parts[1]); err != nil {
		return 0, 0, err
	}
	if end <= start {
		return 0, 0, fmt.Errorf("end must be after start")
	}
	return start, end, nil
}

// parseTimeIn parses RFC3339, "YYYY-MM-DD HH:MM", or "YYYY-MM-DD" with
// zone-less values interpreted in loc. dateOnly reports the last form.
func parseTimeIn(s string, loc *time.Location) (t time.Time, dateOnly bool, err error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, false, nil
	}
	// Zone-less values are read as wall-clock time in loc through
	// civilInstant, so a time that a DST change skips moves forward to the
	// first real instant instead of backward, before what was asked for.
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return civilInstant(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), loc), false, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return startOfCivilDate(t.Year(), t.Month(), t.Day(), loc), true, nil
	}
	return time.Time{}, false, fmt.Errorf("unrecognized time format: %s (use RFC3339, 'YYYY-MM-DD HH:MM', or 'YYYY-MM-DD')", s)
}

// civilInstant returns the instant at which the wall clock in loc reads
// the given civil date and time. Overflow is normalised (day 32, hour 24).
//
// A DST change can skip wall-clock times; for those time.Date picks an
// instant on one side of the gap, which can read earlier than requested.
// This returns the first instant whose wall clock is at or after the
// requested time instead, so a boundary never moves before the time asked
// for. Times that occur twice resolve to the first occurrence.
func civilInstant(year int, month time.Month, day, hour, min int, loc *time.Location) time.Time {
	// The requested wall time, normalised, with UTC as a neutral label.
	want := time.Date(year, month, day, hour, min, 0, 0, time.UTC)
	wall := func(t time.Time) time.Time {
		l := t.In(loc)
		return time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), l.Minute(), 0, 0, time.UTC)
	}

	t := time.Date(year, month, day, hour, min, 0, 0, loc)
	if wall(t).Equal(want) {
		// The time exists. If a fall-back change makes it occur twice,
		// time.Date may pick either one; take the earliest explicitly.
		for back := 3 * time.Hour; back > 0; back -= 15 * time.Minute {
			if earlier := t.Add(-back); wall(earlier).Equal(want) {
				return earlier
			}
		}
		return t
	}
	// The requested time does not exist: scan around it. The window is
	// wide enough for a zone that skipped a whole calendar day (Samoa
	// crossed the date line by dropping 2011-12-30).
	for c := t.Add(-26 * time.Hour); !c.After(t.Add(26 * time.Hour)); c = c.Add(time.Minute) {
		if !wall(c).Before(want) {
			return c
		}
	}
	return t
}

// startOfCivilDate returns the first instant of the given civil date in
// loc, including when a DST change skips midnight.
func startOfCivilDate(year int, month time.Month, day int, loc *time.Location) time.Time {
	return civilInstant(year, month, day, 0, 0, loc)
}

// endOfDateOnly returns the exclusive end of a "YYYY-MM-DD" day in loc:
// the first instant of the following civil date. It works from the
// written date, not from an instant derived from it, so a date the zone
// skipped entirely cannot shift the boundary onto another day.
func endOfDateOnly(date string, loc *time.Location) (time.Time, error) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(date))
	if err != nil {
		return time.Time{}, err
	}
	return startOfCivilDate(d.Year(), d.Month(), d.Day()+1, loc), nil
}

// mergeSpans sorts spans and merges the ones that overlap or touch.
func mergeSpans(spans []timeSpan) []timeSpan {
	if len(spans) == 0 {
		return nil
	}
	sorted := append([]timeSpan(nil), spans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })
	merged := []timeSpan{sorted[0]}
	for _, s := range sorted[1:] {
		last := &merged[len(merged)-1]
		if !s.Start.After(last.End) {
			if s.End.After(last.End) {
				last.End = s.End
			}
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

// suggestSlots returns up to max free windows of at least dur between from
// and to, restricted to the daily [whStart, whEnd) window in loc.
//
// Days are tracked as civil dates (year, month, day) rather than by adding
// 24h or stepping a local-midnight timestamp, so a DST change — including
// one that skips midnight — cannot shift or drop a day's working window.
// Windows that touch (e.g. whole-day windows on consecutive days) are
// merged, so a slot may span midnight when working hours allow it.
func suggestSlots(busy []timeSpan, from, to time.Time, dur, whStart, whEnd time.Duration, loc *time.Location, includeWeekends bool, max int) []suggestion {
	busy = mergeSpans(busy)

	// Start one day early: a window ending at 24:00 belongs to the
	// previous civil date but can still overlap the range.
	first := from.In(loc)
	y, m, d := first.Year(), first.Month(), first.Day()-1

	var windows []timeSpan
	for i := 0; ; i++ {
		// Noon UTC on the civil date: a stable handle for the weekday and
		// for normalising day overflow, independent of loc's offsets.
		civil := time.Date(y, m, d+i, 12, 0, 0, 0, time.UTC)
		if !atDayOffset(civil, 0, loc).Before(to) {
			break
		}
		if wd := civil.Weekday(); !includeWeekends && (wd == time.Saturday || wd == time.Sunday) {
			continue
		}
		for _, w := range dayWindows(civil, whStart, whEnd, loc) {
			if w.Start.Before(from) {
				w.Start = from
			}
			if w.End.After(to) {
				w.End = to
			}
			if w.End.After(w.Start) {
				windows = append(windows, w)
			}
		}
	}

	var out []suggestion
	for _, w := range mergeSpans(windows) {
		cursor := w.Start
		emit := func(gapEnd time.Time) {
			if len(out) < max && gapEnd.Sub(cursor) >= dur {
				out = append(out, suggestion{Start: cursor, End: cursor.Add(dur), FreeUntil: gapEnd})
			}
		}
		for _, b := range busy {
			if !b.End.After(cursor) {
				continue
			}
			if !b.Start.Before(w.End) {
				break
			}
			if b.Start.After(cursor) {
				emit(b.Start)
			}
			cursor = b.End
		}
		if cursor.Before(w.End) {
			emit(w.End)
		}
		if len(out) >= max {
			break
		}
	}
	return out
}

// dayWindows returns the instants of one civil date whose wall-clock time
// falls inside [whStart, whEnd). On an ordinary day that is one interval.
// On a day with a DST change it is computed minute by minute, because the
// set is not a single interval: when clocks fall back, an hour of wall
// time occurs twice, and the stretch between the two occurrences of the
// working hours lies outside them.
func dayWindows(civil time.Time, whStart, whEnd time.Duration, loc *time.Location) []timeSpan {
	dayStart := atDayOffset(civil, 0, loc)
	dayEnd := atDayOffset(civil, 24*time.Hour, loc)

	_, startOffset := dayStart.In(loc).Zone()
	_, endOffset := dayEnd.Add(-time.Second).In(loc).Zone()
	if startOffset == endOffset && dayEnd.Sub(dayStart) == 24*time.Hour {
		return []timeSpan{{Start: atDayOffset(civil, whStart, loc), End: atDayOffset(civil, whEnd, loc)}}
	}

	var spans []timeSpan
	open := false
	for c := dayStart; c.Before(dayEnd); c = c.Add(time.Minute) {
		l := c.In(loc)
		wall := time.Duration(l.Hour())*time.Hour + time.Duration(l.Minute())*time.Minute
		inside := wall >= whStart && wall < whEnd
		switch {
		case inside && !open:
			spans = append(spans, timeSpan{Start: c, End: c.Add(time.Minute)})
			open = true
		case inside:
			spans[len(spans)-1].End = c.Add(time.Minute)
		default:
			open = false
		}
	}
	return spans
}

// atDayOffset returns the instant `offset` of wall-clock time after
// midnight on the civil date of day, in loc. An offset of 24h is the start
// of the next day. Boundaries that fall in a DST gap move forward to the
// first real instant (see civilInstant).
func atDayOffset(day time.Time, offset time.Duration, loc *time.Location) time.Time {
	h := int(offset / time.Hour)
	m := int((offset % time.Hour) / time.Minute)
	return civilInstant(day.Year(), day.Month(), day.Day(), h, m, loc)
}

// formatInstant renders t as RFC3339 in loc. RFC3339 offsets have only
// hours and minutes, so where loc's offset at t has a seconds component
// (historical local mean time) the result would name a different instant;
// those are rendered in UTC instead.
func formatInstant(t time.Time, loc *time.Location) string {
	local := t.In(loc)
	if _, offset := local.Zone(); offset%60 != 0 {
		return t.UTC().Format(time.RFC3339)
	}
	return local.Format(time.RFC3339)
}

// ceilSecond rounds t up to the next whole second (unchanged if it
// already is one).
func ceilSecond(t time.Time) time.Time {
	if truncated := t.Truncate(time.Second); !truncated.Equal(t) {
		return truncated.Add(time.Second)
	}
	return t
}

// splitList splits a comma-separated list, dropping blanks.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func runCalendarSuggestTime(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")
	durStr, _ := cmd.Flags().GetString("duration")
	calendarsStr, _ := cmd.Flags().GetString("calendars")
	attendeesStr, _ := cmd.Flags().GetString("attendees")
	whStr, _ := cmd.Flags().GetString("working-hours")
	tzStr, _ := cmd.Flags().GetString("timezone")
	max, _ := cmd.Flags().GetInt("max")
	includeWeekends, _ := cmd.Flags().GetBool("include-weekends")
	allowPartial, _ := cmd.Flags().GetBool("allow-partial")

	loc := time.Local
	if tzStr != "" {
		l, err := time.LoadLocation(tzStr)
		if err != nil {
			return usageErrorf("invalid --timezone %q: use an IANA name such as Europe/London", tzStr)
		}
		loc = l
	}
	from, _, err := parseTimeIn(fromStr, loc)
	if err != nil {
		return usageErrorf("invalid --from: %s", err.Error())
	}
	to, toDateOnly, err := parseTimeIn(toStr, loc)
	if err != nil {
		return usageErrorf("invalid --to: %s", err.Error())
	}
	if toDateOnly {
		if to, err = endOfDateOnly(toStr, loc); err != nil {
			return usageErrorf("invalid --to: %s", err.Error())
		}
	}
	// Work in whole seconds throughout: the free/busy query and the output
	// are RFC3339 without fractions. Shrinking the range inward keeps every
	// suggestion inside what was asked for and inside what was queried.
	from = ceilSecond(from)
	to = to.Truncate(time.Second)
	if !to.After(from) {
		return usageErrorf("--to must be after --from")
	}
	// Bounds the work: every day in the range is examined, however small
	// --max is.
	if to.Sub(from) > maxSuggestRange {
		return usageErrorf("the search range is too long: at most %d days between --from and --to", int(maxSuggestRange/(24*time.Hour)))
	}
	dur, err := parseSuggestDuration(durStr)
	if err != nil {
		return usageErrorf("invalid --duration %q: %s", durStr, err.Error())
	}
	whStart, whEnd, err := parseWorkingHours(whStr)
	if err != nil {
		return usageErrorf("invalid --working-hours %q: %s", whStr, err.Error())
	}
	if max < 1 {
		return usageErrorf("--max must be >= 1")
	}

	seen := map[string]bool{}
	var items []*calendar.FreeBusyRequestItem
	for _, id := range append(splitList(calendarsStr), splitList(attendeesStr)...) {
		if !seen[id] {
			seen[id] = true
			items = append(items, &calendar.FreeBusyRequestItem{Id: id})
		}
	}
	if len(items) == 0 {
		return usageErrorf("at least one calendar is required: set --calendars or --attendees")
	}
	if len(items) > maxFreeBusyCalendars {
		return usageErrorf("too many calendars: %d given, but one free/busy query covers at most %d (--calendars and --attendees combined)", len(items), maxFreeBusyCalendars)
	}

	svc, err := suggestTimeCalendarService(context.Background())
	if err != nil {
		return p.PrintError(err)
	}

	busy, checked, unavailable, err := queryBusySpans(svc, items, from, to)
	if err != nil {
		return p.PrintError(err)
	}
	if err := unreadableCalendarsError(unavailable, allowPartial); err != nil {
		return p.PrintError(err)
	}

	slots := suggestSlots(busy, from, to, dur, whStart, whEnd, loc, includeWeekends, max)
	suggestions := make([]map[string]interface{}, 0, len(slots))
	for _, s := range slots {
		suggestions = append(suggestions, map[string]interface{}{
			"start":        formatInstant(s.Start, loc),
			"end":          formatInstant(s.End, loc),
			"free_until":   formatInstant(s.FreeUntil, loc),
			"free_minutes": int(s.FreeUntil.Sub(s.Start) / time.Minute),
		})
	}

	return p.Print(map[string]interface{}{
		"time_min":          formatInstant(from, loc),
		"time_max":          formatInstant(to, loc),
		"timezone":          loc.String(),
		"duration_minutes":  int(dur / time.Minute),
		"calendars_checked": checked,
		"unavailable":       unavailable,
		"partial":           len(unavailable) > 0,
		"suggestions":       suggestions,
		"count":             len(suggestions),
	})
}

// firstErrorReason returns the reason of the first usable entry in a
// free/busy error list. The list holds pointers, so a null entry in the
// response arrives as nil.
func firstErrorReason(errs []*calendar.Error) string {
	for _, e := range errs {
		if e != nil && e.Reason != "" {
			return e.Reason
		}
	}
	return "unspecified error"
}

// unreadableCalendarsError returns an error when some requested calendars
// could not be read and partial results were not asked for: a suggestion
// is only meaningful if everyone's availability is known.
func unreadableCalendarsError(unavailable []map[string]interface{}, allowPartial bool) error {
	if len(unavailable) == 0 || allowPartial {
		return nil
	}
	names := make([]string, 0, len(unavailable))
	for _, u := range unavailable {
		names = append(names, fmt.Sprintf("%v (%v)", u["calendar"], u["reason"]))
	}
	return fmt.Errorf("could not read free/busy for %s; no suggestions made. Fix the calendar IDs or sharing, or pass --allow-partial to suggest times from the calendars that could be read", strings.Join(names, ", "))
}

// queryBusySpans runs the free/busy query and returns the busy periods of
// the calendars that could be read, the IDs of those calendars, and the
// calendars that could not be read (with the reason).
func queryBusySpans(svc *calendar.Service, items []*calendar.FreeBusyRequestItem, from, to time.Time) (busy []timeSpan, checked []string, unavailable []map[string]interface{}, err error) {
	resp, err := svc.Freebusy.Query(&calendar.FreeBusyRequest{
		// Sent in UTC, so the instant is exact whatever the local offset
		// (RFC3339 cannot express offsets with a seconds component).
		TimeMin: from.UTC().Format(time.RFC3339),
		TimeMax: to.UTC().Format(time.RFC3339),
		Items:   items,
		// Stated explicitly rather than left to server defaults: a group
		// is expanded into at most this many members, and at most this
		// many calendars are reported on in total, members included. A
		// group or request beyond these comes back with an error entry,
		// which is reported as unavailable.
		GroupExpansionMax:    maxGroupExpansion,
		CalendarExpansionMax: maxFreeBusyCalendars,
	}).Do()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to query free/busy: %w", err)
	}

	checked = []string{}
	unavailable = []map[string]interface{}{}
	markUnavailable := func(id, reason string) {
		unavailable = append(unavailable, map[string]interface{}{"calendar": id, "reason": reason})
	}
	// readCalendars counts the individual calendars whose availability was
	// actually read, including members of a group that is unavailable as
	// a whole because another member could not be read.
	readCalendars := 0
	// addCalendar records one calendar's busy periods. It reports false,
	// with the reason, when that calendar's availability is unknown.
	addCalendar := func(id string) (bool, string) {
		fb, ok := resp.Calendars[id]
		if !ok {
			return false, "missing from response"
		}
		if len(fb.Errors) > 0 {
			return false, firstErrorReason(fb.Errors)
		}
		// A readable calendar always carries a busy list, empty when it
		// is free ("busy": []). No list at all — a null or empty entry —
		// is not the same as free: nothing was reported, so availability
		// is unknown.
		if fb.Busy == nil {
			return false, "no availability data returned"
		}
		// Collected separately and added only once the whole calendar has
		// been validated, so a bad entry never leaves half of it behind.
		spans := make([]timeSpan, 0, len(fb.Busy))
		for _, period := range fb.Busy {
			// The list holds pointers, so a null entry in the response
			// arrives as nil.
			if period == nil {
				return false, "invalid busy period (null)"
			}
			start, err1 := time.Parse(time.RFC3339, period.Start)
			end, err2 := time.Parse(time.RFC3339, period.End)
			// A period that cannot be parsed, or that ends at or before its
			// start, says nothing usable (and a reversed one would move
			// the slot search backward). Treat this calendar's
			// availability as unknown rather than guess; the other
			// calendars are unaffected.
			if err1 != nil || err2 != nil || !end.After(start) {
				return false, fmt.Sprintf("invalid busy period %s - %s", period.Start, period.End)
			}
			// Widen busy periods to whole seconds so a slot reported
			// without fractions can never sit inside one.
			spans = append(spans, timeSpan{Start: start.Truncate(time.Second), End: ceilSecond(end)})
		}
		busy = append(busy, spans...)
		readCalendars++
		return true, ""
	}

	for _, item := range items {
		// A group address is answered under resp.Groups: the group lists
		// its member calendars, whose busy periods are in resp.Calendars
		// under their own IDs. The group is readable only if every member
		// is; otherwise part of the group's availability is unknown.
		if group, isGroup := resp.Groups[item.Id]; isGroup {
			if len(group.Errors) > 0 {
				markUnavailable(item.Id, firstErrorReason(group.Errors))
				continue
			}
			if len(group.Calendars) == 0 {
				markUnavailable(item.Id, "group has no readable members")
				continue
			}
			// Every member is processed even after one turns out to be
			// unreadable: the busy time of the members that can be read
			// must still block slots when --allow-partial continues.
			var unreadable []string
			for _, member := range group.Calendars {
				if ok, reason := addCalendar(member); !ok {
					unreadable = append(unreadable, fmt.Sprintf("%s: %s", member, reason))
				}
			}
			if len(unreadable) > 0 {
				markUnavailable(item.Id, "group member "+strings.Join(unreadable, "; group member "))
			} else {
				checked = append(checked, item.Id)
			}
			continue
		}

		if ok, reason := addCalendar(item.Id); !ok {
			markUnavailable(item.Id, reason)
			continue
		}
		checked = append(checked, item.Id)
	}
	if readCalendars == 0 {
		// Never suggest from zero calendars, whatever --allow-partial says;
		// name each one so a mistyped ID or missing sharing is visible.
		names := make([]string, 0, len(unavailable))
		for _, u := range unavailable {
			names = append(names, fmt.Sprintf("%v (%v)", u["calendar"], u["reason"]))
		}
		return nil, nil, nil, fmt.Errorf("none of the requested calendars could be read: %s; no suggestions made", strings.Join(names, ", "))
	}

	return busy, checked, unavailable, nil
}
