package ami

import (
	"testing"
)

func TestParseLinkedNodesWithTextNodes(t *testing.T) {
	// Simulate the LinkedNodes line from the user's example
	line := "LinkedNodes: T550465, T58840, T588841, T590110, T595570, KF8S, MKF8S"

	nodes := parseLinkedNodes(line)

	if len(nodes) != 7 {
		t.Fatalf("expected 7 nodes, got %d", len(nodes))
	}

	// Find the KF8S node
	var kf8sNode *LinkedNode
	for i := range nodes {
		// Check if this is a text node (negative ID)
		if nodes[i].Node < 0 {
			// Check if it's registered in the AMI registry
			if name, ok := GetTextNodeFromAMI(nodes[i].Node); ok && name == "KF8S" {
				kf8sNode = &nodes[i]
				break
			}
		}
	}

	if kf8sNode == nil {
		t.Fatal("KF8S node not found in parsed LinkedNodes")
	}

	if kf8sNode.Mode != "" {
		t.Errorf("expected empty mode for raw callsign KF8S, got %s", kf8sNode.Mode)
	}

	// Find the MKF8S node (Monitored KF8S)
	var mkf8sNode *LinkedNode
	for i := range nodes {
		if nodes[i].Node < 0 {
			if name, ok := GetTextNodeFromAMI(nodes[i].Node); ok && name == "KF8S" {
				// The second one we find with the same ID should have mode M
				if nodes[i].Mode == "M" {
					mkf8sNode = &nodes[i]
					break
				}
			}
		}
	}

	if mkf8sNode == nil {
		t.Fatal("MKF8S node not found in parsed LinkedNodes")
	}

	if mkf8sNode.Mode != "M" {
		t.Errorf("expected mode M, got %s", mkf8sNode.Mode)
	}

	if kf8sNode.Node >= 0 {
		t.Errorf("expected negative node ID for text node, got %d", kf8sNode.Node)
	}
}

func TestCombineXStatSawStatWithTextNodes(t *testing.T) {
	// Create XStat result with text node in LinkedNodes but not in Connections
	xstat := &XStatResult{
		Node: 594950,
		Connections: []Connection{
			{Node: 550465, IP: "162.230.255.145", Direction: "IN"},
		},
		LinkedNodes: []LinkedNode{
			{Node: 550465, Mode: "T"},
			// KF8S will be hashed to a negative number
		},
	}

	// Parse the text node
	callsign := "KF8S"
	nodeID := hashTextNodeToInt(callsign)
	registerTextNodeInAMI(nodeID, callsign)
	xstat.LinkedNodes = append(xstat.LinkedNodes, LinkedNode{Node: nodeID, Mode: "T"})

	combined := CombineXStatSawStat(xstat, nil)

	// Should have 2 connections: one from Connections, one synthetic from LinkedNodes
	if len(combined.Connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(combined.Connections))
	}

	// Find the KF8S connection
	var kf8sConn *ConnectionWithHistory
	for i := range combined.Connections {
		if combined.Connections[i].Node == nodeID {
			kf8sConn = &combined.Connections[i]
			break
		}
	}

	if kf8sConn == nil {
		t.Fatal("KF8S connection not found in combined result")
	}

	if kf8sConn.Mode != "T" {
		t.Errorf("expected mode T, got %s", kf8sConn.Mode)
	}
}
