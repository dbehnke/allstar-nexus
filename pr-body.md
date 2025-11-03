# feat: Admin Panel Phase 7 - Advanced Node Management & Task Scheduling (Complete)

## Summary
Implements Phase 7 of the admin panel roadmap, adding advanced node management capabilities and a flexible task scheduling system. This PR includes **both backend infrastructure and complete frontend UI** for node groups, scheduled tasks, and bulk node operations.

## Features

### 🔹 Node Group Management
- **Create and manage node groups**: Organize nodes into logical collections
- **Bulk node operations**: Link/unlink multiple nodes simultaneously
- **Persistent configuration**: Save link configurations per node in groups
- **CRUD API**: Full REST API for node group management

### 🔹 Task Scheduling System
- **Cron-based scheduler**: Flexible scheduling using standard cron expressions
- **Multiple task types**:
  - `link_nodes` - Automatically link two nodes (e.g., scheduled net connections)
  - `unlink_nodes` - Automatically unlink two nodes
  - `execute_ami` - Execute custom AMI actions with parameters
  - `backup_db` - Automated database backups
- **Task execution tracking**: Complete history with success/failure logging
- **Dynamic task management**: Add, update, pause, resume, or delete tasks on the fly
- **Auto-start on boot**: Active tasks resume automatically on server restart

### 🔹 Bulk Node Operations
- **Bulk link**: Connect multiple nodes to target nodes in parallel
- **Bulk unlink**: Disconnect multiple nodes from target nodes
- **Group-based operations**: Apply operations to all nodes in a group
- **Detailed error reporting**: Per-node success/failure feedback

## What's New

### Backend Models
- **`NodeGroup`** ([backend/models/node_group.go](backend/models/node_group.go)): Group metadata with name, description, timestamps
- **`NodeGroupConfig`**: Per-node configuration within groups (node ID, link settings)
- **`ScheduledTask`** ([backend/models/scheduled_task.go](backend/models/scheduled_task.go)): Task definition with cron expression, type, parameters
- **`TaskExecutionLog`**: Execution history with timestamps, duration, results

### Backend Services
- **`NodeGroupManager`** ([backend/admin/node_group_manager.go](backend/admin/node_group_manager.go)):
  - Create, read, update, delete node groups
  - Add/remove nodes from groups
  - Retrieve group configurations

- **`Scheduler`** ([backend/admin/scheduler.go](backend/admin/scheduler.go)):
  - Cron-based task scheduling using `robfig/cron/v3`
  - Task lifecycle management (create, update, pause, resume, delete)
  - Automatic next-run calculation
  - Execution history tracking
  - Graceful shutdown handling

- **`TaskExecutor`** ([backend/admin/task_executor.go](backend/admin/task_executor.go)):
  - `DefaultTaskExecutor` implements all task types
  - AMI command execution with timeout
  - Backup task integration
  - Audit log capture
  - Extensible interface for custom task types

### API Endpoints

#### Node Groups
- `GET /api/admin/node-groups` - List all node groups
- `POST /api/admin/node-groups` - Create new node group
- `PUT /api/admin/node-groups/:id` - Update node group
- `DELETE /api/admin/node-groups/:id` - Delete node group

#### Bulk Operations
- `POST /api/admin/nodes/bulk-link` - Link multiple nodes
- `POST /api/admin/nodes/bulk-unlink` - Unlink multiple nodes

#### Scheduled Tasks
- `GET /api/admin/scheduled-tasks` - List all scheduled tasks
- `POST /api/admin/scheduled-tasks` - Create new scheduled task
- `PUT /api/admin/scheduled-tasks/:id` - Update scheduled task (including pause/resume)
- `DELETE /api/admin/scheduled-tasks/:id` - Delete scheduled task
- `GET /api/admin/scheduled-tasks/:id/logs` - Get execution history for a task

## Frontend Components (NEW)

### NodeGroups.vue
Complete UI for node group management:
- **Grid card layout** displaying all groups with name, description, node count
- **Create modal** with form for group name, description, and comma-separated node IDs
- **Edit modal** pre-populated with existing group data
- **Delete confirmation** modal with safety check
- **Node badges** showing group membership in visual format
- **Real-time feedback** with success/error messages
- **Responsive design** with auto-adjusting grid columns

### ScheduledTasks.vue
Comprehensive task scheduling interface:
- **Task cards** showing type, schedule, status (active/paused), next run time
- **Create/Edit modal** with dynamic form fields based on task type:
  - Link/Unlink nodes: local node, remote node, permanent option
  - Execute AMI: action name, JSON parameters
  - Database backup: comment field
- **Cron expression input** with helpful examples and descriptions
- **Pause/Resume toggle** for quick task control
- **Execution logs viewer** modal showing history with timestamps, duration, success/failure
- **Delete confirmation** with safety check
- **Status badges** (active/paused) with color coding
- **Per-task type icons** and descriptions

### NodeControl.vue (Enhanced)
Extended with bulk operations:
- **New bulk operations section** below existing link/unlink panels
- **Comma-separated input** for multiple local and target nodes
- **Bulk link button** with permanent option
- **Bulk unlink button** for batch disconnections
- **Results display** showing per-node success/failure with details
- **Progress indicators** during bulk operations
- **Detailed error reporting** for each node pair
- **Maintains existing** single-node functionality

### AdminDashboard.vue (Updated)
Added Phase 7 cards:
- **Node Groups card** (📦 icon)
  - Shows total group count
  - "Manage Groups" button → `/admin/node-groups`
- **Scheduled Tasks card** (⏰ icon)
  - Shows active task count
  - "Manage Tasks" button → `/admin/scheduled-tasks`
- **Auto-loading** of counts on mount

### Router Updates
- Added `/admin/node-groups` route with superadmin guard
- Added `/admin/scheduled-tasks` route with superadmin guard
- Imported new components

### Integration
- **Database migrations**: Auto-migrate new tables on startup ([main.go:159-162](main.go#L159-L162))
- **Service initialization**: NodeGroupManager and Scheduler initialized in main ([main.go:238-256](main.go#L238-L256))
- **Route registration**: All API routes registered with superadmin auth ([main.go:322-384](main.go#L322-L384))
- **Audit logging**: All operations logged for compliance tracking

## Technical Details

### Scheduler Architecture
- Uses `robfig/cron/v3` for reliable cron expression parsing and scheduling
- Tasks stored in database with status tracking (`active`, `paused`, `disabled`)
- Execution logs capture:
  - Start/end timestamps
  - Duration in milliseconds
  - Success/failure status
  - Error messages
  - Result details
- Next run time calculated and persisted after each execution
- Failed tasks increment fail counter but continue on schedule

### Node Group Structure
```go
type NodeGroup struct {
    ID          uint
    Name        string  // Unique group name
    Description string
    Nodes       []NodeGroupConfig  // Has-many relationship
}

type NodeGroupConfig struct {
    ID          uint
    NodeGroupID uint
    NodeNumber  int     // Asterisk node number
    Permanent   bool    // Link as permanent
    Transceive  bool    // Enable bidirectional audio
}
```

### Task Execution Flow
1. Scheduler triggers task at cron interval
2. Task loaded from database and validated
3. TaskExecutor routes to appropriate handler based on `task_type`
4. Handler executes operation (AMI command, backup, etc.)
5. Result logged to `task_execution_logs` table
6. Task statistics updated (`run_count`, `fail_count`, `last_run_at`, `next_run_at`)

### Bulk Operation Strategy
- **Parallel execution**: Node operations execute concurrently for performance
- **Partial failure handling**: Some nodes may succeed while others fail
- **Detailed results**: Returns per-node status with error messages
- **Audit trail**: Each operation logged with timestamp and user

## Database Schema

### New Tables
```sql
-- Node groups
CREATE TABLE node_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

-- Node group configurations
CREATE TABLE node_group_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_group_id INTEGER NOT NULL,
    node_number INTEGER NOT NULL,
    permanent BOOLEAN DEFAULT false,
    transceive BOOLEAN DEFAULT false,
    FOREIGN KEY (node_group_id) REFERENCES node_groups(id) ON DELETE CASCADE
);

-- Scheduled tasks
CREATE TABLE scheduled_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    task_type TEXT NOT NULL,  -- link_nodes, unlink_nodes, execute_ami, backup_db
    cron_expression TEXT NOT NULL,
    parameters TEXT,  -- JSON
    status TEXT DEFAULT 'active',  -- active, paused, disabled
    last_run_at DATETIME,
    next_run_at DATETIME,
    run_count INTEGER DEFAULT 0,
    fail_count INTEGER DEFAULT 0,
    last_result TEXT,
    last_error TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME
);

-- Task execution logs
CREATE TABLE task_execution_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    executed_at DATETIME NOT NULL,
    duration_ms INTEGER,
    success BOOLEAN,
    result TEXT,
    error TEXT,
    FOREIGN KEY (task_id) REFERENCES scheduled_tasks(id) ON DELETE CASCADE
);
```

## Task Type Reference

### Supported Task Types

#### `link_nodes`
Automatically links two nodes together at scheduled times.

**Required Parameters:**
- `source_node` (integer) - The node number initiating the link
- `target_node` (integer) - The node number to link to

**Optional Parameters:**
- `permanent` (boolean) - Whether to make the link permanent (default: false)

**Example Parameters:**
```json
{
  "source_node": 12345,
  "target_node": 99999,
  "permanent": false
}
```

#### `unlink_nodes`
Automatically unlinks two nodes at scheduled times.

**Required Parameters:**
- `source_node` (integer) - The node number initiating the unlink
- `target_node` (integer) - The node number to unlink from

**Example Parameters:**
```json
{
  "source_node": 12345,
  "target_node": 99999
}
```

#### `execute_ami`
Executes a custom AMI action with specified parameters.

**Required Parameters:**
- `action` (string) - The AMI action name (e.g., "Command", "GetConfig")

**Optional Parameters:**
- `params` (object) - Key-value pairs of AMI action parameters

**Example Parameters:**
```json
{
  "action": "Command",
  "params": {
    "Command": "rpt stats 12345"
  }
}
```

#### `backup_db`
Creates a scheduled database backup.

**Optional Parameters:**
- `comment` (string) - Comment to attach to the backup (default: "Scheduled backup: {task_name}")

**Example Parameters:**
```json
{
  "comment": "Daily automated backup"
}
```

## Usage Examples

### Create a Node Group
```bash
curl -X POST http://localhost:8080/api/admin/node-groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Regional Repeaters",
    "description": "All repeaters in the metro region",
    "nodes": [
      {"node_number": 12345, "permanent": true, "transceive": true},
      {"node_number": 23456, "permanent": true, "transceive": true}
    ]
  }'
```

### Schedule Daily Database Backup
```bash
curl -X POST http://localhost:8080/api/admin/scheduled-tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Backup",
    "description": "Automatic daily database backup at 2 AM",
    "task_type": "db_backup",
    "cron_expression": "0 2 * * *",
    "parameters": {"comment": "Scheduled daily backup"},
    "status": "active"
  }'
```

### Schedule Weekly Node Link
```bash
curl -X POST http://localhost:8080/api/admin/scheduled-tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sunday Net Auto-Link",
    "description": "Link to net hub every Sunday at 7 PM",
    "task_type": "link_nodes",
    "cron_expression": "0 19 * * 0",
    "parameters": {
      "source_node": 12345,
      "target_node": 99999,
      "permanent": false
    },
    "status": "active"
  }'
```

### Schedule Custom AMI Command
```bash
curl -X POST http://localhost:8080/api/admin/scheduled-tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Status Check",
    "description": "Query node status daily",
    "task_type": "execute_ami",
    "cron_expression": "0 6 * * *",
    "parameters": {
      "action": "Command",
      "params": {
        "Command": "rpt stats 12345"
      }
    },
    "status": "active"
  }'
```

### Bulk Link Nodes
```bash
curl -X POST http://localhost:8080/api/admin/nodes/bulk-link \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_node": 12345,
    "target_nodes": [23456, 34567, 45678],
    "permanent": false
  }'
```

## Migration Notes

### Automatic Migration
- All database schema changes apply automatically on first startup
- No manual SQL required
- Safe to run on existing databases (tables created only if missing)

### Compatibility
- **Backward compatible**: No changes to existing API endpoints or data structures
- **AMI required**: Task types `link_nodes`, `unlink_nodes`, and `execute_ami` require AMI to be enabled and connected
  - Set `ami_enabled: true` in `config.yaml`
  - Tasks will fail if AMI is not connected when scheduled to run
- **Permissions**: All new endpoints require `superadmin` role
- **Cron syntax**: Uses standard cron format (minute hour day month weekday)

## Testing

### Verified
- ✅ Go build passes without errors
- ✅ Database migrations apply successfully
- ✅ Scheduler starts and loads active tasks
- ✅ All API routes registered correctly
- ✅ Service dependencies injected properly

### Manual Testing Checklist
- [ ] Create node group via API
- [ ] Add/remove nodes from group
- [ ] Execute bulk link operation
- [ ] Execute bulk unlink operation
- [ ] Create scheduled task with various cron expressions
- [ ] Verify task executes at scheduled time
- [ ] Check execution logs populate correctly
- [ ] Pause/resume task
- [ ] Delete task and verify cron entry removed
- [ ] Restart server and verify active tasks resume
- [ ] Check audit logs for all operations

## Security

- **Authentication**: All endpoints require valid JWT token
- **Authorization**: All endpoints restricted to `superadmin` role only
- **Audit logging**: Every operation logged with user email, timestamp, and result
- **Input validation**: Cron expressions validated before task creation
- **Parameter sanitization**: Task parameters stored as JSON, validated on execution

## Known Limitations

### Task Types
Currently supports 4 task types. Additional types can be added by:
1. Implementing handler in `TaskExecutor`
2. Adding validation in `CreateScheduledTask` API handler
3. Documenting parameter schema

### Scheduler Limitations
- Task execution is **synchronous** (one task at a time)
- No built-in retry logic for failed tasks
- Maximum execution timeout: 5 minutes (configurable in code)

## Dependencies

### New Go Modules
```go
github.com/robfig/cron/v3 v3.0.1
```

Updated `go.mod` and `go.sum` included in this PR.

## Breaking Changes

**None** - This is purely additive functionality.

## Future Enhancements (Phase 7.5+)

- **Frontend UI**: Vue components for node groups and scheduled tasks
- **Task priority/queuing**: Execute multiple tasks concurrently with priority
- **Task dependencies**: Chain tasks together (run task B after task A succeeds)
- **Retry policies**: Automatic retry with exponential backoff
- **Task templates**: Pre-defined task configurations for common operations
- **Email/webhook notifications**: Alert on task failure
- **Advanced cron UI**: Visual cron expression builder

## Documentation Updates

- Updated [ADMIN-PHASES-5-6-COMPLETE.md](ADMIN-PHASES-5-6-COMPLETE.md) → Renamed to include Phase 7
- Backend file structure updated to reflect new files
- API endpoint documentation updated

## Checklist

- [x] Build passes (`go build`)
- [x] All services initialize correctly
- [x] Database migrations tested
- [x] API routes registered
- [x] Audit logging integrated
- [x] No secrets committed
- [x] Dependencies added to `go.mod`
- [x] Code follows existing patterns
- [x] Frontend UI complete (NodeGroups.vue, ScheduledTasks.vue, bulk ops)
- [x] Router configured with new routes
- [x] Admin dashboard updated with Phase 7 cards
- [x] Frontend builds successfully
- [ ] Integration tests (deferred)

## Related Issues/PRs

- Closes Phase 7 backend tasks from admin panel roadmap
- Builds on Phase 5-6 (Database Backup & System Monitoring)
- Prepares infrastructure for Phase 8 (Log Management)

---

**Review Focus**:
- Scheduler reliability and graceful shutdown
- Task executor extensibility
- Bulk operation error handling
- Database schema design
- API endpoint security
