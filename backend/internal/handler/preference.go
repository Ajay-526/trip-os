package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/preference"
	"github.com/tripos/backend/internal/domain/trip"
	appmw "github.com/tripos/backend/internal/middleware"
	"github.com/tripos/backend/internal/platform/httpx"
	"github.com/tripos/backend/internal/service"
)

type PreferenceHandler struct {
	preferences *service.PreferenceService
}

func NewPreferenceHandler(preferences *service.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{preferences: preferences}
}

type upsertPreferenceRequest struct {
	BudgetPaise       *int64   `json:"budget_paise"`
	Pace              string   `json:"pace"`
	Interests         []string `json:"interests"`
	Dietary           []string `json:"dietary"`
	MobilityNotes     *string  `json:"mobility_notes"`
	EarliestStartTime *string  `json:"earliest_start_time"`
	LatestEndTime     *string  `json:"latest_end_time"`
	DateFlexibility   string   `json:"date_flexibility"`
	Notes             *string  `json:"notes"`
}

func (h *PreferenceHandler) UpsertMine(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	var req upsertPreferenceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}
	p, err := h.preferences.UpsertMine(r.Context(), callerID, preference.UpsertInput{
		TripID: tripID, BudgetPaise: req.BudgetPaise, Pace: preference.Pace(req.Pace),
		Interests: req.Interests, Dietary: req.Dietary, MobilityNotes: req.MobilityNotes,
		EarliestStartTime: req.EarliestStartTime, LatestEndTime: req.LatestEndTime,
		DateFlexibility: preference.DateFlexibility(req.DateFlexibility), Notes: req.Notes,
	})
	if err != nil {
		writePreferenceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"preference": toPreferenceDTO(p)})
}

func (h *PreferenceHandler) List(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	prefs, err := h.preferences.ListForTrip(r.Context(), tripID, callerID)
	if err != nil {
		writePreferenceError(w, err)
		return
	}
	items := make([]PreferenceDTO, 0, len(prefs))
	for _, p := range prefs {
		if p == nil {
			continue
		}
		items = append(items, toPreferenceDTO(p))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func writePreferenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotMember), errors.Is(err, trip.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "trip not found")
	case errors.Is(err, preference.ErrNegativeBudget), errors.Is(err, preference.ErrInvalidPace),
		errors.Is(err, preference.ErrInvalidFlex), errors.Is(err, preference.ErrInvalidTimeSpan):
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "something went wrong")
	}
}
