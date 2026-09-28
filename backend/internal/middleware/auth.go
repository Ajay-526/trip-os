package middleware

import (
	"net/http"

	"github.com/tripos/backend/internal/platform/httpx"
	"github.com/tripos/backend/internal/platform/jwt"
)

const SessionCookieName = "trip_os_session"

// RequireAuth reads the access-token cookie, validates it, and injects the
// caller's user ID into the request context. It never trusts a header the
// browser could set on its own (no "X-User-Id" style backdoor).
func RequireAuth(issuer *jwt.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "sign in required")
				return
			}
			userID, err := issuer.ParseAccessToken(cookie.Value)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "session expired or invalid")
				return
			}
			ctx := withUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
