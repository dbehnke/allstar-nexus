package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NodeInfoRepository handles database operations for node information
type NodeInfoRepository struct {
	db *gorm.DB
}

// NewNodeInfoRepository creates a new node info repository
func NewNodeInfoRepository(db *gorm.DB) *NodeInfoRepository {
	return &NodeInfoRepository{db: db}
}

// GetByNodeID retrieves node info by node ID
func (r *NodeInfoRepository) GetByNodeID(ctx context.Context, nodeID int) (*models.NodeInfo, error) {
	var node models.NodeInfo
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&node).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &node, err
}

// GetByCallsign retrieves all nodes with a specific callsign (can be multiple)
func (r *NodeInfoRepository) GetByCallsign(ctx context.Context, callsign string) ([]models.NodeInfo, error) {
	var nodes []models.NodeInfo
	err := r.db.WithContext(ctx).Where("callsign = ?", callsign).Find(&nodes).Error
	return nodes, err
}

// GetByLocationPrefix retrieves nodes with location matching prefix
func (r *NodeInfoRepository) GetByLocationPrefix(ctx context.Context, prefix string) ([]models.NodeInfo, error) {
	var nodes []models.NodeInfo
	err := r.db.WithContext(ctx).Where("location LIKE ?", prefix+"%").Find(&nodes).Error
	return nodes, err
}

// Search performs a full-text search across callsign, description, and location
func (r *NodeInfoRepository) Search(ctx context.Context, query string, limit int) ([]models.NodeInfo, error) {
	var nodes []models.NodeInfo
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("callsign LIKE ? OR description LIKE ? OR location LIKE ?", searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Find(&nodes).Error
	return nodes, err
}

// Upsert inserts or updates a node record (uses ON CONFLICT for efficiency)
func (r *NodeInfoRepository) Upsert(ctx context.Context, node *models.NodeInfo) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "node_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"callsign",
			"description",
			"location",
			"last_seen",
			"updated_at",
		}),
	}).Create(node).Error
}

// BulkUpsert efficiently upserts multiple nodes in a single transaction
func (r *NodeInfoRepository) BulkUpsert(ctx context.Context, nodes []models.NodeInfo, batchSize int) error {
	if len(nodes) == 0 {
		return nil
	}

	// Process in batches to avoid overwhelming the database
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(nodes); i += batchSize {
			end := i + batchSize
			if end > len(nodes) {
				end = len(nodes)
			}
			batch := nodes[i:end]

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "node_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"callsign",
					"description",
					"location",
					"last_seen",
					"updated_at",
				}),
			}).Create(&batch).Error; err != nil {
				return fmt.Errorf("batch upsert failed: %w", err)
			}
		}
		return nil
	})
}

// DeleteStaleNodes removes nodes that haven't been seen since the specified timestamp
// This is useful for cleaning up nodes that are no longer in the astdb
func (r *NodeInfoRepository) DeleteStaleNodes(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("last_seen < ?", olderThan).Delete(&models.NodeInfo{})
	return result.RowsAffected, result.Error
}

// GetCount returns the total number of nodes in the database
func (r *NodeInfoRepository) GetCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeInfo{}).Count(&count).Error
	return count, err
}

// GetStaleCount returns the count of nodes that haven't been seen since the specified timestamp
func (r *NodeInfoRepository) GetStaleCount(ctx context.Context, olderThan time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeInfo{}).Where("last_seen < ?", olderThan).Count(&count).Error
	return count, err
}

// GetRecentlyUpdated returns nodes updated within the specified duration
func (r *NodeInfoRepository) GetRecentlyUpdated(ctx context.Context, since time.Duration) ([]models.NodeInfo, error) {
	var nodes []models.NodeInfo
	cutoff := time.Now().Add(-since)
	err := r.db.WithContext(ctx).Where("updated_at > ?", cutoff).Find(&nodes).Error
	return nodes, err
}

// GetAll returns all nodes (use with caution on large datasets)
func (r *NodeInfoRepository) GetAll(ctx context.Context, limit int) ([]models.NodeInfo, error) {
	var nodes []models.NodeInfo
	query := r.db.WithContext(ctx)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&nodes).Error
	return nodes, err
}

// DeleteAll removes all nodes (useful for complete refresh)
func (r *NodeInfoRepository) DeleteAll(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM node_info").Error
}
