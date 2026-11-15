// Package auth holds sign-in primitives and the role-based access policy
// (REQUIREMENTS FR-71, FR-72):
//
//   - passwords are hashed with PBKDF2-HMAC-SHA256 (standard library, no
//     extra dependency), a random salt per user and 600 000 iterations (OWASP);
//   - sessions are random 256-bit tokens; only their SHA-256 is stored;
//   - crew members see only themselves; the commander, the deputy and the
//     flight surgeon see the whole crew; nobody logs readings for someone else.
//
// No I/O besides crypto/rand.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"aura/internal/domain"
)

// Iterations is the PBKDF2 work factor for new hashes. Tests lower it.
var Iterations = 600_000

const (
	scheme  = "pbkdf2-sha256"
	saltLen = 16
	keyLen  = 32
)

var b64 = base64.RawStdEncoding

// HashPassword returns "pbkdf2-sha256$<iterations>$<salt>$<key>".
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, Iterations, keyLen)
	if err != nil {
		return "", fmt.Errorf("auth: derive key: %w", err)
	}
	return fmt.Sprintf("%s$%d$%s$%s", scheme, Iterations, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

var errBadHash = errors.New("auth: malformed password hash")

// CheckPassword reports whether password matches hash (constant-time compare).
func CheckPassword(hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != scheme {
		return false, errBadHash
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 {
		return false, errBadHash
	}
	salt, err1 := b64.DecodeString(parts[2])
	want, err2 := b64.DecodeString(parts[3])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false, errBadHash
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false, fmt.Errorf("auth: derive key: %w", err)
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// NewToken returns a random session token for the cookie.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is what the database stores for a session token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// SeesWholeCrew reports whether a role may view every crew member (FR-72).
func SeesWholeCrew(r domain.Role) bool {
	switch r {
	case domain.RoleCommander, domain.RoleDeputyCommander, domain.RoleFlightSurgeon:
		return true
	}
	return false
}

// CanView reports whether viewer may read target's health data.
func CanView(viewer, target domain.Astronaut) bool {
	return viewer.ID == target.ID || SeesWholeCrew(viewer.Role)
}

// CanLog reports whether viewer may log readings for target: only their own,
// and the flight surgeon (on Earth) has no health data to log.
func CanLog(viewer, target domain.Astronaut) bool {
	return viewer.ID == target.ID && target.Role != domain.RoleFlightSurgeon
}
