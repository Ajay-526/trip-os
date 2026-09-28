package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/preference"
)

type PreferenceRepo struct {
	pool *pgxpool.Pool
}

func NewPreferenceRepo(pool *pgxpool.Pool) *PreferenceRepo {
	return &PreferenceRepo{pool: pool}
}

func (r *PreferenceRepo) Upsert(ctx context.Context, p *preference.Preference) error {
	const q = `
		insert into trip_member_preferences (id, trip_id, user_id, budget_paise, pace, interests,
			dietary, mobility_notes, earliest_start_time, latest_end_time, date_flexibility, notes,
			created_at, updated_at)
		values ($1, $2, $3, $4, nullif($5,''), $6, $7, $8, nullif($9,'')::time, nullif($10,'')::time,
			nullif($11,''), $12, $13, $14)
		on conflict (trip_id, user_id) do update set
			budget_paise = excluded.budget_paise,
			pace = excluded.pace,
			interests = excluded.interests,
			dietary = excluded.dietary,
			mobility_notes = excluded.mobility_notes,
			earliest_start_time = excluded.earliest_start_time,
			latest_end_time = excluded.latest_end_time,
			date_flexibility = excluded.date_flexibility,
			notes = excluded.notes,
			updated_at = excluded.updated_at`
	_, err := r.pool.Exec(ctx, q, p.ID, p.TripID, p.UserID, p.BudgetPaise, string(p.Pace),
		p.Interests, p.Dietary, p.MobilityNotes, deref(p.EarliestStartTime), deref(p.LatestEndTime),
		string(p.DateFlexibility), p.Notes, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *PreferenceRepo) GetByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*preference.Preference, error) {
	const q = `
		select id, trip_id, user_id, budget_paise, coalesce(pace::text, ''), interests, dietary,
			mobility_notes, coalesce(to_char(earliest_start_time, 'HH24:MI'), ''), coalesce(to_char(latest_end_time, 'HH24:MI'), ''),
			coalesce(date_flexibility::text, ''), notes, created_at, updated_at
		from trip_member_preferences where trip_id = $1 and user_id = $2`
	p, err := scanPreference(r.pool.QueryRow(ctx, q, tripID, userID))
	if errors.Is(err, preference.ErrNotFound) {
		return nil, nil
	}
	return p, err
}

func (r *PreferenceRepo) ListByTrip(ctx context.Context, tripID uuid.UUID) ([]*preference.Preference, error) {
	const q = `
		select id, trip_id, user_id, budget_paise, coalesce(pace::text, ''), interests, dietary,
			mobility_notes, coalesce(to_char(earliest_start_time, 'HH24:MI'), ''), coalesce(to_char(latest_end_time, 'HH24:MI'), ''),
			coalesce(date_flexibility::text, ''), notes, created_at, updated_at
		from trip_member_preferences where trip_id = $1`
	rows, err := r.pool.Query(ctx, q, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*preference.Preference
	for rows.Next() {
		p, err := scanPreferenceRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanPreference(row pgx.Row) (*preference.Preference, error) {
	p, err := scanPreferenceRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, preference.ErrNotFound
	}
	return p, err
}

func scanPreferenceRow(row rowScanner) (*preference.Preference, error) {
	var p preference.Preference
	var pace, earliest, latest, flex string
	err := row.Scan(&p.ID, &p.TripID, &p.UserID, &p.BudgetPaise, &pace, &p.Interests, &p.Dietary,
		&p.MobilityNotes, &earliest, &latest, &flex, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Pace = preference.Pace(pace)
	p.DateFlexibility = preference.DateFlexibility(flex)
	if earliest != "" {
		p.EarliestStartTime = &earliest
	}
	if latest != "" {
		p.LatestEndTime = &latest
	}
	return &p, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
