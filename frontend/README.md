# AURA frontend

Web UI for **AURA — Astronaut Universal Readiness Assessment** (NASA Space Apps 2026,
challenge "Create Health Monitoring Software for Astronauts on Space Missions").

React 19 · TypeScript · Vite · Tailwind CSS v4 · TanStack Query · React Router · Recharts

## Quick start

Requires Node 22+. This folder is part of the AURA monorepo; the API lives in `../backend`
and its contract `../backend/api/openapi.yaml` is the source of the frontend's API types.
To run everything in one container, use `docker compose up --build` from the repository root.

Start the backend first (`cd backend && go run ./cmd/aura`), then:

```bash
cd frontend
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
| `npm run gen:api` | Regenerate API types from `../backend/api/openapi.yaml` |

Live demo: https://aura-spaceapps.onrender.com (free tier — first load after idle takes about a minute).
Sign in with one of the demo accounts listed on the sign-in screen (password `aura-mx1`): a crew member
sees their own data, the commander and the flight surgeon see the whole crew.

## Design standard

The interface follows **NASA-STD-3001 Volume 2, Appendix F — Display Standard**
(https://www.nasa.gov/reference/appendix-f-vol-2/): NASA's required colour table
(red warning, yellow caution, blue advisory, cyan stale data, gray unavailable), colour never
as the only cue, 6:1 text contrast, mixed-case text, fixed-width numbers, top navigation with
an always-visible status area (MET/UTC, alerts, link to Earth), time labelled with its time
system. No decorative graphics.

## Use of AI

The AURA team defined the concept, scope, indicators and scenarios and reviewed all results. Most of the code in this repository, its tests and documentation were written with the help of an AI coding assistant — **Claude (Anthropic)**. AI-assisted commits are marked with a `Co-Authored-By: Claude` line in the commit history. No AI-generated images, audio or video are used.

## License

Apache License 2.0 — see [LICENSE](../LICENSE).

## Credits and third-party licenses

| Component | License |
|---|---|
| [Inter](https://rsms.me/inter/) typeface (via `@fontsource/inter`) | SIL Open Font License 1.1 |
| [DM Mono](https://github.com/googlefonts/dm-mono) typeface (via `@fontsource/dm-mono`) | SIL Open Font License 1.1 |
| React, React DOM, React Router, TanStack Query, Recharts, clsx | MIT |
| Vite, Tailwind CSS, Vitest, Testing Library, oxlint, openapi-typescript (dev tools) | MIT |

Sources used for the design:

- NASA, *NASA-STD-3001 Vol. 2, Appendix F: Display Standard* (updated 2026-04-22) — https://www.nasa.gov/reference/appendix-f-vol-2/
- NASA Human Research Program — 5 Hazards of Human Spaceflight (https://www.nasa.gov/hrp/hazards/) and Risks of Human Spaceflight (https://www.nasa.gov/hrp/risks/)
- Typefaces chosen to match those used on nasa.gov (Inter, DM Mono).

AURA is a NASA Space Apps Challenge 2026 project; it is not endorsed by NASA and uses no NASA insignia or logotype.
