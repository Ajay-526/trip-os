package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/membership"
)

type MembershipRepo struct {
	pool *pgxpool.Pool
}

func NewMembershipRepo(pool *pgxpool.Pool) *MembershipRepo {
	return &MembershipRepo{pool: pool}
}

func (r *MembershipRepo) Create(ctx context.Context, m *membership.Membership) error {
	const q = `
		insert into trip_memberships (id, trip_id, user_id, invited_email, role, status,
			invited_by, invited_at, joined_at, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.pool.Exec(ctx, q, m.ID, m.TripID, m.UserID, m.InvitedEmail, m.Role, m.Status,
		m.InvitedBy, m.InvitedAt, m.JoinedAt, m.CreatedAt, m.UpdatedAt)
	if isUniqueViolation(err) {
		return membership.ErrAlreadyMember
	}
	return err
}

const membershipCols = `id, trip_id, user_id, invited_email, role, status, invited_by, invited_at, joined_at, created_at, updated_at`

func (r *MembershipRepo) GetByID(ctx context.Context, id uuid.UUID) (*membership.Membership, error) {
	q := `select ` + membershipCols + ` from trip_memberships where id = $1`
	return scanMembership(r.pool.QueryRow(ctx, q, id))
}

func (r *MembershipRepo) GetByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*membership.Membership, error) {
	q := `select ` + membershipCols + ` from trip_memberships where trip_id = $1 and user_id = $2`
	m, err := scanMembership(r.pool.QueryRow(ctx, q, tripID, userID))
	if errors.Is(err, membership.ErrNotFound) {
		return nil, nil
	}
	return m, err
}

func (r *MembershipRepo) GetByTripAndEmail(ctx context.Context, tripID uuid.UUID, email string) (*membership.Membership, error) {
	q := `select ` + membershipCols + ` from trip_memberships where trip_id = $1 and invited_email = $2`
	m, err := scanMembership(r.pool.QueryRow(ctx, q, tripID, email))
	if errors.Is(err, membership.ErrNotFound) {
		return nil, nil
	}
	return m, err
}

func (r *MembershipRepo) ListByTrip(ctx context.Context, tripID uuid.UUID) ([]*membership.Membership, error) {
	q := `select ` + membershipCols + ` from trip_memberships where trip_id = $1 order by created_at asc`
	rows, err := r.pool.Query(ctx, q, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*membership.Membership
	for rows.Next() {
		m, err := scanMembershipRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MembershipRepo) Update(ctx context.Context, m *membership.Membership) error {
	const q = `
		update trip_memberships set user_id = $2, role = $3, status = $4, joined_at = $5, updated_at = $6
		where id = $1`
	tag, err := r.pool.Exec(ctx, q, m.ID, m.UserID, m.Role, m.Status, m.JoinedAt, m.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return membership.ErrNotFound
	}
	return nil
}

func scanMembership(row pgx.Row) (*membership.Membership, error) {
	m, err := scanMembershipRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, membership.ErrNotFound
	}
	return m, err
}

func scanMembershipRow(row rowScanner) (*membership.Membership, error) {
	var m membership.Membership
	err := row.Scan(&m.ID, &m.TripID, &m.UserID, &m.InvitedEmail, &m.Role, &m.Status,
		&m.InvitedBy, &m.InvitedAt, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// --- Invite links ---

type InviteLinkRepo struct {
	pool *pgxpool.Pool
}

func NewInviteLinkRepo(pool *pgxpool.Pool) *InviteLinkRepo {
	return &InviteLinkRepo{pool: pool}
}

func (r *InviteLinkRepo) Create(ctx context.Context, l *membership.InviteLink) error {
	const q = `
		insert into trip_invite_links (id, trip_id, token_hash, created_by, max_uses, expires_at, created_at)
		values ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, q, l.ID, l.TripID, l.TokenHash, l.CreatedBy, l.MaxUses, l.ExpiresAt, l.CreatedAt)
	return err
}

func (r *InviteLinkRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*membership.InviteLink, error) {
	const q = `
		select id, trip_id, token_hash, created_by, max_uses, use_count, expires_at, revoked_at, created_at
		from trip_invite_links where token_hash = $1`
	var l membership.InviteLink
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&l.ID, &l.TripID, &l.TokenHash, &l.CreatedBy, &l.MaxUses, &l.UseCount, &l.ExpiresAt, &l.RevokedAt, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *InviteLinkRepo) IncrementUse(ctx context.Context, id uuid.UUID) error {
	const q = `update trip_invite_links set use_count = use_count + 1 where id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	return err
}
