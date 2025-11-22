package repository

import (
	"context"

	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LinkStatsFilter struct {
	Since  time.Time
	Nodes  []int
	SortBy string // "tx_seconds_desc", "tx_seconds_asc", "node_asc", "node_desc", "recent_desc"
	Limit  int
}

type LinkStatsRepo struct{ db *gorm.DB }

func NewLinkStatsRepo(db *gorm.DB) *LinkStatsRepo { return &LinkStatsRepo{db: db} }

func (r *LinkStatsRepo) Upsert(ctx context.Context, s models.LinkStat) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "node"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_tx_seconds", "last_tx_start", "last_tx_end", "connected_since", "updated_at"}),
	}).Create(&s).Error
}

func (r *LinkStatsRepo) GetAll(ctx context.Context) ([]models.LinkStat, error) {
	var stats []models.LinkStat
	err := r.db.WithContext(ctx).Find(&stats).Error
	return stats, err
}

func (r *LinkStatsRepo) GetStats(ctx context.Context, filter LinkStatsFilter) ([]models.LinkStat, error) {
	var stats []models.LinkStat
	query := r.db.WithContext(ctx).Model(&models.LinkStat{})

	if !filter.Since.IsZero() {
		query = query.Where("updated_at >= ?", filter.Since)
	}

	if len(filter.Nodes) > 0 {
		query = query.Where("node IN ?", filter.Nodes)
	}

	switch filter.SortBy {
	case "tx_seconds_desc":
		query = query.Order("total_tx_seconds DESC")
	case "tx_seconds_asc":
		query = query.Order("total_tx_seconds ASC")
	case "node_asc":
		query = query.Order("node ASC")
	case "node_desc":
		query = query.Order("node DESC")
	case "recent_desc":
		query = query.Order("updated_at DESC")
	default:
		// Default sort if not specified or unrecognized
		query = query.Order("total_tx_seconds DESC")
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	err := query.Find(&stats).Error
	return stats, err
}

// DeleteNotIn deletes all link stats except those in the provided node list
// This is used to clean up stale/disconnected nodes from the database
func (r *LinkStatsRepo) DeleteNotIn(ctx context.Context, activeNodes []int) (int64, error) {
	if len(activeNodes) == 0 {
		// Delete all
		result := r.db.WithContext(ctx).Where("1 = 1").Delete(&models.LinkStat{})
		return result.RowsAffected, result.Error
	}

	// Delete all nodes NOT in the active list
	result := r.db.WithContext(ctx).Where("node NOT IN ?", activeNodes).Delete(&models.LinkStat{})
	return result.RowsAffected, result.Error
}
