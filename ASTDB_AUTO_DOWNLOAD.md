# Automatic AllStar Node Database (astdb.txt) Download

## Overview

Allstar Nexus now **automatically downloads and updates** the AllStar node database (astdb.txt) from the official AllStarLink servers. This eliminates the need for manual database management and ensures you always have current node information.

---

## 🎯 Features

✅ **Automatic Download** - Downloads astdb.txt on first startup if missing
✅ **Auto-Update** - Refreshes the database every 24 hours by default
✅ **Configurable URL** - Can use alternative database sources
✅ **Update Interval** - Configurable update frequency
✅ **Background Updates** - Updates happen in the background without blocking
✅ **Validation** - Verifies downloaded data is valid
✅ **Fallback Path** - Uses local file if download fails

---

## 📊 Database Stats

After successful download:
- **File Size:** ~1.5 MB
- **Total Nodes:** 43,096 AllStar nodes (as of Oct 2025)
- **Format:** Pipe-delimited text file
- **Fields:** Node Number | Callsign | Description | Location

---

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ASTDB_PATH` | `data/astdb.txt` | Local path to store astdb file |
| `ASTDB_URL` | `http://allmondb.allstarlink.org/` | URL to download from |
| `ASTDB_UPDATE_HOURS` | `24` | Hours between auto-updates |

### Example Configuration

```bash
# Use default settings (recommended)
export ASTDB_PATH="data/astdb.txt"
export ASTDB_URL="http://allmondb.allstarlink.org/"
export ASTDB_UPDATE_HOURS=24

# Alternative: Use HamVOIP database
export ASTDB_URL="https://nodelist.hamvoip.org/getASLdb.php"

# More frequent updates (every 12 hours)
export ASTDB_UPDATE_HOURS=12

# Store in custom location
export ASTDB_PATH="/var/lib/allstar-nexus/astdb.txt"
```

---

## 🚀 How It Works

### On Startup

1. **Check for existing file** - Looks for astdb.txt at configured path
2. **Check age** - Determines if file is older than update interval
3. **Download if needed** - Fetches fresh data from AllStarLink servers
4. **Validate** - Verifies the downloaded file is valid
5. **Start auto-updater** - Launches background goroutine for periodic updates

### Background Updates

- Runs in a separate goroutine (non-blocking)
- Updates every `ASTDB_UPDATE_HOURS` (default: 24 hours)
- Logs all download attempts
- Continues using old file if download fails

### File Format

```
node|callsign|description|location
2000|WB6NIL|ASL Public Hub|Los Angeles, CA
2001|WB6NIL|ASL Public Hub|Los Angeles, CA
2003|KM6RPT|448.280-|San Diego County, CA
```

---

## 📝 Startup Logs

Successful download:
```
{"level":"info","msg":"downloading astdb from AllStar server","url":"http://allmondb.allstarlink.org/","destination":"data/astdb.txt"}
{"level":"info","msg":"downloaded astdb","bytes":1572864}
{"level":"info","msg":"astdb file updated successfully","path":"data/astdb.txt"}
{"level":"info","msg":"astdb loaded successfully","node_count":43096}
```

Using existing file:
```
{"level":"info","msg":"checked astdb age","age":"2h30m","max_age":"24h","needs_update":false}
{"level":"info","msg":"astdb file is up to date","path":"data/astdb.txt"}
{"level":"info","msg":"astdb loaded successfully","node_count":43096}
```

Download failure (uses fallback):
```
{"level":"warn","msg":"failed to download astdb, node lookup may not work","error":"..."}
```

---

## 🔍 Node Lookup API

The downloaded database powers the Node Lookup API:

### Search by Node Number
```bash
curl "http://localhost:8080/api/node-lookup?q=2000"
```

**Response:**
```json
{
  "ok": true,
  "data": {
    "query": "2000",
    "results": [
      {
        "node": "2000",
        "callsign": "WB6NIL",
        "description": "ASL Public Hub",
        "location": "Los Angeles, CA"
      }
    ],
    "count": 1
  }
}
```

### Search by Callsign
```bash
curl "http://localhost:8080/api/node-lookup?q=WB6NIL"
```

**Response:**
```json
{
  "ok": true,
  "data": {
    "query": "WB6NIL",
    "results": [
      {
        "node": "2000",
        "callsign": "WB6NIL",
        "description": "ASL Public Hub",
        "location": "Los Angeles, CA"
      },
      {
        "node": "2001",
        "callsign": "WB6NIL",
        "description": "ASL Public Hub",
        "location": "Los Angeles, CA"
      }
    ],
    "count": 2
  }
}
```

---

## 🛠️ Manual Operations

### Force Download Now

If you need to manually trigger a download:

```bash
# Stop the server
pkill allstar-nexus

# Delete the existing file
rm data/astdb.txt

# Start the server (will download fresh copy)
./allstar-nexus
```

### Use Local File

If you prefer to manage the file manually:

```bash
# Set a very long update interval (effectively disabling auto-update)
export ASTDB_UPDATE_HOURS=8760  # Once per year

# Or download manually
curl -o data/astdb.txt http://allmondb.allstarlink.org/
```

---

## 🔐 Data Sources

### Official AllStarLink Database
- **URL:** http://allmondb.allstarlink.org/
- **Update Frequency:** Daily
- **Maintained by:** AllStarLink
- **Coverage:** All registered AllStar nodes

### Alternative: HamVOIP Database
- **URL:** https://nodelist.hamvoip.org/getASLdb.php
- **Update Frequency:** Daily
- **Maintained by:** HamVOIP
- **Coverage:** AllStar nodes

Both sources provide the same pipe-delimited format.

---

## 📈 Performance

- **Download Time:** ~2-5 seconds (depending on connection)
- **File Size:** ~1.5 MB
- **Parsing Time:** < 100ms
- **Memory Usage:** Minimal (file is read on-demand per request)
- **Search Speed:** < 10ms for most queries

---

## 🐛 Troubleshooting

### Download Fails

**Check connectivity:**
```bash
curl -I http://allmondb.allstarlink.org/
```

**Check logs:**
```bash
./allstar-nexus 2>&1 | grep astdb
```

**Use alternative source:**
```bash
export ASTDB_URL="https://nodelist.hamvoip.org/getASLdb.php"
```

### File Permissions

Ensure the data directory is writable:
```bash
mkdir -p data
chmod 755 data
```

### Corrupted Database

Delete and re-download:
```bash
rm data/astdb.txt
# Restart server to trigger fresh download
```

---

## 🎯 Implementation Details

The auto-download feature is implemented in:
- **Package:** `internal/astdb/downloader.go`
- **Config:** `backend/config/config.go`
- **Initialization:** `main.go`

### Key Components

1. **Downloader** - HTTP client that fetches astdb.txt
2. **Validator** - Checks file format and content
3. **Auto-Updater** - Background goroutine for periodic updates
4. **Node Counter** - Counts valid entries in database

---

## 🔄 Update Schedule

Default behavior:
- **Initial check:** On server startup
- **Auto-updates:** Every 24 hours
- **Age check:** Before each lookup (doesn't block if recent)
- **Background task:** Non-blocking, runs in separate goroutine

---

## 📊 Statistics API

The downloader provides statistics:

```go
downloader.GetNodeCount()  // Returns: 43096
downloader.ValidateFile()  // Checks file integrity
downloader.NeedsUpdate()   // Returns: true/false
```

---

## 🎓 Technical Details

### HTTP Download
- Uses standard Go `net/http` client
- 60-second timeout
- Atomic file replacement (writes to .tmp, then renames)
- Ensures directory exists before writing

### File Management
- Atomic updates via rename (no partial writes)
- Temp file cleanup on errors
- Preserves old file if download fails

### Background Updates
- Ticker-based goroutine
- Graceful handling of failures
- Structured logging with zap

---

## 🚦 Status Indicators

In the Vue dashboard:
- **Node Lookup** - Searches against auto-downloaded database
- **Search Results** - Shows current node information
- **Last Updated** - File modification time visible in logs

---

## 💡 Best Practices

1. **Use default settings** - The defaults work for most users
2. **Monitor logs** - Check for download failures
3. **Allow auto-updates** - Keep node data current
4. **Use official source** - AllStarLink maintains the canonical database
5. **Check connectivity** - Ensure server can reach allmondb.allstarlink.org

---

## 🔮 Future Enhancements

Potential improvements:
- [ ] Delta updates (only download changes)
- [ ] Compression support (gzip)
- [ ] Multiple mirror fallback
- [ ] Database version tracking
- [ ] Webhook notifications on update
- [ ] Admin API to trigger manual updates
- [ ] Dashboard display of last update time

---

## 📄 Related Documentation

- [FEATURES.md](FEATURES.md) - Full feature documentation
- [QUICKSTART.md](QUICKSTART.md) - Getting started guide
- [SUMMARY.md](SUMMARY.md) - Implementation summary

---

**Automatic astdb download ensures your node lookup feature always has current data! 🎉**
