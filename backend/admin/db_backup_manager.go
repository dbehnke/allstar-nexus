package admin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DBBackupManager handles database backup and restore operations
type DBBackupManager struct {
	DBPath     string
	BackupDir  string
	MaxBackups int
	db         *gorm.DB
}

// NewDBBackupManager creates a new database backup manager instance
func NewDBBackupManager(dbPath string, backupDir string, db *gorm.DB) *DBBackupManager {
	if backupDir == "" {
		backupDir = "data/db-backups"
	}
	return &DBBackupManager{
		DBPath:     dbPath,
		BackupDir:  backupDir,
		MaxBackups: 30, // Keep last 30 backups
		db:         db,
	}
}

// DBBackupInfo represents metadata about a database backup
type DBBackupInfo struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Comment   string    `json:"comment,omitempty"`
	Type      string    `json:"type"` // "manual" or "scheduled"
}

// CreateBackup creates a timestamped backup of the database
func (bm *DBBackupManager) CreateBackup(comment string, backupType string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(bm.BackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	// Perform SQLite VACUUM to optimize before backup (optional, but recommended)
	// This creates a clean copy in the backup
	
	// Generate backup ID (timestamp-based)
	timestamp := time.Now()
	backupID := timestamp.Format("2006-01-02-150405")
	backupPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.db", backupID))

	// Use SQLite backup API through GORM
	// For SQLite, we can use the VACUUM INTO command which creates an optimized backup
	sqlDB, err := bm.db.DB()
	if err != nil {
		return "", fmt.Errorf("failed to get underlying DB: %w", err)
	}

	// Execute VACUUM INTO for clean backup
	_, err = sqlDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return "", fmt.Errorf("failed to create database backup: %w", err)
	}

	// Write comment and type metadata if provided
	if comment != "" || backupType != "" {
		metadata := fmt.Sprintf("type: %s\ncomment: %s\n", backupType, comment)
		metadataPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.meta", backupID))
		if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
			// Non-fatal, just log
			fmt.Printf("warning: failed to write backup metadata: %v\n", err)
		}
	}

	// Clean up old backups
	bm.cleanupOldBackups()

	return backupID, nil
}

// ListBackups returns all available backups, newest first
func (bm *DBBackupManager) ListBackups() ([]DBBackupInfo, error) {
	entries, err := os.ReadDir(bm.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []DBBackupInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read backup dir: %w", err)
	}

	var backups []DBBackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}

		// Extract backup ID from filename: nexus-2006-01-02-150405.db
		name := entry.Name()
		if !strings.HasPrefix(name, "nexus-") {
			continue
		}
		backupID := strings.TrimPrefix(name, "nexus-")
		backupID = strings.TrimSuffix(backupID, ".db")

		// Parse timestamp from ID
		timestamp, err := time.Parse("2006-01-02-150405", backupID)
		if err != nil {
			continue // Skip invalid backups
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Read metadata if exists
		metadataPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.meta", backupID))
		comment := ""
		backupType := "manual"
		if metaData, err := os.ReadFile(metadataPath); err == nil {
			lines := strings.Split(string(metaData), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "type: ") {
					backupType = strings.TrimPrefix(line, "type: ")
				} else if strings.HasPrefix(line, "comment: ") {
					comment = strings.TrimPrefix(line, "comment: ")
				}
			}
		}

		backups = append(backups, DBBackupInfo{
			ID:        backupID,
			Timestamp: timestamp,
			Path:      filepath.Join(bm.BackupDir, name),
			Size:      info.Size(),
			Comment:   comment,
			Type:      backupType,
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

// Restore restores the database from a backup
// WARNING: This requires shutting down the application and restarting
func (bm *DBBackupManager) Restore(backupID string) error {
	backupPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.db", backupID))

	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Create a safety backup of current database before restoring
	safetyBackupID, err := bm.CreateBackup(fmt.Sprintf("Auto-backup before restore to %s", backupID), "safety")
	if err != nil {
		return fmt.Errorf("failed to create safety backup: %w", err)
	}
	fmt.Printf("Created safety backup: %s\n", safetyBackupID)

	// Close existing database connections
	sqlDB, err := bm.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Copy backup file to main database path
	if err := copyFile(backupPath, bm.DBPath); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

// DeleteBackup removes a backup file
func (bm *DBBackupManager) DeleteBackup(backupID string) error {
	backupPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.db", backupID))
	metadataPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.meta", backupID))

	// Delete backup file
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// Delete metadata file if exists
	os.Remove(metadataPath) // Ignore error

	return nil
}

// GetBackupStats returns statistics about a backup
func (bm *DBBackupManager) GetBackupStats(backupID string) (map[string]interface{}, error) {
	backupPath := filepath.Join(bm.BackupDir, fmt.Sprintf("nexus-%s.db", backupID))

	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup: %w", err)
	}

	return map[string]interface{}{
		"size":       info.Size(),
		"size_human": FormatBytes(info.Size()),
		"modified":   info.ModTime(),
	}, nil
}

// ScheduleBackup schedules automatic backups at specified intervals
// Returns a stop function that can be called to stop the scheduler
func (bm *DBBackupManager) ScheduleBackup(interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	stopChan := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				backupID, err := bm.CreateBackup("Scheduled backup", "scheduled")
				if err != nil {
					fmt.Printf("scheduled backup failed: %v\n", err)
				} else {
					fmt.Printf("scheduled backup created: %s\n", backupID)
				}
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		stopChan <- true
	}
}

// cleanupOldBackups removes old backups exceeding MaxBackups
func (bm *DBBackupManager) cleanupOldBackups() {
	backups, err := bm.ListBackups()
	if err != nil {
		return
	}

	// Delete oldest backups if we exceed the limit
	if len(backups) > bm.MaxBackups {
		for i := bm.MaxBackups; i < len(backups); i++ {
			// Don't auto-delete safety backups
			if backups[i].Type != "safety" {
				bm.DeleteBackup(backups[i].ID)
			}
		}
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}
