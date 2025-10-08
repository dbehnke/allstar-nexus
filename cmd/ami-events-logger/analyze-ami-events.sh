#!/bin/bash
# analyze-ami-events.sh - Quick analysis of captured AMI events

set -e

# Check if file is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <ami-events.jsonl>"
    echo ""
    echo "Quick analysis tool for AMI events captured by ami-events-logger"
    echo ""
    echo "Examples:"
    echo "  $0 ami-events.jsonl"
    echo "  $0 my-capture.jsonl"
    exit 1
fi

FILE="$1"

if [ ! -f "$FILE" ]; then
    echo "Error: File '$FILE' not found"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: 'jq' is required but not installed"
    echo "Install with: brew install jq  (macOS) or apt-get install jq (Linux)"
    exit 1
fi

echo "=========================================="
echo "AMI Events Analysis: $FILE"
echo "=========================================="
echo ""

# Total counts
echo "📊 Message Counts"
echo "------------------"
TOTAL=$(wc -l < "$FILE" | tr -d ' ')
EVENTS=$(jq -r 'select(.type == "EVENT") | .type' "$FILE" | wc -l | tr -d ' ')
RESPONSES=$(jq -r 'select(.type == "RESPONSE") | .type' "$FILE" | wc -l | tr -d ' ')
echo "Total messages:  $TOTAL"
echo "Events:          $EVENTS"
echo "Responses:       $RESPONSES"
echo ""

# Event types
echo "📋 Event Types (Top 20)"
echo "------------------------"
jq -r 'select(.type == "EVENT") | .event' "$FILE" | sort | uniq -c | sort -rn | head -20
echo ""

# Time range
echo "⏱️  Time Range"
echo "-------------"
FIRST=$(jq -r '.timestamp' "$FILE" | head -1)
LAST=$(jq -r '.timestamp' "$FILE" | tail -1)
echo "First event: $FIRST"
echo "Last event:  $LAST"
echo ""

# Common events of interest
echo "🔍 Events of Interest"
echo "----------------------"

# Linking events
LINK_EVENTS=$(jq -s '[.[] | select(.type == "EVENT") | select(.event | test("LINK|ALINK"; "i"))] | length' "$FILE")
echo "Linking events (RPT_LINKS/RPT_ALINKS): $LINK_EVENTS"

# Keying events
KEY_EVENTS=$(jq -s '[.[] | select(.type == "EVENT") | select(.event | test("KEY"; "i"))] | length' "$FILE")
echo "Keying events (RPT_TXKEYED/RPT_RXKEYED): $KEY_EVENTS"

# Variable events
VAR_EVENTS=$(jq -s '[.[] | select(.type == "EVENT") | select(.headers.Variable != null)] | length' "$FILE")
echo "Events with Variable field: $VAR_EVENTS"

echo ""
echo "=========================================="
echo "💡 Tips"
echo "=========================================="
echo ""
echo "View all unique events:"
echo "  jq -r 'select(.type == \"EVENT\") | .event' $FILE | sort -u"
echo ""
echo "Filter specific event type:"
echo "  jq 'select(.event == \"RPT_ALINKS\")' $FILE"
echo ""
echo "View linking events:"
echo "  jq 'select(.event | test(\"LINK|ALINK\"))' $FILE"
echo ""
echo "View keying events:"
echo "  jq 'select(.event | test(\"KEY\"))' $FILE"
echo ""
echo "Pretty print first 5 events:"
echo "  jq 'select(.type == \"EVENT\")' $FILE | head -5 | jq -s '.'"
echo ""
