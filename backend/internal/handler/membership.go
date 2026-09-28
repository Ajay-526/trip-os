package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/membership"
	"github.com/tripos/backend/internal/domain/trip"
	appmw "github.com/tripos/backend/internal/middleware"
	"github.com/tripos/backend/internal/platform/httpx"
	"github.com/tripos/backend/internal/service"
)

type MembershipHandler struct {
	memberships *service.MembershipService
	publicWebBase string // e.g. https://app.tripos.in — used to build shareable invite URLs
}

func NewMembershipHandler(memberships *service.MembershipService, publicWebBase string) *MembershipHandler {
	return &MembershipHandler{memberships: memberships, publicWebBase: publicWebBase}
}

type createInviteRequest struct {
	Email           *string `json:"email"`
	Role            string  `json:"role"`
	Link            bool    `json:"link"`
	MaxUses         *int    `json:"max_uses"`
	ExpiresInHours  *int    `json:"expires_in_hours"`
}

func (h *MembershipHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	var req createInviteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}

	if req.Link {
		var expiresIn *time.Duration
		if req.ExpiresInHours != nil {
			d := time.Duration(*req.ExpiresInHours) * time.Hour
			expiresIn = &d
		}
		result, err := h.memberships.CreateInviteLink(r.Context(), tripID, callerID, req.MaxUses, expiresIn)
		if err != nil {
			writeMembershipError(w, err)
			return
		}
		url := fmt.Sprintf("%s/invites/%s/accept", h.publicWebBase, result.RawToken)
		httpx.JSON(w, http.StatusCreated, map[string]any{
			"invite_link": map[string]any{
				"url": url, "expires_at": result.Link.ExpiresAt, "max_uses": result.Link.MaxUses,
			},
		})
		return
	}

	if req.Email == nil || *req.Email == "" {
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email is required for a direct invite")
		return
	}
	role := membership.Role(req.Role)
	if role == "" {
		role = membership.RoleMember
	}
	m, err := h.memberships.InviteByEmail(r.Context(), tripID, callerID, *req.Email, role)
	if err != nil {
		writeMembershipError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"membership": toMembershipDTO(m)})
}

func (h *MembershipHandler) List(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	members, err := h.memberships.ListByTrip(r.Context(), tripID, callerID)
	if err != nil {
		writeMembershipError(w, err)
		return
	}
	items := make([]MembershipDTO, 0, len(members))
	for _, m := range members {
		items = append(items, toMembershipDTO(m))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type updateMembershipRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

func (h *MembershipHandler) Update(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	membershipID, err := uuid.Parse(chi.URLParam(r, "membershipId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid membership id")
		return
	}
	var req updateMembershipRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}
	var role *membership.Role
	if req.Role != nil {
		r := membership.Role(*req.Role)
		role = &r
	}
	var status *membership.Status
	if req.Status != nil {
		s := membership.Status(*req.Status)
		status = &s
	}
	m, err := h.memberships.UpdateRoleOrStatus(r.Context(), tripID, callerID, membershipID, role, status)
	if err != nil {
		writeMembershipError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"membership": toMembershipDTO(m)})
}

// AcceptInviteLink accepts a shareable invite link token, at
// POST /invites/{token}/accept (not trip-scoped, since the caller doesn't
// know the trip ID from the token alone).
func (h *MembershipHandler) AcceptInviteLink(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	token := chi.URLParam(r, "token")
	m, err := h.memberships.AcceptInviteLink(r.Context(), token, callerID)
	if err != nil {
		writeMembershipError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"membership": toMembershipDTO(m)})
}

func writeMembershipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotMember), errors.Is(err, trip.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "trip not found")
	case errors.Is(err, membership.ErrNotAuthorized):
		httpx.Error(w, http.StatusForbidden, "NOT_AUTHORIZED", err.Error())
	case errors.Is(err, membership.ErrAlreadyInvited), errors.Is(err, membership.ErrAlreadyMember):
		httpx.Error(w, http.StatusConflict, "ALREADY_INVITED", err.Error())
	case errors.Is(err, membership.ErrInviteExpired):
		httpx.Error(w, http.StatusGone, "INVITE_EXPIRED", err.Error())
	case errors.Is(err, membership.ErrInviteExhausted):
		httpx.Error(w, http.StatusConflict, "INVITE_EXHAUSTED", err.Error())
	case errors.Is(err, membership.ErrInviteNotFound), errors.Is(err, membership.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "INVITE_NOT_FOUND", err.Error())
	case errors.Is(err, membership.ErrInvalidEmail), errors.Is(err, membership.ErrInvalidRole),
		errors.Is(err, membership.ErrCannotRemoveOwner):
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "something went wrong")
	}
}
