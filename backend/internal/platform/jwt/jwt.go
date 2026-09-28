package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("jwt: invalid or expired token")

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	jwt.RegisteredClaims
}

// IssueAccessToken creates a short-lived access token bound to a user ID.
// This token is carried in an httpOnly cookie, never in the JS-accessible
// response body or localStorage.
func (i *Issuer) IssueAccessToken(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(i.ttl)
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString(i.secret)
	return signed, expiresAt, err
}

// ParseAccessToken validates signature and expiry and returns the user ID.
func (i *Issuer) ParseAccessToken(tokenStr string) (uuid.UUID, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return i.secret, nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	c, ok := parsed.Claims.(*claims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return id, nil
}
