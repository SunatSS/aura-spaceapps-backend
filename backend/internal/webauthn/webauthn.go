// Package webauthn verifies passkey registration and sign-in (REQUIREMENTS
// FR-74: Windows Hello, Touch ID, Face ID) with the standard library only.
//
// Scope, chosen for a prototype: attestation "none" (the authenticator's make
// is not verified), algorithms ES256 and RS256, discoverable credentials. The
// checks that make a passkey safe are all here: the challenge is single-use
// and server-generated, the origin and relying-party ID must match, the user
// must be present, and the signature is verified against the stored key.
package webauthn

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// COSE algorithm identifiers.
const (
	AlgES256 = -7
	AlgRS256 = -257
)

// Flags in authenticator data.
const (
	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttested     = 0x40
)

var (
	ErrChallenge = errors.New("webauthn: challenge does not match")
	ErrOrigin    = errors.New("webauthn: origin does not match")
	ErrRPID      = errors.New("webauthn: relying party does not match")
	ErrPresence  = errors.New("webauthn: user was not present")
	ErrSignature = errors.New("webauthn: signature is invalid")
)

var b64 = base64.RawURLEncoding

// Decode accepts base64url with or without padding, as browsers produce.
func Decode(s string) ([]byte, error) {
	return b64.DecodeString(strings.TrimRight(s, "="))
}

// Encode is base64url without padding.
func Encode(b []byte) string { return b64.EncodeToString(b) }

// NewChallenge returns 32 random bytes, base64url-encoded.
func NewChallenge() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("webauthn: challenge: %w", err)
	}
	return Encode(b), nil
}

// Expect is what the server requires of a ceremony.
type Expect struct {
	Challenge string // base64url, as issued
	Origin    string // e.g. https://aura-spaceapps.onrender.com
	RPID      string // e.g. aura-spaceapps.onrender.com
}

// Credential is a registered passkey.
type Credential struct {
	ID        []byte
	PublicKey []byte // COSE_Key as sent by the authenticator
	Alg       int64
	SignCount uint32
}

type clientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

func checkClientData(raw []byte, typ string, e Expect) error {
	var cd clientData
	if err := json.Unmarshal(raw, &cd); err != nil {
		return fmt.Errorf("webauthn: client data: %w", err)
	}
	if cd.Type != typ {
		return fmt.Errorf("webauthn: client data type %q, want %q", cd.Type, typ)
	}
	got, err1 := Decode(cd.Challenge)
	want, err2 := Decode(e.Challenge)
	if err1 != nil || err2 != nil || len(want) == 0 || !bytes.Equal(got, want) {
		return ErrChallenge
	}
	if cd.Origin != e.Origin {
		return ErrOrigin
	}
	return nil
}

type authData struct {
	rpIDHash  []byte
	flags     byte
	signCount uint32
	credID    []byte
	coseKey   []byte
}

func parseAuthData(b []byte) (authData, error) {
	if len(b) < 37 {
		return authData{}, errors.New("webauthn: authenticator data too short")
	}
	ad := authData{rpIDHash: b[:32], flags: b[32], signCount: binary.BigEndian.Uint32(b[33:37])}
	if ad.flags&flagAttested == 0 {
		return ad, nil
	}
	rest := b[37:]
	if len(rest) < 18 {
		return ad, errors.New("webauthn: attested credential data too short")
	}
	n := int(binary.BigEndian.Uint16(rest[16:18]))
	if len(rest) < 18+n {
		return ad, errors.New("webauthn: credential ID truncated")
	}
	ad.credID = rest[18 : 18+n]
	_, used, err := decodeCBOR(rest[18+n:])
	if err != nil {
		return ad, fmt.Errorf("webauthn: credential public key: %w", err)
	}
	ad.coseKey = rest[18+n : 18+n+used]
	return ad, nil
}

func checkAuthData(ad authData, e Expect) error {
	want := sha256.Sum256([]byte(e.RPID))
	if !bytes.Equal(ad.rpIDHash, want[:]) {
		return ErrRPID
	}
	if ad.flags&flagUserPresent == 0 {
		return ErrPresence
	}
	return nil
}

// VerifyRegistration checks navigator.credentials.create() output and returns
// the new credential.
func VerifyRegistration(clientDataJSON, attestationObject []byte, e Expect) (Credential, error) {
	if err := checkClientData(clientDataJSON, "webauthn.create", e); err != nil {
		return Credential{}, err
	}
	v, _, err := decodeCBOR(attestationObject)
	if err != nil {
		return Credential{}, fmt.Errorf("webauthn: attestation object: %w", err)
	}
	m, _ := v.(map[any]any)
	raw, ok := m["authData"].([]byte)
	if !ok {
		return Credential{}, errors.New("webauthn: attestation object has no authData")
	}
	ad, err := parseAuthData(raw)
	if err != nil {
		return Credential{}, err
	}
	if err := checkAuthData(ad, e); err != nil {
		return Credential{}, err
	}
	if ad.credID == nil {
		return Credential{}, errors.New("webauthn: no attested credential")
	}
	alg, err := keyAlg(ad.coseKey)
	if err != nil {
		return Credential{}, err
	}
	return Credential{ID: append([]byte(nil), ad.credID...), PublicKey: append([]byte(nil), ad.coseKey...), Alg: alg, SignCount: ad.signCount}, nil
}

// VerifyAssertion checks navigator.credentials.get() output against a stored
// credential and returns the authenticator's new signature counter.
func VerifyAssertion(c Credential, clientDataJSON, authenticatorData, signature []byte, e Expect) (uint32, error) {
	if err := checkClientData(clientDataJSON, "webauthn.get", e); err != nil {
		return 0, err
	}
	ad, err := parseAuthData(authenticatorData)
	if err != nil {
		return 0, err
	}
	if err := checkAuthData(ad, e); err != nil {
		return 0, err
	}
	// A counter that does not increase suggests a cloned authenticator.
	// Synced passkeys always report 0, which is allowed.
	if ad.signCount != 0 && ad.signCount <= c.SignCount {
		return 0, errors.New("webauthn: signature counter did not increase")
	}
	cdHash := sha256.Sum256(clientDataJSON)
	signed := sha256.Sum256(append(append([]byte(nil), authenticatorData...), cdHash[:]...))
	pub, err := publicKey(c.PublicKey)
	if err != nil {
		return 0, err
	}
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(k, signed[:], signature) {
			return 0, ErrSignature
		}
	case *rsa.PublicKey:
		if rsa.VerifyPKCS1v15(k, crypto.SHA256, signed[:], signature) != nil {
			return 0, ErrSignature
		}
	}
	return ad.signCount, nil
}

// UserVerified reports whether the assertion's authenticator data says the
// user was verified (PIN or biometrics), for display.
func UserVerified(authenticatorData []byte) bool {
	return len(authenticatorData) > 32 && authenticatorData[32]&flagUserVerified != 0
}

func coseMap(b []byte) (map[any]any, error) {
	v, _, err := decodeCBOR(b)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[any]any)
	if !ok {
		return nil, errors.New("webauthn: COSE key is not a map")
	}
	return m, nil
}

func keyAlg(b []byte) (int64, error) {
	m, err := coseMap(b)
	if err != nil {
		return 0, err
	}
	alg, _ := m[int64(3)].(int64)
	if alg != AlgES256 && alg != AlgRS256 {
		return 0, fmt.Errorf("webauthn: unsupported algorithm %d (need ES256 or RS256)", alg)
	}
	if _, err := publicKey(b); err != nil {
		return 0, err
	}
	return alg, nil
}

// publicKey turns a COSE_Key into a Go public key.
func publicKey(b []byte) (crypto.PublicKey, error) {
	m, err := coseMap(b)
	if err != nil {
		return nil, err
	}
	kty, _ := m[int64(1)].(int64)
	alg, _ := m[int64(3)].(int64)
	switch {
	case kty == 2 && alg == AlgES256:
		crv, _ := m[int64(-1)].(int64)
		x, _ := m[int64(-2)].([]byte)
		y, _ := m[int64(-3)].([]byte)
		if crv != 1 || len(x) != 32 || len(y) != 32 {
			return nil, errors.New("webauthn: bad P-256 key")
		}
		k := &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}
		if !k.Curve.IsOnCurve(k.X, k.Y) {
			return nil, errors.New("webauthn: P-256 point is not on the curve")
		}
		return k, nil
	case kty == 3 && alg == AlgRS256:
		n, _ := m[int64(-1)].([]byte)
		eb, _ := m[int64(-2)].([]byte)
		if len(n) < 256 || len(eb) == 0 || len(eb) > 4 {
			return nil, errors.New("webauthn: bad RSA key")
		}
		e := 0
		for _, c := range eb {
			e = e<<8 | int(c)
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: e}, nil
	}
	return nil, fmt.Errorf("webauthn: unsupported key type %d / algorithm %d", kty, alg)
}
