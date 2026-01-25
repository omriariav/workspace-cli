package auth

// AllScopes contains all Google API scopes required by gws.
// We request all scopes upfront so users only need to authenticate once.
var AllScopes = []string{
	// Gmail
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.modify",

	// Calendar
	"https://www.googleapis.com/auth/calendar.readonly",
	"https://www.googleapis.com/auth/calendar.events",

	// Drive
	"https://www.googleapis.com/auth/drive.readonly",
	"https://www.googleapis.com/auth/drive.file",

	// Docs
	"https://www.googleapis.com/auth/documents.readonly",
	"https://www.googleapis.com/auth/documents",

	// Sheets
	"https://www.googleapis.com/auth/spreadsheets.readonly",

	// Slides
	"https://www.googleapis.com/auth/presentations.readonly",
	"https://www.googleapis.com/auth/presentations",

	// Tasks
	"https://www.googleapis.com/auth/tasks.readonly",
	"https://www.googleapis.com/auth/tasks",

	// Chat (requires additional setup)
	"https://www.googleapis.com/auth/chat.spaces.readonly",
	"https://www.googleapis.com/auth/chat.messages",
	"https://www.googleapis.com/auth/chat.messages.create",

	// Forms
	"https://www.googleapis.com/auth/forms.responses.readonly",

	// User info (for status display)
	"https://www.googleapis.com/auth/userinfo.email",
}
