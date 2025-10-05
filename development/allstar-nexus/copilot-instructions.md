# Allstar Nexus – Contributor / AI Assistant Guidance

This file keeps assistants and contributors aligned with the CURRENT (not historical) architecture & objectives.

## Active Architecture Snapshot
| Layer | Technology | Notes |
|-------|------------|-------|
| Backend | Go 1.25.x | Single binary, embeds built static assets (Vue + legacy Next export fallback) |
| Realtime | AMI TCP connector | Login, auto-reconnect, event-driven state (RPT_* / VarSet) |
| Persistence | SQLite (modernc.org/sqlite) | WAL mode, link_stats table (per-link TX totals & timestamps) |
| Frontend | Vue 3 + Vite | Anonymous dashboard + admin login drawer, WS-driven updates |
| WebSockets | github.com/coder/websocket | Message envelope (MessageType, Data, Timestamp) |

Legacy Next.js directory remains for earlier prototype – Vue dashboard is preferred and auto-selected when build artifacts exist.

## Implemented Feature Matrix
* Anonymous read-only dashboard (toggle via `ALLOW_ANON_DASHBOARD`).
* Admin auth: register/login (first user becomes superadmin), role-based endpoints.
* Real-time link tracking & TX batching (per-link START/STOP grouped as LINK_TX_BATCH every ~100ms).
* Persist & restore per-link total TX seconds and connected_since values.
* Filtering & analytics endpoints: `/api/link-stats` (since, node list, sort, limit) and `/api/link-stats/top` (mode=tx_seconds|tx_rate).
* Heartbeat loop pushes periodic STATUS_UPDATE so quiet systems still render.
* Rate limiting: auth RPM + public stats RPM.
* Test coverage for auth flows, link stats filters, core state transitions.

## In-Flight Goals (Keep PRs Small)
1. Admin panel UI (user / role view, system stats).
2. Configurable TX batch window (env) + optional raw event passthrough mode.
3. Additional analytics (top N by rate, rolling 1h TX distribution).
4. CI pipeline (lint / tests / build) + Docker image.
5. Structured logging standardization (zap fields: component, req_id, user, link_count, tx_active).

## Coding Conventions
* Prefer small, single-purpose patches.
* Avoid adding external deps unless necessary (esp. for simple parsing / utilities).
* Keep WebSocket payloads minimal; batch where high frequency (already done for TX edges).
* All exported structs in `repository` & `api` should remain JSON-safe (primitive + time.Time pointer where needed).
* Migrations: additive & idempotent; prefer guarded `ALTER TABLE` statements.

## Testing Guidelines
* Put integration-style HTTP tests in `backend/tests` (mirrors buildMux logic).
* Core realtime logic: extend `internal/core/state_test.go` with synthetic AMI frames.
* Prefer table-driven tests; ensure at least one negative / edge case.

## Security / Privacy Notes
* Do NOT commit secrets (respect `.gitignore` – `.env*`, databases, pcap files, etc.).
* JWT secret must be overridden in non-dev deployments.
* Public endpoints should remain read-only; any mutation (future commands) must require auth + rate limit.

## When Adding Features
1. Update this file & `PRODUCT_GOAL.md` when architectural scope changes.
2. Add config keys to `backend/config/config.go` with sane defaults & doc them.
3. Provide minimal tests (happy + one edge) in same PR.
4. Keep frontend changes resilient to missing fields (defensive null checks).

## WebSocket Message Types (Current)
| Type | Description |
|------|-------------|
| STATUS_UPDATE | Full NodeState snapshot (periodic + on change) |
| TALKER_EVENT | TX start/stop edges (node-level) |
| LINK_ADDED | Array of newly observed links (LinkInfo) |
| LINK_REMOVED | Array of removed link node IDs |
| LINK_TX_BATCH | Array of per-link TX START/STOP events within batch window |

## Avoid / Deprecated
* Reintroducing heavy polling (event-driven now). Poller left only as optional fallback.
* Storing large historical logs in memory (persist aggregates instead).

## Quick Dev Workflow
```
# Build Vue dashboard
cd vue-dashboard && npm run build
cd .. && go run .
# Run tests
go test ./...
```

## Minimal PR Checklist
* [ ] Code builds (`go build ./...` & Vue build if frontend touched)
* [ ] Tests updated / added & passing
* [ ] `PRODUCT_GOAL.md` or this file updated if scope changed
* [ ] No secrets / *.db / .env leaked
* [ ] Rate limiting considered for any new public endpoint

---
Version: 0.6.0-wip

## Phased Plan

### Phase 1: Project Setup ✅ COMPLETED

1.  ✅ **Created `copilot-instructions.md`**: Project planning and instructions
2.  ✅ **Initialized Git and GitHub Repository**: Repository created at `github.com/dbehnke/allstar-nexus`
3.  ✅ **Scaffolded the Project**: 
    - Next.js application with TypeScript, ESLint, and Tailwind CSS
    - Go backend with embedded frontend serving
    - Monorepo structure with separate frontend/backend directories

### Phase 2: Core Feature Development (IN PROGRESS)

Simplified product scope adopted:
* Public dashboard (read-only) accessible without auth (endpoints pending)
* Superadmin (first registered user) with full control
* Optional early second admin if created immediately after first
* All subsequent users default to role `user`

Completed so far in Phase 2:
1. Basic REST API structure (`/api/auth/register`, `/api/auth/login`, `/api/me`, `/api/admin/summary`, `/api/health`).
2. SQLite persistence with auto-migration & role column.
3. Password hashing via bcrypt.
4. Lightweight HMAC-signed JWT-like tokens (email, role, exp, signature) replacing insecure placeholder for new flows.
5. Bootstrap logic: first user becomes `superadmin` automatically; limited second admin capability.
6. Middleware for auth & role guard; public dashboard summary placeholder.
7. Standardized JSON response envelope + structured error codes.
8. Email normalization (lowercase + trim) on register/login.
9. Configurable token TTL via `TOKEN_TTL_SECONDS` env var.
10. Graceful shutdown handling (SIGINT/SIGTERM) with timeout.
11. Initial test suite (JWT unit, handlers integration, token expiry simulation).
12. Duplicate email detection returns HTTP 409.
13. Basic per-IP rate limiting for auth endpoints (`AUTH_RPM`).
14. Public dashboard summary endpoint (placeholder) + test coverage.
15. Negative auth tests (malformed / missing / corrupted token cases).
16. Logging middleware with request IDs & panic recovery.
17. Makefile for common tasks (build, run, test).

Remaining Phase 2 goals (updated):
1. Enrich public dashboard data beyond role counts (e.g., user growth, recent signups).
2. Decide on adopting standard JWT library vs. keeping custom token; if staying custom add refresh/rotation.
3. Frontend auth UI (login form, token storage strategy, admin summary view, conditional rendering).
4. Additional tests: repository layer unit tests (IN PROGRESS - added basic CRUD & counts), rate limiter edge cases (refill correctness), dashboard content validation when enriched.
5. Password policy & validation (minimum length, reject common weak passwords, maybe zxcvbn scoring later).
6. Structured logging upgrade (replace basic log.Printf with slog or zap + JSON output & correlation fields).
7. Security review (token storage strategy, potential CSRF mitigation if moving to cookies, brute-force protections).
8. Add developer runbook & operational docs (logging, rebuilding embedded frontend, env var guide).
9. Optional: Introduce build tags or version injection (ldflags) for build metadata.
10. Optional: Basic benchmark for token generation & login path.

### Phase 3: Advanced Features (Planned / Deferred)

1.  **Real-time Communication**: WebSocket support
2.  **File Uploads**: Handle file uploads and storage
3.  **Background Jobs**: Implement job queue system
4.  **Caching**: Add Redis or in-memory caching

### Phase 4: Deployment and DevOps (Planned)

1.  **CI/CD Pipeline**: GitHub Actions for automated testing and deployment
2.  **Docker**: Containerization setup
3.  **Monitoring**: Application monitoring and logging
4.  **Documentation**: API documentation and user guides

## Current Project Structure

```
allstar-nexus/
├── main.go                  # Main application entry point
├── go.mod                   # Go module definition
├── frontend/                # Next.js application
│   ├── src/
│   │   └── app/            # Next.js App Router pages
│   ├── public/             # Static assets
│   ├── package.json
│   └── next.config.ts      # Next.js config (static export enabled)
└── backend/                # Go backend packages
    └── server/             # Server utilities
        └── frontend.go     # Frontend serving utilities
```

## Development Workflow

1. **Frontend Development**: Run `npm run dev` in `frontend/` directory
2. **Build Frontend**: Run `npm run build` in `frontend/` directory
3. **Run Full Stack**: `go run .` from root directory (ensure `npm run build` first for updated embedded assets)
4. **Build for Production**:
    - `npm run build` (produces `frontend/out`)
    - `go build -o allstar-nexus .`

## Endpoint Reference (Current)

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| /api/health | GET | none | Health probe |
| /api/auth/register | POST | none | Register new user (bootstrap superadmin) |
| /api/auth/login | POST | none | Obtain JWT-like token + role |
| /api/me | GET | Bearer | Return current user info |
| /api/admin/summary | GET | Bearer (admin/superadmin) | Role counts summary |
| /api/dashboard/summary | GET | none | Public summary (placeholder) |

Planned additions: `/api/dashboard/summary`, more admin management endpoints.

## Auth Token Format (Temporary Minimal Scheme)

Token structure (pipe-delimited): `base64(email)|base64(role)|expUnix|base64(hmacSha256(parts[0..2], secret))`

Validation steps:
1. Split into 4 parts.
2. Recompute signature with shared secret.
3. Compare constant-time; reject if mismatch or expired.
4. Load user & ensure stored role matches token role.

Configurable TTL:
* Environment variable `TOKEN_TTL_SECONDS` (default 86400 = 24h) controls issued token lifetime.
* Auth rate limiting: `AUTH_RPM` (default 60) requests per minute per IP for register/login.

Future upgrade options:
* Standard JWT library with HS256 & claims (iss, iat, exp, sub, role).
* Short-lived access token + refresh token rotation.
* Key rotation via JWKS if needed.

## Security Considerations (Short Term)
* Current token lacks standard JWT header/claims – acceptable for early internal prototype only.
* No rate limiting yet (risk: credential stuffing / brute force).
* No CSRF concern for pure bearer tokens if stored in memory; avoid localStorage for production (recommend httpOnly cookie strategy later).
* Password policy minimal; length & strength heuristics pending.

## Testing Priorities
Implemented:
* JWT generation & bad signature tests
* Handler integration tests (bootstrap roles, access control, email normalization)
* Short TTL expiry test

Pending / Next:
1. Repo duplicate email specific test.
2. Public dashboard summary test.
3. Negative auth cases (missing token, malformed token variants).
4. Performance baseline (simple benchmark for token generation optional).

## Cleanup / Debt Tracker
* (Done) Removed unused server utility file.
* (Done) Consolidated error responses.
* (Done) Replaced magic role strings with constants.
* Introduce context-aware structured logger (pending).

## Next Immediate Steps
1. Implement middleware abstraction for auth & role guard.
2. Add public dashboard endpoint skeleton.
3. Standardize response format.
4. Begin test suite (start with auth + repository).

