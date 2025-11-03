package models

import (
	"time"
)

// AuditLog represents an immutable record of an administrative action
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Timestamp time.Time `gorm:"index;not null" json:"timestamp"`
	UserEmail string    `gorm:"index;not null" json:"user_email"`
	UserID    int64     `gorm:"index" json:"user_id"`
	Action    string    `gorm:"index;not null" json:"action"` // e.g., "config.update", "command.execute", "user.create"
	Resource  string    `gorm:"index" json:"resource"`        // What was affected (e.g., "config.yaml", "node:1234", "user:5")
	Details   string    `gorm:"type:text" json:"details"`     // JSON or text details of the action
	IPAddress string    `gorm:"index" json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Success   bool      `gorm:"index;not null" json:"success"`
	ErrorMsg  string    `gorm:"type:text" json:"error_msg,omitempty"`
}

// TableName specifies the table name for AuditLog
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditAction constants for common admin actions
const (
	AuditActionConfigRead     = "config.read"
	AuditActionConfigUpdate   = "config.update"
	AuditActionConfigBackup   = "config.backup"
	AuditActionConfigRestore  = "config.restore"
	AuditActionCommandExecute = "command.execute"
	AuditActionNodeLink       = "node.link"
	AuditActionNodeUnlink     = "node.unlink"
	AuditActionUserCreate     = "user.create"
	AuditActionUserUpdate     = "user.update"
	AuditActionUserDelete     = "user.delete"
	AuditActionUserReset      = "user.password_reset"
	AuditActionLogin          = "auth.login"
	AuditActionLoginFailed    = "auth.login_failed"
	AuditActionDBBackupCreate = "db.backup.create"
	AuditActionDBBackupRestore = "db.backup.restore"
	AuditActionDBBackupDelete = "db.backup.delete"
)
