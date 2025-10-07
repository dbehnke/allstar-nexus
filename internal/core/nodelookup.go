package core

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NodeInfo represents enriched node information from astdb
type NodeInfo struct {
	Node        int
	Callsign    string
	Description string
	Location    string
}

// NodeLookupService provides fast node lookups with caching
type NodeLookupService struct {
	astdbPath string
	mu        sync.RWMutex
	cache     map[int]*NodeInfo
	lastLoad  time.Time
	cacheTTL  time.Duration
}

// NewNodeLookupService creates a new node lookup service
func NewNodeLookupService(astdbPath string) *NodeLookupService {
	return &NodeLookupService{
		astdbPath: astdbPath,
		cache:     make(map[int]*NodeInfo),
		cacheTTL:  5 * time.Minute, // Refresh cache every 5 minutes
	}
}

// LookupNode looks up a node by ID, using cache when available
func (nls *NodeLookupService) LookupNode(nodeID int) *NodeInfo {
	// Check if cache is valid
	nls.mu.RLock()
	needsRefresh := time.Since(nls.lastLoad) > nls.cacheTTL
	if !needsRefresh {
		if info, found := nls.cache[nodeID]; found {
			nls.mu.RUnlock()
			return info
		}
	}
	nls.mu.RUnlock()

	// Refresh cache if needed
	if needsRefresh {
		nls.loadCache()
	}

	// Try again after cache refresh
	nls.mu.RLock()
	info := nls.cache[nodeID]
	nls.mu.RUnlock()

	return info
}

// loadCache loads all nodes from astdb into memory cache
func (nls *NodeLookupService) loadCache() {
	file, err := os.Open(nls.astdbPath)
	if err != nil {
		return // Silently fail - astdb may not exist yet
	}
	defer file.Close()

	newCache := make(map[int]*NodeInfo)
	scanner := bufio.NewScanner(file)

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

		newCache[nodeNum] = &NodeInfo{
			Node:        nodeNum,
			Callsign:    callsign,
			Description: description,
			Location:    location,
		}
	}

	// Update cache atomically
	nls.mu.Lock()
	nls.cache = newCache
	nls.lastLoad = time.Now()
	nls.mu.Unlock()
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
