# Talker Log Persistence Fix

## Date: October 7, 2025

### Problem
When reloading the page, the talker log would show entries but without callsign or description information. The entries would appear as just node numbers (e.g., "594950 — START" instead of "W8CPT (594950) — START").

### Root Cause
The TalkerLog stores events at the time they're emitted. When events are created:
1. The `emitTalker()` function tries to enrich events with callsign/description from `LinksDetailed`
2. However, if a link hasn't been enriched with astdb data yet, the event is stored without that info
3. When the page reloads and fetches the snapshot, it receives the old un-enriched events
4. The stored events remain un-enriched even after astdb data becomes available

### Solution
Added dynamic enrichment to the `TalkerLogSnapshot()` method:

**File**: `internal/core/state.go`

**Changes**:
1. Modified `TalkerLogSnapshot()` to call a new `enrichTalkerSnapshot()` method
2. Created `enrichTalkerSnapshot()` which:
   - Takes the raw snapshot from the ring buffer
   - For each event that lacks callsign data:
     - First checks current `LinksDetailed` for enrichment
     - Falls back to direct `NodeLookup` service query
     - Handles text nodes (negative node IDs) specially
   - Returns fully enriched events with current callsign/description data

### Code Flow

#### Before:
```
TalkerLogSnapshot() -> log.Snapshot() -> [un-enriched old events]
```

#### After:
```
TalkerLogSnapshot() -> log.Snapshot() -> enrichTalkerSnapshot() -> [enriched events]
                                            ↓
                                       Check LinksDetailed
                                            ↓
                                       Check NodeLookup
                                            ↓
                                       Return with callsigns
```

### Benefits
1. **Persistent callsigns**: Events retain callsign info even after page reload
2. **No storage changes**: Uses existing TalkerLog ring buffer
3. **On-demand enrichment**: Only enriches when snapshot is requested
4. **Handles all node types**: Regular nodes, text nodes, and current links
5. **Skip already enriched**: Doesn't re-process events that already have callsign data

### Testing
After this fix:
1. ✅ Events shown in real-time have callsigns (existing behavior)
2. ✅ Page reload shows callsigns for historical events (NEW)
3. ✅ WebSocket reconnect preserves callsigns (NEW)
4. ✅ Server restart events get enriched from astdb (NEW)

### Performance
- Enrichment only happens when snapshot is requested (not on every event)
- Uses existing in-memory astdb cache
- Minimal overhead: single lookup per event without callsign
- Read lock only (doesn't block state updates)

---

## Related Files
- `internal/core/state.go` - Added `enrichTalkerSnapshot()` method
- `internal/core/talker.go` - TalkerEvent struct (unchanged)
- `internal/core/nodelookup.go` - NodeLookup service (unchanged)
- `backend/api/handlers.go` - TalkerLog endpoint (unchanged, uses new enriched snapshot)
