package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a rotating refresh token. Only its hash is ever
// persisted; the raw token is handed to the client once, in an httpOnly
// cookie, and never stored or logged.
type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID
}

func (t *RefreshToken) Active(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

type Repository interface {
	Create(ctx context.Context, t *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}
