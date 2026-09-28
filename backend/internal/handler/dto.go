package handler

import (
	"time"

	"github.com/tripos/backend/internal/domain/membership"
	"github.com/tripos/backend/internal/domain/preference"
	"github.com/tripos/backend/internal/domain/trip"
)

const dateLayout = "2006-01-02"

func dateToStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

func strToDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type TripDTO struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	DestinationQuery string  `json:"destination_query"`
	StartDate        *string `json:"start_date"`
	EndDate          *string `json:"end_date"`
	BudgetTotalPaise *int64  `json:"budget_total_paise"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
	CreatedBy        string  `json:"created_by"`
	CreatedAt        string  `json:"created_at"`
}

func toTripDTO(t *trip.Trip) TripDTO {
	return TripDTO{
		ID:               t.ID.String(),
		Name:             t.Name,
		DestinationQuery: t.DestinationQuery,
		StartDate:        dateToStr(t.StartDate),
		EndDate:          dateToStr(t.EndDate),
		BudgetTotalPaise: t.BudgetTotalPaise,
		Currency:         t.Currency,
		Status:           string(t.Status),
		CreatedBy:        t.CreatedBy.String(),
		CreatedAt:        t.CreatedAt.Format(time.RFC3339),
	}
}

type MembershipDTO struct {
	ID           string  `json:"id"`
	TripID       string  `json:"trip_id"`
	UserID       *string `json:"user_id"`
	InvitedEmail *string `json:"invited_email"`
	Role         string  `json:"role"`
	Status       string  `json:"status"`
	JoinedAt     *string `json:"joined_at"`
}

func toMembershipDTO(m *membership.Membership) MembershipDTO {
	dto := MembershipDTO{
		ID:           m.ID.String(),
		TripID:       m.TripID.String(),
		InvitedEmail: m.InvitedEmail,
		Role:         string(m.Role),
		Status:       string(m.Status),
	}
	if m.UserID != nil {
		s := m.UserID.String()
		dto.UserID = &s
	}
	if m.JoinedAt != nil {
		s := m.JoinedAt.Format(time.RFC3339)
		dto.JoinedAt = &s
	}
	return dto
}

type PreferenceDTO struct {
	TripID            string   `json:"trip_id"`
	UserID            string   `json:"user_id"`
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

func toPreferenceDTO(p *preference.Preference) PreferenceDTO {
	return PreferenceDTO{
		TripID:            p.TripID.String(),
		UserID:            p.UserID.String(),
		BudgetPaise:       p.BudgetPaise,
		Pace:              string(p.Pace),
		Interests:         p.Interests,
		Dietary:           p.Dietary,
		MobilityNotes:     p.MobilityNotes,
		EarliestStartTime: p.EarliestStartTime,
		LatestEndTime:     p.LatestEndTime,
		DateFlexibility:   string(p.DateFlexibility),
		Notes:             p.Notes,
	}
}
