# aura-frontend

Web UI for **AURA — Astronaut Universal Readiness Assessment** (NASA Space Apps 2026,
challenge "Create Health Monitoring Software for Astronauts on Space Missions").

React 19 · TypeScript · Vite · Tailwind CSS v4 · TanStack Query · React Router · Recharts

## Quick start

Requires Node 22+. Start the backend first (`../aura-backend`, `go run ./cmd/aura`).

```bash
npm install
npm run dev        # http://localhost:5173
```

## Scripts

| Script | What it does |
|---|---|
| `npm run dev` | Dev server; `/api` is proxied to `localhost:8080` |
| `npm run build` | Type-check and production build to `dist/` |
| `npm test` | Unit tests (Vitest) |
| `npm run lint` | oxlint |
| `npm run gen:api` | Regenerate API types from `../aura-backend/api/openapi.yaml` |

Product requirements and the work plan live in the workspace root: `../docs/`.
