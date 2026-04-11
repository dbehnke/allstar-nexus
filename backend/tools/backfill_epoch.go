package tools

import (
	"database/sql"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BackfillTransmissionEpochs ensures start_unix/end_unix are populated from timestamp_start/timestamp_end
// for all rows where they are zero or NULL. Safe to run repeatedly.
func BackfillTransmissionEpochs(db *gorm.DB, logger *zap.Logger) error {
	type row struct {
		ID int
		S  sql.NullString `gorm:"column:ts_start"`
		E  sql.NullString `gorm:"column:ts_end"`
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Find candidates where epoch fields are missing
		rows, err := tx.Raw(
			"SELECT id, CAST(timestamp_start AS TEXT) AS ts_start, CAST(timestamp_end AS TEXT) AS ts_end FROM transmission_logs WHERE start_unix IS NULL OR start_unix = 0 OR end_unix IS NULL OR end_unix = 0",
		).Rows()
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()

		updated := 0
		scanned := 0
		failed := 0

		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.S, &r.E); err != nil {
				failed++
				continue
			}
			scanned++

			var startUnix, endUnix int64
			var haveStart, haveEnd bool

			if r.S.Valid {
				if t, err := time.Parse(time.RFC3339Nano, r.S.String); err == nil {
					startUnix = t.Unix()
					haveStart = true
				}
			}
			if r.E.Valid {
				if t, err := time.Parse(time.RFC3339Nano, r.E.String); err == nil {
					endUnix = t.Unix()
					haveEnd = true
				}
			}

			if haveStart || haveEnd {
				q := tx.Model(&struct{}{}).Table("transmission_logs").Where("id = ?", r.ID)
				if haveStart && haveEnd {
					if err := q.UpdateColumns(map[string]any{"start_unix": startUnix, "end_unix": endUnix}).Error; err != nil {
						failed++
						continue
					}
				} else if haveStart {
					if err := q.UpdateColumn("start_unix", startUnix).Error; err != nil {
						failed++
						continue
					}
				} else if haveEnd {
					if err := q.UpdateColumn("end_unix", endUnix).Error; err != nil {
						failed++
						continue
					}
				}
				updated++
			}
		}

		logger.Info("epoch backfill summary", zap.Int("rows_scanned", scanned), zap.Int("rows_updated", updated), zap.Int("rows_failed", failed))
		return nil
	})
}
