# Allstar Nexus Administration Panel - Implementation Plan

## Executive Summary

This document outlines the comprehensive plan to add web-based administration capabilities to allstar-nexus, modernizing and extending functionality inspired by Supermon's PHP-based control panel. The implementation will leverage our existing Vue 3 + Go stack to create a powerful, secure, and user-friendly administration interface.

## Background

Supermon (in `tmp/supermon`) provides web-based administration features including:
- Control panel for sending Asterisk commands
- Configuration file viewing/editing
- Node linking/unlinking via web interface
- Real-time status monitoring

Our goal is to build a modern equivalent that:
- Uses our Vue 3 + Go architecture
- Provides superior UX with real-time updates
- Includes comprehensive security and audit logging
- Extends capabilities beyond Supermon's feature set
- Integrates seamlessly with our existing AMI connector and authentication system

## Core Features

### 1. Configuration File Editor (Priority: HIGH)

**Capabilities:**
- View and edit `config.yaml` through web interface
- Real-time YAML syntax validation and linting
- Visual feedback for configuration errors
- Preview changes before applying
- Automatic backup creation before each edit
- Backup management (list, restore, download)
- Service restart/reload after config changes
- Validation against config schema

**Technical Implementation:**
- Backend: Config read/write with file locking
- Frontend: Monaco Editor or CodeMirror for YAML editing
- Use existing `config.Validate()` function for validation
- Store backups in `data/config-backups/` with timestamps

**Safety Features:**
- Require confirmation before saving
- Automatic rollback on validation failure
- Config history with diff view
- Dry-run mode to test without applying

### 2. Asterisk Command Interface (Priority: HIGH)

**Capabilities:**
- Send arbitrary AMI commands to Asterisk
- Pre-defined command buttons for common operations:
  - Link/unlink nodes
  - Restart modules
  - Show status
  - Originate calls
  - Custom macros
- Real-time command output display
- Command history log
- Command templates/favorites
- Multi-node support (execute on specific node)

**Technical Implementation:**
- Extend `internal/ami/connector.go` with synchronous action methods
- New backend package `backend/admin/command_executor.go`
- WebSocket or Server-Sent Events for real-time output
- Command queueing to prevent concurrent conflicts
- Timeout handling for long-running commands

**Safety Features:**
- Dangerous commands require explicit confirmation
- Configurable command whitelist/blacklist
- Rate limiting per user
- Full audit trail of executed commands
- Read-only mode for non-superadmin users

### 3. Node Management Panel (Priority: HIGH)

**Capabilities:**
- Visual node connection manager
- Quick link/unlink interface with favorites
- Node status overview (connected, mode, direction)
- Ban/allow list management
- Connection monitoring and statistics
- Favorite node lists per user
- Bulk operations (connect/disconnect multiple nodes)
- Schedule recurring connections

**Technical Implementation:**
- New API endpoints for node operations
- Integration with existing AMI connector
- Persist favorites in user preferences or database
- Real-time status updates via WebSocket
- Node metadata from astdb integration

**UI Components:**
- Interactive node tree/graph
- Search/filter connected nodes
- Drag-and-drop favorites
- One-click connect/disconnect buttons
- Color-coded status indicators

### 4. User Management (Priority: MEDIUM)

**Capabilities:**
- Create, read, update, delete users
- Assign roles (superadmin, admin, user)
- Password reset/change
- Force password change on next login
- Enable/disable user accounts
- View user activity/audit logs
- Bulk user operations

**Technical Implementation:**
- Extend existing `backend/repository/user_repo.go`
- New admin API endpoints
- Password complexity requirements
- Email notifications (optional)
- Integration with existing JWT auth

**UI Features:**
- User list with search/filter
- Inline editing
- Role badge indicators
- Last login timestamp
- Activity timeline per user

### 5. Real-Time Log Viewer (Priority: MEDIUM)

**Capabilities:**
- Stream Asterisk logs in real-time
- Stream allstar-nexus application logs
- Filter by log level, source, time range
- Search within logs
- Download log segments
- Tail mode (follow new entries)
- Syntax highlighting for stack traces

**Technical Implementation:**
- Server-Sent Events or WebSocket for streaming
- Backend log parsing and filtering
- Configurable log retention
- Log rotation awareness
- Support for journald or file-based logs

**UI Features:**
- Virtual scrolling for large logs
- Pinnable log entries
- Color-coded severity levels
- Full-text search with regex support
- Export to file

### 6. System Status Dashboard (Priority: MEDIUM)

**Capabilities:**
- CPU, memory, disk usage monitoring
- Asterisk process status
- Database statistics (size, query performance)
- Network statistics
- Service uptime
- Active connections count
- Recent error summary

**Technical Implementation:**
- Collect system metrics via Go's `syscall` or `gopsutil`
- Periodic polling or push-based updates
- Time-series data for historical graphs
- Alert thresholds (optional)

**UI Features:**
- Gauge charts for resource usage
- Line graphs for trends
- Status indicators (green/yellow/red)
- Quick links to detailed views

## Technical Architecture

### Backend Components

#### 1. API Layer: `backend/api/admin.go`

```go
// Admin API handler struct
type AdminAPI struct {
    ConfigManager  *admin.ConfigManager
    CommandExec    *admin.CommandExecutor
    UserRepo       *repository.UserRepo
    AMIConnector   *ami.Connector
    AuditLogger    *admin.AuditLogger
}

// Endpoints (all require superadmin role):
// - POST /api/admin/config/read
// - POST /api/admin/config/update
// - POST /api/admin/config/backup
// - GET  /api/admin/config/backups
// - POST /api/admin/config/restore/:id
// - POST /api/admin/ami/command
// - POST /api/admin/ami/link
// - POST /api/admin/ami/unlink
// - GET  /api/admin/users
// - POST /api/admin/users
// - PUT  /api/admin/users/:id
// - DELETE /api/admin/users/:id
// - GET  /api/admin/logs/stream
// - GET  /api/admin/system/status
```

#### 2. Config Manager: `backend/admin/config_manager.go`

Handles safe configuration file operations:
- Read current config with comments preserved
- Validate changes before writing
- Create automatic backups
- Restore from backup
- Diff generation between versions
- File locking to prevent concurrent edits

#### 3. Command Executor: `backend/admin/command_executor.go`

Executes AMI commands with safety checks:
- Command validation and sanitization
- Timeout handling
- Output buffering and streaming
- Error handling and recovery
- Audit logging

#### 4. Audit Logger: `backend/admin/audit_logger.go`

Records all admin actions:
- Who performed the action
- What was changed
- When it occurred
- Result (success/failure)
- IP address and user agent
- Queryable audit trail

#### 5. Extend AMI Connector: `internal/ami/actions.go`

Add synchronous AMI action methods:
```go
func (c *Connector) ExecuteAction(action string, params map[string]string) (*Response, error)
func (c *Connector) LinkNode(nodeID int, permanent bool) error
func (c *Connector) UnlinkNode(nodeID int) error
func (c *Connector) OriginateCall(channel, context, exten string) error
// ... more action methods
```

### Frontend Components

#### Views (in `frontend/src/views/Admin/`)

1. **AdminDashboard.vue** - Landing page with overview cards:
   - System health summary
   - Recent admin actions
   - Quick action buttons
   - Active alerts/warnings

2. **ConfigEditor.vue** - YAML editor with features:
   - Monaco/CodeMirror integration
   - Syntax highlighting and validation
   - Side-by-side diff view
   - Backup management panel
   - Restart service button

3. **CommandPanel.vue** - Command execution interface:
   - Command input with autocomplete
   - Pre-defined command buttons
   - Real-time output terminal
   - Command history sidebar
   - Multi-tab support for concurrent commands

4. **NodeControl.vue** - Node management:
   - Connected nodes list with actions
   - Favorites/quick-connect panel
   - Node search and filter
   - Bulk operations toolbar
   - Connection scheduler (future)

5. **UserManager.vue** - User CRUD interface:
   - User table with inline editing
   - Role assignment dropdown
   - Password reset modal
   - User activity timeline
   - Bulk user import/export (future)

6. **LogViewer.vue** - Real-time log streaming:
   - Log source selector (Asterisk/Nexus)
   - Filter controls (level, search, time)
   - Virtual scrolling list
   - Download button
   - Tail mode toggle

7. **SystemStatus.vue** - Monitoring dashboard:
   - Resource usage gauges
   - Service status indicators
   - Historical graphs
   - Recent errors list

#### Shared Components (in `frontend/src/components/`)

1. **CommandButton.vue** - Reusable command executor button:
   - Props: label, command, requireConfirm
   - Emits: success, error
   - Shows loading spinner during execution
   - Toast notification on result

2. **ConfirmModal.vue** - Generic confirmation dialog:
   - Props: message, dangerLevel (info/warning/danger)
   - Emits: confirm, cancel
   - Customizable action button text

3. **AuditLogViewer.vue** - Audit trail component:
   - Filterable table of admin actions
   - Time-based search
   - User filter
   - Export functionality

### API Endpoint Specifications

All admin endpoints require `Authorization: Bearer <token>` with `superadmin` role.

#### Config Management

**POST `/api/admin/config/read`**
- Response: `{ ok: true, data: { content: "yaml string", path: "config.yaml" } }`

**POST `/api/admin/config/update`**
- Request: `{ content: "new yaml", create_backup: true }`
- Response: `{ ok: true, data: { backup_id: "2025-11-01-123456" } }`

**POST `/api/admin/config/backup`**
- Response: `{ ok: true, data: { backup_id: "...", path: "..." } }`

**GET `/api/admin/config/backups`**
- Response: `{ ok: true, data: [{ id, timestamp, size, comment }] }`

**POST `/api/admin/config/restore/:id`**
- Response: `{ ok: true, data: { restored: true } }`

**GET `/api/admin/config/diff`**
- Query: `?from=backup_id&to=current`
- Response: `{ ok: true, data: { diff: "unified diff string" } }`

#### AMI Commands

**POST `/api/admin/ami/command`**
- Request: `{ command: "rpt fun 594950 *3594950", node_id: 594950 }`
- Response: `{ ok: true, data: { output: "...", success: true } }`

**POST `/api/admin/ami/link`**
- Request: `{ node_id: 594950, target: 1234, permanent: false }`
- Response: `{ ok: true, data: { linked: true } }`

**POST `/api/admin/ami/unlink`**
- Request: `{ node_id: 594950, target: 1234 }`
- Response: `{ ok: true, data: { unlinked: true } }`

#### User Management

**GET `/api/admin/users`**
- Response: `{ ok: true, data: [{ id, email, role, created_at, last_login }] }`

**POST `/api/admin/users`**
- Request: `{ email, password, role }`
- Response: `{ ok: true, data: { id, email, role } }`

**PUT `/api/admin/users/:id`**
- Request: `{ email?, role?, password? }`
- Response: `{ ok: true, data: { id, email, role } }`

**DELETE `/api/admin/users/:id`**
- Response: `{ ok: true, data: { deleted: true } }`

**POST `/api/admin/users/:id/reset-password`**
- Request: `{ new_password, force_change: false }`
- Response: `{ ok: true, data: { reset: true } }`

#### Logs and Monitoring

**GET `/api/admin/logs/stream`**
- Query: `?source=asterisk&level=info&tail=true`
- Response: Server-Sent Events stream of log lines

**GET `/api/admin/system/status`**
- Response: `{ ok: true, data: { cpu, memory, disk, uptime, asterisk_status } }`

**GET `/api/admin/audit`**
- Query: `?user=email&since=timestamp&action=type`
- Response: `{ ok: true, data: [{ timestamp, user, action, details, result }] }`

## Security Implementation

### Authentication & Authorization

1. **Role-Based Access Control**
   - Most admin features require `superadmin` role
   - Some read-only features accessible to `admin` role
   - Middleware: `RequireRole("superadmin")`

2. **Audit Logging**
   - All admin actions logged to database
   - Include: user, timestamp, IP, action, result
   - Immutable audit log (append-only)
   - Queryable via admin UI

3. **Rate Limiting**
   - Command execution: 10 per minute per user
   - Config updates: 5 per hour per user
   - API requests: 100 per minute per user

4. **Input Validation**
   - Sanitize all command inputs
   - Validate config YAML before saving
   - Prevent path traversal in file operations
   - Limit file upload sizes

5. **Dangerous Operations**
   - Require explicit confirmation (frontend)
   - Secondary validation (backend)
   - Dry-run mode where applicable
   - Automatic backups before destructive changes

### CSRF Protection

- Token-based CSRF protection for state-changing operations
- Double-submit cookie pattern or synchronizer tokens
- SameSite cookie attributes

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

**Backend:**
- [ ] Create `backend/admin/` package structure
- [ ] Implement audit logging system
- [ ] Add superadmin middleware
- [ ] Create admin API skeleton in `backend/api/admin.go`
- [ ] Add admin routes to `main.go`

**Frontend:**
- [ ] Create `frontend/src/views/Admin/` directory
- [ ] Implement AdminDashboard.vue skeleton
- [ ] Add "Admin" nav link (conditional on role)
- [ ] Set up admin routing with role guards

**Testing:**
- [ ] Unit tests for middleware
- [ ] Integration tests for audit logging

### Phase 2: Config Management (Week 3)

**Backend:**
- [ ] Implement ConfigManager with backup logic
- [ ] Config read/update endpoints
- [ ] Backup list/restore endpoints
- [ ] Config diff generation
- [ ] Validation integration

**Frontend:**
- [ ] Integrate Monaco or CodeMirror editor
- [ ] Build ConfigEditor.vue with YAML validation
- [ ] Backup management UI
- [ ] Diff viewer component

**Testing:**
- [ ] Config backup/restore tests
- [ ] Validation error handling tests

### Phase 3: AMI Command Interface (Week 4-5)

**Backend:**
- [ ] Extend AMI connector with action methods
- [ ] Implement CommandExecutor with safety checks
- [ ] Command execution endpoint
- [ ] Link/unlink node endpoints
- [ ] Command audit logging

**Frontend:**
- [ ] Build CommandPanel.vue with terminal output
- [ ] Implement CommandButton.vue component
- [ ] Command history sidebar
- [ ] Pre-defined command templates
- [ ] Real-time output streaming (SSE or WS)

**Testing:**
- [ ] AMI command execution tests
- [ ] Timeout and error handling tests
- [ ] Command sanitization tests

### Phase 4: Node Management & Users (Week 6)

**Backend:**
- [ ] Node control endpoints (link/unlink/status)
- [ ] User management CRUD endpoints
- [ ] Password reset functionality
- [ ] User activity logging

**Frontend:**
- [ ] NodeControl.vue with favorites
- [ ] UserManager.vue with inline editing
- [ ] User role assignment UI
- [ ] Password reset modal

**Testing:**
- [ ] User CRUD operation tests
- [ ] Role permission tests

### Phase 5: Monitoring & Logs (Week 7)

**Backend:**
- [ ] System status collection
- [ ] Log streaming endpoint (SSE)
- [ ] Log filtering and search
- [ ] Audit log query endpoint

**Frontend:**
- [ ] SystemStatus.vue dashboard
- [ ] LogViewer.vue with streaming
- [ ] AuditLogViewer.vue component
- [ ] Virtual scrolling for logs

**Testing:**
- [ ] Log streaming tests
- [ ] System metrics collection tests

### Phase 6: Polish & Documentation (Week 8)

- [ ] Error handling improvements
- [ ] Loading states and spinners
- [ ] Confirmation modals for dangerous actions
- [ ] Toast notifications
- [ ] Responsive design adjustments
- [ ] Accessibility improvements (ARIA labels, keyboard nav)
- [ ] Write admin user guide
- [ ] API documentation
- [ ] Video tutorials (optional)

## File Structure

```
allstar-nexus/
├── backend/
│   ├── admin/                     # NEW
│   │   ├── audit_logger.go        # Audit trail recording
│   │   ├── config_manager.go      # Config file operations
│   │   └── command_executor.go    # AMI command execution
│   ├── api/
│   │   ├── admin.go               # NEW - Admin API handlers
│   │   └── ...
│   ├── middleware/
│   │   └── middleware.go          # MODIFIED - Add RequireSuperAdmin
│   └── models/
│       └── audit_log.go           # NEW - Audit log model
├── frontend/
│   └── src/
│       ├── views/
│       │   └── Admin/             # NEW
│       │       ├── AdminDashboard.vue
│       │       ├── ConfigEditor.vue
│       │       ├── CommandPanel.vue
│       │       ├── NodeControl.vue
│       │       ├── UserManager.vue
│       │       ├── LogViewer.vue
│       │       └── SystemStatus.vue
│       ├── components/
│       │   ├── CommandButton.vue  # NEW
│       │   ├── ConfirmModal.vue   # NEW
│       │   └── AuditLogViewer.vue # NEW
│       ├── App.vue                # MODIFIED - Add admin nav link
│       └── router/
│           └── index.js           # MODIFIED - Add admin routes
├── internal/
│   └── ami/
│       ├── actions.go             # NEW - Synchronous AMI actions
│       └── connector.go           # MODIFIED - Extend capabilities
├── data/
│   └── config-backups/            # NEW - Config backup storage
└── ADMIN-PLAN.md                  # This file
```

## UI/UX Design Principles

1. **Consistency**: Match existing allstar-nexus design language
2. **Safety**: Confirm dangerous operations, show clear warnings
3. **Feedback**: Immediate visual feedback for all actions
4. **Efficiency**: Quick access to common tasks
5. **Accessibility**: Keyboard navigation, screen reader support
6. **Responsiveness**: Work on mobile devices (where practical)
7. **Dark Theme**: Full dark mode support matching main app

## Success Criteria

- [ ] Config can be edited and validated through web UI
- [ ] Asterisk commands can be executed safely with audit trail
- [ ] Nodes can be linked/unlinked via web interface
- [ ] Users can be managed (CRUD operations)
- [ ] Logs can be viewed in real-time
- [ ] System status visible on dashboard
- [ ] All admin actions are logged
- [ ] Zero security vulnerabilities in admin panel
- [ ] Mobile-responsive design
- [ ] Comprehensive test coverage (>80%)
- [ ] Documentation complete

## Future Enhancements

- **Scheduled Tasks**: Cron-like scheduler for recurring node connections
- **Alert System**: Email/SMS/Push notifications for critical events
- **Multi-Server Management**: Manage multiple Asterisk servers from one UI
- **Graph Visualizations**: Network topology graphs, resource usage trends
- **Macro System**: Record and replay command sequences
- **API Key Management**: Generate API keys for external integrations
- **Two-Factor Authentication**: TOTP-based 2FA for superadmins
- **Backup Encryption**: Encrypt sensitive config backups
- **Change Approval Workflow**: Require approval for critical changes
- **Integration with External Tools**: Nagios, Grafana, etc.

## References

- Supermon source code: `tmp/srv/http/supermon/`
- Existing AMI connector: `internal/ami/connector.go`
- Current config system: `backend/config/config.go`
- Authentication: `backend/auth/auth.go`
- Frontend stores: `frontend/src/stores/`

## Conclusion

This implementation plan provides a comprehensive roadmap for building a modern, secure, and powerful administration panel for allstar-nexus. By following this phased approach, we'll deliver incremental value while maintaining code quality and security standards. The resulting admin panel will significantly enhance the operator experience and extend beyond Supermon's capabilities with a modern tech stack.
