package user

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("user: not found")
	ErrEmailTaken    = errors.New("user: email already registered")
	ErrInvalidEmail  = errors.New("user: invalid email")
	ErrInvalidName   = errors.New("user: display name must be 1-80 characters")
	ErrWeakPassword  = errors.New("user: password must be at least 8 characters")
	ErrBadCredential = errors.New("user: invalid credentials")
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// User is the identity aggregate root. PasswordHash is never serialized to
// API responses — callers must use ToPublic() at the transport boundary.
type User struct {
	ID              uuid.UUID
	Email           string
	Phone           *string
	DisplayName     string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Public is the safe, external representation of a User.
type Public struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
}

func (u *User) ToPublic() Public {
	return Public{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName}
}

// NewUserInput validates and normalizes signup input. It never touches
// persistence — the service layer is responsible for hashing the password
// and checking uniqueness against the repository.
type NewUserInput struct {
	Email       string
	DisplayName string
	Password    string
}

func (in NewUserInput) Normalize() (email, displayName string, err error) {
	email = strings.ToLower(strings.TrimSpace(in.Email))
	displayName = strings.TrimSpace(in.DisplayName)

	if !emailRe.MatchString(email) {
		return "", "", ErrInvalidEmail
	}
	if len(displayName) == 0 || len(displayName) > 80 {
		return "", "", ErrInvalidName
	}
	if len(in.Password) < 8 {
		return "", "", ErrWeakPassword
	}
	return email, displayName, nil
}

// Repository is the persistence boundary for the user aggregate. Domain and
// service code depend only on this interface — never on a concrete SQL
// driver — so persistence can be swapped or faked in tests.
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}
