# Discord Notifications for Gamification

This document describes the Discord webhook notifications feature for the Allstar Nexus gamification system.

## Overview

When gamification is enabled, Allstar Nexus can send Discord notifications to celebrate user achievements:

1. **Level Up Notifications** - When a user reaches a new level
2. **Group Transition Notifications** - When a user enters a new tier (e.g., from "General" to "Advanced")
3. **Renown Notifications** - When a user gains a renown level (prestige system)

## Setup

### 1. Create a Discord Webhook

1. Go to your Discord server
2. Navigate to Server Settings → Integrations → Webhooks
3. Click "New Webhook"
4. Give it a name (e.g., "Allstar Nexus")
5. Select the channel where notifications should be posted
6. Copy the webhook URL

### 2. Configure Allstar Nexus

Add the following to your `config.yaml`:

```yaml
gamification:
  enabled: true
  
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
```

Or set via environment variables:

```bash
GAMIFICATION_DISCORD_ENABLED=true
GAMIFICATION_DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
```

### 3. Restart Allstar Nexus

Restart the service to apply the configuration changes.

## Notification Examples

### Level Up
```
🎉 **K8FBI** has achieved level **15**!
```

### Group Transition
```
⭐ **KF8S** is now **Advanced**!
```

### Renown Level
```
🌟 **W1ABC** has achieved **Renown Level 3**!
```

## Default Level Groups

The default level groupings are:
- **Novice** (Levels 1-9) - 🌱
- **Technician** (Levels 10-19) - 🔧
- **General** (Levels 20-29) - 📡
- **Advanced** (Levels 30-39) - 🎯
- **Extra** (Levels 40-49) - 💎
- **Elmer** (Levels 50-55) - 🧙
- **Professor** (Levels 56-60) - 🎓

You can customize these groupings in your `config.yaml`.

## Technical Details

### Implementation

- Notifications are sent asynchronously to avoid blocking the gamification tally service
- The Discord webhook URL is validated on startup
- Failed webhook calls are logged but do not interrupt the gamification system
- Rate limiting is handled by sending notifications asynchronously

### Security

- Keep your webhook URL secret - anyone with the URL can post to your Discord channel
- Consider rotating webhook URLs periodically if they're exposed
- Set appropriate channel permissions to control who can see notifications

### Troubleshooting

#### Notifications not appearing

1. Check that `gamification.discord.enabled` is set to `true`
2. Verify the webhook URL is correct and active
3. Check the logs for any errors related to Discord notifications
4. Ensure the Discord channel hasn't been deleted or the webhook revoked

#### Too many/few notifications

The frequency of notifications depends on:
- How often users level up (based on transmission activity)
- The tally interval (default 30 minutes)
- Level progression curve settings

To adjust notification frequency, modify the gamification settings in `config.yaml`.

## API Integration

The Discord notifier can be accessed programmatically through the `gamification.DiscordNotifier` interface:

```go
notifier := gamification.NewDiscordNotifier(webhookURL, enabled, logger)
notifier.NotifyLevelUp("K8FBI", 15)
notifier.NotifyGroupChange("KF8S", "Advanced")
notifier.NotifyRenownGained("W1ABC", 3)
```

## Future Enhancements

Potential future features:
- Configurable message templates
- Support for Discord embeds with rich formatting
- Mention/role pings for specific achievements
- Configurable emoji for different levels
- Daily/weekly summary reports
- Top talker leaderboard posts
