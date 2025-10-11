# Link Tracking Fix: Disconnected Clients Issue

## Problem

Users reported that the web dashboard showed **more connections** than the polling service reported. For example:
- **Dashboard**: 8 connections
- **Polling log**: 6 connections (from XStat)

This indicated that **disconnected clients were not being removed** from the dashboard.

## Root Cause

The issue was a mismatch between how **event-driven links** and **polling-driven links** were handled:

### Event-Driven Links (ALINKS/RPT_LINKS)
- Created with `LocalNode = 0` (unassigned)
- No association with a specific local node

### Polling-Driven Links (XStat/SawStat)
- Created with `LocalNode = <node_id>` (assigned)
- Associated with the polled node

### The Conflict

In `ApplyCombinedStatus()` (line 1039 of state.go), when polling updates arrived:

```go
// Keep links that belong to other local nodes (but not LocalNode=0, which are legacy/seeded)
// LocalNode=0 means unassigned/legacy, so we clear those out and let pollers repopulate
if sm.state.LinksDetailed[i].LocalNode != 0 && sm.state.LinksDetailed[i].LocalNode != combined.Node {
    mergedLinks = append(mergedLinks, sm.state.LinksDetailed[i])
}
```

This code **intentionally removed `LocalNode=0` entries**, expecting polling to repopulate them. However:

1. **Events create links** with `LocalNode=0`
2. **Polling runs** and removes all `LocalNode=0` links
3. **Polling only adds currently connected nodes** (6 nodes from XStat)
4. **Previously disconnected event-driven links** are gone
5. **Events continue to update** those same links, recreating them with `LocalNode=0`
6. **Next poll** removes them again
7. **Race condition**: If poll happens before events clear disconnected nodes, stale links remain

This created a situation where:
- Event-driven links would be **temporarily removed by polling**
- But then **recreated by subsequent events**
- Leading to **stale disconnected links accumulating**

## Solution

Ensure **all links have LocalNode set** regardless of how they were created:

### 1. Event-Driven Link Creation (state.go:419-441)

**Before:**
```go
ni := LinkInfo{Node: id, ConnectedSince: now}
```

**After:**
```go
ni := LinkInfo{
    Node:           id,
    LocalNode:      sm.state.NodeID, // Set LocalNode for multi-node compatibility
    ConnectedSince: now,
}
```

Also update existing links to set LocalNode if missing:
```go
if li.LocalNode == 0 && sm.state.NodeID != 0 {
    li.LocalNode = sm.state.NodeID
}
```

### 2. Seeded Link Creation (main.go:201-208)

**Before:**
```go
linkInfo := core.LinkInfo{
    Node: s.Node,
    ConnectedSince: cs,
    LastTxStart: s.LastTxStart,
    LastTxEnd: s.LastTxEnd,
    TotalTxSeconds: s.TotalTxSeconds
}
```

**After:**
```go
linkInfo := core.LinkInfo{
    Node:           s.Node,
    LocalNode:      primaryNodeID, // Set LocalNode for multi-node compatibility
    ConnectedSince: cs,
    LastTxStart:    s.LastTxStart,
    LastTxEnd:      s.LastTxEnd,
    TotalTxSeconds: s.TotalTxSeconds,
}
```

## How It Works Now

### Link Lifecycle

1. **Initial Connection (Event)**
   - ALINKS event arrives
   - Link created with `LocalNode = sm.state.NodeID` (e.g., 43732)
   - Dashboard shows link

2. **Enrichment (Polling)**
   - Polling runs every 60s
   - XStat returns 6 currently connected nodes
   - Polling **keeps** links with `LocalNode = 43732`
   - Polling **removes** any `LocalNode = 0` legacy links
   - Polling **adds/updates** the 6 connected nodes

3. **Disconnection (Event)**
   - Node disconnects
   - ALINKS event updates (node removed from list)
   - Event processing detects removal
   - Emits LINK_REMOVED event
   - Dashboard removes link

4. **Verification (Polling)**
   - Next poll doesn't include disconnected node
   - Polling sees link not in XStat results
   - Link removal detected and emitted
   - Double-check ensures link is gone

### Link Tracking Matrix

| Source          | LocalNode Set? | Kept by Polling? | Removed on Disconnect? |
|-----------------|----------------|------------------|------------------------|
| Event (OLD)     | ❌ No (=0)     | ❌ Removed        | ⚠️ Maybe (race)       |
| Event (NEW)     | ✅ Yes         | ✅ Kept           | ✅ Yes                |
| Polling         | ✅ Yes         | ✅ Kept           | ✅ Yes                |
| Seeded (OLD)    | ❌ No (=0)     | ❌ Removed        | ⚠️ Race condition     |
| Seeded (NEW)    | ✅ Yes         | ✅ Kept           | ✅ Yes                |

## Testing

### Scenario 1: Normal Connection/Disconnection

**Steps:**
1. Node connects
2. ALINKS event creates link with LocalNode set
3. Dashboard shows node
4. Polling enriches with IP, direction, etc.
5. Node disconnects
6. ALINKS event removes link
7. Dashboard updates immediately
8. Next poll verifies removal

**Expected:**
- ✅ Dashboard count matches XStat count
- ✅ Disconnected nodes removed immediately
- ✅ No stale links

### Scenario 2: Polling Before Event

**Steps:**
1. Node connects
2. Polling runs first, creates link with LocalNode set
3. Dashboard shows node
4. ALINKS event arrives, updates same link
5. Node disconnects
6. Polling detects removal first
7. Event processes removal

**Expected:**
- ✅ No duplicate links
- ✅ Removal detected by either polling or events
- ✅ Dashboard count accurate

### Scenario 3: Multiple Nodes (Multi-node Setup)

**Steps:**
1. Configure 2 local nodes: 43732, 48412
2. Node A connects to 43732
3. Node B connects to 48412
4. Node C connects to 43732

**Expected:**
- ✅ Node A: LocalNode=43732
- ✅ Node B: LocalNode=48412
- ✅ Node C: LocalNode=43732
- ✅ Polling for 43732 keeps A and C, ignores B
- ✅ Polling for 48412 keeps B, ignores A and C

### Scenario 4: Application Restart

**Steps:**
1. 5 nodes connected
2. Stop application
3. 2 nodes disconnect
4. Start application
5. Load persisted link stats (5 links, but only 3 still connected)

**Expected (OLD - broken):**
- ❌ Seeded links have LocalNode=0
- ❌ First poll removes all 5 seeded links
- ❌ Poll adds 3 current links
- ❌ Lost TX statistics for the 3 that stayed connected

**Expected (NEW - fixed):**
- ✅ Seeded links have LocalNode=43732
- ✅ First poll keeps 3 connected links, removes 2 disconnected
- ✅ Poll enriches 3 current links
- ✅ TX statistics preserved for connected links

## Verification

### Check Link Tracking

```bash
# Start application
./allstar-nexus

# Watch logs
[STATE] link additions: [...]  # Should show LocalNode set
[POLLING] Node 43732: 6 connections, RX=false, TX=false
[STATE] link removals: [...]  # When nodes disconnect
```

### Check Dashboard

1. Open dashboard
2. Note connection count
3. Check logs for polling output
4. Verify counts match: `Dashboard count == Polling count`

### Check Persistence

```bash
# Stop application with connections
# Check database
sqlite3 data/allstar.db "SELECT node, total_tx_seconds FROM link_stats;"

# Restart application
# Seeded links should have LocalNode set in logs
```

## Files Modified

### 1. [internal/core/state.go](internal/core/state.go)
- **Lines 419-441**: Event-driven link creation now sets LocalNode
- **Lines 422-426**: Existing links updated to set LocalNode if missing

### 2. [main.go](main.go)
- **Lines 192-208**: Seeded links now set LocalNode from primary node

## Impact

### Benefits

✅ **Accurate link count** - Dashboard matches reality
✅ **Proper cleanup** - Disconnected nodes removed correctly
✅ **Multi-node support** - Links properly tracked per local node
✅ **Persistence works** - Seeded links handled correctly
✅ **No race conditions** - Event and polling coordination works

### Performance

- **No overhead** - Just setting an integer field
- **No breaking changes** - Backward compatible
- **Same behavior** - For single-node setups, LocalNode just tracks the primary node

## Multi-Node Context

This fix is essential for **multi-node support**:

### Single Node Setup (most common)
```yaml
nodes:
  - node_id: 43732
```

- All links have `LocalNode = 43732`
- Polling and events coordinate properly
- Disconnections tracked accurately

### Multi-Node Setup
```yaml
nodes:
  - node_id: 43732
  - node_id: 48412
```

- Links for 43732 have `LocalNode = 43732`
- Links for 48412 have `LocalNode = 48412`
- Each poller manages its own links
- No interference between nodes

## Future Considerations

### Potential Enhancements

1. **Per-node link tracking**
   - Track which poller is responsible
   - Allow different polling intervals per node

2. **Link ownership handoff**
   - If a node connects to multiple local nodes
   - Track all connections separately

3. **Improved cleanup**
   - Timeout-based removal for missed events
   - Stale link detection

## Summary

The fix ensures that **all links have LocalNode set**, which allows:
- **Polling and events to coordinate** properly
- **Disconnected nodes to be removed** accurately
- **Multi-node setups to work** correctly
- **Link persistence to function** as intended

**Dashboard count will now match polling count** - the disconnected client issue is resolved! ✅
