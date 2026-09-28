package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/membership"
	"github.com/tripos/backend/internal/domain/preference"
)

type PreferenceService struct {
	preferences preference.Repository
	memberships membership.Repository
}

func NewPreferenceService(preferences preference.Repository, memberships membership.Repository) *PreferenceService {
	return &PreferenceService{preferences: preferences, memberships: memberships}
}

func (s *PreferenceService) requireJoined(ctx context.Context, tripID, callerID uuid.UUID) (*membership.Membership, error) {
	m, err := s.memberships.GetByTripAndUser(ctx, tripID, callerID)
	if err != nil || m == nil || m.Status != membership.StatusJoined {
		return nil, ErrNotMember
	}
	return m, nil
}

func (s *PreferenceService) UpsertMine(ctx context.Context, callerID uuid.UUID, in preference.UpsertInput) (*preference.Preference, error) {
	if _, err := s.requireJoined(ctx, in.TripID, callerID); err != nil {
		return nil, err
	}
	in.UserID = callerID
	p, err := preference.New(in)
	if err != nil {
		return nil, err
	}
	if err := s.preferences.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ListForTrip returns all members' preferences to organizers (needed as AI
// planning context), or just the caller's own preference otherwise.
func (s *PreferenceService) ListForTrip(ctx context.Context, tripID, callerID uuid.UUID) ([]*preference.Preference, error) {
	m, err := s.requireJoined(ctx, tripID, callerID)
	if err != nil {
		return nil, err
	}
	if m.Role.CanManageTrip() {
		return s.preferences.ListByTrip(ctx, tripID)
	}
	mine, err := s.preferences.GetByTripAndUser(ctx, tripID, callerID)
	if err != nil {
		return []*preference.Preference{}, nil
	}
	return []*preference.Preference{mine}, nil
}
