package repository

import (
	"context"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CallsignProfileRepo struct {
	db *gorm.DB
}

func NewCallsignProfileRepo(db *gorm.DB) *CallsignProfileRepo {
	return &CallsignProfileRepo{db: db}
}

// GetByCallsign returns profile or creates new one if not exists
func (r *CallsignProfileRepo) GetByCallsign(ctx context.Context, callsign string) (*models.CallsignProfile, error) {
	// Normalize callsign (uppercase, trim whitespace)
	callsign = strings.ToUpper(strings.TrimSpace(callsign))

	var profile models.CallsignProfile
	err := r.db.WithContext(ctx).Where("callsign = ?", callsign).First(&profile).Error

	if err == gorm.ErrRecordNotFound {
		// Create new profile with defaults
		profile = models.CallsignProfile{
			Callsign:           callsign,
			Level:              1,
			ExperiencePoints:   0,
			RenownLevel:        0,
			LastTallyAt:        time.Now(),
			LastTransmissionAt: time.Now(),
			RestedBonusSeconds: 0,
		}
		if err := r.db.WithContext(ctx).Create(&profile).Error; err != nil {
			return nil, err
		}
		return &profile, nil
	}

	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// Upsert creates or updates a profile
func (r *CallsignProfileRepo) Upsert(ctx context.Context, profile *models.CallsignProfile) error {
	// Normalize callsign
	profile.Callsign = strings.ToUpper(strings.TrimSpace(profile.Callsign))

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "callsign"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"level", "experience_points", "renown_level",
			"last_tally_at", "last_transmission_at", "rested_bonus_seconds",
			"updated_at",
		}),
	}).Create(profile).Error
}

// AddExperience increments XP atomically
func (r *CallsignProfileRepo) AddExperience(ctx context.Context, callsign string, xpToAdd int) error {
	callsign = strings.ToUpper(strings.TrimSpace(callsign))

	return r.db.WithContext(ctx).
		Model(&models.CallsignProfile{}).
		Where("callsign = ?", callsign).
		Update("experience_points", gorm.Expr("experience_points + ?", xpToAdd)).
		Error
}

// GetLeaderboard returns top N callsigns ranked by renown, level, and XP
func (r *CallsignProfileRepo) GetLeaderboard(ctx context.Context, limit int) ([]models.CallsignProfile, error) {
	var profiles []models.CallsignProfile
	err := r.db.WithContext(ctx).
		Order("renown_level DESC, level DESC, experience_points DESC").
		Limit(limit).
		Find(&profiles).Error
	return profiles, err
}

// GetProfilesNeedingLevelUp finds profiles where XP >= required for next level
func (r *CallsignProfileRepo) GetProfilesNeedingLevelUp(ctx context.Context, levelConfigs map[int]int) ([]models.CallsignProfile, error) {
	var profiles []models.CallsignProfile
	// This is a simplified query - in practice, the tally service handles level-ups
	// during processing rather than querying for them
	err := r.db.WithContext(ctx).Find(&profiles).Error
	return profiles, err
}

// BulkUpdate updates multiple profiles in a transaction
func (r *CallsignProfileRepo) BulkUpdate(ctx context.Context, profiles []models.CallsignProfile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, profile := range profiles {
			if err := tx.Save(&profile).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAllProfiles returns all callsign profiles (for rested XP accumulation)
func (r *CallsignProfileRepo) GetAllProfiles(ctx context.Context) ([]models.CallsignProfile, error) {
	var profiles []models.CallsignProfile
	err := r.db.WithContext(ctx).Find(&profiles).Error
	return profiles, err
}
