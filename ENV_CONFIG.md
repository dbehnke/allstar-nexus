# Environment Configuration

## Required Configuration for Enhanced Features

To enable the enhanced AMI polling with XStat/SawStat (COS/PTT indicators, link modes, last heard, etc.), you **MUST** set the `AMI_NODE_ID` environment variable.

### Minimum Required Variables

```bash
# Your local AllStar node number (REQUIRED for enhanced features)
export AMI_NODE_ID=43732

# AMI connection details
export AMI_HOST=127.0.0.1
export AMI_PORT=5038
export AMI_USERNAME=admin
export AMI_PASSWORD=your-ami-password
```

### Complete Configuration Reference

#### Server Configuration
```bash
PORT=8080                          # HTTP server port (default: 8080)
APP_ENV=production                 # Environment: development, production (default: development)
```

#### Database
```bash
DB_PATH=data/allstar.db           # SQLite database path (default: data/allstar.db)
ASTDB_PATH=data/astdb.txt         # AllStar node database path (default: data/astdb.txt)
ASTDB_URL=http://allmondb.allstarlink.org/  # AllStar DB download URL
ASTDB_UPDATE_HOURS=24             # Hours between astdb auto-updates (default: 24)
```

#### Authentication
```bash
JWT_SECRET=your-secret-key        # JWT signing secret (CHANGE IN PRODUCTION!)
TOKEN_TTL_SECONDS=86400           # Token TTL in seconds (default: 86400 = 24 hours)
```

#### Rate Limiting
```bash
AUTH_RPM=60                       # Auth endpoints rate limit (requests/min, default: 60)
PUBLIC_STATS_RPM=120              # Public stats rate limit (requests/min, default: 120)
```

#### AMI Configuration
```bash
# Core AMI settings
AMI_ENABLED=true                  # Enable AMI connection (default: true)
AMI_HOST=127.0.0.1               # Asterisk AMI host (default: 127.0.0.1)
AMI_PORT=5038                     # Asterisk AMI port (default: 5038)
AMI_USERNAME=admin                # AMI username (default: admin)
AMI_PASSWORD=change-me            # AMI password (REQUIRED - change this!)

# IMPORTANT: Node ID for enhanced polling
AMI_NODE_ID=0                     # Your AllStar node number (default: 0 = disabled)
                                  # Set this to your node number to enable XStat/SawStat polling
                                  # Example: AMI_NODE_ID=43732

# Advanced AMI settings
AMI_EVENTS=on                     # AMI events mode (default: on)
AMI_RETRY_INTERVAL=15s            # Retry interval for AMI reconnection (default: 15s)
AMI_RETRY_MAX=60s                 # Max retry duration (default: 60s)
```

#### Feature Toggles
```bash
DISABLE_LINK_POLLER=false         # Disable link polling entirely (default: false)
ALLOW_ANON_DASHBOARD=true         # Allow anonymous dashboard access (default: true)
```

## Polling Behavior

### With AMI_NODE_ID Set (RECOMMENDED)
```bash
export AMI_NODE_ID=43732
```
- **EnhancedPoller** polls XStat + SawStat every **5 seconds**
- Provides:
  - ✅ Real-time COS/PTT indicators
  - ✅ Link modes (T/R/C/M)
  - ✅ IP addresses and directions
  - ✅ Last heard times
  - ✅ Connection details
  - ✅ Keying history

### Without AMI_NODE_ID (Legacy Mode)
```bash
# AMI_NODE_ID not set or =0
```
- **LinkPoller** polls `rpt stats` every **30 seconds**
- Provides:
  - ✅ Basic link list (node numbers only)
  - ❌ No COS/PTT indicators
  - ❌ No link modes
  - ❌ No last heard times
  - ❌ No connection details

## Example: Production Setup

Create a `.env` file or systemd environment file:

```bash
# Production configuration
APP_ENV=production
PORT=8080

# Security (CHANGE THESE!)
JWT_SECRET=your-strong-random-secret-here
AMI_PASSWORD=your-ami-password-here

# Your AllStar Node
AMI_NODE_ID=43732
AMI_HOST=127.0.0.1
AMI_PORT=5038
AMI_USERNAME=admin

# Database
DB_PATH=/var/lib/allstar-nexus/allstar.db
ASTDB_PATH=/var/lib/allstar-nexus/astdb.txt

# Optional: Restrict anonymous access
ALLOW_ANON_DASHBOARD=false
```

## Running with Environment Variables

### Command Line
```bash
AMI_NODE_ID=43732 AMI_PASSWORD=yourpass ./allstar-nexus
```

### systemd Service
```ini
[Unit]
Description=AllStar Nexus Dashboard
After=network.target asterisk.service

[Service]
Type=simple
User=asterisk
WorkingDirectory=/opt/allstar-nexus
ExecStart=/opt/allstar-nexus/allstar-nexus
Restart=always

# Environment variables
Environment="AMI_NODE_ID=43732"
Environment="AMI_HOST=127.0.0.1"
Environment="AMI_PASSWORD=yourpass"
Environment="JWT_SECRET=your-secret"
Environment="APP_ENV=production"
Environment="ALLOW_ANON_DASHBOARD=false"

[Install]
WantedBy=multi-user.target
```

### Docker
```dockerfile
FROM scratch
COPY allstar-nexus /
ENV AMI_NODE_ID=43732
ENV AMI_HOST=127.0.0.1
ENV AMI_PASSWORD=yourpass
EXPOSE 8080
CMD ["/allstar-nexus"]
```

## Troubleshooting

### "using legacy link poller" warning
**Cause:** `AMI_NODE_ID` not set or set to 0

**Fix:**
```bash
export AMI_NODE_ID=43732  # Your actual node number
```

### COS/PTT indicators not showing
**Symptoms:** Dashboard shows "Idle" but node is actually transmitting

**Cause:** EnhancedPoller not running (AMI_NODE_ID not configured)

**Fix:** Set AMI_NODE_ID to your node number and restart

### Link modes not displaying
**Cause:** Same as above - EnhancedPoller needs AMI_NODE_ID

**Fix:** Configure AMI_NODE_ID environment variable

### "AMI start error: connection refused"
**Cause:** AMI not reachable at configured host/port

**Fix:** Verify Asterisk is running and AMI is enabled in `/etc/asterisk/manager.conf`

## Verification

After starting with `AMI_NODE_ID` set, you should see in the logs:

```
{"level":"info","msg":"enhanced AMI poller started","node_id":43732,"interval":"5s"}
```

If you see:
```
{"level":"warn","msg":"using legacy link poller - set AMI_NODE_ID for enhanced features"}
```

Then `AMI_NODE_ID` is not properly configured.

## Security Notes

1. **NEVER commit `.env` files** with real credentials
2. **Change JWT_SECRET** in production (use a long random string)
3. **Change AMI_PASSWORD** from default
4. **Use HTTPS** in production (proxy with nginx/caddy)
5. **Restrict ALLOW_ANON_DASHBOARD=false** for private nodes

## Performance Impact

- **EnhancedPoller:** ~2 AMI requests per 5 seconds (XStat + SawStat)
- **LinkPoller:** ~1 AMI request per 30 seconds
- **Recommendation:** EnhancedPoller has minimal overhead and provides significantly better UX
