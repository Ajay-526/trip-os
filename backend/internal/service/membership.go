package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/membership"
)

type MembershipService struct {
	memberships membership.Repository
	inviteLinks membership.InviteLinkRepository
}

func NewMembershipService(memberships membership.Repository, inviteLinks membership.InviteLinkRepository) *MembershipService {
	return &MembershipService{memberships: memberships, inviteLinks: inviteLinks}
}

func (s *MembershipService) requireManager(ctx context.Context, tripID, callerID uuid.UUID) (*membership.Membership, error) {
	m, err := s.memberships.GetByTripAndUser(ctx, tripID, callerID)
	if err != nil || m == nil || m.Status != membership.StatusJoined {
		return nil, ErrNotMember
	}
	if !m.Role.CanManageTrip() {
		return nil, membership.ErrNotAuthorized
	}
	return m, nil
}

func (s *MembershipService) InviteByEmail(ctx context.Context, tripID, callerID uuid.UUID, email string, role membership.Role) (*membership.Membership, error) {
	if _, err := s.requireManager(ctx, tripID, callerID); err != nil {
		return nil, err
	}
	if existing, _ := s.memberships.GetByTripAndEmail(ctx, tripID, email); existing != nil {
		return nil, membership.ErrAlreadyInvited
	}
	m, err := membership.NewEmailInvite(tripID, email, role, callerID)
	if err != nil {
		return nil, err
	}
	if err := s.memberships.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

type InviteLinkResult struct {
	RawToken string
	Link     *membership.InviteLink
}

func (s *MembershipService) CreateInviteLink(ctx context.Context, tripID, callerID uuid.UUID, maxUses *int, expiresIn *time.Duration) (*InviteLinkResult, error) {
	if _, err := s.requireManager(ctx, tripID, callerID); err != nil {
		return nil, err
	}
	raw, hash, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}
	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().UTC().Add(*expiresIn)
		expiresAt = &t
	}
	link := &membership.InviteLink{
		ID:        uuid.New(),
		TripID:    tripID,
		TokenHash: hash,
		CreatedBy: callerID,
		MaxUses:   maxUses,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.inviteLinks.Create(ctx, link); err != nil {
		return nil, err
	}
	return &InviteLinkResult{RawToken: raw, Link: link}, nil
}

// AcceptInviteLink joins the caller to a trip via a shareable link token.
func (s *MembershipService) AcceptInviteLink(ctx context.Context, rawToken string, callerID uuid.UUID) (*membership.Membership, error) {
	hash := hashOpaqueToken(rawToken)
	link, err := s.inviteLinks.GetByTokenHash(ctx, hash)
	if err != nil || link == nil {
		return nil, membership.ErrInviteNotFound
	}
	if err := link.Usable(time.Now().UTC()); err != nil {
		return nil, err
	}
	if existing, _ := s.memberships.GetByTripAndUser(ctx, link.TripID, callerID); existing != nil {
		return existing, nil // idempotent re-join
	}
	m := &membership.Membership{
		ID:        uuid.New(),
		TripID:    link.TripID,
		UserID:    &callerID,
		Role:      membership.RoleMember,
		Status:    membership.StatusJoined,
		InvitedBy: &link.CreatedBy,
		InvitedAt: time.Now().UTC(),
	}
	now := time.Now().UTC()
	m.JoinedAt = &now
	m.CreatedAt = now
	m.UpdatedAt = now
	if err := s.memberships.Create(ctx, m); err != nil {
		return nil, err
	}
	if err := s.inviteLinks.IncrementUse(ctx, link.ID); err != nil {
		return nil, err
	}
	return m, nil
}

// AcceptEmailInvite joins the caller (an authenticated user whose email
// matches the invite) to the trip.
func (s *MembershipService) AcceptEmailInvite(ctx context.Context, membershipID uuid.UUID, callerID uuid.UUID) (*membership.Membership, error) {
	m, err := s.memberships.GetByID(ctx, membershipID)
	if err != nil || m == nil {
		return nil, membership.ErrInviteNotFound
	}
	if m.Status != membership.StatusInvited {
		return nil, membership.ErrAlreadyMember
	}
	m.Accept(callerID)
	if err := s.memberships.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MembershipService) ListByTrip(ctx context.Context, tripID, callerID uuid.UUID) ([]*membership.Membership, error) {
	caller, err := s.memberships.GetByTripAndUser(ctx, tripID, callerID)
	if err != nil || caller == nil || caller.Status != membership.StatusJoined {
		return nil, ErrNotMember
	}
	return s.memberships.ListByTrip(ctx, tripID)
}

func (s *MembershipService) UpdateRoleOrStatus(ctx context.Context, tripID, callerID, targetMembershipID uuid.UUID, newRole *membership.Role, newStatus *membership.Status) (*membership.Membership, error) {
	caller, err := s.requireManager(ctx, tripID, callerID)
	if err != nil {
		return nil, err
	}
	if caller.Role != membership.RoleOwner {
		return nil, membership.ErrNotAuthorized
	}
	target, err := s.memberships.GetByID(ctx, targetMembershipID)
	if err != nil || target == nil || target.TripID != tripID {
		return nil, membership.ErrNotFound
	}
	if target.Role == membership.RoleOwner && newStatus != nil && *newStatus == membership.StatusRemoved {
		return nil, membership.ErrCannotRemoveOwner
	}
	if newRole != nil {
		if !newRole.Valid() || *newRole == membership.RoleOwner {
			return nil, membership.ErrInvalidRole
		}
		target.Role = *newRole
	}
	if newStatus != nil {
		target.Status = *newStatus
	}
	target.UpdatedAt = time.Now().UTC()
	if err := s.memberships.Update(ctx, target); err != nil {
		return nil, err
	}
	return target, nil
}

func newOpaqueToken() (raw, hash string, err error) {
	buf := make([]byte, 24)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	hash = hashOpaqueToken(raw)
	return raw, hash, nil
}

func hashOpaqueToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
