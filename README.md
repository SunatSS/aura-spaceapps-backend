# aura-spaceapps-frontend

Web UI for **AURA — Astronaut Universal Readiness Assessment** (NASA Space Apps 2026,
challenge "Create Health Monitoring Software for Astronauts on Space Missions").

React 19 · TypeScript · Vite · Tailwind CSS v4 · TanStack Query · React Router · Recharts

## Quick start

Requires Node 22+. Clone both repositories **side by side** — the frontend reads the API
contract from `../aura-spaceapps-backend/api/openapi.yaml`:

```bash
git clone https://github.com/SunatSS/aura-spaceapps-backend.git
git clone https://github.com/SunatSS/aura-spaceapps-frontend.git
```

Start the backend first (`cd aura-spaceapps-backend && go run ./cmd/aura`), then:

```bash
cd aura-spaceapps-frontend
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
| `npm run gen:api` | Regenerate API types from `../aura-spaceapps-backend/api/openapi.yaml` |
