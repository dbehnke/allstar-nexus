# AMI Debug Tool

Standalone Go CLI for capturing AMI events and producing debug snapshots.

## Usage

```bash
# Basic - stream events, save snapshot on Ctrl+C
AMI_PASSWORD=allstar go run . --host 127.0.0.1:5038

# Capture for 30 seconds
AMI_PASSWORD=allstar go run . --host 127.0.0.1:5038 --duration 30s

# Filter events
AMI_PASSWORD=allstar go run . --host 127.0.0.1:5038 --filter "RPT_"

# Custom output file
AMI_PASSWORD=allstar go run . --host 127.0.0.1:5038 --output debug.log
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `127.0.0.1:5038` | AMI host:port |
| `--username` | `admin` | AMI username |
| `--duration` | `0` | Capture duration (0 = indefinite) |
| `--filter` | `""` | Regex filter for events |
| `--output` | auto | Snapshot output file |
| `--buffer-size` | `1000` | Ring buffer size |
| `--no-color` | `false` | Disable colors |

## Environment Variables

- `AMI_PASSWORD` - Required. AMI password.
- `AMI_HOST` - Optional. AMI host:port (overrides `--host` flag).

## Snapshot Format

The snapshot file includes:
- Connection metadata (host, duration, event count)
- Event summary sorted by frequency
- All captured raw frames with timestamps
