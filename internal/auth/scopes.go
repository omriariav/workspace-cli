package auth

import "strings"

const scopePrefix = "https://www.googleapis.com/auth/"

// ServiceScopes maps each service to its required Google API scopes.
var ServiceScopes = map[string][]string{
	"gmail":    {"gmail.readonly", "gmail.send", "gmail.modify", "gmail.settings.basic", "gmail.settings.sharing"},
	"calendar": {"calendar.readonly", "calendar.events", "calendar"},
	"drive":    {"drive.readonly", "drive.file", "drive"},
	"docs":     {"documents.readonly", "documents"},
	"sheets":   {"spreadsheets"},
	"slides":   {"presentations.readonly", "presentations"},
	"tasks":    {"tasks.readonly", "tasks"},
	"chat":     {"chat.spaces", "chat.messages", "chat.messages.create", "chat.memberships", "chat.messages.reactions", "chat.users.readstate"},
	"forms":    {"forms.responses.readonly", "forms.body", "forms.body.readonly"},
	"contacts": {"contacts.readonly", "contacts", "directory.readonly"},
	// `people` surfaces the People API for `gws people get`. Includes
	// `userinfo.profile` which the API requires for `people/me` requests.
	// Does NOT include admin.directory.group.* — those belong to `groups`.
	"people":        {"userinfo.profile", "contacts.readonly", "contacts", "directory.readonly"},
	"groups":        {"admin.directory.group.readonly", "admin.directory.group.member.readonly"},
	"keep":          {"keep.readonly", "keep"},
	"driveactivity": {"drive.activity.readonly"},
	// `events` covers Workspace Events API subscriptions with user
	// credentials: least-privilege readonly scopes per GA subscription
	// class (Chat space/user events incl. read-state, baseline Drive
	// file/permission events, Meet events).
	"events":   {"chat.spaces.readonly", "chat.memberships.readonly", "chat.messages.readonly", "chat.messages.reactions.readonly", "chat.users.readstate.readonly", "drive.metadata", "meetings.space.readonly", "meetings.space.created"},
	"userinfo": {"userinfo.email"},
}

// Note on `events`: the Workspace Events API accepts read scopes for Chat
// and Meet subscriptions but requires drive.metadata (not its .readonly
// variant) for Drive ones, so a read-only login covers Chat and Meet
// subscriptions only.
//
// ReadOnlyServiceScopes maps each service to the read-only scopes requested
// by `gws auth login --readonly`. Every service has at least one read-only
// scope; commands that create, modify, or delete data fail with an
// insufficient-scope error until write access is granted.
var ReadOnlyServiceScopes = map[string][]string{
	"gmail":         {"gmail.readonly"},
	"calendar":      {"calendar.readonly"},
	"drive":         {"drive.readonly"},
	"docs":          {"documents.readonly"},
	"sheets":        {"spreadsheets.readonly"},
	"slides":        {"presentations.readonly"},
	"tasks":         {"tasks.readonly"},
	"chat":          {"chat.spaces.readonly", "chat.messages.readonly", "chat.memberships.readonly", "chat.messages.reactions.readonly", "chat.users.readstate.readonly"},
	"forms":         {"forms.responses.readonly", "forms.body.readonly"},
	"contacts":      {"contacts.readonly", "directory.readonly"},
	"people":        {"userinfo.profile", "contacts.readonly", "directory.readonly"},
	"groups":        {"admin.directory.group.readonly", "admin.directory.group.member.readonly"},
	"keep":          {"keep.readonly"},
	"driveactivity": {"drive.activity.readonly"},
	"events":        {"chat.spaces.readonly", "chat.memberships.readonly", "chat.messages.readonly", "chat.messages.reactions.readonly", "chat.users.readstate.readonly", "meetings.space.readonly"},
	"userinfo":      {"userinfo.email"},
}

// Access levels recorded per service at login time.
const (
	AccessFull     = "full"
	AccessReadOnly = "readonly"
	// AccessPartial marks a service that holds at least one write scope
	// but not the complete full set (the user unticked some scopes).
	AccessPartial = "partial"
	// AccessPartialReadOnly marks a requested service that holds some of
	// its read-only scopes, but not all of them, and no write scope.
	AccessPartialReadOnly = "partial-readonly"
)

// AllScopes is the union of all service scopes. Computed at init time for backward compat.
var AllScopes = computeAllScopes()

func computeAllScopes() []string {
	seen := make(map[string]bool)
	var scopes []string
	// Deterministic order: iterate known service names
	order := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "contacts", "people", "groups", "keep", "driveactivity", "events", "userinfo"}
	for _, svc := range order {
		for _, s := range ServiceScopes[svc] {
			full := scopePrefix + s
			if !seen[full] {
				seen[full] = true
				scopes = append(scopes, full)
			}
		}
	}
	return scopes
}

// ScopesForServices returns the full scope URLs for the given service names.
// Always includes userinfo scopes.
func ScopesForServices(services []string) []string {
	return scopesFromMap(ServiceScopes, services)
}

// ReadOnlyScopesForServices returns the read-only scope URLs for the given
// service names. Always includes userinfo scopes.
func ReadOnlyScopesForServices(services []string) []string {
	return scopesFromMap(ReadOnlyServiceScopes, services)
}

func scopesFromMap(table map[string][]string, services []string) []string {
	seen := make(map[string]bool)
	var scopes []string

	// Always include userinfo
	addService := func(svc string) {
		if ss, ok := table[svc]; ok {
			for _, s := range ss {
				full := scopePrefix + s
				if !seen[full] {
					seen[full] = true
					scopes = append(scopes, full)
				}
			}
		}
	}

	addService("userinfo")
	for _, svc := range services {
		addService(svc)
	}

	return scopes
}

// ServiceForScope returns the service name for a given full scope URL.
// Returns empty string if the scope is not recognized. Iteration order is
// deterministic so aliases (e.g. `people` ↔ `contacts`) always resolve to
// the canonical name first.
func ServiceForScope(scope string) string {
	short := strings.TrimPrefix(scope, scopePrefix)
	order := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "contacts", "people", "groups", "keep", "driveactivity", "events", "userinfo"}
	for _, svc := range order {
		for _, s := range ServiceScopes[svc] {
			if s == short {
				return svc
			}
		}
	}
	// Read-only-only scopes (e.g. spreadsheets.readonly) are not listed in
	// ServiceScopes; resolve them through the read-only table.
	for _, svc := range order {
		for _, s := range ReadOnlyServiceScopes[svc] {
			if s == short {
				return svc
			}
		}
	}
	return ""
}

// ValidServiceNames returns all known service names.
func ValidServiceNames() []string {
	return []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "contacts", "people", "groups", "keep", "driveactivity", "events"}
}

// ValidateServices checks that all service names are recognized.
// Returns a list of unknown service names.
func ValidateServices(services []string) []string {
	valid := make(map[string]bool)
	for _, s := range ValidServiceNames() {
		valid[s] = true
	}
	valid["userinfo"] = true

	var unknown []string
	for _, s := range services {
		if !valid[s] {
			unknown = append(unknown, s)
		}
	}
	return unknown
}
