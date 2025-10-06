# AllStar AMI Commands Reference

## Overview

This document catalogs the Asterisk Manager Interface (AMI) commands used by Supermon and needed for full AllStar node monitoring.

---

## Key AMI Commands from Supermon Analysis

### 1. RptStatus with XStat
**Purpose:** Get extended status of connected nodes

**Command:**
```
ACTION: RptStatus
COMMAND: XStat
NODE: 1999
ActionID: xstat12345
```

**Response Format:**
```
ActionID: xstat12345
Response: Success
Message: Command output follows

Conn: 2000 192.168.1.10 1 OUT 00:15:30 ESTABLISHED
Conn: 2001 192.168.1.11 1 IN 00:10:20 ESTABLISHED
Conn: 3000123 (no-ip) 1 IN 00:05:10 ESTABLISHED
LinkedNodes: T2000, R2001, C2002
Var: RPT_RXKEYED=1
Var: RPT_TXKEYED=0
--END COMMAND--
```

**Conn Line Format:**
- **Standard:** `NodeNum IP IsKeyed Direction Elapsed LinkStatus`
- **EchoLink:** `NodeNum (no-ip) IsKeyed Direction Elapsed LinkStatus`

**Fields:**
- `NodeNum` - Remote node number
- `IP` - IP address (or empty for EchoLink)
- `IsKeyed` - Currently keyed (1=yes, 0=no)
- `Direction` - IN (incoming) or OUT (outgoing)
- `Elapsed` - Time connected (HH:MM:SS)
- `LinkStatus` - ESTABLISHED, CONNECTING, etc.

**Special Variables:**
- `RPT_RXKEYED` - Local receiver keyed (COS detected)
- `RPT_TXKEYED` - Local transmitter keyed (PTT active)

---

### 2. RptStatus with SawStat
**Purpose:** Get keying history for connected nodes

**Command:**
```
ACTION: RptStatus
COMMAND: SawStat
NODE: 1999
ActionID: sawstat12345
```

**Response Format:**
```
ActionID: sawstat12345
Response: Success
Message: Command output follows

Conn: 2000 0 45 120
Conn: 2001 1 0 180
Conn: 2002 0 300 60
--END COMMAND--
```

**Conn Line Format:** `NodeNum IsKeyed SecsSinceKeyed SecsSinceUnkeyed`

**Fields:**
- `NodeNum` - Remote node number
- `IsKeyed` - Currently keyed (1=yes, 0=no)
- `SecsSinceKeyed` - Seconds since last key-up (or 0 if currently keyed)
- `SecsSinceUnkeyed` - Seconds since last key-down

**Usage:**
- Determines last heard time
- Shows currently active transmissions
- Used for "Never" detection (large values = never keyed)

---

### 3. Generic CLI Commands
**Purpose:** Execute arbitrary Asterisk CLI commands

**Command:**
```
ACTION: COMMAND
COMMAND: rpt stats 1999
ActionID: cmd12345
```

**Common Commands:**
- `rpt stats <node>` - Get detailed RPT statistics
- `rpt lstats <node>` - Get link statistics
- `rpt nodes <node>` - List connected nodes
- `database show` - Show Asterisk database
- `core show channels` - Show active channels

---

### 4. Voter Commands (if app_voter loaded)
**Purpose:** Get RTCM receiver data

**Commands to try:**
```
ACTION: COMMAND
COMMAND: voter show <node>
ActionID: voter12345
```

Alternative formats:
```
rpt voter <node>
rpt localplay <node> /path/to/voter/status
```

**Expected Response:**
```
Receiver    RSSI  Voted  State
--------    ----  -----  -----
RX1         145   YES    ACTIVE
RX2         120   NO     ACTIVE
RX3         85    NO     STANDBY
```

---

## Key Parsing Patterns from Supermon

### 1. Extract Conn Lines

**Pattern:** `/Conn: (.*)/`

**Example:**
```php
foreach ($lines as $line) {
    if (preg_match('/Conn: (.*)/', $line, $matches)) {
        $arr = preg_split("/\s+/", trim($matches[1]));
        // $arr[0] = NodeNum
        // $arr[1] = IP (or empty)
        // $arr[2] = IsKeyed
        // ... etc
    }
}
```

---

### 2. Extract LinkedNodes

**Pattern:** `/LinkedNodes: (.*)/`

**Format:** `T2000, R2001, C2002`

**Modes:**
- `T` - Transceive (send/receive)
- `R` - Receive only
- `C` - Connecting
- `M` - Monitor

---

### 3. Extract Variables

**Pattern:** `/Var: RPT_RXKEYED=(.)/`

**Variables:**
- `RPT_RXKEYED` - Local COS (1=receiving, 0=idle)
- `RPT_TXKEYED` - Local PTT (1=transmitting, 0=idle)
- `RPT_ASEL` - Audio source selected
- `RPT_TELE` - Telemetry active

---

### 4. Node Type Detection

**EchoLink:** Node > 3000000
**IRLP:** Node 80000-89999
**AllStar:** Node < 80000 (typically < 60000)
**Web/Phone:** Special IP patterns

---

## Implementation Gaps in Current Code

### Missing Features

1. ✅ **Basic AMI connectivity** - Already implemented
2. ✅ **SendCommand** - Already implemented
3. ❌ **RptStatus action** - Need to add
4. ❌ **XStat/SawStat parsing** - Need to implement
5. ❌ **RPT variable extraction** - Need to add
6. ❌ **Voter command support** - Need to implement
7. ❌ **Node type detection** - Need to add helpers

---

## Enhanced AMI Helper Functions Needed

### 1. RptStatus Helper

```go
func (c *Connector) RptStatus(ctx context.Context, node, command string) (Message, error) {
    // Send: ACTION: RptStatus\r\nCOMMAND: XStat\r\nNODE: 1999\r\n
    // Parse response
}
```

### 2. XStat Parser

```go
type XStatResult struct {
    Connections []Connection
    LinkedNodes []LinkedNode
    RxKeyed     bool
    TxKeyed     bool
}

func ParseXStat(response string) (*XStatResult, error)
```

### 3. SawStat Parser

```go
type SawStatResult struct {
    Nodes map[int]*KeyingInfo
}

type KeyingInfo struct {
    IsKeyed          bool
    SecsSinceKeyed   int
    SecsSinceUnkeyed int
}

func ParseSawStat(response string) (*SawStatResult, error)
```

### 4. Voter Parser

```go
type VoterReceiver struct {
    ID     string
    RSSI   int
    Voted  bool
    State  string
    IP     string
}

func ParseVoterOutput(response string) ([]VoterReceiver, error)
```

---

## Supermon's Polling Strategy

### Main Loop (server.php)

1. **Connect to AMI** for each unique host
2. **Login** once per host
3. **Loop forever** (0.5 second interval):
   - Issue `XStat` for each node
   - Issue `SawStat` for each node
   - Parse responses
   - Combine data
   - Send SSE events to browser
   - Wait 0.5 seconds
   - Repeat

### Data Combination

1. **XStat** provides:
   - Connection list
   - IP addresses
   - Direction (IN/OUT)
   - Link modes
   - RX/TX keying status

2. **SawStat** provides:
   - Last heard times
   - Currently keyed nodes
   - Keying history

3. **Combined** gives:
   - Full node status
   - Sortable by last heard
   - TX indicators
   - Connection details

---

## Event Stream Format (SSE)

Supermon uses Server-Sent Events with two event types:

### nodes Event
**Frequency:** Only when data changes
**Content:** Full node status

```javascript
event: nodes
data: {"1999": {"node": "1999", "info": "...", "remote_nodes": [...]}}
```

### nodetimes Event
**Frequency:** Every cycle (0.5s)
**Content:** Just time fields

```javascript
event: nodetimes
data: {"1999": {"remote_nodes": [{"elapsed":"00:15:30","last_keyed":"00:01:15"}]}}
```

---

## Recommended Implementation Plan

### Phase 1: Enhanced AMI Commands ✅
- [x] Basic AMI connector (already done)
- [ ] Add RptStatus action support
- [ ] Add XStat/SawStat parsing
- [ ] Add voter command support

### Phase 2: State Management
- [ ] Track connection state per node
- [ ] Track keying state per link
- [ ] Calculate elapsed times
- [ ] Sort by last heard

### Phase 3: WebSocket Events
- [ ] Map AMI data to WebSocket messages
- [ ] Implement differential updates
- [ ] Add RX/TX indicators
- [ ] Send voter data

### Phase 4: Frontend Display
- [ ] Show COS/PTT indicators
- [ ] Display link modes (T/R/C)
- [ ] Show last heard times
- [ ] Voter receiver visualization

---

## Testing Without Real Hardware

### Mock AMI Responses

For development without AllStar hardware:

```go
// Mock XStat response
mockXStat := `ActionID: xstat123
Response: Success

Conn: 2000 192.168.1.10 0 OUT 00:15:30 ESTABLISHED
Conn: 2001 192.168.1.11 1 IN 00:10:20 ESTABLISHED
LinkedNodes: T2000, R2001
Var: RPT_RXKEYED=1
Var: RPT_TXKEYED=0
--END COMMAND--
`

// Mock SawStat response
mockSawStat := `ActionID: sawstat123
Response: Success

Conn: 2000 0 90 1800
Conn: 2001 1 0 600
--END COMMAND--
`
```

---

## References

- **Supermon server.php** - Main SSE server loop
- **amifunctions.inc** - AMI helper functions
- **nodeinfo.inc** - Node info lookup (EchoLink/IRLP)
- **AllStar Wiki** - https://wiki.allstarlink.org/
- **app_rpt Commands** - Asterisk RPT module documentation

---

## Next Steps

1. Enhance `internal/ami/connector.go` with RptStatus support
2. Create parsers for XStat/SawStat in new file `internal/ami/parsers.go`
3. Update `internal/core/state.go` to use enhanced AMI data
4. Add RX/TX keying to WebSocket events
5. Update Vue dashboard to show COS/PTT indicators

---

**This reference enables full feature parity with Supermon's AMI integration!**
