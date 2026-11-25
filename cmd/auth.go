package cmd

import (
	"context"
	"fmt"
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
	if add {
		// --add is for adding specific services. Without a selection it
		// would fall back to every service and ask for full access to all
		// of them, the opposite of an incremental request.
		if !hasServiceSelection(cmd) {
			return usageErrorf("--add requires --services: name the service(s) to add, e.g. gws auth login --add --services drive")
		}
		existingAccess = existingGrantedAccess()
	}
	access := mergeGrantedAccess(existingAccess, services, readOnly)
	scopes := loginScopes(services, readOnly, hasServiceSelection(cmd))
	if add {
		// Ask for the whole merged grant explicitly. Google's incremental
		// authorization (include_granted_scopes) is not dependable for
		// Desktop OAuth clients, so the previous scopes are re-requested
		// rather than assumed to carry over.
		scopes = scopesForAccess(access)
	}

	if config.GetAccessToken() != "" {
		fmt.Fprintln(os.Stderr, "Warning: GWS_ACCESS_TOKEN / --access-token is set and takes precedence over the token this login stores.")
	}

	client := auth.NewOAuthClient(clientID, clientSecret, scopes)
	client.SetIncludeGrantedScopes(add)

	ctx := context.Background()
	token, err := client.Login(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	// Report what Google actually granted, not what was requested: the
	// user can untick scopes on the consent screen.
	if granted, ok := token.Extra("scope").(string); ok && strings.TrimSpace(granted) != "" {
		var notGranted []string
		access, notGranted = deriveGrantedAccess(access, strings.Fields(granted))
		// An incremental login must not cost the user access they already
		// had. If the new grant no longer covers a previously authorized
		// service, keep the stored authorization instead.
		if lost := lostServices(existingAccess, access); len(lost) > 0 {
			return p.PrintError(fmt.Errorf("the new grant does not include previously authorized access for %s, so the stored authorization was left unchanged; approve every requested scope, or run `gws auth login` without --add to replace it", strings.Join(lost, ", ")))
		}
		if len(notGranted) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: Google did not grant the requested access for: %s\n", strings.Join(notGranted, ", "))
		}
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
	errAccess := auth.SaveGrantedAccess(access, token)

	if err := auth.SaveToken(token); err != nil {
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

// scopesForAccess returns the scopes that make up the given per-service
// access levels: the read-only set for read-only services and the full
// set for everything else. Always includes userinfo scopes.
func scopesForAccess(access map[string]string) []string {
	var full, readOnly []string
	for _, svc := range auth.ValidServiceNames() {
		switch access[svc] {
		case "":
		case auth.AccessReadOnly, auth.AccessPartialReadOnly:
			readOnly = append(readOnly, svc)
		default:
			full = append(full, svc)
		}
	}
	seen := map[string]bool{}
	var scopes []string
	for _, s := range append(auth.ScopesForServices(full), auth.ReadOnlyScopesForServices(readOnly)...) {
		if !seen[s] {
			seen[s] = true
			scopes = append(scopes, s)
		}
	}
	return scopes
}

// accessRank orders access levels from least to most capable.
func accessRank(level string) int {
	switch level {
	case auth.AccessFull:
		return 4
	case auth.AccessPartial:
		return 3
	case auth.AccessReadOnly:
		return 2
	case auth.AccessPartialReadOnly:
		return 1
	}
	return 0
}

// lostServices lists the services whose access in after is lower than it
// was in before (or gone), in canonical order.
func lostServices(before, after map[string]string) []string {
	var lost []string
	for _, svc := range auth.ValidServiceNames() {
		if prev, ok := before[svc]; ok && accessRank(after[svc]) < accessRank(prev) {
			lost = append(lost, svc)
		}
	}
	return lost
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
		if readOnly && access[svc] == auth.AccessFull {
			continue
		}
		access[svc] = level
	}
	return access
}

// deriveGrantedAccess builds the per-service access levels from the
// complete set of scopes Google returned. Every known service is examined,
// not just the requested ones: incremental authorization can return scopes
// granted in an earlier login that local metadata no longer knows about.
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

	// Check if token is expired
	if token.Expiry.Before(time.Now()) && !token.Expiry.IsZero() {
		// Try to get user info to trigger refresh
		clientID := config.GetClientID()
		clientSecret := config.GetClientSecret()

		if clientID == "" || clientSecret == "" {
			return p.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Token expired, run: gws auth login",
			})
		}

		// Try refreshing
		ctx := context.Background()
		ts := auth.GetTokenSource(ctx, clientID, clientSecret, token)
		newToken, err := ts.Token()
		if err != nil {
			return p.Print(map[string]interface{}{
				"authenticated": false,
				"message":       "Token expired and refresh failed, run: gws auth login",
			})
		}

		// Save refreshed token
		_ = auth.SaveToken(newToken)
		token = newToken
	}

	// Get user info
	userInfo, err := getUserInfo(token)
	if err != nil {
		return p.Print(map[string]interface{}{
			"authenticated": true,
			"expires":       token.Expiry.Format(time.RFC3339),
			"user":          "unknown (failed to fetch user info)",
		})
	}

	result := map[string]interface{}{
		"authenticated": true,
		"email":         userInfo.Email,
		"expires":       token.Expiry.Format(time.RFC3339),
	}
	if access := auth.LoadGrantedAccess(); len(access) > 0 {
		result["services"] = access
		result["read_only"] = allReadOnly(access)
	}
	return p.Print(result)
}

func getUserInfo(token *oauth2.Token) (*oauth2api.Userinfo, error) {
	ctx := context.Background()
	clientID := config.GetClientID()
	clientSecret := config.GetClientSecret()

	ts := auth.GetTokenSource(ctx, clientID, clientSecret, token)
	svc, err := oauth2api.NewService(ctx, option.WithTokenSource(ts))
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
