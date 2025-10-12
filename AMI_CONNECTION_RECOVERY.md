# AMI Connection Recovery & Resilience

## Overview

The AMI (Asterisk Manager Interface) connector has been enhanced with robust connection recovery, exponential backoff, and comprehensive status monitoring. The application **never exits** on AMI connection failures - it automatically reconnects indefinitely.

## Key Features

### 1. **Automatic Reconnection with Exponential Backoff**

The connector automatically attempts to reconnect when the connection is lost, using exponential backoff to avoid overwhelming the server:

- **Initial backoff**: 15 seconds (configurable via `ami_retry_interval`)
- **Maximum backoff**: 60 seconds (configurable via `ami_retry_max`)
- **Backoff strategy**: Doubles with each failed attempt until reaching the maximum
- **Infinite retries**: Never gives up - continues attempting reconnection indefinitely

### 2. **Connection State Tracking**

The connector maintains accurate connection state:

- `IsConnected()` - Returns current connection status
- `ConnectionStatusChan()` - Broadcasts connection state changes
- Connection status includes timestamp and error information

### 3. **Graceful Degradation**

When AMI is disconnected:

- **Event-driven updates**: Paused (no events received)
- **Polling service**: Automatically skips polls and waits for reconnection
- **WebSocket clients**: Continue receiving heartbeats with last known state
- **Application**: Continues running normally, waiting for AMI to recover

### 4. **Enhanced Logging**

Detailed logging at every stage:

```
[AMI] attempting initial connection to 127.0.0.1:5038
[AMI] connected to 127.0.0.1:5038
[AMI] login payload sent to 127.0.0.1:5038
AMI connection established

[AMI] connection failed (attempt #2): dial tcp 127.0.0.1:5038: connect: connection refused
[AMI] retrying connection in 30s...
[AMI] reconnection attempt #3 to 127.0.0.1:5038 (backoff: 30s)
```

## Configuration

### AMI Connection Settings

```yaml
# config.yaml
ami_enabled: true
ami_host: 127.0.0.1
ami_port: 5038
ami_username: admin
ami_password: change-me
ami_events: "on"
ami_retry_interval: 15s   # Initial backoff
ami_retry_max: 60s        # Maximum backoff
```

Or via environment variables:
```bash
AMI_ENABLED=true
AMI_HOST=127.0.0.1
AMI_PORT=5038
AMI_USERNAME=admin
AMI_PASSWORD=change-me
AMI_EVENTS=on
AMI_RETRY_INTERVAL=15s
AMI_RETRY_MAX=60s
```

### Backoff Calculation

| Attempt | Backoff Time | Cumulative Wait |
|---------|--------------|-----------------|
| 1       | 0s (immediate) | 0s           |
| 2       | 15s          | 15s             |
| 3       | 30s          | 45s             |
| 4       | 60s (max)    | 1m 45s          |
| 5       | 60s (max)    | 2m 45s          |
| ...     | 60s (max)    | ...             |

## Architecture

### Connection Lifecycle

```
┌─────────────────────────────────────────────────┐
│          AMI Connection Lifecycle                │
├─────────────────────────────────────────────────┤
│                                                  │
│  1. Start()                                      │
│     └─> Spawn connection loop goroutine         │
│                                                  │
│  2. Connection Loop                              │
│     ├─> Attempt TCP connection                  │
│     ├─> Send login                               │
│     ├─> Broadcast "connected" status            │
│     ├─> Read frames until disconnect            │
│     └─> On error:                                │
│         ├─> Broadcast "disconnected" status     │
│         ├─> Clear pending actions               │
│         ├─> Wait with exponential backoff       │
│         └─> Retry (back to step 2)              │
│                                                  │
│  3. Graceful Shutdown                            │
│     ├─> Cancel context                           │
│     ├─> Connection loop exits                    │
│     └─> Broadcast "closed" status               │
│                                                  │
└─────────────────────────────────────────────────┘
```

### Status Broadcasting

The connector broadcasts connection status changes via a buffered channel:

```go
type ConnectionStatus struct {
    Connected bool
    Timestamp time.Time
    Error     error
}
```

Consumers (like the polling service and main application) can monitor this channel:

```go
for status := range conn.ConnectionStatusChan() {
    if status.Connected {
        log.Printf("AMI connected at %s", status.Timestamp)
    } else {
        log.Printf("AMI disconnected: %v", status.Error)
    }
}
```

## Failure Scenarios & Recovery

### Scenario 1: Asterisk Not Running

**Symptoms:**
```
[AMI] attempting initial connection to 127.0.0.1:5038
[AMI] connection failed (attempt #1): dial tcp 127.0.0.1:5038: connect: connection refused
[AMI] retrying connection in 15s...
```

**Behavior:**
- Application starts normally
- AMI connector retries every 15s, 30s, 60s, ...
- Polling service skips polls: `[POLLING] Skipping poll (AMI not connected)`
- When Asterisk starts, connection automatically established

**Recovery:**
- Automatic - as soon as Asterisk is available
- No manual intervention required

### Scenario 2: Network Interruption During Operation

**Symptoms:**
```
[AMI] connection failed (attempt #5): read tcp 127.0.0.1:5038: i/o timeout
AMI connection lost
[AMI] retrying connection in 60s...
```

**Behavior:**
- Event stream stops
- Polling service pauses
- WebSocket clients see last known state
- Connector continues retry attempts

**Recovery:**
- Automatic reconnection when network restored
- Events resume immediately
- Polling resumes on next interval

### Scenario 3: Invalid Credentials

**Symptoms:**
```
[AMI] attempting initial connection to 127.0.0.1:5038
[AMI] connected to 127.0.0.1:5038
[AMI] login payload sent
[AMI] connection failed (attempt #1): authentication failed
[AMI] retrying connection in 15s...
```

**Behavior:**
- Connector retries indefinitely
- Log clearly shows authentication failure
- Application continues running

**Recovery:**
- Update credentials in config
- Restart application
- Or: Fix AMI credentials in Asterisk and wait for next retry

### Scenario 4: Asterisk Restart

**Symptoms:**
```
AMI connection lost
[AMI] retrying connection in 30s...
[AMI] reconnection attempt #2 to 127.0.0.1:5038 (backoff: 30s)
AMI connection established
```

**Behavior:**
- Graceful disconnect detected
- Automatic reconnection within backoff period
- Seamless resume of operations

**Recovery:**
- Fully automatic
- Typical downtime: 15-60 seconds depending on when disconnect detected

## Polling Service Integration

The polling service is aware of AMI connection state:

```go
func (ps *PollingService) performPoll(nodeID int) {
    // Check if AMI is connected before polling
    if !ps.connector.IsConnected() {
        log.Printf("[POLLING] Skipping poll for node %d (AMI not connected, waiting for reconnection...)", nodeID)
        return
    }

    // Proceed with poll...
}
```

**Benefits:**
- No wasted poll attempts when disconnected
- Clear logging of skipped polls
- Automatic resume when connection restored
- No errors from attempting to poll disconnected AMI

## Monitoring & Observability

### Log Patterns to Monitor

**Normal Operation:**
```
[AMI] attempting initial connection to 127.0.0.1:5038
[AMI] connected to 127.0.0.1:5038
[AMI] login payload sent to 127.0.0.1:5038
AMI connection established
```

**Connection Issues:**
```
[AMI] connection failed (attempt #N): <error>
[AMI] retrying connection in Xs...
```

**Recovery:**
```
[AMI] reconnection attempt #N to 127.0.0.1:5038 (backoff: Xs)
AMI connection established
```

### Metrics to Track

1. **Connection uptime** - Time since last successful connection
2. **Reconnection attempts** - Number of failed attempts before success
3. **Backoff time** - Current retry backoff duration
4. **Poll skip rate** - Percentage of polls skipped due to disconnection

### Health Check API

The application provides health status at `/api/health`:

```json
{
  "status": "ok",
  "timestamp": "2025-10-10T21:50:00Z",
  "ami_connected": true
}
```

## Best Practices

### 1. **Configure Appropriate Backoff Times**

For local Asterisk (same machine):
```yaml
ami_retry_interval: 5s
ami_retry_max: 30s
```

For remote Asterisk (across network):
```yaml
ami_retry_interval: 15s
ami_retry_max: 60s
```

For unstable networks:
```yaml
ami_retry_interval: 30s
ami_retry_max: 120s
```

### 2. **Monitor Connection Stability**

- Set up alerts for frequent reconnections
- Log analysis to detect patterns
- Network diagnostics if reconnections are frequent

### 3. **Graceful Asterisk Restarts**

When restarting Asterisk:
1. Application detects disconnect
2. Waits with backoff
3. Reconnects automatically when Asterisk ready
4. No need to restart the application

### 4. **Testing Connection Recovery**

**Test 1: Asterisk restart**
```bash
# On Asterisk machine
sudo systemctl restart asterisk

# Watch application logs - should see reconnection
```

**Test 2: Firewall block**
```bash
# Block AMI port temporarily
sudo iptables -A INPUT -p tcp --dport 5038 -j DROP

# Wait for disconnect
# Unblock
sudo iptables -D INPUT -p tcp --dport 5038 -j DROP

# Should see automatic reconnection
```

**Test 3: Invalid credentials**
```yaml
# Set wrong password in config
ami_password: wrong-password

# Start application - will retry indefinitely
# Fix password and restart - connects successfully
```

## Troubleshooting

### Issue: "AMI connection lost" repeatedly

**Possible Causes:**
1. Network instability
2. Asterisk overloaded
3. Firewall issues
4. AMI timeout configuration

**Solutions:**
- Check Asterisk load (`asterisk -rx "core show uptime"`)
- Verify network connectivity
- Increase `ami_retry_max` for unstable networks
- Check Asterisk AMI logs: `/var/log/asterisk/manager.log`

### Issue: "authentication failed" on reconnect

**Possible Causes:**
1. AMI credentials changed
2. AMI user disabled in Asterisk
3. Permissions insufficient

**Solutions:**
- Verify `/etc/asterisk/manager.conf` has correct user/password
- Ensure AMI user has required permissions: `read=all,write=all`
- Restart Asterisk after config changes

### Issue: Polling fails even when AMI connected

**Possible Causes:**
1. Node not responding to RptStatus
2. app_rpt not loaded
3. Node number incorrect

**Solutions:**
- Check Asterisk modules: `asterisk -rx "module show like rpt"`
- Verify node number matches Asterisk config
- Test manually: `asterisk -rx "rpt xstat 43732"`

## Summary

The enhanced AMI connection recovery provides:

✅ **Automatic reconnection** with exponential backoff
✅ **Infinite retry attempts** - never gives up
✅ **Connection state tracking** for dependent services
✅ **Graceful degradation** when disconnected
✅ **Enhanced logging** for observability
✅ **No application crashes** on AMI failure

The application is now highly resilient to AMI connection issues and can recover from any transient or persistent connection failure without manual intervention.
