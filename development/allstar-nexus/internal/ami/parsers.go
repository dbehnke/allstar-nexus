package ami

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseXStat parses the response from RptStatus XStat command
func ParseXStat(node int, response string) (*XStatResult, error) {
	result := &XStatResult{
		Node:        node,
		Connections: make([]Connection, 0),
		LinkedNodes: make([]LinkedNode, 0),
		Variables:   make(map[string]string),
		Timestamp:   time.Now(),
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse Conn: lines
		// Format: Conn: NodeNum IP IsKeyed Direction Elapsed LinkType
		// EchoLink format: Conn: NodeNum IsKeyed Direction Elapsed
		if strings.HasPrefix(line, "Conn:") {
			conn, err := parseConnLine(line)
			if err != nil {
				// Log but don't fail - continue parsing
				continue
			}
			result.Connections = append(result.Connections, conn)
		}

		// Parse LinkedNodes: line
		// Format: LinkedNodes: T2000, R2001, C2002
		if strings.HasPrefix(line, "LinkedNodes:") {
			nodes := parseLinkedNodes(line)
			result.LinkedNodes = nodes
		}

		// Parse Var: lines
		// Format: Var: RPT_RXKEYED=1
		if strings.HasPrefix(line, "Var:") {
			key, value := parseVar(line)
			if key != "" {
				result.Variables[key] = value

				// Extract common variables
				if key == "RPT_RXKEYED" {
					result.RxKeyed = value == "1"
				} else if key == "RPT_TXKEYED" {
					result.TxKeyed = value == "1"
				}
			}
		}
	}

	return result, nil
}

// parseConnLine parses a Conn: line from XStat
func parseConnLine(line string) (Connection, error) {
	// Remove "Conn: " prefix
	line = strings.TrimPrefix(line, "Conn:")
	line = strings.TrimSpace(line)

	fields := strings.Fields(line)
	if len(fields) < 4 {
		return Connection{}, fmt.Errorf("invalid conn line: too few fields")
	}

	conn := Connection{
		Timestamp: time.Now(),
	}

	// Parse node number
	nodeNum, err := strconv.Atoi(fields[0])
	if err != nil {
		return Connection{}, fmt.Errorf("invalid node number: %w", err)
	}
	conn.Node = nodeNum

	// Detect EchoLink vs standard format
	// EchoLink nodes > 3000000 and don't have IP field
	isEchoLink := nodeNum > 3000000

	if isEchoLink {
		// Format: NodeNum IsKeyed Direction Elapsed [LinkType]
		if len(fields) >= 4 {
			conn.IsKeyed = fields[1] == "1"
			conn.Direction = fields[2]
			conn.Elapsed = fields[3]
			if len(fields) >= 5 {
				conn.LinkType = fields[4]
			}
		}
	} else {
		// Format: NodeNum IP IsKeyed Direction Elapsed [LinkType]
		if len(fields) >= 5 {
			conn.IP = fields[1]
			conn.IsKeyed = fields[2] == "1"
			conn.Direction = fields[3]
			conn.Elapsed = fields[4]
			if len(fields) >= 6 {
				conn.LinkType = fields[5]
			}
		}
	}

	return conn, nil
}

// parseLinkedNodes parses the LinkedNodes: line
func parseLinkedNodes(line string) []LinkedNode {
	// Remove "LinkedNodes: " prefix
	line = strings.TrimPrefix(line, "LinkedNodes:")
	line = strings.TrimSpace(line)

	if line == "" {
		return []LinkedNode{}
	}

	// Split by comma
	parts := strings.Split(line, ",")
	nodes := make([]LinkedNode, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 2 {
			continue
		}

		// First character is mode (T/R/C/M)
		mode := string(part[0])
		nodeStr := part[1:]

		nodeNum, err := strconv.Atoi(nodeStr)
		if err != nil {
			continue
		}

		nodes = append(nodes, LinkedNode{
			Node: nodeNum,
			Mode: mode,
		})
	}

	return nodes
}

// parseVar parses a Var: line
func parseVar(line string) (string, string) {
	// Remove "Var: " prefix
	line = strings.TrimPrefix(line, "Var:")
	line = strings.TrimSpace(line)

	// Split on '='
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", ""
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	return key, value
}

// ParseSawStat parses the response from RptStatus SawStat command
func ParseSawStat(node int, response string) (*SawStatResult, error) {
	result := &SawStatResult{
		Node:      node,
		Nodes:     make(map[int]*KeyingInfo),
		Timestamp: time.Now(),
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse Conn: lines
		// Format: Conn: NodeNum IsKeyed SecsSinceKeyed SecsSinceUnkeyed
		if strings.HasPrefix(line, "Conn:") {
			ki, err := parseSawStatLine(line)
			if err != nil {
				continue
			}
			result.Nodes[ki.Node] = ki
		}
	}

	return result, nil
}

// parseSawStatLine parses a Conn: line from SawStat
func parseSawStatLine(line string) (*KeyingInfo, error) {
	// Remove "Conn: " prefix
	line = strings.TrimPrefix(line, "Conn:")
	line = strings.TrimSpace(line)

	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil, fmt.Errorf("invalid sawstat line: too few fields")
	}

	ki := &KeyingInfo{}

	// Parse fields
	nodeNum, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("invalid node number: %w", err)
	}
	ki.Node = nodeNum

	ki.IsKeyed = fields[1] == "1"

	secsSinceKeyed, err := strconv.Atoi(fields[2])
	if err != nil {
		return nil, fmt.Errorf("invalid secs since keyed: %w", err)
	}
	ki.SecsSinceKeyed = secsSinceKeyed

	secsSinceUnkeyed, err := strconv.Atoi(fields[3])
	if err != nil {
		return nil, fmt.Errorf("invalid secs since unkeyed: %w", err)
	}
	ki.SecsSinceUnkeyed = secsSinceUnkeyed

	// Calculate timestamps
	now := time.Now()
	ki.LastKeyedTime = now.Add(-time.Duration(secsSinceKeyed) * time.Second)
	ki.LastUnkeyedTime = now.Add(-time.Duration(secsSinceUnkeyed) * time.Second)

	return ki, nil
}

// CombineXStatSawStat merges XStat and SawStat results
func CombineXStatSawStat(xstat *XStatResult, sawstat *SawStatResult) *CombinedNodeStatus {
	combined := &CombinedNodeStatus{
		Node:        xstat.Node,
		RxKeyed:     xstat.RxKeyed,
		TxKeyed:     xstat.TxKeyed,
		Connections: make([]ConnectionWithHistory, 0, len(xstat.Connections)),
		Timestamp:   time.Now(),
	}

	// Create mode lookup from LinkedNodes
	modes := make(map[int]string)
	for _, ln := range xstat.LinkedNodes {
		modes[ln.Node] = ln.Mode
	}

	// Merge connections with keying info
	for _, conn := range xstat.Connections {
		cwh := ConnectionWithHistory{
			Connection: conn,
		}

		// Add keying info if available
		if sawstat != nil {
			if ki, ok := sawstat.Nodes[conn.Node]; ok {
				cwh.KeyingInfo = ki
				cwh.LastHeard = FormatLastHeard(ki)
			}
		}

		// Add mode from LinkedNodes
		if mode, ok := modes[conn.Node]; ok {
			cwh.Mode = mode
		}

		combined.Connections = append(combined.Connections, cwh)
	}

	return combined
}

// ParseVoterOutput parses voter command output
func ParseVoterOutput(node int, response string) (*VoterResult, error) {
	result := &VoterResult{
		Node:      node,
		Receivers: make([]VoterReceiver, 0),
		Timestamp: time.Now(),
	}

	lines := strings.Split(response, "\n")
	inReceiverSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Look for header line
		if strings.Contains(line, "Receiver") && strings.Contains(line, "RSSI") {
			inReceiverSection = true
			continue
		}

		// Skip separator lines
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
			continue
		}

		// Parse receiver lines
		if inReceiverSection {
			rx := parseVoterLine(line)
			if rx.ID != "" {
				result.Receivers = append(result.Receivers, rx)
			}
		}
	}

	return result, nil
}

// parseVoterLine parses a single voter receiver line
func parseVoterLine(line string) VoterReceiver {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return VoterReceiver{}
	}

	rx := VoterReceiver{
		ID: fields[0],
	}

	// Try to parse RSSI (second field)
	if len(fields) >= 2 {
		if rssi, err := strconv.Atoi(fields[1]); err == nil {
			rx.RSSI = rssi
		}
	}

	// Look for "YES" or "VOTED" in remaining fields
	for i := 2; i < len(fields); i++ {
		upper := strings.ToUpper(fields[i])
		if upper == "YES" || upper == "VOTED" {
			rx.Voted = true
		}
		if upper == "ACTIVE" || upper == "STANDBY" || upper == "INACTIVE" {
			rx.State = upper
		}
	}

	// If no state found, default to ACTIVE if RSSI > 0
	if rx.State == "" {
		if rx.RSSI > 0 {
			rx.State = "ACTIVE"
		} else {
			rx.State = "INACTIVE"
		}
	}

	return rx
}
