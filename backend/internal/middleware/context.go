package middleware

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const userIDKey ctxKey = iota

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext returns the authenticated caller's user ID. Handlers
// behind RequireAuth can rely on this being present.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
