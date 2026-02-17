package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

const googleRevokeEndpoint = "https://oauth2.googleapis.com/revoke"

// RevokeToken revokes the given OAuth token server-side via Google's revocation endpoint.
// Prefers revoking the refresh token (which invalidates both tokens), falling back
// to the access token if no refresh token is available.
func RevokeToken(ctx context.Context, token *oauth2.Token) error {
	return revokeWithEndpoint(ctx, googleRevokeEndpoint, token)
}

// revokeWithEndpoint is the internal implementation, accepting an endpoint for testability.
func revokeWithEndpoint(ctx context.Context, endpoint string, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("no token to revoke")
	}

	// Prefer refresh token — revoking it invalidates both access and refresh tokens
	tokenValue := token.RefreshToken
	if tokenValue == "" {
		tokenValue = token.AccessToken
	}
	if tokenValue == "" {
		return fmt.Errorf("token has no refresh or access token to revoke")
	}

	form := url.Values{"token": {tokenValue}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revocation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("revocation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("revocation failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
