package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestProfileTasksAndEnvironment(t *testing.T) {
	e := newTestEnv(t, true, "")
	lucas := findAstronaut(t, e, "Lucas Ferreira")
	alex := findAstronaut(t, e, "Alex Morgan")
	surgeon := e.login("chen")
	base := fmt.Sprintf("/api/v1/astronauts/%d", lucas.ID)

	// Demo profile: athlete's heart range, EVA-lead weight, baselines.
	var p struct {
		Conditions string `json:"conditions"`
		Indicators map[string]struct {
			Baseline   *float64 `json:"baseline"`
			Thresholds *struct {
				Nominal struct{ Min, Max *float64 } `json:"nominal"`
			} `json:"thresholds"`
		} `json:"indicators"`
		Weights map[string]float64 `json:"weights"`
	}
	surgeon.do(http.MethodGet, base+"/profile", nil, http.StatusOK, &p)
	hr := p.Indicators["heart_rate"]
	if !strings.Contains(p.Conditions, "athlete") || hr.Thresholds == nil || *hr.Thresholds.Nominal.Min != 42 || p.Weights["respiratory"] != 1.5 || hr.Baseline == nil {
		t.Fatalf("profile = %+v", p)
	}

	// Only the flight surgeon edits; ranges are validated.
	upd := map[string]any{"conditions": "EVA lead", "indicators": map[string]any{
		"glucose": map[string]any{"baseline": 92, "thresholds": map[string]any{"nominal": map[string]any{"min": 75, "max": 130}, "caution": map[string]any{"min": 60, "max": 170}}, "note": "test"},
	}, "weights": map[string]any{"respiratory": 0}}
	e.login("ferreira").do(http.MethodPut, base+"/profile", upd, http.StatusForbidden, nil)
	surgeon.do(http.MethodPut, base+"/profile", map[string]any{"indicators": map[string]any{
		"glucose": map[string]any{"thresholds": map[string]any{"nominal": map[string]any{"min": 75, "max": 130}, "caution": map[string]any{"min": 80, "max": 120}}},
	}}, http.StatusBadRequest, nil)
	surgeon.do(http.MethodPut, base+"/profile", map[string]any{"weights": map[string]any{"respiratory": 9}}, http.StatusBadRequest, nil)
	p.Indicators, p.Weights = nil, nil
	surgeon.do(http.MethodPut, base+"/profile", upd, http.StatusOK, &p)
	if p.Conditions != "EVA lead" || p.Indicators["glucose"].Thresholds == nil || p.Weights["respiratory"] != 0 {
		t.Errorf("after update = %+v", p)
	}

	// Personal range shows in readiness.
	var rd struct {
		Systems []struct {
			Indicators []struct {
				Code     string `json:"code"`
				Personal bool   `json:"personal"`
			} `json:"indicators"`
		} `json:"systems"`
	}
	surgeon.do(http.MethodGet, base+"/readiness", nil, http.StatusOK, &rd)
	personal := map[string]bool{}
	for _, s := range rd.Systems {
		for _, i := range s.Indicators {
			personal[i.Code] = i.Personal
		}
	}
	if !personal["heart_rate"] || !personal["glucose"] || personal["spo2"] {
		t.Errorf("personal flags = %v", personal)
	}

	// Trend over 30 days of history: Lucas's body mass, nobody else.
	var g guidanceBody
	surgeon.do(http.MethodGet, base+"/guidance", nil, http.StatusOK, &g)
	ids := map[string]bool{}
	for _, r := range g.Recommendations {
		ids[r.ID] = true
	}
	if !ids["trend:body_mass_change"] {
		t.Errorf("Lucas guidance lacks the body-mass trend: %v", ids)
	}
	surgeon.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/guidance", alex.ID), nil, http.StatusOK, &g)
	if len(g.Recommendations) != 0 {
		t.Errorf("Alex should have no recommendations: %+v", g.Recommendations)
	}

	// Tasks: streak from the seeded exercise marks; mark today's exercise.
	me := e.login("ferreira")
	var ts struct {
		Total  int `json:"total"`
		Streak int `json:"streak"`
		Tasks  []struct {
			Code string `json:"code"`
			Done bool   `json:"done"`
		} `json:"tasks"`
		Badges []struct{ Code string } `json:"badges"`
	}
	me.do(http.MethodGet, base+"/tasks", nil, http.StatusOK, &ts)
	if ts.Total != 7 || ts.Streak < 7 || len(ts.Badges) == 0 {
		t.Errorf("tasks = %+v", ts)
	}
	me.do(http.MethodPost, base+"/tasks/exercise/complete", nil, http.StatusOK, &ts)
	for _, tk := range ts.Tasks {
		if tk.Code == "exercise" && !tk.Done {
			t.Error("exercise not marked done")
		}
	}
	me.do(http.MethodPost, base+"/tasks/morning_vitals/complete", nil, http.StatusBadRequest, nil)
	surgeon.do(http.MethodPost, base+"/tasks/exercise/complete", nil, http.StatusForbidden, nil)
	me.do(http.MethodDelete, base+"/tasks/exercise/complete", nil, http.StatusOK, nil)

	// Demo environment switch changes the phase and the Earth delay.
	var m struct {
		CurrentPhase struct{ Environment string } `json:"currentPhase"`
		Environment  struct{ CommDelayMaxS int }  `json:"environment"`
	}
	me.do(http.MethodPost, "/api/v1/demo/environment", map[string]string{"environment": "iss"}, http.StatusOK, &m)
	if m.CurrentPhase.Environment != "iss" || m.Environment.CommDelayMaxS > 5 {
		t.Errorf("after switch = %+v", m)
	}
	me.do(http.MethodPost, "/api/v1/demo/environment", map[string]string{"environment": "venus"}, http.StatusBadRequest, nil)
}
