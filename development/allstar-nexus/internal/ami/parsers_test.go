package ami

import (
	"os"
	"testing"
)

func TestParseXStat(t *testing.T) {
	data, err := os.ReadFile("testdata/xstat_basic.txt")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	result, err := ParseXStat(1999, string(data))
	if err != nil {
		t.Fatalf("ParseXStat failed: %v", err)
	}

	// Check basic fields
	if result.Node != 1999 {
		t.Errorf("expected node 1999, got %d", result.Node)
	}

	if !result.RxKeyed {
		t.Error("expected RxKeyed=true")
	}

	if result.TxKeyed {
		t.Error("expected TxKeyed=false")
	}

	// Check connections
	if len(result.Connections) != 3 {
		t.Fatalf("expected 3 connections, got %d", len(result.Connections))
	}

	// Check first connection (2000)
	conn := result.Connections[0]
	if conn.Node != 2000 {
		t.Errorf("expected node 2000, got %d", conn.Node)
	}
	if conn.IP != "192.168.1.10" {
		t.Errorf("expected IP 192.168.1.10, got %s", conn.IP)
	}
	if conn.IsKeyed {
		t.Error("expected IsKeyed=false for node 2000")
	}
	if conn.Direction != "OUT" {
		t.Errorf("expected Direction=OUT, got %s", conn.Direction)
	}

	// Check second connection (2001) - keyed
	conn2 := result.Connections[1]
	if !conn2.IsKeyed {
		t.Error("expected IsKeyed=true for node 2001")
	}
	if conn2.Direction != "IN" {
		t.Errorf("expected Direction=IN, got %s", conn2.Direction)
	}

	// Check LinkedNodes
	if len(result.LinkedNodes) != 3 {
		t.Fatalf("expected 3 linked nodes, got %d", len(result.LinkedNodes))
	}

	// Check modes
	if result.LinkedNodes[0].Mode != "T" {
		t.Errorf("expected mode T for node 2000, got %s", result.LinkedNodes[0].Mode)
	}
	if result.LinkedNodes[1].Mode != "R" {
		t.Errorf("expected mode R for node 2001, got %s", result.LinkedNodes[1].Mode)
	}
	if result.LinkedNodes[2].Mode != "C" {
		t.Errorf("expected mode C for node 2002, got %s", result.LinkedNodes[2].Mode)
	}

	// Check variables
	if result.Variables["RPT_ASEL"] != "1" {
		t.Errorf("expected RPT_ASEL=1, got %s", result.Variables["RPT_ASEL"])
	}
}

func TestParseXStatEchoLink(t *testing.T) {
	data, err := os.ReadFile("testdata/xstat_echolink.txt")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	result, err := ParseXStat(1999, string(data))
	if err != nil {
		t.Fatalf("ParseXStat failed: %v", err)
	}

	// Check connections
	if len(result.Connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(result.Connections))
	}

	// Check EchoLink connection (3123456) - no IP field
	var echoConn *Connection
	for i := range result.Connections {
		if result.Connections[i].Node == 3123456 {
			echoConn = &result.Connections[i]
			break
		}
	}

	if echoConn == nil {
		t.Fatal("EchoLink connection not found")
	}

	if echoConn.IP != "" {
		t.Errorf("expected empty IP for EchoLink node, got %s", echoConn.IP)
	}

	if !echoConn.IsKeyed {
		t.Error("expected EchoLink node to be keyed")
	}

	// Check RX/TX state
	if result.RxKeyed {
		t.Error("expected RxKeyed=false")
	}

	if !result.TxKeyed {
		t.Error("expected TxKeyed=true")
	}
}

func TestParseSawStat(t *testing.T) {
	data, err := os.ReadFile("testdata/sawstat_basic.txt")
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	result, err := ParseSawStat(1999, string(data))
	if err != nil {
		t.Fatalf("ParseSawStat failed: %v", err)
	}

	// Check basic fields
	if result.Node != 1999 {
		t.Errorf("expected node 1999, got %d", result.Node)
	}

	// Check nodes
	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}

	// Check node 2000 - not keyed, 90s since last keyed
	ki := result.Nodes[2000]
	if ki == nil {
		t.Fatal("node 2000 not found")
	}
	if ki.IsKeyed {
		t.Error("expected IsKeyed=false for node 2000")
	}
	if ki.SecsSinceKeyed != 90 {
		t.Errorf("expected 90 secs since keyed, got %d", ki.SecsSinceKeyed)
	}
	if ki.SecsSinceUnkeyed != 1800 {
		t.Errorf("expected 1800 secs since unkeyed, got %d", ki.SecsSinceUnkeyed)
	}

	// Check node 2001 - currently keyed
	ki2 := result.Nodes[2001]
	if ki2 == nil {
		t.Fatal("node 2001 not found")
	}
	if !ki2.IsKeyed {
		t.Error("expected IsKeyed=true for node 2001")
	}
	if ki2.SecsSinceKeyed != 0 {
		t.Errorf("expected 0 secs since keyed (currently keying), got %d", ki2.SecsSinceKeyed)
	}
}

func TestCombineXStatSawStat(t *testing.T) {
	// Load test data
	xstatData, err := os.ReadFile("testdata/xstat_basic.txt")
	if err != nil {
		t.Fatalf("failed to read xstat data: %v", err)
	}

	sawstatData, err := os.ReadFile("testdata/sawstat_basic.txt")
	if err != nil {
		t.Fatalf("failed to read sawstat data: %v", err)
	}

	// Parse both
	xstat, err := ParseXStat(1999, string(xstatData))
	if err != nil {
		t.Fatalf("ParseXStat failed: %v", err)
	}

	sawstat, err := ParseSawStat(1999, string(sawstatData))
	if err != nil {
		t.Fatalf("ParseSawStat failed: %v", err)
	}

	// Combine
	combined := CombineXStatSawStat(xstat, sawstat)

	// Check combined fields
	if combined.Node != 1999 {
		t.Errorf("expected node 1999, got %d", combined.Node)
	}

	if !combined.RxKeyed {
		t.Error("expected RxKeyed=true")
	}

	if combined.TxKeyed {
		t.Error("expected TxKeyed=false")
	}

	// Check connections with history
	if len(combined.Connections) != 3 {
		t.Fatalf("expected 3 connections, got %d", len(combined.Connections))
	}

	// Check node 2000 - has keying info
	conn := combined.Connections[0]
	if conn.Node != 2000 {
		t.Errorf("expected node 2000, got %d", conn.Node)
	}

	if conn.KeyingInfo == nil {
		t.Fatal("expected keying info for node 2000")
	}

	if conn.LastHeard == "" {
		t.Error("expected last heard to be set")
	}

	// Check mode is set from LinkedNodes
	if conn.Mode != "T" {
		t.Errorf("expected mode T, got %s", conn.Mode)
	}

	// Check node 2001 - currently keyed
	conn2 := combined.Connections[1]
	if conn2.KeyingInfo == nil {
		t.Fatal("expected keying info for node 2001")
	}

	if conn2.LastHeard != "Keying" {
		t.Errorf("expected LastHeard=Keying for currently keyed node, got %s", conn2.LastHeard)
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "000:00:00"},
		{30, "000:00:30"},
		{90, "000:01:30"},
		{3600, "001:00:00"},
		{3665, "001:01:05"},
		{86400, "024:00:00"},
	}

	for _, tt := range tests {
		result := FormatElapsed(tt.seconds)
		if result != tt.expected {
			t.Errorf("FormatElapsed(%d) = %s, expected %s", tt.seconds, result, tt.expected)
		}
	}
}

func TestFormatLastHeard(t *testing.T) {
	tests := []struct {
		name     string
		ki       *KeyingInfo
		expected string
	}{
		{
			name:     "nil keying info",
			ki:       nil,
			expected: "n/a",
		},
		{
			name: "currently keying",
			ki: &KeyingInfo{
				IsKeyed:        true,
				SecsSinceKeyed: 0,
			},
			expected: "Keying",
		},
		{
			name: "never heard",
			ki: &KeyingInfo{
				IsKeyed:        false,
				SecsSinceKeyed: 999999999,
			},
			expected: "Never",
		},
		{
			name: "recent",
			ki: &KeyingInfo{
				IsKeyed:        false,
				SecsSinceKeyed: 90,
			},
			expected: "000:01:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLastHeard(tt.ki)
			if result != tt.expected {
				t.Errorf("FormatLastHeard() = %s, expected %s", result, tt.expected)
			}
		})
	}
}
