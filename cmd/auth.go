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
	loginCmd.Flags().Bool("readonly", false, "Request only read-only scopes for the selected services (events subscriptions can still be managed: they are authorized by read scopes)")
	loginCmd.Flags().Bool("add", false, "Add the selected services to the existing authorization instead of replacing it")
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
		existingAccess = existingGrantedAccess()
	}
	scopes := loginScopes(services, readOnly, cmd.Flags().Changed("services") || len(config.GetServices()) > 0)
	access := mergeGrantedAccess(existingAccess, services, readOnly)

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
		if len(notGranted) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: Google did not grant the requested access for: %s\n", strings.Join(notGranted, ", "))
		}
	}

	token, err = loginTokenToStore(token)
	if err != nil {
		return p.PrintError(err)
	}

	if err := auth.SaveToken(token); err != nil {
		return p.PrintError(err)
	}

	// Persist granted services and access levels for scope diagnostics.
	// If either write fails, remove both files: no metadata is better than
	// metadata that still describes the previous token.
	errServices := auth.SaveGrantedServices(grantedServiceNames(access))
	errAccess := auth.SaveGrantedAccess(access)
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
	access := map[string]string{}
	for _, svc := range auth.LoadGrantedServices() {
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
// A service is full when every one of its full scopes is present,
// read-only when only its read-only scopes are, and absent otherwise.
// Where the two sets coincide (e.g. groups, whose scopes are all
// read-only) the requested label is kept, defaulting to read-only.
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

	access = map[string]string{}
	for _, svc := range auth.ValidServiceNames() {
		level, wasRequested := requested[svc]
		one := []string{svc}
		full := auth.ScopesForServices(one)
		readOnly := auth.ReadOnlyScopesForServices(one)

		switch {
		case hasAll(full) && (len(full) > len(readOnly) || level == auth.AccessFull):
			access[svc] = auth.AccessFull
		case hasAll(readOnly):
			access[svc] = auth.AccessReadOnly
			if level == auth.AccessFull {
				notGranted = append(notGranted, svc+" (write access)")
			}
		case wasRequested:
			notGranted = append(notGranted, svc)
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
		if level != auth.AccessReadOnly {
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

// resolveServices determines which services to authorize from the
// --services flag, the config file, or (by default) every service.
func resolveServices(cmd *cobra.Command) []string {
	if servicesFlag, _ := cmd.Flags().GetString("services"); servicesFlag != "" {
		services := strings.Split(servicesFlag, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
		return services
	}
	if configServices := config.GetServices(); len(configServices) > 0 {
		return configServices
	}
	return auth.ValidServiceNames()
}
