package httpapi

import (
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"aura/internal/storage"
	"aura/internal/webauthn"
)

// Passkeys (FR-74): sign in with Windows Hello, Touch ID or Face ID. The
// relying party is the host the browser talks to, so the same code works on
// Render and on localhost (WebAuthn needs HTTPS or localhost).

const (
	challengeTTL      = 5 * time.Minute
	challengeLogin    = "login"
	challengeRegister = "register"
	rpName            = "AURA"
)

// relyingParty derives the expected origin and RP ID from the request.
func relyingParty(r *http.Request) webauthn.Expect {
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	host := r.Host
	rpID := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		rpID = h
	}
	return webauthn.Expect{Origin: scheme + "://" + host, RPID: strings.ToLower(rpID)}
}

func (h *handlers) issueChallenge(w http.ResponseWriter, r *http.Request, kind string, who *int64) (string, bool) {
	ch, err := webauthn.NewChallenge()
	if err == nil {
		err = h.deps.DB.Q().CreateChallenge(r.Context(), ch, kind, who, h.deps.Now().Add(challengeTTL))
	}
	if err != nil {
		h.internal(w, "issue challenge", err)
		return "", false
	}
	return ch, true
}

type credentialParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type registerOptions struct {
	Challenge string `json:"challenge"`
	RP        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"rp"`
	User struct {
		ID          string `json:"id"` // base64url of the member ID
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	PubKeyCredParams []credentialParam `json:"pubKeyCredParams"`
	ExcludeIDs       []string          `json:"excludeCredentialIds"`
	TimeoutMs        int               `json:"timeout"`
}

// beginPasskeyRegistration returns options for navigator.credentials.create().
func (h *handlers) beginPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	acc, _ := r.Context().Value(userKey).(storage.Account)
	ch, ok := h.issueChallenge(w, r, challengeRegister, &acc.Astronaut.ID)
	if !ok {
		return
	}
	existing, err := h.deps.DB.Q().ListPasskeys(r.Context(), acc.Astronaut.ID)
	if err != nil {
		h.internal(w, "list passkeys", err)
		return
	}
	rp := relyingParty(r)
	var o registerOptions
	o.Challenge, o.TimeoutMs = ch, int(challengeTTL/time.Millisecond)
	o.RP.ID, o.RP.Name = rp.RPID, rpName
	o.User.ID = webauthn.Encode(binary.BigEndian.AppendUint64(nil, uint64(acc.Astronaut.ID)))
	o.User.Name, o.User.DisplayName = acc.Username, acc.Astronaut.Name
	o.PubKeyCredParams = []credentialParam{{"public-key", webauthn.AlgES256}, {"public-key", webauthn.AlgRS256}}
	o.ExcludeIDs = []string{}
	for _, p := range existing {
		o.ExcludeIDs = append(o.ExcludeIDs, p.ID)
	}
	writeJSON(w, http.StatusOK, o)
}

type finishRegistrationRequest struct {
	Challenge         string `json:"challenge"`
	ClientDataJSON    string `json:"clientDataJSON"`
	AttestationObject string `json:"attestationObject"`
	Label             string `json:"label"`
}

func (h *handlers) finishPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	var req finishRegistrationRequest
	if !decodeStrictN(w, r, &req, 64<<10) {
		return
	}
	me := currentUser(r)
	q := h.deps.DB.Q()
	who, err := q.TakeChallenge(r.Context(), req.Challenge, challengeRegister, h.deps.Now().UTC())
	if errors.Is(err, storage.ErrNotFound) || (err == nil && (who == nil || *who != me.ID)) {
		writeError(w, http.StatusBadRequest, "the passkey request expired — try again")
		return
	}
	if err != nil {
		h.internal(w, "take challenge", err)
		return
	}
	cdj, err1 := webauthn.Decode(req.ClientDataJSON)
	att, err2 := webauthn.Decode(req.AttestationObject)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "clientDataJSON and attestationObject must be base64url")
		return
	}
	expect := relyingParty(r)
	expect.Challenge = req.Challenge
	cred, err := webauthn.VerifyRegistration(cdj, att, expect)
	if err != nil {
		writeError(w, http.StatusBadRequest, "passkey not accepted: "+err.Error())
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "Passkey"
	}
	if len(label) > 60 {
		label = label[:60]
	}
	p := storage.Passkey{ID: webauthn.Encode(cred.ID), AstronautID: me.ID, PublicKey: cred.PublicKey, Alg: cred.Alg,
		SignCount: cred.SignCount, Label: label, CreatedAt: h.deps.Now().UTC()}
	if err := q.CreatePasskey(r.Context(), p); err != nil {
		writeError(w, http.StatusConflict, "this passkey is already registered")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// beginPasskeyLogin returns options for navigator.credentials.get() with a
// discoverable credential: the authenticator offers the member's passkeys.
func (h *handlers) beginPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ch, ok := h.issueChallenge(w, r, challengeLogin, nil)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenge": ch, "rpId": relyingParty(r).RPID, "timeout": int(challengeTTL / time.Millisecond)})
}

type finishLoginRequest struct {
	Challenge         string `json:"challenge"`
	ID                string `json:"id"`
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
}

func (h *handlers) finishPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	var req finishLoginRequest
	if !decodeStrictN(w, r, &req, 32<<10) {
		return
	}
	q := h.deps.DB.Q()
	now := h.deps.Now().UTC()
	if _, err := q.TakeChallenge(r.Context(), req.Challenge, challengeLogin, now); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "the sign-in request expired — try again")
			return
		}
		h.internal(w, "take challenge", err)
		return
	}
	fail := func() { writeError(w, http.StatusUnauthorized, "passkey not recognised") }
	p, err := q.GetPasskey(r.Context(), strings.TrimRight(req.ID, "="))
	if errors.Is(err, storage.ErrNotFound) {
		fail()
		return
	}
	if err != nil {
		h.internal(w, "get passkey", err)
		return
	}
	cdj, err1 := webauthn.Decode(req.ClientDataJSON)
	ad, err2 := webauthn.Decode(req.AuthenticatorData)
	sig, err3 := webauthn.Decode(req.Signature)
	if err1 != nil || err2 != nil || err3 != nil {
		writeError(w, http.StatusBadRequest, "fields must be base64url")
		return
	}
	expect := relyingParty(r)
	expect.Challenge = req.Challenge
	count, err := webauthn.VerifyAssertion(webauthn.Credential{PublicKey: p.PublicKey, Alg: p.Alg, SignCount: p.SignCount}, cdj, ad, sig, expect)
	if err != nil {
		h.deps.Logger.Info("passkey sign-in rejected", "err", err)
		fail()
		return
	}
	if err := q.TouchPasskey(r.Context(), p.ID, count, now); err != nil {
		h.internal(w, "touch passkey", err)
		return
	}
	acc, err := q.AccountByID(r.Context(), p.AstronautID)
	if err != nil {
		h.internal(w, "account", err)
		return
	}
	if err := h.startSession(w, r, acc.Astronaut.ID); err != nil {
		h.internal(w, "create session", err)
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{User: acc.Astronaut, Username: acc.Username})
}

func (h *handlers) listPasskeys(w http.ResponseWriter, r *http.Request) {
	ps, err := h.deps.DB.Q().ListPasskeys(r.Context(), currentUser(r).ID)
	if err != nil {
		h.internal(w, "list passkeys", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"passkeys": ps})
}

func (h *handlers) deletePasskey(w http.ResponseWriter, r *http.Request) {
	err := h.deps.DB.Q().DeletePasskey(r.Context(), r.PathValue("id"), currentUser(r).ID)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "passkey not found")
		return
	}
	if err != nil {
		h.internal(w, "delete passkey", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
