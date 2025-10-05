package repository

import (
	"context"
	"database/sql"
	"time"
)

type LinkStat struct {
	Node           int
	TotalTxSeconds int
	LastTxStart    *time.Time
	LastTxEnd      *time.Time
	ConnectedSince *time.Time
	UpdatedAt      time.Time
}

type LinkStatsRepo struct{ db *sql.DB }

func NewLinkStatsRepo(db *sql.DB) *LinkStatsRepo { return &LinkStatsRepo{db: db} }

func (r *LinkStatsRepo) Upsert(ctx context.Context, s LinkStat) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO link_stats(node,total_tx_seconds,last_tx_start,last_tx_end,connected_since,updated_at)
		VALUES(?,?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(node) DO UPDATE SET total_tx_seconds=excluded.total_tx_seconds,last_tx_start=excluded.last_tx_start,last_tx_end=excluded.last_tx_end,updated_at=CURRENT_TIMESTAMP`,
		s.Node, s.TotalTxSeconds, s.LastTxStart, s.LastTxEnd, s.ConnectedSince)
	return err
}

func (r *LinkStatsRepo) GetAll(ctx context.Context) ([]LinkStat, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT node,total_tx_seconds,last_tx_start,last_tx_end,connected_since,updated_at FROM link_stats`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LinkStat{}
	for rows.Next() {
		var s LinkStat
		var start, end, connected sql.NullTime
		if err := rows.Scan(&s.Node, &s.TotalTxSeconds, &start, &end, &connected, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if start.Valid {
			s.LastTxStart = &start.Time
		}
		if end.Valid {
			s.LastTxEnd = &end.Time
		}
		if connected.Valid {
			s.ConnectedSince = &connected.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
