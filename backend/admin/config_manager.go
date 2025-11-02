package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/pmezard/go-difflib/difflib"
)

// ConfigManager handles safe configuration file operations
type ConfigManager struct {
	ConfigPath  string
	BackupDir   string
	MaxBackups  int
	AutoBackup  bool
}

// NewConfigManager creates a new config manager instance
func NewConfigManager(configPath, backupDir string) *ConfigManager {
	if backupDir == "" {
		backupDir = "data/config-backups"
	}
	return &ConfigManager{
		ConfigPath: configPath,
		BackupDir:  backupDir,
		MaxBackups: 50, // Keep last 50 backups
		AutoBackup: true,
	}
}

// BackupInfo represents metadata about a config backup
type BackupInfo struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Comment   string    `json:"comment,omitempty"`
}

// Read returns the current config file contents
func (cm *ConfigManager) Read() (string, error) {
	data, err := os.ReadFile(cm.ConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}
	return string(data), nil
}

// Write saves new config content after validation and backup
func (cm *ConfigManager) Write(content string, comment string) (string, error) {
	// Validate the new config by writing to temp file and validating
	tmpFile, err := os.CreateTemp("", "config-validate-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write temp config: %w", err)
	}
	tmpFile.Close()

	// Validate using the config package validator
	if err := config.Validate(tmpFile.Name()); err != nil {
		return "", fmt.Errorf("config validation failed: %w", err)
	}

	// Create backup before writing
	var backupID string
	if cm.AutoBackup {
		backupID, err = cm.CreateBackup(comment)
		if err != nil {
			return "", fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Write the new config
	if err := os.WriteFile(cm.ConfigPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	return backupID, nil
}

// CreateBackup creates a timestamped backup of the current config
func (cm *ConfigManager) CreateBackup(comment string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(cm.BackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	// Read current config
	content, err := cm.Read()
	if err != nil {
		return "", err
	}

	// Generate backup ID (timestamp-based)
	timestamp := time.Now()
	backupID := timestamp.Format("2006-01-02-150405")
	backupPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.yaml", backupID))

	// Write backup file
	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	// Write comment file if provided
	if comment != "" {
		commentPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.comment", backupID))
		if err := os.WriteFile(commentPath, []byte(comment), 0644); err != nil {
			// Non-fatal, just log
			fmt.Printf("warning: failed to write backup comment: %v\n", err)
		}
	}

	// Clean up old backups
	cm.cleanupOldBackups()

	return backupID, nil
}

// ListBackups returns all available backups, newest first
func (cm *ConfigManager) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(cm.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read backup dir: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Extract backup ID from filename: config-2006-01-02-150405.yaml
		name := entry.Name()
		if !strings.HasPrefix(name, "config-") {
			continue
		}
		backupID := strings.TrimPrefix(name, "config-")
		backupID = strings.TrimSuffix(backupID, ".yaml")

		// Parse timestamp from ID
		timestamp, err := time.Parse("2006-01-02-150405", backupID)
		if err != nil {
			continue // Skip invalid backups
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Read comment if exists
		commentPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.comment", backupID))
		comment := ""
		if commentData, err := os.ReadFile(commentPath); err == nil {
			comment = string(commentData)
		}

		backups = append(backups, BackupInfo{
			ID:        backupID,
			Timestamp: timestamp,
			Path:      filepath.Join(cm.BackupDir, name),
			Size:      info.Size(),
			Comment:   comment,
		})
	}

	// Sort by timestamp, newest first
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].Timestamp.Before(backups[j].Timestamp) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	return backups, nil
}

// Restore restores a config from a backup
func (cm *ConfigManager) Restore(backupID string) error {
	backupPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.yaml", backupID))

	// Read backup content
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Validate before restoring
	tmpFile, err := os.CreateTemp("", "config-restore-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	tmpFile.Close()

	if err := config.Validate(tmpFile.Name()); err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	// Create a backup of current config before restoring
	if cm.AutoBackup {
		if _, err := cm.CreateBackup(fmt.Sprintf("Auto-backup before restore to %s", backupID)); err != nil {
			return fmt.Errorf("failed to create pre-restore backup: %w", err)
		}
	}

	// Restore the backup
	if err := os.WriteFile(cm.ConfigPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write restored config: %w", err)
	}

	return nil
}

// GetBackup returns the content of a specific backup
func (cm *ConfigManager) GetBackup(backupID string) (string, error) {
	backupPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.yaml", backupID))
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to read backup: %w", err)
	}
	return string(content), nil
}

// DeleteBackup removes a backup file
func (cm *ConfigManager) DeleteBackup(backupID string) error {
	backupPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.yaml", backupID))
	commentPath := filepath.Join(cm.BackupDir, fmt.Sprintf("config-%s.comment", backupID))

	// Delete backup file
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// Delete comment file if exists
	os.Remove(commentPath) // Ignore error

	return nil
}

// Diff generates a unified diff between two configs
func (cm *ConfigManager) Diff(fromID, toID string) (string, error) {
	var fromContent, toContent string
	var err error

	// Get "from" content
	if fromID == "current" {
		fromContent, err = cm.Read()
		if err != nil {
			return "", err
		}
	} else {
		fromContent, err = cm.GetBackup(fromID)
		if err != nil {
			return "", err
		}
	}

	// Get "to" content
	if toID == "current" {
		toContent, err = cm.Read()
		if err != nil {
			return "", err
		}
	} else {
		toContent, err = cm.GetBackup(toID)
		if err != nil {
			return "", err
		}
	}

	// Generate unified diff
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(fromContent),
		B:        difflib.SplitLines(toContent),
		FromFile: "Old",
		ToFile:   "New",
		Context:  3,
	}

	return difflib.GetUnifiedDiffString(diff)
}

// cleanupOldBackups removes old backups exceeding MaxBackups
func (cm *ConfigManager) cleanupOldBackups() {
	backups, err := cm.ListBackups()
	if err != nil {
		return
	}

	// Delete oldest backups if we exceed the limit
	if len(backups) > cm.MaxBackups {
		for i := cm.MaxBackups; i < len(backups); i++ {
			cm.DeleteBackup(backups[i].ID)
		}
	}
}
