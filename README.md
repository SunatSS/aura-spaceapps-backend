# AURA — Astronaut Universal Readiness Assessment

A health self-check for crews far from Earth (NASA Space Apps 2026, challenge
"Create Health Monitoring Software for Astronauts on Space Missions"): log vitals, see what is
off and why it matters in space, get recommendations — including when to report to the flight surgeon.

Live demo: https://aura-spaceapps.onrender.com · API docs: https://aura-spaceapps.onrender.com/api/docs/

| Folder | What | Stack |
|---|---|---|
| [`backend/`](backend) | API, health rules, demo data; serves the built UI | Go 1.26+, SQLite (pure Go), `net/http` |
| [`frontend/`](frontend) | Web UI | React 19, TypeScript, Vite, Tailwind CSS v4, TanStack Query |

The API contract [`backend/api/openapi.yaml`](backend/api/openapi.yaml) is the single source of
truth: the frontend generates its types from it (`npm run gen:api`).

## Quick start with Docker

```bash
docker compose up --build    # http://localhost:8080
```

One image serves the API and the UI. The database lives in the `aura-data` volume; on first
start with an empty database the server seeds a demo mission (Mars Expedition MX-1, mission day
210, crew of four + flight surgeon, 30 days of readings). `docker compose down -v` resets it.
Settings: copy [`.env.example`](.env.example) to `.env` (all optional).

## Development without Docker

Requires Go 1.26+ and Node 22+.

```bash
cd backend && go run ./cmd/aura           # API on :8080
cd frontend && npm install && npm run dev  # UI on :5173, /api is proxied to :8080
```

Or serve the built UI from the Go binary: `cd frontend && npm run build`, then
`cd backend && AURA_STATIC_DIR=../frontend/dist go run ./cmd/aura`.

Sign in with a demo account — the password for all of them is `aura-mx1` (also shown on the sign-in screen):

| Username | Role | Sees |
|---|---|---|
| `rahimova`, `ferreira` | Crew | Own readings, recommendations and report |
| `morgan`, `sato` | Commander, deputy | Whole crew; log only their own readings |
| `chen` | Flight surgeon (on Earth) | Crew overview and reports |

## Commands

| Task | Backend (`backend/`) | Frontend (`frontend/`) |
|---|---|---|
| Run | `go run ./cmd/aura` | `npm run dev` |
| Test | `go test ./...` | `npm test` |
| Lint | `go vet ./...` · `gofmt -w .` | `npm run lint` |
| Build | `go build -o bin/aura ./cmd/aura` | `npm run build` |
| API types | — | `npm run gen:api` |
| Image | `docker build -t aura .` (from the repository root) | |

## API

Contract: [`backend/api/openapi.yaml`](backend/api/openapi.yaml), served at `/api/v1/openapi.yaml`; interactive docs (Swagger UI, embedded, works offline) at `/api/docs/`. Main endpoints under `/api/v1`:
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
[`backend/internal/catalog/catalog.yaml`](backend/internal/catalog/catalog.yaml).

## Design standard

The interface follows **NASA-STD-3001 Volume 2, Appendix F — Display Standard**: NASA's colour
table (red warning, yellow caution, blue advisory, cyan stale data), colour never the only cue,
6:1 text contrast, fixed-width numbers, an always-visible status area (MET/UTC, alerts, link to
Earth). See [`frontend/README.md`](frontend/README.md).

## Deploy (Render, free)

Render → New → Blueprint → this repository ([`render.yaml`](render.yaml)). The root
[`Dockerfile`](Dockerfile) builds the frontend and the Go server into one image that serves API + UI
on one URL. The free plan has no persistent disk: the demo data is re-seeded on every start.

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
| [Swagger UI](https://github.com/swagger-api/swagger-ui) (embedded API docs) | Apache 2.0 |
| [Inter](https://rsms.me/inter/) and [DM Mono](https://github.com/googlefonts/dm-mono) typefaces (via `@fontsource`) | SIL Open Font License 1.1 |
| React, React DOM, React Router, TanStack Query, Recharts, clsx | MIT |
| Vite, Tailwind CSS, Vitest, Testing Library, oxlint, openapi-typescript (dev tools) | MIT |

Sources used for the health content and the design:

- NASA Human Research Program — [5 Hazards of Human Spaceflight](https://www.nasa.gov/hrp/hazards/), [Risks of Human Spaceflight](https://www.nasa.gov/hrp/risks/), [Ongoing HRP Research](https://www.nasa.gov/hrp/ongoing-hrp-research/) (incl. Standard Measures and ARCHeR).
- Canadian Space Agency — [Monitoring radiation for astronaut health](https://www.asc-csa.gc.ca/eng/news/articles/2026/2026-02-26-monitoring-radiation-astronaut-health.asp), [Why do we conduct science experiments in space?](https://www.asc-csa.gc.ca/eng/iss/science/why-do-we-conduct-science-experiments-in-space.asp).
- NASA, [NASA-STD-3001 Vol. 2, Appendix F: Display Standard](https://www.nasa.gov/reference/appendix-f-vol-2/) — alert classes, colours and time formats.
- Indicator thresholds are illustrative general clinical references (ESC/ESH blood-pressure grades, ADA glycemic levels, adult sleep recommendations); the NASA resources define which risks matter, not numeric limits.

AURA is a NASA Space Apps Challenge 2026 project; it is not endorsed by NASA and uses no NASA insignia or logotype.
