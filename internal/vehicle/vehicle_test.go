package vehicle

import (
	"testing"
	"time"

	"aura/internal/catalog"
	"aura/internal/domain"
)

var now = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func reading(module, param string, v float64, ago time.Duration) domain.Telemetry {
	return domain.Telemetry{Module: module, Parameter: param, Value: v, MeasuredAt: now.Add(-ago)}
}

func TestAssess(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	latest := map[string]map[string]domain.Telemetry{
		"hab": {
			"ppco2":      reading("hab", "ppco2", 3.8, 10*time.Minute),
			"ppo2":       reading("hab", "ppo2", 21.2, 10*time.Minute),
			"cabin_temp": reading("hab", "cabin_temp", 22, 10*time.Hour), // stale
		},
		"lab": {
			"ppco2": reading("lab", "ppco2", 2.1, time.Minute),
			"ppo2":  reading("lab", "ppo2", 17.0, time.Minute), // below caution → critical
		},
	}
	res := Assess(Input{
		Catalog: cat,
		Modules: []domain.Module{{Code: "hab", Name: "Habitat"}, {Code: "lab", Name: "Laboratory"}, {Code: "airlock", Name: "Airlock"}},
		Latest:  latest,
		Crew:    []domain.Astronaut{{ID: 4, Name: "Amina", Module: "hab"}, {ID: 2, Name: "Alex", Module: "lab"}},
		Now:     now,
	})

	hab, _ := res.Module("hab")
	lab, _ := res.Module("lab")
	air, _ := res.Module("airlock")
	tests := []struct {
		name string
		got  domain.Status
		want domain.Status
	}{
		{"habitat: worst fresh parameter is CO₂ caution", hab.Status, domain.StatusCaution},
		{"laboratory: low oxygen is critical", lab.Status, domain.StatusCritical},
		{"airlock without telemetry is unknown", air.Status, domain.StatusUnknown},
		{"vehicle status is the worst module", res.Status, domain.StatusCritical},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, tt.got, tt.want)
		}
	}
	for _, p := range hab.Parameters {
		if p.Code == "cabin_temp" && (p.Fresh || p.Status != domain.StatusUnknown || p.Value == nil) {
			t.Errorf("stale temperature: %+v", p)
		}
	}
	if len(hab.Crew) != 1 || hab.Crew[0].Name != "Amina" || len(hab.Parameters) != len(cat.VehicleParameters) {
		t.Errorf("habitat = %+v", hab)
	}
}
