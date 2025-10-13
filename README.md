
# Allstar Nexus

Allstar Nexus is a full-stack application: a Go backend that serves APIs and an embedded frontend built with Vue 3 + Vite. It was built as the Dashboard / Admin interface for Allstar Link (ASL) 3 nodes and serves as the monitoring Dashboard for the Who Cares Allstar Hub (WC8MI) site. The project is distributed as a single binary that embeds the compiled frontend assets for simple deployment.

## At a glance

- Backend: Go (HTTP API, gamification logic, repository layer)
- Frontend: Vue 3 + Vite (single-page app in `/frontend`)
- Single-binary distribution: frontend static files are built and embedded into the Go binary
 - Purpose: Dashboard / Admin for Allstar Link (ASL) 3 nodes; serves as a monitoring dashboard for the Who Cares site at https://asl3.whocaresradio.com

## Quickstart

Prerequisites

- Go (1.18+ recommended)
- Node.js (for frontend development) and npm or pnpm

Run locally (development)

1. Install frontend dependencies and start the dev server (frontend only):

```bash
cd frontend
npm install
npm run dev
```

2. Run the backend (serves API only in dev mode):

```bash
# from repository root
go run .
```

Build and run the full application (production-like)

```bash
# build frontend
cd frontend && npm run build && cd ..

# build Go binary (frontend is embedded when built/packaged)
go build -o allstar-nexus main.go

# run the binary
./allstar-nexus --config ./config.yaml
```

Useful developer tasks

- Run backend tests:

```bash
go test ./backend/...
```

- Run frontend unit tests and e2e (from `frontend`):

```bash
npm run test
npm run test:e2e
```

## Project layout (high level)

- `/frontend` — Vue 3 + Vite app (dev scripts, builds, tests)
- `/backend` — Go packages: api, repository, models, gamification, middleware, tests
- `main.go` — application entrypoint; embeds/serves frontend

## Notes

- The frontend package.json indicates a Vite + Vue 3 stack (not Next.js). The README was updated to reflect the actual stack.
- See `/backend/README.md` for backend-specific development notes and the various `Makefile` / `tasks` present in the repo.

- This project powers the Dashboard for The Who Cares Allstar Hub (WC8MI): https://asl3.whocaresradio.com

If you want I can also update the repository description on GitHub using the `gh` CLI.

