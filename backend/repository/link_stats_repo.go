package repository

import (
	"context"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
