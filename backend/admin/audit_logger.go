package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
)

// AuditLogger handles recording of administrative actions
type AuditLogger struct {
	db *gorm.DB
}

// NewAuditLogger creates a new audit logger instance
func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// LogEntry represents a single audit log entry to be recorded
type LogEntry struct {
	UserEmail string
	UserID    int64
	Action    string
	Resource  string
	Details   interface{} // Will be JSON-encoded
	IPAddress string
	UserAgent string
	Success   bool
	ErrorMsg  string
}

// Log records an audit entry to the database
func (a *AuditLogger) Log(ctx context.Context, entry LogEntry) error {
	// Serialize details to JSON if provided
	var detailsJSON string
	if entry.Details != nil {
		bytes, err := json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal audit details: %w", err)
		}
		detailsJSON = string(bytes)
	}

	auditLog := models.AuditLog{
		Timestamp: time.Now(),
		UserEmail: entry.UserEmail,
		UserID:    entry.UserID,
		Action:    entry.Action,
		Resource:  entry.Resource,
		Details:   detailsJSON,
		IPAddress: entry.IPAddress,
		UserAgent: entry.UserAgent,
		Success:   entry.Success,
		ErrorMsg:  entry.ErrorMsg,
	}

	if err := a.db.WithContext(ctx).Create(&auditLog).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// LogSuccess is a convenience method for logging successful actions
func (a *AuditLogger) LogSuccess(ctx context.Context, userEmail string, userID int64, action, resource string, details interface{}, ip, userAgent string) error {
	return a.Log(ctx, LogEntry{
		UserEmail: userEmail,
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   true,
	})
}

// LogFailure is a convenience method for logging failed actions
func (a *AuditLogger) LogFailure(ctx context.Context, userEmail string, userID int64, action, resource, errorMsg string, details interface{}, ip, userAgent string) error {
	return a.Log(ctx, LogEntry{
		UserEmail: userEmail,
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   false,
		ErrorMsg:  errorMsg,
	})
}

// Query returns audit logs matching the given filters
func (a *AuditLogger) Query(ctx context.Context, filters AuditQueryFilters) ([]models.AuditLog, error) {
	query := a.db.WithContext(ctx).Model(&models.AuditLog{})

	// Apply filters
	if filters.UserEmail != "" {
		query = query.Where("user_email = ?", filters.UserEmail)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if !filters.Since.IsZero() {
		query = query.Where("timestamp >= ?", filters.Since)
	}
	if !filters.Until.IsZero() {
		query = query.Where("timestamp <= ?", filters.Until)
	}
	if filters.SuccessOnly {
		query = query.Where("success = ?", true)
	}
	if filters.FailuresOnly {
		query = query.Where("success = ?", false)
	}

	// Order by most recent first
	query = query.Order("timestamp DESC")

	// Apply limit
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	} else {
		query = query.Limit(100) // Default limit
	}

	var logs []models.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}

	return logs, nil
}

// AuditQueryFilters defines filters for querying audit logs
type AuditQueryFilters struct {
	UserEmail    string
	Action       string
	Resource     string
	Since        time.Time
	Until        time.Time
	SuccessOnly  bool
	FailuresOnly bool
	Limit        int
}

// Count returns the total number of audit log entries matching the filters
func (a *AuditLogger) Count(ctx context.Context, filters AuditQueryFilters) (int64, error) {
	query := a.db.WithContext(ctx).Model(&models.AuditLog{})

	if filters.UserEmail != "" {
		query = query.Where("user_email = ?", filters.UserEmail)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if !filters.Since.IsZero() {
		query = query.Where("timestamp >= ?", filters.Since)
	}
	if !filters.Until.IsZero() {
		query = query.Where("timestamp <= ?", filters.Until)
	}
	if filters.SuccessOnly {
		query = query.Where("success = ?", true)
	}
	if filters.FailuresOnly {
		query = query.Where("success = ?", false)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// GetRecent returns the most recent N audit log entries
func (a *AuditLogger) GetRecent(ctx context.Context, limit int) ([]models.AuditLog, error) {
	return a.Query(ctx, AuditQueryFilters{Limit: limit})
}

// DeleteOlderThan removes audit logs older than the specified time
// This should be used carefully and only for log retention policies
func (a *AuditLogger) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := a.db.WithContext(ctx).Where("timestamp < ?", cutoff).Delete(&models.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", result.Error)
	}
	return result.RowsAffected, nil
}
