package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/tripos/backend/internal/domain/user"
	appmw "github.com/tripos/backend/internal/middleware"
	"github.com/tripos/backend/internal/platform/httpx"
	"github.com/tripos/backend/internal/service"
)

const refreshCookieName = "trip_os_refresh"

type AuthHandler struct {
	auth         *service.AuthService
	users        user.Repository
	csrfSecret   string
	cookieDomain string
	cookieSecure bool
	crossSiteCookies bool
}

func NewAuthHandler(auth *service.AuthService, users user.Repository, csrfSecret, cookieDomain string, cookieSecure, crossSiteCookies bool) *AuthHandler {
	return &AuthHandler{auth: auth, users: users, csrfSecret: csrfSecret, cookieDomain: cookieDomain, cookieSecure: cookieSecure, crossSiteCookies: crossSiteCookies}
}

func (h *AuthHandler) sameSite() http.SameSite {
	if h.crossSiteCookies {
		return http.SameSiteNoneMode
	}
	return http.SameSiteStrictMode
}

type signupRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}
	u, err := h.auth.SignUp(r.Context(), user.NewUserInput{
		Email: req.Email, DisplayName: req.DisplayName, Password: req.Password,
	})
	if err != nil {
		writeUserError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"user": u.ToPublic()})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}
	result, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect")
		return
	}
	h.setSessionCookies(w, result)
	httpx.JSON(w, http.StatusOK, map[string]any{"user": result.User.ToPublic()})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil {
		_ = h.auth.Logout(r.Context(), c.Value)
	}
	h.clearSessionCookies(w)
	httpx.NoContent(w)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no refresh token")
		return
	}
	result, err := h.auth.Refresh(r.Context(), c.Value)
	if err != nil {
		h.clearSessionCookies(w)
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token invalid or expired")
		return
	}
	h.setSessionCookies(w, result)
	httpx.JSON(w, http.StatusOK, map[string]any{"user": result.User.ToPublic()})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "sign in required")
		return
	}
	u, err := h.users.GetByID(r.Context(), userID)
	if err != nil || u == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "sign in required")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"user": u.ToPublic()})
}

func (h *AuthHandler) CSRF(w http.ResponseWriter, r *http.Request) {
	sess, err := r.Cookie(appmw.SessionCookieName)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "sign in required")
		return
	}
	token := appmw.CSRFToken(h.csrfSecret, sess.Value)
	httpx.JSON(w, http.StatusOK, map[string]any{"csrf_token": token})
}

func (h *AuthHandler) setSessionCookies(w http.ResponseWriter, result *service.AuthResult) {
	http.SetCookie(w, &http.Cookie{
		Name: appmw.SessionCookieName, Value: result.AccessToken, Path: "/",
		Domain: h.cookieDomain, Expires: result.AccessTokenExpiry,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: h.sameSite(),
	})
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookieName, Value: result.RefreshToken, Path: "/api/v1/auth/refresh",
		Domain: h.cookieDomain, Expires: result.RefreshTokenExpiry,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: h.sameSite(),
	})
}

func (h *AuthHandler) clearSessionCookies(w http.ResponseWriter) {
	past := time.Unix(0, 0)
	http.SetCookie(w, &http.Cookie{Name: appmw.SessionCookieName, Value: "", Path: "/", Domain: h.cookieDomain, Expires: past, HttpOnly: true, Secure: h.cookieSecure, SameSite: h.sameSite()})
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/v1/auth/refresh", Domain: h.cookieDomain, Expires: past, HttpOnly: true, Secure: h.cookieSecure, SameSite: h.sameSite()})
}

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, "EMAIL_TAKEN", err.Error())
	case errors.Is(err, user.ErrInvalidEmail), errors.Is(err, user.ErrInvalidName), errors.Is(err, user.ErrWeakPassword):
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "something went wrong")
	}
}
