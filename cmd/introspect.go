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

// commandScopeOverrides lists the commands that call a different API than
// the rest of their group, so their scopes are not the group's.
var commandScopeOverrides = map[string][]string{
	// Talks only to the Drive Activity API: one read-only scope, not
	// Drive file access.
	"drive activity": {"driveactivity"},
	// Trashes or deletes the file through the Drive API.
	"docs trash": {"drive"},
	// Decodes a CloudEvent from stdin; never calls Google.
	"events decode": {},
	// These also call the People API to turn member IDs into names and
	// email addresses. Without the contacts scopes they still run, but
	// members stay unresolved.
	"chat members":     {"chat", "contacts"},
	"chat build-cache": {"chat", "contacts"},
	"chat find-group":  {"chat", "contacts"},
	"chat find-space":  {"chat", "contacts"},
}

// scopeServicesFor returns the scope services a command uses.
func scopeServicesFor(cmd *cobra.Command) []string {
	key := commandKey(cmd)
	if key == "" {
		return nil
	}
	if services, ok := commandScopeOverrides[key]; ok {
		return append([]string(nil), services...)
	}
	group := strings.SplitN(key, " ", 2)[0]
	return append([]string(nil), commandScopeServices[group]...)
}

// handleIntrospection answers --schema and --show-scopes. It returns
// handled=false when neither flag is present so normal execution proceeds.
// Introspection never calls a Google API and never needs credentials.
func handleIntrospection(root *cobra.Command, args []string, out io.Writer) (bool, error) {
	// Cheap pre-check: most invocations never mention either flag.
	mentioned := false
	for _, a := range args {
		if strings.HasPrefix(a, "--schema") || strings.HasPrefix(a, "--show-scopes") {
			mentioned = true
			break
		}
	}
	if !mentioned {
		return false, nil
	}

	cmd, rest, err := root.Find(args)
	if err != nil {
		// Let normal execution report the unknown command.
		return false, nil
	}

	// Decide with real flag parsing, so that a flag *value* which happens
	// to read "--schema" (e.g. `--body --schema`) is not mistaken for the
	// flag. Parsing happens on a throwaway flag set that mirrors the
	// command's flags, leaving the command's own flag state untouched for
	// the normal execution path.
	wantSchema, wantScopes, ok := parseIntrospectionFlags(cmd, rest)
	if !ok || (!wantSchema && !wantScopes) {
		return false, nil
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

	if effect, ok := sensitiveEffect(cmd); ok {
		schema["sensitive"] = true
		schema["effect"] = effect
	}
	// Commands that need confirmation only when a flag is used. Reported
	// statically: --schema never looks at the other flag values.
	if rule, ok := conditionalSensitive[commandKey(cmd)]; ok {
		schema["sensitive_when"] = map[string]interface{}{
			"flag":   rule.flag,
			"effect": rule.effect,
		}
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
	// `auth login` is the command that requests scopes, so "none" would be
	// wrong: what it asks for depends on its own flags.
	if commandKey(cmd) == "auth login" {
		all := auth.ValidServiceNames()
		result["services"] = all
		result["scopes"] = auth.ScopesForServices(all)
		result["readonly_scopes"] = auth.ReadOnlyScopesForServices(all)
		result["note"] = "auth login requests the scopes of the services named by --services (every service when omitted). " +
			"--readonly requests only their read-only scopes. --add also re-requests the scopes the stored login already holds. " +
			"`scopes` and `readonly_scopes` here list the most a login can request; " +
			"run `gws <service> <command> --show-scopes` for one service."
		return result
	}
	// `auth status` asks Google who the stored login belongs to.
	if commandKey(cmd) == "auth status" {
		identity := auth.ScopesForServices(nil)
		result["services"] = []string{}
		result["scopes"] = identity
		result["readonly_scopes"] = identity
		result["note"] = "With a stored OAuth login, auth status calls Google's user-info endpoint, which needs the identity scope every login already includes. " +
			"With an external access token or a service-account mode it only checks local configuration and uses no scopes."
		return result
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
	// Suggested logins use --add so that following them extends the
	// stored authorization instead of replacing it.
	result["login"] = fmt.Sprintf("gws auth login --add --services %s", joined)
	result["login_readonly"] = fmt.Sprintf("gws auth login --add --services %s --readonly", joined)
	result["login_note"] = "--add keeps the services already authorized; without it, a login replaces the stored authorization."
	return result
}

// readOnlyScopeHint returns a one-line hint when a command failed with an
// insufficient-scope error on a service that was authorized read-only or
// only partially.
// Returns "" when the failure has another cause.
func readOnlyScopeHint(cmd *cobra.Command, err error, access map[string]string) string {
	if cmd == nil || err == nil || len(access) == 0 {
		return ""
	}
	if !strings.Contains(strings.ToLower(err.Error()), "insufficient") {
		return ""
	}
	var lacking []string
	onlyReadAccess := true
	for _, svc := range scopeServicesFor(cmd) {
		level, ok := access[svc]
		if !ok || level == auth.AccessFull {
			continue
		}
		lacking = append(lacking, svc)
		if level != auth.AccessReadOnly && level != auth.AccessPartialReadOnly {
			onlyReadAccess = false
		}
	}
	if len(lacking) == 0 {
		return ""
	}
	joined := strings.Join(lacking, ",")
	// Worded without "write access": some services (groups) have only
	// read scopes, and a partial grant can be missing one of those.
	hint := fmt.Sprintf("Hint: the stored login does not hold every scope %s needs. ", joined)
	if onlyReadAccess {
		// The login is read-only so far: offer to keep it that way before
		// suggesting the wider grant.
		return hint + fmt.Sprintf("To complete read-only access: gws auth login --add --services %s --readonly. For full access, run it without --readonly.", joined)
	}
	return hint + fmt.Sprintf("Grant them with: gws auth login --add --services %s", joined)
}

// discardValue is a flag value that accepts anything and stores nothing.
// It reports the mirrored flag's type so pflag keeps treating bool flags
// as value-less.
type discardValue struct{ typ string }

func (d *discardValue) String() string   { return "" }
func (d *discardValue) Set(string) error { return nil }
func (d *discardValue) Type() string     { return d.typ }
func (d *discardValue) IsBoolFlag() bool { return d.typ == "bool" }

// parseIntrospectionFlags parses args against a copy of cmd's flag
// definitions and reports whether --schema / --show-scopes were set. ok is
// false when the arguments do not parse; normal execution then reports
// the error.
func parseIntrospectionFlags(cmd *cobra.Command, args []string) (schema, scopes, ok bool) {
	fs := pflag.NewFlagSet("introspection", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	fs.BoolVar(&schema, "schema", false, "")
	fs.BoolVar(&scopes, "show-scopes", false, "")

	mirror := func(f *pflag.Flag) {
		if fs.Lookup(f.Name) != nil {
			return
		}
		shorthand := f.Shorthand
		if shorthand != "" && fs.ShorthandLookup(shorthand) != nil {
			shorthand = ""
		}
		fs.AddFlag(&pflag.Flag{
			Name:        f.Name,
			Shorthand:   shorthand,
			Value:       &discardValue{typ: f.Value.Type()},
			NoOptDefVal: f.NoOptDefVal,
		})
	}
	cmd.LocalFlags().VisitAll(mirror)
	cmd.InheritedFlags().VisitAll(mirror)
	if fs.Lookup("help") == nil {
		fs.BoolP("help", "h", false, "")
	}

	if err := fs.Parse(args); err != nil {
		return false, false, false
	}
	return schema, scopes, true
}
