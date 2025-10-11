package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
)

// VoterReceiver represents a single receiver in the RTCM voter system
type VoterReceiver struct {
	Index      int     `json:"index"`
	Name       string  `json:"name"`
	Address    string  `json:"address,omitempty"`
	RSSI       float64 `json:"rssi"`
	Voted      bool    `json:"voted"`
	State      string  `json:"state,omitempty"`
	LastUpdate string  `json:"last_update,omitempty"`
}

// VoterStats handles retrieving RTCM voter receiver data for a given node.
// Endpoint: GET /api/voter-stats?node=<node_number>
// Requires authentication
func (a *API) VoterStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method_not_allowed", "only GET supported")
		return
	}

	// Determine if caller is admin to decide masking behavior
	isAdmin := false
	if u, status := a.currentUser(r); status == 200 {
		if u.Role == models.RoleAdmin || u.Role == models.RoleSuperAdmin {
			isAdmin = true
		}
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

	// Execute voter command via AMI
	// Common voter commands: "rpt fun <node> *980", "voter show <node>", or custom commands
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Try multiple command formats as different systems may use different commands
	commands := []string{
		fmt.Sprintf("rpt fun %s *980", nodeStr),      // AllStar voter display command
		fmt.Sprintf("voter show %s", nodeStr),        // Direct voter command
		fmt.Sprintf("rtcm show clients %s", nodeStr), // RTCM clients
	}

	var output string
	var lastErr error

	for _, command := range commands {
		msg, err := a.AMIConnector.SendCommand(ctx, command)
		if err != nil {
			lastErr = err
			continue
		}

		// Get output from AMI response
		output = msg.Headers["Message"]
		if output == "" {
			output = strings.Join(msg.Raw, "\n")
		}

		// If we got meaningful output, break
		if output != "" && !strings.Contains(strings.ToLower(output), "no such command") {
			break
		}
	}

	if output == "" && lastErr != nil {
		writeError(w, 500, "ami_error", "failed to execute AMI voter command: "+lastErr.Error())
		return
	}

	// Parse the voter stats
	receivers := parseVoterStats(output)
	if !isAdmin {
		// Mask receiver addresses for non-admins
		for i := range receivers {
			if receivers[i].Address != "" {
				receivers[i].Address = maskIPv4(receivers[i].Address)
			}
		}
	}

	writeJSON(w, 200, map[string]any{
		"node":       nodeStr,
		"receivers":  receivers,
		"count":      len(receivers),
		"raw_output": output,
	})
}

// parseVoterStats parses the raw voter output into structured receiver data.
// The format varies significantly between implementations, so this is a best-effort parser.
func parseVoterStats(output string) []VoterReceiver {
	var receivers []VoterReceiver
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Common voter output patterns:
		// "Receiver 1: RSSI=-85 dBm [VOTED]"
		// "Client 192.168.1.100: RSSI -92 dBm"
		// "1 WB6NIL-1 -85.0 *" (tabular format)

		// Try to extract receiver info
		receiver := VoterReceiver{Index: i + 1}

		// Check for voted indicator
		if strings.Contains(line, "[VOTED]") || strings.HasSuffix(line, "*") {
			receiver.Voted = true
		}

		// Extract RSSI value (look for patterns like -85, -85.0, etc.)
		rssi := extractRSSI(line)
		if rssi != 0 {
			receiver.RSSI = rssi
		}

		// Extract receiver name/identifier
		name := extractReceiverName(line)
		if name != "" {
			receiver.Name = name
		}

		// Extract IP address if present
		if addr := extractIPAddress(line); addr != "" {
			receiver.Address = addr
			if receiver.Name == "" {
				receiver.Name = addr
			}
		}

		// Determine state
		if receiver.Voted {
			receiver.State = "voted"
		} else if receiver.RSSI != 0 {
			receiver.State = "active"
		} else {
			receiver.State = "inactive"
		}

		// Only add if we extracted meaningful data
		if receiver.RSSI != 0 || receiver.Name != "" {
			receivers = append(receivers, receiver)
		}
	}

	return receivers
}

// extractRSSI attempts to find and parse an RSSI value from a line of text.
// Returns 0 if no valid RSSI is found.
func extractRSSI(line string) float64 {
	// Look for patterns like: -85, -85.0, RSSI=-85, RSSI: -85 dBm
	line = strings.ToLower(line)

	// Remove common keywords to isolate the number
	line = strings.ReplaceAll(line, "rssi", "")
	line = strings.ReplaceAll(line, "dbm", "")
	line = strings.ReplaceAll(line, "=", " ")
	line = strings.ReplaceAll(line, ":", " ")

	// Split into tokens and look for negative numbers
	tokens := strings.Fields(line)
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if strings.HasPrefix(token, "-") {
			if val, err := strconv.ParseFloat(token, 64); err == nil {
				// Sanity check: RSSI values typically range from -40 to -120 dBm
				if val >= -120 && val <= -30 {
					return val
				}
			}
		}
	}

	return 0
}

// extractReceiverName attempts to extract a receiver name/callsign from the line.
func extractReceiverName(line string) string {
	// Look for patterns like "Receiver 1", "WB6NIL-1", or client identifiers
	line = strings.TrimSpace(line)

	// If line starts with a word followed by number, that's likely the name
	tokens := strings.Fields(line)
	if len(tokens) > 0 {
		// Check if first token looks like a callsign or identifier
		first := tokens[0]
		// Callsign pattern: letters followed by numbers, possibly with dash
		if len(first) > 2 && hasLetters(first) && (hasNumbers(first) || strings.Contains(first, "-")) {
			return first
		}
	}

	return ""
}

// extractIPAddress attempts to find an IP address in the line.
func extractIPAddress(line string) string {
	tokens := strings.Fields(line)
	for _, token := range tokens {
		// Simple IP address check: xxx.xxx.xxx.xxx format
		if strings.Count(token, ".") == 3 {
			parts := strings.Split(token, ".")
			if len(parts) == 4 {
				valid := true
				for _, part := range parts {
					if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
						valid = false
						break
					}
				}
				if valid {
					return token
				}
			}
		}
	}
	return ""
}

// maskIPv4 masks last two octets of IPv4 address, leaves others unchanged
func maskIPv4(ip string) string {
	if ip == "" {
		return ip
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	return ip
}

// hasLetters checks if a string contains any letters.
func hasLetters(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

// hasNumbers checks if a string contains any numbers.
func hasNumbers(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
