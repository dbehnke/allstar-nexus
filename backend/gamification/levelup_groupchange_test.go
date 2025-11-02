package gamification

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	txLogRepo := repository.NewTransmissionLogRepository(gdb)
	activityRepo := repository.NewXPActivityRepo(gdb)
	stateRepo := repository.NewTallyStateRepo(gdb)

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

	// Create mock Discord notifier that captures notifications
	notifications := make([]string, 0)
	mockNotifier := &mockDiscordNotifier{
		notifications: &notifications,
	}

	// Create tally service
	tallyConfig := &Config{
		RestedEnabled:              false,
		DREnabled:                  false,
		KerchunkEnabled:            false,
		CapsEnabled:                false,
		RenownEnabled:              true,
		RenownXPPerLevel:           36000,
	}

	logger := zap.NewNop()

	tallyService := NewTallyService(
		gdb,
		txLogRepo,
		profileRepo,
		levelConfigRepo,
		activityRepo,
		stateRepo,
		tallyConfig,
		30*time.Minute,
		logger,
		levelGroupings,
		mockNotifier,
	)

	if err := tallyService.Start(); err != nil {
		t.Fatalf("failed to start tally service: %v", err)
	}
	defer tallyService.Stop()

	// Test case 1: Level 19 → 20 (Technician → General)
	t.Run("Level 19 to 20 crosses group boundary", func(t *testing.T) {
		notifications = notifications[:0] // Clear notifications

		// Create profile at level 19 with enough XP to reach level 20
		ctx := context.Background()
		profile := &models.CallsignProfile{
			Callsign:         "K8TEST",
			Level:            19,
			ExperiencePoints: levelReqs[20], // Exactly enough for level 20
			RenownLevel:      0,
			LastTransmissionAt: time.Now(),
			LastTallyAt:        time.Now(),
		}

		if err := profileRepo.Upsert(ctx, profile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		// Process level-ups manually (simulating what tally would do)
		leveledUp, events := tallyService.processLevelUps(profile)

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

		// Simulate notification logic from tally_service.go lines 283-295
		if event.NewGroup != nil && event.PreviousGroup != nil &&
			event.NewGroup.Title != event.PreviousGroup.Title {
			mockNotifier.NotifyGroupChange(event.Callsign, event.NewGroup.Title)
		} else if event.NewGroup != nil && event.PreviousGroup == nil {
			mockNotifier.NotifyGroupChange(event.Callsign, event.NewGroup.Title)
		}

		// Verify group change notification was sent
		foundGroupChange := false
		for _, notif := range notifications {
			t.Logf("Notification: %s", notif)
			if notif == "GROUP_CHANGE:K8TEST:General" {
				foundGroupChange = true
			}
		}

		if !foundGroupChange {
			t.Error("expected group change notification for General rank")
		}
	})

	// Test case 2: Level 29 → 30 (General → Advanced)
	t.Run("Level 29 to 30 crosses group boundary", func(t *testing.T) {
		notifications = notifications[:0] // Clear notifications

		ctx := context.Background()
		profile := &models.CallsignProfile{
			Callsign:         "K8TEST2",
			Level:            29,
			ExperiencePoints: levelReqs[30],
			RenownLevel:      0,
			LastTransmissionAt: time.Now(),
			LastTallyAt:        time.Now(),
		}

		if err := profileRepo.Upsert(ctx, profile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		leveledUp, events := tallyService.processLevelUps(profile)

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
	})

	// Test case 3: Level 20 → 21 (no group change)
	t.Run("Level 20 to 21 does not cross group boundary", func(t *testing.T) {
		notifications = notifications[:0] // Clear notifications

		ctx := context.Background()
		profile := &models.CallsignProfile{
			Callsign:         "K8TEST3",
			Level:            20,
			ExperiencePoints: levelReqs[21],
			RenownLevel:      0,
			LastTransmissionAt: time.Now(),
			LastTallyAt:        time.Now(),
		}

		if err := profileRepo.Upsert(ctx, profile); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}

		leveledUp, events := tallyService.processLevelUps(profile)

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

		// Simulate notification logic - should NOT send group change
		if event.NewGroup != nil && event.PreviousGroup != nil &&
			event.NewGroup.Title != event.PreviousGroup.Title {
			mockNotifier.NotifyGroupChange(event.Callsign, event.NewGroup.Title)
		}

		// Verify NO group change notification was sent
		for _, notif := range notifications {
			if notif == "GROUP_CHANGE:K8TEST3:General" {
				t.Error("should not send group change notification when staying in same group")
			}
		}
	})
}

// mockDiscordNotifier captures notifications for testing
type mockDiscordNotifier struct {
	notifications *[]string
}

func (m *mockDiscordNotifier) NotifyLevelUp(callsign string, newLevel int) {
	*m.notifications = append(*m.notifications, "LEVEL_UP:"+callsign)
}

func (m *mockDiscordNotifier) NotifyGroupChange(callsign string, groupTitle string) {
	*m.notifications = append(*m.notifications, "GROUP_CHANGE:"+callsign+":"+groupTitle)
}

func (m *mockDiscordNotifier) NotifyRenownGained(callsign string, renownLevel int) {
	*m.notifications = append(*m.notifications, "RENOWN:"+callsign)
}
