package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndParse_RoundTrip(t *testing.T) {
	issuer := NewIssuer("test-secret", 15*time.Minute)
	userID := uuid.New()

	token, expiry, err := issuer.IssueAccessToken(userID)
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
	if !expiry.After(time.Now().UTC()) {
		t.Errorf("expected expiry to be in the future")
	}

	parsed, err := issuer.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("unexpected error parsing token: %v", err)
	}
	if parsed != userID {
		t.Errorf("expected parsed user id %s, got %s", userID, parsed)
	}
}

func TestParseAccessToken_RejectsGarbage(t *testing.T) {
	issuer := NewIssuer("test-secret", 15*time.Minute)
	_, err := issuer.ParseAccessToken("not-a-jwt")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseAccessToken_RejectsWrongSecret(t *testing.T) {
	issuerA := NewIssuer("secret-a", 15*time.Minute)
	issuerB := NewIssuer("secret-b", 15*time.Minute)

	token, _, _ := issuerA.IssueAccessToken(uuid.New())
	_, err := issuerB.ParseAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for token signed with a different secret, got %v", err)
	}
}

func TestParseAccessToken_RejectsExpiredToken(t *testing.T) {
	issuer := NewIssuer("test-secret", -1*time.Minute) // already expired
	token, _, _ := issuer.IssueAccessToken(uuid.New())

	_, err := issuer.ParseAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}
