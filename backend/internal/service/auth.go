package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tripos/backend/internal/domain/session"
	"github.com/tripos/backend/internal/domain/user"
	"github.com/tripos/backend/internal/platform/jwt"
	"github.com/tripos/backend/internal/platform/passwordhash"
)

var ErrInvalidCredentials = errors.New("auth: invalid credentials")

type AuthService struct {
	users      user.Repository
	sessions   session.Repository
	issuer     *jwt.Issuer
	refreshTTL time.Duration
	hashParams passwordhash.Params
}

func NewAuthService(users user.Repository, sessions session.Repository, issuer *jwt.Issuer, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		users:      users,
		sessions:   sessions,
		issuer:     issuer,
		refreshTTL: refreshTTL,
		hashParams: passwordhash.DefaultParams(),
	}
}

// AuthResult carries the newly-issued tokens; the handler is responsible
// for placing them into httpOnly cookies — they are never returned in the
// JSON body.
type AuthResult struct {
	User               *user.User
	AccessToken        string
	AccessTokenExpiry  time.Time
	RefreshToken       string
	RefreshTokenExpiry time.Time
}

func (s *AuthService) SignUp(ctx context.Context, in user.NewUserInput) (*user.User, error) {
	email, displayName, err := in.Normalize()
	if err != nil {
		return nil, err
	}
	if existing, _ := s.users.GetByEmail(ctx, email); existing != nil {
		return nil, user.ErrEmailTaken
	}
	hash, err := passwordhash.Hash(in.Password, s.hashParams)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}
	now := time.Now().UTC()
	u := &user.User{
		ID:           uuid.New(),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return nil, ErrInvalidCredentials
	}
	ok, err := passwordhash.Verify(password, u.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u)
}

func (s *AuthService) issueTokens(ctx context.Context, u *user.User) (*AuthResult, error) {
	access, accessExp, err := s.issuer.IssueAccessToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: issue access token: %w", err)
	}
	rawRefresh, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("auth: issue refresh token: %w", err)
	}
	refreshExp := time.Now().UTC().Add(s.refreshTTL)
	rt := &session.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: refreshHash,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: refreshExp,
	}
	if err := s.sessions.Create(ctx, rt); err != nil {
		return nil, err
	}
	return &AuthResult{
		User:               u,
		AccessToken:        access,
		AccessTokenExpiry:  accessExp,
		RefreshToken:       rawRefresh,
		RefreshTokenExpiry: refreshExp,
	}, nil
}

// Refresh rotates a refresh token: the presented raw token is looked up by
// hash, checked for validity, revoked, and replaced by a new one — so a
// stolen-and-reused old token is detectable (it will already be revoked).
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*AuthResult, error) {
	hash := hashToken(rawToken)
	existing, err := s.sessions.GetByHash(ctx, hash)
	if err != nil || existing == nil || !existing.Active(time.Now().UTC()) {
		return nil, ErrInvalidCredentials
	}
	u, err := s.users.GetByID(ctx, existing.UserID)
	if err != nil || u == nil {
		return nil, ErrInvalidCredentials
	}
	result, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	// Best-effort revoke of the old token; failure here should not block
	// issuing the new session, but is logged upstream by the caller.
	_ = s.sessions.Revoke(ctx, existing.ID, nil)
	return result, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	hash := hashToken(rawToken)
	existing, err := s.sessions.GetByHash(ctx, hash)
	if err != nil || existing == nil {
		return nil
	}
	return s.sessions.Revoke(ctx, existing.ID, nil)
}

func newRefreshToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	hash = hashToken(raw)
	return raw, hash, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
