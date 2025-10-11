package gamification

import (
	"context"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config holds anti-cheating and XP configuration
type Config struct {
	// Rested Bonus
	RestedEnabled          bool
	RestedAccumulationRate float64 // hours bonus per hour offline (1.5 = generous)
	RestedMaxSeconds       int     // max rested bonus cap (336 hours = 14 days)
	RestedMultiplier       float64 // XP multiplier when rested (2.0 = double XP)

	// Diminishing Returns
	DREnabled bool
	DRTiers   []DRTier

	// Kerchunk Detection
	KerchunkEnabled       bool
	KerchunkThreshold     int // seconds (transmissions < this are kerchunks)
	KerchunkWindow        int // seconds (check consecutive kerchunks in this window)
	KerchunkSinglePenalty float64
	Kerchunk2to3Penalty   float64
	Kerchunk4to5Penalty   float64
	Kerchunk6PlusPenalty  float64

	// XP Caps
	CapsEnabled      bool
	DailyCapSeconds  int // 1,200 = 20 minutes/day for low-activity hub
	WeeklyCapSeconds int // 7,200 = 2 hours/week for low-activity hub
}

type DRTier struct {
	MaxSeconds int     // upper bound for this tier
	Multiplier float64 // XP multiplier for this tier
}

// TallyService processes XP from transmission logs periodically
type TallyService struct {
	db              *gorm.DB
	txLogRepo       *repository.TransmissionLogRepository
	profileRepo     *repository.CallsignProfileRepo
	levelConfigRepo *repository.LevelConfigRepo
	activityRepo    *repository.XPActivityRepo
	config          *Config
	levelRequirements map[int]int // level -> xp_required
	tallyInterval   time.Duration
	ticker          *time.Ticker
	stopChan        chan struct{}
	lastTallyTime   time.Time
	logger          *zap.Logger
}

func NewTallyService(
	db *gorm.DB,
	txRepo *repository.TransmissionLogRepository,
	profileRepo *repository.CallsignProfileRepo,
	levelRepo *repository.LevelConfigRepo,
	activityRepo *repository.XPActivityRepo,
	config *Config,
	interval time.Duration,
	logger *zap.Logger,
) *TallyService {
	return &TallyService{
		db:              db,
		txLogRepo:       txRepo,
		profileRepo:     profileRepo,
		levelConfigRepo: levelRepo,
		activityRepo:    activityRepo,
		config:          config,
		tallyInterval:   interval,
		stopChan:        make(chan struct{}),
		lastTallyTime:   time.Now().Add(-interval), // Process logs from last interval on startup
		logger:          logger,
	}
}

func (s *TallyService) Start() error {
	// Load level requirements
	levelMap, err := s.levelConfigRepo.GetAllAsMap(context.Background())
	if err != nil {
		return err
	}
	s.levelRequirements = levelMap

	s.logger.Info("TallyService starting", zap.Duration("interval", s.tallyInterval))

	// Run initial tally
	if err := s.ProcessTally(); err != nil {
		s.logger.Error("Initial tally failed", zap.Error(err))
	}

	// Start ticker
	s.ticker = time.NewTicker(s.tallyInterval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				if err := s.ProcessTally(); err != nil {
					s.logger.Error("Tally processing failed", zap.Error(err))
				}
			case <-s.stopChan:
				return
			}
		}
	}()

	return nil
}

func (s *TallyService) Stop() {
	s.logger.Info("TallyService stopping")
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
}

func (s *TallyService) ProcessTally() error {
	ctx := context.Background()

	s.logger.Info("Processing XP tally", zap.Time("since", s.lastTallyTime))

	// 1. Get transmissions since last tally, grouped by callsign
	transmissions, err := s.txLogRepo.GetLogsSince(s.lastTallyTime)
	if err != nil {
		return err
	}

	s.logger.Info("Tally batch", zap.Int("callsigns", len(transmissions)))

	for callsign, txLogs := range transmissions {
		if callsign == "" {
			continue // Skip empty callsigns
		}

		// 2. Load or create profile
		profile, err := s.profileRepo.GetByCallsign(ctx, callsign)
		if err != nil {
			s.logger.Error("Failed to get profile", zap.String("callsign", callsign), zap.Error(err))
			continue
		}

		// 3. Update rested bonus accumulation
		s.updateRestedBonus(profile)

		// 4. Get current daily/weekly XP for caps
		weeklyXP, dailyXP := 0, 0
		if s.config.CapsEnabled {
			weeklyXP, _ = s.activityRepo.GetWeeklyXP(ctx, callsign)
			dailyXP, _ = s.activityRepo.GetDailyXP(ctx, callsign)
		}

		// 5. Get recent activity for diminishing returns calculation
		currentDailySeconds := 0
		if s.config.DREnabled {
			recentActivity, _ := s.activityRepo.GetLast24Hours(ctx, callsign)
			for _, activity := range recentActivity {
				currentDailySeconds += activity.RawXP
			}
		}

		// 6. Process each transmission
		for i, tx := range txLogs {
			rawXP := tx.DurationSeconds

			// Skip if at weekly cap
			if s.config.CapsEnabled && weeklyXP >= s.config.WeeklyCapSeconds {
				s.activityRepo.LogActivity(ctx, callsign, rawXP, 0, 0, 0, 0)
				continue
			}

			// Skip if at daily cap
			if s.config.CapsEnabled && dailyXP >= s.config.DailyCapSeconds {
				s.activityRepo.LogActivity(ctx, callsign, rawXP, 0, 0, 0, 0)
				continue
			}

			// Apply rested bonus
			restedMultiplier := s.applyRestedBonus(profile, tx.DurationSeconds)

			// Apply diminishing returns
			drMultiplier := s.calculateDRMultiplier(currentDailySeconds)
			currentDailySeconds += tx.DurationSeconds

			// Apply kerchunk penalty
			kerchunkPenalty := s.calculateKerchunkPenalty(txLogs[:i], tx)

			// Calculate final XP
			finalXP := float64(rawXP) * restedMultiplier * drMultiplier * kerchunkPenalty
			awardedXP := int(finalXP)

			// Apply daily cap to awarded amount
			if s.config.CapsEnabled {
				remainingDaily := s.config.DailyCapSeconds - dailyXP
				if awardedXP > remainingDaily {
					awardedXP = remainingDaily
				}

				// Apply weekly cap to awarded amount
				remainingWeekly := s.config.WeeklyCapSeconds - weeklyXP
				if awardedXP > remainingWeekly {
					awardedXP = remainingWeekly
				}
			}

			// Log activity (transparency)
			s.activityRepo.LogActivity(ctx, callsign, rawXP, awardedXP, restedMultiplier, drMultiplier, kerchunkPenalty)

			// Award XP to profile
			profile.ExperiencePoints += awardedXP

			// Update running totals
			weeklyXP += awardedXP
			dailyXP += awardedXP
		}

		// 7. Update last transmission timestamp
		if len(txLogs) > 0 {
			profile.LastTransmissionAt = txLogs[len(txLogs)-1].TimestampEnd
		}

		// 8. Process level-ups
		leveledUp := s.processLevelUps(profile)
		if leveledUp {
			s.logger.Info("Level up!",
				zap.String("callsign", profile.Callsign),
				zap.Int("level", profile.Level),
				zap.Int("renown", profile.RenownLevel),
			)
		}

		// 9. Save updated profile
		if err := s.profileRepo.Upsert(ctx, profile); err != nil {
			s.logger.Error("Failed to save profile", zap.String("callsign", callsign), zap.Error(err))
		}
	}

	s.lastTallyTime = time.Now()
	return nil
}

// updateRestedBonus accumulates rested bonus for inactive callsigns
func (s *TallyService) updateRestedBonus(profile *models.CallsignProfile) {
	if !s.config.RestedEnabled {
		return
	}

	hoursOffline := time.Since(profile.LastTransmissionAt).Hours()
	if hoursOffline >= 24 {
		// Accumulate rested bonus: 1 hour offline = 1.5 hours bonus (generous!)
		bonusHours := hoursOffline * s.config.RestedAccumulationRate
		profile.RestedBonusSeconds += int(bonusHours * 3600)

		// Cap at maximum (14 days worth)
		if profile.RestedBonusSeconds > s.config.RestedMaxSeconds {
			profile.RestedBonusSeconds = s.config.RestedMaxSeconds
		}
	}
}

// applyRestedBonus consumes rested bonus and returns multiplier
func (s *TallyService) applyRestedBonus(profile *models.CallsignProfile, durationSeconds int) float64 {
	if !s.config.RestedEnabled || profile.RestedBonusSeconds == 0 {
		return 1.0
	}

	// Consume rested bonus
	profile.RestedBonusSeconds -= durationSeconds

	// If partial bonus (running out mid-transmission)
	if profile.RestedBonusSeconds < 0 {
		secondsWithBonus := durationSeconds + profile.RestedBonusSeconds
		secondsWithoutBonus := -profile.RestedBonusSeconds
		multiplier := (float64(secondsWithBonus)*s.config.RestedMultiplier + float64(secondsWithoutBonus)) / float64(durationSeconds)
		profile.RestedBonusSeconds = 0
		return multiplier
	}

	return s.config.RestedMultiplier
}

// calculateDRMultiplier returns diminishing returns multiplier based on daily talk time
func (s *TallyService) calculateDRMultiplier(secondsInLast24Hours int) float64 {
	if !s.config.DREnabled {
		return 1.0
	}

	for _, tier := range s.config.DRTiers {
		if secondsInLast24Hours <= tier.MaxSeconds {
			return tier.Multiplier
		}
	}

	// Fallback to last tier
	if len(s.config.DRTiers) > 0 {
		return s.config.DRTiers[len(s.config.DRTiers)-1].Multiplier
	}

	return 1.0
}

// calculateKerchunkPenalty detects spam (rapid short transmissions)
func (s *TallyService) calculateKerchunkPenalty(previousLogs []models.TransmissionLog, currentTX models.TransmissionLog) float64 {
	if !s.config.KerchunkEnabled {
		return 1.0
	}

	// Normal transmission (>=3 seconds) = no penalty
	if currentTX.DurationSeconds >= s.config.KerchunkThreshold {
		return 1.0
	}

	// Count consecutive kerchunks in last 30 seconds
	consecutiveCount := 0
	cutoff := currentTX.TimestampStart.Add(-time.Duration(s.config.KerchunkWindow) * time.Second)

	for i := len(previousLogs) - 1; i >= 0; i-- {
		log := previousLogs[i]
		if log.TimestampStart.Before(cutoff) {
			break // Outside window
		}
		if log.DurationSeconds < s.config.KerchunkThreshold {
			consecutiveCount++
		} else {
			break // Normal TX breaks the chain
		}
	}

	// Apply penalty based on count
	switch consecutiveCount {
	case 0:
		return s.config.KerchunkSinglePenalty // First kerchunk
	case 1, 2:
		return s.config.Kerchunk2to3Penalty // 2-3 consecutive
	case 3, 4:
		return s.config.Kerchunk4to5Penalty // 4-5 consecutive
	default:
		return s.config.Kerchunk6PlusPenalty // 6+ = no XP
	}
}

// processLevelUps handles leveling up (including renown/prestige)
func (s *TallyService) processLevelUps(profile *models.CallsignProfile) bool {
	leveledUp := false

	// Loop to handle multiple level-ups at once
	for {
		nextLevel := profile.Level + 1
		if nextLevel > 60 {
			nextLevel = 61 // Renown reset
		}

		requiredXP, ok := s.levelRequirements[nextLevel]
		if !ok || profile.ExperiencePoints < requiredXP {
			break // Not enough XP for next level
		}

		// Level up!
		profile.ExperiencePoints -= requiredXP
		profile.Level++
		leveledUp = true

		// Check for renown (prestige) at level 60
		if profile.Level >= 60 {
			profile.RenownLevel++
			profile.Level = 1
			profile.ExperiencePoints = 0
			s.logger.Info("Renown gained!",
				zap.String("callsign", profile.Callsign),
				zap.Int("renown", profile.RenownLevel),
			)
			break // Stop after renown to avoid infinite loop
		}
	}

	return leveledUp
}
