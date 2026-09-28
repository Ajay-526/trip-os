package preference

import (
	"testing"

	"github.com/google/uuid"
)

func TestNew_Valid(t *testing.T) {
	start, end := "08:00", "22:00"
	budget := int64(150000)
	p, err := New(UpsertInput{
		TripID: uuid.New(), UserID: uuid.New(), BudgetPaise: &budget,
		Pace: PaceBalanced, Interests: []string{"trekking", "food"},
		EarliestStartTime: &start, LatestEndTime: &end, DateFlexibility: DatePlusMinus2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Pace != PaceBalanced {
		t.Errorf("expected pace balanced, got %s", p.Pace)
	}
}

func TestNew_RejectsNegativeBudget(t *testing.T) {
	budget := int64(-500)
	_, err := New(UpsertInput{TripID: uuid.New(), UserID: uuid.New(), BudgetPaise: &budget})
	if err != ErrNegativeBudget {
		t.Fatalf("expected ErrNegativeBudget, got %v", err)
	}
}

func TestNew_RejectsInvalidPace(t *testing.T) {
	_, err := New(UpsertInput{TripID: uuid.New(), UserID: uuid.New(), Pace: Pace("chaotic")})
	if err != ErrInvalidPace {
		t.Fatalf("expected ErrInvalidPace, got %v", err)
	}
}

func TestNew_RejectsInvalidDateFlexibility(t *testing.T) {
	_, err := New(UpsertInput{TripID: uuid.New(), UserID: uuid.New(), DateFlexibility: DateFlexibility("whenever")})
	if err != ErrInvalidFlex {
		t.Fatalf("expected ErrInvalidFlex, got %v", err)
	}
}

func TestNew_RejectsEndTimeBeforeOrEqualStartTime(t *testing.T) {
	start, end := "09:00", "09:00"
	_, err := New(UpsertInput{TripID: uuid.New(), UserID: uuid.New(), EarliestStartTime: &start, LatestEndTime: &end})
	if err != ErrInvalidTimeSpan {
		t.Fatalf("expected ErrInvalidTimeSpan for equal start/end, got %v", err)
	}

	start2, end2 := "22:00", "08:00"
	_, err = New(UpsertInput{TripID: uuid.New(), UserID: uuid.New(), EarliestStartTime: &start2, LatestEndTime: &end2})
	if err != ErrInvalidTimeSpan {
		t.Fatalf("expected ErrInvalidTimeSpan for end before start, got %v", err)
	}
}

func TestNew_AllowsEmptyOptionalFields(t *testing.T) {
	p, err := New(UpsertInput{TripID: uuid.New(), UserID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error with all-empty optional fields: %v", err)
	}
	if p.Pace != "" || p.DateFlexibility != "" {
		t.Errorf("expected empty pace/flexibility to be preserved as empty, got %q/%q", p.Pace, p.DateFlexibility)
	}
}
