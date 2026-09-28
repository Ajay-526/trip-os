// Package passwordhash implements Argon2id password hashing with a
// self-describing encoded format ("$argon2id$v=19$m=...,t=...,p=...$salt$hash")
// so parameters can change over time without breaking verification of
// previously-issued hashes.
package passwordhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidHash = errors.New("passwordhash: invalid encoded hash")

type Params struct {
	MemoryKB    uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParams() Params {
	return Params{
		MemoryKB:    64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func Hash(password string, p Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("passwordhash: read salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKB, p.Parallelism, p.KeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.MemoryKB, p.Iterations, p.Parallelism, b64Salt, b64Hash)
	return encoded, nil
}

func Verify(password, encodedHash string) (bool, error) {
	p, salt, hash, err := decode(encodedHash)
	if err != nil {
		return false, err
	}
	candidate := argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKB, p.Parallelism, p.KeyLength)
	if subtle.ConstantTimeCompare(candidate, hash) == 1 {
		return true, nil
	}
	return false, nil
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKB, &p.Iterations, &p.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(hash))
	return p, salt, hash, nil
}
