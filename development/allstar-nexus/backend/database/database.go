package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps sql.DB for future helpers.
type DB struct {
	*sql.DB
}

// Open opens (and creates if needed) a SQLite database at path.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	// Optimize for write bursts on link tx events
	_, _ = db.ExecContext(ctx, "PRAGMA journal_mode=WAL;")
	_, _ = db.ExecContext(ctx, "PRAGMA synchronous=NORMAL;")
	return &DB{db}, nil
}

// Migrate creates initial tables.
func (db *DB) Migrate() error {
	// Initial table (without role column originally)
	createUsers := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createUsers); err != nil {
		return err
	}

	// Link stats table for persisting cumulative per-link TX stats
	createLinkStats := `CREATE TABLE IF NOT EXISTS link_stats (
		node INTEGER PRIMARY KEY,
		total_tx_seconds INTEGER NOT NULL DEFAULT 0,
		last_tx_start TIMESTAMP NULL,
		last_tx_end TIMESTAMP NULL,
		connected_since TIMESTAMP NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createLinkStats); err != nil {
		return err
	}
	// Add connected_since if migrating from earlier version
	if _, err := db.Exec(`ALTER TABLE link_stats ADD COLUMN connected_since TIMESTAMP NULL`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			log.Printf("migration: add connected_since skipped: %v", err)
		}
	}

	// Attempt to add role column if existing DB from earlier revisions
	if _, err := db.Exec("ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			// benign if duplicate
			log.Printf("migration: add role column skipped: %v", err)
		}
	}
	return nil
}

// CountUsers returns total users.
func (db *DB) CountUsers(ctx context.Context) (int64, error) {
	row := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users")
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// CloseSafe closes ignoring nil.
func (db *DB) CloseSafe() error {
	if db == nil || db.DB == nil {
		return errors.New("db is nil")
	}
	return db.Close()
}
