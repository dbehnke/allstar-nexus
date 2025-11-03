package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/admin"
	"github.com/dbehnke/allstar-nexus/backend/auth"
	"github.com/dbehnke/allstar-nexus/backend/middleware"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
)

// AdminAPI handles administrative endpoints
type AdminAPI struct {
	AuditLogger     *admin.AuditLogger
	UserRepo        *repository.UserRepo
	ConfigManager   *admin.ConfigManager
	CommandExecutor *admin.CommandExecutor
	DBBackupManager *admin.DBBackupManager
}

// NewAdminAPI creates a new admin API instance
func NewAdminAPI(auditLogger *admin.AuditLogger, userRepo *repository.UserRepo) *AdminAPI {
	return &AdminAPI{
		AuditLogger: auditLogger,
		UserRepo:    userRepo,
	}
}

// ===============================================
// Audit Log Endpoints
// ===============================================

// GetAuditLogs returns audit logs with optional filtering
// GET /api/admin/audit
func (a *AdminAPI) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Parse query parameters
	q := r.URL.Query()
	filters := admin.AuditQueryFilters{
		UserEmail: q.Get("user"),
		Action:    q.Get("action"),
		Resource:  q.Get("resource"),
		Limit:     100, // default
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filters.Limit = limit
		}
	}

	if sinceStr := q.Get("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			filters.Since = since
		}
	}

	if untilStr := q.Get("until"); untilStr != "" {
		if until, err := time.Parse(time.RFC3339, untilStr); err == nil {
			filters.Until = until
		}
	}

	if q.Get("success_only") == "true" {
		filters.SuccessOnly = true
	}
	if q.Get("failures_only") == "true" {
		filters.FailuresOnly = true
	}

	logs, err := a.AuditLogger.Query(ctx, filters)
	if err != nil {
		writeError(w, 500, "db_error", "failed to query audit logs")
		return
	}

	writeJSON(w, 200, logs)
}

// GetRecentAuditLogs returns the most recent audit log entries
// GET /api/admin/audit/recent
func (a *AdminAPI) GetRecentAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	logs, err := a.AuditLogger.GetRecent(ctx, limit)
	if err != nil {
		writeError(w, 500, "db_error", "failed to get recent audit logs")
		return
	}

	writeJSON(w, 200, logs)
}

// ===============================================
// User Management Endpoints
// ===============================================

// ListUsers returns all users (without password hashes)
// GET /api/admin/users
func (a *AdminAPI) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, err := a.UserRepo.GetAll(ctx)
	if err != nil {
		writeError(w, 500, "db_error", "failed to list users")
		return
	}

	// Convert to safe representation (no password hashes)
	safeUsers := make([]map[string]interface{}, len(users))
	for i, u := range users {
		safeUsers[i] = map[string]interface{}{
			"id":         u.ID,
			"email":      u.Email,
			"role":       u.Role,
			"created_at": u.CreatedAt,
		}
	}

	writeJSON(w, 200, safeUsers)
}

// CreateUser creates a new user (admin operation)
// POST /api/admin/users
func (a *AdminAPI) CreateUser(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}

	// Validate input
	if body.Email == "" || body.Password == "" {
		writeError(w, 400, "validation_error", "email and password required")
		return
	}

	// Validate role
	validRoles := map[string]bool{
		models.RoleUser:       true,
		models.RoleAdmin:      true,
		models.RoleSuperAdmin: true,
	}
	if !validRoles[body.Role] {
		writeError(w, 400, "validation_error", "invalid role")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get current user for audit
	currentUser, _ := middleware.UserFromContext(r.Context())

	// Hash password
	hash, err := hashPassword(body.Password)
	if err != nil {
		writeError(w, 500, "server_error", "failed to hash password")
		return
	}

	// Create user
	newUser, err := a.UserRepo.Create(ctx, body.Email, hash, body.Role)
	if err != nil {
		writeError(w, 500, "db_error", "failed to create user")
		return
	}

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionUserCreate,
			"user:"+strconv.Itoa(int(newUser.ID)),
			map[string]interface{}{"email": body.Email, "role": body.Role},
			middleware.GetClientIP(r), r.UserAgent())
	}

	writeJSON(w, 201, map[string]interface{}{
		"id":    newUser.ID,
		"email": newUser.Email,
		"role":  newUser.Role,
	})
}

// UpdateUser updates an existing user
// PUT /api/admin/users/:id
func (a *AdminAPI) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path
	// Note: This assumes you have a router that extracts path params
	// For now, we'll use a simple approach
	idStr := r.URL.Path[len("/api/admin/users/"):]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, 400, "bad_request", "invalid user ID")
		return
	}

	type req struct {
		Email    *string `json:"email,omitempty"`
		Role     *string `json:"role,omitempty"`
		Password *string `json:"password,omitempty"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get current user for audit
	currentUser, _ := middleware.UserFromContext(r.Context())

	// Update user
	updates := make(map[string]interface{})
	if body.Email != nil {
		updates["email"] = *body.Email
	}
	if body.Role != nil {
		// Validate role
		validRoles := map[string]bool{
			models.RoleUser:       true,
			models.RoleAdmin:      true,
			models.RoleSuperAdmin: true,
		}
		if !validRoles[*body.Role] {
			writeError(w, 400, "validation_error", "invalid role")
			return
		}
		updates["role"] = *body.Role
	}
	if body.Password != nil {
		hash, err := hashPassword(*body.Password)
		if err != nil {
			writeError(w, 500, "server_error", "failed to hash password")
			return
		}
		updates["password_hash"] = hash
	}

	if err := a.UserRepo.Update(ctx, userID, updates); err != nil {
		writeError(w, 500, "db_error", "failed to update user")
		return
	}

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionUserUpdate,
			"user:"+strconv.FormatInt(userID, 10), updates,
			middleware.GetClientIP(r), r.UserAgent())
	}

	writeJSON(w, 200, map[string]interface{}{"updated": true})
}

// DeleteUser deletes a user
// DELETE /api/admin/users/:id
func (a *AdminAPI) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/admin/users/"):]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, 400, "bad_request", "invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get current user for audit
	currentUser, _ := middleware.UserFromContext(r.Context())

	// Don't allow deleting yourself
	if currentUser != nil && currentUser.ID == userID {
		writeError(w, 400, "validation_error", "cannot delete your own account")
		return
	}

	if err := a.UserRepo.Delete(ctx, userID); err != nil {
		writeError(w, 500, "db_error", "failed to delete user")
		return
	}

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionUserDelete,
			"user:"+strconv.FormatInt(userID, 10), nil,
			middleware.GetClientIP(r), r.UserAgent())
	}

	writeJSON(w, 200, map[string]interface{}{"deleted": true})
}

// ===============================================
// System Status Endpoints
// ===============================================

// GetSystemStatus returns basic system health information
// GET /api/admin/system/status
func (a *AdminAPI) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual system metrics collection
	// For now, return a placeholder
	status := map[string]interface{}{
		"timestamp": time.Now(),
		"uptime":    "placeholder",
		"cpu":       "N/A",
		"memory":    "N/A",
		"disk":      "N/A",
	}

	writeJSON(w, 200, status)
}

// ===============================================
// Config Management Endpoints
// ===============================================

// GetConfig returns the current config file contents
// GET /api/admin/config
func (a *AdminAPI) GetConfig(w http.ResponseWriter, r *http.Request) {
	if a.ConfigManager == nil {
		writeError(w, 500, "not_configured", "config manager not initialized")
		return
	}

	content, err := a.ConfigManager.Read()
	if err != nil {
		writeError(w, 500, "read_error", "failed to read config")
		return
	}

	// Audit log
	ctx := r.Context()
	if a.AuditLogger != nil {
		if currentUser, ok := middleware.UserFromContext(ctx); ok {
			_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionConfigRead,
				"config.yaml", nil, middleware.GetClientIP(r), r.UserAgent())
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"content": content,
		"path":    a.ConfigManager.ConfigPath,
	})
}

// UpdateConfig updates the config file with validation
// POST /api/admin/config
func (a *AdminAPI) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if a.ConfigManager == nil {
		writeError(w, 500, "not_configured", "config manager not initialized")
		return
	}

	type req struct {
		Content string `json:"content"`
		Comment string `json:"comment"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}

	if body.Content == "" {
		writeError(w, 400, "validation_error", "content is required")
		return
	}

	ctx := r.Context()
	currentUser, _ := middleware.UserFromContext(ctx)

	// Write config (validates and creates backup)
	backupID, err := a.ConfigManager.Write(body.Content, body.Comment)
	if err != nil {
		// Audit failure
		if a.AuditLogger != nil && currentUser != nil {
			_ = a.AuditLogger.LogFailure(ctx, currentUser.Email, currentUser.ID, models.AuditActionConfigUpdate,
				"config.yaml", err.Error(), nil, middleware.GetClientIP(r), r.UserAgent())
		}
		writeError(w, 400, "validation_error", err.Error())
		return
	}

	// Audit success
	if a.AuditLogger != nil && currentUser != nil {
		_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionConfigUpdate,
			"config.yaml", map[string]interface{}{"backup_id": backupID, "comment": body.Comment},
			middleware.GetClientIP(r), r.UserAgent())
	}

	writeJSON(w, 200, map[string]interface{}{
		"success":   true,
		"backup_id": backupID,
	})
}

// CreateConfigBackup creates a manual backup
// POST /api/admin/config/backup
func (a *AdminAPI) CreateConfigBackup(w http.ResponseWriter, r *http.Request) {
	if a.ConfigManager == nil {
		writeError(w, 500, "not_configured", "config manager not initialized")
		return
	}

	type req struct {
		Comment string `json:"comment"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Allow empty body
		body.Comment = ""
	}

	ctx := r.Context()
	currentUser, _ := middleware.UserFromContext(ctx)

	backupID, err := a.ConfigManager.CreateBackup(body.Comment)
	if err != nil {
		writeError(w, 500, "backup_error", err.Error())
		return
	}

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionConfigBackup,
			"config.yaml", map[string]interface{}{"backup_id": backupID, "comment": body.Comment},
			middleware.GetClientIP(r), r.UserAgent())
	}

	writeJSON(w, 200, map[string]interface{}{
		"backup_id": backupID,
	})
}

// ListConfigBackups returns all available backups
// GET /api/admin/config/backups
func (a *AdminAPI) ListConfigBackups(w http.ResponseWriter, r *http.Request) {
	if a.ConfigManager == nil {
		writeError(w, 500, "not_configured", "config manager not initialized")
		return
	}

	backups, err := a.ConfigManager.ListBackups()
	if err != nil {
		writeError(w, 500, "list_error", err.Error())
		return
	}

	writeJSON(w, 200, backups)
}

// RestoreConfigBackup restores a config from backup
// POST /api/admin/config/restore/:id
func (a *AdminAPI) RestoreConfigBackup(w http.ResponseWriter, r *http.Request) {
	if a.ConfigManager == nil {
		writeError(w, 500, "not_configured", "config manager not initialized")
		return
	}

	// Extract backup ID from path
	backupID := r.URL.Path[len("/api/admin/config/restore/"):]
	if backupID == "" {
		writeError(w, 400, "bad_request", "backup ID required")
		return
	}

	ctx := r.Context()
	currentUser, _ := middleware.UserFromContext(ctx)

	if err := a.ConfigManager.Restore(backupID); err != nil {
		// Audit failure
		if a.AuditLogger != nil && currentUser != nil {
			_ = a.AuditLogger.LogFailure(ctx, currentUser.Email, currentUser.ID, models.AuditActionConfigRestore,
				"config.yaml", err.Error(), map[string]interface{}{"backup_id": backupID},
				middleware.GetClientIP(r), r.UserAgent())
		}
		writeError(w, 400, "restore_error", err.Error())
		return
	}

	// Audit success
	if a.AuditLogger != nil && currentUser != nil {
		_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionConfigRestore,
			"config.yaml", map[string]interface{}{"backup_id": backupID},
			middleware.GetClientIP(r), r.UserAgent())
	}

	writeJSON(w, 200, map[string]interface{}{
		"success": true,
	})
}

// GetConfigDiff returns a diff between two configs
// GET /api/admin/config/diff?from=<id>&to=<id>
func (a *AdminAPI) GetConfigDiff(w http.ResponseWriter, r *http.Request) {
	if a.ConfigManager == nil {
		writeError(w, 500, "not_configured", "config manager not initialized")
		return
	}

	fromID := r.URL.Query().Get("from")
	toID := r.URL.Query().Get("to")

	if fromID == "" || toID == "" {
		writeError(w, 400, "bad_request", "from and to parameters required")
		return
	}

	diff, err := a.ConfigManager.Diff(fromID, toID)
	if err != nil {
		writeError(w, 400, "diff_error", err.Error())
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"diff": diff,
		"from": fromID,
		"to":   toID,
	})
}

// ===============================================
// AMI Command Endpoints
// ===============================================

// ExecuteAMICommand executes a raw AMI command
// POST /api/admin/ami/command
func (a *AdminAPI) ExecuteAMICommand(w http.ResponseWriter, r *http.Request) {
	if a.CommandExecutor == nil {
		writeError(w, 500, "not_configured", "command executor not initialized")
		return
	}

	type req struct {
		Command string `json:"command"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}

	if body.Command == "" {
		writeError(w, 400, "validation_error", "command is required")
		return
	}

	ctx := r.Context()
	currentUser, _ := middleware.UserFromContext(ctx)

	result, err := a.CommandExecutor.ExecuteCommand(body.Command)

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		if err != nil || !result.Success {
			_ = a.AuditLogger.LogFailure(ctx, currentUser.Email, currentUser.ID, models.AuditActionCommandExecute,
				"ami", result.Error, map[string]interface{}{"command": body.Command},
				middleware.GetClientIP(r), r.UserAgent())
		} else {
			_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionCommandExecute,
				"ami", map[string]interface{}{"command": body.Command, "output": result.Output},
				middleware.GetClientIP(r), r.UserAgent())
		}
	}

	if err != nil && !result.Success {
		writeError(w, 400, "execution_error", err.Error())
		return
	}

	writeJSON(w, 200, result)
}

// LinkNode links two nodes via AMI
// POST /api/admin/ami/link
func (a *AdminAPI) LinkNode(w http.ResponseWriter, r *http.Request) {
	if a.CommandExecutor == nil {
		writeError(w, 500, "not_configured", "command executor not initialized")
		return
	}

	type req struct {
		LocalNode  int  `json:"local_node"`
		RemoteNode int  `json:"remote_node"`
		Permanent  bool `json:"permanent"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}

	if body.LocalNode == 0 || body.RemoteNode == 0 {
		writeError(w, 400, "validation_error", "local_node and remote_node are required")
		return
	}

	ctx := r.Context()
	currentUser, _ := middleware.UserFromContext(ctx)

	result, err := a.CommandExecutor.LinkNode(body.LocalNode, body.RemoteNode, body.Permanent)

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		resource := fmt.Sprintf("node:%d->%d", body.LocalNode, body.RemoteNode)
		if err != nil || !result.Success {
			_ = a.AuditLogger.LogFailure(ctx, currentUser.Email, currentUser.ID, models.AuditActionNodeLink,
				resource, result.Error, map[string]interface{}{"permanent": body.Permanent},
				middleware.GetClientIP(r), r.UserAgent())
		} else {
			_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionNodeLink,
				resource, map[string]interface{}{"permanent": body.Permanent},
				middleware.GetClientIP(r), r.UserAgent())
		}
	}

	if err != nil && !result.Success {
		writeError(w, 400, "execution_error", err.Error())
		return
	}

	writeJSON(w, 200, result)
}

// UnlinkNode disconnects two nodes via AMI
// POST /api/admin/ami/unlink
func (a *AdminAPI) UnlinkNode(w http.ResponseWriter, r *http.Request) {
	if a.CommandExecutor == nil {
		writeError(w, 500, "not_configured", "command executor not initialized")
		return
	}

	type req struct {
		LocalNode  int `json:"local_node"`
		RemoteNode int `json:"remote_node"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_request", "invalid json body")
		return
	}

	if body.LocalNode == 0 || body.RemoteNode == 0 {
		writeError(w, 400, "validation_error", "local_node and remote_node are required")
		return
	}

	ctx := r.Context()
	currentUser, _ := middleware.UserFromContext(ctx)

	result, err := a.CommandExecutor.UnlinkNode(body.LocalNode, body.RemoteNode)

	// Audit log
	if a.AuditLogger != nil && currentUser != nil {
		resource := fmt.Sprintf("node:%d->%d", body.LocalNode, body.RemoteNode)
		if err != nil || !result.Success {
			_ = a.AuditLogger.LogFailure(ctx, currentUser.Email, currentUser.ID, models.AuditActionNodeUnlink,
				resource, result.Error, nil,
				middleware.GetClientIP(r), r.UserAgent())
		} else {
			_ = a.AuditLogger.LogSuccess(ctx, currentUser.Email, currentUser.ID, models.AuditActionNodeUnlink,
				resource, nil,
				middleware.GetClientIP(r), r.UserAgent())
		}
	}

	if err != nil && !result.Success {
		writeError(w, 400, "execution_error", err.Error())
		return
	}

	writeJSON(w, 200, result)
}

// GetPredefinedCommands returns a list of safe predefined commands
// GET /api/admin/ami/commands/predefined
func (a *AdminAPI) GetPredefinedCommands(w http.ResponseWriter, r *http.Request) {
	if a.CommandExecutor == nil {
		writeError(w, 500, "not_configured", "command executor not initialized")
		return
	}

	commands := a.CommandExecutor.GetPredefinedCommands()
	writeJSON(w, 200, commands)
}

// ===============================================
// Database Backup Endpoints
// ===============================================

// CreateDBBackup creates a manual database backup
// POST /api/admin/db/backup
func (a *AdminAPI) CreateDBBackup(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if a.DBBackupManager == nil {
		writeError(w, 500, "not_configured", "database backup manager not initialized")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req.Comment = "Manual backup"
	}

	// Get user from context for audit logging
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		writeError(w, 401, "unauthorized", "user not found in context")
		return
	}

	backupID, err := a.DBBackupManager.CreateBackup(req.Comment, "manual")
	if err != nil {
		_ = a.AuditLogger.LogFailure(ctx, user.Email, user.ID, models.AuditActionDBBackupCreate, "", err.Error(), nil, r.RemoteAddr, r.UserAgent())
		writeError(w, 500, "backup_failed", fmt.Sprintf("failed to create backup: %v", err))
		return
	}

	_ = a.AuditLogger.LogSuccess(ctx, user.Email, user.ID, models.AuditActionDBBackupCreate, backupID, nil, r.RemoteAddr, r.UserAgent())

	writeJSON(w, 200, map[string]interface{}{
		"backup_id": backupID,
		"comment":   req.Comment,
	})
}

// ListDBBackups returns all available database backups
// GET /api/admin/db/backups
func (a *AdminAPI) ListDBBackups(w http.ResponseWriter, r *http.Request) {
	if a.DBBackupManager == nil {
		writeError(w, 500, "not_configured", "database backup manager not initialized")
		return
	}

	backups, err := a.DBBackupManager.ListBackups()
	if err != nil {
		writeError(w, 500, "list_failed", fmt.Sprintf("failed to list backups: %v", err))
		return
	}

	writeJSON(w, 200, backups)
}

// RestoreDBBackup restores database from a backup
// POST /api/admin/db/restore/:id
// WARNING: This requires application restart
func (a *AdminAPI) RestoreDBBackup(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if a.DBBackupManager == nil {
		writeError(w, 500, "not_configured", "database backup manager not initialized")
		return
	}

	// Extract backup ID from URL path
	backupID := r.PathValue("id")
	if backupID == "" {
		writeError(w, 400, "invalid_request", "backup ID required")
		return
	}

	// Get user from context for audit logging
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		writeError(w, 401, "unauthorized", "user not found in context")
		return
	}

	// Restore the backup
	if err := a.DBBackupManager.Restore(backupID); err != nil {
		_ = a.AuditLogger.LogFailure(ctx, user.Email, user.ID, models.AuditActionDBBackupRestore, backupID, err.Error(), nil, r.RemoteAddr, r.UserAgent())
		writeError(w, 500, "restore_failed", fmt.Sprintf("failed to restore backup: %v", err))
		return
	}

	_ = a.AuditLogger.LogSuccess(ctx, user.Email, user.ID, models.AuditActionDBBackupRestore, backupID, nil, r.RemoteAddr, r.UserAgent())

	writeJSON(w, 200, map[string]interface{}{
		"restored":  true,
		"backup_id": backupID,
		"message":   "Database restored. Application restart required.",
	})
}

// DeleteDBBackup deletes a database backup
// DELETE /api/admin/db/backups/:id
func (a *AdminAPI) DeleteDBBackup(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if a.DBBackupManager == nil {
		writeError(w, 500, "not_configured", "database backup manager not initialized")
		return
	}

	// Extract backup ID from URL path
	backupID := r.PathValue("id")
	if backupID == "" {
		writeError(w, 400, "invalid_request", "backup ID required")
		return
	}

	// Get user from context for audit logging
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		writeError(w, 401, "unauthorized", "user not found in context")
		return
	}

	if err := a.DBBackupManager.DeleteBackup(backupID); err != nil {
		_ = a.AuditLogger.LogFailure(ctx, user.Email, user.ID, models.AuditActionDBBackupDelete, backupID, err.Error(), nil, r.RemoteAddr, r.UserAgent())
		writeError(w, 500, "delete_failed", fmt.Sprintf("failed to delete backup: %v", err))
		return
	}

	_ = a.AuditLogger.LogSuccess(ctx, user.Email, user.ID, models.AuditActionDBBackupDelete, backupID, nil, r.RemoteAddr, r.UserAgent())

	writeJSON(w, 200, map[string]interface{}{
		"deleted":   true,
		"backup_id": backupID,
	})
}

// GetDBBackupStats returns statistics about a database backup
// GET /api/admin/db/backups/:id/stats
func (a *AdminAPI) GetDBBackupStats(w http.ResponseWriter, r *http.Request) {
	if a.DBBackupManager == nil {
		writeError(w, 500, "not_configured", "database backup manager not initialized")
		return
	}

	// Extract backup ID from URL path
	backupID := r.PathValue("id")
	if backupID == "" {
		writeError(w, 400, "invalid_request", "backup ID required")
		return
	}

	stats, err := a.DBBackupManager.GetBackupStats(backupID)
	if err != nil {
		writeError(w, 404, "not_found", fmt.Sprintf("failed to get backup stats: %v", err))
		return
	}

	writeJSON(w, 200, stats)
}

// ===============================================
// Helper Functions
// ===============================================

// hashPassword wraps auth.HashPassword for convenience
func hashPassword(password string) (string, error) {
	return auth.HashPassword(password)
}
