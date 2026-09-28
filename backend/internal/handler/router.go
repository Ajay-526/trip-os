package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	appmw "github.com/tripos/backend/internal/middleware"
	"github.com/tripos/backend/internal/platform/jwt"
)

type Deps struct {
	Issuer       *jwt.Issuer
	CSRFSecret   string
	AllowedOrigin string
	Auth         *AuthHandler
	Trips        *TripHandler
	Memberships  *MembershipHandler
	Preferences  *PreferenceHandler
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	if d.AllowedOrigin != "" {
		r.Use(appmw.CORS(d.AllowedOrigin))
	}

	requireAuth := appmw.RequireAuth(d.Issuer)
	requireCSRF := appmw.RequireCSRF(d.CSRFSecret)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})

		api.Route("/auth", func(a chi.Router) {
			a.Post("/signup", d.Auth.SignUp)
			a.Post("/login", d.Auth.Login)
			a.Post("/refresh", d.Auth.Refresh)
			a.With(requireAuth).Get("/csrf", d.Auth.CSRF)
			a.With(requireAuth).Get("/me", d.Auth.Me)
			a.With(requireAuth, requireCSRF).Post("/logout", d.Auth.Logout)
		})

		// Everything below requires an authenticated session; every
		// mutating verb additionally requires a valid CSRF token.
		api.Group(func(auth chi.Router) {
			auth.Use(requireAuth)

			auth.Route("/trips", func(tr chi.Router) {
				tr.With(requireCSRF).Post("/", d.Trips.Create)
				tr.Get("/", d.Trips.List)
				tr.Get("/{tripId}", d.Trips.Get)
				tr.With(requireCSRF).Patch("/{tripId}", d.Trips.Update)
				tr.With(requireCSRF).Delete("/{tripId}", d.Trips.Delete)

				tr.With(requireCSRF).Post("/{tripId}/invites", d.Memberships.CreateInvite)
				tr.Get("/{tripId}/members", d.Memberships.List)
				tr.With(requireCSRF).Patch("/{tripId}/members/{membershipId}", d.Memberships.Update)

				tr.With(requireCSRF).Put("/{tripId}/preferences/me", d.Preferences.UpsertMine)
				tr.Get("/{tripId}/preferences", d.Preferences.List)
			})

			auth.With(requireCSRF).Post("/invites/{token}/accept", d.Memberships.AcceptInviteLink)
		})
	})

	return r
}
