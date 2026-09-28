package user

import "testing"

func TestNormalize_Valid(t *testing.T) {
	in := NewUserInput{Email: "  Ajay@Example.com ", DisplayName: "  Ajay Kumar ", Password: "correcthorse"}
	email, name, err := in.Normalize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "ajay@example.com" {
		t.Errorf("expected trimmed+lowercased email, got %q", email)
	}
	if name != "Ajay Kumar" {
		t.Errorf("expected trimmed display name, got %q", name)
	}
}

func TestNormalize_RejectsInvalidEmail(t *testing.T) {
	in := NewUserInput{Email: "not-an-email", DisplayName: "A", Password: "correcthorse"}
	_, _, err := in.Normalize()
	if err != ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestNormalize_RejectsEmptyDisplayName(t *testing.T) {
	in := NewUserInput{Email: "a@example.com", DisplayName: "   ", Password: "correcthorse"}
	_, _, err := in.Normalize()
	if err != ErrInvalidName {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestNormalize_RejectsShortPassword(t *testing.T) {
	in := NewUserInput{Email: "a@example.com", DisplayName: "A", Password: "short"}
	_, _, err := in.Normalize()
	if err != ErrWeakPassword {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

func TestNormalize_RejectsOverlongDisplayName(t *testing.T) {
	long := make([]byte, 81)
	for i := range long {
		long[i] = 'a'
	}
	in := NewUserInput{Email: "a@example.com", DisplayName: string(long), Password: "correcthorse"}
	_, _, err := in.Normalize()
	if err != ErrInvalidName {
		t.Fatalf("expected ErrInvalidName for overlong name, got %v", err)
	}
}
