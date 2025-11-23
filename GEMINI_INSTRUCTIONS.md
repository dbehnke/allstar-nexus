# Allstar Nexus - Gemini Agent Instructions

> **Note to Agent**: Read this file at the start of every session to understand the project context, architecture, and current status.

## 1. Project Overview

**Allstar Nexus** is a modern, web-based interface for monitoring and managing AllStarLink nodes (Asterisk-based amateur radio systems). It replaces legacy PHP/Perl scripts (Supermon) with a single-binary Go application and a reactive Vue 3 frontend.

## 2. Architecture Stack

| Layer | Technology | Details |
|-------|------------|---------|
| **Backend** | Go 1.25+ | Single binary, embeds frontend assets. Uses `net/http` standard lib. |
| **Database** | SQLite | `modernc.org/sqlite` (CGO-free). WAL mode enabled. |
| **Realtime** | WebSockets | `github.com/coder/websocket`. Event-driven updates (no polling). |
| **AMI** | TCP | Custom AMI connector for Asterisk Manager Interface. |
| **Frontend** | Vue 3 | Vite, Composition API (`<script setup>`), TypeScript. |
| **State** | Pinia | Centralized state management for nodes, links, and auth. |
| **Styling** | Tailwind CSS | Dark mode default. Responsive design. |

## 3. Current Project Status (Phases 1-4 Complete)

The project has completed the core monitoring and AMI integration features.

### ✅ Completed Features

- **Public Dashboard**: Real-time node monitoring, RPT stats, Voter display.
- **Authentication**: JWT-based auth.
- **Core AMI Enhancement**: `RptStatus`, `XStat`, `SawStat` support.
- **State Management**: Connection tracking, RX/TX keying detection, Last Heard tracking.
- **WebSocket Events**: Real-time broadcasting of node state and link changes.
- **Frontend Updates**: COS/PTT indicators, Link Mode badges, Last Heard sorting.

### 🔄 In Progress / Next Steps (Phase 5)

- **Testing & Refinement**:
  - Test with real AllStar node hardware.
  - Validate voter commands.
  - Performance testing.
  - Bug fixes.

### ❌ Pending / Not Yet Implemented

- **Admin Panel**: Configuration editor, Command execution, User management (Planned but not fully merged on this branch).
- **Node Type Detection**: Distinguishing AllStar/IRLP/EchoLink.

## 4. Coding Conventions

### Backend (Go)

- **Structure**:
  - `cmd/`: Entry points.
  - `internal/`: Private application code (AMI, Core logic).
  - `backend/api/`: HTTP handlers.
  - `backend/repository/`: Database access.
- **Style**:
  - Prefer standard library over external deps where reasonable.
  - Use `context` for cancellation and timeouts.
  - **Error Handling**: Return wrapped errors, handle gracefully.
  - **Concurrency**: Use channels and goroutines for AMI events.

### Frontend (Vue 3)

- **Components**: Use Single File Components (`.vue`) with `<script setup lang="ts">`.
- **State**: Use Pinia stores (`frontend/src/stores/`).
- **Styling**: Utility-first Tailwind CSS. Avoid custom CSS unless necessary.
- **API**: Use `fetch` or a lightweight wrapper. Handle 401/403 errors globally.

### Testing

- **Go**: Table-driven tests. Place in `_test.go` files next to code.
- **Mocks**: Use interfaces for AMI and Repository layers to facilitate mocking.

## 5. Agent Workflow Guidelines

1. **Task Boundary**: Always use `task_boundary` to track progress.
2. **File Edits**:
    - Use `view_file` to understand context before editing.
    - Use `replace_file_content` for single blocks, `multi_replace_file_content` for scattered edits.
    - **Verify**: After editing, verify the build (`go build .` or `npm run build`).
    - **Test**: ALWAYS run frontend (`cd frontend && npm test`) and backend (`go test ./...`) tests before adding, committing, or pushing changes.
    - **Audit**: Run `npm audit` in `frontend/` to check for vulnerabilities before pushing.
    - **CI Checks**: After pushing, run `gh pr checks` to monitor CI status and resolve any failures immediately.
3. **Safety**:
    - **NEVER** commit secrets (API keys, passwords).
    - **NEVER** delete the database (`allstar.db`) without user permission.
    - Respect `.gitignore`.

## 6. Key Directories

- `/`: Root (Go `main.go`, `go.mod`).
- `frontend/`: Vue application.
- `backend/`: Go packages.
- `internal/ami/`: Asterisk Manager Interface logic.
- `data/`: SQLite DB and config backups (gitignored).

## 7. Common Commands

```bash
# Run Full Stack (Dev)
# Terminal 1:
cd frontend && npm run dev
# Terminal 2:
go run .

# Build for Production
cd frontend && npm run build
cd ..
go build -o allstar-nexus .
```
