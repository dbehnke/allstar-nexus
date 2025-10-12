package repository

import (
	"context"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
)

type TallyStateRepo struct {
	db *gorm.DB
}

func NewTallyStateRepo(db *gorm.DB) *TallyStateRepo {
	return &TallyStateRepo{db: db}
}

// GetOrInit loads the singleton state row, creating it if missing.
func (r *TallyStateRepo) GetOrInit(ctx context.Context) (*models.TallyState, error) {
	var state models.TallyState
	err := r.db.WithContext(ctx).First(&state, 1).Error
	if err == gorm.ErrRecordNotFound {
		state = models.TallyState{ID: 1, LastTallyAt: time.Time{}}
		if err := r.db.WithContext(ctx).Create(&state).Error; err != nil {
			return nil, err
		}
		return &state, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// UpdateLastTally sets the global last tally timestamp.
func (r *TallyStateRepo) UpdateLastTally(ctx context.Context, t time.Time) error {
	return r.db.WithContext(ctx).Model(&models.TallyState{}).Where("id = ?", 1).Update("last_tally_at", t).Error
}
