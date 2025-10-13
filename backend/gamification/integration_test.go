package gamification

import (
	"testing"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
)

// TestIntegration_DefaultGroupings ensures default groupings work with validation
func TestIntegration_DefaultGroupings(t *testing.T) {
	groupings := DefaultLevelGroupings()

	// Should not have overlaps
	if err := ValidateGroupings(groupings); err != nil {
		t.Fatalf("default groupings should be valid: %v", err)
	}

	// Check that common levels are covered
	testCases := []struct {
		level    int
		expected string
	}{
		{1, "Novice"},
		{5, "Novice"},
		{9, "Novice"},
		{11, "Technician"},
		{15, "Technician"},
		{19, "Technician"},
		{21, "General"},
		{25, "General"},
		{30, "Advanced"},
		{35, "Advanced"},
		{40, "Extra"},
		{45, "Extra"},
		{50, "Elmer"},
		{55, "Elmer"},
		{56, "Professor"},
		{60, "Professor"},
	}

	for _, tc := range testCases {
		info := GetGroupingForLevel(tc.level, groupings)
		if info == nil {
			t.Errorf("level %d should have a grouping, got nil", tc.level)
		} else if info.Title != tc.expected {
			t.Errorf("level %d: expected %s, got %s", tc.level, tc.expected, info.Title)
		}
	}

	// Check gaps (levels 10 and 20 should not be assigned)
	if info := GetGroupingForLevel(10, groupings); info != nil {
		t.Errorf("level 10 should not have a grouping (gap), got %+v", info)
	}
	if info := GetGroupingForLevel(20, groupings); info != nil {
		t.Errorf("level 20 should not have a grouping (gap), got %+v", info)
	}
}

// TestIntegration_CustomGroupingsValidation ensures custom groupings are validated
func TestIntegration_CustomGroupingsValidation(t *testing.T) {
	// Valid custom groupings
	validGroupings := []cfgpkg.LevelGrouping{
		{Levels: "1-20", Title: "Beginner", Badge: "🌱"},
		{Levels: "21-40", Title: "Advanced", Badge: "⚡"},
		{Levels: "41-60", Title: "Expert", Badge: "👑"},
	}

	if err := ValidateGroupings(validGroupings); err != nil {
		t.Errorf("valid custom groupings should pass validation: %v", err)
	}

	// Invalid groupings with overlap
	invalidGroupings := []cfgpkg.LevelGrouping{
		{Levels: "1-25", Title: "Beginner", Badge: "🌱"},
		{Levels: "20-40", Title: "Advanced", Badge: "⚡"}, // overlaps at 20-25
	}

	if err := ValidateGroupings(invalidGroupings); err == nil {
		t.Error("overlapping groupings should fail validation")
	}
}

// TestIntegration_RenownOverridesBadge ensures renown users don't get level groupings
func TestIntegration_RenownOverridesBadge(t *testing.T) {
	// This test documents the expected behavior:
	// - If a user has renown_level > 0, the frontend shows the renown badge (⭐)
	// - If renown_level == 0, the frontend shows the level grouping badge
	// - The backend API should return grouping as nil for renown users

	// This is tested in the API layer, but we document it here for clarity
	t.Log("Renown users (renown_level > 0) should display ⭐ instead of level group badges")
	t.Log("This is handled in the backend API (gamification.go) by not including grouping when renown_level > 0")
	t.Log("The frontend ScoreboardCard.vue checks for renown_level and shows ⭐ badge accordingly")
}
