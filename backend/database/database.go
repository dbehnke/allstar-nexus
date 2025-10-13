package database

import (
	"context"
	"errors"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

// DB wraps gorm.DB for convenience.
// NOTE: This wrapper is deprecated. Use gorm.DB directly instead.
type DB struct {
	*gorm.DB
}

// Open opens (and creates if needed) a SQLite database at path using GORM with modernc.org/sqlite (pure Go, no CGO).
func Open(path string) (*DB, error) {
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        path,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Get underlying sql.DB for PRAGMA settings
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	// Optimize for write bursts on link tx events
	_, _ = sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL;")
	_, _ = sqlDB.ExecContext(ctx, "PRAGMA synchronous=NORMAL;")

	return &DB{gormDB}, nil
}

// CountUsers returns total users using GORM.
func (db *DB) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Table("users").Count(&count).Error
	return count, err
}

// CloseSafe closes ignoring nil.
func (db *DB) CloseSafe() error {
	if db == nil || db.DB == nil {
		return errors.New("db is nil")
	}
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
