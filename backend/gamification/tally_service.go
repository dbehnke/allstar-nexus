package gamification

import (
	"context"
	"log"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config holds anti-cheating and XP configuration
type Config struct {
	// Rested Bonus
	RestedEnabled              bool
	RestedAccumulationRate     float64 // hours bonus per hour idle (1.5 = generous)
	RestedMaxSeconds           int     // max rested bonus cap (336 hours = 14 days)
	RestedMultiplier           float64 // XP multiplier when rested (2.0 = double XP)
	RestedIdleThresholdSeconds int     // begin accruing after this many seconds idle (default 5 minutes)

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

	// Renown (prestige)
	RenownEnabled    bool
	RenownXPPerLevel int // fixed XP required per renown-level (applies after level 60)
}

type DRTier struct {
	MaxSeconds int     // upper bound for this tier
	Multiplier float64 // XP multiplier for this tier
}

// TallyService processes XP from transmission logs periodically
type TallyService struct {
	db                *gorm.DB
	txLogRepo         *repository.TransmissionLogRepository
	profileRepo       *repository.CallsignProfileRepo
	levelConfigRepo   *repository.LevelConfigRepo
	activityRepo      *repository.XPActivityRepo
	stateRepo         *repository.TallyStateRepo
	config            *Config
	levelRequirements map[int]int // level -> xp_required
	tallyInterval     time.Duration
	ticker            *time.Ticker
	stopChan          chan struct{}
	lastTallyTime     time.Time
	logger            *zap.Logger
	// Optional hook invoked after each tally completes
	OnTallyComplete func(summary TallySummary)
}

// TallySummary contains basic metrics about a completed tally run
type TallySummary struct {
	CallsignsProcessed   int       `json:"callsigns_processed"`
	TransmissionsHandled int       `json:"transmissions_handled"`
	StartedAt            time.Time `json:"started_at"`
	CompletedAt          time.Time `json:"completed_at"`
}

func NewTallyService(
	db *gorm.DB,
	txRepo *repository.TransmissionLogRepository,
	profileRepo *repository.CallsignProfileRepo,
	levelRepo *repository.LevelConfigRepo,
	activityRepo *repository.XPActivityRepo,
	stateRepo *repository.TallyStateRepo,
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
		stateRepo:       stateRepo,
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

	// Load persisted last tally time (if any)
	if s.stateRepo != nil {
		if state, err := s.stateRepo.GetOrInit(context.Background()); err == nil {
			if !state.LastTallyAt.IsZero() {
				s.lastTallyTime = state.LastTallyAt
			} else {
				// No persisted time — seed from oldest transmission log if available
				if oldest, err := s.txLogRepo.GetOldestLogTime(); err == nil && !oldest.IsZero() {
					s.lastTallyTime = oldest
					s.logger.Info("Initialized last tally time from oldest log", zap.Time("oldest", oldest))
				}
			}
		} else {
			s.logger.Warn("failed to load tally state; using default window", zap.Error(err))
		}
	}
	// Test-trace: emit at error-level so it shows in tests
	s.logger.Error("tally.start.seed", zap.Time("lastTallyTime", s.lastTallyTime))

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

	now := time.Now().UTC()
	// Test-trace
	s.logger.Error("tally.process.begin", zap.Time("from", s.lastTallyTime), zap.Time("to", now))
	s.logger.Info("Processing XP tally (windowed)", zap.Time("from", s.lastTallyTime), zap.Time("to", now), zap.Duration("step", s.tallyInterval))

	processed := make(map[string]struct{})
	summary := TallySummary{
		CallsignsProcessed:   0,
		TransmissionsHandled: 0,
		StartedAt:            time.Now(),
	}

	// Helper: process grouped logs for a window
	processGroup := func(transmissions map[string][]models.TransmissionLog) {
		for callsign, txLogs := range transmissions {
			if callsign == "" {
				continue
			}
			processed[callsign] = struct{}{}

			// Load or create profile
			profile, err := s.profileRepo.GetByCallsign(ctx, callsign)
			if err != nil {
				s.logger.Error("Failed to get profile", zap.String("callsign", callsign), zap.Error(err))
				continue
			}

			// Update rested bonus accumulation
			s.updateRestedBonus(profile)

			// Caps context
			weeklyXP, dailyXP := 0, 0
			if s.config.CapsEnabled {
				weeklyXP, _ = s.activityRepo.GetWeeklyXP(ctx, callsign)
				dailyXP, _ = s.activityRepo.GetDailyXP(ctx, callsign)
			}

			// DR context
			currentDailySeconds := 0
			if s.config.DREnabled {
				recentActivity, _ := s.activityRepo.GetLast24Hours(ctx, callsign)
				for _, activity := range recentActivity {
					currentDailySeconds += activity.RawXP
				}
			}

			// Process each transmission in order
			for i, tx := range txLogs {
				rawXP := tx.DurationSeconds
				summary.TransmissionsHandled++

				if s.config.CapsEnabled && weeklyXP >= s.config.WeeklyCapSeconds {
					_ = s.activityRepo.LogActivity(ctx, callsign, rawXP, 0, 0, 0, 0)
					continue
				}
				if s.config.CapsEnabled && dailyXP >= s.config.DailyCapSeconds {
					_ = s.activityRepo.LogActivity(ctx, callsign, rawXP, 0, 0, 0, 0)
					continue
				}

				restedMultiplier := s.applyRestedBonus(profile, tx.DurationSeconds)
				drMultiplier := s.calculateDRMultiplier(currentDailySeconds)
				currentDailySeconds += tx.DurationSeconds
				kerchunkPenalty := s.calculateKerchunkPenalty(txLogs[:i], tx)

				finalXP := float64(rawXP) * restedMultiplier * drMultiplier * kerchunkPenalty
				awardedXP := int(finalXP)

				if s.config.CapsEnabled {
					remainingDaily := s.config.DailyCapSeconds - dailyXP
					if awardedXP > remainingDaily {
						awardedXP = remainingDaily
					}
					remainingWeekly := s.config.WeeklyCapSeconds - weeklyXP
					if awardedXP > remainingWeekly {
						awardedXP = remainingWeekly
					}
				}

				_ = s.activityRepo.LogActivity(ctx, callsign, rawXP, awardedXP, restedMultiplier, drMultiplier, kerchunkPenalty)
				profile.ExperiencePoints += awardedXP
				profile.DailyXP += awardedXP
				profile.WeeklyXP += awardedXP
				weeklyXP += awardedXP
				dailyXP += awardedXP
			}

			if len(txLogs) > 0 {
				profile.LastTransmissionAt = txLogs[len(txLogs)-1].TimestampEnd
				profile.LastTallyAt = time.Now().UTC()
			}

			leveledUp := s.processLevelUps(profile)
			if leveledUp {
				s.logger.Info("Level up!", zap.String("callsign", profile.Callsign), zap.Int("level", profile.Level), zap.Int("renown", profile.RenownLevel))
			}

			if err := s.profileRepo.Upsert(ctx, profile); err != nil {
				s.logger.Error("Failed to save profile", zap.String("callsign", callsign), zap.Error(err))
			}
		}
	}

	// Iterate windows from lastTallyTime to now
	originalStart := s.lastTallyTime
	cursor := s.lastTallyTime
	if cursor.IsZero() || cursor.After(now) {
		cursor = now.Add(-s.tallyInterval)
	}
	for cursor.Before(now) {
		next := cursor.Add(s.tallyInterval)
		if next.After(now) {
			next = now
		}
		s.logger.Info("Tally window", zap.Time("from", cursor), zap.Time("to", next))
		// Test-trace
		s.logger.Error("tally.process.window", zap.Time("from", cursor), zap.Time("to", next))
		transmissions, err := s.txLogRepo.GetLogsBetween(cursor, next)
		if err != nil {
			return err
		}
		if len(transmissions) > 0 {
			// Test-trace: log counts
			txCount := 0
			for _, arr := range transmissions {
				txCount += len(arr)
			}
			s.logger.Error("tally.window.results", zap.Int("callsigns", len(transmissions)), zap.Int("tx_count", txCount))
			processGroup(transmissions)
		}
		// Persist window completion
		s.lastTallyTime = next
		if s.stateRepo != nil {
			if err := s.stateRepo.UpdateLastTally(ctx, s.lastTallyTime); err != nil {
				s.logger.Warn("failed to persist last tally time", zap.Error(err))
			}
		}
		cursor = next
	}

	summary.CallsignsProcessed = len(processed)
	summary.CompletedAt = s.lastTallyTime

	// Fallback: if no transmissions were handled (e.g., due to DB datetime format edge cases),
	// run a single-batch tally using the legacy GetLogsSince path to preserve compatibility.
	if summary.TransmissionsHandled == 0 {
		s.logger.Info("Windowed tally handled 0 transmissions; falling back to single-batch GetLogsSince",
			zap.Time("since", originalStart))
		transmissions, err := s.txLogRepo.GetLogsSince(originalStart)
		if err != nil {
			return err
		}
		if len(transmissions) > 0 {
			processGroup(transmissions)
			s.lastTallyTime = now
			summary.CompletedAt = s.lastTallyTime
			if s.stateRepo != nil {
				if err := s.stateRepo.UpdateLastTally(ctx, s.lastTallyTime); err != nil {
					s.logger.Warn("failed to persist last tally time (fallback)", zap.Error(err))
				}
			}
			summary.CallsignsProcessed = len(processed)
		}
	}

	if s.OnTallyComplete != nil {
		go func(cb func(TallySummary), sum TallySummary) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in OnTallyComplete callback: %v", r)
				}
			}()
			cb(sum)
		}(s.OnTallyComplete, summary)
	}
	return nil
}

// updateRestedBonus accumulates rested bonus for inactive callsigns
func (s *TallyService) updateRestedBonus(profile *models.CallsignProfile) {
	if !s.config.RestedEnabled {
		return
	}
	
	// Determine idle time and apply threshold before accruing
	idleSince := time.Since(profile.LastTransmissionAt)
	threshold := time.Duration(s.config.RestedIdleThresholdSeconds) * time.Second
	if threshold <= 0 {
		// Default threshold to 5 minutes if not provided
		threshold = 5 * time.Minute
	}
	
	if idleSince < threshold {
		return
	}
	
	// Initialize LastRestedCalculationAt if this is the first calculation
	if profile.LastRestedCalculationAt.IsZero() {
		profile.LastRestedCalculationAt = profile.LastTransmissionAt
	}
	
	// Only accumulate for NEW idle time since last calculation
	timeSinceLastCalculation := time.Since(profile.LastRestedCalculationAt)
	if timeSinceLastCalculation <= 0 {
		return
	}
	
	// Accumulate rested bonus: per-hour idle scaled by accumulation rate
	hoursIdle := timeSinceLastCalculation.Hours()
	bonusHours := hoursIdle * s.config.RestedAccumulationRate
	profile.RestedBonusSeconds += int(bonusHours * 3600)

	// Cap at maximum
	if profile.RestedBonusSeconds > s.config.RestedMaxSeconds {
		profile.RestedBonusSeconds = s.config.RestedMaxSeconds
	}
	
	// Update the last calculation timestamp
	profile.LastRestedCalculationAt = time.Now()
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

		var requiredXP int
		var ok bool

		if nextLevel <= 60 {
			requiredXP, ok = s.levelRequirements[nextLevel]
		} else {
			// Renown levels beyond 60 use fixed XP-per-level when enabled
			if s.config.RenownEnabled && s.config.RenownXPPerLevel > 0 {
				requiredXP = s.config.RenownXPPerLevel
				ok = true
			} else {
				// No renown configured; don't allow leveling beyond 60
				ok = false
			}
		}

		if !ok || profile.ExperiencePoints < requiredXP {
			break // Not enough XP for next level
		}

		// Level up!
		profile.ExperiencePoints -= requiredXP
		profile.Level++
		leveledUp = true

		// If we've reached level 60 (i.e., reached renown threshold), award renown
		if profile.Level >= 60 {
			profile.RenownLevel++
			profile.Level = 1
			// Preserve carryover XP for the new renown cycle (leftover after subtracting requiredXP)
			// Example: If profile.ExperiencePoints = 2500 and requiredXP = 2000,
			// after leveling up, profile.ExperiencePoints = 2500 - 2000 = 500.
			// When renown is gained, the leftover XP (500) is preserved for the next renown cycle.
			// profile.ExperiencePoints already contains the leftover at this point.
			s.logger.Info("Renown gained!",
				zap.String("callsign", profile.Callsign),
				zap.Int("renown", profile.RenownLevel),
				zap.Int("carryover_xp", profile.ExperiencePoints),
			)
			break // Stop after renown to avoid infinite loop
		}
	}

	return leveledUp
}
