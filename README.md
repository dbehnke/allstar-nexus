
# Allstar Nexus

Allstar Nexus is a full-stack application: a Go backend that serves APIs and an embedded frontend built with Vue 3 + Vite. It was built as the Dashboard / Admin interface for Allstar Link (ASL) 3 nodes and serves as the monitoring Dashboard for the Who Cares Allstar Hub (WC8MI) site. The project is distributed as a single binary that embeds the compiled frontend assets for simple deployment.

## At a glance

- Backend: Go (HTTP API, gamification logic, repository layer)
- Frontend: Vue 3 + Vite (single-page app in `/frontend`)
- Single-binary distribution: frontend static files are built and embedded into the Go binary
- Purpose: Dashboard / Admin for Allstar Link (ASL) 3 nodes; serves as a monitoring dashboard for the Who Cares site at https://asl3.whocaresradio.com
- **Discord Integration**: Real-time notifications for node activity and QSOs (see [DISCORD.md](DISCORD.md))

## Features

- **Real-time Monitoring**: WebSocket-based live updates of node activity
- **Link Tracking**: Monitor connected nodes and transmission statistics
- **Gamification System**: Optional XP/leveling system for operators (see configuration)
- **Discord Webhooks**: Get notified in Discord when your node is active, QSOs start/end, and individual stations talk
- **Multi-Node Support**: Monitor multiple Allstar nodes from a single dashboard
- **Authentication**: Secure admin interface with role-based access control

## Installation

For production deployment, see the **[Installation Guide](INSTALL.md)** which covers:

- **Package Installation**: Pre-built Debian (.deb) and RedHat (.rpm) packages with systemd service
- **Binary Installation**: Download and run pre-built binaries
- **Source Installation**: Build and install using `make install`
- **Configuration**: Setting up AMI credentials, node numbers, and other options
- **Systemd Service**: Automatic service management with proper security hardening

Quick install on Debian/Ubuntu:
```bash
wget https://github.com/dbehnke/allstar-nexus/releases/latest/download/allstar-nexus_*_linux_amd64.deb
sudo dpkg -i allstar-nexus_*_linux_amd64.deb
sudo nano /etc/allstar-nexus/config.yaml  # Configure your setup
sudo systemctl enable allstar-nexus
sudo systemctl start allstar-nexus
```

## Development Quickstart

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

Config validation
-----------------

Before starting the server, you can lint and validate your YAML to catch common mistakes (for example, accidental tab indentation which YAML does not allow):

```bash
# validate the default or provided config file
./allstar-nexus config validate --config ./config.yaml
```

If validation fails you can either fix the config or bypass validation at your own risk using `--force` when starting the server:

```bash
# start server even if validation fails
./allstar-nexus --config ./config.yaml --force
```

Using `--force` will allow the server to continue startup even if linting/parsing detects issues; this is intended for temporary debugging only.


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
- `/tools` — Standalone utilities (e.g., cgnat-whitelist generator)
- `main.go` — application entrypoint; embeds/serves frontend

## Utilities

### CGNAT Whitelist Generator

A standalone tool for generating AllStar node whitelist entries for CGNAT forwarding scenarios.

Location: `tools/cgnat-whitelist/`

See [tools/cgnat-whitelist/README.md](tools/cgnat-whitelist/README.md) for detailed usage instructions.

Quick example:
```bash
cd tools/cgnat-whitelist
go build -o cgnat-whitelist .
./cgnat-whitelist -f callsigns.txt -o whitelist.txt -i 100.89.118.58
```

## Notes

- The frontend package.json indicates a Vite + Vue 3 stack (not Next.js). The README was updated to reflect the actual stack.
- See `/backend/README.md` for backend-specific development notes and the various `Makefile` / `tasks` present in the repo.
- For information on setting up automated releases, see [`.github/RELEASE_SETUP.md`](.github/RELEASE_SETUP.md)

- This project powers the Dashboard for The Who Cares Allstar Hub (WC8MI): https://asl3.whocaresradio.com

If you want I can also update the repository description on GitHub using the `gh` CLI.

