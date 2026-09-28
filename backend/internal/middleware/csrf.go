package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/tripos/backend/internal/platform/httpx"
)

const CSRFCookieName = "trip_os_csrf"
const CSRFHeaderName = "X-CSRF-Token"

// CSRFToken derives a deterministic, unguessable token from the session
// value and a server secret (double-submit pattern): the browser can read
// it (it's not httpOnly) and must echo it back in a custom header, which a
// cross-site form post cannot do.
func CSRFToken(secret, sessionValue string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionValue))
	return hex.EncodeToString(mac.Sum(nil))
}

// RequireCSRF rejects state-changing requests unless the header token
// matches the token derived from the caller's own session cookie.
func RequireCSRF(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			session, err := r.Cookie(SessionCookieName)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "sign in required")
				return
			}
			want := CSRFToken(secret, session.Value)
			got := r.Header.Get(CSRFHeaderName)
			if got == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
				httpx.Error(w, http.StatusForbidden, "CSRF_INVALID", "missing or invalid CSRF token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
