package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"aura/internal/auth"
	"aura/internal/domain"
	"aura/internal/seed"
	"aura/internal/storage"
)

const (
	sessionCookie = "aura_session"
	sessionTTL    = 7 * 24 * time.Hour
)

type ctxKey int

const userKey ctxKey = iota

// sessionResponse is who is signed in (openapi #/components/schemas/Session).
type sessionResponse struct {
	User     domain.Astronaut `json:"user"`
	Username string           `json:"username"`
}

// authed resolves the session cookie into the signed-in member and rejects
// the request with 401 if there is none (FR-71).
func (h *handlers) authed(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acc, ok := h.sessionAccount(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in required")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, acc)))
	}
}

func (h *handlers) sessionAccount(r *http.Request) (storage.Account, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return storage.Account{}, false
	}
	acc, err := h.deps.DB.Q().SessionAccount(r.Context(), auth.HashToken(c.Value), h.deps.Now().UTC())
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			h.deps.Logger.Error("session lookup failed", "err", err)
		}
		return storage.Account{}, false
	}
	return acc, true
}

// currentUser is the signed-in member; only valid inside handlers wrapped by authed.
func currentUser(r *http.Request) domain.Astronaut {
	acc, _ := r.Context().Value(userKey).(storage.Account)
	return acc.Astronaut
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// dummyHash keeps the response time similar for unknown usernames.
var dummyHash, _ = auth.HashPassword("not-a-real-password")

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	if username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	acc, err := h.deps.DB.Q().AccountByUsername(r.Context(), username)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		h.internal(w, "find account", err)
		return
	}
	hash := acc.PasswordHash
	if err != nil || hash == "" {
		hash = dummyHash
	}
	match, cerr := auth.CheckPassword(hash, req.Password)
	if err != nil || acc.PasswordHash == "" || cerr != nil || !match {
		writeError(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	if err := h.startSession(w, r, acc.Astronaut.ID); err != nil {
		h.internal(w, "create session", err)
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{User: acc.Astronaut, Username: acc.Username})
}

// startSession issues a session token in an HTTP-only cookie.
func (h *handlers) startSession(w http.ResponseWriter, r *http.Request, astronautID int64) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	if err := h.deps.DB.Q().CreateSession(r.Context(), auth.HashToken(token), astronautID, h.deps.Now().Add(sessionTTL)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", MaxAge: int(sessionTTL / time.Second),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
	})
	return nil
}

// isHTTPS is true behind Render's TLS-terminating proxy as well.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (h *handlers) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := h.deps.DB.Q().DeleteSession(r.Context(), auth.HashToken(c.Value)); err != nil {
			h.internal(w, "delete session", err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r)})
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) me(w http.ResponseWriter, r *http.Request) {
	acc, _ := r.Context().Value(userKey).(storage.Account)
	writeJSON(w, http.StatusOK, sessionResponse{User: acc.Astronaut, Username: acc.Username})
}

type demoAccount struct {
	Username string      `json:"username"`
	Name     string      `json:"name"`
	Role     domain.Role `json:"role"`
}

// demoAccounts lists the demo logins for the sign-in screen (FR-71). Only
// available when the server runs the demo mission.
func (h *handlers) demoAccounts(w http.ResponseWriter, r *http.Request) {
	if !h.deps.Demo {
		writeError(w, http.StatusNotFound, "demo accounts are disabled")
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	accs, err := h.deps.DB.Q().ListAccounts(r.Context(), m.ID)
	if err != nil {
		h.internal(w, "list accounts", err)
		return
	}
	out := make([]demoAccount, 0, len(accs))
	for _, a := range accs {
		out = append(out, demoAccount{Username: a.Username, Name: a.Astronaut.Name, Role: a.Astronaut.Role})
	}
	writeJSON(w, http.StatusOK, map[string]any{"password": seed.DemoPassword, "accounts": out})
}

// resetDemo replaces all data with a fresh demo mission (FR-62) and signs the
// caller back in to the same account, since IDs and sessions are recreated.
func (h *handlers) resetDemo(w http.ResponseWriter, r *http.Request) {
	if !h.deps.Demo {
		writeError(w, http.StatusNotFound, "demo reset is disabled")
		return
	}
	acc, _ := r.Context().Value(userKey).(storage.Account)
	err := h.deps.DB.Tx(r.Context(), func(q storage.Queries) error {
		if err := q.DeleteAll(r.Context()); err != nil {
			return err
		}
		return seed.Fill(r.Context(), q, h.deps.Catalog, h.deps.Now())
	})
	if err != nil {
		h.internal(w, "reset demo", err)
		return
	}
	h.deps.Logger.Info("demo reset", "by", acc.Username)

	fresh, err := h.deps.DB.Q().AccountByUsername(r.Context(), acc.Username)
	if err != nil {
		h.internal(w, "find account after reset", err)
		return
	}
	if err := h.startSession(w, r, fresh.Astronaut.ID); err != nil {
		h.internal(w, "create session", err)
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{User: fresh.Astronaut, Username: fresh.Username})
}
