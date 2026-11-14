package httpapi

import (
	"fmt"
	"net/http"
	"testing"

	"aura/internal/domain"
)

type astronautsBody struct {
	Astronauts []domain.Astronaut `json:"astronauts"`
}

// findAstronaut looks a member up as the commander, who sees the whole crew.
func findAstronaut(t *testing.T, e testEnv, name string) domain.Astronaut {
	t.Helper()
	var body astronautsBody
	e.login("morgan").do(http.MethodGet, "/api/v1/astronauts", nil, http.StatusOK, &body)
	for _, a := range body.Astronauts {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("astronaut %q not found in %+v", name, body.Astronauts)
	return domain.Astronaut{}
}

func TestMissionAndCatalog(t *testing.T) {
	e := newTestEnv(t, true, "")

	var mission struct {
		MissionDay   int `json:"missionDay"`
		CurrentPhase struct {
			Environment string `json:"environment"`
		} `json:"currentPhase"`
		Environment struct {
			GravityG float64 `json:"gravityG"`
		} `json:"environment"`
		Phases []any `json:"phases"`
	}
	e.do(http.MethodGet, "/api/v1/missions/current", nil, http.StatusOK, &mission)
	if mission.MissionDay != 210 || mission.CurrentPhase.Environment != "mars_surface" || mission.Environment.GravityG != 0.38 || len(mission.Phases) != 2 {
		t.Errorf("mission = %+v", mission)
	}

	var cat struct {
		Disclaimer string           `json:"disclaimer"`
		Indicators []map[string]any `json:"indicators"`
		Systems    []any            `json:"systems"`
		Risks      []any            `json:"risks"`
	}
	e.do(http.MethodGet, "/api/v1/indicators", nil, http.StatusOK, &cat)
	if cat.Disclaimer == "" || len(cat.Indicators) != 10 || len(cat.Systems) != 7 || len(cat.Risks) == 0 {
		t.Errorf("catalog: %d indicators, %d systems, %d risks", len(cat.Indicators), len(cat.Systems), len(cat.Risks))
	}
	if _, ok := cat.Indicators[0]["howToMeasure"]; !ok {
		t.Errorf("indicator JSON lacks howToMeasure: %v", cat.Indicators[0])
	}

	var crew astronautsBody
	e.login("chen").do(http.MethodGet, "/api/v1/astronauts", nil, http.StatusOK, &crew)
	if len(crew.Astronauts) != 5 {
		t.Errorf("crew size = %d, want 5", len(crew.Astronauts))
	}
}

type readinessBody struct {
	Score            *int          `json:"score"`
	Status           domain.Status `json:"status"`
	CappedByCritical bool          `json:"cappedByCritical"`
	Systems          []struct {
		Code   string        `json:"code"`
		Status domain.Status `json:"status"`
	} `json:"systems"`
}

type guidanceBody struct {
	CommDelayMaxS   int `json:"commDelayMaxS"`
	Recommendations []struct {
		ID       string `json:"id"`
		Escalate bool   `json:"escalate"`
	} `json:"recommendations"`
}

// TestDemoScenario follows the presentation script: enter a critical blood
// pressure for Amina and watch readiness and guidance react.
func TestDemoScenario(t *testing.T) {
	e := newTestEnv(t, true, "")
	amina := findAstronaut(t, e, "Amina Rahimova")
	e = e.login("rahimova")
	base := fmt.Sprintf("/api/v1/astronauts/%d", amina.ID)

	var before readinessBody
	e.do(http.MethodGet, base+"/readiness", nil, http.StatusOK, &before)
	if before.Score == nil || *before.Score < 50 {
		t.Fatalf("before: score = %v, want >= 50", before.Score)
	}

	var created struct {
		Measurements []struct {
			Status domain.Status `json:"status"`
		} `json:"measurements"`
	}
	e.do(http.MethodPost, base+"/measurements", map[string]any{"measurements": []map[string]any{
		{"indicator": "bp_systolic", "value": 165},
		{"indicator": "bp_diastolic", "value": 100},
	}}, http.StatusCreated, &created)
	if len(created.Measurements) != 2 || created.Measurements[0].Status != domain.StatusCritical {
		t.Fatalf("created = %+v", created)
	}

	var after readinessBody
	e.do(http.MethodGet, base+"/readiness", nil, http.StatusOK, &after)
	if after.Score == nil || *after.Score > 49 || !after.CappedByCritical || after.Status != domain.StatusCritical {
		t.Fatalf("after: %+v", after)
	}

	var g guidanceBody
	e.do(http.MethodGet, base+"/guidance", nil, http.StatusOK, &g)
	ids := map[string]bool{}
	for _, r := range g.Recommendations {
		ids[r.ID] = true
	}
	if !ids["indicator_status:bp_systolic"] || ids["indicator_status:bp_diastolic"] || !ids["sleep_debt:sleep_hours"] {
		t.Errorf("recommendations = %+v", g.Recommendations)
	}
	if g.CommDelayMaxS != 1320 {
		t.Errorf("comm delay = %d, want 1320 on Mars", g.CommDelayMaxS)
	}

	var hist struct {
		Measurements []struct {
			MissionDay  int    `json:"missionDay"`
			Environment string `json:"environment"`
		} `json:"measurements"`
	}
	e.do(http.MethodGet, base+"/measurements?indicator=bp_systolic", nil, http.StatusOK, &hist)
	if n := len(hist.Measurements); n < 50 {
		t.Fatalf("history length = %d", n)
	}
	first, last := hist.Measurements[0], hist.Measurements[len(hist.Measurements)-1]
	if first.Environment != "mars_transit" || last.Environment != "mars_surface" || last.MissionDay != 210 {
		t.Errorf("history spans %+v → %+v", first, last)
	}

	var acts struct {
		Activities []domain.Activity `json:"activities"`
	}
	e.do(http.MethodGet, base+"/activities", nil, http.StatusOK, &acts)
	kinds := map[string]int{}
	for _, a := range acts.Activities {
		kinds[a.Kind]++
	}
	if kinds["landing"] != 1 || kinds["eva"] != 2 {
		t.Errorf("activities = %v", kinds)
	}
}

func TestCreateMeasurementValidation(t *testing.T) {
	e := newTestEnv(t, true, "")
	lucas := findAstronaut(t, e, "Lucas Ferreira")
	surgeon := findAstronaut(t, e, "Dr. Sarah Chen")
	e = e.login("ferreira")
	path := fmt.Sprintf("/api/v1/astronauts/%d/measurements", lucas.ID)

	tests := []struct {
		name string
		body any
	}{
		{"empty batch", map[string]any{"measurements": []any{}}},
		{"unknown indicator", map[string]any{"measurements": []any{map[string]any{"indicator": "x", "value": 1}}}},
		{"missing value", map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate"}}}},
		{"impossible value", map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 700}}}},
		{"fractional scale", map[string]any{"measurements": []any{map[string]any{"indicator": "mood", "value": 3.5}}}},
		{"future time", map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 70, "measuredAt": "2030-01-01T00:00:00Z"}}}},
		{"bad context", map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 70, "context": "lunch"}}}},
		{"unknown field", map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 70, "unit": "bpm"}}}},
		{"bad source", map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 70, "source": "satellite"}}}},
		{"device on a scale", map[string]any{"measurements": []any{map[string]any{"indicator": "mood", "value": 3, "source": "device"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.do(http.MethodPost, path, tt.body, http.StatusBadRequest, nil)
		})
	}

	ok := map[string]any{"measurements": []any{map[string]any{"indicator": "heart_rate", "value": 70}}}
	e.do(http.MethodPost, fmt.Sprintf("/api/v1/astronauts/%d/measurements", surgeon.ID), ok, http.StatusForbidden, nil)
	e.do(http.MethodPost, "/api/v1/astronauts/9999/measurements", ok, http.StatusNotFound, nil)
	e.do(http.MethodGet, "/api/v1/astronauts/abc/readiness", nil, http.StatusBadRequest, nil)
	e.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/measurements?indicator=nope", lucas.ID), nil, http.StatusBadRequest, nil)
	e.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/measurements?from=yesterday", lucas.ID), nil, http.StatusBadRequest, nil)

	var g guidanceBody
	e.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/guidance", lucas.ID), nil, http.StatusOK, &g)
	if len(g.Recommendations) == 0 || g.Recommendations[0].ID != "post_eva_hypoxia:spo2" || !g.Recommendations[0].Escalate {
		t.Errorf("Lucas guidance = %+v", g.Recommendations)
	}
}
