# Allstar Nexus - Features & Architecture

This document describes the Allstar Nexus application architecture, key features, and design decisions.

## 🎯 Core Features

### 1. Dashboard (Home)
- **Real-time Node Status Card** — Live status updates with uptime, RX/TX status, version info
- **Active Links Card** — Dynamic table showing all connected nodes with TX indicators
- **Top Links Card** — Visual ranking of most active nodes by TX time
- **Talker Log Card** — Real-time activity log of recent events
- **WebSocket Integration** — Live updates without page refresh
- **Responsive Grid Layout** — Modern card-based UI that adapts to screen size

### 2. Node/Callsign Lookup
- Search AllStar, IRLP, or EchoLink nodes by number or callsign
- Backed by auto-downloaded `astdb.txt` from allmondb.allstarlink.org
- Clean, searchable interface with real-time results

### 3. RPT Statistics (Admin Only)
- View detailed Asterisk RPT statistics for any connected node
- Select from available connected nodes
- Terminal-style output display

### 4. Voter/RTCM Display (Admin Only)
- Visual display of voter receiver signal strength
- Color-coded RSSI bars (0–255 range)
- Shows voted receiver in green
- Real-time signal strength monitoring

### 5. Gamification System (Optional)
- XP/leveling system for operators across one or more nodes
- Per-node or cross-node tracking
- Configurable level groupings and thresholds
- Anti-cheat protections (duplicate detection, time-gating)
- Leaderboard and operator statistics

### 6. Discord Webhooks (Optional)
- Real-time notifications for node activity
- QSO start/end announcements
- Individual station talk notifications
- Per-node and per-type deduplication
- See [DISCORD.md](DISCORD.md) for setup

---

## 🎨 UI Architecture

### Design Highlights

- **Dark Theme** — Modern dark color scheme optimized for readability
- **Card-Based Layout** — Clean, organized information architecture
- **Responsive Design** — Works on desktop, tablet, and mobile
- **Real-time Indicators** — Pulsing badges for active TX/RX
- **Single Page Application** — Vue 3 with client-side routing

### Component Architecture

```
frontend/src/
├── components/          # Reusable UI components
│   ├── Card.vue
│   ├── StatusCard.vue
│   ├── LinksCard.vue
│   └── TopLinksCard.vue
├── views/              # Page-level components
│   ├── Dashboard.vue
│   ├── NodeLookup.vue
│   ├── RptStats.vue
│   └── VoterDisplay.vue
├── stores/             # Pinia state management
│   ├── auth.ts
│   └── node.ts
├── router/             # Vue Router config
│   └── index.ts
└── main.ts             # App entry point
```

---

## 🔧 Backend Architecture

### API Endpoints

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/node-lookup?q=` | Public | Search nodes/callsigns |
| `GET /api/rpt-stats?node=` | Required | Asterisk RPT statistics |
| `GET /api/voter-stats?node=` | Required | RTCM voter RSSI data |
| `GET /api/nodes` | Public | List configured nodes |
| `GET /api/nodes/:id/status` | Public | Real-time node status (WebSocket) |
| `POST /api/auth/login` | Public | JWT authentication |
| `POST /api/auth/register` | Public | First user becomes superadmin |

### Go Package Layout

```
backend/
├── api/                # HTTP handlers and middleware
├── ami/                # Asterisk Manager Interface client
├── auth/               # JWT authentication
├── config/             # YAML configuration loader
├── gamification/       # XP/leveling logic
├── models/             # Data models (Node, Link, Talker, etc.)
├── repository/         # SQLite persistence layer
└── websocket/          # WebSocket hub for real-time updates
```

---

## ⚙️ Configuration

Allstar Nexus uses a single YAML configuration file. See `config.yaml.example` for a fully annotated template.

Key sections:

- `ami_*` — Asterisk Manager Interface connection
- `nodes` — List of AllStar node numbers to monitor
- `jwt_secret` — JWT signing secret (change in production!)
- `gamification` — Enable/disable and configure XP/leveling
- `discord` — Webhook URLs and notification settings
- `astdb_path` / `astdb_url` — Node database location

---

## 📊 Technology Comparison

| Feature | Legacy Supermon | Allstar Nexus |
|---------|----------------|---------------|
| **UI Framework** | jQuery + PHP | Vue 3 + Go |
| **Real-time Updates** | Server-Sent Events | WebSockets |
| **Authentication** | Session-based | JWT tokens |
| **Design** | Classic tables | Modern cards |
| **Responsive** | Limited | Fully responsive |
| **State Management** | DOM manipulation | Pinia stores |
| **API** | PHP scripts | RESTful Go APIs |
| **Node Lookup** | ✅ PHP | ✅ Go + Vue |
| **RPT Stats** | ✅ PHP | ✅ Go + Vue |
| **Voter Display** | ✅ PHP | ✅ Go + Vue |
| **Link Status** | ✅ SSE | ✅ WebSocket |
| **Gamification** | ❌ | ✅ |
| **Discord Notifications** | ❌ | ✅ |

---

## 🚀 Running the Application

### Development Mode

```bash
# Terminal 1: Vue dev server (hot reload, port 5173)
cd frontend
npm run dev

# Terminal 2: Go backend (port 8080)
task run
```

### Production Build

```bash
# Build everything (frontend + Go binary)
task build

# Run the binary
./allstar-nexus --config ./config.yaml
```

---

## 🔐 Authentication Flow

1. User clicks "Admin Login" in navbar
2. Login panel slides down
3. User enters email/password
4. JWT token stored in localStorage
5. Token sent with authenticated API requests
6. Protected routes (RPT Stats, Voter) become accessible

---

## 🛠️ Future Enhancements

Potential additions:

- Connection logs viewer
- System stats (CPU temp, memory usage)
- Weather integration
- DTMF command interface
- Node restriction/ban management
- Configuration editor in UI
- Archive playback

---

## 📝 Notes

- The Vue dashboard is served from the same Go binary (embedded static files)
- All API endpoints follow RESTful conventions
- Error handling provides clear, actionable messages
- Rate limiting protects public endpoints
- CORS is disabled by default (same-origin)

---

## 🤝 Contributing

To add new features:

1. Create Vue components in `frontend/src/components/`
2. Add views in `frontend/src/views/`
3. Create Go handlers in `backend/api/`
4. Wire routes in `main.go`
5. Update relevant documentation

---

## 📄 License

Ham radio use only — NOT for commercial use.

---

**Built with ❤️ for the AllStar community**
