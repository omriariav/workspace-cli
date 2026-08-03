package hooks

import (
	"fmt"
	"strings"
	"time"
)

// FormatTimeContext intentionally uses Go's English weekday names rather than
// locale-sensitive shell date output. The returned string is later encoded by
// encoding/json in the host envelope, never interpolated into JSON.
func FormatTimeContext(now time.Time, configured string, local *time.Location) (string, string) {
	location, iana, diagnostic := resolveTimezone(configured, local)
	localNow := now.In(location)
	abbreviation, offset := localNow.Zone()
	offsetText := formatOffset(offset)
	zone := iana
	if zone == "" {
		zone = "system-local"
	}
	context := fmt.Sprintf("Current time: %s, %s %s (%s) [%s]. Treat this as authoritative for relative-time reasoning.", localNow.Weekday().String(), localNow.Format("2006-01-02 15:04:05"), abbreviation, offsetText, zone)
	return context, diagnostic
}

func resolveTimezone(configured string, local *time.Location) (*time.Location, string, string) {
	if configured = strings.TrimSpace(configured); configured != "" {
		location, err := time.LoadLocation(configured)
		if err == nil {
			return location, configured, ""
		}
		if local == nil {
			local = time.Local
		}
		return local, systemIANA(local), fmt.Sprintf("Configured timezone %q could not be resolved; using the system timezone", configured)
	}
	if local == nil {
		local = time.Local
	}
	return local, systemIANA(local), ""
}

func systemIANA(location *time.Location) string {
	if location == nil {
		return ""
	}
	name := location.String()
	if name == "" || name == "Local" || !strings.Contains(name, "/") {
		return ""
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ""
	}
	return name
}

func formatOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	return fmt.Sprintf("%s%02d:%02d", sign, seconds/3600, (seconds%3600)/60)
}
