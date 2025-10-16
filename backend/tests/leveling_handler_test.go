package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/dbehnke/allstar-nexus/backend/api"
	"github.com/dbehnke/allstar-nexus/backend/gamification"
	"github.com/dbehnke/allstar-nexus/backend/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestLevelConfigHandler_UsesPrecomputedValues tests that the LevelConfig handler
// uses the precomputed values from the gamification package accessor instead of
// recalculating on every request.
func TestLevelConfigHandler_UsesPrecomputedValues(t *testing.T) {
	// Set up known test level requirements
	testRequirements := map[int]int{
		1:  100,
		2:  200,
		3:  300,
		4:  400,
		5:  500,
		10: 1000,
		20: 2000,
		30: 3000,
		60: 6000,
	}

	// Seed the gamification package with test values
	gamification.SetLevelRequirements(testRequirements)

	// Set up test database (in-memory)
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DSN: ":memory:",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create API with minimal dependencies
	txLogRepo := repository.NewTransmissionLogRepository(db)
	profileRepo := repository.NewCallsignProfileRepo(db)
	levelRepo := repository.NewLevelConfigRepo(db)
	activityRepo := repository.NewXPActivityRepo(db)

	gamificationAPI := api.NewGamificationAPI(
		profileRepo,
		txLogRepo,
		levelRepo,
		activityRepo,
		nil, // levelGroupings
		false, // renownEnabled
		0,     // renownXPPerLevel
		false, // restedEnabled
		0.0,   // restedAccumulationRate
		0,     // restedMaxHours
		0.0,   // restedMultiplier
		0,     // restedIdleThresholdSec
		0,     // weeklyCapSeconds
		0,     // dailyCapSeconds
		nil,   // drTiers
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/gamification/level-config", nil)
	w := httptest.NewRecorder()

	// Call handler
	gamificationAPI.LevelConfig(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify "config" field exists
	configField, ok := response["config"]
	if !ok {
		t.Fatal("response missing 'config' field")
	}

	// The config is actually a map[int]int that gets marshaled to JSON.
	// When unmarshaling back, JSON will give us the numbers as the structure depends on encoding.
	// Let's re-marshal and unmarshal to get the proper structure.
	configBytes, err := json.Marshal(configField)
	if err != nil {
		t.Fatalf("failed to re-marshal config: %v", err)
	}
	
	var configMapInt map[string]int
	if err := json.Unmarshal(configBytes, &configMapInt); err != nil {
		t.Fatalf("failed to unmarshal config as map[string]int: %v", err)
	}

	// Verify key test values match what we seeded
	testCases := []struct {
		level    int
		expected int
	}{
		{1, 100},
		{2, 200},
		{5, 500},
		{10, 1000},
		{20, 2000},
		{60, 6000},
	}

	for _, tc := range testCases {
		// JSON marshals int map keys as strings
		levelKey := strconv.Itoa(tc.level)
		
		actual, ok := configMapInt[levelKey]
		if !ok {
			t.Errorf("level %d (key '%s') not found in config map", tc.level, levelKey)
			t.Logf("Available keys: %v", getKeysFromIntMap(configMapInt))
			continue
		}

		if actual != tc.expected {
			t.Errorf("level %d: expected %d XP, got %d", tc.level, tc.expected, actual)
		}
	}

	// Verify response includes other expected fields
	if _, ok := response["groupings"]; !ok {
		t.Error("response missing 'groupings' field")
	}
	if _, ok := response["renown_enabled"]; !ok {
		t.Error("response missing 'renown_enabled' field")
	}
}

// Helper function to get keys from a map for debugging
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function to get keys from a int map for debugging
func getKeysFromIntMap(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
