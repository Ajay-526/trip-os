package membership

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner       Role = "owner"
	RoleCoOrganizer Role = "co_organizer"
	RoleMember      Role = "member"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleCoOrganizer, RoleMember:
		return true
	}
	return false
}

// CanManageTrip reports whether this role can edit trip settings, invite
// members, generate itineraries, and create polls.
func (r Role) CanManageTrip() bool {
	return r == RoleOwner || r == RoleCoOrganizer
}

type Status string

const (
	StatusInvited Status = "invited"
	StatusJoined  Status = "joined"
	StatusDeclined Status = "declined"
	StatusRemoved Status = "removed"
)

var (
	ErrNotFound           = errors.New("membership: not found")
	ErrAlreadyMember      = errors.New("membership: user is already a member of this trip")
	ErrAlreadyInvited     = errors.New("membership: email already invited to this trip")
	ErrInvalidEmail       = errors.New("membership: invalid email")
	ErrInvalidRole        = errors.New("membership: invalid role")
	ErrCannotRemoveOwner  = errors.New("membership: cannot remove the trip owner")
	ErrInviteExpired      = errors.New("membership: invite link expired")
	ErrInviteExhausted    = errors.New("membership: invite link has reached its use limit")
	ErrInviteNotFound     = errors.New("membership: invite not found")
	ErrNotAuthorized      = errors.New("membership: not authorized")
)

// Membership ties a user (or a pending invited email) to a trip with a role.
type Membership struct {
	ID           uuid.UUID
	TripID       uuid.UUID
	UserID       *uuid.UUID
	InvitedEmail *string
	Role         Role
	Status       Status
	InvitedBy    *uuid.UUID
	InvitedAt    time.Time
	JoinedAt     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewOwner(tripID, userID uuid.UUID) *Membership {
	now := time.Now().UTC()
	return &Membership{
		ID:        uuid.New(),
		TripID:    tripID,
		UserID:    &userID,
		Role:      RoleOwner,
		Status:    StatusJoined,
		InvitedAt: now,
		JoinedAt:  &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewEmailInvite(tripID uuid.UUID, email string, role Role, invitedBy uuid.UUID) (*Membership, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(email) < 3 {
		return nil, ErrInvalidEmail
	}
	if !role.Valid() || role == RoleOwner {
		return nil, ErrInvalidRole
	}
	now := time.Now().UTC()
	return &Membership{
		ID:           uuid.New(),
		TripID:       tripID,
		InvitedEmail: &email,
		Role:         role,
		Status:       StatusInvited,
		InvitedBy:    &invitedBy,
		InvitedAt:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (m *Membership) Accept(userID uuid.UUID) {
	m.UserID = &userID
	m.Status = StatusJoined
	now := time.Now().UTC()
	m.JoinedAt = &now
	m.UpdatedAt = now
}

// InviteLink is a shareable, multi-use join link for a trip.
type InviteLink struct {
	ID        uuid.UUID
	TripID    uuid.UUID
	TokenHash string
	CreatedBy uuid.UUID
	MaxUses   *int
	UseCount  int
	ExpiresAt *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (l *InviteLink) Usable(now time.Time) error {
	if l.RevokedAt != nil {
		return ErrInviteExpired
	}
	if l.ExpiresAt != nil && now.After(*l.ExpiresAt) {
		return ErrInviteExpired
	}
	if l.MaxUses != nil && l.UseCount >= *l.MaxUses {
		return ErrInviteExhausted
	}
	return nil
}

type Repository interface {
	Create(ctx context.Context, m *Membership) error
	GetByID(ctx context.Context, id uuid.UUID) (*Membership, error)
	GetByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*Membership, error)
	GetByTripAndEmail(ctx context.Context, tripID uuid.UUID, email string) (*Membership, error)
	ListByTrip(ctx context.Context, tripID uuid.UUID) ([]*Membership, error)
	Update(ctx context.Context, m *Membership) error
}

type InviteLinkRepository interface {
	Create(ctx context.Context, l *InviteLink) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*InviteLink, error)
	IncrementUse(ctx context.Context, id uuid.UUID) error
}
