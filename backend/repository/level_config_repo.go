package repository

import (
	"context"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LevelConfigRepo struct {
	db *gorm.DB
}

func NewLevelConfigRepo(db *gorm.DB) *LevelConfigRepo {
	return &LevelConfigRepo{db: db}
}

// GetAll loads all level configs (should be cached in memory by caller)
func (r *LevelConfigRepo) GetAll(ctx context.Context) ([]models.LevelConfig, error) {
	var configs []models.LevelConfig
	err := r.db.WithContext(ctx).Order("level ASC").Find(&configs).Error
	return configs, err
}

// GetAllAsMap returns level configs as a map[level]requiredXP for easy lookup
func (r *LevelConfigRepo) GetAllAsMap(ctx context.Context) (map[int]int, error) {
	configs, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[int]int)
	for _, cfg := range configs {
		result[cfg.Level] = cfg.RequiredExperience
	}
	return result, nil
}

// SeedDefaults inserts default level scaling on first run
// Uses ON CONFLICT DO NOTHING to avoid duplicates
func (r *LevelConfigRepo) SeedDefaults(ctx context.Context, levelRequirements map[int]int) error {
	var configs []models.LevelConfig

	for level := 1; level <= 60; level++ {
		if xp, ok := levelRequirements[level]; ok {
			configs = append(configs, models.LevelConfig{
				Level:              level,
				RequiredExperience: xp,
				Name:               "", // Future: add level names
			})
		}
	}

	// Use ON CONFLICT DO NOTHING to avoid duplicates on restart
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&configs).Error
}
