# CGNAT Whitelist Generator

A standalone utility to generate AllStar node whitelist entries for CGNAT (Carrier-Grade NAT) forwarding scenarios.

## Problem

When an AllStar hub is behind CGNAT, you cannot determine the caller's external IP address. This requires creating manual whitelist entries for nodes that connect through specific IP addresses. This tool automates the process of generating these whitelist entries.

## Features

- ✅ Reads callsigns from a text file (one per line)
- ✅ Looks up all nodes associated with each callsign from the AllStar database
- ✅ Generates properly formatted whitelist entries
- ✅ Uses independent SQLite database (doesn't interfere with main application)
- ✅ Automatically downloads and imports astdb.txt if database is empty
- ✅ Groups entries by callsign with comment headers
- ✅ Handles missing callsigns gracefully

## Building

```bash
cd tools/cgnat-whitelist
go build -o cgnat-whitelist .
```

## Usage

### Basic Usage

```bash
./cgnat-whitelist -f callsigns.txt -o whitelist.txt -i 100.89.118.58
```

### Command-Line Options

| Flag | Description | Required | Default |
|------|-------------|----------|---------|
| `-f` | Path to callsigns file (one callsign per line) | Yes | - |
| `-o` | Path to output whitelist file | Yes | - |
| `-i` | IP address for the whitelist entries | Yes | - |
| `-db` | Path to SQLite database | No | `data/cgnat-whitelist.db` |
| `-astdb-url` | URL to download astdb from | No | `http://allmondb.allstarlink.org/` |

## Input File Format

Create a text file with one callsign per line:

```
# Sample callsigns.txt
KF8S
KE8VSI
W1ABC
```

Empty lines and lines starting with `#` or `;` are ignored.

## Output File Format

The tool generates a whitelist file in the format required by AllStar:

```
;KF8S Nodes
550460 = radio@100.89.118.58/550460,NONE
550461 = radio@100.89.118.58/550461,NONE
550462 = radio@100.89.118.58/550462,NONE
550463 = radio@100.89.118.58/550463,NONE
;KE8VSI Nodes
58840  = radio@100.89.118.58/58840,NONE
```

## Examples

### Example 1: Generate whitelist for Tailscale IP

```bash
./cgnat-whitelist -f my-nodes.txt -o rpt.conf.whitelist -i 100.89.118.58
```

### Example 2: Use custom database location

```bash
./cgnat-whitelist -f callsigns.txt -o whitelist.txt -i 10.0.0.1 -db /var/lib/cgnat/db.sqlite
```

### Example 3: First run (downloads astdb automatically)

On the first run, if the database is empty, the tool will automatically download and import the AllStar node database:

```bash
$ ./cgnat-whitelist -f callsigns.txt -o whitelist.txt -i 100.89.118.58
{"level":"info","msg":"Database is empty or needs initialization. Downloading astdb..."}
{"level":"info","msg":"downloading astdb from AllStar server"}
{"level":"info","msg":"astdb imported successfully","node_count":50000}
{"level":"info","msg":"Whitelist generation completed"}

Whitelist generation completed successfully!
  Callsigns processed: 2
  Total entries: 5
  Output file: whitelist.txt
```

## How It Works

1. **Database Initialization**: On first run, downloads astdb.txt from AllStarLink and imports it into a local SQLite database
2. **Callsign Lookup**: Reads callsigns from the input file and queries the database for all nodes associated with each callsign
3. **Whitelist Generation**: Formats each node as a whitelist entry with the specified IP address
4. **Output**: Writes the formatted entries to the output file, grouped by callsign

## Database Management

The tool uses an independent SQLite database (default: `data/cgnat-whitelist.db`) to store node information. This database:

- Is separate from the main Allstar Nexus database
- Will be automatically created and populated on first run
- Can be reused for multiple whitelist generations
- Can be safely deleted to force a fresh download

## Notes

- Callsigns are automatically converted to uppercase
- The tool handles missing callsigns gracefully (writes a comment indicating no nodes were found)
- Node IDs are left-padded to 6 characters for consistent formatting
- The database is reusable across multiple runs - it won't re-download unless empty

## Troubleshooting

### Database Won't Download

If the astdb download fails (network issues, blocked domain, etc.), you can:

1. Manually download astdb.txt from http://allmondb.allstarlink.org/
2. Import it using the main Allstar Nexus application
3. Copy the database file to the cgnat-whitelist location

### Empty Output

If the output file is empty or has no entries:

- Verify the callsigns in your input file are correct
- Check that the database was populated (check log messages)
- Try running with a known callsign to test

### Permission Errors

Ensure you have write permissions to:
- The output file location
- The database path (default: `data/` directory)

## Integration with AllStar

To use the generated whitelist:

1. Generate the whitelist file using this tool
2. Copy the contents to your AllStar configuration file (typically `/etc/asterisk/rpt.conf`)
3. Reload your AllStar configuration
4. Test connectivity from the whitelisted nodes

## Related Documentation

- Main project: [Allstar Nexus](../../README.md)
- AllStar node database: [ASTDB_SOLUTION.md](../../ASTDB_SOLUTION.md)
