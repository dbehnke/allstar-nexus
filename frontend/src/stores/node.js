import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from './auth'
import { logger } from '../utils/logger'

export const useNodeStore = defineStore('node', () => {
  const status = ref(null)
  const links = ref([])
  const talker = ref([])
  const topLinks = ref([])
  const nowTick = ref(Date.now())
  // Persistent per-node first-seen timestamps so components can show steadily-increasing connected timers
  const connectionSeenAt = ref({})
  // Persistent map of disconnect timestamps: key `${sourceID}:${adjacentID}` -> ms epoch
  const removedAt = ref(loadRemovedAt())
  // pending stop timers: node -> timer id
  const pendingStops = new Map()
  // track server session to detect restarts
  const lastSessionStart = ref(null)
  // Track last seen state version and when we last received a STATUS_UPDATE
  const lastStateVersion = ref(null)
  const lastStatusAt = ref(0)
  // source node keying state: sourceNodeID -> { adjacentNodes: {...}, txKeyed, rxKeyed }
  const sourceNodes = ref({})
  // internal: tracks stale markers for sourceNodes during soft resync
  const _staleMarkers = new Map()

  function handleWSMessage(msg) {
    // Temporary instrumentation: log every incoming WS envelope for debugging
    try { logger.debug('[WS RECV]', msg && msg.messageType, msg) } catch {}
    if (msg.messageType === 'STATUS_UPDATE') {
  logger.debug('[WS] STATUS_UPDATE received, session_start=', msg.data && msg.data.session_start, 'state_version=', msg.data && msg.data.state_version)
      // If we were disconnected for a while, and server state_version changed while we were away,
      // force a full reload to avoid subtle client-side drift.
      const nowMs = Date.now()
      const lastAt = lastStatusAt.value || 0
      const gapMs = nowMs - lastAt
      const newVersion = msg.data && msg.data.state_version
      if (lastAt > 0 && gapMs > 30000 && lastStateVersion.value != null && newVersion != null && newVersion !== lastStateVersion.value) {
  logger.warn('[WS] Detected long gap since last STATUS_UPDATE', { gapMs, from: lastStateVersion.value, to: newVersion })
        // Perform in-place soft re-sync using the provided snapshot to avoid a full page reload.
        softResync(msg.data)
        // Update trackers and bail out of normal assignment since softResync applied the snapshot
        lastStatusAt.value = nowMs
        lastStateVersion.value = newVersion
        return
      }
      lastStatusAt.value = nowMs
      lastStateVersion.value = newVersion
      // Detect server restart by checking session_start
      if (msg.data.session_start && lastSessionStart.value && msg.data.session_start !== lastSessionStart.value) {
        // Server restarted - clear talker log
        talker.value = []
  logger.info('Server restarted, cleared talker log')
      }
      lastSessionStart.value = msg.data.session_start

      status.value = msg.data
  logger.debug('[WS] assigned status, links_detailed present=', !!(msg.data && msg.data.links_detailed))
      if (msg.data.links_detailed) {
        links.value = msg.data.links_detailed
        // Rebuild per-source adjacency strictly from server snapshot to keep browser in sync
        try {
          const grouped = new Map()
          for (const li of links.value) {
            const sid = li.local_node || li.LocalNode || 0
            if (!sid) continue
            if (!grouped.has(sid)) grouped.set(sid, [])
            grouped.get(sid).push(li)
          }
          const now = Date.now()
          for (const [sid, arr] of grouped.entries()) {
            const key = String(sid)
            const prev = sourceNodes.value[key] || { sourceNodeID: sid, adjacentNodes: {}, txKeyed: false, rxKeyed: false, timestamp: now }
            const prevAdj = prev.adjacentNodes || {}
            const mergedAdj = { ...prevAdj }
            for (const li of arr) {
              const nodeId = li.node || li.Node || 0
              if (!nodeId) continue
              const prevNode = prevAdj[nodeId] || { NodeID: nodeId, IsTransmitting: false, IsKeyed: false, KeyedStartTime: null, TotalTxSeconds: 0 }
              const base = { ...prevNode }
              // Enrichment fields from snapshot
              base.Callsign = li.node_callsign || li.NodeCallsign || base.Callsign || ''
              base.Description = li.node_description || li.NodeDescription || base.Description || ''
              base.Mode = li.mode || li.Mode || base.Mode || ''
              base.Direction = li.direction || li.Direction || base.Direction || ''
              base.IP = li.ip || li.IP || base.IP || ''
              base.ConnectedSince = li.connected_since || li.ConnectedSince || base.ConnectedSince || null
              // Update totals if provided
              if ((li.total_tx_seconds ?? li.TotalTxSeconds) != null) {
                base.TotalTxSeconds = li.total_tx_seconds ?? li.TotalTxSeconds
              }
              // If snapshot indicates currently transmitting and we lack a start time, adopt it
              const snapTx = !!(li.current_tx || li.CurrentTx)
              if (snapTx) {
                base.IsTransmitting = true
                if (!base.KeyedStartTime) {
                  base.KeyedStartTime = li.last_tx_start || li.LastTxStart || base.KeyedStartTime || null
                }
              }
              mergedAdj[nodeId] = base
              // Clear any persisted removal timestamp for present links
              clearRemovedAt(sid, nodeId)
            }
            // Remove nodes missing from this snapshot unless they are actively transmitting (avoid flicker)
            for (const nidStr of Object.keys(prevAdj)) {
              const nid = parseInt(nidStr, 10)
              if (!mergedAdj[nid]) {
                const prevNode = prevAdj[nid]
                if (prevNode && prevNode.IsTransmitting) {
                  // keep for now to avoid TX flicker
                  mergedAdj[nid] = prevNode
                } else {
                  // persist removal timestamp and drop from adjacency
                  setRemovedAt(sid, nid, now)
                  delete mergedAdj[nid]
                }
              }
            }
            sourceNodes.value[key] = { ...prev, adjacentNodes: mergedAdj, timestamp: now }
          }
        } catch (e) { logger.debug('[WS] STATUS_UPDATE rebuild sourceNodes failed', e) }
      }
    } else if (msg.messageType === 'TALKER_LOG_SNAPSHOT') {
      // Full talker log snapshot from server (on connect or periodic refresh)
      // Replace entire talker log with enriched server data
      // Note: events may not have a node field at all (omitempty), so keep all events
      talker.value = Array.isArray(msg.data) ? msg.data : []
  logger.debug('[WS] Received TALKER_LOG_SNAPSHOT count=', talker.value.length)
    } else if (msg.messageType === 'TALKER_EVENT') {
      try {
        const incoming = msg.data
        
        // Skip events without a valid node number (node 0 or missing)
        if (!incoming.node || incoming.node === 0) { logger.debug('[WS] TALKER_EVENT skipped missing node', incoming); return }
        
        const now = new Date(incoming.at).getTime()
        
        // Check last 5 events for duplicates (same node+kind within 2s)
        const recentEvents = talker.value.slice(-5)
        for (const recent of recentEvents) {
          if (recent.kind === incoming.kind && recent.node === incoming.node) {
            const recentTs = new Date(recent.at).getTime()
            if (Math.abs(now - recentTs) < 2000) {
              logger.info('Skipping duplicate talker event:', incoming.kind, incoming.node)
              return
            }
          }
        }

        // If this is a STOP without a known start, buffer it briefly to allow a delayed START to arrive
        if (incoming.kind === 'TX_STOP') {
          const nodeId = incoming.node || 0
          const hasRecentStart = talker.value.some(t => (t.node || 0) === nodeId && t.kind === 'TX_START' && (new Date(incoming.at).getTime() - new Date(t.at).getTime()) < 30000)
          if (!hasRecentStart) {
            // set a 1s timer to push the STOP unless a START arrives in the meantime
            if (pendingStops.has(nodeId)) {
              clearTimeout(pendingStops.get(nodeId))
            }
            const tid = setTimeout(() => {
              talker.value.push(incoming)
              if (talker.value.length > 100) talker.value.shift()
              pendingStops.delete(nodeId)
            }, 1000)
            pendingStops.set(nodeId, tid)
            return
          }
        }

  // Normal push
        talker.value.push(incoming)
  logger.debug('[WS] TALKER_EVENT pushed', incoming.kind, incoming.node)
        if (talker.value.length > 100) talker.value.shift()
        // If this is a START, cancel any pending STOP timers for the same node
        if (incoming.kind === 'TX_START') {
          const nodeId = incoming.node || 0
          if (pendingStops.has(nodeId)) {
            clearTimeout(pendingStops.get(nodeId))
            pendingStops.delete(nodeId)
          }
        }
      } catch (e) {
        talker.value.push(msg.data)
        if (talker.value.length > 100) talker.value.shift()
      }
    } else if (msg.messageType === 'LINK_ADDED') {
      for (const add of msg.data) {
        if (!links.value.find(l => l.node === add.node)) links.value.push(add)
        // Merge into per-source adjacency so SourceNodeCard updates even without TX events
        try {
          const sid = add.local_node || add.LocalNode || 0
          const nodeId = add.node || add.Node || 0
          if (sid && nodeId) {
            const key = String(sid)
            const existing = sourceNodes.value[key] || {
              sourceNodeID: sid,
              adjacentNodes: {},
              txKeyed: false,
              rxKeyed: false,
              timestamp: Date.now()
            }
            const adj = existing.adjacentNodes || {}
            // Build a minimal adjacent node entry (fields align with SourceNodeCard normalization)
            adj[nodeId] = {
              NodeID: nodeId,
              Callsign: add.node_callsign || add.NodeCallsign || '',
              Description: add.node_description || add.NodeDescription || '',
              IsTransmitting: !!(add.current_tx || add.CurrentTx),
              IsKeyed: !!(add.is_keyed || add.IsKeyed),
              KeyedStartTime: add.last_tx_start || add.LastTxStart || null,
              TotalTxSeconds: add.total_tx_seconds || add.TotalTxSeconds || 0,
              Mode: add.mode || add.Mode || '',
              Direction: add.direction || add.Direction || '',
              IP: add.ip || add.IP || '',
              ConnectedSince: add.connected_since || add.ConnectedSince || Date.now()
            }
            // Reassign to trigger reactivity
            sourceNodes.value[key] = { ...existing, adjacentNodes: { ...adj } }
            // Clear any persisted removal timestamp for this pair (reconnected)
            clearRemovedAt(sid, nodeId)
          }
        } catch (e) { logger.debug('[WS] LINK_ADDED merge failed', e) }
        logger.debug('[WS] LINK_ADDED', add.node)
      }
    } else if (msg.messageType === 'LINK_REMOVED') {
      for (const id of msg.data) {
        const idx = links.value.findIndex(l => l.node === id)
        if (idx >= 0) links.value.splice(idx, 1)
        // Remove from any source node adjacency maps so cards update immediately
        try {
          const keys = Object.keys(sourceNodes.value)
          for (const key of keys) {
            const entry = sourceNodes.value[key]
            if (entry && entry.adjacentNodes && entry.adjacentNodes[id]) {
              // Persist removal timestamp so Lost timer survives reload
              setRemovedAt(entry.sourceNodeID || parseInt(key, 10) || 0, id, Date.now())
              const adj = { ...entry.adjacentNodes }
              delete adj[id]
              sourceNodes.value[key] = { ...entry, adjacentNodes: adj }
            }
          }
        } catch (e) { logger.debug('[WS] LINK_REMOVED merge failed', e) }
        logger.debug('[WS] LINK_REMOVED', id)
      }
    } else if (msg.messageType === 'LINK_TX') {
  updateLinkTX([msg.data])
  logger.debug('[WS] LINK_TX', msg.data && msg.data.node, msg.data && msg.data.kind)
    } else if (msg.messageType === 'LINK_TX_BATCH') {
  updateLinkTX(msg.data)
  logger.debug('[WS] LINK_TX_BATCH count=', Array.isArray(msg.data) ? msg.data.length : 0)
    } else if (msg.messageType === 'SOURCE_NODE_KEYING') {
      // Update source node keying state (incremental merge)
      const data = msg.data
      const sid = String(data.source_node_id)
      const entry = {
        sourceNodeID: data.source_node_id,
        adjacentNodes: data.adjacent_nodes || {},
        txKeyed: data.tx_keyed || false,
        rxKeyed: data.rx_keyed || false,
        // server-provided per-source totals when available
        num_links: (typeof data.num_links === 'number') ? data.num_links : undefined,
        num_alinks: (typeof data.num_alinks === 'number') ? data.num_alinks : undefined,
        timestamp: data.timestamp
      }
      // If this entry was marked stale, clear the stale marker and remove the flag
      if (sourceNodes.value[sid] && sourceNodes.value[sid]._stale) {
        delete sourceNodes.value[sid]._stale
      }
      sourceNodes.value[sid] = entry
  if (_staleMarkers.has(sid)) _staleMarkers.delete(sid)
  logger.debug('[WS] SOURCE_NODE_KEYING', data.source_node_id, 'txKeyed=', data.tx_keyed, 'adjacent_count=', data.adjacent_nodes ? Object.keys(data.adjacent_nodes).length : 0)
    }
  }

  // Soft re-sync: replace store state in-place using a STATUS_UPDATE snapshot from server.
  // This avoids a full page reload and preserves runtime UI state (open panels, theme, etc.).
  async function softResync(snap) {
    try {
      if (!snap) return
  logger.debug('[WS] softResync applying snapshot state_version=', snap.state_version)
      // Replace status and links in-place
      status.value = snap
      if (snap.links_detailed) {
        links.value = snap.links_detailed
      } else if (snap.links) {
        // Some snapshots may include just IDs; keep links array minimal
        links.value = snap.links.map(id => ({ node: id }))
      }
      // Reset talker feed to avoid duplicates; we'll rely on TALKER_LOG_SNAPSHOT to repopulate if available
      talker.value = []
      // Mark existing sourceNodes entries as stale; incoming SOURCE_NODE_KEYING messages
      // will clear the stale marker and update entries incrementally.
      for (const k of Object.keys(sourceNodes.value)) {
        _staleMarkers.set(k, Date.now())
        if (sourceNodes.value[k]) sourceNodes.value[k]._stale = true
      }
      // Schedule cleanup of stale entries after 6s
      setTimeout(() => {
        const now = Date.now()
        for (const [k, ts] of Array.from(_staleMarkers.entries())) {
          if (now - ts > 6000) {
            delete sourceNodes.value[k]
            _staleMarkers.delete(k)
          }
        }
      }, 7000)
      // Immediately fetch talker history and top links to speed up re-sync
      try {
        const auth = useAuthStore()
        const headers = auth.getAuthHeaders()
        // Fetch top links and talker log in parallel
        const [topResp, talkerResp] = await Promise.all([
          fetch('/api/link-stats/top?limit=5', { headers }).then(r => r.json()).catch(() => null),
          fetch('/api/talker-log', { headers }).then(r => r.json()).catch(() => null)
        ])
        if (topResp && topResp.ok) {
          topLinks.value = topResp.data.results || []
        }
        if (talkerResp && talkerResp.ok && talkerResp.events) {
          talker.value = talkerResp.events
        }
      } catch (e) { logger.debug('[WS] softResync additional fetches failed', e) }

  logger.debug('[WS] softResync complete; updated talker and topLinks from API')
    } catch (e) {
  logger.error('[WS] softResync failed, falling back to full reload', e)
      window.location.reload()
    }
  }

  function updateLinkTX(events) {
    for (const e of events) {
      const li = links.value.find(l => l.node === e.node)
      if (li) {
        if (e.kind === 'START') {
          li.current_tx = true
          li.last_tx_start = e.last_tx_start || e.at
        } else if (e.kind === 'STOP') {
          li.current_tx = false
          li.last_tx_end = e.last_tx_end || e.at
          li.total_tx_seconds = e.total_tx_seconds
        }
      }
    }
  }

  function setTopLinks(data) {
    topLinks.value = data
  }

  function startTickTimer() {
    setInterval(() => {
      nowTick.value = Date.now()
    }, 1000)
    // Periodic cleanup of stale removedAt entries (every 10s)
    setInterval(() => {
      pruneRemovedAt()
    }, 10000)
  }

  function loadTalkerHistory(events) {
    // Load historical talker events (from API on page load)
    // Keep all events - node field may be omitted entirely due to omitempty
    talker.value = events || []
  }

  function setRemovedAt(sourceId, nodeId, ts) {
    try {
      if (!sourceId || !nodeId) return
      const key = `${sourceId}:${nodeId}`
      const map = removedAt.value || {}
      if (!map[key]) {
        map[key] = ts
        removedAt.value = { ...map }
        saveRemovedAt(removedAt.value)
      }
    } catch (e) {}
  }

  function clearRemovedAt(sourceId, nodeId) {
    try {
      if (!sourceId || !nodeId) return
      const key = `${sourceId}:${nodeId}`
      const map = removedAt.value || {}
      if (key in map) {
        delete map[key]
        removedAt.value = { ...map }
        saveRemovedAt(removedAt.value)
      }
    } catch (e) {}
  }

  function getRemovedAt(sourceId, nodeId) {
    try {
      if (!sourceId || !nodeId) return null
      const key = `${sourceId}:${nodeId}`
      const map = removedAt.value || {}
      return map[key] || null
    } catch (e) {
      return null
    }
  }

  function pruneRemovedAt() {
    try {
      const map = removedAt.value || {}
      const now = Date.now()
      const ttl = 60 * 60 * 1000 // 1 hour max retention as safety
      let changed = false
      for (const [k, v] of Object.entries(map)) {
        if (typeof v !== 'number' || v <= 0) {
          delete map[k]
          changed = true
          continue
        }
        if (now - v > 6 * 60 * 60 * 1000) { // hard-stop after 6 hours
          delete map[k]
          changed = true
        }
      }
      if (changed) {
        removedAt.value = { ...map }
        saveRemovedAt(removedAt.value)
      }
    } catch (e) {}
  }

  return {
    status,
    links,
    talker,
    topLinks,
    nowTick,
    connectionSeenAt,
    removedAt,
    sourceNodes,
    handleWSMessage,
    setTopLinks,
    startTickTimer,
    loadTalkerHistory,
    // expose helpers
    setRemovedAt,
    clearRemovedAt,
    getRemovedAt,
    pruneRemovedAt,
  }
})

// Persistence helpers (defined in-module to access closure vars)
// NOTE: These sit after defineStore to avoid Vue SSR constraints; they use localStorage directly.
function loadRemovedAt() {
  try {
    const raw = localStorage.getItem('nexus_removedAt')
    return raw ? JSON.parse(raw) : {}
  } catch (e) {
    return {}
  }
}

function saveRemovedAt(map) {
  try {
    localStorage.setItem('nexus_removedAt', JSON.stringify(map || {}))
  } catch (e) {}
}

