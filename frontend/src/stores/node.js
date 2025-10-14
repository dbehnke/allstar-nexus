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
  const renownEnabled = ref(false)
  const renownXPPerLevel = ref(36000)
  const weeklyCapSeconds = ref(null)
  // Rested server config values (from API)
  const restedEnabled = ref(false)
  const restedAccumulationRate = ref(0) // hours bonus per hour idle
  const restedMaxHours = ref(0)
  const restedMultiplier = ref(1.0)
  const restedIdleThresholdSeconds = ref(null)
  // XP cap and DR config
  const dailyCapSeconds = ref(1200)
  const drTiers = ref([])

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

          // Opportunistic enrichment: if adjacent entries are missing Callsign/Description,
          // fill from the latest STATUS_UPDATE links_detailed when available.
          try {
            const map = snapshot.adjacentNodes
            const st = (status && status.value) ? status.value : {}
            const ld = Array.isArray(st.links_detailed) ? st.links_detailed : []
            if (map && ld && ld.length > 0) {
              // Build a lookup by node id from links_detailed
              const byId = {}
              for (const li of ld) {
                const nid = Number(li.Node || li.node || li.node_id || li.NodeID)
                if (!isNaN(nid)) {
                  byId[nid] = li
                }
              }
              for (const k of Object.keys(map)) {
                const entry = map[k]
                const nid = Number(entry.NodeID || k)
                const link = byId[nid]
                if (link) {
                  // Preserve any existing values; only set when missing/empty
                  const cs = entry.Callsign || entry.callsign
                  const desc = entry.Description || entry.description
                  const loc = entry.Location || entry.location
                  if (!cs) {
                    const fromLD = link.NodeCallsign || link.node_callsign || link.callsign
                    if (fromLD) entry.Callsign = fromLD
                  }
                  if (!desc) {
                    const fromLD = link.NodeDescription || link.node_description || link.description
                    if (fromLD) entry.Description = fromLD
                  }
                  if (!loc) {
                    const fromLD = link.NodeLocation || link.node_location || link.location
                    if (fromLD) entry.Location = fromLD
                  }
                }
              }
            }
          } catch (e) { logger.debug('adjacent enrichment from status failed', e) }

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

  // Helper to safely read numeric/boolean values from API payloads without throwing
  function safeSet(targetRef, getter) {
    try {
      const v = getter()
      if (v !== undefined) targetRef.value = v
    } catch (e) {}
  }

  async function fetchScoreboard(limit = 50) {
    try {
      let headers = {}
      try { const auth = useAuthStore(); headers = (auth && typeof auth.getAuthHeaders === 'function') ? auth.getAuthHeaders() : {} } catch (e) {}
      const res = await fetch(`/api/gamification/scoreboard?limit=${limit}`, { headers })
      const data = await res.json().catch(() => ({}))
      scoreboard.value = (data && (data.scoreboard || data.data || data.results)) || []
      gamificationEnabled.value = !!(data && (data.enabled || data.ok))
      // Capture renown metadata if present
      safeSet(renownEnabled, () => !!data.renown_enabled)
      safeSet(renownXPPerLevel, () => Number(data.renown_xp_per_level) || renownXPPerLevel.value)

      // Capture rested server config if present
      safeSet(restedEnabled, () => !!data.rested_enabled)
      safeSet(restedAccumulationRate, () => Number(data.rested_accumulation_rate) || restedAccumulationRate.value)
      safeSet(restedMaxHours, () => Number(data.rested_max_hours) || restedMaxHours.value)
      safeSet(restedMultiplier, () => Number(data.rested_multiplier) || restedMultiplier.value)
      safeSet(restedIdleThresholdSeconds, () => (data.rested_idle_threshold_seconds != null) ? Number(data.rested_idle_threshold_seconds) : restedIdleThresholdSeconds.value)

      // Capture XP cap and DR config if present
      safeSet(dailyCapSeconds, () => (data.daily_cap_seconds != null) ? Number(data.daily_cap_seconds) : dailyCapSeconds.value)
      safeSet(weeklyCapSeconds, () => (data.weekly_cap_seconds != null) ? Number(data.weekly_cap_seconds) : weeklyCapSeconds.value)
      safeSet(drTiers, () => Array.isArray(data.dr_tiers) ? data.dr_tiers : drTiers.value)
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
      // Capture renown metadata if present in level config response
      safeSet(renownEnabled, () => !!data.renown_enabled)
      safeSet(renownXPPerLevel, () => Number(data.renown_xp_per_level) || renownXPPerLevel.value)
      safeSet(weeklyCapSeconds, () => (data.weekly_cap_seconds != null) ? Number(data.weekly_cap_seconds) : (data.weeklyCapSeconds != null ? Number(data.weeklyCapSeconds) : weeklyCapSeconds.value))
      // Capture rested server config from level-config if present
      safeSet(restedEnabled, () => !!data.rested_enabled)
      safeSet(restedAccumulationRate, () => Number(data.rested_accumulation_rate) || restedAccumulationRate.value)
      safeSet(restedMaxHours, () => Number(data.rested_max_hours) || restedMaxHours.value)
      safeSet(restedMultiplier, () => Number(data.rested_multiplier) || restedMultiplier.value)
      safeSet(restedIdleThresholdSeconds, () => (data.rested_idle_threshold_seconds != null) ? Number(data.rested_idle_threshold_seconds) : restedIdleThresholdSeconds.value)
      // Capture XP cap and DR config from level-config if present
      safeSet(dailyCapSeconds, () => (data.daily_cap_seconds != null) ? Number(data.daily_cap_seconds) : dailyCapSeconds.value)
      safeSet(drTiers, () => Array.isArray(data.dr_tiers) ? data.dr_tiers : drTiers.value)
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
  renownEnabled,
  renownXPPerLevel,
  weeklyCapSeconds,
  dailyCapSeconds,
  restedEnabled,
  restedAccumulationRate,
  restedMaxHours,
  restedMultiplier,
  restedIdleThresholdSeconds,
  drTiers,
    // restored helpers
    setTopLinks,
    loadTalkerHistory,
    startTickTimer,
    stopTickTimer
  }
})

