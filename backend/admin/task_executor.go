package admin

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/internal/ami"
)

// DefaultTaskExecutor implements TaskExecutor for executing scheduled tasks
type DefaultTaskExecutor struct {
	amiConnector    *ami.Connector
	dbBackupManager *DBBackupManager
	auditLogger     *AuditLogger
}

// NewDefaultTaskExecutor creates a new task executor
func NewDefaultTaskExecutor(amiConnector *ami.Connector, dbBackupManager *DBBackupManager, auditLogger *AuditLogger) *DefaultTaskExecutor {
	return &DefaultTaskExecutor{
		amiConnector:    amiConnector,
		dbBackupManager: dbBackupManager,
		auditLogger:     auditLogger,
	}
}

// Execute runs a scheduled task based on its type
func (e *DefaultTaskExecutor) Execute(ctx context.Context, task *models.ScheduledTask) error {
	switch task.TaskType {
	case models.TaskTypeLinkNodes:
		return e.executeLinkNodes(ctx, task)
	case models.TaskTypeUnlinkNodes:
		return e.executeUnlinkNodes(ctx, task)
	case models.TaskTypeExecuteAMI:
		return e.executeAMICommand(ctx, task)
	case models.TaskTypeBackupDB:
		return e.executeBackupDB(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.TaskType)
	}
}

// executeLinkNodes links nodes according to task parameters
func (e *DefaultTaskExecutor) executeLinkNodes(ctx context.Context, task *models.ScheduledTask) error {
	// Extract parameters
	sourceNodeRaw, ok := task.Parameters["source_node"]
	if !ok {
		return fmt.Errorf("source_node parameter is required")
	}
	targetNodeRaw, ok := task.Parameters["target_node"]
	if !ok {
		return fmt.Errorf("target_node parameter is required")
	}

	sourceNode := int(sourceNodeRaw.(float64))
	targetNode := int(targetNodeRaw.(float64))

	permanent := false
	if p, ok := task.Parameters["permanent"]; ok {
		permanent = p.(bool)
	}

	if e.amiConnector == nil || !e.amiConnector.IsConnected() {
		return fmt.Errorf("AMI not connected")
	}

	// Execute link command
	result, err := e.amiConnector.LinkNode(sourceNode, targetNode, permanent)
	if err != nil {
		return fmt.Errorf("failed to link nodes %d -> %d: %w", sourceNode, targetNode, err)
	}

	if !result.Success {
		return fmt.Errorf("link failed: %s", result.Message)
	}

	// Log to audit trail
	if e.auditLogger != nil {
		_ = e.auditLogger.Log(ctx, LogEntry{
			UserEmail: "system",
			UserID:    0,
			Action:    "scheduled_link",
			Resource:  fmt.Sprintf("nodes/%d/%d", sourceNode, targetNode),
			Details:   map[string]interface{}{"task_name": task.Name, "permanent": permanent},
			Success:   true,
		})
	}

	log.Printf("Scheduled task %s: Successfully linked %d -> %d", task.Name, sourceNode, targetNode)
	return nil
}

// executeUnlinkNodes unlinks nodes according to task parameters
func (e *DefaultTaskExecutor) executeUnlinkNodes(ctx context.Context, task *models.ScheduledTask) error {
	sourceNodeRaw, ok := task.Parameters["source_node"]
	if !ok {
		return fmt.Errorf("source_node parameter is required")
	}
	targetNodeRaw, ok := task.Parameters["target_node"]
	if !ok {
		return fmt.Errorf("target_node parameter is required")
	}

	sourceNode := int(sourceNodeRaw.(float64))
	targetNode := int(targetNodeRaw.(float64))

	if e.amiConnector == nil || !e.amiConnector.IsConnected() {
		return fmt.Errorf("AMI not connected")
	}

	// Execute unlink command
	result, err := e.amiConnector.UnlinkNode(sourceNode, targetNode)
	if err != nil {
		return fmt.Errorf("failed to unlink nodes %d -> %d: %w", sourceNode, targetNode, err)
	}

	if !result.Success {
		return fmt.Errorf("unlink failed: %s", result.Message)
	}

	// Log to audit trail
	if e.auditLogger != nil {
		_ = e.auditLogger.Log(ctx, LogEntry{
			UserEmail: "system",
			UserID:    0,
			Action:    "scheduled_unlink",
			Resource:  fmt.Sprintf("nodes/%d/%d", sourceNode, targetNode),
			Details:   map[string]interface{}{"task_name": task.Name},
			Success:   true,
		})
	}

	log.Printf("Scheduled task %s: Successfully unlinked %d -> %d", task.Name, sourceNode, targetNode)
	return nil
}

// executeAMICommand executes a custom AMI command
func (e *DefaultTaskExecutor) executeAMICommand(ctx context.Context, task *models.ScheduledTask) error {
	actionRaw, ok := task.Parameters["action"]
	if !ok {
		return fmt.Errorf("action parameter is required")
	}
	action := actionRaw.(string)

	// Extract additional parameters
	params := make(map[string]string)
	if paramsRaw, ok := task.Parameters["params"]; ok {
		if paramsMap, ok := paramsRaw.(map[string]interface{}); ok {
			for k, v := range paramsMap {
				params[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	if e.amiConnector == nil || !e.amiConnector.IsConnected() {
		return fmt.Errorf("AMI not connected")
	}

	// Execute AMI action
	result, err := e.amiConnector.ExecuteAction(action, params, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to execute AMI action %s: %w", action, err)
	}

	if !result.Success {
		return fmt.Errorf("AMI action failed: %s", result.Message)
	}

	// Log to audit trail
	if e.auditLogger != nil {
		_ = e.auditLogger.Log(ctx, LogEntry{
			UserEmail: "system",
			UserID:    0,
			Action:    "scheduled_ami_command",
			Resource:  action,
			Details:   map[string]interface{}{"task_name": task.Name, "params": params},
			Success:   true,
		})
	}

	log.Printf("Scheduled task %s: Successfully executed AMI action %s", task.Name, action)
	return nil
}

// executeBackupDB creates a scheduled database backup
func (e *DefaultTaskExecutor) executeBackupDB(ctx context.Context, task *models.ScheduledTask) error {
	if e.dbBackupManager == nil {
		return fmt.Errorf("database backup manager not configured")
	}

	comment := fmt.Sprintf("Scheduled backup: %s", task.Name)
	if commentRaw, ok := task.Parameters["comment"]; ok {
		if commentStr, ok := commentRaw.(string); ok && commentStr != "" {
			comment = commentStr
		}
	}

	backupID, err := e.dbBackupManager.CreateBackup(comment, "scheduled")
	if err != nil {
		return fmt.Errorf("failed to create scheduled backup: %w", err)
	}

	// Log to audit trail
	if e.auditLogger != nil {
		_ = e.auditLogger.Log(ctx, LogEntry{
			UserEmail: "system",
			UserID:    0,
			Action:    "scheduled_backup",
			Resource:  fmt.Sprintf("backup/%s", backupID),
			Details:   map[string]interface{}{"task_name": task.Name},
			Success:   true,
		})
	}

	log.Printf("Scheduled task %s: Successfully created backup %s", task.Name, backupID)
	return nil
}
