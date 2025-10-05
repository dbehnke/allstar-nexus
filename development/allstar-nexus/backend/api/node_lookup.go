package api

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	return results, nil
}
