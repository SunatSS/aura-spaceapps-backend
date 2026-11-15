package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aura/internal/domain"
)

const testIngestToken = "test-ingest-token"

type vehicleBody struct {
	Status  domain.Status `json:"status"`
	Modules []struct {
		Code       string        `json:"code"`
		Status     domain.Status `json:"status"`
		Crew       []struct{ Name string }
		Parameters []struct {
			Code   string        `json:"code"`
			Status domain.Status `json:"status"`
			Value  *float64      `json:"value"`
		} `json:"parameters"`
	} `json:"modules"`
}

func (e testEnv) ingest(token string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(b))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	e.router.ServeHTTP(rr, req)
	return rr
}

func TestVehicleAndIngest(t *testing.T) {
	e := newTestEnv(t, true, "")
	amina := e.login("rahimova")

	var v vehicleBody
	amina.do(http.MethodGet, "/api/v1/vehicle", nil, http.StatusOK, &v)
	status := map[string]domain.Status{}
	for _, m := range v.Modules {
		status[m.Code] = m.Status
	}
	if status["hab"] != domain.StatusCaution || status["lab"] != domain.StatusNominal || len(v.Modules) != 3 {
		t.Fatalf("vehicle = %+v", status)
	}

	// Amina is in the habitat: CO₂ advice reaches her, with the cabin note on her headache.
	var me sessionBody
	amina.do(http.MethodGet, "/api/v1/auth/me", nil, http.StatusOK, &me)
	var g struct {
		Recommendations []struct {
			ID       string `json:"id"`
			Basis    string `json:"basis"`
			Supplies []struct {
				Code     string  `json:"code"`
				Location string  `json:"location"`
				Quantity float64 `json:"quantity"`
			} `json:"supplies"`
		} `json:"recommendations"`
	}
	amina.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/guidance", me.User.ID), nil, http.StatusOK, &g)
	got := map[string]string{}
	for _, r := range g.Recommendations {
		got[r.ID] = r.Basis
		if r.ID == "sleep_debt:sleep_hours" && (len(r.Supplies) != 1 || r.Supplies[0].Location == "" || r.Supplies[0].Quantity == 0) {
			t.Errorf("sleep debt supplies = %+v", r.Supplies)
		}
	}
	if !strings.HasPrefix(got["cabin_air:ppco2"], "Habitat: ppCO₂") || !strings.Contains(got["indicator_status:headache"], "Cabin air in Habitat") {
		t.Errorf("guidance = %v", got)
	}

	// Ingest: token required; a CO₂ spike in the lab makes it critical.
	reading := map[string]any{"telemetry": []any{map[string]any{"module": "lab", "parameter": "ppco2", "value": 6.2}}}
	if rr := e.ingest("", reading); rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d", rr.Code)
	}
	if rr := e.ingest("wrong", reading); rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: %d", rr.Code)
	}
	bad := []map[string]any{
		{"telemetry": []any{map[string]any{"module": "mars", "parameter": "ppco2", "value": 1}}},
		{"telemetry": []any{map[string]any{"module": "lab", "parameter": "co2", "value": 1}}},
		{"telemetry": []any{map[string]any{"module": "lab", "parameter": "ppco2", "value": 99}}},
		{"measurements": []any{map[string]any{"astronautId": me.User.ID, "indicator": "mood", "value": 3}}},
		{"measurements": []any{map[string]any{"astronautId": 9999, "indicator": "heart_rate", "value": 70}}},
		{},
	}
	for i, b := range bad {
		if rr := e.ingest(testIngestToken, b); rr.Code != http.StatusBadRequest {
			t.Errorf("bad[%d]: %d %s", i, rr.Code, rr.Body.String())
		}
	}
	rr := e.ingest(testIngestToken, map[string]any{
		"telemetry":    []any{map[string]any{"module": "lab", "parameter": "ppco2", "value": 6.2}},
		"measurements": []any{map[string]any{"astronautId": me.User.ID, "indicator": "heart_rate", "value": 71}},
	})
	if rr.Code != http.StatusAccepted || !strings.Contains(rr.Body.String(), `"telemetry":1`) {
		t.Fatalf("ingest: %d %s", rr.Code, rr.Body.String())
	}
	amina.do(http.MethodGet, "/api/v1/vehicle", nil, http.StatusOK, &v)
	for _, m := range v.Modules {
		if m.Code == "lab" && m.Status != domain.StatusCritical {
			t.Errorf("lab after CO₂ spike = %s, want critical", m.Status)
		}
	}
	var hist struct {
		Measurements []struct {
			Source string `json:"source"`
		} `json:"measurements"`
	}
	amina.do(http.MethodGet, fmt.Sprintf("/api/v1/astronauts/%d/measurements?indicator=heart_rate", me.User.ID), nil, http.StatusOK, &hist)
	if last := hist.Measurements[len(hist.Measurements)-1]; last.Source != "device" {
		t.Errorf("ingested reading source = %q", last.Source)
	}

	var tel struct {
		Telemetry []struct {
			Value  float64       `json:"value"`
			Status domain.Status `json:"status"`
		} `json:"telemetry"`
	}
	amina.do(http.MethodGet, "/api/v1/vehicle/telemetry?module=hab&parameter=ppco2", nil, http.StatusOK, &tel)
	if len(tel.Telemetry) < 100 {
		t.Errorf("7-day habitat CO₂ history = %d points", len(tel.Telemetry))
	}
	amina.do(http.MethodGet, "/api/v1/vehicle/telemetry?parameter=nope", nil, http.StatusBadRequest, nil)
}

type supplyBody struct {
	ID          int64         `json:"id"`
	Code        string        `json:"code"`
	Quantity    float64       `json:"quantity"`
	Status      domain.Status `json:"status"`
	Discrepancy float64       `json:"discrepancy"`
	Reasons     []string      `json:"reasons"`
}

func TestSupplies(t *testing.T) {
	e := newTestEnv(t, true, "")
	amina := e.login("rahimova")
	var list struct {
		Items []supplyBody `json:"items"`
	}
	amina.do(http.MethodGet, "/api/v1/supplies", nil, http.StatusOK, &list)
	byCode := map[string]supplyBody{}
	for _, it := range list.Items {
		byCode[it.Code] = it
	}
	gel, mel := byCode["glucose_gel"], byCode["melatonin"]
	if gel.ID == 0 || mel.Status != domain.StatusCritical {
		t.Fatalf("supplies = %+v", list.Items)
	}
	path := func(x supplyBody, action string) string { return fmt.Sprintf("/api/v1/supplies/%d/%s", x.ID, action) }

	var after supplyBody
	amina.do(http.MethodPost, path(gel, "use"), map[string]any{"quantity": 2, "reason": "Low glucose", "recommendationId": "indicator_status:glucose"}, http.StatusOK, &after)
	if after.Quantity != gel.Quantity-2 {
		t.Errorf("after use: %+v", after)
	}
	amina.do(http.MethodPost, path(gel, "use"), map[string]any{"quantity": 999, "reason": "x"}, http.StatusBadRequest, nil)
	amina.do(http.MethodPost, path(gel, "use"), map[string]any{"quantity": 1, "reason": " "}, http.StatusBadRequest, nil)
	amina.do(http.MethodPost, path(mel, "use"), map[string]any{"quantity": 1, "reason": "sleep"}, http.StatusBadRequest, nil) // expired

	amina.do(http.MethodPost, path(gel, "count"), map[string]any{"quantity": after.Quantity - 1}, http.StatusOK, &after)
	if after.Discrepancy != -1 || after.Status != domain.StatusCaution {
		t.Errorf("after count with a mismatch: %+v", after)
	}
	amina.do(http.MethodPost, path(gel, "dispose"), nil, http.StatusBadRequest, nil) // not expired
	amina.do(http.MethodPost, path(mel, "dispose"), nil, http.StatusOK, &after)
	if after.Quantity != 0 || after.Status != domain.StatusCritical || after.Reasons[0] != "Out of stock" {
		t.Errorf("after dispose: %+v", after)
	}

	var journal struct {
		Entries []struct {
			Kind          string `json:"kind"`
			SupplyName    string `json:"supplyName"`
			AstronautName string `json:"astronautName"`
			Reason        string `json:"reason"`
		} `json:"entries"`
	}
	amina.do(http.MethodGet, "/api/v1/supplies/journal", nil, http.StatusOK, &journal)
	if j := journal.Entries[0]; j.Kind != "dispose" || j.SupplyName != "Melatonin 3 mg" || j.AstronautName != "Amina Rahimova" {
		t.Errorf("latest journal entry = %+v", j)
	}

	e.login("chen").do(http.MethodPost, path(gel, "use"), map[string]any{"quantity": 1, "reason": "x"}, http.StatusForbidden, nil)
	amina.do(http.MethodPost, "/api/v1/supplies/9999/use", map[string]any{"quantity": 1, "reason": "x"}, http.StatusNotFound, nil)
}
