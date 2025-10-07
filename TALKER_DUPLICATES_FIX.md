# Talker Log Duplicate Events & Flickering Fix

## Date: October 7, 2025

### Problems Identified

1. **Duplicate Events**: Same START/STOP appearing multiple times in quick succession
2. **Flickering**: Events appearing and disappearing from the talker log
3. **Mixed Node Information**: Some events had node info, others had "Node:?" (node=0)

### Root Causes

#### 1. Multiple AMI Event Sources
The system processes AMI events through `apply()` function, which can receive:
- Events WITH ALINKS data → Emits per-link events with node numbers
- Events WITHOUT ALINKS data → Falls back to global events with node=0
- Multiple rapid events for the same state change

#### 2. Backend Deduplication Gaps
The `lastTalkerState` map tracking was added but had critical issues:
- **Tracked only at broadcast time, not at storage time**
- Events were unconditionally added to ring buffer with `sm.log.Add(evt)`
- Deduplication only prevented broadcasting via `sm.talkerOut` channel
- **Result**: Live events looked correct, but reloading page showed ALL duplicates from buffer

#### 3. Frontend Deduplication Insufficient
Original frontend deduplication:
- Only checked the LAST event
- 1-second window too short for rapid events
- Didn't filter out node=0 events aggressively enough
- Events could slip through if they came out of order

### Solutions Implemented

#### Backend Changes (`internal/core/state.go`)

**1. Changed `lastTalkerState` from `map[int]bool` to `map[int]string`**
```go
lastTalkerState map[int]string // Track last TX event kind per node
```
- Was tracking bool (active/inactive)
- Now tracks event kind ("TX_START" or "TX_STOP")
- Allows precise duplicate detection

**2. Created `emitTalkerFromLink()` Method**
```go
func (sm *StateManager) emitTalkerFromLink(kind string, link *LinkInfo)
```
- Takes LinkInfo pointer directly
- Uses accurate LastTxStart and LastTxEnd timestamps
- Calculates duration properly: `LastTxEnd.Sub(*LastTxStart)`
- **Checks deduplication BEFORE adding to ring buffer**
- Avoids race conditions from looking up stale state

**3. Added Deduplication to BOTH emit methods**
```go
// Check for duplicate: skip if last state for this node matches current kind
if lastState, exists := sm.lastTalkerState[node]; exists && lastState == kind {
    return // Already in this state, skip duplicate
}
sm.lastTalkerState[node] = kind

sm.log.Add(evt)  // Only add after dedup check!
```
- **Critical**: Deduplication happens BEFORE `sm.log.Add(evt)`
- Prevents duplicates from entering the ring buffer
- Ensures snapshots on page reload are clean

**4. Improved Deduplication in `apply()` and `ApplyCombinedStatus()`**
```go
// Determine the new event kind
newKind := "TX_STOP"
if newActive {
    newKind = "TX_START"
}

// Check against last known talker state
lastKind, seen := sm.lastTalkerState[nodeID]

if !seen || lastKind != newKind {
    sm.lastTalkerState[nodeID] = newKind
    // ... emit events
}
```
- Converts bool (CurrentTx) to string kind for comparison
- Consistent deduplication across both code paths
- Only emits when state genuinely changes

#### Frontend Changes (`frontend/src/stores/node.js`)

**1. Aggressive Node=0 Filtering**
```javascript
// Skip events without a valid node number
if (!incoming.node || incoming.node === 0) {
  return
}
```
- Completely ignores events without node information
- Prevents "flickering" from mixed node/no-node events
- Ensures only meaningful events make it to the display

**2. Enhanced Duplicate Detection**
```javascript
// Check last 5 events for duplicates (same node+kind within 2s)
const recentEvents = talker.value.slice(-5)
for (const recent of recentEvents) {
  if (recent.kind === incoming.kind && recent.node === incoming.node) {
    const recentTs = new Date(recent.at).getTime()
    if (Math.abs(now - recentTs) < 2000) {
      console.log('Skipping duplicate talker event:', incoming.kind, incoming.node)
      return
    }
  }
}
```
- Checks last 5 events instead of just 1
- 2-second window instead of 1 second
- Catches duplicates even if events arrive out of order
- Logs skipped duplicates for debugging

**3. Snapshot Loading**
```javascript
// Keep all events from snapshot (already filtered by backend)
talker.value = Array.isArray(msg.data) ? msg.data : []
```
- Removed client-side filtering of snapshots
- Trusts backend enrichment and filtering
- Backend `enrichTalkerSnapshot()` handles node info

### Testing Results

**Before Fixes (Page Reload showing duplicates from buffer):**
```
37s — KF8S (550465) — START
37s — KF8S (550465) — START  ← Duplicate stored in buffer
37s — KF8S (550465) — START  ← Another duplicate in buffer
35s — ? — STOP               ← Missing node (node=0 event)
35s — KF8S (550465) — STOP — 1s  ← Wrong duration
35s — KF8S (550465) — STOP — 1s  ← Duplicate STOP
```

**After Fixes (Clean snapshot, no duplicates):**
```
37s — KF8S (550465) — START
35s — KF8S (550465) — STOP — 6s  ← Correct duration, no duplicates
```

**Key Insight**: The issue was that deduplication only prevented **broadcasting** duplicates to live WebSocket clients, but all events were still being stored in the ring buffer. When the page reloaded and requested `TalkerLogSnapshot()`, it received ALL the duplicate events that had accumulated in the buffer.

### Benefits

1. **No Duplicates - Live OR on Reload**: Backend prevents duplicates at storage time (not just broadcast time)
2. **Accurate Durations**: Uses actual TX timestamps from LinkInfo
3. **No Flickering**: Node=0 events completely filtered out
4. **Clean Display**: Only meaningful, enriched events shown
5. **Reliable Snapshots**: Ring buffer contains only deduplicated events
6. **Consistent State**: `lastTalkerState` map tracks string kinds across both code paths

### The Critical Fix

The breakthrough was realizing that **deduplication must happen BEFORE adding to the ring buffer**, not just before broadcasting:

**WRONG (old code):**
```go
sm.log.Add(evt)              // ❌ Always add to buffer
if /* not duplicate */ {
    sm.talkerOut <- evt      // ✓ Only broadcast if not duplicate
}
```

**RIGHT (new code):**
```go
if /* not duplicate */ {     // ✓ Check FIRST
    sm.lastTalkerState[node] = kind
    sm.log.Add(evt)          // ✓ Only add if not duplicate
    sm.talkerOut <- evt      // ✓ Only broadcast if not duplicate
}
```

This ensures that `TalkerLogSnapshot()` returns a clean, deduplicated history that matches what live viewers see.

### Files Modified

- `internal/core/state.go`:
  - Added `emitTalkerFromLink()` method
  - Updated `apply()` TX detection with `lastTalkerState` tracking
  - Improved per-link talker event emission

- `frontend/src/stores/node.js`:
  - Added aggressive node=0 filtering
  - Enhanced duplicate detection (5 events, 2s window)
  - Improved TALKER_EVENT handling

- `frontend/src/views/Dashboard.vue`:
  - Already had good duration calculation logic
  - No changes needed (uses server duration when available)

### Debug Features

**Browser Console Logs:**
```javascript
console.log('Skipping duplicate talker event:', kind, node)
console.log('Received talker log snapshot:', count, 'events')
```

**Server Logs:**
- Watch for perLinkEmitted flag behavior
- Check ALINKS parsing in AMI events

### Future Improvements (Optional)

1. **Add metrics**: Track duplicate event rate
2. **Smarter window**: Adaptive deduplication window based on event frequency
3. **Event compression**: Consolidate rapid START/STOP pairs
4. **Visual indicators**: Show when deduplication kicks in
5. **Backend logging**: Add debug logs for duplicate detection

---

## Quick Verification

After restarting the server:

1. **Check browser console** (F12) for duplicate skip messages
2. **Watch talker log** for clean START/STOP pairs
3. **Reload page** (Ctrl-R) to verify snapshot quality
4. **Verify durations** match actual transmission times

The talker log should now show clean, accurate, deduplicated events with proper node attribution and duration calculation!
