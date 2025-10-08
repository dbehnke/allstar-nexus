# Getting Started with AMI Events Logger

## What is it?

The AMI Events Logger is a CLI tool that connects to your AllStar node's AMI interface and logs all incoming events to a file. This helps you understand what events are generated during various activities, which is essential for planning dashboard features.

## Quick Start (3 steps)

### 1. Build the tool
```bash
make ami-events-logger
```

### 2. Ensure config.yaml is set up
The tool uses your existing `config.yaml`. Make sure AMI is enabled:
```yaml
ami_enabled: true
ami_host: 127.0.0.1
ami_port: 5038
ami_username: admin
ami_password: your-password
ami_events: "on"
```

### 3. Run and capture events
```bash
# Capture events for 10 minutes
./ami-events-logger -events-only -verbose -duration 10m
```

While it's running, perform various actions on your node:
- Link/unlink to other nodes
- Transmit audio
- Receive from linked nodes
- Use DTMF commands

## What You'll See

### During Capture (with -verbose)
```
2025-01-15 14:32:15.123] EVENT: RPT_ALINKS
  Node: 43732
  Variable: RPT_ALINKS=48412

[2025-01-15 14:32:16.456] EVENT: RPT_TXKEYED
  Node: 43732
  Channel: 48412
  Keyed: 1
```

### After Stopping (Ctrl+C)
```
Shutdown complete
Statistics:
  Duration:  10m23s
  Events:    142
  Responses: 28
  Unknown:   0
  Total:     170
Output saved to: ami-events.jsonl
```

## Analyze Your Capture

### Quick Analysis
```bash
./cmd/ami-events-logger/analyze-ami-events.sh ami-events.jsonl
```

This shows:
- Total message counts
- Top 20 event types
- Time range
- Counts of linking, keying, and variable events

### Manual Analysis with jq

See what events happened most often:
```bash
jq -r '.event' ami-events.jsonl | sort | uniq -c | sort -rn
```

View all linking events:
```bash
jq 'select(.event | test("LINK|ALINK"))' ami-events.jsonl
```

View all keying events:
```bash
jq 'select(.event | test("KEY"))' ami-events.jsonl
```

## Example Workflow

```bash
# 1. Start capturing events
./ami-events-logger -events-only -verbose -duration 15m -output baseline.jsonl

# 2. In your radio/node, perform these actions:
#    - Link to another node
#    - Transmit on your local input
#    - Listen to remote transmissions
#    - Unlink
#    - Use DTMF commands

# 3. After capture completes, analyze
./cmd/ami-events-logger/analyze-ami-events.sh baseline.jsonl

# 4. Dig deeper into specific events
jq 'select(.event == "RPT_ALINKS")' baseline.jsonl | jq -s '.'
jq 'select(.event == "RPT_TXKEYED")' baseline.jsonl | jq -s '.'

# 5. Plan your dashboard features based on the events you see!
```

## What to Look For

When analyzing your captured events, pay attention to:

1. **Linking Events** (RPT_LINKS, RPT_ALINKS)
   - When do they fire?
   - What information do they contain?
   - How can we use this for real-time link status?

2. **Keying Events** (RPT_TXKEYED, RPT_RXKEYED)  
   - How do local vs. remote keying differ?
   - What timing information is available?
   - Can we track who's talking?

3. **Variable Events**
   - What variables are exposed?
   - What state information can we extract?
   - Are there undocumented events?

4. **Event Frequency**
   - Which events fire most often?
   - Are there periodic events?
   - What's the event rate during heavy activity?

5. **Event Relationships**
   - Do events come in sequences?
   - What's the timing between related events?
   - Can we detect patterns?

## Next Steps

After capturing and analyzing events:

1. **Identify Key Events**: Determine which events are most useful for your dashboard
2. **Design Event Handlers**: Plan how to process each event type
3. **Update State Management**: Modify the StateManager to handle new events
4. **Test Event Processing**: Use captured events for testing
5. **Build Dashboard Features**: Implement UI components based on the events

## Documentation

- **Quick Reference**: `cmd/ami-events-logger/QUICKREF.md`
- **Full Documentation**: `cmd/ami-events-logger/README.md`
- **Tool Overview**: `AMI_EVENTS_LOGGER.md`

## Tips for Success

✅ **DO**:
- Start with short captures (5-10 minutes)
- Use `-events-only` to reduce noise
- Perform various actions while capturing
- Analyze events before building features
- Keep captures for future reference

❌ **DON'T**:
- Run for hours without analyzing first
- Capture without performing actions on your node
- Forget to enable `-verbose` if you want real-time feedback
- Ignore low-frequency events (they might be important)

## Need Help?

Check the full documentation in `cmd/ami-events-logger/README.md` for:
- All command-line flags
- Detailed output format
- Advanced jq examples
- Troubleshooting guide
- Common issues and solutions

Happy event capturing! 🎉
