# AMI Events Logger - Quick Start

This tool captures AMI events from your AllStar node to a log file, making it easy to analyze events and plan dashboard features.

## Quick Start

```bash
# Build the tool
make ami-events-logger

# Run with defaults (uses config.yaml)
./ami-events-logger

# Capture for 10 minutes with verbose output
./ami-events-logger -verbose -duration 10m

# Capture only events (no responses)
./ami-events-logger -events-only -output my-capture.jsonl
```

## What Gets Captured

The tool logs all AMI messages in JSON Lines format:
- **Events**: AMI events like RPT_ALINKS, RPT_TXKEYED, etc.
- **Responses**: Command responses from AMI
- **Timestamps**: When each message was received
- **Headers**: All key-value pairs from the AMI message

## Example Output

```json
{"timestamp":"2025-01-15T14:32:15.123Z","type":"EVENT","event":"RPT_ALINKS","headers":{"Event":"RPT_ALINKS","Node":"43732","Variable":"RPT_ALINKS=48412"}}
{"timestamp":"2025-01-15T14:32:16.456Z","type":"EVENT","event":"RPT_TXKEYED","headers":{"Event":"RPT_TXKEYED","Node":"43732","Channel":"48412","Keyed":"1"}}
```

## Common Use Cases

### 1. Capture baseline events
```bash
./ami-events-logger -events-only -duration 30m -output baseline.jsonl
```

### 2. Debug specific events
```bash
# Capture with verbose output
./ami-events-logger -verbose -events-only

# In another terminal, analyze as they come in
tail -f ami-events.jsonl | jq '.'
```

### 3. Analyze event patterns
```bash
# Count events by type
jq -r '.event' ami-events.jsonl | sort | uniq -c | sort -rn

# View all keying events
jq 'select(.event | contains("KEY"))' ami-events.jsonl
```

## Configuration

Uses your existing `config.yaml`. Ensure AMI is enabled:

```yaml
ami_enabled: true
ami_host: 127.0.0.1
ami_port: 5038
ami_username: admin
ami_password: your-password
ami_events: "on"
```

## Full Documentation

See [cmd/ami-events-logger/README.md](cmd/ami-events-logger/README.md) for complete documentation, including:
- All command-line flags
- Output format details
- Analysis examples with `jq`
- Troubleshooting tips
- Example workflows
