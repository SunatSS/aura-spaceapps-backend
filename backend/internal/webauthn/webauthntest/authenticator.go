// Package webauthntest is a software passkey authenticator for tests: it
// produces real attestation objects and signed assertions (ES256 or RS256).
package webauthntest

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"sort"
)

// COSE algorithms.
const (
	ES256 = -7
	RS256 = -257
)

const (
	flagUP = 0x01
	flagUV = 0x04
	flagAT = 0x40
)

// Authenticator holds one credential.
type Authenticator struct {
	ec      *ecdsa.PrivateKey
	rsa     *rsa.PrivateKey
	CredID  []byte
	Counter uint32
}

// New creates an authenticator with a fresh key.
func New(alg int, credID string) (*Authenticator, error) {
	a := &Authenticator{CredID: []byte(credID)}
	var err error
	if alg == ES256 {
		a.ec, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	} else {
		a.rsa, err = rsa.GenerateKey(rand.Reader, 2048)
	}
	return a, err
}

// Create answers navigator.credentials.create().
func (a *Authenticator) Create(rpID, challenge, origin string) (clientDataJSON, attestationObject []byte) {
	clientDataJSON = ClientData("webauthn.create", challenge, origin)
	attestationObject = Encode(map[any]any{"fmt": "none", "attStmt": map[any]any{}, "authData": a.AuthData(rpID, flagUP|flagUV|flagAT, true)})
	return
}

// Get answers navigator.credentials.get().
func (a *Authenticator) Get(rpID, challenge, origin string) (clientDataJSON, authenticatorData, signature []byte) {
	a.Counter++
	clientDataJSON = ClientData("webauthn.get", challenge, origin)
	authenticatorData = a.AuthData(rpID, flagUP|flagUV, false)
	cdHash := sha256.Sum256(clientDataJSON)
	digest := sha256.Sum256(append(append([]byte(nil), authenticatorData...), cdHash[:]...))
	if a.ec != nil {
		signature, _ = ecdsa.SignASN1(rand.Reader, a.ec, digest[:])
	} else {
		signature, _ = rsa.SignPKCS1v15(rand.Reader, a.rsa, crypto.SHA256, digest[:])
	}
	return
}

// AuthData builds authenticator data; attested adds the credential and its key.
func (a *Authenticator) AuthData(rpID string, flags byte, attested bool) []byte {
	h := sha256.Sum256([]byte(rpID))
	b := append([]byte(nil), h[:]...)
	b = append(b, flags)
	b = binary.BigEndian.AppendUint32(b, a.Counter)
	if attested {
		b = append(b, make([]byte, 16)...) // AAGUID
		b = binary.BigEndian.AppendUint16(b, uint16(len(a.CredID)))
		b = append(b, a.CredID...)
		b = append(b, a.coseKey()...)
	}
	return b
}

func (a *Authenticator) coseKey() []byte {
	if a.ec != nil {
		x, y := make([]byte, 32), make([]byte, 32)
		a.ec.X.FillBytes(x)
		a.ec.Y.FillBytes(y)
		return Encode(map[any]any{1: 2, 3: ES256, -1: 1, -2: x, -3: y})
	}
	return Encode(map[any]any{1: 3, 3: RS256, -1: a.rsa.N.Bytes(), -2: big.NewInt(int64(a.rsa.E)).Bytes()})
}

// ClientData is the JSON the browser signs over.
func ClientData(typ, challenge, origin string) []byte {
	b, _ := json.Marshal(map[string]string{"type": typ, "challenge": challenge, "origin": origin})
	return b
}

func head(major byte, n uint64) []byte {
	switch {
	case n < 24:
		return []byte{major<<5 | byte(n)}
	case n < 1<<8:
		return []byte{major<<5 | 24, byte(n)}
	case n < 1<<16:
		return binary.BigEndian.AppendUint16([]byte{major<<5 | 25}, uint16(n))
	default:
		return binary.BigEndian.AppendUint32([]byte{major<<5 | 26}, uint32(n))
	}
}

// Encode is a tiny CBOR encoder: ints, byte and text strings, maps.
func Encode(v any) []byte {
	switch x := v.(type) {
	case int:
		if x >= 0 {
			return head(0, uint64(x))
		}
		return head(1, uint64(-1-x))
	case []byte:
		return append(head(2, uint64(len(x))), x...)
	case string:
		return append(head(3, uint64(len(x))), x...)
	case map[any]any:
		keys := make([]any, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return string(Encode(keys[i])) < string(Encode(keys[j])) })
		out := head(5, uint64(len(x)))
		for _, k := range keys {
			out = append(out, Encode(k)...)
			out = append(out, Encode(x[k])...)
		}
		return out
	}
	panic("webauthntest: unsupported CBOR type")
}
