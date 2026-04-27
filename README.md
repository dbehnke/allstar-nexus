# Allstar Nexus

Allstar Nexus is a full-stack application: a Go backend that serves APIs and an embedded frontend built with Vue 3 + Vite. It was built as the Dashboard / Admin interface for AllStar Link (ASL) 3 nodes and serves as the monitoring Dashboard for the Who Cares Allstar Hub (WC8MI) site. The project is distributed as a single binary that embeds the compiled frontend assets for simple deployment.

## At a glance

- **Backend**: Go (HTTP API, WebSocket hub, AMI connector, gamification logic)
- **Frontend**: Vue 3 + Vite (single-page app in `/frontend`)
- **Single-binary distribution**: frontend static files are built and embedded into the Go binary
- **Purpose**: Dashboard / Admin for AllStar Link (ASL) 3 nodes; serves as a monitoring dashboard for the Who Cares site at https://asl3.whocaresradio.com
- **Discord Integration**: Real-time notifications for node activity and QSOs (see [DISCORD.md](DISCORD.md))
- **Gamification**: Optional XP/leveling system for operators across multiple nodes

## Features

- **Real-time Monitoring**: WebSocket-based live updates of node activity
- **Link Tracking**: Monitor connected nodes and transmission statistics
- **Multi-Node Support**: Monitor multiple AllStar nodes from a single dashboard
- **Gamification System**: Optional XP/leveling system for operators (per-node or cross-node)
- **Discord Webhooks**: Get notified in Discord when your node is active, QSOs start/end, and individual stations talk
- **Authentication**: Secure admin interface with role-based access control
- **Node Lookup**: Search AllStar, IRLP, and EchoLink nodes by number or callsign
- **RPT Stats**: View detailed Asterisk statistics for connected nodes
- **Voter Display**: Visual RSSI monitoring for RTCM receivers

## Installation

For production deployment, see the **[Installation Guide](INSTALL.md)** which covers:

- **Package Installation**: Pre-built Debian (.deb) and RedHat (.rpm) packages with systemd service
- **Binary Installation**: Download and run pre-built binaries
- **Source Installation**: Build and install from source
- **Configuration**: Setting up AMI credentials, node numbers, and other options
- **Systemd Service**: Automatic service management with proper security hardening

Quick install on Debian/Ubuntu:
```bash
# Download the latest release from:
# https://github.com/dbehnke/allstar-nexus/releases/latest
wget https://github.com/dbehnke/allstar-nexus/releases/download/VERSION/allstar-nexus_VERSION_linux_amd64.deb
sudo dpkg -i allstar-nexus_*_linux_amd64.deb
sudo nano /etc/allstar-nexus/config.yaml  # Configure your setup
sudo systemctl enable allstar-nexus
sudo systemctl start allstar-nexus
```

## Development Quickstart

Prerequisites

- Go 1.21+
- Node.js and npm (for frontend development)
- [Task](https://taskfile.dev/) (optional but recommended — replaces Makefile)

Run locally (development)

1. Install frontend dependencies and start the dev server:

```bash
cd frontend
npm install
npm run dev
```

2. Run the backend (serves API + embedded frontend):

```bash
# from repository root
task run
# or: go run .
```

Build and run the full application (production-like)

```bash
# Build everything (frontend + Go binary)
task build

# Run the binary
./allstar-nexus --config ./config.yaml
```

Config validation
-----------------

Before starting the server, validate your YAML to catch common mistakes:

```bash
# Validate the default or provided config file
./allstar-nexus config validate --config ./config.yaml
```

If validation fails due to tab characters, fix them automatically:

```bash
expand -t 2 config.yaml > config-fixed.yaml
mv config-fixed.yaml config.yaml
```

If validation fails you can bypass it at your own risk using `--force`:

```bash
./allstar-nexus --config ./config.yaml --force
```

**Warning**: Using `--force` allows startup even if linting detects issues. When config parsing fails, the application falls back to default values (including `ami_host: 127.0.0.1`), which may not be what you want. This is intended for temporary debugging only.


Useful developer tasks

- Run backend tests:

```bash
task test-backend
# or: go test ./...
```

- Run frontend unit tests:

```bash
cd frontend
npm run test
```

- Run end-to-end tests:

```bash
cd frontend
npm run test:e2e
```

- Lint Go code:

```bash
task lint
```

## Project layout (high level)

- `/frontend` — Vue 3 + Vite app (dev scripts, builds, tests)
- `/backend` — Go packages: api, repository, models, gamification, middleware, ami, config
- `/tools` — Standalone utilities (e.g., cgnat-whitelist generator, ami-debug)
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

## Documentation

| Document | Description |
|----------|-------------|
| [INSTALL.md](INSTALL.md) | Production installation guide (packages, systemd, config) |
| [QUICKSTART.md](QUICKSTART.md) | Development quickstart and local setup |
| [FEATURES.md](FEATURES.md) | Feature overview and architecture details |
| [DISCORD.md](DISCORD.md) | Discord webhook integration setup |
| [DOCKER.md](DOCKER.md) | Docker deployment guide |
| [backend/README.md](backend/README.md) | Backend-specific development notes |
| [CHANGELOG.md](CHANGELOG.md) | Release notes and version history |

## Notes

- The frontend package.json indicates a Vite + Vue 3 stack. The README was updated to reflect the actual stack.
- For information on setting up automated releases, see [`.github/RELEASE_SETUP.md`](.github/RELEASE_SETUP.md)
- This project powers the Dashboard for The Who Cares Allstar Hub (WC8MI): https://asl3.whocaresradio.com

