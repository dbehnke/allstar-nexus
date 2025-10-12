import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from './auth'
import { logger } from '../utils/logger'
import { useUIStore } from './ui'

// Clean, minimal Pinia node store focused on scoreboard behaviors used by unit tests.
export const useNodeStore = defineStore('node', () => {
  const scoreboard = ref([])
  const recentTransmissions = ref([])
  const levelConfig = ref({})
  const gamificationEnabled = ref(false)

  // Restored shape expected by the Dashboard and other components
  const links = ref([]) // array of link objects { node, current_tx, node_callsign, ... }
  const talker = ref([]) // talker events
  const topLinks = ref([])
  const sourceNodes = ref({}) // keyed by source node id
  const nowTick = ref(Date.now())
  const status = ref({}) // most recent STATUS_UPDATE payload
  const connectionSeenAt = ref({}) // per-node first-seen timestamps (used by SourceNodeCard)
  const lastEnvelopes = ref([]) // small ring buffer of recent WS envelopes for debugging

  let _scoreboardReloadTimer = null
  let _scoreboardPollTimer = null
  let _recentTxTimer = null
  let _tickTimer = null

  function handleWSMessage(msg) {
    try { logger.debug('[WS RECV]', msg && msg.messageType, msg) } catch (_) {}
    if (!msg || !msg.messageType) return
    // Handle several envelope types that update store state used by UI
    if (msg.messageType === 'STATUS_UPDATE') {
      try {
        const data = msg.data || {}
        status.value = data
        // Populate canonical links array from links_detailed or links
        if (Array.isArray(data.links_detailed) && data.links_detailed.length > 0) {
          links.value = data.links_detailed
        } else if (Array.isArray(data.links) && data.links.length > 0) {
          // If only ids provided, map to simple objects
          links.value = data.links.map(id => ({ node: id }))
        } else {
          links.value = []
        }
      } catch (e) { logger.debug('STATUS_UPDATE handler failed', e) }
      return
    }
    if (msg.messageType === 'TALKER_LOG_SNAPSHOT') {
      try { talker.value = Array.isArray(msg.data) ? msg.data : (msg.data && msg.data.events ? msg.data.events : []) } catch (e) { logger.debug('TALKER_LOG_SNAPSHOT handler failed', e) }
      return
    }
    if (msg.messageType === 'SOURCE_NODE_KEYING') {
      try {
        const snapshot = msg.data || {}
        const id = snapshot.source_node_id || snapshot.sourceNodeID || snapshot.sourceNode || snapshot.SourceNodeID || snapshot.SourceNode
        if (id) {
          // Normalize common snake_case -> camelCase fields so components can read either
          try {
            // source node id variants
            if (!snapshot.sourceNodeID && snapshot.source_node_id) snapshot.sourceNodeID = snapshot.source_node_id
            if (!snapshot.sourceNodeID && snapshot.SourceNodeID) snapshot.sourceNodeID = snapshot.SourceNodeID
            if (!snapshot.source_node_id && snapshot.sourceNodeID) snapshot.source_node_id = snapshot.sourceNodeID

            // adjacent nodes variants
            if (!snapshot.adjacentNodes && snapshot.adjacent_nodes) snapshot.adjacentNodes = snapshot.adjacent_nodes
            if (!snapshot.adjacentNodes && snapshot.AdjacentNodes) snapshot.adjacentNodes = snapshot.AdjacentNodes
            if (!snapshot.adjacent_nodes && snapshot.adjacentNodes) snapshot.adjacent_nodes = snapshot.adjacentNodes
            if (!snapshot.AdjacentNodes && snapshot.adjacentNodes) snapshot.AdjacentNodes = snapshot.adjacentNodes

            // links_detailed variants
            if (!snapshot.links_detailed && snapshot.linksDetailed) snapshot.links_detailed = snapshot.linksDetailed
            if (!snapshot.linksDetailed && snapshot.links_detailed) snapshot.linksDetailed = snapshot.links_detailed
            if (!snapshot.links_detailed && snapshot.LinksDetailed) snapshot.links_detailed = snapshot.LinksDetailed
            if (!snapshot.LinksDetailed && snapshot.links_detailed) snapshot.LinksDetailed = snapshot.links_detailed
          } catch (e) {}
          // Ensure adjacent node entries include numeric NodeID for templates
          try {
            const map = snapshot.adjacentNodes || snapshot.adjacent_nodes || snapshot.AdjacentNodes
            if (map && typeof map === 'object') {
              for (const k of Object.keys(map)) {
                const entry = map[k] || {}
                // If numeric NodeID missing, populate from key
                if (entry.NodeID == null && entry.node == null && entry.node_id == null) {
                  entry.NodeID = Number(k)
                }
                // coerce NodeID to number
                if (entry.NodeID != null) entry.NodeID = Number(entry.NodeID)
                map[k] = entry
              }
              // ensure snapshot.adjacentNodes points to the normalized map
              snapshot.adjacentNodes = map
            }
          } catch (e) {}

          // assign in an immutable way so reactivity picks up the change
          sourceNodes.value = Object.assign({}, sourceNodes.value, { [id]: snapshot })
        }
      } catch (e) { logger.debug('SOURCE_NODE_KEYING handler failed', e) }
      return
    }

    if (msg.messageType === 'GAMIFICATION_TALLY_COMPLETED') {
      try {
        const payload = msg.data || {}
        if (Array.isArray(payload.scoreboard) && payload.scoreboard.length > 0) {
          scoreboard.value = payload.scoreboard
        } else if (Array.isArray(payload)) {
          scoreboard.value = payload
        } else {
          fetchScoreboard(50)
        }
        triggerRecentTxRefresh()
        try { const ui = useUIStore(); ui.addToast && ui.addToast('Scoreboard updated', { type: 'success' }) } catch (e) { logger.debug('toast show failed', e) }
      } catch (e) {
        logger.debug('[WS] GAMIFICATION_TALLY_COMPLETED handler error', e)
        fetchScoreboard(50)
        triggerRecentTxRefresh()
      }
    }
  }

    // Record a brief summary of the envelope for debugging (keep last 20)
    try {
      const summary = { type: msg.messageType, ts: Date.now(), keys: msg && msg.data ? Object.keys(msg.data) : [] }
      lastEnvelopes.value = lastEnvelopes.value.concat([summary]).slice(-20)
    } catch (e) {}

  // Restore API expected by Dashboard.vue and other components
  function setTopLinks(arr) {
    try { topLinks.value = Array.isArray(arr) ? arr : [] } catch (e) {}
  }

  function loadTalkerHistory(events) {
    try { talker.value = Array.isArray(events) ? events : [] } catch (e) {}
  }

  function startTickTimer() {
    if (_tickTimer) return
    nowTick.value = Date.now()
    _tickTimer = setInterval(() => { nowTick.value = Date.now() }, 1000)
  }

  function stopTickTimer() {
    if (_tickTimer) { clearInterval(_tickTimer); _tickTimer = null }
  }

  function getRemovedAt(sourceNodeID, nodeID) {
    // Stub: persistent removed timestamps not implemented yet
    return null
  }

  async function fetchScoreboard(limit = 50) {
    try {
      let headers = {}
      try { const auth = useAuthStore(); headers = (auth && typeof auth.getAuthHeaders === 'function') ? auth.getAuthHeaders() : {} } catch (e) {}
      const res = await fetch(`/api/gamification/scoreboard?limit=${limit}`, { headers })
      const data = await res.json().catch(() => ({}))
      scoreboard.value = (data && (data.scoreboard || data.data || data.results)) || []
      gamificationEnabled.value = !!(data && (data.enabled || data.ok))
    } catch (e) { logger.debug('fetchScoreboard failed', e) }
  }

  function queueScoreboardReload(delayMs = 500, limit = 50) {
    if (_scoreboardReloadTimer) clearTimeout(_scoreboardReloadTimer)
    _scoreboardReloadTimer = setTimeout(() => {
      try {
        if (Array.isArray(scoreboard.value) && scoreboard.value.length > 0) return
        fetchScoreboard(limit)
      } catch (e) { logger.debug('queued scoreboard reload failed', e) }
      _scoreboardReloadTimer = null
    }, delayMs)
  }

  function startScoreboardPoll(intervalMs = 60000, limit = 50) {
    if (_scoreboardPollTimer) clearInterval(_scoreboardPollTimer)
    try { if (!Array.isArray(scoreboard.value) || scoreboard.value.length === 0) fetchScoreboard(limit) } catch (e) { logger.debug('initial scoreboard poll failed', e) }
    _scoreboardPollTimer = setInterval(() => {
      try { fetchScoreboard(limit) } catch (e) { logger.debug('scoreboard poll fetch failed', e) }
    }, intervalMs)
  }

  function stopScoreboardPoll() {
    if (_scoreboardReloadTimer) { clearTimeout(_scoreboardReloadTimer); _scoreboardReloadTimer = null }
    if (_scoreboardPollTimer) { clearInterval(_scoreboardPollTimer); _scoreboardPollTimer = null }
  }

  function triggerRecentTxRefresh() {
    if (_recentTxTimer) clearTimeout(_recentTxTimer)
    _recentTxTimer = setTimeout(() => {
      try {
        let headers = {}
        try { const auth = useAuthStore(); headers = (auth && typeof auth.getAuthHeaders === 'function') ? auth.getAuthHeaders() : {} } catch (e) {}
        fetch(`/api/gamification/recent-transmissions?limit=50&offset=0`, { headers })
          .then(r => r.json())
          .then(data => { recentTransmissions.value = (data && (data.transmissions || data.data || data.results)) || [] })
          .catch(() => {})
      } catch (e) {}
    }, 800)
  }

  async function fetchRecentTransmissions(limit = 50, offset = 0) {
    try {
      let headers = {}
      try { const auth = useAuthStore(); headers = (auth && typeof auth.getAuthHeaders === 'function') ? auth.getAuthHeaders() : {} } catch (e) {}
      const res = await fetch(`/api/gamification/recent-transmissions?limit=${limit}&offset=${offset}`, { headers })
      const data = await res.json()
      recentTransmissions.value = (data && (data.transmissions || data.data || data.results)) || []
    } catch (e) { logger.debug('fetchRecentTransmissions failed', e) }
  }

  async function fetchLevelConfig() {
    try {
      let headers = {}
      try { const auth = useAuthStore(); headers = (auth && typeof auth.getAuthHeaders === 'function') ? auth.getAuthHeaders() : {} } catch (e) {}
      const res = await fetch('/api/gamification/level-config', { headers })
      const data = await res.json()
      levelConfig.value = (data && (data.config || data.data)) || {}
    } catch (e) { logger.debug('fetchLevelConfig failed', e) }
  }

  return {
    scoreboard,
    recentTransmissions,
    levelConfig,
    gamificationEnabled,
    // restored fields
    links,
    talker,
    topLinks,
    sourceNodes,
    nowTick,
    handleWSMessage,
    fetchScoreboard,
    queueScoreboardReload,
    startScoreboardPoll,
    stopScoreboardPoll,
    triggerRecentTxRefresh,
    fetchRecentTransmissions,
    fetchLevelConfig,
    // restored helpers
    setTopLinks,
    loadTalkerHistory,
    startTickTimer,
    stopTickTimer
  }
})

