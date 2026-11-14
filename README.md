# aura-backend

Go API for **AURA — Astronaut Universal Readiness Assessment** (NASA Space Apps 2026,
challenge "Create Health Monitoring Software for Astronauts on Space Missions").

## Quick start

Requires Go 1.26+.

```bash
go mod tidy          # first run only: resolves deps, creates go.sum
go run ./cmd/aura    # http://localhost:8080/api/v1/health
```

## Commands

| Task | Command |
|---|---|
| Run | `go run ./cmd/aura` |
| Test | `go test ./...` |
| Vet | `go vet ./...` |
| Format | `gofmt -w .` |
| Build | `go build -o bin/aura ./cmd/aura` |
| Docker | `docker build -t aura-backend .` |

Configuration: see `.env.example` (all variables optional).

Product requirements and the work plan live in the workspace root: `../docs/`.
