package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/membership"
	"github.com/tripos/backend/internal/domain/trip"
)

// ErrNotMember signals the caller is not on this trip at all. Handlers must
// map this to HTTP 404, not 403 — we do not reveal a trip's existence to
// non-members.
var ErrNotMember = errors.New("service: caller is not a member of this trip")

type TripService struct {
	trips       trip.Repository
	memberships membership.Repository
}

func NewTripService(trips trip.Repository, memberships membership.Repository) *TripService {
	return &TripService{trips: trips, memberships: memberships}
}

func (s *TripService) Create(ctx context.Context, ownerID uuid.UUID, in trip.CreateInput) (*trip.Trip, error) {
	in.CreatedBy = ownerID
	t, err := trip.New(in)
	if err != nil {
		return nil, err
	}
	if err := s.trips.Create(ctx, t); err != nil {
		return nil, err
	}
	owner := membership.NewOwner(t.ID, ownerID)
	if err := s.memberships.Create(ctx, owner); err != nil {
		return nil, err
	}
	return t, nil
}

// membershipFor loads the caller's membership row for a trip, or ErrNotMember.
func (s *TripService) membershipFor(ctx context.Context, tripID, callerID uuid.UUID) (*membership.Membership, error) {
	m, err := s.memberships.GetByTripAndUser(ctx, tripID, callerID)
	if err != nil || m == nil || m.Status != membership.StatusJoined {
		return nil, ErrNotMember
	}
	return m, nil
}

func (s *TripService) Get(ctx context.Context, tripID, callerID uuid.UUID) (*trip.Trip, *membership.Membership, error) {
	m, err := s.membershipFor(ctx, tripID, callerID)
	if err != nil {
		return nil, nil, err
	}
	t, err := s.trips.GetByID(ctx, tripID)
	if err != nil {
		return nil, nil, err
	}
	return t, m, nil
}

func (s *TripService) ListForUser(ctx context.Context, callerID uuid.UUID) ([]*trip.Trip, error) {
	return s.trips.ListForUser(ctx, callerID)
}

func (s *TripService) Update(ctx context.Context, tripID, callerID uuid.UUID, in trip.UpdateInput) (*trip.Trip, error) {
	m, err := s.membershipFor(ctx, tripID, callerID)
	if err != nil {
		return nil, err
	}
	if !m.Role.CanManageTrip() {
		return nil, trip.ErrNotAuthorized
	}
	t, err := s.trips.GetByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if err := t.Apply(in); err != nil {
		return nil, err
	}
	if err := s.trips.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TripService) Delete(ctx context.Context, tripID, callerID uuid.UUID) error {
	m, err := s.membershipFor(ctx, tripID, callerID)
	if err != nil {
		return err
	}
	if m.Role != membership.RoleOwner {
		return trip.ErrNotAuthorized
	}
	return s.trips.SoftDelete(ctx, tripID)
}
