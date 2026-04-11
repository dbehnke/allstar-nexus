package gamification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// DiscordNotifier sends notifications to Discord via webhook
type DiscordNotifier struct {
	webhookURL string
	enabled    bool
	logger     *zap.Logger
	httpClient *http.Client
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(webhookURL string, enabled bool, logger *zap.Logger) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		enabled:    enabled,
		logger:     logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// discordWebhookPayload represents the JSON structure for Discord webhooks
type discordWebhookPayload struct {
	Content string `json:"content"`
}

// NotifyLevelUp sends a notification when a user levels up
func (d *DiscordNotifier) NotifyLevelUp(callsign string, newLevel int) {
	if !d.enabled || d.webhookURL == "" {
		return
	}

	message := fmt.Sprintf("🎉 **%s** has achieved level **%d**!", callsign, newLevel)
	d.sendNotification(message)
}

// NotifyGroupChange sends a notification when a user enters a new level grouping
func (d *DiscordNotifier) NotifyGroupChange(callsign string, groupTitle string) {
	if !d.enabled || d.webhookURL == "" {
		return
	}

	message := fmt.Sprintf("🎖️ **%s** has reached new rank of **%s**!", callsign, groupTitle)
	d.sendNotification(message)
}

// NotifyRenownGained sends a notification when a user gains a renown level
func (d *DiscordNotifier) NotifyRenownGained(callsign string, renownLevel int) {
	if !d.enabled || d.webhookURL == "" {
		return
	}

	message := fmt.Sprintf("🌟 **%s** has achieved **Renown Level %d**!", callsign, renownLevel)
	d.sendNotification(message)
}

// sendNotification sends a message to Discord
func (d *DiscordNotifier) sendNotification(message string) {
	// Send async to avoid blocking the tally service
	go func() {
		payload := discordWebhookPayload{
			Content: message,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			d.logger.Error("failed to marshal Discord payload", zap.Error(err))
			return
		}

		resp, err := d.httpClient.Post(d.webhookURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			d.logger.Error("failed to send Discord notification", zap.Error(err))
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			d.logger.Warn("Discord webhook returned non-success status",
				zap.Int("status", resp.StatusCode),
				zap.String("message", message))
		} else {
			d.logger.Debug("Discord notification sent successfully",
				zap.String("message", message))
		}
	}()
}
