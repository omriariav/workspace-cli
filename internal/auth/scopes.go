package auth

import "strings"

const scopePrefix = "https://www.googleapis.com/auth/"

// ServiceScopes maps each service to its required Google API scopes.
var ServiceScopes = map[string][]string{
	"gmail":    {"gmail.readonly", "gmail.send", "gmail.modify"},
	"calendar": {"calendar.readonly", "calendar.events"},
	"drive":    {"drive.readonly", "drive.file"},
	"docs":     {"documents.readonly", "documents"},
	"sheets":   {"spreadsheets"},
	"slides":   {"presentations.readonly", "presentations"},
	"tasks":    {"tasks.readonly", "tasks"},
	"chat":     {"chat.spaces.readonly", "chat.messages", "chat.messages.create", "chat.memberships.readonly", "chat.messages.reactions", "chat.messages.reactions.create", "chat.users.readstate", "chat.users.readstate.readonly", "chat.spaces", "chat.memberships"},
	"forms":    {"forms.responses.readonly"},
	"contacts": {"contacts.readonly", "contacts"},
	"userinfo": {"userinfo.email"},
}

// AllScopes is the union of all service scopes. Computed at init time for backward compat.
var AllScopes = computeAllScopes()

func computeAllScopes() []string {
	seen := make(map[string]bool)
	var scopes []string
	// Deterministic order: iterate known service names
	order := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "contacts", "userinfo"}
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
	seen := make(map[string]bool)
	var scopes []string

	// Always include userinfo
	addService := func(svc string) {
		if ss, ok := ServiceScopes[svc]; ok {
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
// Returns empty string if the scope is not recognized.
func ServiceForScope(scope string) string {
	short := strings.TrimPrefix(scope, scopePrefix)
	for svc, ss := range ServiceScopes {
		for _, s := range ss {
			if s == short {
				return svc
			}
		}
	}
	return ""
}

// ValidServiceNames returns all known service names.
func ValidServiceNames() []string {
	return []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "contacts"}
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
