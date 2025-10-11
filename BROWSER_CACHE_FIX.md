# Browser Cache Issue: Dashboard Still Shows Disconnected Nodes

## Problem

After database cleanup, dashboard still shows 8 connections (including K8JDS/529821) even though:
- ✅ Database only has 6 entries
- ✅ Server logs show 6 connections
- ✅ Polling reports 6 connections

## Root Cause

**Browser cache/state is stale**

When the database cleanup happened:
1. Server detected removals and emitted `LINK_REMOVED` events
2. Browser was already loaded and may have missed the WebSocket event
3. Browser's in-memory state still has the old 8 connections
4. Dashboard displays the stale browser state

## Verification

### Check Server State (Accurate)

```bash
# Check database
sqlite3 data/allstar.db "SELECT COUNT(*) FROM link_stats;"
# Should show: 6

# Check which nodes are in database
sqlite3 data/allstar.db "SELECT node FROM link_stats ORDER BY node;"
# Should NOT include 529821 (K8JDS)

# Check server logs
grep "POLLING.*connections" logs/*.log
# Should show: Node XXXX: 6 connections
```

### Check Browser State (Stale)

Open browser DevTools console (F12) and run:
```javascript
// Check WebSocket connection state
console.log('Links:', JSON.parse(localStorage.getItem('links') || '[]'))

// Or check the node store directly
console.log('Store links:', window.__VUE_DEVTOOLS_GLOBAL_HOOK__.apps[0]._instance.proxy.$store.state.links)
```

If browser shows 8 links but server shows 6, **browser cache is stale**.

## Solution

### Quick Fix: Hard Refresh Browser

**Windows/Linux:**
- `Ctrl + Shift + R`
- OR `Ctrl + F5`

**Mac:**
- `Cmd + Shift + R`

**OR use DevTools:**
1. Open DevTools (F12)
2. Right-click the refresh button
3. Select "Empty Cache and Hard Reload"

### Alternative: Clear Browser State

```javascript
// In browser console
localStorage.clear()
sessionStorage.clear()
location.reload()
```

### Last Resort: Close and Reopen Tab

Simply close the dashboard tab and open a new one. The WebSocket will reconnect and receive the current state from the server.

## Why This Happens

### WebSocket Event Timing

When the server starts and cleans up the database:

```
Time 0s:  Server starts, loads 8 links from database
Time 1s:  Browser connects, receives initial state (8 links)
Time 5s:  First poll runs, detects 2 disconnected nodes
Time 5s:  Server emits LINK_REMOVED for 529821, 595570
Time 5s:  ❌ Browser may miss the event if connection briefly dropped
Time 6s:  Browser still shows 8 links (stale)
```

### Browser Caching

The frontend may cache link state in:
1. **Component state** (Vue reactive data)
2. **Vuex/Pinia store** (in-memory state)
3. **LocalStorage** (persistent cache)

A hard refresh clears all of these and fetches fresh state from the server.

## Prevention

### For Users

**Always hard refresh after server restart:**
```bash
# After restarting server
# Hard refresh browser: Ctrl+Shift+R (or Cmd+Shift+R on Mac)
```

### For Developers

To prevent this in the future, add a **state version** to force cache invalidation:

**Backend (main.go):**
```go
// Add version to initial WebSocket message
stateVersion := time.Now().Unix() // Changes on server restart

// Send in STATUS_UPDATE:
{
  "messageType": "STATUS_UPDATE",
  "data": {
    "state_version": stateVersion,
    "links": [...]
  }
}
```

**Frontend (node.js store):**
```javascript
// Check state version and reload if stale
if (msg.data.state_version > currentStateVersion) {
  // Clear stale state and reload
  links.value = []
  currentStateVersion = msg.data.state_version
}
```

## Testing

### Verify Cleanup Worked

```bash
# 1. Check database before restart
sqlite3 data/allstar.db "SELECT COUNT(*) FROM link_stats;"
# Shows: 8

# 2. Restart server with new build
./allstar-nexus

# 3. Wait 5 seconds for first poll

# 4. Check database after cleanup
sqlite3 data/allstar.db "SELECT COUNT(*) FROM link_stats;"
# Shows: 6 ✅

# 5. Check logs
grep "cleaned up stale link stats" logs/*.log
# Should show: cleaned up stale link stats from database (deleted_count=2)
```

### Verify Browser Shows Correct Count

1. **Hard refresh browser** (`Ctrl+Shift+R`)
2. Dashboard should now show **6 connections** ✅
3. K8JDS (529821) should **NOT** be in the list ✅

## Summary

The fix **is working correctly**:
- ✅ Database cleaned up (6 entries)
- ✅ Server state accurate (6 connections)
- ✅ LINK_REMOVED events emitted

The issue is **browser cache needs to be refreshed**.

**Solution: Hard refresh the browser** (`Ctrl+Shift+R` or `Cmd+Shift+R`)

After hard refresh, dashboard will show the correct 6 connections! 🎉
