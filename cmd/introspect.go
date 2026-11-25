package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

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

// commandExtraScopes lists commands whose endpoint requires a scope that
// `gws auth login` does not request, with how to supply it.
var commandExtraScopes = map[string]struct {
	scopes []string
	note   string
}{
	"gmail delete":        {[]string{gmailFullScope}, gmailFullScopeNote},
	"gmail delete-thread": {[]string{gmailFullScope}, gmailFullScopeNote},
	"gmail batch-delete":  {[]string{gmailFullScope}, gmailFullScopeNote},
}

const (
	gmailFullScope     = "https://mail.google.com/"
	gmailFullScopeNote = "Permanent deletion requires the full Gmail scope, which gws auth login does not request. " +
		"Supply a token that carries it through GWS_ACCESS_TOKEN, or use the trash commands instead."
)

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
		when := map[string]interface{}{
			"flag":   rule.flag,
			"effect": rule.effect,
		}
		// Present only when the flag's value decides: any other value
		// runs without confirmation.
		if len(rule.values) > 0 {
			when["values"] = rule.values
		}
		schema["sensitive_when"] = when
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
	// `hooks test` (and the hooks it exercises) can run read-only Workspace
	// commands under the stored login when the Workspace provider is on.
	if commandKey(cmd) == "hooks test" {
		sources := []string{"gmail", "calendar", "tasks", "chat"}
		result["services"] = sources
		result["scopes"] = auth.ReadOnlyScopesForServices(sources)
		result["readonly_scopes"] = auth.ReadOnlyScopesForServices(sources)
		result["note"] = "Only when the Workspace provider is enabled: the hook then reads from the configured sources (Gmail, Calendar, Tasks, Chat) with the stored login, read-only. " +
			"With only the time provider it uses no scopes. An external access token is never used for hooks."
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
	// Some endpoints need a scope `gws auth login` never requests. Say so,
	// or the login suggested below would look like it is enough.
	if extra, ok := commandExtraScopes[commandKey(cmd)]; ok {
		result["additional_scopes"] = extra.scopes
		result["additional_scopes_note"] = extra.note
	}
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
	if cmd == nil || err == nil {
		return ""
	}
	if !strings.Contains(strings.ToLower(err.Error()), "insufficient") {
		return ""
	}
	// Some endpoints need a scope no gws login requests. Suggesting a
	// login for those would send the user through consent for nothing.
	if extra, ok := commandExtraScopes[commandKey(cmd)]; ok {
		return fmt.Sprintf("Hint: this command needs %s. %s", strings.Join(extra.scopes, ", "), extra.note)
	}

	// The rest needs to know what the login holds; a login from before
	// access levels were recorded gives nothing to go on.
	if len(access) == 0 {
		return ""
	}

	var lacking []string
	completeReadOnly, incompleteReadOnly, holdsWrite := 0, 0, false
	for _, svc := range scopeServicesFor(cmd) {
		// A service that is not in the record at all was left out of a
		// scoped login: as much a missing grant as a read-only one.
		level, ok := access[svc]
		if ok && level == auth.AccessFull {
			continue
		}
		lacking = append(lacking, svc)
		switch level {
		case auth.AccessReadOnly:
			completeReadOnly++
		case auth.AccessPartialReadOnly:
			incompleteReadOnly++
		case auth.AccessPartial:
			holdsWrite = true
		}
	}
	if len(lacking) == 0 {
		return ""
	}
	joined := strings.Join(lacking, ",")
	switch {
	case completeReadOnly == len(lacking):
		// Every read scope gws knows for the service is already held, so
		// repeating the read-only login would change nothing. The usual
		// cause is a command that changes data; say that rather than
		// assert it, since the exact scope the endpoint wanted is not
		// known here.
		if len(lacking) == 1 && lacking[0] == "events" {
			// Changing data is not the test here: Chat and Meet
			// subscriptions can be created and managed with read scopes.
			// Only Drive targets, and Meet with --meet-created-scope
			// (meetings.space.created), need more.
			return "Hint: events is authorized read-only, which covers Chat subscriptions and Meet subscriptions that do not use --meet-created-scope. Subscriptions to Drive resources, and Meet ones with --meet-created-scope, need full access: gws auth login --add --services events. Otherwise, check that you can access the target itself."
		}
		return fmt.Sprintf("Hint: %s is authorized read-only. If this command changes data, grant full access with: gws auth login --add --services %s", joined, joined)
	case incompleteReadOnly == len(lacking):
		// Some read scopes were declined. Completing read-only access may
		// be all that is needed, so offer that before the wider grant.
		return fmt.Sprintf("Hint: the stored login does not hold every scope %s needs. To complete read-only access: gws auth login --add --services %s --readonly. For full access, run it without --readonly.", joined, joined)
	}
	// Worded without "write access": some services (groups) have only
	// read scopes, and a partial grant can be missing one of those.
	if holdsWrite {
		// The login already has some write access to the service; a
		// read-only retry would not add what is missing.
		return fmt.Sprintf("Hint: the stored login does not hold every scope %s needs. Grant them with: gws auth login --add --services %s", joined, joined)
	}
	// Whether the command only reads is not known here, so both ways in
	// are named rather than steering straight to full access.
	return fmt.Sprintf("Hint: the stored login does not hold every scope %s needs. Grant them with: gws auth login --add --services %s (append --readonly if read-only access is enough)", joined, joined)
}

// discardValue is a flag value that stores nothing. It reports the
// mirrored flag's type so pflag keeps treating bool flags as value-less,
// and it rejects values the real flag would reject for the common scalar
// types, so `--max=abc --schema` is still reported as the usage error it
// is rather than answered as if the command line were valid.
type discardValue struct{ typ string }

func (d *discardValue) String() string { return "" }
func (d *discardValue) Set(v string) error {
	var err error
	switch d.typ {
	case "bool":
		_, err = strconv.ParseBool(v)
	case "int", "int8", "int16", "int32", "int64", "count":
		_, err = strconv.ParseInt(v, 0, 64)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		_, err = strconv.ParseUint(v, 0, 64)
	case "float32", "float64":
		_, err = strconv.ParseFloat(v, 64)
	case "duration":
		_, err = time.ParseDuration(v)
	case "stringSlice", "intSlice", "boolSlice", "durationSlice":
		// pflag reads slice values as one CSV record and rejects bad
		// quoting; an empty value is an empty slice.
		if v != "" {
			_, err = csv.NewReader(strings.NewReader(v)).Read()
		}
	}
	return err
}
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
