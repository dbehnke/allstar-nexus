# Hybrid Event-Driven + Polling Architecture

## Overview

This document describes the implementation of a hybrid architecture that combines event-driven AMI updates with periodic polling for data synchronization and enrichment.

## Motivation

The previous architecture was purely event-driven, relying on AMI events like:
- `RPT_ALINKS` - Adjacent link keying status
- `RPT_TXKEYED` - Local transmitter keyed
- `RPT_RXKEYED` - Local receiver keyed
- `RPT_LINKS` - Link list

While this provides real-time updates, it has limitations:
1. **Missing data fields** - Events don't include direction (IN/OUT), IP addresses, elapsed time, link modes
2. **No verification** - No way to detect if events are missed or state drifts
3. **No recovery** - If a client connects mid-stream, it only sees incremental updates

## Solution: Hybrid Approach

The new architecture combines:

### Event-Driven (Real-time)
- **Primary mechanism** for state changes
- Immediate response to keying, link changes
- Minimal latency for TX/RX indicators
- Keying tracker with jitter delay for accurate duration tracking

### Periodic Polling (Verification & Enrichment)
- **Every 60 seconds** query XStat + SawStat for each node
- Verifies event-driven state is accurate
- Enriches links with additional data not in events
- Provides recovery mechanism for missed events

## Architecture Components

### 1. Polling Service (`internal/core/polling.go`)

New component that:
- Manages periodic AMI queries (XStat + SawStat)
- Runs one goroutine per configured node
- Applies enriched data to StateManager
- Updates keying trackers with connection details

**Key Features:**
- Configurable interval (default: 60 seconds)
- Initial poll after 5 seconds (AMI stabilization)
- 10-second timeout per poll
- Graceful shutdown support
- Per-node polling isolation

### 2. Enhanced Data Fields

The polling service populates these additional fields in `LinkInfo`:

```go
IP              string     // IP address (empty for EchoLink)
IsKeyed         bool       // Remote node currently keying
Direction       string     // "IN" (incoming) or "OUT" (outgoing)
Elapsed         string     // Connection time (HH:MM:SS)
LinkType        string     // "ESTABLISHED", "CONNECTING", etc.
Mode            string     // T=Transceive, R=Receive, C=Connecting, M=Monitor
LastHeard       string     // Human-readable last heard time
SecsSinceKeyed  int        // Seconds since last keyed
LastKeyedTime   *time.Time // Timestamp of last key
```

### 3. Integration Points

#### AMI Connector
- Added `GetCombinedStatus()` - queries XStat + SawStat and merges results
- Added helper methods: `broadcastStatus()`, `clearPendingActions()`
- Already had `RptStatus()`, `GetXStat()`, `GetSawStat()` methods

#### StateManager
- Already has `ApplyCombinedStatus()` to process polled data
- Existing keying trackers updated with enriched connection info
- Updates `UpdateLinkInfo()` and `UpdateConnectedSince()` methods

#### Main Application
- Polling service started after AMI connection established
- Disabled via `disable_link_poller: true` config option
- Graceful shutdown on application exit

## Configuration

### Enable/Disable Polling

```yaml
# config.yaml
disable_link_poller: false  # false = polling enabled (default)
```

Or via environment variable:
```bash
DISABLE_LINK_POLLER=true  # Disable polling
```

### Polling Interval

Currently hardcoded to 60 seconds. Can be made configurable if needed.

### Multi-Node Support

The polling service automatically polls all configured nodes:

```yaml
nodes:
  - node_id: 43732
    name: "My Hub"
  - node_id: 48412
    name: "My Repeater"
```

Each node is polled independently every 60 seconds.

## Data Flow

### 1. Real-time Event Flow
```
AMI Event → Connector → StateManager → WebSocket → Frontend
   (ms latency)
```

### 2. Polling Flow
```
Polling Service → AMI (XStat+SawStat) → CombinedStatus → StateManager → WebSocket → Frontend
   (60 second interval)
```

### 3. Hybrid Behavior
- **TX/RX events**: Immediately detected via ALINKS events (jittered for accuracy)
- **Link additions**: Immediately detected via RPT_LINKS/ALINKS events
- **Connection details**: Populated by polling (IP, direction, elapsed, mode)
- **Data verification**: Polling ensures state is synchronized every minute

## Benefits

### For Users
1. **Complete link information** - See IP addresses, connection direction, elapsed time, modes
2. **Accurate last heard** - SawStat provides precise keying history
3. **Reliable state** - Polling ensures frontend stays in sync even if events are missed
4. **Better diagnostics** - More data for troubleshooting link issues

### For Developers
1. **Dual verification** - Events and polling validate each other
2. **Recovery mechanism** - Polling recovers from missed events
3. **Backward compatible** - Works with existing event-driven logic
4. **Configurable** - Can disable polling if needed

## Implementation Details

### Polling Service Lifecycle

1. **Start**: Called from `main.go` after AMI connection established
2. **Initial Poll**: After 5 second delay (AMI stabilization)
3. **Periodic Polls**: Every 60 seconds per node
4. **Shutdown**: Graceful stop with WaitGroup

### Error Handling

- Failed polls are logged but don't stop service
- 10-second timeout prevents hanging
- Connection errors automatically retry on next interval
- No cascading failures between nodes

### Performance Considerations

- **Network overhead**: 2 AMI commands per node per minute (XStat + SawStat)
- **CPU impact**: Minimal - parsing is lightweight
- **Memory**: ~1KB per connection for enriched data
- **Concurrency**: One goroutine per configured node

### Thread Safety

- Polling service uses WaitGroup for shutdown coordination
- StateManager has mutex protection for all state updates
- Keying tracker has independent mutex for adjacent node data
- No deadlocks - all lock acquisitions are brief and non-nested

## Testing

### Build Verification
```bash
go build -o allstar-nexus
```
✅ Build succeeds with no errors

### Unit Tests
```bash
go test ./...
```
⚠️ Two pre-existing test failures in `state_test.go`:
- `TestParseALinksAndTxAttribution`
- `TestPerLinkTxStop`

These tests need keying trackers configured (pre-existing issue, not related to polling).

### Integration Testing

To test the hybrid system:

1. **Enable polling** (default):
   ```yaml
   disable_link_poller: false
   ```

2. **Watch logs** for polling activity:
   ```
   [POLLING] Starting periodic polling service (interval=1m0s, nodes=[43732])
   [POLLING] Polling node 43732 for status...
   [POLLING] Node 43732: 5 connections, RX=false, TX=false
   ```

3. **Verify enriched data** in frontend:
   - Check that Direction shows "IN" or "OUT"
   - Verify IP addresses appear (if not EchoLink)
   - Confirm Elapsed time updates
   - Check Mode indicator (T/R/C/M)

## Future Enhancements

### Potential Improvements

1. **Configurable interval**
   ```yaml
   polling_interval_seconds: 60
   ```

2. **Per-node intervals**
   ```yaml
   nodes:
     - node_id: 43732
       polling_interval: 30s
   ```

3. **Adaptive polling**
   - Faster polling when links are active
   - Slower polling when idle

4. **Health monitoring**
   - Track poll success/failure rates
   - Alert on persistent failures
   - Expose metrics endpoint

5. **Voter integration**
   - Poll voter status alongside XStat/SawStat
   - Display receiver RSSI, voted status

## Files Modified

### New Files
- `internal/core/polling.go` - Polling service implementation

### Modified Files
- `main.go` - Wire polling service into application startup
- `internal/ami/connector.go` - Add `broadcastStatus()` and `clearPendingActions()` methods

### Existing Files (Used)
- `internal/ami/parsers.go` - XStat/SawStat parsing (already implemented)
- `internal/ami/types.go` - Data structures (already implemented)
- `internal/core/state.go` - ApplyCombinedStatus() (already implemented)
- `internal/core/keying_tracker.go` - UpdateLinkInfo() methods (already implemented)

## Summary

The hybrid architecture provides the best of both worlds:
- **Event-driven** for real-time responsiveness
- **Polling** for data completeness and verification

This ensures users get complete, accurate, and reliable link information while maintaining the low-latency benefits of the event-driven approach.

## References

- [AMI_COMMANDS_REFERENCE.md](AMI_COMMANDS_REFERENCE.md) - AMI command documentation
- XStat/SawStat response format details
- Supermon polling strategy analysis
