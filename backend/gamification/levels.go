package gamification

import (
	"math"
	"strconv"
	"strings"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
)

// CalculateLevelRequirements generates XP requirements for all 60 levels
// This is the LOW-ACTIVITY HUB version with 10x reduction
func CalculateLevelRequirements() map[int]int {
	requirements := make(map[int]int)

	// Levels 1-10: Linear scaling (360 XP = 6 minutes each)
	// Low-activity scaling: 10x reduction from standard (was 3600)
	for level := 1; level <= 10; level++ {
		requirements[level] = 360 // 6 minutes per level
	}

	// Levels 11-60: Logarithmic scaling
	// Target: 259,200 total XP (72 hours = ~36 weeks at 2hr/week cap)
	// Already used: 10 * 360 = 3,600 XP
	// Remaining: 255,600 XP across 50 levels
	// Using k = level-1 so level 11 anchors to k=10 (ensuring it's > 360)

	totalRemaining := 255600.0
	sum := 0.0
	for level := 11; level <= 60; level++ {
		sum += math.Pow(float64(level-1), 1.8)
	}

	scaleFactor := totalRemaining / sum
	for level := 11; level <= 60; level++ {
		xp := int(math.Pow(float64(level-1), 1.8) * scaleFactor)
		requirements[level] = xp
	}

	return requirements
}

// CalculateLevelRequirementsWithScale builds the level requirements using provided scale config.
// If scale is empty or invalid, falls back to low-activity defaults (CalculateLevelRequirements).
func CalculateLevelRequirementsWithScale(scale []cfgpkg.LevelScaleConfig) map[int]int {
	if len(scale) == 0 {
		return CalculateLevelRequirements()
	}

	// Start with empty map; fill based on entries in order. Later entries override earlier ones.
	req := make(map[int]int)

	for _, s := range scale {
		start, end := parseLevelRange(s.Levels)
		if start < 1 {
			start = 1
		}
		if end > 60 {
			end = 60
		}
		if start > end {
			continue
		}

		// Linear: explicit xp_per_level
		if s.XPPerLevel > 0 || strings.EqualFold(s.Scaling, "linear") {
			xp := s.XPPerLevel
			if xp <= 0 {
				// sensible default if scaling=linear but xp not provided
				xp = 360
			}
			for lvl := start; lvl <= end; lvl++ {
				req[lvl] = xp
			}
			continue
		}

		// Logarithmic: distribute TargetTotalSeconds across range using (i^1.8)
		if strings.EqualFold(s.Scaling, "logarithmic") && s.TargetTotalSeconds > 0 {
			totalRemaining := float64(s.TargetTotalSeconds)
			sum := 0.0
			for lvl := start; lvl <= end; lvl++ {
				sum += math.Pow(float64(lvl-start+1), 1.8)
			}
			if sum <= 0 {
				continue
			}
			scaleFactor := totalRemaining / sum
			for lvl := start; lvl <= end; lvl++ {
				xp := int(math.Pow(float64(lvl-start+1), 1.8) * scaleFactor)
				if xp < 1 {
					xp = 1
				}
				req[lvl] = xp
			}
			continue
		}
	}

	// Ensure we have all levels; if any missing, backfill with default plan for those.
	if len(req) < 60 {
		def := CalculateLevelRequirements()
		for lvl := 1; lvl <= 60; lvl++ {
			if _, ok := req[lvl]; !ok {
				req[lvl] = def[lvl]
			}
		}
	}

	return req
}

func parseLevelRange(r string) (int, int) {
	r = strings.TrimSpace(r)
	if r == "" {
		return 1, 60
	}
	parts := strings.Split(r, "-")
	if len(parts) == 1 {
		v, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		return v, v
	}
	a, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	b, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	if a > b {
		a, b = b, a
	}
	return a, b
}

// GetTotalXPForLevel returns cumulative XP needed to reach a given level
func GetTotalXPForLevel(level int, requirements map[int]int) int {
	total := 0
	for lvl := 1; lvl < level; lvl++ {
		if xp, ok := requirements[lvl]; ok {
			total += xp
		}
	}
	return total
}

// GetLevelFromXP returns the level for a given total XP amount
func GetLevelFromXP(totalXP int, requirements map[int]int) int {
	accumulated := 0
	for level := 1; level <= 60; level++ {
		if xp, ok := requirements[level]; ok {
			accumulated += xp
			if totalXP < accumulated {
				return level - 1
			}
		}
	}
	return 60 // Max level
}
