package tests

import (
	"testing"
	"time"

	cfgpkg "github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/gamification"
)

func TestDefaultLevelRequirements_BasicShape(t *testing.T) {
	req := gamification.CalculateLevelRequirements()
	if len(req) != 60 {
		t.Fatalf("expected 60 levels, got %d", len(req))
	}
	// Levels 1-10 should be 360 each
	for lvl := 1; lvl <= 10; lvl++ {
		if req[lvl] != 360 {
			t.Fatalf("level %d expected 360 got %d", lvl, req[lvl])
		}
	}
	// Level 11 should be greater than 360 (using level-anchored weights)
	if req[11] <= 360 {
		t.Fatalf("level 11 expected to be > 360, got %d", req[11])
	}
	// Total should be close to 259,200 (allow small rounding variance)
	total := 0
	for lvl := 1; lvl <= 60; lvl++ {
		total += req[lvl]
	}
	if total < 255000 || total > 262000 {
		t.Fatalf("unexpected total xp %d", total)
	}
}

func TestLevelRequirementsWithScale_LinearOverride(t *testing.T) {
	scale := []cfgpkg.LevelScaleConfig{{Levels: "1-5", XPPerLevel: 100, Scaling: "linear"}}
	req := gamification.CalculateLevelRequirementsWithScale(scale)
	for lvl := 1; lvl <= 5; lvl++ {
		if req[lvl] != 100 {
			t.Fatalf("level %d expected 100 got %d", lvl, req[lvl])
		}
	}
	// Spot check fallback uses default for an out-of-range level
	def := gamification.CalculateLevelRequirements()
	if req[12] != def[12] {
		t.Fatalf("expected fallback for level 12 to match default")
	}
}

func TestLevelRequirementsWithScale_LogarithmicTargetSum(t *testing.T) {
	target := 300
	scale := []cfgpkg.LevelScaleConfig{{Levels: "10-12", Scaling: "logarithmic", TargetTotalSeconds: target}}
	req := gamification.CalculateLevelRequirementsWithScale(scale)
	// Sum across 10-12 should be near target
	sum := req[10] + req[11] + req[12]
	// Allow ±15% due to integer rounding
	lower := int(float64(target) * 0.85)
	upper := int(float64(target) * 1.15)
	if sum < lower || sum > upper {
		t.Fatalf("sum(10..12)=%d not within [%d,%d]", sum, lower, upper)
	}
	_ = time.Now() // keep import for potential future timing checks
}
