package admin

import (
	"context"
	"fmt"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
)

// NodeGroupManager handles node group CRUD operations
type NodeGroupManager struct {
	db *gorm.DB
}

// NewNodeGroupManager creates a new node group manager
func NewNodeGroupManager(db *gorm.DB) *NodeGroupManager {
	return &NodeGroupManager{db: db}
}

// CreateGroup creates a new node group
func (m *NodeGroupManager) CreateGroup(ctx context.Context, group *models.NodeGroup) error {
	if group.Name == "" {
		return fmt.Errorf("group name is required")
	}
	if len(group.NodeNumbers) == 0 {
		return fmt.Errorf("at least one node must be specified")
	}
	if group.CreatedBy == "" {
		return fmt.Errorf("creator email is required")
	}

	// Check for duplicate name
	var existing models.NodeGroup
	if err := m.db.WithContext(ctx).Where("name = ?", group.Name).First(&existing).Error; err == nil {
		return fmt.Errorf("group with name '%s' already exists", group.Name)
	}

	return m.db.WithContext(ctx).Create(group).Error
}

// GetGroup retrieves a group by ID
func (m *NodeGroupManager) GetGroup(ctx context.Context, id uint) (*models.NodeGroup, error) {
	var group models.NodeGroup
	if err := m.db.WithContext(ctx).First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// ListGroups returns all node groups
func (m *NodeGroupManager) ListGroups(ctx context.Context) ([]models.NodeGroup, error) {
	var groups []models.NodeGroup
	if err := m.db.WithContext(ctx).Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// UpdateGroup updates an existing group
func (m *NodeGroupManager) UpdateGroup(ctx context.Context, id uint, updates *models.NodeGroup) error {
	var group models.NodeGroup
	if err := m.db.WithContext(ctx).First(&group, id).Error; err != nil {
		return err
	}

	// Update fields
	if updates.Name != "" {
		// Check for duplicate name (excluding current group)
		var existing models.NodeGroup
		if err := m.db.WithContext(ctx).Where("name = ? AND id != ?", updates.Name, id).First(&existing).Error; err == nil {
			return fmt.Errorf("group with name '%s' already exists", updates.Name)
		}
		group.Name = updates.Name
	}
	if updates.Description != "" {
		group.Description = updates.Description
	}
	if len(updates.NodeNumbers) > 0 {
		group.NodeNumbers = updates.NodeNumbers
	}

	return m.db.WithContext(ctx).Save(&group).Error
}

// DeleteGroup deletes a group
func (m *NodeGroupManager) DeleteGroup(ctx context.Context, id uint) error {
	result := m.db.WithContext(ctx).Delete(&models.NodeGroup{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("group not found")
	}
	return nil
}

// CreateGroupConfig saves a link configuration for a group
func (m *NodeGroupManager) CreateGroupConfig(ctx context.Context, config *models.NodeGroupConfig) error {
	if config.GroupID == 0 {
		return fmt.Errorf("group ID is required")
	}
	if config.Name == "" {
		return fmt.Errorf("config name is required")
	}

	// Verify group exists
	var group models.NodeGroup
	if err := m.db.WithContext(ctx).First(&group, config.GroupID).Error; err != nil {
		return fmt.Errorf("group not found")
	}

	return m.db.WithContext(ctx).Create(config).Error
}

// ListGroupConfigs returns all configurations for a group
func (m *NodeGroupManager) ListGroupConfigs(ctx context.Context, groupID uint) ([]models.NodeGroupConfig, error) {
	var configs []models.NodeGroupConfig
	if err := m.db.WithContext(ctx).Where("group_id = ?", groupID).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// DeleteGroupConfig deletes a saved configuration
func (m *NodeGroupManager) DeleteGroupConfig(ctx context.Context, configID uint) error {
	result := m.db.WithContext(ctx).Delete(&models.NodeGroupConfig{}, configID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("configuration not found")
	}
	return nil
}
