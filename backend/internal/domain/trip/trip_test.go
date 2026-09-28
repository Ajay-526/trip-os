package trip

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func TestNew_ValidTrip(t *testing.T) {
	owner := uuid.New()
	start := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 24, 0, 0, 0, 0, time.UTC)

	tr, err := New(CreateInput{
		Name: "Manali Squad Trip", DestinationQuery: "Manali",
		StartDate: &start, EndDate: &end, BudgetTotalPaise: ptr(int64(300000)),
		Currency: "inr", CreatedBy: owner,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Status != StatusDraft {
		t.Errorf("expected draft status, got %s", tr.Status)
	}
	if tr.Currency != "INR" {
		t.Errorf("expected currency normalized to INR, got %s", tr.Currency)
	}
}

func TestNew_RejectsEmptyName(t *testing.T) {
	_, err := New(CreateInput{Name: "  ", DestinationQuery: "Goa", CreatedBy: uuid.New()})
	if err != ErrNameRequired {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}
}

func TestNew_RejectsEmptyDestination(t *testing.T) {
	_, err := New(CreateInput{Name: "Trip", DestinationQuery: "", CreatedBy: uuid.New()})
	if err != ErrDestRequired {
		t.Fatalf("expected ErrDestRequired, got %v", err)
	}
}

func TestNew_RejectsBadDateRange(t *testing.T) {
	start := time.Date(2026, 12, 24, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
	_, err := New(CreateInput{
		Name: "Trip", DestinationQuery: "Goa", StartDate: &start, EndDate: &end, CreatedBy: uuid.New(),
	})
	if err != ErrBadDateRange {
		t.Fatalf("expected ErrBadDateRange, got %v", err)
	}
}

func TestNew_RejectsNegativeBudget(t *testing.T) {
	_, err := New(CreateInput{
		Name: "Trip", DestinationQuery: "Goa", BudgetTotalPaise: ptr(int64(-1)), CreatedBy: uuid.New(),
	})
	if err != ErrNegativeBudget {
		t.Fatalf("expected ErrNegativeBudget, got %v", err)
	}
}

func TestNew_DefaultsCurrencyToINR(t *testing.T) {
	tr, err := New(CreateInput{Name: "Trip", DestinationQuery: "Goa", CreatedBy: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Currency != "INR" {
		t.Errorf("expected default currency INR, got %s", tr.Currency)
	}
}

func TestApply_UpdatesFieldsAndValidates(t *testing.T) {
	tr, _ := New(CreateInput{Name: "Trip", DestinationQuery: "Goa", CreatedBy: uuid.New()})

	newName := "Renamed Trip"
	err := tr.Apply(UpdateInput{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name != newName {
		t.Errorf("expected name to update, got %s", tr.Name)
	}
}

func TestApply_RejectsBadDateRangeOnUpdate(t *testing.T) {
	tr, _ := New(CreateInput{Name: "Trip", DestinationQuery: "Goa", CreatedBy: uuid.New()})

	start := time.Date(2026, 12, 24, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
	startPtr := &start
	endPtr := &end

	err := tr.Apply(UpdateInput{StartDate: &startPtr, EndDate: &endPtr})
	if err != ErrBadDateRange {
		t.Fatalf("expected ErrBadDateRange, got %v", err)
	}
	// Original trip must remain unchanged on rejection... actually Apply
	// mutates fields before the final check only for name; date/budget are
	// computed into locals first. Verify no partial mutation occurred.
	if tr.StartDate != nil {
		t.Errorf("expected StartDate to remain nil after rejected update, got %v", tr.StartDate)
	}
}

func TestApply_RejectsInvalidStatus(t *testing.T) {
	tr, _ := New(CreateInput{Name: "Trip", DestinationQuery: "Goa", CreatedBy: uuid.New()})
	bad := Status("not_a_real_status")
	err := tr.Apply(UpdateInput{Status: &bad})
	if err != ErrInvalidStatus {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestStatus_Valid(t *testing.T) {
	valid := []Status{StatusDraft, StatusPlanning, StatusVoting, StatusBooked, StatusInProgress, StatusCompleted, StatusCancelled}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %s to be valid", s)
		}
	}
	if Status("bogus").Valid() {
		t.Errorf("expected bogus status to be invalid")
	}
}
