# Quick Start Guide - Modern Allstar Nexus Dashboard

Get up and running with the new modern Vue.js dashboard in minutes!

## 🚀 Quick Setup (5 Minutes)

### Step 1: Install Dependencies

```bash
# Install Vue dashboard dependencies
cd vue-dashboard
npm install

# Go dependencies (already vendored)
cd ..
go mod download
```

### Step 2: Build Everything

```bash
# Build Vue dashboard for production
cd vue-dashboard
npm run build
cd ..

# Build Go backend (includes embedded Vue dashboard)
go build -o allstar-nexus .
```

### Step 3: Configure (Optional)

Create a `.env` file or set environment variables:

```bash
# ✨ NEW: astdb.txt is now AUTO-DOWNLOADED!
# The AllStar node database is automatically downloaded from allmondb.allstarlink.org
# No manual setup required - just works out of the box! 🎉

# Optional: Customize astdb location (default: data/astdb.txt)
export ASTDB_PATH="data/astdb.txt"

# Optional: Use alternative database source
export ASTDB_URL="http://allmondb.allstarlink.org/"

# Optional: Change update frequency (default: 24 hours)
export ASTDB_UPDATE_HOURS=24

# 🔥 IMPORTANT: Enable enhanced AMI features (COS/PTT, link modes, last heard)
export AMI_NODE_ID=43732  # YOUR AllStar node number - REQUIRED for enhanced features!

# AMI connection settings
export AMI_ENABLED=true
export AMI_HOST="localhost"
export AMI_PORT=5038
export AMI_USER="admin"
export AMI_PASSWORD="yourpassword"

# Optional: Allow public dashboard access
export ALLOW_ANON_DASHBOARD=true
```

### Step 4: Run It!

```bash
./allstar-nexus
```

Open your browser to `http://localhost:8080`

---

## 🎮 Using the Dashboard

### First Time Setup

1. **Create Admin Account**
   - Click "Admin Login" in the navbar
   - The login panel will appear
   - Enter your email and password
   - Click "Login"
   - First user becomes superadmin automatically

2. **Explore the Dashboard**
   - **Dashboard** - View real-time node status and active links
   - **Node Lookup** - Search for nodes and callsigns
   - **RPT Stats** - View detailed node statistics (requires login)
   - **Voter** - Monitor RTCM receivers (requires login)

---

## 🔍 Features at a Glance

### Dashboard Page
- Real-time status updates via WebSocket
- Active links with TX indicators
- Top talker rankings
- Event log

### Node Lookup
- Search by node number or callsign
- AllStar, IRLP, EchoLink support
- Instant results

### RPT Stats (Admin Only)
- Select any connected node
- View detailed Asterisk statistics
- Terminal-style output

### Voter Display (Admin Only)
- Visual RSSI bars
- Color-coded receiver status
- Real-time signal strength

---

## 🛠️ Development Mode

For development with hot-reload:

```bash
# Terminal 1: Vue dev server with hot reload
cd vue-dashboard
npm run dev

# Terminal 2: Go backend
go run main.go
```

The Vue dev server runs on port 5173, proxying API calls to the Go backend on port 8080.

---

## 📁 Project Structure

```
allstar-nexus/
├── vue-dashboard/          # Modern Vue.js frontend
│   ├── src/
│   │   ├── components/     # Reusable UI components
│   │   ├── views/          # Page components
│   │   ├── stores/         # Pinia state management
│   │   ├── router/         # Vue Router
│   │   └── App.vue         # Root component
│   ├── dist/               # Production build output
│   └── package.json
├── backend/                # Go backend
│   ├── api/                # API handlers
│   ├── auth/               # Authentication
│   ├── config/             # Configuration
│   ├── database/           # Database layer
│   └── models/             # Data models
├── internal/               # Internal packages
│   ├── ami/                # AMI connector
│   ├── core/               # Core logic
│   └── web/                # WebSocket hub
└── main.go                 # Application entry point
```

---

## 🐛 Troubleshooting

### Configuration file issues (YAML syntax errors)

If you're using `config.yaml` instead of environment variables and it's not being read:

1. **Validate your config file**:
   ```bash
   ./allstar-nexus config validate
   ```

2. **Check for tabs in your config** - YAML requires SPACES, not TABS:
   ```bash
   cat -A config.yaml | grep "^I"
   ```
   If you see `^I`, those are tabs that need to be replaced with spaces.

3. **Fix tabs automatically**:
   ```bash
   expand -t 2 config.yaml > config-fixed.yaml
   mv config-fixed.yaml config.yaml
   ```

4. **Watch startup logs for config errors**:
   ```bash
   ./allstar-nexus 2>&1 | grep -i "config\|ERROR"
   ```

When config parsing fails, the app falls back to defaults (including `ami_host: 127.0.0.1`).

### Dashboard not loading?
- Make sure you built the Vue app: `cd vue-dashboard && npm run build`
- Check the `vue-dashboard/dist/` directory exists

### AMI features not working?
- Verify AMI is enabled in Asterisk (`/etc/asterisk/manager.conf`)
- Check AMI credentials are correct
- Ensure `AMI_ENABLED=true` is set

### Can't login?
- The first user created becomes superadmin
- Check the database file exists (default: `nexus.db`)
- Verify JWT_SECRET is set

### Node lookup returns no results?
- Check `ASTDB_PATH` points to valid astdb.txt file
- Default location: `/var/lib/asterisk/astdb.txt`
- File format: `node|callsign|description|location`

---

## 📊 API Testing

Test the new API endpoints:

```bash
# Node lookup (public)
curl "http://localhost:8080/api/node-lookup?q=1999"

# Login to get token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"yourpassword"}'

# RPT stats (authenticated)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/rpt-stats?node=1999"

# Voter stats (authenticated)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/voter-stats?node=1999"
```

---

## 🔒 Security Notes

- Change default JWT_SECRET in production
- Use HTTPS in production
- Set strong passwords for admin accounts
- Restrict AMI access to localhost if possible
- Consider firewall rules for port 8080

---

## 📝 Configuration Reference

### All Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `nexus.db` | SQLite database path |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `TOKEN_TTL` | `24h` | JWT expiration time |
| `ASTDB_PATH` | `/var/lib/asterisk/astdb.txt` | AllStar database path |
| `ALLOW_ANON_DASHBOARD` | `true` | Allow public dashboard access |
| `AMI_ENABLED` | `true` | Enable AMI connectivity (set to false to run without AMI) |
| `AMI_HOST` | `localhost` | AMI host address |
| `AMI_PORT` | `5038` | AMI port |
| `AMI_USER` | `admin` | AMI username |
| `AMI_PASSWORD` | `password` | AMI password |
| `AUTH_RATE_LIMIT_RPM` | `10` | Login attempts per minute |
| `PUBLIC_STATS_RATE_LIMIT_RPM` | `60` | Public API rate limit |

---

## 🎯 Next Steps

1. **Customize the theme** - Edit colors in `vue-dashboard/src/App.vue`
2. **Add more nodes** - Configure multiple nodes in your allmon.ini
3. **Enable AMI** - Unlock RPT Stats and Voter features
4. **Set up monitoring** - Use the real-time dashboard for node monitoring
5. **Explore the code** - Check out `FEATURES.md` for architecture details

---

## 💡 Tips

- Use the browser dev tools (F12) to debug WebSocket connections
- Check the browser console for any JavaScript errors
- Monitor the Go backend logs for API issues
- The dashboard updates in real-time without page refreshes
- Mobile users: bookmark the dashboard for quick access

---

## 🤝 Getting Help

If you run into issues:

1. Check this guide and `FEATURES.md`
2. Review the browser console for errors
3. Check backend logs for API errors
4. Verify AMI connectivity if using RPT/Voter features
5. Search GitHub issues or create a new one

---

**Happy monitoring! 73 from the AllStar Nexus team** 📡
