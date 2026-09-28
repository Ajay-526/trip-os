package membership

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewOwner(t *testing.T) {
	tripID, userID := uuid.New(), uuid.New()
	m := NewOwner(tripID, userID)

	if m.Role != RoleOwner {
		t.Errorf("expected owner role, got %s", m.Role)
	}
	if m.Status != StatusJoined {
		t.Errorf("expected joined status, got %s", m.Status)
	}
	if m.UserID == nil || *m.UserID != userID {
		t.Errorf("expected user id to be set")
	}
	if m.JoinedAt == nil {
		t.Errorf("expected joined_at to be set for the owner")
	}
}

func TestNewEmailInvite_Valid(t *testing.T) {
	m, err := NewEmailInvite(uuid.New(), "Friend@Example.com", RoleMember, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *m.InvitedEmail != "friend@example.com" {
		t.Errorf("expected email to be lowercased, got %s", *m.InvitedEmail)
	}
	if m.Status != StatusInvited {
		t.Errorf("expected invited status, got %s", m.Status)
	}
}

func TestNewEmailInvite_RejectsInvalidEmail(t *testing.T) {
	_, err := NewEmailInvite(uuid.New(), "not-an-email", RoleMember, uuid.New())
	if err != ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestNewEmailInvite_RejectsOwnerRole(t *testing.T) {
	_, err := NewEmailInvite(uuid.New(), "friend@example.com", RoleOwner, uuid.New())
	if err != ErrInvalidRole {
		t.Fatalf("expected ErrInvalidRole for owner invite, got %v", err)
	}
}

func TestAccept_SetsUserAndStatus(t *testing.T) {
	m, _ := NewEmailInvite(uuid.New(), "friend@example.com", RoleMember, uuid.New())
	userID := uuid.New()
	m.Accept(userID)

	if m.Status != StatusJoined {
		t.Errorf("expected joined status after accept, got %s", m.Status)
	}
	if m.UserID == nil || *m.UserID != userID {
		t.Errorf("expected user id to be set after accept")
	}
	if m.JoinedAt == nil {
		t.Errorf("expected joined_at to be set after accept")
	}
}

func TestRole_CanManageTrip(t *testing.T) {
	cases := map[Role]bool{
		RoleOwner:       true,
		RoleCoOrganizer: true,
		RoleMember:      false,
	}
	for role, want := range cases {
		if got := role.CanManageTrip(); got != want {
			t.Errorf("role %s: expected CanManageTrip=%v, got %v", role, want, got)
		}
	}
}

func TestInviteLink_Usable(t *testing.T) {
	now := time.Now().UTC()

	t.Run("revoked link is not usable", func(t *testing.T) {
		revokedAt := now
		l := &InviteLink{RevokedAt: &revokedAt}
		if err := l.Usable(now); err != ErrInviteExpired {
			t.Fatalf("expected ErrInviteExpired, got %v", err)
		}
	})

	t.Run("expired link is not usable", func(t *testing.T) {
		past := now.Add(-time.Hour)
		l := &InviteLink{ExpiresAt: &past}
		if err := l.Usable(now); err != ErrInviteExpired {
			t.Fatalf("expected ErrInviteExpired, got %v", err)
		}
	})

	t.Run("exhausted link is not usable", func(t *testing.T) {
		max := 1
		l := &InviteLink{MaxUses: &max, UseCount: 1}
		if err := l.Usable(now); err != ErrInviteExhausted {
			t.Fatalf("expected ErrInviteExhausted, got %v", err)
		}
	})

	t.Run("fresh unlimited link is usable", func(t *testing.T) {
		l := &InviteLink{}
		if err := l.Usable(now); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("link with uses remaining is usable", func(t *testing.T) {
		max := 5
		l := &InviteLink{MaxUses: &max, UseCount: 3}
		if err := l.Usable(now); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
