package admin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// LogLevel represents a log severity level
type LogLevel string

const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
	LogLevelAll     LogLevel = "ALL"
)

// LogSource represents a log file source
type LogSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
}

// StreamLogEntry represents a single log line from a log file
type StreamLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Raw       string    `json:"raw"`
}

// LogFilter holds filtering criteria
type LogFilter struct {
	Level      LogLevel
	Source     string
	Keyword    string
	Regex      *regexp.Regexp
	Since      time.Time
	Until      time.Time
	TailLines  int
	FollowMode bool
}

// LogStreamer manages real-time log streaming with filtering
type LogStreamer struct {
	sources      []LogSource
	watchers     map[string]*fsnotify.Watcher
	subscribers  map[string]chan StreamLogEntry
	mu           sync.RWMutex
	stopChans    map[string]chan struct{}
	auditLogger  *AuditLogger
}

// NewLogStreamer creates a new log streaming manager
func NewLogStreamer(auditLogger *AuditLogger) *LogStreamer {
	ls := &LogStreamer{
		sources:     detectLogSources(),
		watchers:    make(map[string]*fsnotify.Watcher),
		subscribers: make(map[string]chan StreamLogEntry),
		stopChans:   make(map[string]chan struct{}),
		auditLogger: auditLogger,
	}
	return ls
}

// detectLogSources discovers available log files
func detectLogSources() []LogSource {
	sources := []LogSource{
		{
			ID:          "nexus",
			Name:        "Allstar Nexus",
			Path:        "logs/nexus.log",
			Description: "Allstar Nexus application logs",
		},
		{
			ID:          "asterisk-messages",
			Name:        "Asterisk Messages",
			Path:        "/var/log/asterisk/messages",
			Description: "Asterisk general messages log",
		},
		{
			ID:          "asterisk-full",
			Name:        "Asterisk Full",
			Path:        "/var/log/asterisk/full",
			Description: "Asterisk full debug log",
		},
		{
			ID:          "asterisk-event",
			Name:        "Asterisk Event",
			Path:        "/var/log/asterisk/event_log",
			Description: "Asterisk AMI event log",
		},
	}

	// Check if each source is available
	for i := range sources {
		if _, err := os.Stat(sources[i].Path); err == nil {
			sources[i].Available = true
		}
	}

	return sources
}

// GetSources returns all detected log sources
func (ls *LogStreamer) GetSources() []LogSource {
	return ls.sources
}

// StreamLogs streams log entries matching the filter
func (ls *LogStreamer) StreamLogs(ctx context.Context, filter LogFilter, output chan<- StreamLogEntry) error {
	if filter.Source == "" {
		return fmt.Errorf("source is required")
	}

	// Find the source
	var source *LogSource
	for i := range ls.sources {
		if ls.sources[i].ID == filter.Source {
			source = &ls.sources[i]
			break
		}
	}

	if source == nil {
		return fmt.Errorf("source not found: %s", filter.Source)
	}

	if !source.Available {
		return fmt.Errorf("source not available: %s", filter.Source)
	}

	// Open the log file
	file, err := os.Open(source.Path)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// If tail mode, seek to near end
	if filter.TailLines > 0 {
		if err := seekToTail(file, filter.TailLines); err != nil {
			log.Printf("failed to seek to tail: %v", err)
		}
	}

	// Read existing lines
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry := parseLogLine(scanner.Text(), source.ID)
		if matchesFilter(entry, filter) {
			select {
			case output <- entry:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	// If not following, we're done
	if !filter.FollowMode {
		return nil
	}

	// Watch for new lines
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(source.Path); err != nil {
		return fmt.Errorf("failed to watch file: %w", err)
	}

	// Follow file changes
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Read new lines
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					entry := parseLogLine(scanner.Text(), source.ID)
					if matchesFilter(entry, filter) {
						select {
						case output <- entry:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
			}
		case err := <-watcher.Errors:
			log.Printf("watcher error: %v", err)
		}
	}
}

// ExportLogs exports filtered logs as text
func (ls *LogStreamer) ExportLogs(filter LogFilter) ([]byte, error) {
	if filter.Source == "" {
		return nil, fmt.Errorf("source is required")
	}

	// Find the source
	var source *LogSource
	for i := range ls.sources {
		if ls.sources[i].ID == filter.Source {
			source = &ls.sources[i]
			break
		}
	}

	if source == nil {
		return nil, fmt.Errorf("source not found: %s", filter.Source)
	}

	if !source.Available {
		return nil, fmt.Errorf("source not available: %s", filter.Source)
	}

	file, err := os.Open(source.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	var result strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry := parseLogLine(scanner.Text(), source.ID)
		if matchesFilter(entry, filter) {
			result.WriteString(entry.Raw)
			result.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log file: %w", err)
	}

	return []byte(result.String()), nil
}

// parseLogLine parses a raw log line into a StreamLogEntry
func parseLogLine(line, source string) StreamLogEntry {
	entry := StreamLogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Source:    source,
		Message:   line,
		Raw:       line,
	}

	// Try to parse timestamp and level from common formats
	// Format 1: [2024-01-01 12:00:00] LEVEL: message
	if strings.HasPrefix(line, "[") {
		parts := strings.SplitN(line, "]", 2)
		if len(parts) == 2 {
			tsStr := strings.TrimPrefix(parts[0], "[")
			if ts, err := time.Parse("2006-01-02 15:04:05", tsStr); err == nil {
				entry.Timestamp = ts
			}
			rest := strings.TrimSpace(parts[1])
			if idx := strings.Index(rest, ":"); idx > 0 {
				levelStr := strings.TrimSpace(rest[:idx])
				entry.Level = parseLevel(levelStr)
				entry.Message = strings.TrimSpace(rest[idx+1:])
			}
		}
	}

	// Format 2: 2024-01-01T12:00:00Z LEVEL message
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		if ts, err := time.Parse(time.RFC3339, parts[0]); err == nil {
			entry.Timestamp = ts
			entry.Level = parseLevel(parts[1])
			entry.Message = strings.Join(parts[2:], " ")
		}
	}

	return entry
}

// parseLevel converts a string to a LogLevel
func parseLevel(s string) LogLevel {
	upper := strings.ToUpper(s)
	switch {
	case strings.Contains(upper, "DEBUG"):
		return LogLevelDebug
	case strings.Contains(upper, "INFO"):
		return LogLevelInfo
	case strings.Contains(upper, "WARN"):
		return LogLevelWarning
	case strings.Contains(upper, "ERROR"), strings.Contains(upper, "FATAL"):
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// matchesFilter checks if a log entry matches the filter criteria
func matchesFilter(entry StreamLogEntry, filter LogFilter) bool {
	// Level filter
	if filter.Level != LogLevelAll && filter.Level != "" {
		if entry.Level != filter.Level {
			return false
		}
	}

	// Time range filter
	if !filter.Since.IsZero() && entry.Timestamp.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && entry.Timestamp.After(filter.Until) {
		return false
	}

	// Keyword filter
	if filter.Keyword != "" {
		if !strings.Contains(strings.ToLower(entry.Raw), strings.ToLower(filter.Keyword)) {
			return false
		}
	}

	// Regex filter
	if filter.Regex != nil {
		if !filter.Regex.MatchString(entry.Raw) {
			return false
		}
	}

	return true
}

// seekToTail seeks the file to approximately N lines from the end
func seekToTail(file *os.File, lines int) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return nil
	}

	// Estimate: assume average 80 chars per line
	estimatedBytes := int64(lines * 80)
	seekPos := fileSize - estimatedBytes
	if seekPos < 0 {
		seekPos = 0
	}

	if _, err := file.Seek(seekPos, io.SeekStart); err != nil {
		return err
	}

	// Read to next newline to avoid partial line
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		// Skip the partial line
	}

	return nil
}

// Cleanup closes all watchers and stops all streams
func (ls *LogStreamer) Cleanup() {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	for _, stopChan := range ls.stopChans {
		close(stopChan)
	}

	for _, watcher := range ls.watchers {
		watcher.Close()
	}

	ls.watchers = make(map[string]*fsnotify.Watcher)
	ls.stopChans = make(map[string]chan struct{})
}

// AuditLogStreamAccess logs when a user accesses logs
func (ls *LogStreamer) AuditLogStreamAccess(ctx context.Context, userEmail, source string, filter LogFilter) {
	if ls.auditLogger != nil {
		details := map[string]string{
			"source":  source,
			"level":   string(filter.Level),
			"keyword": filter.Keyword,
		}
		ls.auditLogger.Log(ctx, LogEntry{
			UserEmail: userEmail,
			Action:    "log_stream_access",
			Resource:  source,
			Success:   true,
			Details:   details,
		})
	}
}

// AuditLogExport logs when a user exports logs
func (ls *LogStreamer) AuditLogExport(ctx context.Context, userEmail, source string, filter LogFilter) {
	if ls.auditLogger != nil {
		details := map[string]string{
			"source":  source,
			"level":   string(filter.Level),
			"keyword": filter.Keyword,
		}
		ls.auditLogger.Log(ctx, LogEntry{
			UserEmail: userEmail,
			Action:    "log_export",
			Resource:  source,
			Success:   true,
			Details:   details,
		})
	}
}

// GetLogPath returns the absolute path for a given log source
func GetLogPath(source string) (string, error) {
	sources := detectLogSources()
	for _, s := range sources {
		if s.ID == source {
			// Convert to absolute path
			absPath, err := filepath.Abs(s.Path)
			if err != nil {
				return "", err
			}
			return absPath, nil
		}
	}
	return "", fmt.Errorf("unknown source: %s", source)
}
