package api

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dbehnke/allstar-nexus/internal/core"
)

// NodeRecord represents a single AllStar node entry from astdb.txt
type NodeRecord struct {
	Node        int    `json:"node"`
	Callsign    string `json:"callsign"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
}

// NodeLookup handles searching for AllStar nodes by number or callsign.
// Endpoint: GET /api/node-lookup?q=<search_term>
func (a *API) NodeLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method_not_allowed", "only GET supported")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, 400, "bad_request", "query parameter 'q' is required")
		return
	}

	// Read and parse astdb.txt
	nodes, err := a.searchAstDB(query)
	if err != nil {
		writeError(w, 500, "file_error", "unable to read astdb.txt: "+err.Error())
		return
	}

	writeJSON(w, 200, map[string]any{
		"query":   query,
		"results": nodes,
		"count":   len(nodes),
	})
}

// searchAstDB reads the astdb.txt file and returns matching nodes.
// The astdb.txt file format is typically: node|callsign|description|location
// If query is not numeric and not found in database, returns the query as a callsign
func (a *API) searchAstDB(query string) ([]NodeRecord, error) {
	file, err := os.Open(a.AstDBPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []NodeRecord
	scanner := bufio.NewScanner(file)
	queryLower := strings.ToLower(query)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse pipe-delimited format: node|callsign|description|location
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		nodeStr := strings.TrimSpace(parts[0])
		callsign := strings.TrimSpace(parts[1])
		description := ""
		location := ""

		if len(parts) > 2 {
			description = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 {
			location = strings.TrimSpace(parts[3])
		}

		// Convert node to int
		nodeNum, err := strconv.Atoi(nodeStr)
		if err != nil {
			continue
		}

		// Match against node number or callsign
		if strings.Contains(nodeStr, query) ||
			strings.Contains(strings.ToLower(callsign), queryLower) ||
			strings.Contains(strings.ToLower(description), queryLower) ||
			strings.Contains(strings.ToLower(location), queryLower) {
			results = append(results, NodeRecord{
				Node:        nodeNum,
				Callsign:    callsign,
				Description: description,
				Location:    location,
			})
		}

		// Limit results to avoid overwhelming responses
		if len(results) >= 100 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// If no results found and query is not numeric, assume it's a callsign/text node ID
	// Return a synthetic record using the query as the callsign
	if len(results) == 0 {
		// Check if it's a negative number (hashed text node ID)
		if nodeID, err := strconv.Atoi(query); err != nil || nodeID < 0 {
			var callsign string
			var desc string

			if err == nil && nodeID < 0 {
				// This is a hashed text node ID - look up original name
				if name, found := core.GetTextNodeName(nodeID); found {
					callsign = name
					desc = "VOIP Node"
				} else {
					// Fallback if not in map
					callsign = strings.ToUpper(query)
					desc = "VOIP Node (hash)"
				}
				results = append(results, NodeRecord{
					Node:        nodeID,
					Callsign:    callsign,
					Description: desc,
					Location:    "",
				})
			} else {
				// Plain text query - return as callsign
				results = append(results, NodeRecord{
					Node:        0,
					Callsign:    strings.ToUpper(query),
					Description: "VOIP Node",
					Location:    "",
				})
			}
		}
	}

	return results, nil
}

// LookupNodeByID performs a fast lookup of a single node by ID.
// Returns nil if not found.
func (a *API) LookupNodeByID(nodeID int) *NodeRecord {
	file, err := os.Open(a.AstDBPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	targetStr := strconv.Itoa(nodeID)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse pipe-delimited format: node|callsign|description|location
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		nodeStr := strings.TrimSpace(parts[0])
		if nodeStr != targetStr {
			continue
		}

		callsign := strings.TrimSpace(parts[1])
		description := ""
		location := ""

		if len(parts) > 2 {
			description = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 {
			location = strings.TrimSpace(parts[3])
		}

		return &NodeRecord{
			Node:        nodeID,
			Callsign:    callsign,
			Description: description,
			Location:    location,
		}
	}

	return nil
}
