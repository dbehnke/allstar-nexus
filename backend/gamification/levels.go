package gamification

import "math"

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

	totalRemaining := 255600.0
	sum := 0.0
	for level := 11; level <= 60; level++ {
		sum += math.Pow(float64(level-10), 1.8)
	}

	scaleFactor := totalRemaining / sum
	for level := 11; level <= 60; level++ {
		xp := int(math.Pow(float64(level-10), 1.8) * scaleFactor)
		requirements[level] = xp
	}

	return requirements
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
