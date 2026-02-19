package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// skillsDir returns the path to the skills directory relative to the project root.
func skillsDir(t *testing.T) string {
	t.Helper()
	// Tests run from the package directory (cmd/), so skills/ is one level up.
	dir := filepath.Join("..", "skills")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("skills directory not found at %s", dir)
	}
	return dir
}

func pluginDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", ".claude-plugin")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf(".claude-plugin directory not found at %s", dir)
	}
	return dir
}

// --- Plugin Manifest Tests ---

func TestMarketplaceJSON_Valid(t *testing.T) {
	path := filepath.Join(pluginDir(t), "marketplace.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read marketplace.json: %v", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("marketplace.json is not valid JSON: %v", err)
	}

	// Check required top-level fields
	requiredFields := []string{"name", "owner", "metadata", "plugins"}
	for _, field := range requiredFields {
		if _, ok := manifest[field]; !ok {
			t.Errorf("marketplace.json missing required field: %s", field)
		}
	}

	// Validate name
	if name, ok := manifest["name"].(string); !ok || name != "workspace-cli" {
		t.Errorf("expected marketplace name 'workspace-cli', got '%v'", manifest["name"])
	}

	// Validate owner has name and email
	if owner, ok := manifest["owner"].(map[string]interface{}); ok {
		if _, ok := owner["name"]; !ok {
			t.Error("marketplace.json owner missing 'name'")
		}
		if _, ok := owner["email"]; !ok {
			t.Error("marketplace.json owner missing 'email'")
		}
	} else {
		t.Error("marketplace.json owner is not an object")
	}

	// Validate metadata has description and version
	if meta, ok := manifest["metadata"].(map[string]interface{}); ok {
		if _, ok := meta["description"]; !ok {
			t.Error("marketplace.json metadata missing 'description'")
		}
		if _, ok := meta["version"]; !ok {
			t.Error("marketplace.json metadata missing 'version'")
		}
	} else {
		t.Error("marketplace.json metadata is not an object")
	}

	// Validate plugins array
	plugins, ok := manifest["plugins"].([]interface{})
	if !ok || len(plugins) == 0 {
		t.Fatal("marketplace.json plugins should be a non-empty array")
	}

	// Validate the gws plugin entry
	gwsPlugin, ok := plugins[0].(map[string]interface{})
	if !ok {
		t.Fatal("first plugin entry is not an object")
	}

	if name, ok := gwsPlugin["name"].(string); !ok || name != "gws" {
		t.Errorf("expected plugin name 'gws', got '%v'", gwsPlugin["name"])
	}

	// Validate skills array lists all 11 skill paths
	skills, ok := gwsPlugin["skills"].([]interface{})
	if !ok {
		t.Fatal("plugin skills is not an array")
	}

	if len(skills) != len(expectedSkills) {
		t.Errorf("expected %d skills in marketplace.json, got %d", len(expectedSkills), len(skills))
	}

	// Verify each expected skill has a path entry
	for _, expected := range expectedSkills {
		expectedPath := "./skills/" + expected
		found := false
		for _, s := range skills {
			if s.(string) == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("marketplace.json missing skill path: %s", expectedPath)
		}
	}
}

// --- Skill Directory Structure Tests ---

var expectedSkills = []string{
	"gmail", "calendar", "drive", "docs", "sheets",
	"slides", "tasks", "chat", "forms", "search", "auth",
	"contacts",
}

func TestSkillDirectories_AllExist(t *testing.T) {
	base := skillsDir(t)
	for _, skill := range expectedSkills {
		dir := filepath.Join(base, skill)
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("skill directory missing: %s", skill)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", skill)
		}
	}
}

func TestSkillFiles_AllHaveSKILLmd(t *testing.T) {
	base := skillsDir(t)
	for _, skill := range expectedSkills {
		path := filepath.Join(base, skill, "SKILL.md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("SKILL.md missing for skill: %s", skill)
		}
	}
}

func TestSkillFiles_AllHaveReferences(t *testing.T) {
	base := skillsDir(t)

	// Services with references/commands.md
	servicesWithCommands := []string{
		"gmail", "calendar", "drive", "docs", "sheets",
		"slides", "tasks", "chat", "forms", "search",
		"contacts",
	}
	for _, skill := range servicesWithCommands {
		path := filepath.Join(base, skill, "references", "commands.md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("references/commands.md missing for skill: %s", skill)
		}
	}

	// Auth has setup-guide.md instead
	authGuide := filepath.Join(base, "auth", "references", "setup-guide.md")
	if _, err := os.Stat(authGuide); os.IsNotExist(err) {
		t.Error("references/setup-guide.md missing for auth skill")
	}
}

func TestSkillFiles_NoUnexpectedSkills(t *testing.T) {
	base := skillsDir(t)
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("failed to read skills directory: %v", err)
	}

	expectedSet := make(map[string]bool)
	for _, s := range expectedSkills {
		expectedSet[s] = true
	}

	for _, entry := range entries {
		if entry.IsDir() && !expectedSet[entry.Name()] {
			t.Errorf("unexpected skill directory: %s", entry.Name())
		}
	}
}

// --- SKILL.md Content Tests ---

func TestSKILLmd_HasYAMLFrontmatter(t *testing.T) {
	base := skillsDir(t)
	for _, skill := range expectedSkills {
		t.Run(skill, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			if !strings.HasPrefix(content, "---\n") {
				t.Error("SKILL.md does not start with YAML frontmatter (---)")
			}

			// Find closing ---
			secondDash := strings.Index(content[4:], "\n---\n")
			if secondDash == -1 {
				t.Error("SKILL.md missing closing YAML frontmatter (---)")
				return
			}

			frontmatter := content[4 : 4+secondDash]

			// Check required frontmatter fields
			requiredFields := []string{"name:", "version:", "description:"}
			for _, field := range requiredFields {
				if !strings.Contains(frontmatter, field) {
					t.Errorf("SKILL.md frontmatter missing field: %s", field)
				}
			}

			// Verify name follows gws-{service} pattern
			expectedName := "gws-" + skill
			if !strings.Contains(frontmatter, "name: "+expectedName) {
				t.Errorf("expected frontmatter name '%s'", expectedName)
			}

			// Verify version is set (1.0.0 for service skills, 0.x.0 for workflow skills)
			if !strings.Contains(frontmatter, "version:") {
				t.Error("expected frontmatter to contain version field")
			}
		})
	}
}

func TestSKILLmd_HasDisclaimer(t *testing.T) {
	base := skillsDir(t)
	for _, skill := range expectedSkills {
		t.Run(skill, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "not the official Google CLI") || !strings.Contains(content, "not endorsed by") {
				t.Error("SKILL.md missing not-official-Google-CLI disclaimer")
			}
		})
	}
}

func TestSKILLmd_HasDependencyCheck(t *testing.T) {
	base := skillsDir(t)
	for _, skill := range expectedSkills {
		t.Run(skill, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "gws version") {
				t.Error("SKILL.md missing dependency check (gws version)")
			}
		})
	}
}

func TestSKILLmd_HasAuthSection(t *testing.T) {
	base := skillsDir(t)
	// All service skills (not auth itself) should reference authentication
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "contacts"}
	for _, skill := range services {
		t.Run(skill, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "gws auth") {
				t.Error("SKILL.md missing authentication section referencing gws auth")
			}
		})
	}
}

func TestSKILLmd_HasOutputModes(t *testing.T) {
	base := skillsDir(t)
	// All service skills should document output modes
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "search", "contacts"}
	for _, skill := range services {
		t.Run(skill, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "--format json") || !strings.Contains(content, "--format text") {
				t.Error("SKILL.md missing output modes documentation (--format json/text)")
			}
		})
	}
}

func TestSKILLmd_HasAgentTips(t *testing.T) {
	base := skillsDir(t)
	for _, skill := range expectedSkills {
		t.Run(skill, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, skill, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "Tips for AI Agents") && !strings.Contains(content, "Tips") {
				t.Error("SKILL.md missing AI agent tips section")
			}
		})
	}
}

// --- Cross-Reference: Skills Document Real CLI Commands ---

func TestSkillCommands_MatchCLI(t *testing.T) {
	// Map of service name to the cobra parent command and expected subcommand names
	type serviceCommands struct {
		parentCmd   *cobra.Command
		subcommands []string
	}

	services := map[string]serviceCommands{
		"gmail": {
			parentCmd:   gmailCmd,
			subcommands: []string{"list", "read", "send", "labels", "label", "archive", "archive-thread", "trash", "thread", "reply", "event-id"},
		},
		"calendar": {
			parentCmd:   calendarCmd,
			subcommands: []string{"list", "events", "create", "update", "delete", "rsvp"},
		},
		"drive": {
			parentCmd:   driveCmd,
			subcommands: []string{"list", "search", "info", "download", "upload", "create-folder", "move", "delete", "comments", "copy"},
		},
		"docs": {
			parentCmd:   docsCmd,
			subcommands: []string{"read", "info", "create", "append", "insert", "replace", "delete", "add-table"},
		},
		"sheets": {
			parentCmd: sheetsCmd,
			subcommands: []string{
				"info", "list", "read", "create", "write", "append",
				"add-sheet", "delete-sheet", "clear",
				"insert-rows", "delete-rows", "insert-cols", "delete-cols",
				"rename-sheet", "duplicate-sheet",
				"merge", "unmerge", "sort", "find-replace",
				"copy-to", "batch-read", "batch-write",
			},
		},
		"slides": {
			parentCmd: slidesCmd,
			subcommands: []string{
				"info", "list", "read", "create",
				"add-slide", "delete-slide", "duplicate-slide",
				"add-shape", "add-image", "add-text", "replace-text",
				"delete-object", "delete-text",
				"update-text-style", "update-transform",
				"create-table", "insert-table-rows", "delete-table-row",
				"update-table-cell", "update-table-border",
				"update-paragraph-style", "update-shape",
				"reorder-slides",
				"update-slide-background", "list-layouts",
				"add-line", "group", "ungroup", "thumbnail",
			},
		},
		"tasks": {
			parentCmd:   tasksCmd,
			subcommands: []string{"lists", "list", "create", "update", "complete"},
		},
		"chat": {
			parentCmd:   chatCmd,
			subcommands: []string{"list", "messages", "send"},
		},
		"forms": {
			parentCmd:   formsCmd,
			subcommands: []string{"info", "responses"},
		},
		"contacts": {
			parentCmd:   contactsCmd,
			subcommands: []string{"list", "search", "get", "create", "delete"},
		},
	}

	base := skillsDir(t)

	for svcName, svc := range services {
		t.Run(svcName, func(t *testing.T) {
			// 1. Verify every expected subcommand exists in the CLI
			for _, subName := range svc.subcommands {
				cmd := findSubcommand(svc.parentCmd, subName)
				if cmd == nil {
					t.Errorf("CLI command 'gws %s %s' not found but expected by skill", svcName, subName)
				}
			}

			// 2. Verify every CLI subcommand is documented in the SKILL.md
			data, err := os.ReadFile(filepath.Join(base, svcName, "SKILL.md"))
			if err != nil {
				t.Fatalf("failed to read SKILL.md: %v", err)
			}
			content := string(data)

			for _, subCmd := range svc.parentCmd.Commands() {
				// Check that the subcommand name appears in the skill documentation
				cmdRef := "gws " + svcName + " " + subCmd.Name()
				if !strings.Contains(content, subCmd.Name()) {
					t.Errorf("CLI command '%s' exists but not documented in SKILL.md", cmdRef)
				}
			}

			// 3. Verify every CLI subcommand is documented in references/commands.md
			refData, err := os.ReadFile(filepath.Join(base, svcName, "references", "commands.md"))
			if err != nil {
				t.Fatalf("failed to read references/commands.md: %v", err)
			}
			refContent := string(refData)

			for _, subCmd := range svc.parentCmd.Commands() {
				cmdRef := "gws " + svcName + " " + subCmd.Name()
				if !strings.Contains(refContent, subCmd.Name()) {
					t.Errorf("CLI command '%s' exists but not documented in references/commands.md", cmdRef)
				}
			}
		})
	}
}

// TestSearchSkill_DocumentsCLI tests search separately since it's not a parent+subcommand structure.
func TestSearchSkill_DocumentsCLI(t *testing.T) {
	base := skillsDir(t)

	data, err := os.ReadFile(filepath.Join(base, "search", "SKILL.md"))
	if err != nil {
		t.Fatalf("failed to read search SKILL.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "gws search") {
		t.Error("search SKILL.md missing 'gws search' command documentation")
	}

	// Verify key flags are documented
	requiredFlags := []string{"--max", "--site", "--type", "--start", "--api-key", "--engine-id"}
	for _, flag := range requiredFlags {
		if !strings.Contains(content, flag) {
			t.Errorf("search SKILL.md missing flag documentation: %s", flag)
		}
	}

	// Cross-reference: verify documented flags actually exist on the CLI command
	for _, flag := range requiredFlags {
		flagName := strings.TrimPrefix(flag, "--")
		if searchCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("documented flag %s does not exist on CLI search command", flag)
		}
	}
}

// TestAuthSkill_HasSetupGuide tests the auth skill has the GCP setup guide.
func TestAuthSkill_HasSetupGuide(t *testing.T) {
	base := skillsDir(t)

	data, err := os.ReadFile(filepath.Join(base, "auth", "references", "setup-guide.md"))
	if err != nil {
		t.Fatalf("failed to read auth setup-guide.md: %v", err)
	}
	content := string(data)

	// Verify key setup steps are documented
	requiredSections := []string{
		"Google Cloud",
		"OAuth",
		"client_id",
		"client_secret",
		"GWS_CLIENT_ID",
		"GWS_CLIENT_SECRET",
		"token.json",
		"gws auth login",
		"Troubleshooting",
	}
	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("auth setup-guide.md missing content: %s", section)
		}
	}
}

// --- Reference File Content Tests ---

func TestReferenceFiles_HaveDisclaimer(t *testing.T) {
	base := skillsDir(t)

	// commands.md files
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "search", "contacts"}
	for _, svc := range services {
		t.Run(svc+"/commands.md", func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, svc, "references", "commands.md"))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			if !strings.Contains(string(data), "not the official Google CLI") {
				t.Error("references/commands.md missing unofficial disclaimer")
			}
		})
	}

	// auth setup-guide.md
	t.Run("auth/setup-guide.md", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(base, "auth", "references", "setup-guide.md"))
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if !strings.Contains(string(data), "not the official Google CLI") {
			t.Error("auth setup-guide.md missing unofficial disclaimer")
		}
	})
}

func TestReferenceFiles_DocumentGlobalFlags(t *testing.T) {
	base := skillsDir(t)
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "search", "contacts"}

	for _, svc := range services {
		t.Run(svc, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, svc, "references", "commands.md"))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "--config") {
				t.Error("references/commands.md missing --config global flag")
			}
			if !strings.Contains(content, "--format") {
				t.Error("references/commands.md missing --format global flag")
			}
		})
	}
}

func TestReferenceFiles_DocumentQuietFlag(t *testing.T) {
	base := skillsDir(t)
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "search", "contacts"}

	for _, svc := range services {
		t.Run(svc, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(base, svc, "references", "commands.md"))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			content := string(data)

			if !strings.Contains(content, "--quiet") {
				t.Error("references/commands.md missing --quiet global flag")
			}
		})
	}
}

// --- File Count Validation ---

func TestSkillFiles_TotalCount(t *testing.T) {
	base := skillsDir(t)
	pDir := pluginDir(t)

	count := 0

	// marketplace.json
	if _, err := os.Stat(filepath.Join(pDir, "marketplace.json")); err == nil {
		count++
	}

	// SKILL.md files
	for _, skill := range expectedSkills {
		if _, err := os.Stat(filepath.Join(base, skill, "SKILL.md")); err == nil {
			count++
		}
	}

	// references/commands.md files
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "slides", "tasks", "chat", "forms", "search", "contacts"}
	for _, svc := range services {
		if _, err := os.Stat(filepath.Join(base, svc, "references", "commands.md")); err == nil {
			count++
		}
	}

	// auth setup-guide.md
	if _, err := os.Stat(filepath.Join(base, "auth", "references", "setup-guide.md")); err == nil {
		count++
	}

	expectedTotal := 25 // 1 marketplace.json + 12 SKILL.md + 11 commands.md + 1 setup-guide.md
	if count != expectedTotal {
		t.Errorf("expected %d skill files, found %d", expectedTotal, count)
	}
}
