package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/user"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	const q = `
		insert into users (id, email, phone, display_name, password_hash, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, q, u.ID, u.Email, u.Phone, u.DisplayName, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	if isUniqueViolation(err) {
		return user.ErrEmailTaken
	}
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	const q = `
		select id, email, phone, display_name, password_hash, email_verified_at, created_at, updated_at
		from users where id = $1 and deleted_at is null`
	return scanUser(r.pool.QueryRow(ctx, q, id))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	const q = `
		select id, email, phone, display_name, password_hash, email_verified_at, created_at, updated_at
		from users where email = $1 and deleted_at is null`
	return scanUser(r.pool.QueryRow(ctx, q, email))
}

func scanUser(row pgx.Row) (*user.User, error) {
	var u user.User
	err := row.Scan(&u.ID, &u.Email, &u.Phone, &u.DisplayName, &u.PasswordHash, &u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
