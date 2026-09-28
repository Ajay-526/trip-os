package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/session"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, t *session.RefreshToken) error {
	const q = `
		insert into refresh_tokens (id, user_id, token_hash, issued_at, expires_at)
		values ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, q, t.ID, t.UserID, t.TokenHash, t.IssuedAt, t.ExpiresAt)
	return err
}

func (r *SessionRepo) GetByHash(ctx context.Context, tokenHash string) (*session.RefreshToken, error) {
	const q = `
		select id, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by
		from refresh_tokens where token_hash = $1`
	var t session.RefreshToken
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &t.RevokedAt, &t.ReplacedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *SessionRepo) Revoke(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error {
	const q = `update refresh_tokens set revoked_at = now(), replaced_by = $2 where id = $1`
	_, err := r.pool.Exec(ctx, q, id, replacedBy)
	return err
}

func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const q = `update refresh_tokens set revoked_at = now() where user_id = $1 and revoked_at is null`
	_, err := r.pool.Exec(ctx, q, userID)
	return err
}
