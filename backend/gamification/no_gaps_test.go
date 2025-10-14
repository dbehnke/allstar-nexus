package gamification

import "testing"

// TestDefaultGroupingsCoverAllLevels ensures there are no gaps in the default
// groupings across levels 1..60.
func TestDefaultGroupingsCoverAllLevels(t *testing.T) {
	m := BuildGroupingsMap(DefaultLevelGroupings())

	for lvl := 1; lvl <= 60; lvl++ {
		if _, ok := m[lvl]; !ok {
			t.Fatalf("default groupings missing level %d", lvl)
		}
	}
}
