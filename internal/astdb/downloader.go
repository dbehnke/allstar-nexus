package astdb

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"go.uber.org/zap"
)

// Downloader handles downloading and updating the AllStar node database
type Downloader struct {
	URL          string
	FilePath     string
	UpdateHours  int // Update interval in hours (default 24)
	CleanupDays  int // Days before cleaning up stale nodes (default 7)
	logger       *zap.Logger
	nodeInfoRepo *repository.NodeInfoRepository
}

// NewDownloader creates a new astdb downloader
func NewDownloader(url, filePath string, updateHours int, logger *zap.Logger) *Downloader {
	if url == "" {
		url = "http://allmondb.allstarlink.org/"
	}
	if updateHours <= 0 {
		updateHours = 24 // Default: update daily
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Downloader{
		URL:         url,
		FilePath:    filePath,
		UpdateHours: updateHours,
		CleanupDays: 7, // Default: clean up nodes not seen in 7 days
		logger:      logger,
	}
}

// SetNodeInfoRepository injects the node info repository for database operations
func (d *Downloader) SetNodeInfoRepository(repo *repository.NodeInfoRepository) {
	d.nodeInfoRepo = repo
}

// Download fetches the astdb file from the AllStar server
func (d *Downloader) Download() error {
	d.logger.Info("downloading astdb from AllStar server",
		zap.String("url", d.URL),
		zap.String("destination", d.FilePath))

	// Create a temporary file
	tmpPath := d.FilePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpPath) }() // Clean up temp file if we error

	// Download the file
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Get(d.URL)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	// Write to temp file
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	d.logger.Info("downloaded astdb",
		zap.Int64("bytes", written))

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(d.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, d.FilePath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	d.logger.Info("astdb file updated successfully",
		zap.String("path", d.FilePath))

	return nil
}

// DownloadAndImport downloads astdb and imports it into SQLite database
func (d *Downloader) DownloadAndImport() error {
	// First download to temp file as before
	if err := d.Download(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// If no repository configured, just keep the file
	if d.nodeInfoRepo == nil {
		d.logger.Info("no repository configured, skipping database import")
		return nil
	}

	// Parse and import into database
	return d.ImportToDatabase()
}

// ImportToDatabase parses the astdb file and imports it into the SQLite database
func (d *Downloader) ImportToDatabase() error {
	if d.nodeInfoRepo == nil {
		return fmt.Errorf("node info repository not configured")
	}

	d.logger.Info("importing astdb to database", zap.String("file", d.FilePath))

	file, err := os.Open(d.FilePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	now := time.Now()
	nodes := make([]models.NodeInfo, 0, 1000) // Batch buffer
	scanner := bufio.NewScanner(file)
	lineCount := 0
	importCount := 0

	for scanner.Scan() {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse pipe-delimited format: NodeID|Callsign|Description|Location
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			d.logger.Warn("invalid astdb line", zap.Int("line", lineCount), zap.String("content", line))
			continue
		}

		nodeID, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			d.logger.Warn("invalid node ID", zap.Int("line", lineCount), zap.String("node_id", parts[0]))
			continue
		}

		callsign := strings.TrimSpace(parts[1])
		description := ""
		location := ""

		if len(parts) > 2 {
			description = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 {
			location = strings.TrimSpace(parts[3])
		}

		nodes = append(nodes, models.NodeInfo{
			NodeID:      nodeID,
			Callsign:    callsign,
			Description: description,
			Location:    location,
			LastSeen:    now,
		})

		// Batch upsert when buffer is full
		if len(nodes) >= 1000 {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := d.nodeInfoRepo.BulkUpsert(ctx, nodes, 500); err != nil {
				cancel()
				return fmt.Errorf("bulk upsert failed: %w", err)
			}
			cancel()
			importCount += len(nodes)
			nodes = nodes[:0] // Clear buffer
			d.logger.Info("imported batch", zap.Int("count", importCount))
		}
	}

	// Import remaining nodes
	if len(nodes) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := d.nodeInfoRepo.BulkUpsert(ctx, nodes, 500); err != nil {
			cancel()
			return fmt.Errorf("final bulk upsert failed: %w", err)
		}
		cancel()
		importCount += len(nodes)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	d.logger.Info("astdb import completed",
		zap.Int("lines_read", lineCount),
		zap.Int("nodes_imported", importCount))

	// Clean up stale nodes
	if d.CleanupDays > 0 {
		if err := d.CleanupStaleNodes(); err != nil {
			d.logger.Warn("cleanup failed", zap.Error(err))
		}
	}

	return nil
}

// CleanupStaleNodes removes nodes that haven't been seen in the configured number of days
func (d *Downloader) CleanupStaleNodes() error {
	if d.nodeInfoRepo == nil {
		return fmt.Errorf("node info repository not configured")
	}

	cutoff := time.Now().AddDate(0, 0, -d.CleanupDays)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check how many will be deleted
	staleCount, err := d.nodeInfoRepo.GetStaleCount(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("get stale count: %w", err)
	}

	if staleCount == 0 {
		d.logger.Info("no stale nodes to clean up")
		return nil
	}

	d.logger.Info("cleaning up stale nodes",
		zap.Int64("count", staleCount),
		zap.Int("days", d.CleanupDays))

	deleted, err := d.nodeInfoRepo.DeleteStaleNodes(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("delete stale nodes: %w", err)
	}

	d.logger.Info("cleaned up stale nodes", zap.Int64("deleted", deleted))
	return nil
}

// NeedsUpdate checks if the astdb file needs to be updated
func (d *Downloader) NeedsUpdate() bool {
	info, err := os.Stat(d.FilePath)
	if os.IsNotExist(err) {
		d.logger.Info("astdb file does not exist, needs download")
		return true
	}
	if err != nil {
		d.logger.Warn("error checking astdb file", zap.Error(err))
		return true
	}

	age := time.Since(info.ModTime())
	maxAge := time.Duration(d.UpdateHours) * time.Hour

	needsUpdate := age > maxAge
	d.logger.Info("checked astdb age",
		zap.Duration("age", age),
		zap.Duration("max_age", maxAge),
		zap.Bool("needs_update", needsUpdate))

	return needsUpdate
}

// EnsureExists downloads the astdb file if it doesn't exist or is too old
// If repository is configured, also imports to database
func (d *Downloader) EnsureExists() error {
	if d.NeedsUpdate() {
		if d.nodeInfoRepo != nil {
			return d.DownloadAndImport()
		}
		return d.Download()
	}
	d.logger.Info("astdb file is up to date", zap.String("path", d.FilePath))

	// Even if file is up to date, check if database needs initial import
	if d.nodeInfoRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		count, err := d.nodeInfoRepo.GetCount(ctx)
		if err == nil && count == 0 {
			d.logger.Info("database is empty, importing astdb file")
			return d.ImportToDatabase()
		}
	}

	return nil
}

// StartAutoUpdater starts a background goroutine that periodically updates the astdb
func (d *Downloader) StartAutoUpdater() {
	go func() {
		// Initial download if needed
		if err := d.EnsureExists(); err != nil {
			d.logger.Error("initial astdb download failed", zap.Error(err))
		}

		// Update periodically
		ticker := time.NewTicker(time.Duration(d.UpdateHours) * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if d.nodeInfoRepo != nil {
				if err := d.DownloadAndImport(); err != nil {
					d.logger.Error("periodic astdb update failed", zap.Error(err))
				}
			} else {
				if err := d.Download(); err != nil {
					d.logger.Error("periodic astdb update failed", zap.Error(err))
				}
			}
		}
	}()
}

// GetNodeCount returns the number of nodes in the database
func (d *Downloader) GetNodeCount() (int, error) {
	file, err := os.Open(d.FilePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}

	return count, scanner.Err()
}

// ValidateFile checks if the astdb file is valid
func (d *Downloader) ValidateFile() error {
	file, err := os.Open(d.FilePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	validLines := 0

	for scanner.Scan() {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for pipe-delimited format
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			validLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if validLines == 0 {
		return fmt.Errorf("no valid nodes found in astdb")
	}

	d.logger.Info("validated astdb file",
		zap.Int("total_lines", lineCount),
		zap.Int("valid_nodes", validLines))

	return nil
}
