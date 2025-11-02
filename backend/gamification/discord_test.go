package gamification

import (
	"testing"

	"go.uber.org/zap"
)

func TestDiscordNotifier_NotifyLevelUp(t *testing.T) {
	// Test with disabled notifier
	logger := zap.NewNop()
	notifier := NewDiscordNotifier("", false, logger)

	// Should not panic with empty webhook URL and disabled
	notifier.NotifyLevelUp("K8FBI", 10)
	notifier.NotifyGroupChange("K8FBI", "Technician")
	notifier.NotifyRenownGained("K8FBI", 1)

	// Test with enabled but empty webhook URL (should not panic)
	notifier2 := NewDiscordNotifier("", true, logger)
	notifier2.NotifyLevelUp("K8FBI", 10)
	notifier2.NotifyGroupChange("K8FBI", "Technician")
	notifier2.NotifyRenownGained("K8FBI", 1)

	// Test with fake webhook URL (will fail to send but should not panic)
	notifier3 := NewDiscordNotifier("http://invalid-webhook-url.example.com/webhook", true, logger)
	notifier3.NotifyLevelUp("K8FBI", 10)
	notifier3.NotifyGroupChange("K8FBI", "Technician")
	notifier3.NotifyRenownGained("K8FBI", 1)

	// All tests pass if no panic occurred
}

func TestDiscordNotifier_MessageFormat(t *testing.T) {
	// This test verifies that the messages are formatted correctly
	// We can't easily test the actual HTTP call without mocking, but we can
	// verify that the notifier doesn't panic with various inputs

	logger := zap.NewNop()
	notifier := NewDiscordNotifier("", false, logger)

	tests := []struct {
		name      string
		callsign  string
		level     int
		grouping  string
		operation func()
	}{
		{
			name:     "Level up with simple callsign",
			callsign: "K8FBI",
			level:    15,
			operation: func() {
				notifier.NotifyLevelUp("K8FBI", 15)
			},
		},
		{
			name:     "Level up with compound callsign",
			callsign: "KF8S",
			level:    32,
			operation: func() {
				notifier.NotifyLevelUp("KF8S", 32)
			},
		},
		{
			name:     "Group change to Advanced",
			callsign: "KF8S",
			grouping: "Advanced",
			operation: func() {
				notifier.NotifyGroupChange("KF8S", "Advanced")
			},
		},
		{
			name:     "Group change to General",
			callsign: "K8FBI",
			grouping: "General",
			operation: func() {
				notifier.NotifyGroupChange("K8FBI", "General")
			},
		},
		{
			name:     "Renown level 1",
			callsign: "W1ABC",
			level:    1,
			operation: func() {
				notifier.NotifyRenownGained("W1ABC", 1)
			},
		},
		{
			name:     "Renown level 5",
			callsign: "N2XYZ",
			level:    5,
			operation: func() {
				notifier.NotifyRenownGained("N2XYZ", 5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.operation()
		})
	}
}

func TestDiscordNotifier_WebhookURL(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name       string
		webhookURL string
		enabled    bool
	}{
		{
			name:       "Empty URL disabled",
			webhookURL: "",
			enabled:    false,
		},
		{
			name:       "Empty URL enabled",
			webhookURL: "",
			enabled:    true,
		},
		{
			name:       "Valid looking URL format",
			webhookURL: "https://discord.com/api/webhooks/123456789/abcdefghijklmnop",
			enabled:    true,
		},
		{
			name:       "Invalid URL format",
			webhookURL: "not-a-url",
			enabled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewDiscordNotifier(tt.webhookURL, tt.enabled, logger)
			if notifier == nil {
				t.Fatal("NewDiscordNotifier returned nil")
			}

			// Verify notifier doesn't panic with various calls
			notifier.NotifyLevelUp("TEST", 1)
			notifier.NotifyGroupChange("TEST", "Novice")
			notifier.NotifyRenownGained("TEST", 1)
		})
	}
}
