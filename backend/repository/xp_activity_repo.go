package repository

import (
	"context"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
)

type XPActivityRepo struct {
	db *gorm.DB
}

func NewXPActivityRepo(db *gorm.DB) *XPActivityRepo {
	return &XPActivityRepo{db: db}
}

// LogActivity records XP award with all multipliers for transparency
func (r *XPActivityRepo) LogActivity(
	ctx context.Context,
	callsign string,
	rawXP int,
	awardedXP int,
	restedMultiplier float64,
	drMultiplier float64,
	kerchunkPenalty float64,
) error {
	log := models.XPActivityLog{
		Callsign: callsign,
		// Normalize to UTC to avoid timezone edge cases when querying daily/weekly XP
		HourBucket:       time.Now().UTC().Truncate(time.Hour),
		RawXP:            rawXP,
		AwardedXP:        awardedXP,
		RestedMultiplier: restedMultiplier,
		DRMultiplier:     drMultiplier,
		KerchunkPenalty:  kerchunkPenalty,
	}
	return r.db.WithContext(ctx).Create(&log).Error
}

// GetWeeklyXP returns total awarded XP for a callsign in current week
func (r *XPActivityRepo) GetWeeklyXP(ctx context.Context, callsign string) (int, error) {
	startOfWeek := getStartOfWeek()
	var totalXP int64
	err := r.db.WithContext(ctx).
		Model(&models.XPActivityLog{}).
		Where("callsign = ? AND hour_bucket >= ?", callsign, startOfWeek).
		Select("COALESCE(SUM(awarded_xp), 0)").
		Scan(&totalXP).Error
	return int(totalXP), err
}

// GetDailyXP returns total awarded XP for a callsign today
func (r *XPActivityRepo) GetDailyXP(ctx context.Context, callsign string) (int, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	var totalXP int64
	err := r.db.WithContext(ctx).
		Model(&models.XPActivityLog{}).
		Where("callsign = ? AND hour_bucket >= ?", callsign, startOfDay).
		Select("COALESCE(SUM(awarded_xp), 0)").
		Scan(&totalXP).Error
	return int(totalXP), err
}

// GetLast24Hours returns all activity logs for a callsign in last 24 hours
// Used for calculating diminishing returns
func (r *XPActivityRepo) GetLast24Hours(ctx context.Context, callsign string) ([]models.XPActivityLog, error) {
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	var logs []models.XPActivityLog
	err := r.db.WithContext(ctx).
		Where("callsign = ? AND created_at >= ?", callsign, cutoff).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// GetDailyBreakdown returns activity breakdown for last N days
func (r *XPActivityRepo) GetDailyBreakdown(ctx context.Context, callsign string, days int) ([]DailyActivity, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	type Result struct {
		Date            string
		RawXP           int
		AwardedXP       int
		AvgRestedMult   float64
		AvgDRMult       float64
		AvgKerchunkMult float64
	}

	var results []Result
	err := r.db.WithContext(ctx).
		Model(&models.XPActivityLog{}).
		Select(`
			DATE(hour_bucket) as date,
			SUM(raw_xp) as raw_xp,
			SUM(awarded_xp) as awarded_xp,
			AVG(rested_multiplier) as avg_rested_mult,
			AVG(dr_multiplier) as avg_dr_mult,
			AVG(kerchunk_penalty) as avg_kerchunk_mult
		`).
		Where("callsign = ? AND hour_bucket >= ?", callsign, cutoff).
		Group("DATE(hour_bucket)").
		Order("date DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	var breakdown []DailyActivity
	for _, r := range results {
		breakdown = append(breakdown, DailyActivity{
			Date:      r.Date,
			RawXP:     r.RawXP,
			AwardedXP: r.AwardedXP,
			Multipliers: Multipliers{
				Rested:             r.AvgRestedMult,
				DiminishingReturns: r.AvgDRMult,
				KerchunkPenalty:    r.AvgKerchunkMult,
			},
		})
	}

	return breakdown, nil
}

// Helper types for API responses
type DailyActivity struct {
	Date        string      `json:"date"`
	RawXP       int         `json:"raw_xp"`
	AwardedXP   int         `json:"awarded_xp"`
	Multipliers Multipliers `json:"multipliers"`
}

type Multipliers struct {
	Rested             float64 `json:"rested"`
	DiminishingReturns float64 `json:"diminishing_returns"`
	KerchunkPenalty    float64 `json:"kerchunk_penalty"`
}

// getStartOfWeek returns the start of the current week (Sunday 00:00 UTC)
func getStartOfWeek() time.Time {
	now := time.Now().UTC()
	weekday := int(now.Weekday())
	// Go's Sunday = 0, so we want to go back 'weekday' days
	daysBack := weekday
	startOfWeek := now.AddDate(0, 0, -daysBack).Truncate(24 * time.Hour)
	return startOfWeek
}
