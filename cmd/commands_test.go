package cmd

import (
	"testing"

	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
)

// TestRootCommand tests the root command configuration
func TestRootCommand_Flags(t *testing.T) {
	formatFlag := rootCmd.PersistentFlags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("expected --format flag to exist")
	}
	if formatFlag.DefValue != "json" {
		t.Errorf("expected --format default to be 'json', got '%s'", formatFlag.DefValue)
	}

	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Fatal("expected --config flag to exist")
	}

	quietFlag := rootCmd.PersistentFlags().Lookup("quiet")
	if quietFlag == nil {
		t.Fatal("expected --quiet flag to exist")
	}
	if quietFlag.DefValue != "false" {
		t.Errorf("expected --quiet default to be 'false', got '%s'", quietFlag.DefValue)
	}
}

func TestGetPrinter_QuietMode(t *testing.T) {
	// Save and restore quiet state
	origQuiet := quiet
	defer func() { quiet = origQuiet }()

	quiet = true
	p := GetPrinter()
	if _, ok := p.(*printer.NullPrinter); !ok {
		t.Errorf("expected NullPrinter when quiet=true, got %T", p)
	}
}

func TestGetPrinter_NormalMode(t *testing.T) {
	// Save and restore quiet state
	origQuiet := quiet
	defer func() { quiet = origQuiet }()

	quiet = false
	p := GetPrinter()
	if _, ok := p.(*printer.NullPrinter); ok {
		t.Error("expected non-NullPrinter when quiet=false")
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	subcommands := rootCmd.Commands()
	if len(subcommands) == 0 {
		t.Error("expected root command to have subcommands")
	}

	// Check for expected subcommands
	expected := []string{"auth", "gmail", "calendar", "tasks", "drive", "docs", "sheets", "slides", "chat", "forms", "search", "contacts"}
	for _, name := range expected {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand '%s' not found", name)
		}
	}
}

// TestAuthCommands tests auth command structure
func TestAuthCommands(t *testing.T) {
	// Find auth command
	var authCommand *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "auth" {
			authCommand = cmd
			break
		}
	}

	if authCommand == nil {
		t.Fatal("auth command not found")
	}

	// Check subcommands
	subcommands := authCommand.Commands()
	expectedSubs := []string{"login", "logout", "status"}
	for _, name := range expectedSubs {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected auth subcommand '%s' not found", name)
		}
	}
}

// TestGmailCommands tests gmail command structure
func TestGmailCommands(t *testing.T) {
	tests := []struct {
		name    string
		use     string
		hasArgs bool
	}{
		{"list", "list", false},
		{"read", "read <message-id>", true},
		{"send", "send", false},
		{"labels", "labels", false},
		{"label", "label <message-id>", true},
		{"archive", "archive <message-id>", true},
		{"trash", "trash <message-id>", true},
		{"archive-thread", "archive-thread <thread-id>", true},
		{"thread", "thread <thread-id>", true},
		{"event-id", "event-id <message-id>", true},
		{"reply", "reply <message-id>", true},
		{"untrash", "untrash <message-id>", true},
		{"delete", "delete <message-id>", true},
		{"batch-modify", "batch-modify", false},
		{"batch-delete", "batch-delete", false},
		{"trash-thread", "trash-thread <thread-id>", true},
		{"untrash-thread", "untrash-thread <thread-id>", true},
		{"delete-thread", "delete-thread <thread-id>", true},
		{"label-info", "label-info", false},
		{"create-label", "create-label", false},
		{"update-label", "update-label", false},
		{"delete-label", "delete-label", false},
		{"drafts", "drafts", false},
		{"draft", "draft", false},
		{"create-draft", "create-draft", false},
		{"update-draft", "update-draft", false},
		{"send-draft", "send-draft", false},
		{"delete-draft", "delete-draft", false},
		{"attachment", "attachment", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(gmailCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
			if cmd.Use != tt.use {
				t.Errorf("expected Use '%s', got '%s'", tt.use, cmd.Use)
			}
		})
	}
}

func TestGmailListCommand_Flags(t *testing.T) {
	cmd := findSubcommand(gmailCmd, "list")
	if cmd == nil {
		t.Fatal("gmail list command not found")
	}

	maxFlag := cmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Error("expected --max flag")
	}

	queryFlag := cmd.Flags().Lookup("query")
	if queryFlag == nil {
		t.Error("expected --query flag")
	}
}

func TestGmailSendCommand_Flags(t *testing.T) {
	cmd := findSubcommand(gmailCmd, "send")
	if cmd == nil {
		t.Fatal("gmail send command not found")
	}

	requiredFlags := []string{"to", "subject", "body"}
	for _, flag := range requiredFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}

	optionalFlags := []string{"cc", "bcc", "thread-id", "reply-to-message-id"}
	for _, flag := range optionalFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// TestCalendarCommands tests calendar command structure
func TestCalendarCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"list"},
		{"events"},
		{"create"},
		{"update"},
		{"delete"},
		{"rsvp"},
		{"get"},
		{"quick-add"},
		{"instances"},
		{"move"},
		{"get-calendar"},
		{"create-calendar"},
		{"update-calendar"},
		{"delete-calendar"},
		{"clear"},
		{"subscribe"},
		{"unsubscribe"},
		{"calendar-info"},
		{"update-subscription"},
		{"acl"},
		{"share"},
		{"unshare"},
		{"update-acl"},
		{"freebusy"},
		{"colors"},
		{"settings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(calendarCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

func TestCalendarEventsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(calendarCmd, "events")
	if cmd == nil {
		t.Fatal("calendar events command not found")
	}

	flags := []string{"days", "calendar-id", "max"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

func TestCalendarCreateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(calendarCmd, "create")
	if cmd == nil {
		t.Fatal("calendar create command not found")
	}

	flags := []string{"title", "start", "end", "attendees", "description", "calendar-id"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// TestTasksCommands tests tasks command structure
func TestTasksCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"lists"},
		{"list"},
		{"create"},
		{"update"},
		{"complete"},
		{"list-info"},
		{"create-list"},
		{"update-list"},
		{"delete-list"},
		{"get"},
		{"delete"},
		{"move"},
		{"clear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(tasksCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestDriveCommands tests drive command structure
func TestDriveCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"list"},
		{"search"},
		{"info"},
		{"download"},
		{"comments"},
		{"upload"},
		{"create-folder"},
		{"move"},
		{"delete"},
		{"copy"},
		{"permissions"},
		{"share"},
		{"unshare"},
		{"permission"},
		{"update-permission"},
		{"revisions"},
		{"revision"},
		{"delete-revision"},
		{"replies"},
		{"reply"},
		{"get-reply"},
		{"delete-reply"},
		{"comment"},
		{"add-comment"},
		{"delete-comment"},
		{"export"},
		{"empty-trash"},
		{"update"},
		{"shared-drives"},
		{"shared-drive"},
		{"create-drive"},
		{"delete-drive"},
		{"update-drive"},
		{"about"},
		{"changes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(driveCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestDocsCommands tests docs command structure
// Note: Detailed docs tests are in cmd/docs_test.go
func TestDocsCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"read"},
		{"info"},
		{"create"},
		{"append"},
		{"insert"},
		{"replace"},
		{"format"},
		{"set-paragraph-style"},
		{"add-list"},
		{"remove-list"},
		{"trash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(docsCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestSheetsCommands tests sheets command structure
// Note: Detailed sheets tests are in cmd/sheets_test.go
func TestSheetsCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"info"},
		{"list"},
		{"read"},
		{"create"},
		{"write"},
		{"append"},
		{"add-sheet"},
		{"delete-sheet"},
		{"clear"},
		{"format"},
		{"set-column-width"},
		{"set-row-height"},
		{"freeze"},
		{"copy-to"},
		{"batch-read"},
		{"batch-write"},
		{"add-named-range"},
		{"list-named-ranges"},
		{"delete-named-range"},
		{"add-filter"},
		{"clear-filter"},
		{"add-filter-view"},
		{"add-chart"},
		{"list-charts"},
		{"delete-chart"},
		{"add-conditional-format"},
		{"list-conditional-formats"},
		{"delete-conditional-format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(sheetsCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestSlidesCommands tests slides command structure
// Note: Detailed slides tests are in cmd/slides_test.go
func TestSlidesCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"info"},
		{"list"},
		{"read"},
		{"create"},
		{"add-slide"},
		{"delete-slide"},
		{"duplicate-slide"},
		{"add-shape"},
		{"add-image"},
		{"add-text"},
		{"replace-text"},
		{"delete-object"},
		{"delete-text"},
		{"update-text-style"},
		{"update-transform"},
		{"create-table"},
		{"insert-table-rows"},
		{"delete-table-row"},
		{"update-table-cell"},
		{"update-table-border"},
		{"update-paragraph-style"},
		{"update-shape"},
		{"reorder-slides"},
		{"update-slide-background"},
		{"list-layouts"},
		{"add-line"},
		{"group"},
		{"ungroup"},
		{"thumbnail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(slidesCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// Note: Slides flag tests are in cmd/slides_test.go

// TestChatCommands tests chat command structure
func TestChatCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"list"},
		{"messages"},
		{"send"},
		{"members"},
		{"get"},
		{"update"},
		{"delete"},
		{"reactions"},
		{"react"},
		{"unreact"},
		{"get-space"},
		{"create-space"},
		{"delete-space"},
		{"update-space"},
		{"search-spaces"},
		{"find-dm"},
		{"setup-space"},
		{"get-member"},
		{"add-member"},
		{"remove-member"},
		{"update-member"},
		{"read-state"},
		{"mark-read"},
		{"thread-read-state"},
		{"attachment"},
		{"upload"},
		{"download"},
		{"events"},
		{"event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(chatCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestFormsCommands tests forms command structure
func TestFormsCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"info"},
		{"get"},
		{"responses"},
		{"response"},
		{"create"},
		{"update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(formsCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

// TestVersionCommand tests version command structure
func TestVersionCommand(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("version command not found")
	}

	if versionCmd.Use != "version" {
		t.Errorf("unexpected Use: %s", versionCmd.Use)
	}

	if versionCmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

// TestSearchCommand tests search command structure
func TestSearchCommand(t *testing.T) {
	if searchCmd == nil {
		t.Fatal("search command not found")
	}

	if searchCmd.Use != "search <query>" {
		t.Errorf("unexpected Use: %s", searchCmd.Use)
	}

	maxFlag := searchCmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Error("expected --max flag")
	}
}

// Helper function to find a subcommand by name
func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}
