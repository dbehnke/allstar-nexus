package gamification

import (
	"fmt"
	"strconv"
	"strings"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
)

// GroupingInfo contains information about a level group
type GroupingInfo struct {
	Title string `json:"title"`
	Badge string `json:"badge"`
	Color string `json:"color"`
	Min   int    `json:"min_level"`
	Max   int    `json:"max_level"`
}

// DefaultLevelGroupings returns the default level grouping configuration
func DefaultLevelGroupings() []cfgpkg.LevelGrouping {
	return []cfgpkg.LevelGrouping{
		{Levels: "1-9", Title: "Novice", Badge: "🌱", Color: "#10b981"},
		{Levels: "11-19", Title: "Technician", Badge: "🔧", Color: "#3b82f6"},
		{Levels: "21-29", Title: "General", Badge: "📡", Color: "#8b5cf6"},
		{Levels: "30-39", Title: "Advanced", Badge: "🎯", Color: "#f59e0b"},
		{Levels: "40-49", Title: "Extra", Badge: "💎", Color: "#ef4444"},
		{Levels: "50-55", Title: "Elmer", Badge: "🧙", Color: "#ec4899"},
		{Levels: "56-60", Title: "Professor", Badge: "🎓", Color: "#6366f1"},
	}
}

// ValidateGroupings checks that level groupings don't overlap
func ValidateGroupings(groupings []cfgpkg.LevelGrouping) error {
	if len(groupings) == 0 {
		return nil
	}

	// Track which levels are assigned
	assigned := make(map[int]string)

	for _, g := range groupings {
		start, end, err := parseLevelRangeStrict(g.Levels)
		if err != nil {
			return fmt.Errorf("invalid level range %q: %w", g.Levels, err)
		}

		for level := start; level <= end; level++ {
			if existing, ok := assigned[level]; ok {
				return fmt.Errorf("level %d is assigned to both %q and %q - ranges overlap", level, existing, g.Title)
			}
			assigned[level] = g.Title
		}
	}

	return nil
}

// GetGroupingForLevel returns the grouping info for a given level
// Returns nil if no grouping is defined for that level
func GetGroupingForLevel(level int, groupings []cfgpkg.LevelGrouping) *GroupingInfo {
	for _, g := range groupings {
		start, end, err := parseLevelRangeStrict(g.Levels)
		if err != nil {
			continue
		}

		if level >= start && level <= end {
			return &GroupingInfo{
				Title: g.Title,
				Badge: g.Badge,
				Color: g.Color,
				Min:   start,
				Max:   end,
			}
		}
	}

	return nil
}

// BuildGroupingsMap builds a map of level -> GroupingInfo for quick lookup
func BuildGroupingsMap(groupings []cfgpkg.LevelGrouping) map[int]*GroupingInfo {
	result := make(map[int]*GroupingInfo)

	for _, g := range groupings {
		start, end, err := parseLevelRangeStrict(g.Levels)
		if err != nil {
			continue
		}

		info := &GroupingInfo{
			Title: g.Title,
			Badge: g.Badge,
			Color: g.Color,
			Min:   start,
			Max:   end,
		}

		for level := start; level <= end; level++ {
			result[level] = info
		}
	}

	return result
}

// parseLevelRangeStrict parses a level range string like "1-10" or "50"
// Returns (start, end, error)
func parseLevelRangeStrict(r string) (int, int, error) {
	r = strings.TrimSpace(r)
	if r == "" {
		return 0, 0, fmt.Errorf("empty range")
	}

	if strings.Contains(r, "-") {
		parts := strings.Split(r, "-")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid range format (use START-END)")
		}

		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start: %w", err)
		}

		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end: %w", err)
		}

		if start > end {
			return 0, 0, fmt.Errorf("start (%d) must be <= end (%d)", start, end)
		}

		return start, end, nil
	}

	// Single level
	level, err := strconv.Atoi(r)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid level: %w", err)
	}

	return level, level, nil
}
