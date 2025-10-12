package core

import (
	"context"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/repository"
)

// NodeInfo represents enriched node information from astdb
type NodeInfo struct {
	Node        int
	Callsign    string
	Description string
	Location    string
}

// NodeLookupService provides fast node lookups from SQLite database
type NodeLookupService struct {
	nodeInfoRepo *repository.NodeInfoRepository
}

// NewNodeLookupService creates a new node lookup service
// The astdbPath parameter is kept for backward compatibility but not used
func NewNodeLookupService(astdbPath string) *NodeLookupService {
	return &NodeLookupService{}
}

// SetNodeInfoRepository injects the node info repository
func (nls *NodeLookupService) SetNodeInfoRepository(repo *repository.NodeInfoRepository) {
	nls.nodeInfoRepo = repo
}

// LookupNode looks up a node by ID from the SQLite database
func (nls *NodeLookupService) LookupNode(nodeID int) *NodeInfo {
	if nls.nodeInfoRepo == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dbNode, err := nls.nodeInfoRepo.GetByNodeID(ctx, nodeID)
	if err != nil || dbNode == nil {
		return nil
	}

	return &NodeInfo{
		Node:        dbNode.NodeID,
		Callsign:    dbNode.Callsign,
		Description: dbNode.Description,
		Location:    dbNode.Location,
	}
}

// EnrichLinkInfo enriches a LinkInfo with node information from astdb
func (nls *NodeLookupService) EnrichLinkInfo(link *LinkInfo) {
	if link == nil {
		return
	}

	// Handle negative node IDs (hashed text nodes)
	if link.Node < 0 {
		if name, found := getTextNodeName(link.Node); found {
			link.NodeCallsign = name
			link.NodeDescription = "VOIP Node"
		}
		return
	}

	// Lookup positive node IDs in astdb
	if info := nls.LookupNode(link.Node); info != nil {
		link.NodeCallsign = info.Callsign
		link.NodeDescription = info.Description
		link.NodeLocation = info.Location
	}
}
