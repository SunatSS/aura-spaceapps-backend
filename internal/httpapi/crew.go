package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"aura/internal/auth"
	"aura/internal/domain"
	"aura/internal/guidance"
	"aura/internal/readiness"
	"aura/internal/report"
	"aura/internal/storage"
)

// evaluation is everything the engines say about one member right now.
type evaluation struct {
	result  readiness.Result
	recs    []guidance.Recommendation
	history []domain.Measurement // last guidanceHistory, oldest first
}

func (h *handlers) evaluate(ctx context.Context, a domain.Astronaut, m domain.Mission, now time.Time) (evaluation, error) {
	q := h.deps.DB.Q()
	latest, err := q.LatestMeasurements(ctx, a.ID)
	if err != nil {
		return evaluation{}, fmt.Errorf("latest measurements: %w", err)
	}
	history, err := q.ListMeasurements(ctx, storage.MeasurementFilter{AstronautID: a.ID, From: now.Add(-guidanceHistory)})
	if err != nil {
		return evaluation{}, fmt.Errorf("history: %w", err)
	}
	res := readiness.Assess(readiness.Input{Catalog: h.deps.Catalog, Mission: m, Latest: latest, Now: now})
	recs := guidance.Evaluate(guidance.Input{Catalog: h.deps.Catalog, Mission: m, Assessment: res, History: history, Now: now})
	return evaluation{result: res, recs: recs, history: history}, nil
}

// --- flight-surgeon report (FR-32) --------------------------------------------------

func (h *handlers) report(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAstronaut(w, r)
	if !ok {
		return
	}
	if a.Role == domain.RoleFlightSurgeon {
		writeError(w, http.StatusBadRequest, "the flight surgeon is on Earth and has no health readings")
		return
	}
	format := r.URL.Query().Get("format")
	if format != "" && format != "json" && format != "text" {
		writeError(w, http.StatusBadRequest, "format must be json or text")
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	now := h.deps.Now().UTC()
	ev, err := h.evaluate(r.Context(), a, m, now)
	if err != nil {
		h.internal(w, "report", err)
		return
	}
	rep := report.Build(report.Input{
		Catalog: h.deps.Catalog, Mission: m, Astronaut: a, Assessment: ev.result,
		Recommendations: ev.recs, History: ev.history, Now: now,
	})
	if format != "text" {
		writeJSON(w, http.StatusOK, rep)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, reportFilename(a, rep.Mission.Day, "txt")))
	_, _ = w.Write([]byte(report.Text(rep)))
}

// reportFilename is e.g. "aura-report-rahimova-md210.txt".
func reportFilename(a domain.Astronaut, day int, ext string) string {
	f := strings.Fields(a.Name)
	name := "member"
	if len(f) > 0 {
		name = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, strings.ToLower(f[len(f)-1]))
	}
	return fmt.Sprintf("aura-report-%s-md%d.%s", name, day, ext)
}

// --- crew overview (FR-55, FR-73) -----------------------------------------------------

type trendPoint struct {
	MeasuredAt time.Time     `json:"t"`
	Value      float64       `json:"v"`
	Status     domain.Status `json:"status"`
}

type trendSeries struct {
	Indicator string       `json:"indicator"`
	Points    []trendPoint `json:"points"`
}

type crewMember struct {
	Astronaut       domain.Astronaut          `json:"astronaut"`
	Readiness       readiness.Result          `json:"readiness"`
	Warnings        int                       `json:"warnings"`
	Cautions        int                       `json:"cautions"`
	Escalate        bool                      `json:"escalate"`
	Recommendations []guidance.Recommendation `json:"recommendations"`
	Trends          []trendSeries             `json:"trends"` // last 7 days, every catalog indicator
}

type crewOverviewResponse struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	MissionDay  int          `json:"missionDay"`
	Members     []crewMember `json:"members"`
}

// crewOverview is the whole crew at a glance for the commander, the deputy
// and the flight surgeon: readiness, alerts and 7-day trends per member.
func (h *handlers) crewOverview(w http.ResponseWriter, r *http.Request) {
	if !auth.SeesWholeCrew(currentUser(r).Role) {
		writeError(w, http.StatusForbidden, "the crew overview is for the commander, the deputy and the flight surgeon")
		return
	}
	m, ok := h.loadMission(w, r)
	if !ok {
		return
	}
	all, err := h.deps.DB.Q().ListAstronauts(r.Context(), m.ID)
	if err != nil {
		h.internal(w, "list astronauts", err)
		return
	}
	now := h.deps.Now().UTC()
	resp := crewOverviewResponse{GeneratedAt: now, MissionDay: m.Day(now), Members: []crewMember{}}
	for _, a := range all {
		if a.Role == domain.RoleFlightSurgeon {
			continue
		}
		ev, err := h.evaluate(r.Context(), a, m, now)
		if err != nil {
			h.internal(w, "crew overview", err)
			return
		}
		mem := crewMember{Astronaut: a, Readiness: ev.result, Recommendations: ev.recs}
		for _, rec := range ev.recs {
			switch rec.Severity {
			case domain.StatusCritical:
				mem.Warnings++
			case domain.StatusCaution:
				mem.Cautions++
			}
			mem.Escalate = mem.Escalate || rec.Escalate
		}
		for _, ind := range h.deps.Catalog.Indicators {
			s := trendSeries{Indicator: ind.Code, Points: []trendPoint{}}
			for _, x := range ev.history {
				if x.Indicator == ind.Code {
					s.Points = append(s.Points, trendPoint{MeasuredAt: x.MeasuredAt, Value: x.Value, Status: readiness.IndicatorStatus(ind, x, m)})
				}
			}
			slices.SortFunc(s.Points, func(a, b trendPoint) int { return a.MeasuredAt.Compare(b.MeasuredAt) })
			mem.Trends = append(mem.Trends, s)
		}
		resp.Members = append(resp.Members, mem)
	}
	writeJSON(w, http.StatusOK, resp)
}
