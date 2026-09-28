package trip

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft      Status = "draft"
	StatusPlanning   Status = "planning"
	StatusVoting     Status = "voting"
	StatusBooked     Status = "booked"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusPlanning, StatusVoting, StatusBooked, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

var (
	ErrNotFound        = errors.New("trip: not found")
	ErrNameRequired    = errors.New("trip: name must be 1-140 characters")
	ErrDestRequired    = errors.New("trip: destination_query is required")
	ErrBadDateRange    = errors.New("trip: end_date must not be before start_date")
	ErrNegativeBudget  = errors.New("trip: budget must not be negative")
	ErrInvalidStatus   = errors.New("trip: invalid status")
	ErrNotAuthorized   = errors.New("trip: not authorized")
)

// Trip is the aggregate root for a single trip.
type Trip struct {
	ID                uuid.UUID
	Name              string
	DestinationQuery  string
	StartDate         *time.Time
	EndDate           *time.Time
	BudgetTotalPaise  *int64
	Currency          string
	Status            Status
	CreatedBy         uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// CreateInput is the validated shape needed to construct a new Trip.
type CreateInput struct {
	Name             string
	DestinationQuery string
	StartDate        *time.Time
	EndDate          *time.Time
	BudgetTotalPaise *int64
	Currency         string
	CreatedBy        uuid.UUID
}

func New(in CreateInput) (*Trip, error) {
	name := strings.TrimSpace(in.Name)
	if len(name) == 0 || len(name) > 140 {
		return nil, ErrNameRequired
	}
	dest := strings.TrimSpace(in.DestinationQuery)
	if len(dest) == 0 {
		return nil, ErrDestRequired
	}
	if in.StartDate != nil && in.EndDate != nil && in.EndDate.Before(*in.StartDate) {
		return nil, ErrBadDateRange
	}
	if in.BudgetTotalPaise != nil && *in.BudgetTotalPaise < 0 {
		return nil, ErrNegativeBudget
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "INR"
	}
	now := time.Now().UTC()
	return &Trip{
		ID:               uuid.New(),
		Name:             name,
		DestinationQuery: dest,
		StartDate:        in.StartDate,
		EndDate:          in.EndDate,
		BudgetTotalPaise: in.BudgetTotalPaise,
		Currency:         currency,
		Status:           StatusDraft,
		CreatedBy:        in.CreatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// UpdateInput carries only the fields being changed; nil means "leave as is".
type UpdateInput struct {
	Name             *string
	StartDate        **time.Time
	EndDate          **time.Time
	BudgetTotalPaise **int64
	Status           *Status
}

func (t *Trip) Apply(in UpdateInput) error {
	newName := t.Name
	if in.Name != nil {
		newName = strings.TrimSpace(*in.Name)
		if len(newName) == 0 || len(newName) > 140 {
			return ErrNameRequired
		}
	}
	newStart := t.StartDate
	if in.StartDate != nil {
		newStart = *in.StartDate
	}
	newEnd := t.EndDate
	if in.EndDate != nil {
		newEnd = *in.EndDate
	}
	if newStart != nil && newEnd != nil && newEnd.Before(*newStart) {
		return ErrBadDateRange
	}
	newBudget := t.BudgetTotalPaise
	if in.BudgetTotalPaise != nil {
		newBudget = *in.BudgetTotalPaise
		if newBudget != nil && *newBudget < 0 {
			return ErrNegativeBudget
		}
	}
	newStatus := t.Status
	if in.Status != nil {
		if !in.Status.Valid() {
			return ErrInvalidStatus
		}
		newStatus = *in.Status
	}

	t.Name = newName
	t.StartDate = newStart
	t.EndDate = newEnd
	t.BudgetTotalPaise = newBudget
	t.Status = newStatus
	t.UpdatedAt = time.Now().UTC()
	return nil
}

type Repository interface {
	Create(ctx context.Context, t *Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*Trip, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*Trip, error)
	Update(ctx context.Context, t *Trip) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
