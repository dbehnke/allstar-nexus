# Discord Webhook Quick Start Guide

This is a quick guide to get Discord webhook notifications working in 5 minutes.

## Step 1: Get Your Discord Webhook URL

1. Open Discord and go to the channel where you want notifications
2. Click the ⚙️ (gear icon) next to the channel name
3. Select "Integrations" → "Webhooks" → "New Webhook"
4. Copy the webhook URL

## Step 2: Add to Your config.yaml

```yaml
# ... other config ...

discord:
  enabled: true
  webhook_url: "YOUR_WEBHOOK_URL_HERE"
  qso_inactive_seconds: 120      # Optional: defaults to 120 (2 minutes)
  node_idle_seconds: 300         # Optional: defaults to 300 (5 minutes)
  min_talkers_for_qso: 2         # Optional: defaults to 2
  notify_individual_talks: true  # Optional: defaults to true
```

## Step 3: Restart Allstar Nexus

```bash
./allstar-nexus --config ./config.yaml
```

## Step 4: Look for Startup Message

You should see a log line like:

```
Discord webhook notifier started	{"node_id": 594950, "node_name": "594950", "qso_inactive_seconds": 120, "node_idle_seconds": 300}
```

## That's it! 🎉

Your Discord channel will now receive notifications when:
- Your node becomes active
- Individual stations start talking
- A QSO (conversation) starts
- A QSO ends
- Your node goes idle

## Example Notifications

When someone keys up on your node:

```
Node 594950 has activity!
KF8S (550465) is now talking.
```

When a second person joins:

```
KE8VSI (55532) is now talking.
A QSO has started!
```

When things quiet down:

```
A QSO has ended!
Node 594950 is now idle.
```

## Troubleshooting

**No notifications appearing?**

1. Check the webhook URL is correct
2. Look for errors in the logs: `grep DISCORD your.log`
3. Test the webhook manually:
   ```bash
   curl -X POST "YOUR_WEBHOOK_URL" \
     -H "Content-Type: application/json" \
     -d '{"content": "Test from Allstar Nexus"}'
   ```

**Too many notifications?**

Set `notify_individual_talks: false` to only get QSO start/end and node idle notifications.

**Not enough notifications?**

Lower the timeout values:
```yaml
qso_inactive_seconds: 60   # 1 minute
node_idle_seconds: 180     # 3 minutes
```

## Need More Details?

See [DISCORD.md](DISCORD.md) for the complete documentation.
