package models

import (
	"time"
)

// ScheduledTaskType defines the type of scheduled task
type ScheduledTaskType string

const (
	TaskTypeLinkNodes   ScheduledTaskType = "link_nodes"
	TaskTypeUnlinkNodes ScheduledTaskType = "unlink_nodes"
	TaskTypeExecuteAMI  ScheduledTaskType = "execute_ami"
	TaskTypeBackupDB    ScheduledTaskType = "backup_db"
)

// ScheduledTaskStatus defines the status of a scheduled task
type ScheduledTaskStatus string

const (
	TaskStatusActive   ScheduledTaskStatus = "active"
	TaskStatusPaused   ScheduledTaskStatus = "paused"
	TaskStatusDisabled ScheduledTaskStatus = "disabled"
)

// ScheduledTask represents a scheduled task that runs at specified intervals
type ScheduledTask struct {
	ID          uint                `gorm:"primaryKey" json:"id"`
	Name        string              `gorm:"not null" json:"name"`
	Description string              `json:"description"`
	TaskType    ScheduledTaskType   `gorm:"not null" json:"task_type"`
	Status      ScheduledTaskStatus `gorm:"default:'active'" json:"status"`

	// Cron expression for scheduling (e.g., "0 0 * * *" for daily at midnight)
	CronExpression string `gorm:"not null" json:"cron_expression"`

	// Task-specific parameters (e.g., nodes to link, AMI commands, etc.)
	Parameters map[string]interface{} `gorm:"serializer:json" json:"parameters"`

	// Execution tracking
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	NextRunAt   *time.Time `json:"next_run_at,omitempty"`
	LastResult  string     `json:"last_result,omitempty"`  // "success", "failure", or error message
	LastError   string     `json:"last_error,omitempty"`
	RunCount    int        `gorm:"default:0" json:"run_count"`
	FailCount   int        `gorm:"default:0" json:"fail_count"`

	// Metadata
	CreatedBy string    `gorm:"not null" json:"created_by"` // Email of creator
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskExecutionLog represents a log entry for task execution history
type TaskExecutionLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TaskID      uint      `gorm:"not null;index" json:"task_id"`
	Task        *ScheduledTask `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"task,omitempty"`
	ExecutedAt  time.Time `gorm:"index" json:"executed_at"`
	Success     bool      `json:"success"`
	Result      string    `json:"result"`
	Error       string    `json:"error,omitempty"`
	DurationMs  int64     `json:"duration_ms"` // Execution duration in milliseconds
}

// TableName overrides the table name for ScheduledTask
func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}

// TableName overrides the table name for TaskExecutionLog
func (TaskExecutionLog) TableName() string {
	return "task_execution_logs"
}
