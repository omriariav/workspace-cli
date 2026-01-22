package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
)

// OAuthClient handles the OAuth2 authentication flow.
type OAuthClient struct {
	config *oauth2.Config
}

// NewOAuthClient creates a new OAuth client with the given credentials.
func NewOAuthClient(clientID, clientSecret string) *OAuthClient {
	return &OAuthClient{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       AllScopes,
		},
	}
}

// Login performs the OAuth2 authorization flow with PKCE.
func (c *OAuthClient) Login(ctx context.Context) (*oauth2.Token, error) {
	// Generate PKCE code verifier and challenge
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	challenge := generateCodeChallenge(verifier)

	// Find an available port for the callback server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	c.config.RedirectURL = redirectURL

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Build authorization URL with PKCE
	authURL := c.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("access_type", "offline"),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start callback server
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			// Verify state
			if r.URL.Query().Get("state") != state {
				errChan <- fmt.Errorf("invalid state parameter")
				http.Error(w, "Invalid state parameter", http.StatusBadRequest)
				return
			}

			// Check for errors
			if errMsg := r.URL.Query().Get("error"); errMsg != "" {
				errChan <- fmt.Errorf("authorization failed: %s", errMsg)
				http.Error(w, "Authorization failed", http.StatusBadRequest)
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				errChan <- fmt.Errorf("no authorization code received")
				http.Error(w, "No authorization code", http.StatusBadRequest)
				return
			}

			// Send success response
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>gws - Authorization Successful</title></head>
<body style="font-family: system-ui; text-align: center; padding: 50px;">
<h1>✓ Authorization Successful</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)

			codeChan <- code
		}),
	}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Open browser
	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Wait for callback or timeout
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		server.Shutdown(ctx)
		return nil, err
	case <-time.After(5 * time.Minute):
		server.Shutdown(ctx)
		return nil, fmt.Errorf("authorization timeout")
	case <-ctx.Done():
		server.Shutdown(ctx)
		return nil, ctx.Err()
	}

	// Shutdown callback server
	server.Shutdown(ctx)

	// Exchange code for token with PKCE verifier
	token, err := c.config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// GetConfig returns the OAuth2 config for creating authenticated clients.
func (c *OAuthClient) GetConfig() *oauth2.Config {
	return c.config
}

// GetTokenSource returns a token source that automatically refreshes the token.
func GetTokenSource(ctx context.Context, clientID, clientSecret string, token *oauth2.Token) oauth2.TokenSource {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       AllScopes,
	}
	return config.TokenSource(ctx, token)
}

// generateCodeVerifier generates a random PKCE code verifier.
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge creates a S256 code challenge from a verifier.
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState generates a random state string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
