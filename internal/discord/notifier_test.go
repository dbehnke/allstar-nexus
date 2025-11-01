package discord

import (
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/core"
)

func TestNotifier_StateTransitions(t *testing.T) {
	config := Config{
		WebhookURL:            "", // Empty to avoid actual HTTP calls
		QSOInactiveSeconds:    2,  // 2 seconds for faster testing
		NodeIdleSeconds:       5,  // 5 seconds for faster testing
		MinTalkersForQSO:      2,
		NotifyIndividualTalks: true,
		CheckInterval:         500 * time.Millisecond, // Fast checks for testing
	}

	notifier := NewNotifier(config, 594950, "594950")
	notifier.Start()
	defer notifier.Stop()

	// Initially should be idle
	if notifier.GetCurrentState() != StateIdle {
		t.Errorf("Expected initial state to be idle, got %v", notifier.GetCurrentState())
	}

	// Simulate first talker starting
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind:     "TX_START",
		Node:     550465,
		Callsign: "KF8S",
		At:       time.Now(),
	})

	// Should transition to active
	if notifier.GetCurrentState() != StateActive {
		t.Errorf("Expected state to be active after first talker, got %v", notifier.GetCurrentState())
	}
	if notifier.GetActiveTalkerCount() != 1 {
		t.Errorf("Expected 1 active talker, got %d", notifier.GetActiveTalkerCount())
	}

	// Simulate second talker starting
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind:     "TX_START",
		Node:     55532,
		Callsign: "KE8VSI",
		At:       time.Now(),
	})

	// Should transition to QSO
	if notifier.GetCurrentState() != StateQSO {
		t.Errorf("Expected state to be QSO after second talker, got %v", notifier.GetCurrentState())
	}
	if notifier.GetActiveTalkerCount() != 2 {
		t.Errorf("Expected 2 active talkers, got %d", notifier.GetActiveTalkerCount())
	}

	// Simulate both talkers stopping
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind: "TX_STOP",
		Node: 550465,
		At:   time.Now(),
	})
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind: "TX_STOP",
		Node: 55532,
		At:   time.Now(),
	})

	// Wait for timeout to trigger cleanup (QSO inactive timeout is 2 seconds + check interval margin)
	time.Sleep(3 * time.Second)

	// Should eventually transition to idle after timeout
	if notifier.GetCurrentState() != StateIdle {
		t.Errorf("Expected state to be idle after timeout, got %v", notifier.GetCurrentState())
	}
	if notifier.GetActiveTalkerCount() != 0 {
		t.Errorf("Expected 0 active talkers after timeout, got %d", notifier.GetActiveTalkerCount())
	}
}

func TestNotifier_SkipsNodeZeroEvents(t *testing.T) {
	config := Config{
		WebhookURL:            "",
		QSOInactiveSeconds:    120,
		NodeIdleSeconds:       300,
		MinTalkersForQSO:      2,
		NotifyIndividualTalks: true,
	}

	notifier := NewNotifier(config, 594950, "594950")

	// Process a node==0 event (should be ignored)
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind: "TX_START",
		Node: 0,
		At:   time.Now(),
	})

	// Should still be idle
	if notifier.GetCurrentState() != StateIdle {
		t.Errorf("Expected state to remain idle after node==0 event, got %v", notifier.GetCurrentState())
	}
	if notifier.GetActiveTalkerCount() != 0 {
		t.Errorf("Expected 0 active talkers after node==0 event, got %d", notifier.GetActiveTalkerCount())
	}
}

func TestNotifier_SingleTalkerTransitions(t *testing.T) {
	config := Config{
		WebhookURL:            "",
		QSOInactiveSeconds:    2,
		NodeIdleSeconds:       4,
		MinTalkersForQSO:      2,
		NotifyIndividualTalks: true,
		CheckInterval:         500 * time.Millisecond,
	}

	notifier := NewNotifier(config, 594950, "594950")
	notifier.Start()
	defer notifier.Stop()

	// Single talker starts
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind:     "TX_START",
		Node:     550465,
		Callsign: "KF8S",
		At:       time.Now(),
	})

	// Should be active (not QSO with only 1 talker)
	if notifier.GetCurrentState() != StateActive {
		t.Errorf("Expected state to be active with single talker, got %v", notifier.GetCurrentState())
	}

	// Talker stops
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind: "TX_STOP",
		Node: 550465,
		At:   time.Now(),
	})

	// Wait for node idle timeout
	time.Sleep(5 * time.Second)

	// Should transition to idle
	if notifier.GetCurrentState() != StateIdle {
		t.Errorf("Expected state to be idle after single talker timeout, got %v", notifier.GetCurrentState())
	}
}

func TestActivityState_String(t *testing.T) {
	tests := []struct {
		state    ActivityState
		expected string
	}{
		{StateIdle, "idle"},
		{StateActive, "active"},
		{StateQSO, "qso"},
		{ActivityState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("ActivityState.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestNotifier_MultipleStartEvents(t *testing.T) {
	config := Config{
		WebhookURL:            "",
		QSOInactiveSeconds:    120,
		NodeIdleSeconds:       300,
		MinTalkersForQSO:      2,
		NotifyIndividualTalks: false, // Disable individual notifications
	}

	notifier := NewNotifier(config, 594950, "594950")

	// Same talker starts multiple times (should only count once)
	now := time.Now()
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind:     "TX_START",
		Node:     550465,
		Callsign: "KF8S",
		At:       now,
	})
	notifier.ProcessTalkerEvent(core.TalkerEvent{
		Kind:     "TX_START",
		Node:     550465,
		Callsign: "KF8S",
		At:       now.Add(1 * time.Second),
	})

	// Should still only have 1 talker
	if notifier.GetActiveTalkerCount() != 1 {
		t.Errorf("Expected 1 active talker after duplicate TX_START, got %d", notifier.GetActiveTalkerCount())
	}
}
