package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

const revokeTimeout = 10 * time.Second

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Commands for managing Google OAuth authentication.",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Google",
	Long:  "Starts the OAuth2 authentication flow to obtain access tokens.",
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Long:  "Deletes the stored OAuth token.",
	RunE:  runLogout,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Displays the current authentication status and user info.",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	// Login flags for credentials (can also come from config/env)
	loginCmd.Flags().String("client-id", "", "OAuth client ID")
	loginCmd.Flags().String("client-secret", "", "OAuth client secret")
	loginCmd.Flags().String("services", "", "Comma-separated list of services to authorize (e.g. gmail,calendar,chat)")
	loginCmd.Flags().Bool("readonly", false, "Request only read-only scopes for the selected services (Chat and Meet event subscriptions can still be managed: they are authorized by read scopes)")
	loginCmd.Flags().Bool("add", false, "Add the services named by --services to the existing authorization instead of replacing it")
	viper.BindPFlag("client_id", loginCmd.Flags().Lookup("client-id"))
	viper.BindPFlag("client_secret", loginCmd.Flags().Lookup("client-secret"))
}

// resolveConfiguredAuthMode validates the configured auth-mode settings,
// returning the resolved config or the fail-closed validation error.
func resolveConfiguredAuthMode() (*auth.AuthModeConfig, error) {
	return auth.ResolveAuthMode(
		config.GetAuthMode(),
		config.GetServiceAccountKeyFile(),
		config.GetImpersonateUser(),
	)
}

func runLogin(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	modeCfg, err := resolveConfiguredAuthMode()
	if err != nil {
		return usageErrorf("%s", err.Error())
	}
	if modeCfg.Mode != auth.AuthModeOAuth {
		extra := ""
		if modeCfg.Mode == auth.AuthModeDWD {
			extra = " and impersonate_user (GWS_IMPERSONATE_USER)"
		}
		return usageErrorf("auth login starts the interactive OAuth flow, but auth_mode is %q: service-account modes need no login — configure service_account_key_file (GWS_SERVICE_ACCOUNT_KEY_FILE)%s and run gws commands directly, or unset auth_mode to use OAuth",
			modeCfg.Mode, extra)
	}

	clientID := config.GetClientID()
	clientSecret := config.GetClientSecret()

	if clientID == "" || clientSecret == "" {
		return usageErrorf("missing credentials: set GWS_CLIENT_ID and GWS_CLIENT_SECRET environment variables, or use --client-id and --client-secret flags")
	}

	// An explicitly supplied --services that names nothing (`--services=`,
	// `--services ' , '`, often an unset shell variable) is a mistake, not
	// a request for every service.
	if cmd.Flags().Changed("services") && len(splitList(flagString(cmd, "services"))) == 0 {
		return usageErrorf("--services was given but names no service\nValid services: %s",
			strings.Join(auth.ValidServiceNames(), ", "))
	}

	// Determine the services from the --services flag, config, or all
	services := resolveServices(cmd)

	// Validate service names — fail fast on unknown services
	if unknown := auth.ValidateServices(services); len(unknown) > 0 {
		return usageErrorf("unknown service(s): %s\nValid services: %s",
			strings.Join(unknown, ", "),
			strings.Join(auth.ValidServiceNames(), ", "))
	}

	readOnly, _ := cmd.Flags().GetBool("readonly")
	add, _ := cmd.Flags().GetBool("add")

	var existingAccess map[string]string
	// The authorization this login starts from. --add extends it, so it
	// must still be the stored one when the result is saved.
	baseAuthorization := ""
	if add {
		// --add is for adding specific services. Without a selection it
		// would fall back to every service and ask for full access to all
		// of them, the opposite of an incremental request.
		if !hasServiceSelection(cmd) {
			return usageErrorf("--add requires --services: name the service(s) to add, e.g. gws auth login --add --services drive")
		}
		// --add inspects the stored login (account, scopes). Refresh it
		// once, up front, and save the result, so those lookups reuse one
		// valid token instead of each refreshing on its own — which, if
		// the refresh token is rotated, could leave the stored login dead.
		if err := refreshStoredToken(); err != nil {
			return p.PrintError(fmt.Errorf("could not refresh the stored authorization (%v), so --add was not started; run `gws auth login` without --add to replace it", err))
		}
		baseAuthorization = auth.StoredTokenFingerprint()
		existingAccess = existingGrantedAccess()
	}
	access := mergeGrantedAccess(existingAccess, services, readOnly)
	scopes := loginScopes(services, readOnly, hasServiceSelection(cmd))

	// previousScopes is the exact scope list the stored authorization
	// holds, which --add must preserve. It is nil for a replacing login.
	var previousScopes []string
	previousAccount := ""
	if add {
		var err error
		previousAccount, err = storedAccountID()
		if err != nil {
			// Without knowing whose authorization is stored, there is no
			// way to tell whether the account about to consent matches.
			return p.PrintError(fmt.Errorf("could not confirm which Google account the stored authorization belongs to (%v), so --add was not started; try again, or run `gws auth login` without --add to replace it", err))
		}
		// Ask for the whole merged grant explicitly: the previous scopes
		// are re-requested, never assumed to carry over from an earlier
		// grant, so the token holds exactly what this consent approved.
		previousScopes, err = storedScopes()
		if err != nil {
			// Guessing would mean asking again for scopes the user may
			// have declined, or failing to preserve ones they hold.
			return p.PrintError(fmt.Errorf("could not determine which scopes the stored authorization holds (%v), so --add was not started; try again, or run `gws auth login` without --add to replace it", err))
		}
		scopes = unionScopes(previousScopes, scopes)
	}

	if config.GetAccessToken() != "" {
		fmt.Fprintln(os.Stderr, "Warning: GWS_ACCESS_TOKEN / --access-token is set and takes precedence over the token this login stores.")
	}

	client := auth.NewOAuthClient(clientID, clientSecret, scopes)

	ctx := context.Background()
	token, err := client.Login(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	// Identify the account that just consented. For --add it must be the
	// account the stored token belongs to: the browser may be signed in to
	// a different one, and extending "the existing authorization" must
	// never silently move every later command onto another account.
	account, accountErr := lookupAccountID(token)
	if add {
		if err := checkSameAccount(previousAccount, account, accountErr); err != nil {
			return p.PrintError(err)
		}
	}

	// Work from what Google actually granted, not what was requested: the
	// user can untick scopes on the consent screen. If the response
	// carries no scope list, the requested scopes stand in: all of them
	// were requested in this very exchange (--add re-requests earlier
	// scopes explicitly), so the new token is never credited with a grant
	// that was only made to a previous one.
	grantedScopes := scopes
	if granted, ok := token.Extra("scope").(string); ok && strings.TrimSpace(granted) != "" {
		grantedScopes = strings.Fields(granted)
		// The token must not carry more than was asked for — for example
		// write scopes from an earlier grant that a later read-only login
		// was meant to replace.
		if extra := unrequestedScopes(scopes, grantedScopes); len(extra) > 0 {
			return p.PrintError(fmt.Errorf("the grant includes %d scope(s) that were not requested (%s), so the stored authorization was left unchanged; run `gws auth logout` to revoke earlier grants, then log in again", len(extra), strings.Join(shortScopeNames(extra), ", ")))
		}
		// An incremental login must not cost the user access they already
		// had. Compared scope by scope: access levels are too coarse, as
		// two different sets of write scopes can share one level.
		if lost := missingScopes(previousScopes, grantedScopes); len(lost) > 0 {
			return p.PrintError(fmt.Errorf("the new grant is missing %d previously granted scope(s) (%s), so the stored authorization was left unchanged; approve every requested scope, or run `gws auth login` without --add to replace it", len(lost), strings.Join(shortScopeNames(lost), ", ")))
		}
	}
	var notGranted []string
	access, notGranted = deriveGrantedAccess(access, grantedScopes)
	if len(notGranted) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Google did not grant the requested access for: %s\n", strings.Join(notGranted, ", "))
	}

	token, err = loginTokenToStore(token)
	if err != nil {
		return p.PrintError(err)
	}

	// Write order matters if the process is interrupted. The access record
	// is bound to the new token's fingerprint and goes first: until the
	// token itself is saved it matches nothing and is ignored, and its
	// presence stops the unbound service list from being trusted. The
	// service list goes last, so it never describes a newer login than the
	// access record does.
	// One login at a time may write the token and its metadata, so two
	// overlapping logins cannot pair one's token with the other's record.
	unlock, err := auth.LockLoginState()
	if err != nil {
		return p.PrintError(err)
	}
	defer unlock()

	// Another login may have finished while the browser was open. Saving
	// now would replace its grant with one built from the older state and
	// silently drop whatever it added.
	if add && auth.StoredTokenFingerprint() != baseAuthorization {
		return p.PrintError(fmt.Errorf("the stored authorization changed while this login was in progress (another `gws auth login` finished first), so this one was not saved; run the --add login again"))
	}

	restoreAccessRecord := auth.SnapshotGrantedAccess()
	errAccess := auth.SaveGrantedAccess(access, grantedScopes, account, token)

	if err := auth.SaveToken(token); err != nil {
		// The previous token is still the stored one: put its access
		// record back rather than leave one describing a token that was
		// never saved.
		restoreAccessRecord()
		return p.PrintError(err)
	}

	// If recording the metadata failed, remove both files: no metadata is
	// better than metadata that still describes the previous token.
	errServices := auth.SaveGrantedServices(grantedServiceNames(access))
	if errServices != nil || errAccess != nil {
		auth.DeleteGrantedMetadata()
		fmt.Fprintf(os.Stderr, "Warning: failed to record granted access (services: %v, access levels: %v); `gws auth status` will not show per-service access until the next login\n", errServices, errAccess)
	}

	return p.Print(map[string]interface{}{
		"status":    "success",
		"message":   "Authentication successful",
		"expires":   token.Expiry.Format(time.RFC3339),
		"services":  access,
		"read_only": allReadOnly(access),
	})
}

// loginTokenToStore decides which token a login persists. Every login
// must bring its own refresh token: reusing the previous one could keep a
// wider grant after access was narrowed, miss scopes that were just added,
// or belong to a different account than the one that just consented.
func loginTokenToStore(incoming *oauth2.Token) (*oauth2.Token, error) {
	if incoming == nil || incoming.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token was returned for this login, so the previous authorization was left unchanged; run `gws auth logout` and then `gws auth login` again")
	}
	return incoming, nil
}

// warnIfRefreshNotSaved reports a failure to store a refreshed token. The
// command at hand still works, but if the refresh token was rotated the
// stored login may stop working, so the user should hear about it.
func warnIfRefreshNotSaved(w io.Writer, err error) {
	if err != nil {
		fmt.Fprintf(w, "Warning: the refreshed login could not be saved (%v); if later commands fail to authenticate, run `gws auth login` again\n", err)
	}
}

// addGrantedAccess adds the per-service access levels and the read_only
// summary to a status result. It adds nothing when no levels are recorded.
func addGrantedAccess(result map[string]interface{}, access map[string]string) {
	if len(access) == 0 {
		return
	}
	result["services"] = access
	result["read_only"] = allReadOnly(access)
}

// existingGrantedAccess returns the access levels recorded by earlier
// logins, or nothing when no authorization is stored. Logins made before access levels were recorded only have a
// service list; those services were granted full access.
func existingGrantedAccess() map[string]string {
	// Metadata without a stored authorization describes a login that no
	// longer exists; never merge it into a new one.
	if !auth.TokenExists() {
		return map[string]string{}
	}
	if access := auth.LoadGrantedAccess(); access != nil {
		return access
	}
	// GrantedServicesForToken returns the plain service list only for a
	// login that never wrote an access record (an older version, which
	// always granted full access). If a record exists but belongs to
	// another token, a login was interrupted and nothing is assumed.
	access := map[string]string{}
	for _, svc := range auth.GrantedServicesForToken() {
		access[svc] = auth.AccessFull
	}
	return access
}

// loginScopes returns the scopes to request. An unscoped full login keeps
// requesting auth.AllScopes, exactly as before.
func loginScopes(services []string, readOnly, scoped bool) []string {
	if readOnly {
		return auth.ReadOnlyScopesForServices(services)
	}
	if !scoped {
		return auth.AllScopes
	}
	return auth.ScopesForServices(services)
}

// lookupTokenScopes asks Google which scopes a stored token actually
// holds, refreshing it first if needed. A variable so tests can replace
// the network call.
var lookupTokenScopes = func(token *oauth2.Token) ([]string, error) {
	// The token is used as given, never refreshed here: the caller has
	// already refreshed the stored login through the guarded path.
	ctx := context.Background()
	fresh := token
	svc, err := oauth2api.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	if err != nil {
		return nil, err
	}
	// The tokeninfo endpoint takes the access token as a URL parameter,
	// and a transport error quotes the request URL. Never let that reach
	// the terminal or a log.
	info, err := svc.Tokeninfo().AccessToken(fresh.AccessToken).Do()
	if err != nil {
		return nil, redactSecrets(err, fresh.AccessToken, fresh.RefreshToken, token.AccessToken, token.RefreshToken)
	}
	return strings.Fields(info.Scope), nil
}

// redactSecrets returns an error with the same message as err but with
// every occurrence of the given secrets removed, in plain and URL-escaped
// form. The original error is deliberately not wrapped, so the secret
// cannot be recovered through errors.Unwrap either.
func redactSecrets(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		for _, form := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
			msg = strings.ReplaceAll(msg, form, "[redacted]")
		}
	}
	return errors.New(msg)
}

// refreshStoredToken makes sure the stored token is currently valid,
// refreshing it if needed and saving the result (including a rotated
// refresh token) through the same guarded path every refresh uses. It is
// a no-op when nothing is stored. A variable so tests can replace the
// network call.
var refreshStoredToken = func() error {
	if !auth.TokenExists() {
		return nil
	}
	stored, err := auth.LoadToken()
	if err != nil {
		return err
	}
	_, saveErr, err := auth.EnsureFreshToken(context.Background(), config.GetClientID(), config.GetClientSecret(), stored)
	if err != nil {
		return redactSecrets(err, stored.AccessToken, stored.RefreshToken)
	}
	return saveErr
}

// storedScopes returns the exact scopes the stored authorization holds:
// the list recorded at login, or, for a login that predates that record,
// what Google reports for the stored token. It returns nil with no error
// only when nothing is stored; a stored token whose scopes cannot be
// determined is an error.
func storedScopes() ([]string, error) {
	if scopes := auth.LoadGrantedScopes(); len(scopes) > 0 {
		return scopes, nil
	}
	if !auth.TokenExists() {
		return nil, nil
	}
	token, err := auth.LoadToken()
	if err != nil {
		return nil, err
	}
	scopes, err := lookupTokenScopes(token)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("no scopes were reported for the stored token")
	}
	return scopes, nil
}

// lookupAccountID returns the stable Google account ID a token belongs to.
// A variable so tests can replace the network call.
var lookupAccountID = func(token *oauth2.Token) (string, error) {
	info, err := getUserInfo(token)
	if err != nil {
		return "", err
	}
	return info.Id, nil
}

// storedAccountID returns the account ID of the stored authorization: the
// one recorded at login, or, for a login that predates that record, the
// result of asking Google with the stored token.
//
// It returns "" with no error only when nothing is stored. A stored token
// whose account cannot be determined is an error, so callers never treat
// "could not check" as "nothing to check".
func storedAccountID() (string, error) {
	if id := auth.LoadGrantedAccount(); id != "" {
		return id, nil
	}
	if !auth.TokenExists() {
		return "", nil
	}
	token, err := auth.LoadToken()
	if err != nil {
		return "", err
	}
	id, err := lookupAccountID(token)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("no account ID was returned")
	}
	return id, nil
}

// checkSameAccount rejects an incremental login made with a different
// Google account than the stored one. When the stored account is known,
// the new one must be confirmed to match; an account that cannot be
// identified is not accepted on trust.
func checkSameAccount(previous, current string, lookupErr error) error {
	if previous == "" {
		return nil // nothing stored to compare against
	}
	if lookupErr != nil || current == "" {
		return fmt.Errorf("could not confirm which Google account authorized this login (%v), so the stored authorization was left unchanged; try again, or run `gws auth login` without --add to replace it", lookupErr)
	}
	if previous != current {
		return fmt.Errorf("this login was authorized by a different Google account than the stored one, so the stored authorization was left unchanged; choose the same account in the browser, or run `gws auth logout` and then `gws auth login` to switch accounts")
	}
	return nil
}

// unionScopes returns a followed by the scopes of b not already in a.
func unionScopes(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var out []string
	for _, s := range append(append([]string(nil), a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// identityScopes are the sign-in scopes Google may add to a grant on its
// own. They identify the account and give no access to Workspace data.
var identityScopes = map[string]bool{
	"openid":  true,
	"email":   true,
	"profile": true,
	"https://www.googleapis.com/auth/userinfo.email":   true,
	"https://www.googleapis.com/auth/userinfo.profile": true,
}

// unrequestedScopes returns the granted scopes that were not asked for.
// Only the identity scopes above are tolerated; anything else — including
// a scope gws has never heard of — is reported, since an unknown scope
// could carry access that a read-only login was meant to exclude.
func unrequestedScopes(requested, granted []string) []string {
	asked := make(map[string]bool, len(requested))
	for _, s := range requested {
		asked[s] = true
	}
	var extra []string
	for _, s := range granted {
		if !asked[s] && !identityScopes[s] {
			extra = append(extra, s)
		}
	}
	return extra
}

// missingScopes returns the scopes in previous that are absent from
// granted. Every previously held scope counts, including one gws does not
// itself use: --add promises to preserve what the login had. Only Google's
// identity scopes are skipped, since they are reported under several
// equivalent names (email / userinfo.email) and give no data access.
func missingScopes(previous, granted []string) []string {
	have := make(map[string]bool, len(granted))
	for _, s := range granted {
		have[s] = true
	}
	var missing []string
	for _, s := range previous {
		if !have[s] && !identityScopes[s] {
			missing = append(missing, s)
		}
	}
	return missing
}

// shortScopeNames strips the common Google scope URL prefix for display.
func shortScopeNames(scopes []string) []string {
	out := make([]string, len(scopes))
	for i, s := range scopes {
		out[i] = strings.TrimPrefix(s, "https://www.googleapis.com/auth/")
	}
	return out
}

// mergeGrantedAccess computes the per-service access levels after a login.
// existing is nil for a replacing login. A service that already has full
// access keeps it when re-added read-only, because incremental
// authorization never removes a previously granted scope.
func mergeGrantedAccess(existing map[string]string, services []string, readOnly bool) map[string]string {
	access := map[string]string{}
	for svc, level := range existing {
		access[svc] = level
	}
	level := auth.AccessFull
	if readOnly {
		level = auth.AccessReadOnly
	}
	for _, svc := range services {
		// Adding read-only never lowers a service that already holds
		// write scopes (full or partial).
		if prev := access[svc]; readOnly && (prev == auth.AccessFull || prev == auth.AccessPartial) {
			continue
		}
		access[svc] = level
	}
	return access
}

// deriveGrantedAccess builds the per-service access levels from the
// complete set of scopes Google returned. Every known service is examined,
// not just the requested ones, so the result describes the token rather
// than the request.
//
// A service is:
//   - full when every one of its full scopes is present;
//   - partial when it holds at least one write scope but not all of them
//     (the user unticked some on the consent screen) — never read-only;
//   - read-only when it holds its read-only scopes and no write scope;
//   - partial-readonly when it was requested and holds only some of its
//     read-only scopes;
//   - absent otherwise.
//
// A service whose scopes are all read-only (e.g. groups) is always
// read-only, whatever was requested: the label reflects what the token can
// do, so read_only: true means exactly "no write scope was granted".
// notGranted lists requested services that came back with less than was
// asked for.
func deriveGrantedAccess(requested map[string]string, grantedScopes []string) (access map[string]string, notGranted []string) {
	granted := make(map[string]bool, len(grantedScopes))
	for _, s := range grantedScopes {
		granted[s] = true
	}
	hasAll := func(scopes []string) bool {
		for _, s := range scopes {
			if !granted[s] {
				return false
			}
		}
		return true
	}

	// hasAny ignores the userinfo scopes every login carries.
	hasAny := func(scopes []string) bool {
		for _, s := range scopes {
			if granted[s] && !strings.Contains(s, "/auth/userinfo.") {
				return true
			}
		}
		return false
	}

	access = map[string]string{}
	for _, svc := range auth.ValidServiceNames() {
		level, wasRequested := requested[svc]
		one := []string{svc}
		full := auth.ScopesForServices(one)
		readOnly := auth.ReadOnlyScopesForServices(one)

		// Write scopes: in the full set but not in the read-only set.
		isReadOnlyScope := make(map[string]bool, len(readOnly))
		for _, s := range readOnly {
			isReadOnlyScope[s] = true
		}
		// definesWriteScope: the service has scopes beyond its read-only
		// set (compared by membership — sheets has one scope in each).
		definesWriteScope, hasWriteScope := false, false
		for _, s := range full {
			if !isReadOnlyScope[s] {
				definesWriteScope = true
				if granted[s] {
					hasWriteScope = true
				}
			}
		}

		switch {
		case hasAll(full) && definesWriteScope:
			access[svc] = auth.AccessFull
		case hasWriteScope:
			access[svc] = auth.AccessPartial
			if wasRequested {
				notGranted = append(notGranted, svc+" (only some scopes)")
			}
		case hasAll(readOnly):
			access[svc] = auth.AccessReadOnly
			if level == auth.AccessFull && definesWriteScope {
				notGranted = append(notGranted, svc+" (write access)")
			}
		case wasRequested && hasAny(readOnly):
			// Some read scopes were unticked. Recorded only for requested
			// services: read scopes are shared between services, so for an
			// unrequested one this would just echo another service.
			access[svc] = auth.AccessPartialReadOnly
			notGranted = append(notGranted, svc+" (only some scopes)")
		case wasRequested:
			notGranted = append(notGranted, svc)
		}
	}

	// `people` shares the contacts scopes; when contacts itself is
	// recorded, a partial people entry is only an echo of it.
	if _, ok := access["contacts"]; ok && access["people"] == auth.AccessPartial {
		if _, requestedPeople := requested["people"]; !requestedPeople {
			delete(access, "people")
		}
	}
	return access, notGranted
}

// grantedServiceNames returns the service names in access, in the
// canonical service order.
func grantedServiceNames(access map[string]string) []string {
	var names []string
	for _, svc := range auth.ValidServiceNames() {
		if _, ok := access[svc]; ok {
			names = append(names, svc)
		}
	}
	return names
}

// allReadOnly reports whether every granted service is read-only.
func allReadOnly(access map[string]string) bool {
	if len(access) == 0 {
		return false
	}
	for _, level := range access {
		if level != auth.AccessReadOnly && level != auth.AccessPartialReadOnly {
			return false
		}
	}
	return true
}

func runLogout(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	// Nothing stored at all: a no-op that needs neither the lock nor a
	// writable config directory.
	if !auth.TokenExists() && !auth.GrantedMetadataExists() {
		return p.Print(map[string]interface{}{
			"status":  "success",
			"message": "Not authenticated (no token found)",
		})
	}

	// Logout and login write the same files; holding the login lock keeps
	// a logout from deleting metadata that a login is in the middle of
	// writing (which would leave a token without its access record).
	unlock, err := auth.LockLoginState()
	if err != nil {
		return p.PrintError(err)
	}
	defer unlock()

	if !auth.TokenExists() {
		auth.DeleteGrantedMetadata()
		return p.Print(map[string]interface{}{
			"status":  "success",
			"message": "Not authenticated (no token found)",
		})
	}

	// Best-effort server-side revocation with timeout before deleting local token
	token, err := auth.LoadToken()
	if err == nil && token != nil {
		ctx, cancel := context.WithTimeout(context.Background(), revokeTimeout)
		defer cancel()
		if revokeErr := auth.RevokeToken(ctx, token); revokeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to revoke token server-side: %v\n", revokeErr)
		}
	}

	if err := auth.DeleteToken(); err != nil {
		return p.PrintError(err)
	}
	// The recorded services and access levels describe the grant that was
	// just removed.
	auth.DeleteGrantedMetadata()

	return p.Print(map[string]interface{}{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

func runStatus(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	if config.GetAccessToken() != "" {
		// The token itself is never printed, and it is not verified here:
		// no network call is made, so it is never claimed valid.
		return p.Print(map[string]interface{}{
			"configured": true,
			"auth_mode":  "access-token",
			"message":    "Using an externally managed access token (GWS_ACCESS_TOKEN / --access-token); gws does not refresh it and it is not verified until a command runs",
		})
	}

	if modeCfg, err := resolveConfiguredAuthMode(); err != nil {
		return usageErrorf("%s", err.Error())
	} else if modeCfg.Mode != auth.AuthModeOAuth {
		// Configured is not authenticated: only a local key check is
		// performed here — no token is minted and no network call made,
		// so credentials are never claimed valid beyond that.
		result := map[string]interface{}{
			"configured": true,
			"auth_mode":  modeCfg.Mode,
			"key_file":   modeCfg.KeyFile,
		}
		if modeCfg.ImpersonateUser != "" {
			result["impersonate_user"] = modeCfg.ImpersonateUser
		}
		if err := auth.ValidateServiceAccountKey(modeCfg); err != nil {
			result["key_valid"] = false
			result["error"] = err.Error()
			result["message"] = "Service-account mode configured but the key file failed local validation"
		} else {
			result["key_valid"] = true
			result["message"] = "Service-account mode configured; key file is locally valid (credentials are not verified against Google until a command runs)"
		}
		return p.Print(result)
	}

	token, err := auth.LoadToken()
	if err != nil {
		return p.Print(map[string]interface{}{
			"authenticated": false,
			"message":       "Not authenticated, run: gws auth login",
		})
	}

	// Resolve the token once through the token source, which refreshes
	// it when it is expired or about to expire, and save any refresh
	// (including a rotated refresh token). Everything below — the account
	// lookup and the access record — then uses that same token, so no
	// later call refreshes again and drops the result.
	expired := token.Expiry.Before(time.Now()) && !token.Expiry.IsZero()
	clientID := config.GetClientID()
	clientSecret := config.GetClientSecret()
	if clientID == "" || clientSecret == "" {
		if expired {
			return p.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Token expired, run: gws auth login",
			})
		}
	} else {
		current, saveErr, err := auth.EnsureFreshToken(context.Background(), clientID, clientSecret, token)
		switch {
		case errors.Is(err, auth.ErrRefreshBusy):
			// Another command is refreshing this login right now. Status
			// must not refresh alongside it, and cannot say anything
			// reliable until it is done.
			return p.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Another gws command is refreshing the stored login; run gws auth status again",
			})
		case errors.Is(err, auth.ErrLoginRemoved):
			// Logged out while this command was starting.
			return p.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Not authenticated, run: gws auth login",
			})
		case errors.Is(err, auth.ErrLoginChanged):
			// Another login replaced the stored one: report on that one,
			// not on the token this command happened to load first.
			replaced, loadErr := auth.LoadToken()
			if loadErr != nil {
				return p.Print(map[string]interface{}{
					"authenticated": false,
					"message":       "Not authenticated, run: gws auth login",
				})
			}
			token = replaced
		case err != nil && expired:
			return p.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Token expired and refresh failed, run: gws auth login",
			})
		case err == nil:
			warnIfRefreshNotSaved(os.Stderr, saveErr)
			token = current
		}
	}

	// Get user info
	result := map[string]interface{}{
		"authenticated": true,
		"expires":       token.Expiry.Format(time.RFC3339),
	}
	// The recorded access levels are local: report them whether or not
	// the user-info lookup below succeeds.
	// Read for the same token the account lookup below uses, so the email
	// and the access levels always describe one login.
	addGrantedAccess(result, auth.LoadGrantedAccessFor(token))

	if userInfo, err := getUserInfo(token); err != nil {
		result["user"] = "unknown (failed to fetch user info)"
	} else {
		result["email"] = userInfo.Email
	}
	return p.Print(result)
}

// getUserInfo asks Google who the token belongs to. It uses the token
// exactly as given and never refreshes it: every refresh of the stored
// login goes through auth.EnsureFreshToken, under the login lock, and a
// second, private refresh here could race it. Callers pass a token they
// have already made fresh.
func getUserInfo(token *oauth2.Token) (*oauth2api.Userinfo, error) {
	ctx := context.Background()
	svc, err := oauth2api.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	if err != nil {
		return nil, err
	}

	return svc.Userinfo.Get().Do()
}

// flagString returns a string flag's value, or "" if it is not defined.
func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// hasServiceSelection reports whether the login names specific services,
// through a non-empty --services value or a configured services list.
func hasServiceSelection(cmd *cobra.Command) bool {
	return len(splitList(flagString(cmd, "services"))) > 0 || len(config.GetServices()) > 0
}

// resolveServices determines which services to authorize from the
// --services flag, the config file, or (by default) every service.
func resolveServices(cmd *cobra.Command) []string {
	// Blank entries are dropped, so `--services=` or `--services ,` counts
	// as no selection rather than as a selection of nothing.
	if services := splitList(flagString(cmd, "services")); len(services) > 0 {
		return services
	}
	if configServices := config.GetServices(); len(configServices) > 0 {
		return configServices
	}
	return auth.ValidServiceNames()
}
