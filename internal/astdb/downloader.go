package astdb

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Downloader handles downloading and updating the AllStar node database
type Downloader struct {
	URL         string
	FilePath    string
	UpdateHours int // Update interval in hours (default 24)
	logger      *zap.Logger
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
		logger:      logger,
	}
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
	defer tmpFile.Close()
	defer os.Remove(tmpPath) // Clean up temp file if we error

	// Download the file
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Get(d.URL)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

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
	tmpFile.Close()

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
func (d *Downloader) EnsureExists() error {
	if d.NeedsUpdate() {
		return d.Download()
	}
	d.logger.Info("astdb file is up to date", zap.String("path", d.FilePath))
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
			if err := d.Download(); err != nil {
				d.logger.Error("periodic astdb update failed", zap.Error(err))
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
	defer file.Close()

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
	defer file.Close()

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
