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
