package webauthn

import (
	"errors"
	"testing"

	"aura/internal/webauthn/webauthntest"
)

type authenticator = webauthntest.Authenticator

func newAuthenticator(t *testing.T, alg int) *authenticator {
	t.Helper()
	a, err := webauthntest.New(alg, "credential-0001")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

var (
	enc        = webauthntest.Encode
	clientJSON = webauthntest.ClientData
)

// --- tests -----------------------------------------------------------------------

const (
	rpID   = "aura-spaceapps.onrender.com"
	origin = "https://aura-spaceapps.onrender.com"
)

func TestRegisterAndSignIn(t *testing.T) {
	for _, alg := range []int{AlgES256, AlgRS256} {
		t.Run(map[int]string{AlgES256: "ES256", AlgRS256: "RS256"}[alg], func(t *testing.T) {
			auth := newAuthenticator(t, alg)
			ch, _ := NewChallenge()
			e := Expect{Challenge: ch, Origin: origin, RPID: rpID}

			cdj, att := auth.Create(rpID, ch, origin)
			cred, err := VerifyRegistration(cdj, att, e)
			if err != nil {
				t.Fatalf("registration: %v", err)
			}
			if string(cred.ID) != "credential-0001" || cred.Alg != int64(alg) {
				t.Fatalf("credential = %+v", cred)
			}

			ch2, _ := NewChallenge()
			e2 := Expect{Challenge: ch2, Origin: origin, RPID: rpID}
			cdj, ad, sig := auth.Get(rpID, ch2, origin)
			count, err := VerifyAssertion(cred, cdj, ad, sig, e2)
			if err != nil || count != 1 || !UserVerified(ad) {
				t.Fatalf("assertion: count %d, err %v", count, err)
			}
			cred.SignCount = count

			// Replaying the same assertion fails: the counter did not increase.
			if _, err := VerifyAssertion(cred, cdj, ad, sig, e2); err == nil {
				t.Error("replayed assertion accepted")
			}
		})
	}
}

func TestRejects(t *testing.T) {
	auth := newAuthenticator(t, AlgES256)
	ch, _ := NewChallenge()
	e := Expect{Challenge: ch, Origin: origin, RPID: rpID}
	cdj, att := auth.Create(rpID, ch, origin)
	cred, err := VerifyRegistration(cdj, att, e)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := NewChallenge()

	t.Run("registration", func(t *testing.T) {
		tests := []struct {
			name    string
			cdj     []byte
			att     []byte
			wantErr error
		}{
			{"other challenge", clientJSON("webauthn.create", other, origin), att, ErrChallenge},
			{"phishing origin", clientJSON("webauthn.create", ch, "https://aura-spaceapps.example"), att, ErrOrigin},
			{"wrong ceremony", clientJSON("webauthn.get", ch, origin), att, nil},
			{"other relying party", cdj, enc(map[any]any{"fmt": "none", "attStmt": map[any]any{}, "authData": auth.AuthData("evil.example", flagUserPresent|flagAttested, true)}), ErrRPID},
			{"user not present", cdj, enc(map[any]any{"fmt": "none", "attStmt": map[any]any{}, "authData": auth.AuthData(rpID, flagAttested, true)}), ErrPresence},
			{"garbage", cdj, []byte{0xff, 0x00}, nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := VerifyRegistration(tt.cdj, tt.att, e)
				if err == nil || (tt.wantErr != nil && !errors.Is(err, tt.wantErr)) {
					t.Errorf("err = %v, want %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("assertion", func(t *testing.T) {
		cdj, ad, sig := auth.Get(rpID, ch, origin)
		tampered := append([]byte(nil), sig...)
		tampered[len(tampered)-1] ^= 0xff
		impostor := newAuthenticator(t, AlgES256)
		impostor.Counter = auth.Counter - 1
		icdj, iad, isig := impostor.Get(rpID, ch, origin)

		tests := []struct {
			name         string
			cdj, ad, sig []byte
			expect       Expect
		}{
			{"tampered signature", cdj, ad, tampered, e},
			{"signed by another key", icdj, iad, isig, e},
			{"other challenge", cdj, ad, sig, Expect{Challenge: other, Origin: origin, RPID: rpID}},
			{"other origin", cdj, ad, sig, Expect{Challenge: ch, Origin: "http://localhost:5173", RPID: rpID}},
			{"other relying party", cdj, ad, sig, Expect{Challenge: ch, Origin: origin, RPID: "localhost"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if _, err := VerifyAssertion(cred, tt.cdj, tt.ad, tt.sig, tt.expect); err == nil {
					t.Error("accepted")
				}
			})
		}
	})
}

func TestDecodeBase64URL(t *testing.T) {
	for _, s := range []string{"aGVsbG8", "aGVsbG8="} {
		if b, err := Decode(s); err != nil || string(b) != "hello" {
			t.Errorf("Decode(%q) = %q, %v", s, b, err)
		}
	}
}
