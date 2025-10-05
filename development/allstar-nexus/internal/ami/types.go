package ami

import "time"

// Connection represents a connected node from XStat
type Connection struct {
	Node      int       // Remote node number
	IP        string    // IP address (empty for EchoLink)
	IsKeyed   bool      // Currently keyed (1=yes, 0=no)
	Direction string    // "IN" or "OUT"
	Elapsed   string    // Time connected (HH:MM:SS)
	LinkType  string    // "ESTABLISHED", "CONNECTING", etc.
	Timestamp time.Time // When this data was captured
}

// LinkedNode represents a node from LinkedNodes line
type LinkedNode struct {
	Node int    // Node number
	Mode string // "T"=Transceive, "R"=Receive, "C"=Connecting, "M"=Monitor
}

// XStatResult contains parsed XStat response
type XStatResult struct {
	Node        int            // Local node number
	Connections []Connection   // Connected nodes
	LinkedNodes []LinkedNode   // Linked nodes with modes
	RxKeyed     bool           // RPT_RXKEYED - Local receiver COS
	TxKeyed     bool           // RPT_TXKEYED - Local transmitter PTT
	Variables   map[string]string // All Var: fields
	Timestamp   time.Time      // When this data was captured
}

// KeyingInfo represents keying history from SawStat
type KeyingInfo struct {
	Node             int       // Node number
	IsKeyed          bool      // Currently keyed
	SecsSinceKeyed   int       // Seconds since last key-up (0 if currently keyed)
	SecsSinceUnkeyed int       // Seconds since last key-down
	LastKeyedTime    time.Time // Calculated timestamp of last key
	LastUnkeyedTime  time.Time // Calculated timestamp of last unkey
}

// SawStatResult contains parsed SawStat response
type SawStatResult struct {
	Node      int                    // Local node number
	Nodes     map[int]*KeyingInfo    // Keying info by remote node number
	Timestamp time.Time              // When this data was captured
}

// CombinedNodeStatus merges XStat and SawStat data
type CombinedNodeStatus struct {
	Node        int            // Local node number
	RxKeyed     bool           // Local receiver COS
	TxKeyed     bool           // Local transmitter PTT
	Connections []ConnectionWithHistory // Connections with keying history
	Timestamp   time.Time      // When this data was captured
}

// ConnectionWithHistory combines Connection and KeyingInfo
type ConnectionWithHistory struct {
	Connection
	KeyingInfo *KeyingInfo // nil if no keying info available
	LastHeard  string      // Human-readable last heard (e.g., "00:01:30" or "Never")
	Mode       string      // Link mode from LinkedNodes (T/R/C/M)
}

// VoterReceiver represents an RTCM receiver
type VoterReceiver struct {
	ID       string // Receiver ID/name
	RSSI     int    // Signal strength (0-255)
	Voted    bool   // Currently voted
	State    string // "ACTIVE", "STANDBY", etc.
	IP       string // Receiver IP address
	Callsign string // Receiver callsign (if available)
}

// VoterResult contains parsed voter output
type VoterResult struct {
	Node      int             // Local node number
	Receivers []VoterReceiver // List of receivers
	Timestamp time.Time       // When this data was captured
}

// Helper functions for formatting

// FormatElapsed converts seconds to HH:MM:SS format
func FormatElapsed(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return formatTime(h, m, s)
}

// FormatLastHeard converts keying info to last heard string
func FormatLastHeard(ki *KeyingInfo) string {
	if ki == nil {
		return "n/a"
	}

	// If currently keyed, show "Keying"
	if ki.IsKeyed {
		return "Keying"
	}

	// If never keyed (large value), show "Never"
	if ki.SecsSinceKeyed > 86400*365 { // More than a year
		return "Never"
	}

	// Otherwise format as HH:MM:SS
	return FormatElapsed(ki.SecsSinceKeyed)
}

func formatTime(h, m, s int) string {
	if h > 999 {
		return "Never"
	}
	return sprintf("%03d:%02d:%02d", h, m, s)
}

// Helper to format without fmt import
func sprintf(format string, h, m, s int) string {
	// Simple sprintf for HH:MM:SS format
	hStr := padLeft(itoa(h), 3, '0')
	mStr := padLeft(itoa(m), 2, '0')
	sStr := padLeft(itoa(s), 2, '0')
	return hStr + ":" + mStr + ":" + sStr
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	return string(buf[i+1:])
}

func padLeft(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	padding := make([]rune, length-len(s))
	for i := range padding {
		padding[i] = pad
	}
	return string(padding) + s
}
