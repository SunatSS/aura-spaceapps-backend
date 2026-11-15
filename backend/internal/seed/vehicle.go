package seed

import (
	"context"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"aura/internal/domain"
	"aura/internal/storage"
)

// telemetry generates environment readings for every module (FR-85): normal
// life support, plus a CO₂ episode in the habitat over the last four nights —
// the same nights Amina sleeps badly and reports headaches.
//
// Hourly for the last 7 days, every 3 hours before; the last reading is a few
// minutes old so the vehicle display is fresh. Uses its own random source so
// the crew's health data stays exactly as scripted.
func (g *generator) telemetry(ctx context.Context, q storage.Queries, m domain.Mission, from, landingAt time.Time) error {
	rnd := rand.New(rand.NewPCG(2026, 210))
	n := func(sd float64) float64 { return rnd.NormFloat64() * sd }
	episodeStart := from.Add(co2EpisodeDay*day - 4*time.Hour) // evening before the first short night

	var times []time.Time
	for t := from; t.Before(g.now); {
		times = append(times, t)
		step := 3 * time.Hour
		if g.now.Sub(t) <= 7*day {
			step = time.Hour
		}
		t = t.Add(step)
	}
	times = append(times, g.now.Add(-5*time.Minute))

	var ts []domain.Telemetry
	for _, mod := range modules {
		for _, at := range times {
			hour := at.Hour()
			night := hour >= 21 || hour < 7
			co2 := 2.0 + n(0.15)
			if mod.code == "hab" {
				co2 = 2.3 + n(0.15)
				if night {
					co2 += 0.3 // crew asleep there
				}
				if !at.Before(episodeStart) {
					co2 = 3.6 + n(0.12) // degraded scrubber: above the 3 mmHg limit
					if night {
						co2 = 4.3 + n(0.15)
					}
				}
			}
			temp := 22.5 + n(0.3)
			if mod.code == "airlock" {
				temp = 20 + n(0.4)
			}
			dose := 9 + n(0.7) // Mars surface background
			if at.Before(landingAt) {
				dose = 20 + n(1.2) // deep-space transit
			}
			add := func(param string, v float64) {
				p, _ := g.cat.VehicleParameter(param)
				v = math.Round(math.Min(math.Max(v, p.Min), p.Max)/p.Step) * p.Step
				ts = append(ts, domain.Telemetry{
					MissionID: m.ID, Module: mod.code, Parameter: param, Value: math.Round(v*100) / 100,
					Unit: p.Unit, MeasuredAt: at, Source: domain.SourceDevice,
				})
			}
			add("cabin_pressure", 101.3+n(0.15))
			add("ppo2", 21.2+n(0.2))
			add("ppco2", co2)
			add("cabin_temp", temp)
			add("humidity", 45+n(2.5))
			add("dose_rate", dose)
		}
	}
	return q.CreateTelemetry(ctx, ts)
}

type use struct {
	daysAgo float64
	who     int // crew index, -1 = nobody in particular
	qty     float64
	reason  string
	rec     string
}

type stock struct {
	code, name, category, unit, location string
	quantity, min                        float64
	expiresIn, countedAgo                int // days; 0 = no expiry / never counted
	discrepancy                          float64
	uses                                 []use
}

// kit is the demo medical inventory: mostly in order, with one item of each
// state the display can show (FR-91).
var kit = []stock{
	{code: "glucose_gel", name: "Glucose gel 15 g", category: domain.SupplyConsumable, unit: "sachet", location: "Habitat · rack 2 · bin C",
		quantity: 14, min: 6, expiresIn: 410, countedAgo: 12,
		uses: []use{{daysAgo: 9.4, who: sato, qty: 1, reason: "Low glucose before EVA", rec: "indicator_status:glucose"}}},
	{code: "glucose_strips", name: "Glucometer test strips", category: domain.SupplyConsumable, unit: "strip", location: "Habitat · rack 2 · bin B",
		quantity: 70, min: 40, expiresIn: 300, countedAgo: 8}, // daily morning checks → runs out before the mission ends
	{code: "pulse_oximeter", name: "Pulse oximeter", category: domain.SupplyEquipment, unit: "unit", location: "Habitat · medical locker", quantity: 2, min: 1, countedAgo: 8},
	{code: "bp_cuff", name: "Blood pressure monitor", category: domain.SupplyEquipment, unit: "unit", location: "Habitat · medical locker", quantity: 2, min: 1, countedAgo: 8},
	{code: "thermometer", name: "Digital thermometer", category: domain.SupplyEquipment, unit: "unit", location: "Habitat · medical locker", quantity: 2, min: 1, countedAgo: 8},
	{code: "oxygen_mask", name: "Oxygen mask", category: domain.SupplyEquipment, unit: "unit", location: "Laboratory · emergency panel", quantity: 6, min: 4, countedAgo: 20},
	{code: "oxygen_cylinder", name: "Portable O₂ cylinder", category: domain.SupplyOxygen, unit: "cylinder", location: "Airlock · rack 1", quantity: 3, min: 2, countedAgo: 45,
		uses: []use{{daysAgo: 1.1, who: ferreira, qty: 0.5, reason: "Supplemental oxygen after EVA", rec: "post_eva_hypoxia:spo2"}}},
	{code: "sleep_kit", name: "Sleep kit (eye mask, ear plugs)", category: domain.SupplyConsumable, unit: "set", location: "Habitat · crew quarters", quantity: 9, min: 4, countedAgo: 12,
		uses: []use{{daysAgo: 2.6, who: rahimova, qty: 1, reason: "Poor sleep", rec: "sleep_debt:sleep_hours"}}},
	{code: "paracetamol", name: "Paracetamol 500 mg", category: domain.SupplyMedication, unit: "tablet", location: "Habitat · medical locker · drawer 1",
		quantity: 46, min: 20, expiresIn: 21, countedAgo: 12,
		uses: []use{{daysAgo: 1.6, who: rahimova, qty: 2, reason: "Headache"}}},
	{code: "loratadine", name: "Loratadine 10 mg", category: domain.SupplyMedication, unit: "tablet", location: "Habitat · medical locker · drawer 1", quantity: 8, min: 10, expiresIn: 380, countedAgo: 12},
	{code: "melatonin", name: "Melatonin 3 mg", category: domain.SupplyMedication, unit: "tablet", location: "Habitat · medical locker · drawer 2", quantity: 18, min: 10, expiresIn: -3, countedAgo: 12},
	{code: "eye_drops", name: "Saline eye drops", category: domain.SupplyMedication, unit: "bottle", location: "Habitat · medical locker · drawer 2", quantity: 6, min: 3, expiresIn: 200, countedAgo: 12},
	{code: "bandages", name: "Adhesive bandages", category: domain.SupplyConsumable, unit: "piece", location: "Laboratory · first-aid kit", quantity: 38, min: 20, countedAgo: 2, discrepancy: -2},
	{code: "gloves", name: "Nitrile gloves", category: domain.SupplyConsumable, unit: "pair", location: "Laboratory · rack 4", quantity: 400, min: 100, countedAgo: 20},
}

// supplies stocks the medical kit and writes 30 days of the supply journal.
func (g *generator) supplies(ctx context.Context, q storage.Queries, m domain.Mission) error {
	crewIDs, err := q.ListAstronauts(ctx, m.ID)
	if err != nil {
		return err
	}
	idOf := map[string]int64{}
	for _, a := range crewIDs {
		idOf[a.Name] = a.ID
	}
	for _, k := range kit {
		uses := slices.Clone(k.uses)
		switch k.code {
		case "glucose_strips":
			for d := range windowDays {
				uses = append(uses, use{daysAgo: float64(windowDays-d) - 0.7, who: -1, qty: 4, reason: "Morning glucose checks, 4 crew"})
			}
		case "gloves":
			for d := 0; d < windowDays; d += 2 {
				uses = append(uses, use{daysAgo: float64(windowDays-d) - 0.5, who: -1, qty: 1, reason: "Lab work"})
			}
		}
		slices.SortFunc(uses, func(a, b use) int { return int(math.Copysign(1, b.daysAgo-a.daysAgo)) })

		x := domain.Supply{MissionID: m.ID, Code: k.code, Name: k.name, Category: k.category, Unit: k.unit,
			Quantity: k.quantity, MinQuantity: k.min, Location: k.location, Discrepancy: k.discrepancy}
		if k.expiresIn != 0 {
			t := g.now.Add(time.Duration(k.expiresIn) * day).Truncate(day)
			x.ExpiresAt = &t
		}
		if k.countedAgo != 0 {
			t := g.now.Add(-time.Duration(k.countedAgo) * day)
			x.LastCountedAt = &t
		}
		if err := q.CreateSupply(ctx, &x); err != nil {
			return err
		}

		running := k.quantity
		for _, u := range uses {
			running += u.qty
		}
		for _, u := range uses {
			running -= u.qty
			t := domain.SupplyTransaction{SupplyID: x.ID, Kind: domain.SupplyUse, Delta: -u.qty, QuantityAfter: running,
				Reason: u.reason, RecommendationID: u.rec, CreatedAt: g.now.Add(-time.Duration(u.daysAgo * float64(day)))}
			if u.who >= 0 {
				id := idOf[crew[u.who].name]
				t.AstronautID = &id
			}
			if err := q.CreateSupplyTransaction(ctx, &t); err != nil {
				return err
			}
		}
	}
	return nil
}
