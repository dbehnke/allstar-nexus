# Discord Webhook Integration

Allstar Nexus supports Discord webhook notifications to notify a Discord channel about node activity. This feature is useful for keeping track of when your node is active and when QSOs (conversations) are happening.

## Configuration

To enable Discord webhooks, add the following configuration to your `config.yaml`:

```yaml
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
  qso_inactive_seconds: 120      # Declare QSO ended after 2 minutes of inactivity
  node_idle_seconds: 300         # Declare node idle after 5 minutes of no activity
  min_talkers_for_qso: 2         # Minimum unique talkers to declare QSO started
  notify_individual_talks: true  # Notify when individual stations start talking
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable Discord webhook notifications |
| `webhook_url` | string | `""` | Your Discord webhook URL (see below for how to create one) |
| `qso_inactive_seconds` | int | `120` | Time in seconds after last transmission before declaring QSO has ended (2 minutes) |
| `node_idle_seconds` | int | `300` | Time in seconds after last activity before declaring node is idle (5 minutes) |
| `min_talkers_for_qso` | int | `2` | Minimum number of unique talkers required to declare a QSO is in progress |
| `notify_individual_talks` | boolean | `true` | Send notifications when individual stations start talking |

## Creating a Discord Webhook

1. Open Discord and navigate to the channel where you want notifications
2. Click the gear icon (⚙️) next to the channel name to open Channel Settings
3. Select "Integrations" from the left sidebar
4. Click "Create Webhook" or "View Webhooks" if you already have webhooks
5. Click "New Webhook" if needed
6. Give your webhook a name (e.g., "Allstar Node Activity")
7. Optionally set an avatar for the webhook
8. Click "Copy Webhook URL"
9. Paste the URL into your `config.yaml` as the `webhook_url` value

## Notification Messages

The Discord notifier tracks node activity through three states:

### State: Idle
The node has no recent activity. This is the initial state.

### State: Active
At least one station is talking, but not enough for a QSO.

**Example notification when transitioning from Idle to Active:**
```
Node 594950 has activity!
```

### State: QSO (Conversation in Progress)
Two or more unique stations are talking.

**Example notification when transitioning from Active to QSO:**
```
A QSO has started!
```

### Individual Talker Notifications

When `notify_individual_talks` is enabled, you'll receive notifications when individual stations start talking:

```
KF8S (550465) is now talking.
```

```
KE8VSI (55532) is now talking.
```

```
W8EAP (53534) is now talking.
```

### Returning to Idle

**When a QSO ends (no activity for `qso_inactive_seconds`):**
```
A QSO has ended!
```

**When the node becomes idle (no activity for `node_idle_seconds`):**
```
Node 594950 is now idle.
```

## Activity State Machine

The notifier uses a state machine to track node activity:

```
┌──────┐
│ Idle │ ◄────────┐
└──┬───┘          │
   │              │
   │ First        │ No activity
   │ talker       │ for node_idle_seconds
   ▼              │
┌────────┐        │
│ Active │────────┤
└───┬────┘        │
    │             │
    │ 2+ unique   │
    │ talkers     │
    ▼             │
┌─────┐           │
│ QSO │───────────┘
└─────┘
```

**State Transitions:**
- **Idle → Active**: First talker becomes active
- **Active → QSO**: Second unique talker joins (reaches `min_talkers_for_qso`)
- **QSO → Idle**: All talkers become inactive for `node_idle_seconds`
- **Active → Idle**: Single talker becomes inactive for `node_idle_seconds`
- **QSO → Active**: Number of active talkers drops below `min_talkers_for_qso` (but QSO hasn't ended yet)

## Timing Details

### QSO Inactive Timeout (`qso_inactive_seconds`)
- **Purpose**: Determines when individual talkers are considered inactive
- **Default**: 120 seconds (2 minutes)
- **Behavior**: A talker is removed from the active talker list if they haven't transmitted in this time period

### Node Idle Timeout (`node_idle_seconds`)
- **Purpose**: Determines when the entire node is considered idle
- **Default**: 300 seconds (5 minutes)
- **Behavior**: The node transitions to idle state if there's been no activity for this time period

**Example timeline:**
```
00:00 - KF8S starts talking → "Node has activity!", "KF8S is now talking."
00:30 - KE8VSI starts talking → "KE8VSI is now talking.", "A QSO has started!"
02:00 - Both stations stop talking
04:00 - QSO inactive timeout reached → "A QSO has ended!" (no individual active for 2 min)
07:00 - Node idle timeout reached → "Node is now idle." (no activity for 5 min from 02:00)
```

## Troubleshooting

### No notifications are being sent

1. **Check that Discord webhooks are enabled:**
   ```yaml
   discord:
     enabled: true
   ```

2. **Verify your webhook URL is correct** - it should look like:
   ```
   https://discord.com/api/webhooks/123456789/abcdefghijklmnopqrstuvwxyz
   ```

3. **Check application logs** - look for messages starting with `[DISCORD]`:
   ```
   [DISCORD] notification sent: Node 594950 has activity!
   [DISCORD] webhook URL not configured, skipping: ...
   [DISCORD] webhook returned status 404
   ```

4. **Test your webhook URL** using curl:
   ```bash
   curl -X POST "YOUR_WEBHOOK_URL" \
     -H "Content-Type: application/json" \
     -d '{"content": "Test message from Allstar Nexus"}'
   ```

### Too many or too few notifications

Adjust the timing thresholds:

- **Too many notifications**: Increase `qso_inactive_seconds` and/or `node_idle_seconds`
- **Too few notifications**: Decrease `qso_inactive_seconds` and/or `node_idle_seconds`
- **Too chatty about individual talkers**: Set `notify_individual_talks: false`

### Webhook rate limiting

Discord webhooks have rate limits:
- Per-webhook: 30 requests per minute
- Per-channel: 5 webhooks per 2 seconds

If you're hitting rate limits, the Discord notifier will log warnings. Consider:
- Increasing timeout values to reduce notification frequency
- Disabling individual talker notifications
- Using a dedicated channel for notifications

## Security Notes

- **Keep your webhook URL secret** - anyone with the URL can post to your channel
- Store webhook URLs in environment variables for production deployments:
  ```bash
  export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
  ```
- If your webhook URL is compromised, delete it in Discord and create a new one

## Gamification Notifications

If you have the [gamification system](GAMIFICATION.md) enabled, Discord notifications can also be sent when users level up, gain renown, or reach new rank groupings (like "Technician" → "General").

### Single Webhook Setup (Recommended)

**By default**, if you have `discord.enabled: true` and `discord.webhook_url` configured, gamification notifications will automatically use the same webhook. You don't need any additional configuration!

```yaml
# Node activity + Gamification (single webhook)
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
  # ... other discord settings

gamification:
  enabled: true
  # No discord section needed - will use main webhook above
```

### Separate Webhook Setup (Optional)

If you want gamification notifications in a different channel, you can configure a separate webhook:

```yaml
# Node activity notifications
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/ACTIVITY_WEBHOOK"

# Gamification notifications (optional - separate webhook)
gamification:
  enabled: true
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/GAMIFICATION_WEBHOOK"
```

### Gamification Notification Types

When enabled, you'll receive notifications for:

1. **Level Up**: "🎉 **AD8OD** has achieved level **20**!"
2. **Rank Change**: "🎖️ **AD8OD** has reached new rank of **General**!" (when crossing level group boundaries like 10→20, 20→30, etc.)
3. **Renown Gained**: "🌟 **AD8OD** has achieved **Renown Level 1**!" (when reaching level 60 and beyond)

**Note**: Rank changes occur when crossing into these level groups:
- Level 10-19: Technician (notification at level 10)
- Level 20-29: General (notification at level 20)
- Level 30-39: Advanced (notification at level 30)
- Level 40-49: Extra (notification at level 40)
- Level 50-55: Elmer (notification at level 50)
- Level 56-60: Professor (notification at level 56)

For example, reaching level 20 from level 19 will trigger: "🎖️ **AD8OD** has reached new rank of **General**!"

## Example Configuration

Here's a complete example configuration:

```yaml
# Minimal notifications - only QSO start/end and node idle
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
  qso_inactive_seconds: 180
  node_idle_seconds: 600
  min_talkers_for_qso: 2
  notify_individual_talks: false

# Verbose notifications - every talker and quick timeouts
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
  qso_inactive_seconds: 60
  node_idle_seconds: 180
  min_talkers_for_qso: 2
  notify_individual_talks: true
```
