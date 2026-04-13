# Multi-Node Gamification & UI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement three features: (1) scoring node filter, (2) callsign exclusions, (3A) per-node AMI connections, (3B) node card visibility toggles in the UI.

**Architecture:** Features 1 & 2 are pure config + in-memory filtering in `TallyService`. Feature 3A adds `SourceNodeID` to `ami.Message` and routes ALINKS events to per-node KeyingTrackers. Feature 3B adds localStorage-persisted visibility toggle in `Dashboard.vue`.

**Tech Stack:** Go 1.25, Vue 3, github.com/coder/websocket, modernc.org/sqlite

---

## Chunk 1: Scoring Node Filter + Exclude Callsigns

**Files:**
- Modify: `backend/config/config.go`
- Modify: `backend/gamification/config.go`
- Modify: `backend/gamification/tally_service.go`
- Modify: `backend/tests/tally_service_test.go`
- Modify: `main.go` (wire gamification config fields)

### Task 1: Add `ScoringSourceNodeID` to config structs

**Files:**
- Modify: `backend/config/config.go` — `GamificationConfig` struct
- Modify: `backend/gamification/config.go` — `Config` struct
- Modify: `main.go` — wire `cfg.Gamification.ScoringSourceNodeID` to `gameCfg.ScoringSourceNodeID`

- [ ] **Step 1: Read current structs**

Read `backend/config/config.go` around line 22 (`GamificationConfig` struct) and `backend/gamification/config.go` (`Config` struct) to confirm current fields. Read `main.go` around the `gameCfg` construction block (search for `gameCfg :=`) to find where fields are wired.

- [ ] **Step 2: Add `ScoringSourceNodeID` to `config.GamificationConfig`**

```go
// backend/config/config.go — GamificationConfig struct
type GamificationConfig struct {
    // ... existing fields ...
    ScoringSourceNodeID int `mapstructure:"scoring_source_node_id" yaml:"scoring_source_node_id"`
}
```

- [ ] **Step 3: Add `ScoringSourceNodeID` to `gamification.Config`**

```go
// backend/gamification/config.go — Config struct
type Config struct {
    // ... existing fields ...
    ScoringSourceNodeID int
}
```

- [ ] **Step 4: Wire in `main.go`**

Find where `gameCfg` is constructed from `cfg.Gamification`. Add:
```go
gameCfg.ScoringSourceNodeID = cfg.Gamification.ScoringSourceNodeID
```

- [ ] **Step 5: Commit**

```bash
git add backend/config/config.go backend/gamification/config.go main.go
git commit -m "feat(gamification): add ScoringSourceNodeID to config structs"
```

---

### Task 2: Add `ExcludedCallsigns` to config structs

**Files:**
- Modify: `backend/config/config.go` — `GamificationConfig` struct
- Modify: `backend/gamification/config.go` — `Config` struct
- Modify: `main.go` — wire `ExcludedCallsigns`

- [ ] **Step 1: Add `ExcludedCallsigns` to `config.GamificationConfig`**

```go
// backend/config/config.go — GamificationConfig struct
ExcludedCallsigns []string `mapstructure:"excluded_callsigns" yaml:"excluded_callsigns"`
```

- [ ] **Step 2: Add `ExcludedCallsigns` to `gamification.Config`**

```go
// backend/gamification/config.go — Config struct
ExcludedCallsigns []string
```

- [ ] **Step 3: Wire in `main.go`**

```go
gameCfg.ExcludedCallsigns = cfg.Gamification.ExcludedCallsigns
```

- [ ] **Step 4: Commit**

```bash
git add backend/config/config.go backend/gamification/config.go main.go
git commit -m "feat(gamification): add ExcludedCallsigns to config structs"
```

---

### Task 3: Implement scoring node filter in `ProcessTally`

**Files:**
- Modify: `backend/gamification/tally_service.go` — `ProcessTally()` method

- [ ] **Step 1: Read `ProcessTally()` to find insertion point**

Read `backend/gamification/tally_service.go` around line 330 (`ProcessTally` method). Find where `transmissions` is assigned from `GetLogsBetween()` and what type it is (`map[string][]models.TransmissionLog`).

- [ ] **Step 2: Write failing test**

```go
// backend/tests/tally_service_test.go
// Add to existing test file or create if doesn't exist
func TestScoringNodeID_FiltersLogs(t *testing.T) {
    cfg := &gamification.Config{
        ScoringSourceNodeID: 594950,
        // ... other required fields ...
    }
    svc := gamification.NewTallyService(cfg, txLogRepo, userRepo, linkRepo)

    // Insert test logs: some from 594950, some from 594951
    repo.LogTransmission(context.Background(), models.TransmissionLog{
        SourceID: 594950, Callsign: "W1AW", DurationSeconds: 60,
    })
    repo.LogTransmission(context.Background(), models.TransmissionLog{
        SourceID: 594951, Callsign: "W1AW", DurationSeconds: 60,
    })

    // Process tally
    svc.ProcessTally(context.Background())

    // Verify only 594950's log contributed to score
    // (具体断言取决于 scoreboard 结果 — adjust based on actual test structure)
    scoreboard := svc.GetLastScoreboard()
    assert.Equal(t, 1, len(scoreboard)) // W1AW appears once (from node 594950 only)
}
```

- [ ] **Step 3: Implement the filter**

After `transmissions, err := s.txLogRepo.GetLogsBetween(cursor, next)`:

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

- [ ] **Step 4: Verify existing tests pass**

```bash
go test ./backend/gamification/... -v -run TestScoringNodeID
```

- [ ] **Step 5: Commit**

```bash
git add backend/gamification/tally_service.go backend/tests/tally_service_test.go
git commit -m "feat(gamification): filter transmission logs by scoring_source_node_id"
```

---

### Task 4: Implement exclude callsigns in `processGroup`

**Files:**
- Modify: `backend/gamification/tally_service.go` — `NewTallyService()` and `processGroup()`
- Test: `backend/tests/tally_service_test.go`

- [ ] **Step 1: Read `NewTallyService()` and `processGroup()` signatures**

Read `backend/gamification/tally_service.go` to find the `TallyService` struct fields, `NewTallyService()` function signature, and the top of `processGroup()` loop.

- [ ] **Step 2: Add `excludedCallsigns map[string]struct{}` field to `TallyService`**

```go
type TallyService struct {
    // ... existing fields ...
    excludedCallsigns map[string]struct{}
}
```

- [ ] **Step 3: Initialize in `NewTallyService()`**

```go
excluded := make(map[string]struct{})
for _, cs := range cfg.ExcludedCallsigns {
    excluded[strings.ToUpper(cs)] = struct{}{}
}
svc.excludedCallsigns = excluded
```

- [ ] **Step 4: Write failing test**

```go
func TestExcludedCallsigns_NotScored(t *testing.T) {
    cfg := &gamification.Config{
        ExcludedCallsigns: []string{"W1AW", "WX5NWS"},
        // ... other required fields ...
    }
    svc := gamification.NewTallyService(cfg, txLogRepo, userRepo, linkRepo)

    // Insert logs from excluded and non-excluded callsigns
    repo.LogTransmission(context.Background(), models.TransmissionLog{
        SourceID: 594950, Callsign: "W1AW", DurationSeconds: 60,
    })
    repo.LogTransmission(context.Background(), models.TransmissionLog{
        SourceID: 594950, Callsign: "N0CALL", DurationSeconds: 60,
    })

    svc.ProcessTally(context.Background())

    scoreboard := svc.GetLastScoreboard()
    // W1AW should not appear; N0CALL should appear
    for _, entry := range scoreboard {
        assert.NotEqual(t, "W1AW", entry.Callsign)
    }
}
```

- [ ] **Step 5: Add exclusion check at top of `processGroup()` loop**

Read `processGroup()` — find the `for callsign, txLogs := range transmissions` loop. At the top of the loop body, before any XP logic:

```go
if s.excludedCallsigns != nil {
    if _, ok := s.excludedCallsigns[strings.ToUpper(callsign)]; ok {
        continue
    }
}
```

Note: Use `strings.ToUpper(callsign)` for the lookup since config is uppercase.

- [ ] **Step 6: Run tests**

```bash
go test ./backend/gamification/... -v -run TestExcludedCallsigns
```

- [ ] **Step 7: Commit**

```bash
git add backend/gamification/tally_service.go backend/tests/tally_service_test.go
git commit -m "feat(gamification): exclude callsigns from XP scoring"
```

---

## Chunk 2: Per-Node AMI Connections (Backend)

**Files:**
- Modify: `internal/ami/message.go` — add `SourceNodeID int`
- Modify: `internal/ami/connector.go` — add `sourceNodeID`, tag messages in `Run()`
- Modify: `internal/core/state.go` — fix ALINKS routing
- Modify: `backend/config/config.go` — add `Visible`, `AMIHost`, `AMIPort` to `NodeConfig`
- Modify: `main.go` — per-node connector loop, `SeedKeyingTrackerFromLinks` for all nodes

### Task 5: Add `SourceNodeID` to `ami.Message`

**Files:**
- Modify: `internal/ami/message.go`

- [ ] **Step 1: Read the `Message` struct**

Find and read `internal/ami/message.go` (or wherever `ami.Message` is defined — search for `type Message struct`).

- [ ] **Step 2: Add field**

```go
type Message struct {
    Headers map[string]string
    // ... existing fields ...
    SourceNodeID int  // Set by the producing connector before enqueuing; 0 if not yet tagged
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/ami/message.go
git commit -m "feat(ami): add SourceNodeID to Message struct"
```

---

### Task 6: Add `sourceNodeID` to `Connector` and tag messages

**Files:**
- Modify: `internal/ami/connector.go`

- [ ] **Step 1: Read `Connector` struct and `Run()` method**

Read `internal/ami/connector.go`. Find the `Connector` struct definition and the `Run()` method. Note how messages flow from the AMI connection to `rawOut`.

- [ ] **Step 2: Add `sourceNodeID` field to `Connector`**

```go
type Connector struct {
    // ... existing fields ...
    sourceNodeID int
}
```

- [ ] **Step 3: Add `WithSourceNodeID()` setter**

```go
func (c *Connector) WithSourceNodeID(id int) *Connector {
    c.sourceNodeID = id
    return c
}
```

- [ ] **Step 4: Tag messages in `Run()` — BEFORE putting on `rawOut`**

Find where messages are put on `rawOut`. Before the send, add:
```go
msg.SourceNodeID = c.sourceNodeID
```

The exact location depends on the current code — typically in the `Run()` loop where `c.conn.Output()` is read and pushed to `c.rawOut`. The tag must be set before the message enters any shared channel.

- [ ] **Step 5: Commit**

```bash
git add internal/ami/connector.go
git commit -m "feat(ami): tag messages with sourceNodeID in connector Run() loop"
```

---

### Task 7: Fix ALINKS routing in `StateManager.Apply()`

**Files:**
- Modify: `internal/core/state.go`

- [ ] **Step 1: Read the ALINKS processing section of `Apply()`**

Read `internal/core/state.go` around lines 337-397 (the `RPT_ALINKS` case). Find the loop `for sourceNodeID, tracker := range sm.keyingTrackers` — this is the bug.

- [ ] **Step 2: Replace broadcast loop with single-tracker dispatch**

Replace the broadcast loop:
```go
// BEFORE (BROKEN — all trackers get all events):
for sourceNodeID, tracker := range sm.keyingTrackers {
    tracker.ProcessALinks(ids, keyedMap, now)
    // ... enrichment for all trackers ...
}
```

With:
```go
// AFTER (CORRECT — only the tracker matching this message's source node):
if tracker, exists := sm.keyingTrackers[m.SourceNodeID]; exists {
    tracker.ProcessALinks(ids, keyedMap, now)
    // ... enrichment for only this tracker ...
} else {
    log.Printf("[STATE] no keyingTracker for sourceNodeID=%d, dropping ALINKS event", m.SourceNodeID)
}
```

Also update the enrichment section (lines 343-369) to only enrich the matching tracker:
```go
// Only enrich the tracker matching this sourceNodeID
if tracker, exists := sm.keyingTrackers[m.SourceNodeID]; exists {
    if sm.nodeLookup != nil {
        for _, nodeID := range ids {
            if nodeID > 0 {
                if info := sm.nodeLookup.LookupNode(nodeID); info != nil {
                    tracker.UpdateNodeInfo(nodeID, info.Callsign, info.Description)
                }
            }
        }
    }
}
```

**Important**: The `sm.numALinks` and `sm.perSourceNumALinks` tracking should still work correctly — verify the code paths that update these aren't broken by the routing change.

- [ ] **Step 3: Verify the fix**

Run existing state tests:
```bash
go test ./internal/core/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/core/state.go
git commit -m "fix(core): route ALINKS events only to matching keyingTracker"
```

---

### Task 8: Add `Visible`, `AMIHost`, `AMIPort` to `NodeConfig`

**Files:**
- Modify: `backend/config/config.go`

- [ ] **Step 1: Read `NodeConfig` struct**

Read `backend/config/config.go` around line 15 — find the `NodeConfig` struct.

- [ ] **Step 2: Add fields**

```go
type NodeConfig struct {
    NodeID   int
    Name     string
    Visible  *bool  // nil = true (show in UI by default)
    AMIHost  string `mapstructure:"ami_host" yaml:"ami_host"`  // empty = inherit global
    AMIPort  int    `mapstructure:"ami_port" yaml:"ami_port"`   // 0 = inherit global
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/config/config.go
git commit -m "feat(config): add Visible, AMIHost, AMIPort to NodeConfig"
```

---

### Task 9: Per-node connector loop in `main.go`

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Read current connector creation**

Read `main.go` around line 589 where `ami.NewConnector()` is called. Understand:
- Current `NewConnector` signature
- How connectors are currently fed into `StateManager`
- How `sm.Run()` is called with `conn.Raw()`

- [ ] **Step 2: Replace single connector with per-node loop**

Find the current single `NewConnector` call. Replace with:

```go
// Create one AMI connector per configured node
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
    connector.WithSourceNodeID(node.NodeID)

    // Feed this connector's events into StateManager
    // Note: if using shared sm.Run(conn.Raw()) pattern, replace with per-connector feeding
    go func(c *ami.Connector) {
        for msg := range c.Raw() {
            sm.Apply(msg)  // msg.SourceNodeID is already tagged by connector
        }
    }(connector)

    // Start connector (background, with reconnect)
    go connector.Run()
}
```

**Verify**: Check if `c.Raw()` is the correct channel accessor name (may be `c.Output()` or similar). Read the connector code to confirm the channel name.

- [ ] **Step 3: Update `SeedKeyingTrackerFromLinks` to seed all nodes**

Find the current call at line 485 (`sm.SeedKeyingTrackerFromLinks(cfg.Nodes[0].NodeID)`). Replace with:
```go
for _, node := range cfg.Nodes {
    sm.SeedKeyingTrackerFromLinks(node.NodeID)
}
```

- [ ] **Step 4: Build to verify**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat(multi-node): create per-node AMI connectors with sourceNodeID tagging"
```

---

## Chunk 3: Node Card Visibility Toggle (Frontend)

**Files:**
- Modify: `frontend/src/views/Dashboard.vue`

### Task 10: Add visibility toggle buttons to `Dashboard.vue`

- [ ] **Step 1: Read `Dashboard.vue` and `nodeStore`**

Read `frontend/src/views/Dashboard.vue` to understand the current template structure around the source node cards. Read `frontend/src/stores/node.js` to see `sourceNodes` ref and `handleWSMessage`.

- [ ] **Step 2: Read `SourceNodeCard.vue` to understand node naming**

Read `frontend/src/components/SourceNodeCard.vue` to see how nodes display their name/ID. The toggle buttons should use `data.name || sourceNodeID`.

- [ ] **Step 3: Add toggle button row above source node cards**

Find where the source node cards are rendered (the `v-for` over `nodeStore.sourceNodes`). Add a toggle row above it:

```html
<!-- Dashboard.vue — add above the .full-width div containing SourceNodeCards -->
<div class="node-toggles" v-if="Object.keys(nodeStore.sourceNodes).length > 1">
  <button
    v-for="(data, key) in nodeStore.sourceNodes"
    :key="key"
    :class="['node-toggle-btn', { active: visibleNodes[key] !== false }]"
    @click="toggleNode(key)"
  >
    {{ data.name || data.sourceNodeID || key }}
  </button>
</div>
```

- [ ] **Step 4: Add `visibleNodes` ref and toggle logic**

```js
// In Dashboard.vue — in <script setup>
import { ref, onMounted } from 'vue'
// ... existing imports

const visibleNodes = ref({})  // nodeId string -> boolean

onMounted(() => {
  // Load persisted visibility or default to all visible
  const saved = localStorage.getItem('nodeVisibility')
  if (saved) {
    visibleNodes.value = JSON.parse(saved)
  } else {
    for (const k of Object.keys(nodeStore.sourceNodes)) {
      visibleNodes.value[k] = true
    }
  }
})

function toggleNode(key) {
  if (visibleNodes.value[key] === false) {
    visibleNodes.value[key] = true
  } else {
    visibleNodes.value[key] = false
  }
  localStorage.setItem('nodeVisibility', JSON.stringify(visibleNodes.value))
}
```

- [ ] **Step 5: Add conditional rendering to source node cards**

In the `v-for` div:
```html
<div
  v-for="(entry, key) in nodeStore.sourceNodes"
  :key="key"
  class="source-node-wrapper"
>
  <SourceNodeCard
    v-if="visibleNodes[key] !== false"
    :source-node-id="Number(key)"
    :data="entry"
  />
</div>
```

- [ ] **Step 6: Add CSS for toggle buttons**

```css
/* Add to Dashboard.vue <style scoped> */
.node-toggles {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.node-toggle-btn {
  padding: 0.4rem 0.8rem;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.15s;
}

.node-toggle-btn:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}

.node-toggle-btn.active {
  background: var(--accent-primary);
  border-color: var(--accent-primary);
  color: #fff;
}
```

- [ ] **Step 7: Build frontend to verify**

```bash
cd frontend && npm run build
```

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/Dashboard.vue
git commit -m "feat(ui): add node card visibility toggle buttons to dashboard"
```

---

## Chunk 4: Integration Test

### Task 11: Verify all features work together

- [ ] **Step 1: Build full application**

```bash
go build ./...
cd frontend && npm run build && cd ..
```

- [ ] **Step 2: Run all backend tests**

```bash
go test ./backend/... ./internal/... -v
```

- [ ] **Step 3: Run frontend tests**

```bash
cd frontend && npm test 2>&1 | head -50
```

- [ ] **Step 4: Commit any remaining changes**

```bash
git add -A && git commit -m "feat: multi-node gamification and UI — all features"
```
