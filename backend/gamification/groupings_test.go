package gamification

import (
	"testing"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
)

func TestValidateGroupings_NoOverlap(t *testing.T) {
	groupings := []cfgpkg.LevelGrouping{
		{Levels: "1-9", Title: "Novice", Badge: "🌱"},
		{Levels: "11-19", Title: "Technician", Badge: "📻"},
		{Levels: "21-29", Title: "General", Badge: "⚡"},
	}

	err := ValidateGroupings(groupings)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateGroupings_WithOverlap(t *testing.T) {
	groupings := []cfgpkg.LevelGrouping{
		{Levels: "1-10", Title: "Novice", Badge: "🌱"},
		{Levels: "10-20", Title: "General", Badge: "📻"}, // overlaps at 10
	}

	err := ValidateGroupings(groupings)
	if err == nil {
		t.Error("expected error due to overlap, got nil")
	}
}

func TestValidateGroupings_EmptyList(t *testing.T) {
	err := ValidateGroupings([]cfgpkg.LevelGrouping{})
	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
}

func TestValidateGroupings_InvalidRange(t *testing.T) {
	groupings := []cfgpkg.LevelGrouping{
		{Levels: "invalid", Title: "Test", Badge: "🌱"},
	}

	err := ValidateGroupings(groupings)
	if err == nil {
		t.Error("expected error due to invalid range, got nil")
	}
}

func TestGetGroupingForLevel(t *testing.T) {
	groupings := []cfgpkg.LevelGrouping{
		{Levels: "1-9", Title: "Novice", Badge: "🌱", Color: "#10b981"},
		{Levels: "10-19", Title: "Technician", Badge: "📻", Color: "#3b82f6"},
		{Levels: "20-29", Title: "General", Badge: "📡", Color: "#8b5cf6"},
	}

	tests := []struct {
		level    int
		expected *string // nil means no grouping
	}{
		{5, stringPtr("Novice")},
		{9, stringPtr("Novice")},
		{10, stringPtr("Technician")},
		{11, stringPtr("Technician")},
		{15, stringPtr("Technician")},
		{20, stringPtr("General")},
	}

	for _, tc := range tests {
		info := GetGroupingForLevel(tc.level, groupings)
		if tc.expected == nil {
			if info != nil {
				t.Errorf("level %d: expected nil, got %+v", tc.level, info)
			}
		} else {
			if info == nil {
				t.Errorf("level %d: expected %s, got nil", tc.level, *tc.expected)
			} else if info.Title != *tc.expected {
				t.Errorf("level %d: expected %s, got %s", tc.level, *tc.expected, info.Title)
			}
		}
	}
}

func TestBuildGroupingsMap(t *testing.T) {
	groupings := []cfgpkg.LevelGrouping{
		{Levels: "1-3", Title: "Novice", Badge: "🌱"},
		{Levels: "5-7", Title: "General", Badge: "📻"},
	}

	m := BuildGroupingsMap(groupings)

	// Check expected levels
	if m[1] == nil || m[1].Title != "Novice" {
		t.Error("level 1 should map to Novice")
	}
	if m[3] == nil || m[3].Title != "Novice" {
		t.Error("level 3 should map to Novice")
	}
	if m[4] != nil {
		t.Error("level 4 should not have a grouping (gap)")
	}
	if m[5] == nil || m[5].Title != "General" {
		t.Error("level 5 should map to General")
	}
}

func TestDefaultLevelGroupings(t *testing.T) {
	defaults := DefaultLevelGroupings()

	if len(defaults) == 0 {
		t.Error("expected default groupings, got empty list")
	}

	// Should have no overlaps
	err := ValidateGroupings(defaults)
	if err != nil {
		t.Errorf("default groupings should be valid, got error: %v", err)
	}

	// Check that defaults include expected groups
	titles := make(map[string]bool)
	for _, g := range defaults {
		titles[g.Title] = true
	}

	expected := []string{"Novice", "Technician", "General", "Advanced", "Extra", "Elmer", "Professor"}
	for _, exp := range expected {
		if !titles[exp] {
			t.Errorf("expected default grouping %q not found", exp)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
