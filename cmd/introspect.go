package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandScopeServices maps a top-level command group to the scope
// services (keys of auth.ServiceScopes) its commands authenticate with.
// Groups that need no Google OAuth scopes (auth, search, hooks, version)
// are absent.
var commandScopeServices = map[string][]string{
	"gmail":    {"gmail"},
	"calendar": {"calendar"},
	"drive":    {"drive"},
	"docs":     {"docs"},
	"sheets":   {"sheets"},
	"slides":   {"slides"},
	"tasks":    {"tasks"},
	"chat":     {"chat"},
	"contacts": {"contacts"},
	"people":   {"people"},
	"groups":   {"groups"},
	"keep":     {"keep"},
	"forms":    {"forms"},
	"events":   {"events"},
}

// scopeServicesFor returns the scope services a command uses. `drive
// activity` additionally needs the Drive Activity API scope.
func scopeServicesFor(cmd *cobra.Command) []string {
	key := commandKey(cmd)
	if key == "" {
		return nil
	}
	group := strings.SplitN(key, " ", 2)[0]
	services := append([]string(nil), commandScopeServices[group]...)
	if key == "drive activity" {
		services = append(services, "driveactivity")
	}
	return services
}

// handleIntrospection answers --schema and --show-scopes. It returns
// handled=false when neither flag is present so normal execution proceeds.
// Introspection never calls a Google API and never needs credentials.
func handleIntrospection(root *cobra.Command, args []string, out io.Writer) (bool, error) {
	wantSchema, wantScopes := false, false
	for _, a := range args {
		if a == "--" {
			break
		}
		switch a {
		case "--schema", "--schema=true":
			wantSchema = true
		case "--show-scopes", "--show-scopes=true":
			wantScopes = true
		}
	}
	if !wantSchema && !wantScopes {
		return false, nil
	}

	cmd, _, err := root.Find(args)
	if err != nil {
		return true, err
	}

	result := map[string]interface{}{}
	if wantSchema {
		for k, v := range commandSchema(cmd) {
			result[k] = v
		}
	}
	if wantScopes {
		for k, v := range commandScopes(cmd) {
			result[k] = v
		}
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return true, enc.Encode(result)
}

// commandSchema describes a command's arguments, flags, and subcommands.
// It is derived from the Cobra tree, so it cannot drift from --help.
func commandSchema(cmd *cobra.Command) map[string]interface{} {
	schema := map[string]interface{}{
		"command":     cmd.CommandPath(),
		"usage":       cmd.UseLine(),
		"description": cmd.Short,
	}
	if cmd.Long != "" {
		schema["long_description"] = cmd.Long
	}

	flags := []map[string]interface{}{}
	seen := map[string]bool{}
	addFlags := func(fs *pflag.FlagSet, global bool) {
		fs.VisitAll(func(f *pflag.Flag) {
			if f.Hidden || seen[f.Name] || f.Name == "help" {
				return
			}
			seen[f.Name] = true
			entry := map[string]interface{}{
				"name":        f.Name,
				"type":        f.Value.Type(),
				"default":     f.DefValue,
				"description": f.Usage,
				"required":    false,
			}
			if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
				entry["required"] = true
			}
			if f.Shorthand != "" {
				entry["shorthand"] = f.Shorthand
			}
			if global {
				entry["global"] = true
			}
			flags = append(flags, entry)
		})
	}
	addFlags(cmd.LocalNonPersistentFlags(), false)
	addFlags(cmd.PersistentFlags(), false)
	addFlags(cmd.InheritedFlags(), true)
	sort.SliceStable(flags, func(i, j int) bool {
		gi, _ := flags[i]["global"].(bool)
		gj, _ := flags[j]["global"].(bool)
		if gi != gj {
			return !gi
		}
		return flags[i]["name"].(string) < flags[j]["name"].(string)
	})
	schema["flags"] = flags

	subs := []map[string]interface{}{}
	for _, sub := range cmd.Commands() {
		if sub.Hidden || !sub.IsAvailableCommand() {
			continue
		}
		subs = append(subs, map[string]interface{}{
			"name":        sub.Name(),
			"description": sub.Short,
		})
	}
	if len(subs) > 0 {
		schema["subcommands"] = subs
	}

	if effect, ok := sensitiveCommands[commandKey(cmd)]; ok {
		schema["sensitive"] = true
		schema["effect"] = effect
	}
	return schema
}

// commandScopes reports the OAuth scopes of the service a command belongs
// to. Granularity is per service: a read command needs only the read-only
// set, a command that changes data needs the full set.
func commandScopes(cmd *cobra.Command) map[string]interface{} {
	services := scopeServicesFor(cmd)
	result := map[string]interface{}{
		"command":  cmd.CommandPath(),
		"services": services,
	}
	if len(services) == 0 {
		result["services"] = []string{}
		result["scopes"] = []string{}
		result["readonly_scopes"] = []string{}
		result["note"] = "This command does not use Google OAuth scopes."
		return result
	}
	result["scopes"] = auth.ScopesForServices(services)
	result["readonly_scopes"] = auth.ReadOnlyScopesForServices(services)
	joined := strings.Join(services, ",")
	result["login"] = fmt.Sprintf("gws auth login --services %s", joined)
	result["login_readonly"] = fmt.Sprintf("gws auth login --services %s --readonly", joined)
	return result
}

// readOnlyScopeHint returns a one-line hint when a command failed with an
// insufficient-scope error on a service that was authorized read-only.
// Returns "" when the failure has another cause.
func readOnlyScopeHint(cmd *cobra.Command, err error, access map[string]string) string {
	if cmd == nil || err == nil || len(access) == 0 {
		return ""
	}
	if !strings.Contains(strings.ToLower(err.Error()), "insufficient") {
		return ""
	}
	var readOnly []string
	for _, svc := range scopeServicesFor(cmd) {
		if access[svc] == auth.AccessReadOnly {
			readOnly = append(readOnly, svc)
		}
	}
	if len(readOnly) == 0 {
		return ""
	}
	joined := strings.Join(readOnly, ",")
	return fmt.Sprintf("Hint: %s is authorized read-only. Grant write access with: gws auth login --add --services %s", joined, joined)
}
