# Allstar Nexus - Modern Vue Dashboard Features

This document describes the modern Vue.js dashboard implementation that brings Supermon public-facing features to Allstar Nexus with a modern, card-based UI.

## 🎯 Feature Parity with Supermon

### ✅ Implemented Features

#### 1. **Modern Dashboard (Home)**
- **Real-time Node Status Card** - Live status updates with uptime, RX/TX status, version info
- **Active Links Card** - Dynamic table showing all connected nodes with TX indicators
- **Top Links Card** - Visual ranking of most active nodes by TX time
- **Talker Log Card** - Real-time activity log of recent events
- **WebSocket Integration** - Live updates without page refresh
- **Responsive Grid Layout** - Modern card-based UI that adapts to screen size

#### 2. **Node/Callsign Lookup**
Route: `/lookup`

- Search for AllStar, IRLP, or EchoLink nodes
- Search by node number or callsign
- Displays node info, callsign, description, location
- Clean, searchable interface with real-time results
- Supports searching the astdb.txt database

**API Endpoint:** `GET /api/node-lookup?q=<search_term>`

#### 3. **RPT Statistics**
Route: `/rpt-stats` (Requires Authentication)

- View detailed Asterisk RPT statistics for any node
- Select from available connected nodes
- Terminal-style output display
- Real-time stats loading

**API Endpoint:** `GET /api/rpt-stats?node=<node_number>`

#### 4. **Voter/RTCM Display**
Route: `/voter` (Requires Authentication)

- Visual display of voter receiver signal strength
- Color-coded RSSI bars (0-255 range)
- Shows voted receiver in green
- Displays receiver types and IP addresses
- Real-time signal strength monitoring

**API Endpoint:** `GET /api/voter-stats?node=<node_number>`

---

## 🎨 Modern UI Features

### Design Highlights

- **Dark Theme** - Modern dark color scheme optimized for readability
- **Card-Based Layout** - Clean, organized information architecture
- **Gradient Buttons** - Modern blue gradient primary actions
- **Smooth Animations** - Subtle transitions and hover effects
- **Responsive Design** - Works on desktop, tablet, and mobile
- **Sticky Navigation** - Always-accessible top navbar
- **Real-time Indicators** - Pulsing badges for active TX/RX

### Component Architecture

```
vue-dashboard/src/
├── components/          # Reusable UI components
│   ├── Card.vue        # Base card component
│   ├── StatusCard.vue  # Node status display
│   ├── LinksCard.vue   # Active links table
│   └── TopLinksCard.vue # Top talkers ranking
├── views/              # Page-level components
│   ├── Dashboard.vue   # Main dashboard view
│   ├── NodeLookup.vue  # Search interface
│   ├── RptStats.vue    # RPT statistics
│   └── VoterDisplay.vue # Voter visualization
├── stores/             # Pinia state management
│   ├── auth.js         # Authentication state
│   └── node.js         # Node/link state
├── router/             # Vue Router config
│   └── index.js
└── main.js            # App entry point
```

---

## 🔧 Backend API Endpoints

### 1. Node Lookup
**Endpoint:** `GET /api/node-lookup`

**Query Parameters:**
- `q` - Search term (node number or callsign)

**Response:**
```json
{
  "ok": true,
  "data": {
    "query": "1999",
    "results": [
      {
        "node": "1999",
        "callsign": "W6XYZ",
        "description": "Test Node",
        "location": "San Francisco, CA"
      }
    ],
    "count": 1
  }
}
```

**Authentication:** Configurable (public or authenticated based on `ALLOW_ANON_DASHBOARD`)

---

### 2. RPT Statistics
**Endpoint:** `GET /api/rpt-stats`

**Query Parameters:**
- `node` - Node number (required)

**Response:**
```json
{
  "ok": true,
  "data": {
    "node": "1999",
    "stats": "RPT statistics output...",
    "parsed": {
      "key1": "value1",
      "key2": "value2"
    }
  }
}
```

**Authentication:** Required

---

### 3. Voter Statistics
**Endpoint:** `GET /api/voter-stats`

**Query Parameters:**
- `node` - Voter node number (required)

**Response:**
```json
{
  "ok": true,
  "data": {
    "node": "1999",
    "receivers": [
      {
        "id": "1",
        "name": "Receiver 1",
        "rssi": 142,
        "voted": true,
        "type": "voting",
        "ip": "192.168.1.10",
        "state": "active"
      }
    ]
  }
}
```

**Authentication:** Required

---

## 🚀 Running the Application

### Development Mode

Start the Vue development server:
```bash
cd vue-dashboard
npm run dev
```

Start the Go backend:
```bash
go run main.go
```

### Production Build

Build the Vue dashboard:
```bash
cd vue-dashboard
npm run build
```

Build the Go binary:
```bash
go build -o allstar-nexus .
```

Run the production server:
```bash
./allstar-nexus
```

---

## ⚙️ Configuration

### Environment Variables

- `ASTDB_PATH` - Path to astdb.txt (default: `/var/lib/asterisk/astdb.txt`)
- `ALLOW_ANON_DASHBOARD` - Allow public access to dashboard (default: `true`)
- `AMI_ENABLED` - Enable AMI connectivity (default: `false`)
- `AMI_HOST` - Asterisk Manager Interface host
- `AMI_PORT` - AMI port (default: `5038`)
- `AMI_USER` - AMI username
- `AMI_PASSWORD` - AMI password

### Configuration File

See `backend/config/config.go` for all available configuration options.

---

## 📊 Comparison: Supermon vs Allstar Nexus

| Feature | Supermon | Allstar Nexus |
|---------|----------|---------------|
| **UI Framework** | jQuery + PHP | Vue.js 3 + Go |
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

---

## 🎯 Key Improvements

### Performance
- **Faster page loads** - Single Page Application (SPA) architecture
- **Efficient updates** - WebSocket binary protocol vs HTTP polling
- **Smaller bundle** - Modern build tools with tree-shaking

### User Experience
- **No page reloads** - Smooth client-side routing
- **Real-time feedback** - Instant visual updates
- **Modern design** - Clean, professional interface
- **Mobile-friendly** - Responsive layout works on all devices

### Developer Experience
- **Type safety** - Go's strong typing for backend
- **Component reusability** - Vue.js single-file components
- **State management** - Centralized Pinia stores
- **Easy maintenance** - Modular, well-organized codebase

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

Potential additions to reach full Supermon parity:

- [ ] Connection logs viewer
- [ ] System stats (CPU temp, memory usage)
- [ ] Weather integration (via API)
- [ ] DTMF command interface
- [ ] Node restriction/ban management
- [ ] Configuration editor
- [ ] Archive playback
- [ ] Mobile app (React Native/Flutter)

---

## 📝 Notes

- The Vue dashboard is served from the same Go binary (embedded)
- All API endpoints follow RESTful conventions
- Error handling provides clear, actionable messages
- Rate limiting protects public endpoints
- CORS is disabled by default (same-origin)

---

## 🤝 Contributing

To add new features:

1. Create Vue components in `vue-dashboard/src/components/`
2. Add views in `vue-dashboard/src/views/`
3. Create Go handlers in `backend/api/`
4. Wire routes in `main.go`
5. Update this documentation

---

## 📄 License

Ham radio use only - NOT for commercial use.

---

**Built with ❤️ for the AllStar community**
