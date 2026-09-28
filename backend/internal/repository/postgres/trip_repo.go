package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/trip"
)

type TripRepo struct {
	pool *pgxpool.Pool
}

func NewTripRepo(pool *pgxpool.Pool) *TripRepo {
	return &TripRepo{pool: pool}
}

func (r *TripRepo) Create(ctx context.Context, t *trip.Trip) error {
	const q = `
		insert into trips (id, name, destination_query, start_date, end_date, budget_total_paise,
			currency, status, created_by, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.pool.Exec(ctx, q, t.ID, t.Name, t.DestinationQuery, t.StartDate, t.EndDate,
		t.BudgetTotalPaise, t.Currency, t.Status, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *TripRepo) GetByID(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	const q = `
		select id, name, destination_query, start_date, end_date, budget_total_paise,
			currency, status, created_by, created_at, updated_at, deleted_at
		from trips where id = $1 and deleted_at is null`
	return scanTrip(r.pool.QueryRow(ctx, q, id))
}

func (r *TripRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]*trip.Trip, error) {
	const q = `
		select t.id, t.name, t.destination_query, t.start_date, t.end_date, t.budget_total_paise,
			t.currency, t.status, t.created_by, t.created_at, t.updated_at, t.deleted_at
		from trips t
		join trip_memberships m on m.trip_id = t.id
		where m.user_id = $1 and m.status = 'joined' and t.deleted_at is null
		order by t.created_at desc`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*trip.Trip
	for rows.Next() {
		t, err := scanTripRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TripRepo) Update(ctx context.Context, t *trip.Trip) error {
	const q = `
		update trips set name = $2, start_date = $3, end_date = $4, budget_total_paise = $5,
			status = $6, updated_at = $7
		where id = $1 and deleted_at is null`
	tag, err := r.pool.Exec(ctx, q, t.ID, t.Name, t.StartDate, t.EndDate, t.BudgetTotalPaise, t.Status, t.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return trip.ErrNotFound
	}
	return nil
}

func (r *TripRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const q = `update trips set deleted_at = now(), updated_at = now() where id = $1 and deleted_at is null`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return trip.ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(row pgx.Row) (*trip.Trip, error) {
	t, err := scanTripRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, trip.ErrNotFound
	}
	return t, err
}

func scanTripRow(row rowScanner) (*trip.Trip, error) {
	var t trip.Trip
	err := row.Scan(&t.ID, &t.Name, &t.DestinationQuery, &t.StartDate, &t.EndDate,
		&t.BudgetTotalPaise, &t.Currency, &t.Status, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
