package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"aura/internal/domain"
	"aura/internal/seed"
)

type sessionBody struct {
	User     domain.Astronaut `json:"user"`
	Username string           `json:"username"`
}

func TestLogin(t *testing.T) {
	e := newTestEnv(t, true, "")

	e.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusUnauthorized, nil)
	e.do(http.MethodGet, "/api/v1/astronauts", nil, http.StatusUnauthorized, nil)
	// Public without a session: health, catalog, mission clock, demo accounts.
	e.do(http.MethodGet, "/api/v1/missions/current", nil, http.StatusOK, nil)

	bad := []struct {
		name string
		body any
		want int
	}{
		{"wrong password", map[string]string{"username": "rahimova", "password": "nope"}, http.StatusUnauthorized},
		{"unknown user", map[string]string{"username": "armstrong", "password": seed.DemoPassword}, http.StatusUnauthorized},
		{"empty", map[string]string{"username": "", "password": ""}, http.StatusBadRequest},
		{"unknown field", map[string]any{"username": "rahimova", "password": "x", "role": "commander"}, http.StatusBadRequest},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			e.do(http.MethodPost, "/api/v1/auth/login", tt.body, tt.want, nil)
		})
	}

	// Username is case-insensitive and trimmed; the cookie is HTTP-only.
	rr := e.raw(http.MethodPost, "/api/v1/auth/login", map[string]string{"username": " Rahimova ", "password": seed.DemoPassword})
	if rr.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rr.Code, rr.Body.String())
	}
	c := sessionFrom(t, rr)
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Errorf("cookie = %+v", c)
	}
	e.cookie = c
	var me sessionBody
	e.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusOK, &me)
	if me.User.Name != "Amina Rahimova" || me.Username != "rahimova" || me.User.Role != domain.RoleCrew {
		t.Errorf("me = %+v", me)
	}

	e.do(http.MethodPost, "/api/v1/auth/logout", nil, http.StatusNoContent, nil)
	e.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusUnauthorized, nil)

	var demo struct {
		Password string `json:"password"`
		Accounts []struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"accounts"`
	}
	e.do(http.MethodGet, "/api/v1/auth/demo-accounts", nil, http.StatusOK, &demo)
	if demo.Password != seed.DemoPassword || len(demo.Accounts) != 5 || demo.Accounts[0].Username != "chen" {
		t.Errorf("demo accounts = %+v", demo)
	}
}

func TestRoleAccess(t *testing.T) {
	e := newTestEnv(t, true, "")
	amina := findAstronaut(t, e, "Amina Rahimova")
	lucas := findAstronaut(t, e, "Lucas Ferreira")
	readiness := func(a domain.Astronaut) string { return fmt.Sprintf("/api/v1/astronauts/%d/readiness", a.ID) }
	logPath := fmt.Sprintf("/api/v1/astronauts/%d/measurements", amina.ID)
	reading := map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 70}}}

	tests := []struct {
		user         string
		wantCrew     int // members in GET /astronauts
		seeLucas     int
		logForAmina  int
		overview     int
		reportStatus int // Lucas's report
	}{
		{"rahimova", 1, http.StatusForbidden, http.StatusCreated, http.StatusForbidden, http.StatusForbidden},
		{"ferreira", 1, http.StatusOK, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{"morgan", 5, http.StatusOK, http.StatusForbidden, http.StatusOK, http.StatusOK},
		{"sato", 5, http.StatusOK, http.StatusForbidden, http.StatusOK, http.StatusOK},
		{"chen", 5, http.StatusOK, http.StatusForbidden, http.StatusOK, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.user, func(t *testing.T) {
			u := e.login(tt.user)
			var crew astronautsBody
			u.do(http.MethodGet, "/api/v1/astronauts", nil, http.StatusOK, &crew)
			if len(crew.Astronauts) != tt.wantCrew {
				t.Errorf("sees %d members, want %d", len(crew.Astronauts), tt.wantCrew)
			}
			u.do(http.MethodGet, readiness(lucas), nil, tt.seeLucas, nil)
			u.do(http.MethodPost, logPath, reading, tt.logForAmina, nil)
			u.do(http.MethodGet, "/api/v1/crew/overview", nil, tt.overview, nil)
			u.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/report", lucas.ID), nil, tt.reportStatus, nil)
		})
	}
}

func TestCrewOverviewAndReport(t *testing.T) {
	e := newTestEnv(t, true, "")
	lucas := findAstronaut(t, e, "Lucas Ferreira")
	surgeon := e.login("chen")

	var ov struct {
		MissionDay int `json:"missionDay"`
		Members    []struct {
			Astronaut domain.Astronaut `json:"astronaut"`
			Warnings  int              `json:"warnings"`
			Escalate  bool             `json:"escalate"`
			Trends    []struct {
				Indicator string `json:"indicator"`
				Points    []any  `json:"points"`
			} `json:"trends"`
		} `json:"members"`
	}
	surgeon.do(http.MethodGet, "/api/v1/crew/overview", nil, http.StatusOK, &ov)
	if ov.MissionDay != 210 || len(ov.Members) != 4 {
		t.Fatalf("overview: day %d, %d members (the surgeon is not a crew member)", ov.MissionDay, len(ov.Members))
	}
	for _, m := range ov.Members {
		if len(m.Trends) != 10 {
			t.Errorf("%s: %d trend series, want 10", m.Astronaut.Name, len(m.Trends))
		}
		if m.Astronaut.Name == "Lucas Ferreira" && (!m.Escalate || m.Warnings == 0) {
			t.Errorf("Lucas should escalate with a warning: %+v", m)
		}
		if m.Astronaut.Name == "Alex Morgan" && (m.Escalate || m.Warnings != 0) {
			t.Errorf("Alex is nominal: %+v", m)
		}
	}

	var rep struct {
		Escalate bool `json:"escalate"`
		Alerts   []struct {
			Indicator string `json:"indicator"`
			Direction string `json:"direction"`
		} `json:"alerts"`
		Trends []any `json:"trends"`
	}
	path := fmt.Sprintf("/api/v1/astronauts/%d/report", lucas.ID)
	surgeon.do(http.MethodGet, path, nil, http.StatusOK, &rep)
	if !rep.Escalate || len(rep.Trends) != 10 || len(rep.Alerts) == 0 || rep.Alerts[0].Indicator != "spo2" || rep.Alerts[0].Direction != "below" {
		t.Errorf("report = %+v", rep)
	}

	rr := surgeon.raw(http.MethodGet, path+"?format=text", nil)
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("text report: %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}
	if cd := rr.Header().Get("Content-Disposition"); cd != `attachment; filename="aura-report-ferreira-md210.txt"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if !strings.Contains(rr.Body.String(), "Crew member: Lucas Ferreira (Crew)") {
		t.Errorf("text report:\n%s", rr.Body.String())
	}
	surgeon.do(http.MethodGet, path+"?format=pdf", nil, http.StatusBadRequest, nil)
}

func TestDemoReset(t *testing.T) {
	e := newTestEnv(t, true, "")
	amina := e.login("rahimova")
	var me sessionBody
	amina.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusOK, &me)
	base := fmt.Sprintf("/api/v1/astronauts/%d", me.User.ID)
	amina.do(http.MethodPost, base+"/measurements", map[string]any{"measurements": []any{
		map[string]any{"indicator": "bp_systolic", "value": 165}, map[string]any{"indicator": "bp_diastolic", "value": 100},
	}}, http.StatusCreated, nil)
	var r readinessBody
	amina.do(http.MethodGet, base+"/readiness", nil, http.StatusOK, &r)
	if *r.Score > 49 {
		t.Fatalf("score before reset = %d", *r.Score)
	}

	rr := amina.raw(http.MethodPost, "/api/v1/demo/reset", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", rr.Code, rr.Body.String())
	}
	// The old session is gone with the old data; the response signs the caller in again.
	amina.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusUnauthorized, nil)
	amina.cookie = sessionFrom(t, rr)
	amina.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusOK, &me)
	if me.Username != "rahimova" {
		t.Fatalf("after reset signed in as %+v", me)
	}
	amina.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/readiness", me.User.ID), nil, http.StatusOK, &r)
	if r.Score == nil || *r.Score < 50 {
		t.Errorf("score after reset = %v, want the demo state back", r.Score)
	}
	e.do(http.MethodPost, "/api/v1/demo/reset", nil, http.StatusUnauthorized, nil)
}
