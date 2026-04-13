# Multi-Node Gamification & UI Design Spec

**Date:** 2026-04-13
**Author:** Sisyphus (via brainstorming session)
**Status:** v2 — updated after review
**Repo:** `github.com/dbehnke/allstar-nexus`

---

## Context

The Allstar Nexus backend and frontend were designed with multi-node support in mind, but three issues prevent it from working correctly in practice:

1. **Double-counting in gamification**: When two Allstar nodes are linked, the same physical transmission is independently logged by each node's `KeyingTracker`, producing multiple `TransmissionLog` rows with different `SourceID` values. `TallyService.ProcessTally()` aggregates ALL logs, causing XP inflation.

2. **No callsign exclusion**: Certain callsigns (e.g., weather stations, ARRL HQ) should be tracked but never score XP. There's no mechanism to exclude them.

3. **Broken multi-node UI display**: The frontend shows "No source nodes" or only one node card even when multiple nodes are configured. The backend sends per-node `SOURCE_NODE_KEYING` snapshots, but because one shared AMI connector feeds all trackers indiscriminately, all nodes' keying state gets mixed together. Additionally, there's no UI to show/hide individual node cards.

---

## Feature 1: Scoring Node Filter

### Goal
Prevent double-counting by designating ONE source node as the authoritative scorer for gamification.

### Design

**Config change** (`backend/config/config.go` — `GamificationConfig` struct):

```go
ScoringSourceNodeID int `mapstructure:"scoring_source_node_id" yaml:"scoring_source_node_id"`
// 0 = all nodes score (backward compatible default)
// >0 = only transmissions from this source node count toward XP/levels
```

User config:
```yaml
gamification:
  scoring_source_node_id: 594950  # Only node 594950's transmissions earn XP
```

**Validation**: At startup, if `scoring_source_node_id > 0` and it doesn't match any entry in `cfg.Nodes`, log a warning. Don't fail — allow it so operators can reconfigure nodes without restart.

**Implementation** (`backend/gamification/tally_service.go` — `ProcessTally()`):

After calling `GetLogsBetween()`, filter the returned map to only include logs whose `SourceID` matches `s.cfg.ScoringSourceNodeID`:

```go
if s.cfg.ScoringSourceNodeID > 0 {
    filtered := make(map[string][]models.TransmissionLog, len(transmissions))
    for callsign, logs := range transmissions {
        var kept []models.TransmissionLog
        for _, log := range logs {
            if log.SourceID == s.cfg.ScoringSourceNodeID {
                kept = append(kept, log)
            }
        }
        if len(kept) > 0 {
            filtered[callsign] = kept
        }
    }
    transmissions = filtered
}
```

**Why in-memory filter**: No schema changes, no new repository methods, surgical change, negligible performance impact for typical log volumes.

**TransmissionLog persistence**: Logs from non-scoring nodes are still persisted (to `transmission_logs` table). This preserves full tracking data for future rescore operations.

**Backward compatibility**: `scoring_source_node_id: 0` (default zero value) means all nodes score — identical to current behavior.

### Config Wiring Note

There are **two** config structs:
- `config.GamificationConfig` in `backend/config/config.go` — loaded from YAML
- `gamification.Config` in `backend/gamification/config.go` — used by `TallyService`

The YAML fields go into `config.GamificationConfig`. In `main.go`, a `gameCfg` of type `gamification.Config` is built from `cfg.Gamification`. **Both `ScoringSourceNodeID` and `ExcludedCallsigns` must be added to the `gamification.Config` struct AND wired in `main.go`'s `gameCfg` construction block.**

### Files Changed
- `backend/config/config.go` — add `ScoringSourceNodeID` field to `GamificationConfig`
- `backend/gamification/config.go` — add `ScoringSourceNodeID` and wire from `GamificationConfig` in `main.go`
- `backend/gamification/tally_service.go` — add filter in `ProcessTally()`

### Tests
- `tally_service_test.go`: Add test case with mixed `SourceID` logs and `scoring_source_node_id` set — verify only matching logs are scored

---

## Feature 2: Exclude Callsigns from Scoring

### Goal
Allow specific callsigns to be tracked (persisted to `transmission_logs`) but never earn XP or levels.

### Design

**Config change** (`backend/config/config.go` — `GamificationConfig` struct):

```go
ExcludedCallsigns []string `mapstructure:"excluded_callsigns" yaml:"excluded_callsigns"`
```

User config:
```yaml
gamification:
  excluded_callsigns:
    - "W1AW"       # ARRL HQ
    - "SKYWARN"    # storm spotters
    - "WX5NWS"     # weather service
```

**Normalization**: Callsigns in config should be stored uppercase. At tally time, normalize to uppercase for comparison. This is a UX simplification — users put uppercase in config, code normalizes at runtime.

**TallyService construction** — build a `map[string]struct{}` lookup for O(1) exclusion checks:

```go
type TallyService struct {
    // ...
    excludedCallsigns map[string]struct{}  // uppercase callsigns
}

func NewTallyService(cfg *GamificationConfig, ...) *TallyService {
    excluded := make(map[string]struct{})
    for _, cs := range cfg.ExcludedCallsigns {
        excluded[strings.ToUpper(cs)] = struct{}{}
    }
    return &TallyService{
        // ...
        excludedCallsigns: excluded,
    }
}
```

**Exclusion point** (`processGroup()` — top of loop):

```go
for callsign, txLogs := range transmissions {
    if callsign == "" {
        continue
    }
    if s.excludedCallsigns != nil {
        if _, ok := s.excludedCallsigns[strings.ToUpper(callsign)]; ok {
            continue  // skip XP calculation; log is still persisted
        }
    }
    // ... existing scoring logic
}
```

**Key behavior**: Excluded callsigns are **tracked but not scored** — their `TransmissionLog` rows are still persisted. This preserves data in case the operator changes their mind and wants to run a retroactive rescore.

**Empty list**: If `excluded_callsigns` is absent or empty, behavior is unchanged.

### Files Changed
- `backend/config/config.go` — add `ExcludedCallsigns` field to `GamificationConfig`
- `backend/gamification/config.go` — add `ExcludedCallsigns` and wire from `GamificationConfig` in `main.go`
- `backend/gamification/tally_service.go` — add `excludedCallsigns` field, check in `processGroup()`

### Tests
- `tally_service_test.go`: Add test case with mixed callsigns and exclusion list — verify excluded callsigns are not in scoreboard results

---

## Feature 3: Per-Node AMI Connections + Multi-Node UI

### Goal
Enable correct per-node keying state tracking via dedicated AMI connections per node, and add a UI toggle to show/hide individual node cards.

### Part A: Per-Node AMI Connections (Backend)

#### Config Change — `NodeConfig`

```go
type NodeConfig struct {
    NodeID  int
    Name    string
    Visible *bool  // nil = true by default (show in UI); false = hidden
    // Optional per-node AMI overrides (empty = inherit from global)
    AMIHost string `mapstructure:"ami_host" yaml:"ami_host"`
    AMIPort int    `mapstructure:"ami_port" yaml:"ami_port"`
}
```

User config (all on one server):
```yaml
ami_host: 192.168.1.100
ami_port: 5038
nodes:
  - node_id: 594950
    name: "Primary"
  - node_id: 594951
    name: "Secondary"
    visible: true  # optional, default true if not specified
  - node_id: 594952
    name: "Tertiary"
    visible: false  # hide by default on first load
```

User config (distributed across servers):
```yaml
nodes:
  - node_id: 594950
    ami_host: 192.168.1.100
    ami_port: 5038
    name: "Primary"
  - node_id: 594951
    ami_host: 192.168.1.101
    ami_port: 5038
    name: "Secondary"
```

#### Routing Mechanism: Tag Events with SourceNodeID

The core problem: `sm.Run(conn.Raw())` consumes events from all connectors on a single channel, so the state manager doesn't know which node an event came from. The fix is to tag each message with its source node ID before it enters the state manager.

**Step 1 — Add `SourceNodeID` to `ami.Message` struct** (`internal/ami/message.go` or wherever `Message` is defined):

```go
type Message struct {
    Headers map[string]string
    // ... existing fields ...
    SourceNodeID int  // Set by the producing connector before enqueuing
}
```

**Step 2 — Tag messages in each connector's `Run()` loop** (`internal/ami/connector.go`):

Each connector knows its own `sourceNodeID`. Before putting a message on `rawOut`, tag it:

```go
func (c *AMIConnector) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case msg, ok := <-c.conn.Output():
            if !ok {
                return
            }
            // Tag every message with the source node ID this connector handles
            msg.SourceNodeID = c.sourceNodeID
            select {
            case c.rawOut <- msg:
            default: // non-blocking
            }
        }
    }
}
```

**Step 3 — `StateManager.Apply()` reads `msg.SourceNodeID`** (`internal/core/state.go`):

The `Apply()` signature stays unchanged (`Apply(m ami.Message)`), but it reads the tag from the message:

```go
// Process keying trackers — ONLY the tracker matching this message's sourceNodeID
if tracker, exists := sm.keyingTrackers[m.SourceNodeID]; exists {
    tracker.ProcessALinks(ids, keyedMap, now)
    // ... enrich and update only this tracker
}
```

The current broken behavior (all trackers process every ALINKS event) comes from the loop in `Apply()` at lines 337-341:

```go
// BEFORE (broken):
for sourceNodeID, tracker := range sm.keyingTrackers {
    tracker.ProcessALinks(ids, keyedMap, now)  // all trackers get all events!
}
```

**AFTER (fixed)**:
```go
// AFTER (correct):
if tracker, exists := sm.keyingTrackers[m.SourceNodeID]; exists {
    tracker.ProcessALinks(ids, keyedMap, now)  // only the correct tracker processes this event
}
```

**Why this approach**:
- `ami.Message` is the project's own type — adding a field is safe
- `Connector.Run()` already loops over messages — minimal change to tag each one
- `StateManager.Apply()` stays the same signature — just reads `m.SourceNodeID`
- No new channels or goroutine patterns needed

#### `Connector` Struct — Add `sourceNodeID`

The `Connector` struct needs a `sourceNodeID` field so the `Run()` loop can tag messages:

```go
// internal/ami/connector.go — add to Connector struct
type Connector struct {
    // ... existing fields (host, port, user, pass, etc.) ...
    sourceNodeID int  // set at construction, used to tag messages
}
```

The `NewConnector` function signature may need adjustment to accept `sourceNodeID`. If `NewConnector` currently takes many individual params, consider adding a `WithSourceNodeID(id int)` setter to be called after construction, or refactor to accept a config/options struct.

**IMPORTANT — Verify current `NewConnector` signature** in `connector.go` before implementing. The spec assumes it takes many individual params; if it already uses a handler struct, the approach may differ. Read `NewConnector`'s actual signature and adapt accordingly.

#### `main.go` Connector Creation Loop

```go
// Create one AMI connector per node
for _, node := range cfg.Nodes {
    amiHost := node.AMIHost
    if amiHost == "" {
        amiHost = cfg.AmiHost
    }
    amiPort := node.AMIPort
    if amiPort == 0 {
        amiPort = cfg.AmiPort
    }

    connector := ami.NewConnector(amiHost, amiPort, cfg.AmiUser, cfg.AmiPassword, cfg.AmiReconnectMin, cfg.AmiReconnectMax)
    connector.WithSourceNodeID(node.NodeID)  // tag all this connector's events with its node ID

    // Feed into StateManager (existing pattern — verify connector exposes rawOut or equivalent)
    go func(c *ami.Connector) {
        for msg := range c.Raw() {  // or whatever the channel accessor is
            sm.Apply(msg)  // msg.SourceNodeID is already set by connector
        }
    }(connector)

    // Start connector (background)
    go connector.Run()
}
```

**Note**: If multiple nodes share the same `ami_host:ami_port`, it's safe to create separate connectors — Asterisk handles multiple manager accounts fine. Each connector logs in independently.

#### SeedKeyingTrackerFromLinks — Call for All Nodes

Currently only called for `cfg.Nodes[0]`. Change to call for all nodes:
```go
for _, node := range cfg.Nodes {
    sm.SeedKeyingTrackerFromLinks(node.NodeID)
}
```

#### Files Changed (Backend)
- `internal/ami/message.go` (or wherever `Message` struct is) — add `SourceNodeID int` field
- `internal/ami/connector.go` — add `sourceNodeID` to `Connector` struct, tag messages in `Run()` loop
- `internal/core/state.go` — fix ALINKS routing: replace broadcast-to-all loop with single-tracker dispatch using `m.SourceNodeID`
- `backend/config/config.go` — add `Visible`, `AMIHost`, `AMIPort` to `NodeConfig`
- `main.go` — loop to create per-node connectors, feed into state manager

#### Benefits (Beyond UI)
- Correct `TransmissionLog.SourceID` on every row — each tracker logs only TX events it observes via its own AMI connection
- Per-node `SourceNodeKeyingUpdate` is now accurate — cards show only the links active on that specific node
- Feature 1's scoring node filter remains independently useful as an explicit override (not replaced by this change)

---

### Part B: Node Card Visibility Toggle (Frontend)

#### Config Cascade

The backend should expose node visibility in the API response so the frontend can default correctly. The `STATUS_UPDATE` or a new `CFG_NODES` message type should include:

```json
{
  "messageType": "CFG_NODES",
  "data": [
    { "nodeId": 594950, "name": "Primary", "visible": true },
    { "nodeId": 594951, "name": "Secondary", "visible": true },
    { "nodeId": 594952, "name": "Tertiary", "visible": false }
  ]
}
```

**Alternatively** (simpler): Have the frontend default all nodes to visible, and let `localStorage` override. The backend config doesn't strictly need to flow to the frontend for visibility — the toggle state lives in `localStorage` on the client.

**Decision**: Use `localStorage` only for visibility state. The backend doesn't need to track it. This keeps the backend unchanged for Feature 3 Part A (per-node AMI) focused on correctness, not UI state.

#### Dashboard Toggle UI

Above the source node cards in `Dashboard.vue`, add a row of toggle buttons — one per configured node:

```html
<div class="node-toggles" v-if="Object.keys(nodeStore.sourceNodes).length > 1">
  <button
    v-for="(data, nodeId) in nodeStore.sourceNodes"
    :key="nodeId"
    :class="['node-toggle', { active: visibleNodes[nodeId] }]"
    @click="toggleNode(Number(nodeId))"
  >
    {{ data.name || nodeId }}
  </button>
</div>
```

#### Visibility State

```js
// In Dashboard.vue
const visibleNodes = ref({})  // nodeId -> boolean

onMounted(() => {
  // Load persisted visibility from localStorage
  const saved = localStorage.getItem('nodeVisibility')
  if (saved) {
    visibleNodes.value = JSON.parse(saved)
  } else {
    // Default: all visible
    const nodes = nodeStore.sourceNodes
    for (const k of Object.keys(nodes)) {
      visibleNodes.value[k] = true
    }
  }
})

function toggleNode(nodeId) {
  visibleNodes.value[nodeId] = !visibleNodes.value[nodeId]
  localStorage.setItem('nodeVisibility', JSON.stringify(visibleNodes.value))
}
```

#### Conditional Card Rendering

```html
<div v-for="(entry, key) in nodeStore.sourceNodes" :key="key" class="source-node-wrapper">
  <SourceNodeCard
    v-if="visibleNodes[key] !== false"
    :source-node-id="Number(key)"
    :data="entry"
  />
</div>
```

Note: `visibleNodes[key] !== false` means nodes not in `localStorage` default to **visible** (v-if="true").

#### Collapse/Hide as Future Enhancement
Deferred — not in this spec. The toggle approach above satisfies the immediate need.

#### Files Changed (Frontend)
- `frontend/src/views/Dashboard.vue` — add toggle buttons, visibility state, localStorage persistence

---

## Implementation Order

1. **Feature 1** (Scoring Node Filter) — gamification config + `ProcessTally` filter. Smallest, safest.
2. **Feature 2** (Exclude Callsigns) — config + `processGroup` skip. Small, same file.
3. **Feature 3 Part A** (Per-Node AMI) — `ami.Message` field + connector tagging + routing fix. Largest, most impactful.
4. **Feature 3 Part B** (UI Toggles) — frontend only, independent of Part A.

Features 1 and 2 are independent of each other and of Feature 3. Feature 3 Part B (UI) is also independent of Part A (backend) — the UI toggle works regardless of how the backend routes events.

---

## v1 → v2 Changelog (Review Fixes)

| Change | Reason |
|--------|--------|
| Added `SourceNodeID int` to `ami.Message` struct | `Apply()` needs to know which node an event came from — tagging the message is cleaner than changing `Apply`'s signature |
| Tagged messages in `Connector.Run()` loop | Each connector knows its `sourceNodeID` — set it on every message before enqueuing |
| Changed `Apply()` routing from broadcast-to-all to single-tracker dispatch | Fixes the core bug: all trackers receiving all events |
| Added `gamification.Config` wiring clarification | `TallyService` uses a separate `gamification.Config` struct — fields must be added there AND wired from `config.GamificationConfig` in `main.go` |
| Removed "eliminates double-counting for free" claim | Feature 1 scoring filter remains independently useful; per-node connectors fix routing but don't replace the explicit override |
| Updated `NewConnector` approach | Original spec showed handler-based pattern but `NewConnector` uses individual params — clarified to use `WithSourceNodeID()` setter or equivalent |

---

## Open Questions

| # | Question | Decision |
|---|---|---|
| 1 | `scoring_source_node_id = 0` means all nodes score | ✅ Yes — backward compatible |
| 2 | Excluded callsigns config uppercase or normalize at runtime? | ✅ Uppercase in config, normalized at runtime |
| 3 | Excluded callsigns tracked but not scored? | ✅ Yes — logs persisted for future rescore |
| 4 | Validate `scoring_source_node_id` against `cfg.Nodes`? | ✅ Warn at startup if no match, don't fail |
| 5 | `visible` default if not specified? | ✅ `true` (show all by default) |
| 6 | Backend flow visibility to frontend? | ✅ No — localStorage only, backend unchanged for this |
| 7 | Per-node AMI overrides or global fallback? | ✅ Optional per-node overrides, global fallback |
