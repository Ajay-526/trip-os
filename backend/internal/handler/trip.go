package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/trip"
	appmw "github.com/tripos/backend/internal/middleware"
	"github.com/tripos/backend/internal/platform/httpx"
	"github.com/tripos/backend/internal/service"
)

type TripHandler struct {
	trips *service.TripService
}

func NewTripHandler(trips *service.TripService) *TripHandler {
	return &TripHandler{trips: trips}
}

type createTripRequest struct {
	Name             string  `json:"name"`
	DestinationQuery string  `json:"destination_query"`
	StartDate        *string `json:"start_date"`
	EndDate          *string `json:"end_date"`
	BudgetTotalPaise *int64  `json:"budget_total_paise"`
	Currency         string  `json:"currency"`
}

func (h *TripHandler) Create(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	var req createTripRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}
	start, err := strToDate(req.StartDate)
	if err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "start_date must be YYYY-MM-DD")
		return
	}
	end, err := strToDate(req.EndDate)
	if err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "end_date must be YYYY-MM-DD")
		return
	}
	t, err := h.trips.Create(r.Context(), callerID, trip.CreateInput{
		Name: req.Name, DestinationQuery: req.DestinationQuery,
		StartDate: start, EndDate: end, BudgetTotalPaise: req.BudgetTotalPaise, Currency: req.Currency,
	})
	if err != nil {
		writeTripError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"trip": toTripDTO(t)})
}

func (h *TripHandler) List(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	trips, err := h.trips.ListForUser(r.Context(), callerID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list trips")
		return
	}
	items := make([]TripDTO, 0, len(trips))
	for _, t := range trips {
		items = append(items, toTripDTO(t))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *TripHandler) Get(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	t, m, err := h.trips.Get(r.Context(), tripID, callerID)
	if err != nil {
		writeTripError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"trip": toTripDTO(t), "membership": toMembershipDTO(m)})
}

type updateTripRequest struct {
	Name             *string `json:"name"`
	StartDate        *string `json:"start_date"`
	EndDate          *string `json:"end_date"`
	BudgetTotalPaise *int64  `json:"budget_total_paise"`
	Status           *string `json:"status"`
}

func (h *TripHandler) Update(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	var req updateTripRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "MALFORMED_JSON", "could not parse request body")
		return
	}
	in := trip.UpdateInput{Name: req.Name}
	if req.StartDate != nil {
		d, err := strToDate(req.StartDate)
		if err != nil {
			httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "start_date must be YYYY-MM-DD")
			return
		}
		in.StartDate = &d
	}
	if req.EndDate != nil {
		d, err := strToDate(req.EndDate)
		if err != nil {
			httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "end_date must be YYYY-MM-DD")
			return
		}
		in.EndDate = &d
	}
	if req.BudgetTotalPaise != nil {
		in.BudgetTotalPaise = &req.BudgetTotalPaise
	}
	if req.Status != nil {
		s := trip.Status(*req.Status)
		in.Status = &s
	}
	t, err := h.trips.Update(r.Context(), tripID, callerID, in)
	if err != nil {
		writeTripError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"trip": toTripDTO(t)})
}

func (h *TripHandler) Delete(w http.ResponseWriter, r *http.Request) {
	callerID, _ := appmw.UserIDFromContext(r.Context())
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_ID", "invalid trip id")
		return
	}
	if err := h.trips.Delete(r.Context(), tripID, callerID); err != nil {
		writeTripError(w, err)
		return
	}
	httpx.NoContent(w)
}

func writeTripError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotMember), errors.Is(err, trip.ErrNotFound):
		// Deliberately 404 (not 403) for non-members — a trip's existence
		// is not revealed to callers who are not on it.
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "trip not found")
	case errors.Is(err, trip.ErrNotAuthorized):
		httpx.Error(w, http.StatusForbidden, "NOT_AUTHORIZED", err.Error())
	case errors.Is(err, trip.ErrNameRequired), errors.Is(err, trip.ErrDestRequired),
		errors.Is(err, trip.ErrBadDateRange), errors.Is(err, trip.ErrNegativeBudget),
		errors.Is(err, trip.ErrInvalidStatus):
		httpx.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "something went wrong")
	}
}
