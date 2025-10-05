# Allstar Nexus – Real-Time AMI Dashboard (Updated Product Goal)

## Core Objective
Provide a single self-contained Go binary that connects to an Asterisk/AllStar node (AMI), ingests real-time events, maintains authoritative in-memory link & TX state, persists cumulative per-link transmit statistics, and serves an anonymous-friendly Vue dashboard plus admin auth features over WebSockets + JSON APIs.

## Implemented (Current State)
### Backend / Realtime
* AMI connector with login, automatic reconnect, retry backoff.
* Event-driven state (no polling needed when AMI emits RPT_* / VarSet events) with optional legacy poller toggle.
* Parsing of RPT_LINKS, RPT_ALINKS (with keyed TX attribution), RPT_TXKEYED, RPT_RXKEYED, FullyBooted (uptime), plus VarSet fallbacks.
* Incremental link diff channels: LINK_ADDED / LINK_REMOVED.
* Per-link TX edge detection (START/STOP) with batching (LINK_TX_BATCH) to reduce WS chattyness.
* Heartbeat loop broadcasting periodic STATUS_UPDATE snapshots (prevents perpetual "Waiting for data").
* Persistence (SQLite WAL) of per-link totals + last TX edges + connected_since for seamless restarts.
* Seed logic restoring link stats into in-memory state at boot.
* Repository layer (`link_stats`) with Upsert and query filters.

### API
* Auth: register, login (HMAC-signed token), role-based endpoints (admin summary, /api/me).
* Public/anonymous stats when enabled: `/api/link-stats`, `/api/link-stats/top` (sorting, limiting, filtering: node=, since=, relative since like -1h, tx rate mode).
* Rate limiting: auth RPM + public stats RPM.

### WebSocket Protocol
Message types: STATUS_UPDATE, TALKER_EVENT, LINK_ADDED, LINK_REMOVED, LINK_TX_BATCH (replaces high-frequency LINK_TX), (legacy LINK_TX still supported internally), plus periodic heartbeats.

### Frontend (Vue Dashboard)
* Anonymous mode banner + real-time status, links, talker log, top links list.
* Admin login drawer (JWT stored locally for now), upgrades WS connection (reconnect) & enables admin-only controls (manual refresh, future management panes).
* Live per-link columns: connected since, last heard, current TX flag, elapsed current TX, cumulative TX seconds, session TX percentage.
* Blink highlight for actively transmitting links.
* Timeout fallback message if no status after 5s.

### Testing
* Handler/auth integration tests (bootstrap roles, negative cases, duplicates).
* Core state tests: link add/remove diffs, ALINKS TX attribution, TX stop logic.
* Link stats endpoint filter tests (since relative & absolute, top endpoint path).

## In Progress / Near-Term Targets
1. Admin UI panel (user management, system metrics) gated behind role.
2. Talker analytics (top N talkers, rolling TX heatmap) derived from TX events.
3. Enhanced persistence: per-link session segments (start/end records) if deeper analytics needed.
4. Configurable TX batch window (env) & optional raw event streaming mode for power users.
5. Snapshot compression or diffing if state scales (multi-node future).
6. Structured logging upgrade (zap already imported—standardize fields, request IDs everywhere).
7. CI pipeline & Docker multi-stage build (static binary).

## Deferred / Future
* Multi-node federation / aggregation.
* Prometheus metrics exporter (TX durations, active link counts).
* RBAC granularity beyond admin/user (e.g., operator role with limited actions).
* Command channel (DTMF / *73 resync trigger) reintroduction with rate limiting & audit log.
* Historical analytics store (rollup of daily TX per link / per node).
* Notification hooks (webhook or email) for specific thresholds (excessive TX, link churn).

## Key Risks & Mitigations (Updated)
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Quiet systems show no data | User confusion | HeartbeatLoop + fallback UI banner |
| Burst TX events overwhelm clients | UI lag | Batch window (100ms) + optional configurability |
| Reconnect mid-session loses TX continuity | Under-counting | Persist last_tx_start/last_tx_end + total seconds; resume with seeded state |
| Schema drift over iterations | Migration errors | Idempotent ALTER migration logic + versioned notes |
| Public scraping of stats | Load / abuse | Public stats RPM limiter + optional disable flag |

## Success Criteria (Phase: Public Alpha)
* Anonymous dashboard shows live status within 5s on a quiet node.
* TX events reflected in UI with cumulative totals accuracy ±1s over a 10‑minute test talk window.
* Restart of service preserves >99% of cumulative TX time (no reset) for existing links.
* Under synthetic 50 link add/remove churn + TX bursts: no goroutine leak; memory stable (<80MB after 30m).
* API filters (since, node, sort, limit) behave deterministically (covered by tests).

## Current Configuration Matrix
| Env Var | Default | Notes |
|---------|---------|-------|
| PORT | 8080 | HTTP listen |
| JWT_SECRET | dev-secret-change-me | Replace in prod |
| TOKEN_TTL_SECONDS | 86400 | Auth token lifetime |
| AUTH_RPM | 60 | Auth endpoints rate limit |
| PUBLIC_STATS_RPM | 120 | Anonymous stats rate limit |
| AMI_ENABLED | false | Toggle connector |
| AMI_HOST | 127.0.0.1 | AMI host |
| AMI_PORT | 5038 | AMI port |
| AMI_USERNAME | admin | Manager user |
| AMI_PASSWORD | change-me | Manager password (keep secret) |
| AMI_EVENTS | on | Event preference |
| AMI_RETRY_INTERVAL | 15s | Reconnect base |
| AMI_RETRY_MAX | 60s | Backoff cap |
| DISABLE_LINK_POLLER | false | Skip legacy poller |
| ALLOW_ANON_DASHBOARD | true | Public view switch |

## Next Actionable Steps (Short List)
1. Add admin panel Vue component (users list stub) – fetch `/api/admin/summary` when authed.
2. Add configurable batch window env (LINK_TX_BATCH_MS) & wire to main.
3. Create lightweight CI (lint + `go test ./...` + Vue build).
4. Expand tests: TX percentage accuracy over synthetic timeline.
5. Log field standardization (correlation IDs, component tags).

---
Document owner: Realtime / Platform
Version: 0.6.0-wip
