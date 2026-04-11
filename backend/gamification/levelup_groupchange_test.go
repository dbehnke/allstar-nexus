package gamification

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite" // SQLite driver
)

// Test that group change notifications are triggered when crossing level boundaries
func TestGroupChangeNotifications(t *testing.T) {
	// Setup test database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_groupchange.db")
	gdb, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        dbPath,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// Auto-migrate
	if err := gdb.AutoMigrate(
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.TransmissionLog{},
		&models.XPActivityLog{},
		&models.TallyState{},
	); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	// Initialize repositories
	profileRepo := repository.NewCallsignProfileRepo(gdb)
	levelConfigRepo := repository.NewLevelConfigRepo(gdb)
	_ = repository.NewTransmissionLogRepository(gdb) // Not used but needed for setup
	_ = repository.NewXPActivityRepo(gdb)            // Not used but needed for setup
	_ = repository.NewTallyStateRepo(gdb)            // Not used but needed for setup

	// Seed level config
	levelReqs := CalculateLevelRequirements()
	if err := levelConfigRepo.SeedDefaults(context.Background(), levelReqs); err != nil {
		t.Fatalf("failed to seed level config: %v", err)
	}

	// Use default level groupings
	levelGroupings := DefaultLevelGroupings()

	// Verify default groupings cover the levels we're testing
	t.Logf("Default level groupings:")
	for _, g := range levelGroupings {
		t.Logf("  %s: %s (%s)", g.Levels, g.Title, g.Badge)
	}

	// We can't easily mock the DiscordNotifier since it's a concrete type,
	// so we'll just check the event data directly instead of testing notifications
	logger := zap.NewNop()

	// Test case 1: Level 19 → 20 (Technician → General)
	t.Run("Level 19 to 20 crosses group boundary", func(t *testing.T) {
		// Create profile at level 19 with enough XP to reach level 20
		ctx := context.Background()
		profile := &models.CallsignProfile{
			Callsign:           "K8TEST",
			Level:              19,
			ExperiencePoints:   levelReqs[20], // Exactly enough for level 20
			RenownLevel:        0,
			LastTransmissionAt: time.Now(),
			LastTallyAt:        time.Now(),
		}

		if err := profileRepo.Upsert(ctx, profile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		// Manually call the level-up logic to test it
		leveledUp, events := processLevelUpsForTest(profile, levelReqs, levelGroupings, true, 36000, logger)

		if !leveledUp {
			t.Fatal("expected level-up to occur")
		}

		if profile.Level != 20 {
			t.Errorf("expected level 20, got %d", profile.Level)
		}

		// Check events
		if len(events) != 1 {
			t.Fatalf("expected 1 level-up event, got %d", len(events))
		}

		event := events[0]
		t.Logf("Event: PreviousLevel=%d, NewLevel=%d", event.PreviousLevel, event.NewLevel)
		t.Logf("Event: PreviousGroup=%+v, NewGroup=%+v", event.PreviousGroup, event.NewGroup)

		// Verify previous group is Technician (10-19)
		if event.PreviousGroup == nil {
			t.Error("PreviousGroup should not be nil for level 19")
		} else if event.PreviousGroup.Title != "Technician" {
			t.Errorf("expected PreviousGroup='Technician', got '%s'", event.PreviousGroup.Title)
		}

		// Verify new group is General (20-29)
		if event.NewGroup == nil {
			t.Error("NewGroup should not be nil for level 20")
		} else if event.NewGroup.Title != "General" {
			t.Errorf("expected NewGroup='General', got '%s'", event.NewGroup.Title)
		}

		// Verify the condition that would trigger group change notification
		shouldNotify := event.NewGroup != nil && event.PreviousGroup != nil &&
			event.NewGroup.Title != event.PreviousGroup.Title

		if !shouldNotify {
			t.Error("group change notification should be triggered for Technician → General transition")
		}
	})

	// Test case 2: Level 29 → 30 (General → Advanced)
	t.Run("Level 29 to 30 crosses group boundary", func(t *testing.T) {
		ctx := context.Background()
		profile := &models.CallsignProfile{
			Callsign:           "K8TEST2",
			Level:              29,
			ExperiencePoints:   levelReqs[30],
			RenownLevel:        0,
			LastTransmissionAt: time.Now(),
			LastTallyAt:        time.Now(),
		}

		if err := profileRepo.Upsert(ctx, profile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		leveledUp, events := processLevelUpsForTest(profile, levelReqs, levelGroupings, true, 36000, logger)

		if !leveledUp {
			t.Fatal("expected level-up to occur")
		}

		if len(events) != 1 {
			t.Fatalf("expected 1 level-up event, got %d", len(events))
		}

		event := events[0]

		// Verify group transition
		if event.PreviousGroup == nil || event.PreviousGroup.Title != "General" {
			t.Errorf("expected PreviousGroup='General', got %+v", event.PreviousGroup)
		}
		if event.NewGroup == nil || event.NewGroup.Title != "Advanced" {
			t.Errorf("expected NewGroup='Advanced', got %+v", event.NewGroup)
		}

		// Verify notification condition
		shouldNotify := event.NewGroup != nil && event.PreviousGroup != nil &&
			event.NewGroup.Title != event.PreviousGroup.Title

		if !shouldNotify {
			t.Error("group change notification should be triggered for General → Advanced transition")
		}
	})

	// Test case 3: Level 20 → 21 (no group change)
	t.Run("Level 20 to 21 does not cross group boundary", func(t *testing.T) {
		ctx := context.Background()
		profile := &models.CallsignProfile{
			Callsign:           "K8TEST3",
			Level:              20,
			ExperiencePoints:   levelReqs[21],
			RenownLevel:        0,
			LastTransmissionAt: time.Now(),
			LastTallyAt:        time.Now(),
		}

		if err := profileRepo.Upsert(ctx, profile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		leveledUp, events := processLevelUpsForTest(profile, levelReqs, levelGroupings, true, 36000, logger)

		if !leveledUp {
			t.Fatal("expected level-up to occur")
		}

		if len(events) != 1 {
			t.Fatalf("expected 1 level-up event, got %d", len(events))
		}

		event := events[0]

		// Both should be General
		if event.PreviousGroup == nil || event.PreviousGroup.Title != "General" {
			t.Errorf("expected PreviousGroup='General', got %+v", event.PreviousGroup)
		}
		if event.NewGroup == nil || event.NewGroup.Title != "General" {
			t.Errorf("expected NewGroup='General', got %+v", event.NewGroup)
		}

		// Verify notification should NOT be triggered (same group)
		shouldNotify := event.NewGroup != nil && event.PreviousGroup != nil &&
			event.NewGroup.Title != event.PreviousGroup.Title

		if shouldNotify {
			t.Error("group change notification should NOT be triggered when staying in same group")
		}
	})
}

// processLevelUpsForTest is a standalone version of the processLevelUps logic for testing
// It replicates the logic from tally_service.go without needing a full TallyService instance
func processLevelUpsForTest(profile *models.CallsignProfile, levelRequirements map[int]int, levelGroupings []cfgpkg.LevelGrouping, renownEnabled bool, renownXPPerLevel int, _ *zap.Logger) (bool, []LevelUpInfo) {
	leveledUp := false
	var levelUpEvents []LevelUpInfo

	// Track the original level and grouping before processing
	originalLevel := profile.Level
	originalGroup := GetGroupingForLevel(originalLevel, levelGroupings)

	// Loop to handle multiple level-ups at once
	for {
		nextLevel := profile.Level + 1

		var requiredXP int
		var ok bool

		if nextLevel <= 60 {
			requiredXP, ok = levelRequirements[nextLevel]
		} else {
			// Renown levels beyond 60 use fixed XP-per-level when enabled
			if renownEnabled && renownXPPerLevel > 0 {
				requiredXP = renownXPPerLevel
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
		previousLevel := profile.Level
		profile.ExperiencePoints -= requiredXP
		profile.Level++
		leveledUp = true

		// If we've reached level 60 (i.e., reached renown threshold), award renown
		if profile.Level >= 60 {
			profile.RenownLevel++
			profile.Level = 1

			// Record renown event
			levelUpEvents = append(levelUpEvents, LevelUpInfo{
				Callsign:      profile.Callsign,
				PreviousLevel: previousLevel,
				NewLevel:      profile.Level,
				PreviousGroup: originalGroup,
				NewGroup:      nil, // Renown resets level to 1, so group transitions not applicable here
				RenownGained:  true,
				RenownLevel:   profile.RenownLevel,
			})

			break // Stop after renown to avoid infinite loop
		}

		// Record regular level-up event
		newGroup := GetGroupingForLevel(profile.Level, levelGroupings)

		levelUpEvents = append(levelUpEvents, LevelUpInfo{
			Callsign:      profile.Callsign,
			PreviousLevel: previousLevel,
			NewLevel:      profile.Level,
			PreviousGroup: originalGroup,
			NewGroup:      newGroup,
			RenownGained:  false,
		})

		// Update originalGroup for next iteration
		originalGroup = newGroup
	}

	return leveledUp, levelUpEvents
}
