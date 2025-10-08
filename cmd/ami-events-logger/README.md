# AMI Events Logger

A CLI tool to capture and log AMI (Asterisk Manager Interface) events from your AllStar node for analysis and dashboard planning.

## Overview

This tool connects to your AMI interface using the same `config.yaml` configuration as the main Allstar Nexus application and logs all incoming AMI events to a JSON Lines (JSONL) file. This is useful for:

- Understanding what events are generated during various activities
- Planning event-driven dashboard features
- Debugging AMI connectivity and event handling
- Analyzing event patterns and frequencies

## Building

```bash
# Build the CLI tool
make ami-events-logger

# Or build directly with go
go build -o ami-events-logger cmd/ami-events-logger/main.go
```

## Usage

### Basic Usage

```bash
# Use default config.yaml location and capture to ami-events.jsonl
./ami-events-logger

# Specify a custom config file
./ami-events-logger -config /path/to/config.yaml

# Specify output file
./ami-events-logger -output my-events.jsonl
```

### Advanced Options

```bash
# Capture only events (skip AMI responses)
./ami-events-logger -events-only

# Include raw message lines in output
./ami-events-logger -raw

# Print events to stdout as well as file (verbose mode)
./ami-events-logger -verbose

# Run for a specific duration then stop
./ami-events-logger -duration 1h
./ami-events-logger -duration 30m

# Combine options
./ami-events-logger -events-only -verbose -output test.jsonl -duration 5m
```

### Stopping the Logger

Press `Ctrl+C` to stop logging. The tool will display statistics before exiting:

```
Received interrupt signal, stopping...
Shutdown complete
Statistics:
  Duration:  5m23s
  Events:    142
  Responses: 28
  Unknown:   0
  Total:     170
Output saved to: ami-events.jsonl
```

## Output Format

Events are logged in JSON Lines format (one JSON object per line), making it easy to process with standard tools like `jq`, `grep`, or custom scripts.

### Example Log Entry

```json
{
  "timestamp": "2025-01-15T14:32:15.123Z",
  "type": "EVENT",
  "event": "RPT_ALINKS",
  "headers": {
    "Event": "RPT_ALINKS",
    "Privilege": "reporting,all",
    "Node": "43732",
    "Variable": "RPT_ALINKS=48412"
  }
}
```

### Fields

- `timestamp`: ISO 8601 timestamp when the event was received
- `type`: Message type (`EVENT`, `RESPONSE`, or `UNKNOWN`)
- `event`: Event name (only present for EVENT types)
- `headers`: Map of all AMI headers (key-value pairs)
- `raw`: Raw message lines (only included with `-raw` flag)

## Analyzing Captured Events

### Quick Analysis Script

A helper script is provided for quick event analysis:

```bash
# Analyze captured events
./cmd/ami-events-logger/analyze-ami-events.sh ami-events.jsonl
```

This will show:
- Total message counts (events, responses)
- Top 20 event types by frequency
- Time range of capture
- Count of linking, keying, and variable events
- Helpful jq commands for further analysis

### Manual Analysis with jq

#### Count events by type

```bash
jq -r '.event' ami-events.jsonl | sort | uniq -c | sort -rn
```

### View all unique event types

```bash
jq -r 'select(.type == "EVENT") | .event' ami-events.jsonl | sort -u
```

### Filter specific events

```bash
# Show only RPT_ALINKS events
jq 'select(.event == "RPT_ALINKS")' ami-events.jsonl

# Show all keying-related events
jq 'select(.event | contains("KEY"))' ami-events.jsonl
```

### Pretty print an event

```bash
jq '.' ami-events.jsonl | head -20
```

### Extract events with specific headers

```bash
# Find all events for node 43732
jq 'select(.headers.Node == "43732")' ami-events.jsonl

# Find events with Variable field
jq 'select(.headers.Variable != null)' ami-events.jsonl
```

## Configuration

The tool uses your existing `config.yaml` file. Ensure these settings are configured:

```yaml
# AMI Configuration
ami_enabled: true
ami_host: 127.0.0.1
ami_port: 5038
ami_username: admin
ami_password: your-password
ami_events: "on"
ami_retry_interval: 15s
ami_retry_max: 60s
```

## Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | (auto-detect) | Path to config.yaml file |
| `-output` | `ami-events.jsonl` | Output file path |
| `-events-only` | `false` | Log only events, skip responses |
| `-raw` | `false` | Include raw message lines in output |
| `-verbose` | `false` | Print events to stdout in addition to file |
| `-duration` | `0` (unlimited) | Stop after this duration (e.g., `1h`, `30m`, `5s`) |

## Tips

1. **Start with a short capture**: Use `-duration 5m` to capture a representative sample
2. **Use events-only**: Add `-events-only` to focus on actual events and reduce noise
3. **Enable verbose mode**: Use `-verbose` to see events in real-time while capturing
4. **Capture during activity**: Perform various actions on your node (link/unlink, transmit, etc.) to capture relevant events
5. **Analyze patterns**: Use the captured data to understand event timing and relationships

## Example Workflow

```bash
# 1. Capture events during 10 minutes of typical operation
./ami-events-logger -events-only -verbose -duration 10m -output baseline.jsonl

# 2. See what events we captured
jq -r '.event' baseline.jsonl | sort | uniq -c | sort -rn

# 3. Focus on linking events
jq 'select(.event | test("LINK|ALINK"))' baseline.jsonl | jq -s '.' > linking-events.json

# 4. Analyze keying events
jq 'select(.event | test("KEY"))' baseline.jsonl > keying-events.jsonl
```

## Troubleshooting

### "AMI is disabled in configuration"

Ensure `ami_enabled: true` in your config.yaml file.

### Connection failures

- Check that your AMI host and port are correct
- Verify your AMI username and password
- Ensure Asterisk AMI is running and accessible
- Check firewall settings

### No events captured

- Verify `ami_events: "on"` in config.yaml
- Check that your node is actually active and generating events
- Try the `-verbose` flag to see if any messages are being received

## See Also

- Main documentation: [README.md](../../README.md)
- AMI Commands Reference: [AMI_COMMANDS_REFERENCE.md](../../AMI_COMMANDS_REFERENCE.md)
- Event-driven refactor notes: [EVENT_DRIVEN_REFACTOR.md](../../EVENT_DRIVEN_REFACTOR.md)
