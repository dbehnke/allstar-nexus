package tools

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BackfillTransmissionLogTimestamps normalizes legacy timestamp strings in transmission_logs
// to RFC3339Nano UTC and creates a safety backup before making in-place updates.
func BackfillTransmissionLogTimestamps(dbPath string, db *gorm.DB, logger *zap.Logger) error {
	// 1) Backup the database file
	if err := backupFile(dbPath, logger); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// 2) Iterate rows and normalize timestamps
	type row struct {
		ID      int
		TSStart sql.NullString
		TSEnd   sql.NullString
	}

	// Use a transaction so the operation is atomic
	return db.Transaction(func(tx *gorm.DB) error {
		// Stream rows to avoid large memory usage
		rows, err := tx.Raw("SELECT id, CAST(timestamp_start AS TEXT) AS ts_start, CAST(timestamp_end AS TEXT) AS ts_end FROM transmission_logs ORDER BY id").Rows()
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()

		updated := 0
		scanned := 0
		failed := 0

		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.TSStart, &r.TSEnd); err != nil {
				failed++
				continue
			}
			scanned++

			// Parse both timestamps
			var tStart, tEnd time.Time
			var okStart, okEnd bool

			if r.TSStart.Valid {
				if ts, ok := parseLegacyTimestamp(r.TSStart.String); ok {
					tStart = ts
					okStart = true
				}
			}
			if r.TSEnd.Valid {
				if te, ok := parseLegacyTimestamp(r.TSEnd.String); ok {
					tEnd = te
					okEnd = true
				}
			}

			if !okStart && !okEnd {
				// Nothing we can confidently fix; skip
				continue
			}

			// If one side parsed and the other was NULL/invalid, keep existing text for that side by re-parsing best-effort
			// but typically both sides exist; update those that parsed.
			// Write RFC3339Nano in UTC to guarantee consistent ordering and SQLite compatibility.
			setStart := r.TSStart.Valid && okStart
			setEnd := r.TSEnd.Valid && okEnd

			switch {
			case setStart && setEnd:
				if err := tx.Exec(
					"UPDATE transmission_logs SET timestamp_start = ?, timestamp_end = ? WHERE id = ?",
					tStart.UTC().Format(time.RFC3339Nano),
					tEnd.UTC().Format(time.RFC3339Nano),
					r.ID,
				).Error; err != nil {
					failed++
					continue
				}
				updated++
			case setStart && !setEnd:
				if err := tx.Exec(
					"UPDATE transmission_logs SET timestamp_start = ? WHERE id = ?",
					tStart.UTC().Format(time.RFC3339Nano),
					r.ID,
				).Error; err != nil {
					failed++
					continue
				}
				updated++
			case !setStart && setEnd:
				if err := tx.Exec(
					"UPDATE transmission_logs SET timestamp_end = ? WHERE id = ?",
					tEnd.UTC().Format(time.RFC3339Nano),
					r.ID,
				).Error; err != nil {
					failed++
					continue
				}
				updated++
			}
		}

		logger.Info("timestamp backfill summary",
			zap.Int("rows_scanned", scanned),
			zap.Int("rows_updated", updated),
			zap.Int("rows_failed", failed),
		)
		return nil
	})
}

func backupFile(path string, logger *zap.Logger) error {
	// Create a sibling backup file with a UTC timestamp suffix
	ts := time.Now().UTC().Format("20060102T150405Z")
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	dest := filepath.Join(dir, base+".bak-"+ts)

	srcF, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = srcF.Close() }()

	dstF, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = dstF.Close() }()

	if _, err := io.Copy(dstF, srcF); err != nil {
		return err
	}
	logger.Info("database backup created", zap.String("backup_path", dest))
	return nil
}

// parseLegacyTimestamp attempts to parse a variety of legacy formats into time.Time.
// It also strips Go's monotonic clock suffix (" m=+...") if present.
func parseLegacyTimestamp(s string) (time.Time, bool) {
	ss := strings.TrimSpace(s)
	if i := strings.Index(ss, " m="); i != -1 {
		ss = strings.TrimSpace(ss[:i])
	}

	layouts := []string{
		time.RFC3339Nano,                      // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02 15:04:05.999999999Z07:00", // Space instead of 'T' but RFC offset
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST", // Numeric offset + zone name
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05.999999999", // Naive
		"2006-01-02 15:04:05",           // Naive
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, ss); err == nil {
			return t.UTC(), true
		}
	}

	// Try a quick fix: if space between date and time and there's an offset, swap first space with 'T'
	// Example: 2006-01-02 15:04:05.999999999+00:00
	if strings.Contains(ss, "+") || strings.Contains(ss, "-") {
		if len(ss) > 10 && ss[10] == ' ' {
			alt := ss[:10] + "T" + ss[11:]
			if t, err := time.Parse(time.RFC3339Nano, alt); err == nil {
				return t.UTC(), true
			}
		}
	}

	return time.Time{}, false
}
