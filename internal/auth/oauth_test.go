package auth

import (
	"encoding/base64"
	"regexp"
	"testing"
)

var base64URLNoPadding = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestGenerateCodeVerifier(t *testing.T) {
	v, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier failed: %v", err)
	}

	// RFC 7636 section 4.1: 43 to 128 characters from the unreserved set.
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("verifier length %d outside RFC 7636 range 43-128", len(v))
	}
	if !base64URLNoPadding.MatchString(v) {
		t.Errorf("verifier %q contains characters outside base64url without padding", v)
	}
	raw, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		t.Fatalf("verifier is not valid raw base64url: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("expected 32 random bytes, got %d", len(raw))
	}

	other, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier failed: %v", err)
	}
	if v == other {
		t.Error("two verifiers should not be equal")
	}
}

func TestGenerateCodeChallenge_RFC7636AppendixB(t *testing.T) {
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const want = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if got := generateCodeChallenge(verifier); got != want {
		t.Errorf("generateCodeChallenge(%q) = %q, want %q", verifier, got, want)
	}
}

func TestGenerateCodeChallenge_Properties(t *testing.T) {
	a := generateCodeChallenge("verifier-a")
	if a != generateCodeChallenge("verifier-a") {
		t.Error("challenge must be deterministic for the same verifier")
	}
	if a == generateCodeChallenge("verifier-b") {
		t.Error("different verifiers must produce different challenges")
	}
	// SHA-256 is 32 bytes, so 43 chars in raw base64url.
	if len(a) != 43 || !base64URLNoPadding.MatchString(a) {
		t.Errorf("unexpected challenge format: %q", a)
	}
}

func TestGenerateState(t *testing.T) {
	s, err := generateState()
	if err != nil {
		t.Fatalf("generateState failed: %v", err)
	}

	if !base64URLNoPadding.MatchString(s) {
		t.Errorf("state %q contains characters outside base64url without padding", s)
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("state is not valid raw base64url: %v", err)
	}
	if len(raw) != 16 {
		t.Errorf("expected 16 random bytes, got %d", len(raw))
	}

	other, err := generateState()
	if err != nil {
		t.Fatalf("generateState failed: %v", err)
	}
	if s == other {
		t.Error("two states should not be equal")
	}
}
