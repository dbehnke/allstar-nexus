# Allstar Nexus – Session Summary & Recommended TODOs

_Date: 2025-10-09_
_Branch: `dbehnke/ami-event-improvements`_

This document captures the analysis performed on the AMI keying tracking specification (pasted PDF text), backend implementation, server runtime status, and frontend WebSocket/UI handling. It concludes with prioritized actionable tasks.

---
## 1. Specification Essence (AMI Keying Logic)
Core goals extracted from the provided spec text:
1. Use `RPT_ALINKS` as authoritative list of **adjacent** links with per-link keyed flags (`K` vs `U`).
2. Maintain a per-adjacent-node state machine: `IsTransmitting`, `IsKeyed`, `KeyedStartTime`.
3. On key-up (transition to keyed) start session; on unkey schedule a **2.0s jitter compensation delay** before confirming end.
4. Cancel pending unkey timer if re-key occurs within delay.
5. Record per-session duration and accumulate totals.
6. Distinguish adjacent vs global link counts (`RPT_NUMALINKS` vs `RPT_NUMLINKS`).
7. Provide data suitable for UI attribution of current talker(s).

---
## 2. Backend Implementation Verification
Relevant files: `internal/core/keying_tracker.go`, `internal/core/state.go`.

| Spec Aspect | Implemented? | Notes |
|-------------|--------------|-------|
| ALINKS parsing & keyed extraction | ✅ | `parseALinks` robust (numeric + text node support). |
| Per-node state (transmit/keyed/start time) | ✅ | `AdjacentNodeStatus` struct. |
| 2s delayed unkey (jitter compensation) | ✅ | `UnkeyCheckTimer` queue, default 2000 ms. |
| Timer cancellation on re-key | ✅ | `removeFromQueue` invoked on continued key. |
| Session duration & accumulation | ✅ | Seconds granularity stored in `TotalTxSeconds`. |
| Adjacent vs global counts | ✅ | `NumALinks` / `NumLinks` in `NodeState`. |
| Multi-source node support | ✅ | `keyingTrackers` map supports multiple sources. |
| Duplicate event suppression | ✅ | `lastTalkerState` map. |
| Expose keyed snapshots to frontend | ✅ | `SOURCE_NODE_KEYING` update channel. |
| Include local TX (`RPT_TXKEYED`) in keying update | ❌ (partial) | TODO comment present; not yet propagated into `SourceNodeKeyingUpdate`. |
| Explicit session edge event (with duration) | ❌ | Only cumulative & snapshot; no dedicated start/end payload. |
| Pending-unkey indicator | ❌ | Implicit state (IsTransmitting true & IsKeyed false) not explicitly surfaced. |
| Millisecond precision | ❌ | Seconds only (likely acceptable). |

Risk / Gap Summary:
- No explicit session edge events → frontend must infer from snapshot diffs or link TX events.
- No explicit pending-unkey flag → UI cannot clearly show “Ending…” state without heuristic.
- Local Tx/Rx booleans missing in source node keying update (commented TODO).
- Duration precision limited to seconds.
- No guard for excessively long sessions (stale TX if unkey lost).

---
## 3. Frontend WebSocket & UI Mapping
Relevant files: `frontend/src/stores/node.js`, `SourceNodeCard.vue`, `StatusCard.vue`, `TopLinksCard.vue`, `env.js`.

Strengths:
- Consumes `SOURCE_NODE_KEYING` snapshots and renders adjacent nodes with live duration.
- Separate handling for talker events, link TX edges, and status updates.
- Talker event de-dupe & STOP buffering reduces flicker.

Gaps / Opportunities:
- STOP event 1s buffering may introduce unnecessary latency now that backend already applies 2s jitter. Evaluate removal or gating behind config.
- Adjacent vs global counts not visually juxtaposed (user lacks quick topology context).
- Pending unkey (during 2s delay) not visually indicated (could show “Ending…” badge when `IsTransmitting && !IsKeyed`).
- No staleness indicator if `SOURCE_NODE_KEYING` snapshots pause.
- Global fallback TALKER events (node==0) are discarded—verify backend guarantees per-link events or relax filter.
- Verbose `console.debug` logs always active; should be gated by environment.

---
## 4. Server Runtime Health (Earlier Check)
- Active process on `:8080` healthy (`/api/health` returning 200).
- Stale `server.pid` detected (PID file process not alive); new instance running under different PID.
- Bind error logged (`address already in use`) indicates overlapping or restart attempt.

Recommended operational hygiene:
- Validate PID file on startup; remove if stale.
- Write PID only after successful bind.
- Add enriched `/api/health` fields: uptime, build, number of keying timers, source node count.

---
## 5. Prioritized TODOs
### P1 (High Value / Low Risk)
1. Add `tx_keyed` / `rx_keyed` population in `SourceNodeKeyingUpdate` (complete the TODO).  
2. Include explicit session edge events channel (e.g., `SOURCE_NODE_KEYING_EVENT` with `{type: TX_START|TX_END, source_node_id, node_id, start, end, duration_s}`).  
3. Add explicit `PendingUnkey` boolean (set when unkey timer scheduled) to `AdjacentNodeStatus` JSON; highlight in UI.  
4. Frontend: display adjacent/global counts side-by-side in `StatusCard`.  
5. Frontend: show “Ending…” state (distinct style) when `IsTransmitting && !IsKeyed`.

### P2 (Observability & Robustness)
6. Add jitter metrics counters: `unkey_timers_started`, `unkey_timers_canceled`, `unkey_timers_confirmed`.  
7. Add max session safety (e.g., if transmission > 30 min without unkey event, auto-close and log warning).  
8. Implement stale snapshot detection (`last_update_age > 5s` → UI badge).  
9. PID file sanitation logic & improved bind error message with suggestion.

### P3 (Enhancements / Polish)
10. Millisecond precision for current session live timer (store start timestamp already available).  
11. Optional adaptive jitter window (auto-adjust if >N cancellations in last M seconds).  
12. Replace slice-based timer queue with min-heap if timers scale significantly.  
13. Gate frontend debug logging via `import.meta.env.DEV` or config flag.  
14. Remove TALKER STOP buffering if confirmed unnecessary; make it configurable.  
15. Add `/api/metrics` placeholder (Prometheus-style) for keying stats.

### P4 (Nice-to-Have / Future)
16. Persist per-session records (not just cumulative totals) for historical analytics.  
17. UI timeline view of recent sessions with durations and node metadata.  
18. Multi-source aggregation card: currently transmitting nodes grouped by source.  
19. Alerting hook (webhook or MQTT) on long session or rapid flapping.

---
## 6. Implementation Sketches
Backend (`keying_tracker.go`):
- Add field `pendingUnkey map[int]bool` or infer from timer queue; expose via `AdjacentNodeStatus` (export JSON tag).
- In `ProcessALinks` when scheduling unkey timer: `nodeStatus.PendingUnkey = true`; on re-key or confirmed end: set false.
- Add callback invocation with session details on `processTxEnd`.

Frontend (`SourceNodeCard.vue`):
```vue
<span class="status-badge"
  :class="node.IsTransmitting ? (node.IsKeyed ? 'keyed' : 'ending') : 'idle'">
  {{ node.IsTransmitting ? (node.IsKeyed ? 'Keyed' : 'Ending…') : 'Idle' }}
</span>
```
Styling: `.status-badge.ending { background:#ffb347; color:#222; }` (or theme variable equivalent).

`StatusCard.vue` add:
```html
<div class="status-item">
  <span class="label">Links (Adj/Total):</span>
  <span class="value">{{ status.num_alinks }} / {{ status.num_links }}</span>
</div>
```

---
## 7. Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Added fields break older clients | Version new WS message types; keep existing fields stable. |
| Timer queue complexity growth | Switch to heap only if profile shows contention. |
| Extra events increase WS traffic | Optionally batch session edge events or piggyback on snapshots. |
| Duplicate logic (frontend vs backend jitter) | Remove frontend STOP buffering after verifying backend correctness with logs. |

---
## 8. Suggested Work Order
1. Backend: implement P1 set (fields + events + pending flag) with tests.  
2. Frontend: update UI components & remove redundant buffering behind a flag.  
3. Add observability metrics (P2).  
4. Operational improvements (PID hygiene, enriched health).  
5. Optional performance/adaptive adjustments.  

---
## 9. Quick Acceptance Criteria (for P1)
- SOURCE_NODE_KEYING now includes `tx_keyed`, `rx_keyed` populated, and each adjacent node includes `PendingUnkey` (bool).
- New WS message `SOURCE_NODE_KEYING_EVENT` emits start/end edges with duration.
- Frontend renders “Ending…” state inside 0–2s window after unkey before final end.
- Adjacent/global counts visible in status card.
- No regression in existing talker / top links cards (all tests green).

---
## 10. Follow-Up
Indicate which P1 tasks to implement first and I can raise a PR on this branch. This document can be updated as changes land.

---
_Authored by automated analysis during collaborative session._
