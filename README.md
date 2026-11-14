# aura-spaceapps-backend

Go API for **AURA — Astronaut Universal Readiness Assessment** (NASA Space Apps 2026,
challenge "Create Health Monitoring Software for Astronauts on Space Missions").

## Quick start

Requires Go 1.26+. The web UI lives in
[aura-spaceapps-frontend](https://github.com/SunatSS/aura-spaceapps-frontend) — clone it
next to this repository (same parent folder).

```bash
go run ./cmd/aura    # http://localhost:8080/api/v1/health
```

On first start with an empty database the server seeds a demo mission
(Mars Expedition MX-1, mission day 210, crew of four + flight surgeon, 30 days of readings).

Serve the web UI from the same binary:

```bash
AURA_STATIC_DIR=../aura-spaceapps-frontend/dist go run ./cmd/aura
```

## API

Contract: [`api/openapi.yaml`](api/openapi.yaml). Main endpoints under `/api/v1`:
`/indicators`, `/missions/current`, `/astronauts`, `/astronauts/{id}/readiness`,
`/astronauts/{id}/guidance`, `/astronauts/{id}/measurements` (GET/POST), `/astronauts/{id}/activities`.

Health content (indicators, thresholds, texts, NASA HRP risks) lives in
[`internal/catalog/catalog.yaml`](internal/catalog/catalog.yaml).

## Deploy (Render, free)

Render → New → Blueprint → this repository (`render.yaml`). The Docker image builds the
frontend from `aura-spaceapps-frontend` and serves API + UI on one URL. The free plan has no
persistent disk: the demo data is re-seeded on every start.

## Commands

| Task | Command |
|---|---|
| Run | `go run ./cmd/aura` |
| Test | `go test ./...` |
| Vet | `go vet ./...` |
| Format | `gofmt -w .` |
| Build | `go build -o bin/aura ./cmd/aura` |
| Docker | `docker build -t aura-spaceapps-backend .` |

Configuration: see `.env.example` (all variables optional).

Live demo: https://aura-spaceapps.onrender.com

## License

Apache License 2.0 — see [LICENSE](LICENSE).

## Credits and third-party licenses

| Component | License |
|---|---|
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure-Go SQLite) and its `modernc.org/*` dependencies | BSD 3-Clause |
| [go.yaml.in/yaml/v3](https://github.com/yaml/go-yaml) | MIT and Apache 2.0 |
| Go standard library | BSD 3-Clause |

Sources used for the health content (`internal/catalog/catalog.yaml`):

- NASA Human Research Program — [5 Hazards of Human Spaceflight](https://www.nasa.gov/hrp/hazards/), [Risks of Human Spaceflight](https://www.nasa.gov/hrp/risks/), [Ongoing HRP Research](https://www.nasa.gov/hrp/ongoing-hrp-research/) (incl. Standard Measures and ARCHeR).
- Canadian Space Agency — [Monitoring radiation for astronaut health](https://www.asc-csa.gc.ca/eng/news/articles/2026/2026-02-26-monitoring-radiation-astronaut-health.asp), [Why do we conduct science experiments in space?](https://www.asc-csa.gc.ca/eng/iss/science/why-do-we-conduct-science-experiments-in-space.asp).
- NASA, [NASA-STD-3001 Vol. 2, Appendix F: Display Standard](https://www.nasa.gov/reference/appendix-f-vol-2/) — alert classes and time formats used by the API.
- Indicator thresholds are illustrative general clinical references (AHA blood-pressure categories, ADA hypoglycemia levels, adult sleep recommendations); the NASA resources define which risks matter, not numeric limits.

AURA is a NASA Space Apps Challenge 2026 project; it is not endorsed by NASA and uses no NASA insignia or logotype.
