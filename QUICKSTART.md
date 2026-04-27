# Quick Start Guide

Get Allstar Nexus running locally in a few minutes for development or testing.

> For **production deployment** (systemd, Debian/RPM packages), see [INSTALL.md](INSTALL.md) instead.

---

## Prerequisites

- **Go** 1.21+
- **Node.js** 20+ and npm
- [Task](https://taskfile.dev/) (recommended — runs build/test/lint targets)
- An AllStar node with **AMI access** (optional, but most features depend on it)

---

## 🚀 Quick Setup (5 Minutes)

### Step 1: Clone and Install Dependencies

```bash
git clone https://github.com/dbehnke/allstar-nexus.git
cd allstar-nexus

# Frontend deps
cd frontend && npm install && cd ..

# Go modules
go mod download
```

### Step 2: Configure

Copy and edit the example config:

```bash
cp config.yaml.example config.yaml
# Edit config.yaml — set ami_host, ami_user, ami_password, nodes, jwt_secret
```

Validate the config:

```bash
go run . config validate --config ./config.yaml
```

### Step 3: Build and Run

```bash
# Build frontend + Go binary in one step
task build

# Run the binary
./allstar-nexus --config ./config.yaml
```

Or for development with hot-reload of the frontend:

```bash
# Terminal 1 — Vue dev server (port 5173, proxies API to backend)
cd frontend
npm run dev

# Terminal 2 — Go backend (port 8080)
task run
```

Open your browser to `http://localhost:8080` (production build) or `http://localhost:5173` (dev mode with hot-reload).

---

## 🎮 Using the Dashboard

### First Time Setup

1. **Create Admin Account**
   - Click "Admin Login" in the navbar
   - Enter your email and password
   - The first user registered becomes superadmin automatically

2. **Explore the Dashboard**
   - **Dashboard** — Real-time node status and active links
   - **Node Lookup** — Search nodes/callsigns from astdb
   - **RPT Stats** — Detailed Asterisk node statistics (admin only)
   - **Voter** — RTCM receiver RSSI visualization (admin only)
   - **Gamification** — XP, levels, top operators (if enabled in config)

---

## 🔍 Features at a Glance

### Dashboard
- Real-time WebSocket updates of node activity
- Active links table with TX indicators
- Top talkers ranking
- Live event log

### Node Lookup
- Search AllStar / IRLP / EchoLink by node number or callsign
- Backed by auto-downloaded astdb.txt

### RPT Stats (Admin)
- Per-node Asterisk statistics from AMI

### Voter Display (Admin)
- Color-coded RSSI bars for RTCM receivers

### Gamification (Optional)
- XP/leveling per operator across one or more nodes
- Configurable level groupings, anti-cheat protections
- See `config.yaml.example` for `gamification:` section

### Discord Notifications (Optional)
- Webhooks for node activity, QSO start/end, individual talkers
- See [DISCORD.md](DISCORD.md)

---

## 🛠️ Common Development Tasks

```bash
# Run all tests (Go + Vitest)
task test

# Run only Go tests
task test-backend

# Run only frontend unit tests
task test-frontend

# Run Playwright e2e tests
task test-e2e

# Lint Go code
task lint

# Build standalone tools (e.g., cgnat-whitelist)
task tools

# Clean build artifacts
task clean
```

---

## 📁 Project Structure

```
allstar-nexus/
├── frontend/               # Vue 3 + Vite SPA
│   ├── src/
│   │   ├── components/     # Reusable UI components
│   │   ├── views/          # Page-level components
│   │   ├── stores/         # Pinia state management
│   │   └── router/         # Vue Router
│   └── package.json
├── backend/                # Go backend
│   ├── api/                # HTTP API handlers
│   ├── ami/                # Asterisk Manager Interface client
│   ├── auth/               # Authentication (JWT)
│   ├── config/             # Configuration loader
│   ├── gamification/       # XP/leveling logic
│   ├── repository/         # SQLite data layer
│   └── models/             # Data models
├── tools/                  # Standalone CLI utilities
│   ├── cgnat-whitelist/
│   └── ami-debug/
├── docs/                   # Design docs and plans
├── Taskfile.yml            # Build/test/lint targets
├── config.yaml.example     # Annotated config template
└── main.go                 # Application entrypoint
```

---

## 🐛 Troubleshooting

### Configuration file issues (YAML syntax errors)

If your config isn't being read or AMI is connecting to `127.0.0.1` instead of your configured host:

1. **Validate**:
   ```bash
   ./allstar-nexus config validate --config ./config.yaml
   ```

2. **Check for tabs** — YAML requires SPACES, not TABS:
   ```bash
   grep -P "^\t" config.yaml
   ```

3. **Fix tabs automatically**:
   ```bash
   expand -t 2 config.yaml > config-fixed.yaml
   mv config-fixed.yaml config.yaml
   ```

When config parsing fails, the app falls back to defaults (including `ami_host: 127.0.0.1`).

### Dashboard not loading
- Ensure the frontend was built: `cd frontend && npm run build`
- Check `frontend/dist/` exists
- Or use dev mode: `npm run dev` from `frontend/`

### AMI features not working
- Verify AMI is enabled in `/etc/asterisk/manager.conf` on the AllStar node
- Check credentials in `config.yaml` match Asterisk
- Confirm the node is reachable from the host running Allstar Nexus

### Can't login
- The first registered user becomes superadmin
- Check the database file exists (default `allstar.db`)
- Verify `jwt_secret` is set in config

### Node lookup returns no results
- astdb.txt is auto-downloaded from `allmondb.allstarlink.org` on startup
- Check write permissions on the data directory
- Override with `astdb_path` and `astdb_url` in config if needed

---

## 📊 Quick API Test

```bash
# Public node lookup
curl "http://localhost:8080/api/node-lookup?q=1999"

# Login to get a JWT
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"yourpassword"}'

# Authenticated request (use token from login response)
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/rpt-stats?node=1999"
```

---

## 🔒 Security Notes

- **Always** change `jwt_secret` in production
- Use HTTPS via a reverse proxy (nginx, Caddy) in production
- Set strong admin passwords
- Restrict AMI access to localhost or trusted networks
- Firewall port 8080 to trusted sources

---

## 🎯 Next Steps

1. Read [FEATURES.md](FEATURES.md) for an architecture overview
2. Set up [Discord notifications](DISCORD.md)
3. Configure gamification in `config.yaml`
4. For deployment, follow [INSTALL.md](INSTALL.md)

---

**Happy monitoring! 73 from the AllStar Nexus team** 📡
