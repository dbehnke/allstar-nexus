package repository

import (
	"fmt"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
)

// TransmissionLogRepository handles database operations for transmission logs
type TransmissionLogRepository struct {
	db *gorm.DB
}

// NewTransmissionLogRepository creates a new transmission log repository
func NewTransmissionLogRepository(db *gorm.DB) *TransmissionLogRepository {
	return &TransmissionLogRepository{db: db}
}

// Create inserts a new transmission log entry
func (r *TransmissionLogRepository) Create(log *models.TransmissionLog) error {
	return r.db.Create(log).Error
}

// LogTransmission creates and saves a transmission log entry
func (r *TransmissionLogRepository) LogTransmission(sourceID, adjacentLinkID int, callsign string, start, end time.Time, durationSec int) error {
	log := &models.TransmissionLog{
		SourceID:        sourceID,
		AdjacentLinkID:  adjacentLinkID,
		Callsign:        callsign,
		TimestampStart:  start,
		TimestampEnd:    end,
		DurationSeconds: durationSec,
	}
	return r.Create(log)
}

// GetRecentLogs returns the N most recent transmission logs
func (r *TransmissionLogRepository) GetRecentLogs(limit int) ([]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Order("timestamp_start DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetLogsByCallsign returns transmission logs for a specific callsign
func (r *TransmissionLogRepository) GetLogsByCallsign(callsign string, limit int) ([]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Where("callsign = ?", callsign).Order("timestamp_start DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetLogsBySourceNode returns transmission logs for a specific source node
func (r *TransmissionLogRepository) GetLogsBySourceNode(sourceID int, limit int) ([]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Where("source_id = ?", sourceID).Order("timestamp_start DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetLogsByAdjacentNode returns transmission logs for a specific adjacent node
func (r *TransmissionLogRepository) GetLogsByAdjacentNode(adjacentID int, limit int) ([]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Where("adjacent_link_id = ?", adjacentID).Order("timestamp_start DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetLogsInTimeRange returns logs within a specific time range
func (r *TransmissionLogRepository) GetLogsInTimeRange(start, end time.Time, limit int) ([]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Where("timestamp_start >= ? AND timestamp_start <= ?", start, end).
		Order("timestamp_start DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetLogsBetween returns transmission logs within the specified time range, grouped by callsign
func (r *TransmissionLogRepository) GetLogsBetween(from, to time.Time) (map[string][]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Where("datetime(timestamp_start) >= datetime(?) AND datetime(timestamp_start) < datetime(?)", from, to).Order("timestamp_start").Find(&logs).Error
	if err != nil {
		return nil, err
	}
	groups := make(map[string][]models.TransmissionLog)
	for _, log := range logs {
		groups[log.Callsign] = append(groups[log.Callsign], log)
	}
	return groups, nil
}

// GetOldestLogTime returns the earliest timestamp_start in the transmission_logs table.
// If there are no logs, it returns a zero time and nil error.
func (r *TransmissionLogRepository) GetOldestLogTime() (time.Time, error) {
	// SQLite often returns datetime as TEXT; scan into string and parse flexibly
	type row struct {
		TS string `gorm:"column:ts"`
	}
	var out row
	if err := r.db.Model(&models.TransmissionLog{}).
		Select("MIN(timestamp_start) as ts").
		Scan(&out).Error; err != nil {
		return time.Time{}, err
	}
	if out.TS == "" {
		return time.Time{}, nil
	}
	// Try multiple layouts commonly emitted by SQLite/GORM
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, out.TS); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format for oldest timestamp: %q", out.TS)
}

// GetTotalTransmissionTime returns the total transmission time for a callsign
func (r *TransmissionLogRepository) GetTotalTransmissionTime(callsign string) (int, error) {
	var totalSeconds int64
	err := r.db.Model(&models.TransmissionLog{}).
		Where("callsign = ?", callsign).
		Select("COALESCE(SUM(duration_seconds), 0)").
		Scan(&totalSeconds).Error
	return int(totalSeconds), err
}

// DeleteOldLogs deletes logs older than the specified time
func (r *TransmissionLogRepository) DeleteOldLogs(before time.Time) (int64, error) {
	result := r.db.Where("timestamp_start < ?", before).Delete(&models.TransmissionLog{})
	return result.RowsAffected, result.Error
}

// GetLogsSince returns transmission logs since the specified time, grouped by callsign
func (r *TransmissionLogRepository) GetLogsSince(since time.Time) (map[string][]models.TransmissionLog, error) {
	var logs []models.TransmissionLog
	err := r.db.Where("timestamp_start >= ?", since).
		Order("callsign ASC, timestamp_start ASC").
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	// Group by callsign
	grouped := make(map[string][]models.TransmissionLog)
	for _, log := range logs {
		grouped[log.Callsign] = append(grouped[log.Callsign], log)
	}

	return grouped, nil
}
