import { ref, watch } from 'vue'
import { logger } from '../utils/logger'

// Global cache for node lookups
const nodeCache = new Map()
const pendingRequests = new Map()

/**
 * Composable for looking up node information from astdb
 * Automatically enriches links with callsign, description, and location
 */
export function useNodeLookup(links) {
  const enrichedLinks = ref([])

  async function lookupNode(nodeID) {
    // Check cache first
    if (nodeCache.has(nodeID)) {
      return nodeCache.get(nodeID)
    }

    // Check if request is already pending
    if (pendingRequests.has(nodeID)) {
      return pendingRequests.get(nodeID)
    }

    // Make API request
    const promise = fetch(`/api/node-lookup?q=${nodeID}`)
      .then(r => r.json())
      .then(data => {
        // Backend returns an envelope { ok: true, data: { results: [...] } }
        // but older clients expect a flat { results: [...] }. Support both shapes.
        const results = (data && data.results) || (data && data.data && data.data.results) || []
        if (results && results.length > 0) {
          const node = results[0]
          const result = {
            callsign: node.callsign,
            description: node.description,
            location: node.location
          }
          nodeCache.set(nodeID, result)
          pendingRequests.delete(nodeID)
          return result
        }
        // Not found - cache null to avoid repeated lookups
        nodeCache.set(nodeID, null)
        pendingRequests.delete(nodeID)
        return null
      })
      .catch(err => {
        logger.error(`Node lookup failed for ${nodeID}:`, err)
        pendingRequests.delete(nodeID)
        return null
      })

    pendingRequests.set(nodeID, promise)
    return promise
  }

  async function enrichLinks(linksList) {
    if (!linksList || linksList.length === 0) {
      enrichedLinks.value = []
      return
    }

    // Enrich each link with node information
    const enriched = await Promise.all(
      linksList.map(async (link) => {
        const nodeInfo = await lookupNode(link.node)
        return {
          ...link,
          node_callsign: nodeInfo?.callsign || '',
          node_description: nodeInfo?.description || '',
          node_location: nodeInfo?.location || ''
        }
      })
    )

    enrichedLinks.value = enriched
  }

  // Watch links and enrich when they change
  watch(() => links.value, (newLinks) => {
    enrichLinks(newLinks)
  }, { immediate: true, deep: true })

  return {
    enrichedLinks
  }
}
