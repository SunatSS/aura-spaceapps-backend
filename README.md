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

Sign in with a demo account — the password for all of them is `aura-mx1` (also shown on the sign-in screen):

| Username | Role | Sees |
|---|---|---|
| `rahimova`, `ferreira` | Crew | Own readings, recommendations and report |
| `morgan`, `sato` | Commander, deputy | Whole crew; log only their own readings |
| `chen` | Flight surgeon (on Earth) | Crew overview and reports |

Serve the web UI from the same binary:

```bash
AURA_STATIC_DIR=../aura-spaceapps-frontend/dist go run ./cmd/aura
```

## API

Contract: [`api/openapi.yaml`](api/openapi.yaml), served at `/api/v1/openapi.yaml`; interactive docs (Swagger UI, embedded, works offline) at `/api/docs/` — live: https://aura-spaceapps.onrender.com/api/docs/. Main endpoints under `/api/v1`:
`/auth/login`, `/auth/me`, `/indicators`, `/missions/current`, `/astronauts`, `/astronauts/{id}/readiness`,
`/astronauts/{id}/guidance`, `/astronauts/{id}/measurements` (GET/POST), `/astronauts/{id}/activities`,
`/astronauts/{id}/report` (JSON or `?format=text`), `/crew/overview`, `/demo/reset`,
`/vehicle`, `/vehicle/telemetry`, `/supplies` (+ `/journal`, `/{id}/use|count|dispose`), `/ingest`,
`/astronauts/{id}/profile`, `/astronauts/{id}/tasks`, `/auth/passkeys/*`, `/demo/environment`.

Onboard systems (environment sensors, dosimeters, wearables) push readings to `POST /api/v1/ingest`
with a bearer token (`AURA_INGEST_TOKEN`; the public demo uses `aura-demo-ingest`):

```bash
curl -X POST https://aura-spaceapps.onrender.com/api/v1/ingest   -H "Authorization: Bearer aura-demo-ingest" -H "Content-Type: application/json"   -d '{"telemetry":[{"module":"lab","parameter":"ppco2","value":5.8}]}'
```

Access is checked on the server: the session is an HTTP-only cookie; passwords are hashed with
PBKDF2-SHA256 (Go standard library); crew members can open only their own data, the commander,
the deputy and the flight surgeon the whole crew, and readings are logged only by the crew member themself.

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

## Use of AI

The AURA team defined the concept, scope, indicators and scenarios and reviewed all results. Most of the code in this repository, its tests and documentation were written with the help of an AI coding assistant — **Claude (Anthropic)**. AI-assisted commits are marked with a `Co-Authored-By: Claude` line in the commit history. No AI-generated images, audio or video are used.

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
- Indicator thresholds are illustrative general clinical references (ESC/ESH blood-pressure grades, ADA glycemic levels, adult sleep recommendations); the NASA resources define which risks matter, not numeric limits.

AURA is a NASA Space Apps Challenge 2026 project; it is not endorsed by NASA and uses no NASA insignia or logotype.
