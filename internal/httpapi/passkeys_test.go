package httpapi

import (
	"net/http"
	"testing"

	"aura/internal/webauthn"
	"aura/internal/webauthn/webauthntest"
)

// httptest requests go to http://example.com, so that is the relying party.
const (
	testRPID   = "example.com"
	testOrigin = "http://example.com"
)

func TestPasskeys(t *testing.T) {
	e := newTestEnv(t, true, "")
	amina := e.login("rahimova")
	auth, err := webauthntest.New(webauthntest.ES256, "amina-laptop")
	if err != nil {
		t.Fatal(err)
	}

	// Register.
	var opts struct {
		Challenge string `json:"challenge"`
		RP        struct{ ID string }
		User      struct{ Name, DisplayName string }
	}
	amina.do(http.MethodPost, "/api/v1/auth/passkeys/register/begin", nil, http.StatusOK, &opts)
	if opts.RP.ID != testRPID || opts.User.Name != "rahimova" || opts.User.DisplayName != "Amina Rahimova" {
		t.Fatalf("options = %+v", opts)
	}
	cdj, att := auth.Create(testRPID, opts.Challenge, testOrigin)
	finish := map[string]string{"challenge": opts.Challenge, "clientDataJSON": webauthn.Encode(cdj), "attestationObject": webauthn.Encode(att), "label": "Windows Hello"}
	amina.do(http.MethodPost, "/api/v1/auth/passkeys/register/finish", finish, http.StatusCreated, nil)
	// The challenge is single-use.
	amina.do(http.MethodPost, "/api/v1/auth/passkeys/register/finish", finish, http.StatusBadRequest, nil)

	var list struct {
		Passkeys []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"passkeys"`
	}
	amina.do(http.MethodGet, "/api/v1/auth/passkeys", nil, http.StatusOK, &list)
	if len(list.Passkeys) != 1 || list.Passkeys[0].Label != "Windows Hello" || list.Passkeys[0].ID != webauthn.Encode([]byte("amina-laptop")) {
		t.Fatalf("passkeys = %+v", list)
	}

	// Sign in without a password.
	signIn := func(origin string) (map[string]string, int) {
		var begin struct{ Challenge, RpID string }
		e.do(http.MethodPost, "/api/v1/auth/passkeys/login/begin", nil, http.StatusOK, &begin)
		if begin.RpID != testRPID {
			t.Fatalf("rpId = %q", begin.RpID)
		}
		cdj, ad, sig := auth.Get(testRPID, begin.Challenge, origin)
		body := map[string]string{"challenge": begin.Challenge, "id": list.Passkeys[0].ID,
			"clientDataJSON": webauthn.Encode(cdj), "authenticatorData": webauthn.Encode(ad), "signature": webauthn.Encode(sig)}
		return body, e.raw(http.MethodPost, "/api/v1/auth/passkeys/login/finish", body).Code
	}
	body, code := signIn(testOrigin)
	if code != http.StatusOK {
		t.Fatalf("passkey sign-in: %d", code)
	}
	rr := e.raw(http.MethodPost, "/api/v1/auth/passkeys/login/finish", body)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("replayed sign-in: %d, want 400 (challenge already used)", rr.Code)
	}
	if _, code := signIn("https://evil.example"); code != http.StatusUnauthorized {
		t.Errorf("sign-in from another origin: %d, want 401", code)
	}

	// A fresh sign-in yields a working session for Amina.
	var begin struct{ Challenge string }
	e.do(http.MethodPost, "/api/v1/auth/passkeys/login/begin", nil, http.StatusOK, &begin)
	cdj, ad, sig := auth.Get(testRPID, begin.Challenge, testOrigin)
	rr = e.raw(http.MethodPost, "/api/v1/auth/passkeys/login/finish", map[string]string{"challenge": begin.Challenge, "id": list.Passkeys[0].ID,
		"clientDataJSON": webauthn.Encode(cdj), "authenticatorData": webauthn.Encode(ad), "signature": webauthn.Encode(sig)})
	if rr.Code != http.StatusOK {
		t.Fatalf("second sign-in: %d %s", rr.Code, rr.Body.String())
	}
	s := e
	s.cookie = sessionFrom(t, rr)
	var me sessionBody
	s.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusOK, &me)
	if me.Username != "rahimova" {
		t.Errorf("signed in as %+v", me)
	}

	// Someone else cannot delete it; the owner can.
	e.login("ferreira").do(http.MethodDelete, "/api/v1/auth/passkeys/"+list.Passkeys[0].ID, nil, http.StatusNotFound, nil)
	amina.do(http.MethodDelete, "/api/v1/auth/passkeys/"+list.Passkeys[0].ID, nil, http.StatusNoContent, nil)
	e.do(http.MethodPost, "/api/v1/auth/passkeys/register/begin", nil, http.StatusUnauthorized, nil)
}
