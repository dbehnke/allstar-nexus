package ami

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
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
				switch key {
				case "RPT_RXKEYED":
					result.RxKeyed = value == "1"
				case "RPT_TXKEYED":
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

	// Parse node number (can be numeric or text-based callsign)
	nodeNum, err := strconv.Atoi(fields[0])
	isTextNode := false
	if err != nil {
		// Not a numeric node ID - treat as text node (callsign)
		// Hash to a negative integer for consistent handling
		callsign := strings.ToUpper(strings.TrimSpace(fields[0]))
		nodeNum = hashTextNodeToInt(callsign)
		registerTextNodeInAMI(nodeNum, callsign)
		log.Printf("[AMI] Parsed text node from Conn line: %s -> %d", callsign, nodeNum)
		isTextNode = true
	}
	conn.Node = nodeNum

	// Detect format:
	// - Standard: NodeNum IP IsKeyed Direction Elapsed [LinkType]
	// - EchoLink: NodeNum IsKeyed Direction Elapsed [LinkType] (no IP, node > 3000000)
	// - Text node with IP: Callsign IP IsKeyed Direction Elapsed [LinkType]
	// - Text node without IP: Callsign IsKeyed Direction Elapsed [LinkType]
	isEchoLink := nodeNum > 3000000

	if isEchoLink || (isTextNode && len(fields) < 5) {
		// Format without IP: NodeNum IsKeyed Direction Elapsed [LinkType]
		if len(fields) >= 4 {
			conn.IsKeyed = fields[1] == "1"
			conn.Direction = fields[2]
			conn.Elapsed = fields[3]
			if len(fields) >= 5 {
				conn.LinkType = fields[4]
			}
		}
	} else {
		// Format with IP: NodeNum IP IsKeyed Direction Elapsed [LinkType]
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

		// First character is mode (T/R/C/M) if followed by digits.
		// If it's a callsign (starts with letter but not followed only by digits),
		// we should treat the whole thing as a node.
		mode := ""
		nodeStr := part
		firstChar := part[0]
		isMode := firstChar == 'T' || firstChar == 'R' || firstChar == 'C' || firstChar == 'M' || firstChar == 'P' || firstChar == 'L'

		if isMode && len(part) > 1 {
			// Check if it's followed by digits (standard numeric node)
			rest := part[1:]
			if _, err := strconv.Atoi(rest); err == nil {
				mode = string(firstChar)
				nodeStr = rest
			} else {
				// It's a text node (callsign) with a mode prefix
				// We still want to extract the mode but keep the callsign intact
				mode = string(firstChar)
				nodeStr = rest
			}
		}

		nodeNum, err := strconv.Atoi(nodeStr)
		if err != nil {
			// Not a numeric node - it's a text node (callsign)
			// Hash it to a negative integer and register the mapping
			// This is done in the AMI layer so CombinedNodeStatus can include these nodes
			callsign := strings.ToUpper(strings.TrimSpace(nodeStr))
			nodeNum = hashTextNodeToInt(callsign)
			registerTextNodeInAMI(nodeNum, callsign)
			log.Printf("[AMI] Registered text node: %s -> %d", callsign, nodeNum)
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

	// Track which nodes we've seen in Connections to identify LinkedNodes without Connections
	seenNodes := make(map[int]bool)

	// Merge connections with keying info
	for _, conn := range xstat.Connections {
		seenNodes[conn.Node] = true

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

	// NOTE: We intentionally do NOT create synthetic connections for LinkedNodes
	// entries that don't have a corresponding Connection entry. LinkedNodes shows
	// the full network topology (including indirect connections through the
	// spanning tree), while Conn: lines show only direct connections. Adding
	// LinkedNodes entries as connections would incorrectly show indirect nodes
	// (including text nodes like VOIP clients) as direct connections.
	//
	// Text nodes (callsigns like KF8S) that are direct connections would ideally
	// appear in Conn: lines. parseConnLine now handles text nodes properly
	// (hashes them to stable negative IDs), so if AllStarLink outputs them in
	// Conn: lines, they will be captured as direct connections.

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

// hashTextNodeToInt converts a text node (callsign) to a stable negative integer
// Uses FNV-1a hash to ensure consistent hashing
func hashTextNodeToInt(s string) int {
	s = strings.ToUpper(s) // Normalize to uppercase
	hash := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	// Convert to negative number to distinguish from numeric AllStar nodes
	// Use lower 30 bits to keep values reasonable
	return -int(hash & 0x3FFFFFFF)
}

// AMI-layer text node registry
// This allows the AMI layer to register text nodes when parsing LinkedNodes
// The core layer will also register them when processing RPT_ALINKS
var (
	amiTextNodeRegistry = make(map[int]string)
	amiRegistryMu       sync.RWMutex
)

func registerTextNodeInAMI(nodeID int, text string) {
	amiRegistryMu.Lock()
	defer amiRegistryMu.Unlock()

	upperText := strings.ToUpper(text)

	// Check for hash collisions
	if existing, exists := amiTextNodeRegistry[nodeID]; exists && existing != upperText {
		log.Printf("WARNING: Hash collision detected! Callsigns %s and %s both hash to %d",
			existing, upperText, nodeID)
	}

	amiTextNodeRegistry[nodeID] = upperText
}

// GetTextNodeFromAMI retrieves a text node name from the AMI-layer registry
func GetTextNodeFromAMI(nodeID int) (string, bool) {
	amiRegistryMu.RLock()
	defer amiRegistryMu.RUnlock()
	text, ok := amiTextNodeRegistry[nodeID]
	return text, ok
}

// ParseLStats parses the response from "rpt lstats <node>" command.
// Expected format:
//
//	NODE      PEER                RECONNECTS  DIRECTION  CONNECT TIME        CONNECT STATE
//	----      ----                ----------  ---------  ------------        -------------
//	58840     100.89.118.58       0           IN         20:38:42:43         ESTABLISHED
func ParseLStats(node int, response string) (*LStatsResult, error) {
	result := &LStatsResult{
		Node: node,
	}

	lines := strings.Split(response, "\n")
	var headerParsed bool
	var colPositions []int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip separator lines (all dashes)
		if strings.HasPrefix(line, "----") || strings.HasPrefix(line, "====") {
			continue
		}

		// Parse header line to determine column positions
		if !headerParsed {
			if strings.Contains(line, "NODE") && strings.Contains(line, "PEER") {
				colPositions = findLStatsColumnPositions(line)
				headerParsed = true
				continue
			}
		}

		if !headerParsed || len(colPositions) < 5 {
			continue
		}

		entry, err := parseLStatsLine(line, colPositions)
		if err != nil {
			continue // Skip malformed lines
		}
		result.Entries = append(result.Entries, entry)
	}

	return result, nil
}

// findLStatsColumnPositions finds the starting positions of each column in the header line
func findLStatsColumnPositions(header string) []int {
	positions := []int{}
	words := []string{"NODE", "PEER", "RECONNECTS", "DIRECTION", "CONNECT TIME", "CONNECT STATE"}
	idx := 0
	for _, word := range words {
		pos := strings.Index(header[idx:], word)
		if pos >= 0 {
			positions = append(positions, idx+pos)
			idx += pos
		}
	}
	return positions
}

// parseLStatsLine parses a single data line using column positions from the header
func parseLStatsLine(line string, positions []int) (LStatsEntry, error) {
	entry := LStatsEntry{}

	// Extract each field using column positions
	getField := func(start, end int) string {
		if start >= len(line) {
			return ""
		}
		if end > len(line) {
			end = len(line)
		}
		return strings.TrimSpace(line[start:end])
	}

	var fields []string
	for i := 0; i < len(positions); i++ {
		start := positions[i]
		end := len(line)
		if i+1 < len(positions) {
			end = positions[i+1]
		}
		fields = append(fields, getField(start, end))
	}

	if len(fields) < 5 {
		return entry, fmt.Errorf("not enough fields")
	}

	// Parse node number
	nodeNum, err := strconv.Atoi(fields[0])
	if err != nil {
		return entry, fmt.Errorf("invalid node number: %s", fields[0])
	}
	entry.Node = nodeNum

	// Peer (IP/hostname)
	entry.Peer = fields[1]

	// Reconnects
	if len(fields) > 2 {
		entry.Reconnects, _ = strconv.Atoi(fields[2])
	}

	// Direction
	if len(fields) > 3 {
		entry.Direction = fields[3]
	}

	// Connect time
	if len(fields) > 4 {
		entry.ConnectTime = fields[4]
	}

	// Connect state
	if len(fields) > 5 {
		entry.ConnectState = fields[5]
	}

	return entry, nil
}
