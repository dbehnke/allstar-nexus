package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RPTStats handles retrieving Asterisk RPT statistics for a given node.
// Endpoint: GET /api/rpt-stats?node=<node_number>
// Requires authentication
func (a *API) RPTStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method_not_allowed", "only GET supported")
		return
	}

	nodeStr := strings.TrimSpace(r.URL.Query().Get("node"))
	if nodeStr == "" {
		writeError(w, 400, "bad_request", "query parameter 'node' is required")
		return
	}

	// Check if AMI is enabled
	if a.AMIConnector == nil {
		writeError(w, 503, "service_unavailable", "AMI is not enabled")
		return
	}

	// Execute rpt stats command via AMI
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	command := fmt.Sprintf("rpt stats %s", nodeStr)
	msg, err := a.AMIConnector.SendCommand(ctx, command)
	if err != nil {
		writeError(w, 500, "ami_error", "failed to execute AMI command: "+err.Error())
		return
	}

	// Parse the response
	// AMI Command responses typically have the output in the message field
	output := msg.Headers["Message"]
	if output == "" {
		// Some AMI versions may include output in raw lines
		output = strings.Join(msg.Raw, "\n")
	}

	// Parse the stats into structured format
	stats := parseRPTStats(output, nodeStr)

	writeJSON(w, 200, map[string]any{
		"node":       nodeStr,
		"stats":      stats,
		"raw_output": output,
	})
}

// parseRPTStats parses the raw rpt stats output into a structured format.
// The format varies by Asterisk/app_rpt version, so we'll keep it flexible.
func parseRPTStats(output, node string) map[string]any {
	stats := make(map[string]any)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Common patterns in rpt stats output:
		// "RPT Statistics for node <node>"
		// "Uptime: <time>"
		// "Total connections: <count>"
		// "Variable: Value" format

		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Normalize key names to lowercase with underscores
				key = strings.ToLower(key)
				key = strings.ReplaceAll(key, " ", "_")

				stats[key] = value
			}
		}
	}

	return stats
}
