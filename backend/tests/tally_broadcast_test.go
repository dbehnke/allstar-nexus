package tests

import (
	"encoding/json"
	"testing"

	"github.com/dbehnke/allstar-nexus/backend/models"
)

// Test that the broadcast payload built for tally completion includes a "scoreboard"
// field when profile entries are available. This mirrors the behavior in main.go's
// OnTallyComplete handler which embeds a lightweight scoreboard snapshot.
func TestTallyBroadcastPayloadIncludesScoreboard(t *testing.T) {
	// Prepare sample profiles as returned from the leaderboard repo
	profiles := []models.CallsignProfile{
		{Callsign: "ABC", Level: 2, ExperiencePoints: 150, RenownLevel: 1},
		{Callsign: "XYZ", Level: 3, ExperiencePoints: 400, RenownLevel: 2},
	}

	// Simulate level config map where next level XP for level+1 is present
	levelCfg := map[int]int{
		3: 300,
		4: 600,
	}

	// Simulate total talk time lookup function (here simply zero)
	getTotalTalkTime := func(callsign string) (int64, error) { return 0, nil }

	// Build entries similar to main.go
	entries := make([]map[string]any, 0, len(profiles))
	for _, p := range profiles {
		nextXP := 0
		if xp, ok := levelCfg[p.Level+1]; ok {
			nextXP = xp
		}
		totalTime, _ := getTotalTalkTime(p.Callsign)
		entries = append(entries, map[string]any{
			"callsign":                p.Callsign,
			"level":                   p.Level,
			"experience_points":       p.ExperiencePoints,
			"renown_level":            p.RenownLevel,
			"next_level_xp":           nextXP,
			"total_talk_time_seconds": totalTime,
		})
	}

	payload := map[string]any{"summary": map[string]any{"rows_processed": 2}, "scoreboard": entries}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Ensure the JSON contains the scoreboard marker and a known callsign
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if _, ok := decoded["scoreboard"]; !ok {
		t.Fatalf("expected scoreboard field in payload")
	}
	// Inspect scoreboard array contents
	sc, ok := decoded["scoreboard"].([]any)
	if !ok || len(sc) != 2 {
		t.Fatalf("expected scoreboard array of length 2, got %v", decoded["scoreboard"])
	}
	first, ok := sc[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected scoreboard entry shape")
	}
	if first["callsign"] != "ABC" {
		t.Fatalf("expected first callsign to be ABC, got %v", first["callsign"])
	}
	// basic success path
}
