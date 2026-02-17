package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

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
	viper.BindPFlag("client_id", loginCmd.Flags().Lookup("client-id"))
	viper.BindPFlag("client_secret", loginCmd.Flags().Lookup("client-secret"))
}

func runLogin(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	clientID := config.GetClientID()
	clientSecret := config.GetClientSecret()

	if clientID == "" || clientSecret == "" {
		return p.PrintError(fmt.Errorf("missing credentials: set GWS_CLIENT_ID and GWS_CLIENT_SECRET environment variables, or use --client-id and --client-secret flags"))
	}

	// Determine scopes based on --services flag, config, or all
	scopes := resolveScopes(cmd)

	client := auth.NewOAuthClient(clientID, clientSecret, scopes)

	ctx := context.Background()
	token, err := client.Login(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	// Merge with existing token to preserve refresh token
	existing, _ := auth.LoadToken()
	token = auth.MergeToken(existing, token)

	if err := auth.SaveToken(token); err != nil {
		return p.PrintError(err)
	}

	return p.Print(map[string]interface{}{
		"status":  "success",
		"message": "Authentication successful",
		"expires": token.Expiry.Format(time.RFC3339),
	})
}

func runLogout(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	if !auth.TokenExists() {
		return p.Print(map[string]interface{}{
			"status":  "success",
			"message": "Not authenticated (no token found)",
		})
	}

	// Best-effort server-side revocation before deleting local token
	token, err := auth.LoadToken()
	if err == nil && token != nil {
		ctx := context.Background()
		if revokeErr := auth.RevokeToken(ctx, token); revokeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to revoke token server-side: %v\n", revokeErr)
		}
	}

	if err := auth.DeleteToken(); err != nil {
		return p.PrintError(err)
	}

	return p.Print(map[string]interface{}{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

func runStatus(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

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

	return p.Print(map[string]interface{}{
		"authenticated": true,
		"email":         userInfo.Email,
		"expires":       token.Expiry.Format(time.RFC3339),
	})
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

// resolveScopes determines which scopes to request based on the --services flag,
// config file, or defaults to all scopes.
func resolveScopes(cmd *cobra.Command) []string {
	// 1. Check --services flag
	if servicesFlag, _ := cmd.Flags().GetString("services"); servicesFlag != "" {
		services := strings.Split(servicesFlag, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
		return auth.ScopesForServices(services)
	}

	// 2. Check config file
	if configServices := config.GetServices(); len(configServices) > 0 {
		return auth.ScopesForServices(configServices)
	}

	// 3. Default: all scopes
	return auth.AllScopes
}
