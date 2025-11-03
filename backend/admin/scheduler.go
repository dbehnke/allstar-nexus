package admin

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// TaskExecutor defines the interface for executing different task types
type TaskExecutor interface {
	Execute(ctx context.Context, task *models.ScheduledTask) error
}

// Scheduler manages scheduled tasks with cron-like scheduling
type Scheduler struct {
	db       *gorm.DB
	cron     *cron.Cron
	executor TaskExecutor
	tasks    map[uint]cron.EntryID // map task ID to cron entry ID
	mu       sync.RWMutex
}

// NewScheduler creates a new scheduler instance
func NewScheduler(db *gorm.DB, executor TaskExecutor) *Scheduler {
	return &Scheduler{
		db:       db,
		cron:     cron.New(),
		executor: executor,
		tasks:    make(map[uint]cron.EntryID),
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	// Load all active tasks from database
	var tasks []models.ScheduledTask
	if err := s.db.Where("status = ?", models.TaskStatusActive).Find(&tasks).Error; err != nil {
		return fmt.Errorf("failed to load scheduled tasks: %w", err)
	}

	// Schedule each task
	for _, task := range tasks {
		if err := s.scheduleTask(&task); err != nil {
			log.Printf("Warning: failed to schedule task %d (%s): %v", task.ID, task.Name, err)
		}
	}

	s.cron.Start()
	log.Printf("Scheduler started with %d active tasks", len(tasks))
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Scheduler stopped")
}

// scheduleTask adds a task to the cron scheduler
func (s *Scheduler) scheduleTask(task *models.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if present
	if entryID, exists := s.tasks[task.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.tasks, task.ID)
	}

	// Parse cron expression
	entryID, err := s.cron.AddFunc(task.CronExpression, func() {
		s.executeTask(task.ID)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", task.CronExpression, err)
	}

	s.tasks[task.ID] = entryID

	// Calculate next run time
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	task.NextRunAt = &nextRun

	// Update next run time in database
	s.db.Model(task).Update("next_run_at", nextRun)

	return nil
}

// executeTask runs a scheduled task
func (s *Scheduler) executeTask(taskID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()

	// Load task from database
	var task models.ScheduledTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		log.Printf("Error loading task %d: %v", taskID, err)
		return
	}

	// Check if task is still active
	if task.Status != models.TaskStatusActive {
		log.Printf("Task %d (%s) is not active, skipping execution", taskID, task.Name)
		return
	}

	log.Printf("Executing scheduled task %d: %s (type: %s)", taskID, task.Name, task.TaskType)

	// Execute task
	var execErr error
	if s.executor != nil {
		execErr = s.executor.Execute(ctx, &task)
	} else {
		execErr = fmt.Errorf("no executor configured")
	}

	duration := time.Since(startTime).Milliseconds()

	// Update task execution stats
	now := time.Now()
	updates := map[string]interface{}{
		"last_run_at": now,
		"run_count":   task.RunCount + 1,
	}

	success := execErr == nil
	if success {
		updates["last_result"] = "success"
		updates["last_error"] = ""
	} else {
		updates["last_result"] = "failure"
		updates["last_error"] = execErr.Error()
		updates["fail_count"] = task.FailCount + 1
		log.Printf("Task %d (%s) failed: %v", taskID, task.Name, execErr)
	}

	// Calculate next run time
	s.mu.RLock()
	entryID, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if exists {
		entry := s.cron.Entry(entryID)
		nextRun := entry.Next
		updates["next_run_at"] = nextRun
	}

	// Update task in database
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		log.Printf("Error updating task %d stats: %v", taskID, err)
	}

	// Log execution
	execLog := models.TaskExecutionLog{
		TaskID:     taskID,
		ExecutedAt: now,
		Success:    success,
		DurationMs: duration,
	}
	if success {
		execLog.Result = "Task completed successfully"
	} else {
		execLog.Result = "Task failed"
		execLog.Error = execErr.Error()
	}

	if err := s.db.Create(&execLog).Error; err != nil {
		log.Printf("Error creating execution log for task %d: %v", taskID, err)
	}
}

// CreateTask creates a new scheduled task
func (s *Scheduler) CreateTask(ctx context.Context, task *models.ScheduledTask) error {
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.CronExpression == "" {
		return fmt.Errorf("cron expression is required")
	}
	if task.TaskType == "" {
		return fmt.Errorf("task type is required")
	}

	// Validate cron expression
	if _, err := cron.ParseStandard(task.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Create task in database
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return err
	}

	// Schedule task if active
	if task.Status == models.TaskStatusActive {
		if err := s.scheduleTask(task); err != nil {
			return fmt.Errorf("failed to schedule task: %w", err)
		}
	}

	return nil
}

// GetTask retrieves a task by ID
func (s *Scheduler) GetTask(ctx context.Context, id uint) (*models.ScheduledTask, error) {
	var task models.ScheduledTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks returns all scheduled tasks
func (s *Scheduler) ListTasks(ctx context.Context) ([]models.ScheduledTask, error) {
	var tasks []models.ScheduledTask
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTask updates a task's status or configuration
func (s *Scheduler) UpdateTask(ctx context.Context, id uint, updates *models.ScheduledTask) error {
	var task models.ScheduledTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return err
	}

	// Update fields
	updateMap := make(map[string]interface{})
	if updates.Name != "" {
		updateMap["name"] = updates.Name
	}
	if updates.Description != "" {
		updateMap["description"] = updates.Description
	}
	if updates.Status != "" {
		updateMap["status"] = updates.Status
	}
	if updates.CronExpression != "" {
		// Validate new cron expression
		if _, err := cron.ParseStandard(updates.CronExpression); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		updateMap["cron_expression"] = updates.CronExpression
	}
	if updates.Parameters != nil {
		updateMap["parameters"] = updates.Parameters
	}

	if err := s.db.WithContext(ctx).Model(&task).Updates(updateMap).Error; err != nil {
		return err
	}

	// Reload task to get updated values
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return err
	}

	// Reschedule if active, or remove from schedule if not
	s.mu.Lock()
	if entryID, exists := s.tasks[id]; exists {
		s.cron.Remove(entryID)
		delete(s.tasks, id)
	}
	s.mu.Unlock()

	if task.Status == models.TaskStatusActive {
		if err := s.scheduleTask(&task); err != nil {
			return fmt.Errorf("failed to reschedule task: %w", err)
		}
	}

	return nil
}

// DeleteTask deletes a scheduled task
func (s *Scheduler) DeleteTask(ctx context.Context, id uint) error {
	// Remove from scheduler
	s.mu.Lock()
	if entryID, exists := s.tasks[id]; exists {
		s.cron.Remove(entryID)
		delete(s.tasks, id)
	}
	s.mu.Unlock()

	// Delete from database
	result := s.db.WithContext(ctx).Delete(&models.ScheduledTask{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// GetTaskExecutionLogs returns execution history for a task
func (s *Scheduler) GetTaskExecutionLogs(ctx context.Context, taskID uint, limit int) ([]models.TaskExecutionLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var logs []models.TaskExecutionLog
	if err := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("executed_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
