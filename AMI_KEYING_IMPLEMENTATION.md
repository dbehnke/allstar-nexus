# AMI Keying Tracker Implementation Summary

## Overview

This document summarizes the implementation of the jitter-compensated AMI keying tracker system based on the AMI events analysis. The system provides accurate, per-source-node tracking of adjacent link keying activity with 2-second jitter compensation.

## Files Created

### Backend (Go)

1. **`internal/core/keying_tracker.go`** (NEW)
   - Implements the `KeyingTracker` struct with 2-second delay timer queue
   - Manages `AdjacentNodeStatus` tracking for each adjacent link
   - Processes `RPT_ALINKS` events with keying state
   - Implements timer queue for jitter compensation
   - Callbacks for TX_START and TX_END events

### Frontend (Vue)

2. **`frontend/src/components/SourceNodeCard.vue`** (NEW)
   - Displays source node information with TX/RX indicators
   - Shows adjacent links table with:
     - Node ID (linked to stats.allstarlink.org)
     - Callsign (linked to QRZ.com)
     - Keying status (Keyed/Idle)
     - Connected time
     - TX state with pulsing animation
     - Duration (live timer or total)

### Documentation

3. **`AMI_KEYING_TRACKER.md`** (NEW)
   - Comprehensive documentation of the keying tracker system
   - Mermaid flowchart of state machine logic
   - Data structure definitions
   - WebSocket event formats
   - Frontend display specifications

## Files Modified

### Backend

1. **`internal/core/state.go`**
   - Added `keyingTrackers map[int]*KeyingTracker` to StateManager
   - Added `keyingOut chan SourceNodeKeyingUpdate` for WebSocket events
   - Added `numLinks` and `numALinks` tracking from RPT_NUMLINKS/RPT_NUMALINKS
   - Added `SourceNodeKeyingUpdate` struct for WebSocket messages
   - Integrated keying tracker processing in `apply()` method
   - Added methods:
     - `AddSourceNode()` - Initialize keying tracker for a source node
     - `emitKeyingUpdate()` - Emit keying state updates
     - `GetSourceNodes()` - Get list of configured source nodes
     - `GetSourceNodeSnapshot()` - Get snapshot of source node state
     - `KeyingUpdates()` - Channel accessor for keying updates
   - Enhanced NodeState with `NumLinks` and `NumALinks` fields

2. **`internal/web/ws.go`**
   - Added `SourceNodeKeyingLoop()` to broadcast SOURCE_NODE_KEYING events

3. **`main.go`**
   - Initialize keying trackers for all configured source nodes
   - Start `SourceNodeKeyingLoop()` goroutine

### Frontend

4. **`frontend/src/stores/node.js`**
   - Added `sourceNodes` Map to track source node keying states
   - Added handler for `SOURCE_NODE_KEYING` WebSocket message type
   - Stores adjacent node status per source node

5. **`frontend/src/views/Dashboard.vue`**
   - Added link count indicators for NUM_LINKS and NUM_ALINKS
   - Added source node cards display
   - Imported `SourceNodeCard` component
   - Added CSS for link indicators

## Key Features

### Jitter Compensation

- **2-second delay timer** prevents false unkey events from network glitches
- Timer queue manages pending unkey confirmations
- Re-keying within 2 seconds cancels the unkey timer

### Per-Source-Node Tracking

- Separate keying tracker for each configured source node
- Multi-node support via config file
- Each source node independently tracks its adjacent links

### Real-time Updates

- WebSocket events (`SOURCE_NODE_KEYING`) provide instant updates
- Live transmission timer in frontend
- Pulsing TX indicator during active transmission

### Accurate Duration Tracking

- Sub-second precision for transmission duration
- Total TX seconds accumulated per adjacent node
- Persistent stats (survives server restarts via existing persistence layer)

## Configuration

Source nodes are configured in `config.yaml`:

```yaml
# Simple format - node IDs only
nodes: [594950, 595123]

# Advanced format - with custom names
nodes:
  - node_id: 594950
    name: "Main Repeater"
  - node_id: 595123
    name: "Remote Base"
```

## AMI Event Flow

1. **RPT_ALINKS Event Received**
   - Parse adjacent node IDs and keying status (K/U flags)
   - Update RPT_NUMALINKS count

2. **For Each Source Node Tracker**
   - Process timer queue (check for expired unkey confirmations)
   - Process each adjacent node:
     - **TX START**: Link keyed + not transmitting → Start tracking, emit TX_START
     - **Potential STOP**: Link unkeyed + transmitting → Schedule 2s timer
     - **Ongoing TX**: Link keyed + transmitting → Cancel timers (was jitter)
     - **Confirmed STOP**: Timer expired + still unkeyed → Calculate duration, emit TX_END
   - Enrich with node lookup data (callsign, description)
   - Emit `SourceNodeKeyingUpdate` via WebSocket

3. **Frontend Updates**
   - Receive `SOURCE_NODE_KEYING` event
   - Update sourceNodes Map
   - Re-render source node cards with new data
   - Live timers update every second via `nowTick`

## WebSocket Message Types

### New Message Type

**`SOURCE_NODE_KEYING`** - Source node keying state update
```json
{
  "messageType": "SOURCE_NODE_KEYING",
  "data": {
    "source_node_id": 594950,
    "adjacent_nodes": {
      "634021": { /* AdjacentNodeStatus */ }
    },
    "tx_keyed": true,
    "rx_keyed": false,
    "timestamp": "2025-10-08T23:45:12Z"
  },
  "timestamp": 1696807512000
}
```

### Enhanced STATUS_UPDATE

Now includes:
- `num_links` - Global link count (RPT_NUMLINKS)
- `num_alinks` - Adjacent link count (RPT_NUMALINKS)

## UI Components

### Link Count Indicators

Two prominent indicators at the top of the dashboard:
- **Global Links** - Total network size
- **Adjacent Links** - Direct connections

### Source Node Cards

One card per configured source node showing:
- **Header**: Source node ID, TX/RX indicators
- **Table**: Adjacent links with columns:
  - Node ID (clickable)
  - Callsign (clickable to QRZ)
  - Status (Keyed/Idle badge)
  - Connected time
  - TX state (pulsing indicator)
  - Duration (live or total)

## Benefits

1. ✅ **Accurate source identification** - Know exactly which adjacent node is transmitting
2. ✅ **Jitter-proof** - 2-second delay eliminates false events
3. ✅ **Multi-node support** - Track multiple source nodes independently
4. ✅ **Real-time monitoring** - Live updates via WebSocket
5. ✅ **Visual clarity** - Dedicated cards per source node
6. ✅ **Network visibility** - Global vs. adjacent link counts
7. ✅ **Duration tracking** - Accurate TX time measurement
8. ✅ **Persistent stats** - Totals survive restarts

## Testing Checklist

- [ ] Verify keying trackers initialize for all configured nodes
- [ ] Test TX START detection (node keys up)
- [ ] Test TX STOP with jitter (brief unkey followed by re-key within 2s)
- [ ] Test confirmed TX STOP (unkey persists for 2+ seconds)
- [ ] Verify duration calculations are accurate
- [ ] Check WebSocket events are emitted correctly
- [ ] Test frontend source node cards display properly
- [ ] Verify link count indicators update
- [ ] Test multi-node configuration
- [ ] Verify node lookup enrichment (callsign, description)

## Future Enhancements

- [ ] Add RX_KEYED/TX_KEYED tracking per source node (currently TODO in code)
- [ ] Add historical charts for TX activity
- [ ] Add configurable jitter delay (currently hardcoded to 2000ms)
- [ ] Add alerts for extended transmissions
- [ ] Add export functionality for TX logs
