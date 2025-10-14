<template>
  <Card :title="cardTitle">
    <div class="source-node-info">
      <div class="node-header">
        <div class="header-right">
          <div class="link-counters">
            <div class="counter">
              <div class="counter-label">Adjacent</div>
              <div class="counter-value">{{ adjacentCount }}</div>
            </div>
            <div class="counter">
              <div class="counter-label">Links</div>
              <div class="counter-value">{{ totalLinks }}</div>
            </div>
          </div>
          <div class="status-indicators">
            <div class="indicator" :class="{ active: txActive }">
              <span class="dot"></span>
              <span>TX</span>
            </div>
            <div class="indicator" :class="{ active: rxActive }">
              <span class="dot"></span>
              <span>RX</span>
            </div>
            <div class="indicator renown-indicator" v-if="nodeStore.renownEnabled">
              <span>⭐</span>
              <span class="renown-text">Renown {{ nodeStore.renownXPPerLevel ? '~' + Math.round(nodeStore.renownXPPerLevel/3600) + 'h' : '' }}</span>
            </div>
          </div>
          <div class="settings-area">
            <button @click="txNotif.openSettings()" class="settings-btn" title="Notification Settings">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
                <circle cx="12" cy="12" r="3"></circle>
              </svg>
            </button>
            <button v-if="txNotif.notificationsEnabled.value && txNotif.notificationPermission.value === 'default'" @click="txNotif.openSettings()" class="cta-btn" title="Enable browser notifications">
              🔓 Enable Browser Notifications
            </button>
            <button v-if="txNotif.notificationsEnabled.value && (txNotif.soundEnabled.value || txNotif.speechEnabled.value) && txNotif.audioSuspended.value" @click="txNotif.openAudio()" class="cta-btn audio-cta" title="Enable sound and speech">
              🔊 Enable Sound & Speech
            </button>
            <button class="btn-secondary" @click="showHelp = true" title="How leveling works">?</button>
          </div>
        </div>
      </div>

      <!-- Notification Settings Panel -->
      <div v-if="txNotif.showSettings.value" class="notification-settings">
        <div class="setting-row">
          <label class="setting-label">
            <input type="checkbox" v-model="txNotif.notificationsEnabled.value" @change="txNotif.onNotificationToggle" />
          </label>
        </div>
        <div v-if="txNotif.notificationsEnabled.value" class="setting-row">
          <label class="setting-label">
            <span>Cooldown Period:</span>
            <select v-model="txNotif.notificationCooldown.value" class="cooldown-select">
              <option :value="30">30 seconds</option>
              <option :value="60">1 minute</option>
              <option :value="120">2 minutes</option>
              <option :value="300">5 minutes</option>
              <option :value="600">10 minutes</option>
              <option :value="3">Debug Mode</option>
            </select>
          </label>
          <p class="setting-help">System must be idle (no TX) for this long before you'll be notified of new activity</p>
        </div>
        <div v-if="txNotif.notificationsEnabled.value" class="setting-row">
          <label class="setting-label">
            <input type="checkbox" v-model="txNotif.soundEnabled.value" />
            <span>Play notification sound</span>
          </label>
        </div>
        <div v-if="txNotif.notificationsEnabled.value && txNotif.soundEnabled.value" class="setting-row">
          <label class="setting-label">
            <span>Sound Type:</span>
            <select v-model="txNotif.soundType.value" class="sound-select">
              <option value="two-tone">Two-Tone Beep</option>
              <option value="single-beep">Single Beep</option>
              <option value="triple-beep">Triple Beep</option>
              <option value="ascending">Ascending Tones</option>
              <option value="descending">Descending Tones</option>
              <option value="chirp">Chirp</option>
            </select>
          </label>
        </div>
        <div v-if="txNotif.notificationsEnabled.value && txNotif.soundEnabled.value" class="setting-row">
          <label class="setting-label volume-control">
            <span>🔊 Volume:</span>
            <input
              type="range"
              v-model="txNotif.soundVolume.value"
              min="0"
              max="100"
              class="volume-slider"
            />
            <span class="volume-value">{{ txNotif.soundVolume.value }}%</span>
          </label>
        </div>
        <div v-if="txNotif.notificationsEnabled.value" class="setting-row">
          <label class="setting-label">
            <input type="checkbox" v-model="txNotif.speechEnabled.value" />
            <span>Use speech synthesis</span>
          </label>
        </div>
        <div v-if="txNotif.notificationsEnabled.value" class="setting-row button-row">
          <button @click="txNotif.sendTestNotification" class="test-notification-btn">
            🔔 Test Notification
          </button>
          <button v-if="txNotif.soundEnabled.value" @click="txNotif.playNotificationSound" class="test-notification-btn secondary">
            🔊 Test Sound
          </button>
          <button v-if="txNotif.speechEnabled.value" @click="txNotif.testSpeech" class="test-notification-btn secondary">
            🗣️ Test Speech
          </button>
        </div>
        <div v-if="txNotif.notificationsEnabled.value && txNotif.notificationPermission.value === 'default'" class="setting-row">
          <button @click="txNotif.requestPermission()" class="test-notification-btn">
            🔓 Request Browser Permission
          </button>
        </div>
        <div v-if="txNotif.notificationsEnabled.value && txNotif.cooldownRemaining.value > 0" class="cooldown-status">
          ⏱️ Cooldown active: {{ txNotif.formatCooldownTime(txNotif.cooldownRemaining.value) }} remaining
        </div>
        <div v-if="txNotif.notificationsEnabled.value && (txNotif.soundEnabled.value || txNotif.speechEnabled.value) && txNotif.audioSuspended.value" class="setting-row">
          <button @click="txNotif.openAudio()" class="test-notification-btn">
            🔊 Enable Sound & Speech (resume audio)
          </button>
          <p class="setting-help">Audio playback is suspended by the browser until you interact. Click to resume audio and play a test.</p>
        </div>
        <div v-if="txNotif.notificationsEnabled.value" class="debug-info">
          <small style="color: var(--text-muted);">
            Debug: Permission={{ txNotif.notificationPermission.value }},
            LastNotif=0,
            Cooldown={{ txNotif.cooldownRemaining.value }}s
          </small>
        </div>
        <div v-if="txNotif.notificationPermission.value === 'denied'" class="notification-warning">
          ⚠️ Notifications are blocked. Please enable them in your browser settings.
        </div>
      </div>

      <div v-if="adjacentList.length === 0" class="no-links">
        No adjacent links
      </div>

      <div v-else class="adjacent-links-table">
        <table>
          <thead>
            <tr>
              <th>Node</th>
              <th>Callsign</th>
              <th>Description</th>
              <th>Status</th>
              <th class="hide-mobile">Mode</th>
              <th class="hide-mobile">Direction</th>
              <th class="hide-mobile">IP</th>
              <th class="hide-mobile">Connected</th>
              <th v-if="showLostColumn">Lost</th>
              <th>Duration</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="node in adjacentList" :key="node.NodeID" :class="{ transmitting: node.IsTransmitting, stale: node._stale, 'newly-connected': node.IsNew }">
              <td>
                <a v-if="node.NodeID >= 0"
                   :href="`https://stats.allstarlink.org/stats/${node.NodeID}`"
                   target="_blank"
                   rel="noopener noreferrer"
                   class="node-link">
                  {{ node.NodeID }}
                </a>
                <span v-else class="text-node">
                  {{ node.Callsign || node.NodeID }}
                </span>
              </td>
              <td>
                <a v-if="node.Callsign"
                   :href="`https://www.qrz.com/db/${node.Callsign.toUpperCase()}`"
                   target="_blank"
                   rel="noopener noreferrer"
                   class="callsign-link">
                  {{ node.Callsign }}
                  <span v-if="getCallsignGrouping(node.Callsign)" class="callsign-badge">{{ getCallsignGrouping(node.Callsign).badge }}</span>
                </a>
                <span v-else class="no-data">-</span>
              </td>
              <td class="description-cell">
                <div v-if="node.Description || node.Location" class="description-wrapper">
                  <div v-if="node.Description" class="desc-line">{{ node.Description }}</div>
                  <div v-if="node.Location" class="loc-line">{{ node.Location }}</div>
                </div>
                <span v-else class="no-data">-</span>
              </td>
              <td>
                <span class="status-badge" :class="getStatusClass(node)">
                  {{ getStatusText(node) }}
                </span>
              </td>
              <td class="hide-mobile">{{ node.Mode || '-' }}</td>
              <td class="hide-mobile">{{ node.Direction || '-' }}</td>
              <td class="ip-cell hide-mobile">{{ node.IP || '-' }}</td>
              <td class="time-cell hide-mobile">{{ formatConnectedTime(node.ConnectedSince) }}</td>
              <td v-if="showLostColumn" class="time-cell">{{ formatLostTime(node.RemovedAt) }}</td>
              <td class="time-cell">
                <span v-if="node.IsTransmitting && node.KeyedStartTime">
                  {{ formatDuration(node.KeyedStartTime) }}
                </span>
                <span v-else-if="node.TotalTxSeconds > 0">
                  {{ formatSeconds(node.TotalTxSeconds) }} total
                </span>
                <span v-else class="no-data">-</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </Card>
  <LevelingHelpModal
    :visible="showHelp"
    :levelConfig="nodeStore.levelConfig"
    :renownXP="nodeStore.renownXPPerLevel"
    :renownEnabled="nodeStore.renownEnabled"
    :weeklyCapSeconds="nodeStore.weeklyCapSeconds"
    :restedEnabled="nodeStore.restedEnabled"
    :restedAccumulationRate="nodeStore.restedAccumulationRate"
    :restedMaxHours="nodeStore.restedMaxHours"
    :restedMultiplier="nodeStore.restedMultiplier"
    @close="showHelp = false"
  />
</template>

<script setup>
import { computed, watch, reactive, ref, onMounted, onUnmounted } from 'vue'
import LevelingHelpModal from './LevelingHelpModal.vue'
import Card from './Card.vue'
import { useNodeStore } from '../stores/node'
import { useTxNotifications } from '../composables/useTxNotifications'
import { cfg as defaultCfg } from '../env'
import { useAuthStore } from '../stores/auth'
import { logger } from '../utils/logger'

const props = defineProps({
  sourceNodeID: {
    type: Number,
    required: true
  },
  data: {
    type: Object,
    required: true
  },
  showLostColumn: {
    type: Boolean,
    required: false,
    default: false
  },
  // Optional injected providers for easier testing/mounting
  nodeStore: { type: Object, required: false },
  txNotifications: { type: Object, required: false },
  cfgProp: { type: Object, required: false }
})

logger.debug('[SourceNodeCard] Component loaded, props:', {
  sourceNodeID: props.sourceNodeID,
  sourceNodeIDType: typeof props.sourceNodeID,
  hasData: !!props.data,
  dataSourceNodeID: props.data?.source_node_id || props.data?.sourceNodeID
})

// Prefer an injected nodeStore for tests; otherwise use the canonical Pinia store
const nodeStore = props.nodeStore || useNodeStore()

const showHelp = ref(false)

// Get the actual source node ID from props or data
const actualSourceNodeID = computed(() => {
  return props.sourceNodeID || props.data?.source_node_id || props.data?.sourceNodeID
})

logger.debug('[SourceNodeCard] Actual source node ID:', actualSourceNodeID.value)

// Initialize TX notifications for this source node; allow injection for tests
const txNotif = props.txNotifications || useTxNotifications(actualSourceNodeID.value)
logger.info('[SourceNodeCard] TX notifications initialized for node:', actualSourceNodeID.value)

// Get display node ID from either prop or data
const displayNodeID = computed(() => {
  return actualSourceNodeID.value || 'Unknown'
})

// Fetch ASTDB info for the source node to show Description and Location in the card title
const sourceNodeInfo = ref(null)

async function fetchSourceNodeInfo(nodeId) {
  if (!nodeId) {
    sourceNodeInfo.value = null
    return
  }
  try {
    const res = await fetch(`/api/node-lookup?q=${nodeId}`)
    const data = await res.json()
    const results = (data && data.results) || (data && data.data && data.data.results) || []
    sourceNodeInfo.value = results && results.length > 0 ? results[0] : null
  } catch (e) {
  logger.error('[SourceNodeCard] Failed to fetch node info:', e)
    sourceNodeInfo.value = null
  }
}

watch(() => actualSourceNodeID.value, (nodeId) => {
  fetchSourceNodeInfo(nodeId)
}, { immediate: true })

// Title format: node number - description - location
const cardTitle = computed(() => {
  const id = displayNodeID.value
  const desc = sourceNodeInfo.value?.description?.trim()
  const loc = sourceNodeInfo.value?.location?.trim()
  if (desc && loc) return `${id} - ${desc} - ${loc}`
  if (desc) return `${id} - ${desc}`
  return `${id}`
})

// Local caches to track recently removed adjacent nodes and last-seen data
const lastKnownNodes = reactive({}) // nodeID -> raw object
const recentRemoved = reactive({}) // nodeID -> { raw, removedAt }
// Use the Pinia ref value for persistent per-node first-seen timestamps so components
// can show steadily-increasing connected timers even across remounts.
// nodeStore.connectionSeenAt is a Ref, so use its .value (fall back to a local reactive map).
const seenAt = (nodeStore.connectionSeenAt && nodeStore.connectionSeenAt.value) ? nodeStore.connectionSeenAt.value : reactive({}) // nodeID -> first-seen timestamp (ms)

// Provide cfg from injected prop or default
const cfg = props.cfgProp || defaultCfg

// Convert adjacent nodes map to sorted array - MUST BE DEFINED BEFORE WATCH
const adjacentList = computed(() => {
  // Touch the reactive nowTick so this computed re-runs every second and Lost/Connected timers tick.
  const _tick = nodeStore.nowTick
  const now = (typeof _tick === 'number' ? _tick : (_tick && _tick.value)) || Date.now()
  const removeExpiryMs = (cfg && cfg.STALE_RETENTION_MS) ? cfg.STALE_RETENTION_MS : 5 * 60 * 1000 // keep removed nodes for configured retention

  const currentMap = (props.data && props.data.adjacentNodes) || {}
  // Update lastKnownNodes with current data and record seenAt when we first observe a node
  for (const [k, v] of Object.entries(currentMap)) {
    try {
      const id = String(v.NodeID || v.node || v.node_id || v.Node || v.NodeId)
      if (id) {
        lastKnownNodes[id] = v
        // If we haven't recorded a seenAt timestamp for this node, set it now
        if (!seenAt[id]) seenAt[id] = now
      }
    } catch (e) {
      // ignore
    }
  }

  // Prune expired recentRemoved entries (>= retention); keep persisted tombstone so we can skip re-adding
  for (const [id, info] of Object.entries(recentRemoved)) {
    const age = now - (info.removedAt || 0)
    if (age >= removeExpiryMs) {
      delete recentRemoved[id]
    }
  }

  // Build a combined list: current nodes + recentRemoved (not expired)
  const combined = {}
  for (const [k, v] of Object.entries(currentMap)) combined[k] = { raw: v, _stale: false }
  for (const [id, info] of Object.entries(recentRemoved)) {
    if (!combined[id]) combined[id] = { raw: info.raw, _stale: true, removedAt: info.removedAt || now }
  }

  // Build set of global links from nodeStore.status to detect global removals
  const globalSet = new Set()
  try {
    const addCandidate = (cand) => {
      if (cand == null) return
      // cand may be an object with Node/node fields or a raw numeric id
      const maybeNum = Number(cand && (cand.Node ?? cand.node ?? cand.node_id ?? cand))
      if (!isNaN(maybeNum)) globalSet.add(maybeNum)
    }

    // Prefer the canonical store.links (kept up-to-date by STATUS_UPDATE/LINK_ADDED/LINK_REMOVED)
    const linksMaybe = (nodeStore.links && nodeStore.links.value) ? nodeStore.links.value : nodeStore.links
    if (Array.isArray(linksMaybe)) {
      for (const l of linksMaybe) addCandidate(l)
    }

    // Also include any ids present in the status snapshot (links_detailed or links)
    const statusMaybe = (nodeStore.status && nodeStore.status.value) ? nodeStore.status.value : nodeStore.status
    if (Array.isArray(statusMaybe.links_detailed)) {
      for (const ld of statusMaybe.links_detailed) addCandidate(ld)
    } else if (Array.isArray(statusMaybe.links)) {
      for (const id of statusMaybe.links) addCandidate(id)
    }
  } catch (e) {
    // ignore
  }

  // Resilience: if we have a lastKnownNodes entry that is not in the current map and
  // not already in recentRemoved, but is absent from the global links set, treat it as removed.
  // Honor the retention window: if a persisted removedAt exists and has expired, do not resurrect.
  try {
    for (const id of Object.keys(lastKnownNodes)) {
      if (combined[id]) continue
      if (recentRemoved[id]) continue
      const last = lastKnownNodes[id]
      const nid = Number(id)
      if (!isNaN(nid) && globalSet.size > 0 && !globalSet.has(nid)) {
        const sid = Number(actualSourceNodeID.value || props.data?.source_node_id || props.data?.sourceNodeID || 0)
        let persistedTs = null
        try { if (typeof nodeStore.getRemovedAt === 'function') persistedTs = nodeStore.getRemovedAt(sid, nid) } catch (e) {}
        const ts = persistedTs || now
        // If persisted removal timestamp exists and is beyond or equal to retention, skip re-adding
        if (persistedTs && (now - persistedTs >= removeExpiryMs)) {
          continue
        }
        // record removal (respecting persisted timestamp)
        recentRemoved[id] = { raw: last, removedAt: ts }
        combined[id] = { raw: last, _stale: true, removedAt: ts }
  logger.debug('[SourceNodeCard] resilient mark removed', { id, removedAt: now })
      }
    }
  } catch (e) {
    // ignore
  }

  const normalized = Object.entries(combined)
    .filter(([id, n]) => n && n.raw)
    .map(([id, entry]) => {
  // debug: per-row normalize start (silenced to reduce console spam)
      const raw = entry.raw
      // Normalize removedAt to a numeric ms timestamp when present so formatLostTime can compute diffs reliably.
      if (entry.removedAt) {
        try {
          const maybeNum = Number(entry.removedAt)
          if (!isNaN(maybeNum) && maybeNum > 0) {
            entry.removedAt = maybeNum
          } else {
            const parsed = new Date(entry.removedAt).getTime()
            entry.removedAt = !isNaN(parsed) && parsed > 0 ? parsed : entry.removedAt
          }
        } catch (e) {
          // leave as-is if coercion fails
        }
      }
      // Accept multiple naming conventions (snake_case from server or camelCase from client)
      const nodeID = raw.NodeID || raw.node || raw.node_id || raw.Node || raw.NodeId
      const callsign = raw.Callsign || raw.callsign || raw.node_callsign || raw.nodeCallsign || raw.node_callsign
      const description = raw.Description || raw.description || raw.node_description || raw.nodeDescription
      const location = raw.Location || raw.location || raw.node_location || raw.nodeLocation
      const isTransmitting = raw.IsTransmitting || raw.is_transmitting || raw.current_tx || raw.currentTx || raw.current_tx || false
      const isKeyed = raw.IsKeyed || raw.is_keyed || raw.is_keyed || raw.isKeyed || raw.is_keyed || false
      const keyedStart = raw.KeyedStartTime || raw.keyed_start_time || raw.keyedStartTime || raw.keyed_start || raw.KeyedStart
      // Determine connectedSince; server-provided ConnectedSince preferred. For stale entries use
      // the removal timestamp so the UI shows time-since-lost. For present nodes, use a persistent
      // seenAt timestamp so the counter increments rather than resetting to now on every recompute.
      let connectedSince = raw.ConnectedSince || raw.connected_since || raw.connectedSince || raw.connected_at || null
      // Normalize to milliseconds when provided as string/seconds
      if (connectedSince) {
        const parsed = parseAnyToMs(connectedSince)
        connectedSince = Number.isFinite(parsed) ? parsed : connectedSince
      }
      if (entry._stale) {
        if (entry.removedAt) {
          // use the removedAt (ms since epoch) so formatConnectedTime shows time-since-lost
          connectedSince = entry.removedAt
        } else {
          // fallback to removedAt or now so the UI shows 0s immediately after removal
          connectedSince = entry.removedAt || now
        }
      } else {
        // Node is present in current map.
        // If it was in recentRemoved, this is a reconnection edge: set seenAt once and clear the removal cache.
        if (recentRemoved && recentRemoved[id]) {
          seenAt[id] = now // first-seen after reconnection
          delete recentRemoved[id]
          // Clear persisted removal timestamp for this pair
          try {
            const sid = Number(actualSourceNodeID.value || props.data?.source_node_id || props.data?.sourceNodeID || 0)
            const nid = Number(id)
            if (!isNaN(sid) && !isNaN(nid) && typeof nodeStore.clearRemovedAt === 'function') {
              nodeStore.clearRemovedAt(sid, nid)
            }
          } catch (e) {}
        }
        if (connectedSince) {
          // server supplied timestamp - already normalized
        } else if (seenAt[id]) {
          // use previously recorded first-seen timestamp so the counter increments
          connectedSince = seenAt[id]
        } else {
          // first time we see this node and no server timestamp; record and use now
          seenAt[id] = now
          connectedSince = now
        }
      }
      const mode = raw.Mode || raw.mode || null
      const direction = raw.Direction || raw.direction || raw.dir || null
      const ip = raw.IP || raw.ip || null
      const totalTxSeconds = raw.TotalTxSeconds || raw.total_tx_seconds || raw.total_tx || raw.totalTxSeconds || 0
      let isStale = !!entry._stale
      // If this node is not present in the global links set, mark it stale (node likely disconnected)
      try {
        const nid = Number(nodeID)
        const nidKey = String(nodeID)
        if (!isNaN(nid) && globalSet.size > 0 && !globalSet.has(nid)) {
          isStale = true
          // Prefer persisted removal timestamp from store for stability across reloads
          const sid = Number(actualSourceNodeID.value || props.data?.source_node_id || props.data?.sourceNodeID || 0)
          const persisted = (typeof nodeStore.getRemovedAt === 'function') ? nodeStore.getRemovedAt(sid, nid) : null
          const cached = recentRemoved[nidKey]
          const ts = persisted || (cached && cached.removedAt) || null
          if (ts) {
            entry.removedAt = ts
            recentRemoved[nidKey] = { raw: raw, removedAt: ts }
          }
        }
      } catch (e) {
        // ignore
      }
      // If entry is stale and past retention, cull it (even if currentMap still contains it)
      try {
        if (isStale) {
          const sid = Number(actualSourceNodeID.value || props.data?.source_node_id || props.data?.sourceNodeID || 0)
          const nid = Number(nodeID)
          const tsNum = Number(entry.removedAt || ((typeof nodeStore.getRemovedAt === 'function') ? nodeStore.getRemovedAt(sid, nid) : null))
          if (!isNaN(tsNum) && tsNum > 0 && (now - tsNum) >= removeExpiryMs) {
            // Clear persisted and cache, then cull by returning null
            try { delete recentRemoved[String(nodeID)] } catch (e) {}
            return null
          }
          // If we lack a timestamp entirely while marked stale, cull as well (avoid resetting timer)
          if (isNaN(tsNum) || tsNum <= 0) {
            try { delete recentRemoved[String(nodeID)] } catch (e) {}
            return null
          }
        }
      } catch (e) {}
      // Newly connected if connectedSince is within 60s
      let isNew = false
      try {
        if (connectedSince) {
          const cs = new Date(connectedSince).getTime()
          const newWindow = (cfg && cfg.NEW_NODE_HIGHLIGHT_MS) ? cfg.NEW_NODE_HIGHLIGHT_MS : 60 * 1000
          if (!isNaN(cs) && (now - cs) < newWindow) isNew = true
        }
      } catch (e) {
        isNew = false
      }

      const numericNodeID = Number(nodeID)
      return {
        NodeID: isNaN(numericNodeID) ? nodeID : numericNodeID,
        Callsign: callsign || '',
        Description: description || '',
        Location: location || '',
        IsTransmitting: !!isTransmitting,
        IsKeyed: !!isKeyed,
    KeyedStartTime: keyedStart || null,
  ConnectedSince: connectedSince || null,
    RemovedAt: entry.removedAt || null,
        Mode: mode || null,
        Direction: direction || null,
        IP: ip || null,
        TotalTxSeconds: totalTxSeconds || 0
        ,
        _stale: isStale,
        IsNew: isNew
      }
    })
    .filter(node => node && node.NodeID)
    .sort((a, b) => {
      if (a.IsTransmitting && !b.IsTransmitting) return -1
      if (!a.IsTransmitting && b.IsTransmitting) return 1
      // Ensure numeric comparison when possible
      const na = Number(a.NodeID)
      const nb = Number(b.NodeID)
      if (!isNaN(na) && !isNaN(nb)) return na - nb
      if (a.NodeID < b.NodeID) return -1
      if (a.NodeID > b.NodeID) return 1
      return 0
    })

  return normalized
})

// render LevelingHelpModal with authoritative server values
// (placed after the computed block so setup above is available)

// Watch for removed nodes and keep them visible in recentRemoved
try {
  watch(() => props.data && props.data.adjacentNodes, (newMap, oldMap) => {
    // Use a fixed timestamp when recording removals (don't use reactive nowTick here)
    const now = Date.now()
    const newIds = newMap ? Object.keys(newMap).map(String) : []
    const oldIds = oldMap ? Object.keys(oldMap).map(String) : []
    for (const id of oldIds) {
      if (!newIds.includes(id)) {
        // node was removed; capture its last known data
        const last = lastKnownNodes[id]
        if (last) {
          recentRemoved[id] = { raw: last, removedAt: now }
          // Persist removal timestamp for durability across reloads
          try {
            const sid = Number(actualSourceNodeID.value || props.data?.source_node_id || props.data?.sourceNodeID || 0)
            const nid = Number(id)
            if (!isNaN(sid) && !isNaN(nid) && typeof nodeStore.setRemovedAt === 'function') {
              nodeStore.setRemovedAt(sid, nid, now)
            }
          } catch (e) {}
          logger.debug('[SourceNodeCard] recentRemoved set', { id, removedAt: now, last })
        }
      }
    }
  }, { immediate: false })
} catch (e) {
  logger.debug('[SourceNodeCard] failed to install removed-node watcher', e)
}

// Debug: log TX/RX state when it changes
try {
  logger.info('[SourceNodeCard] Setting up TX/RX watch')
  watch(() => [props.data.txKeyed, props.data.rxKeyed], ([tx, rx]) => {
  logger.debug('[SourceNodeCard] TX/RX state:', { tx, rx, sourceNodeID: actualSourceNodeID.value })
  }, { immediate: true })
  logger.info('[SourceNodeCard] TX/RX watch set up successfully')
} catch (err) {
  logger.error('[SourceNodeCard] Failed to set up TX/RX watch:', err)
}

// Watch for any adjacent node transmitting and trigger notifications
try {
  logger.info('[SourceNodeCard] Setting up TX notification watch')

  // Watch adjacency/transmitting state but defer invoking the composable until it's initialized
  watch(
    () => {
      try {
        const list = adjacentList.value
        const result = list.some(n => n.IsTransmitting)
  logger.debug('[TxWatch] Watch running, sourceNode:', actualSourceNodeID.value, 'adjacentList:', list.length, 'anyTx:', result)
        return result
      } catch (err) {
  logger.error('[TxWatch] Error in watch source:', err)
        return false
      }
    },
    (isAnyTransmitting, wasTransmitting) => {
      try {
  logger.info('[TxWatch] TX state changed:', {
          sourceNodeID: actualSourceNodeID.value,
          isAnyTransmitting,
          wasTransmitting,
          adjacentCount: adjacentList.value.length,
          txNodes: adjacentList.value.filter(n => n.IsTransmitting).map(n => ({ id: n.NodeID, callsign: n.Callsign }))
        })

        if (isAnyTransmitting) {
          const txNode = adjacentList.value.find(n => n.IsTransmitting)
          if (txNode) {
            logger.debug('[TxWatch] Calling watchTxState(true) with node:', txNode.NodeID, txNode.Callsign)
            txNotif.watchTxState(true, txNode)
          }
        } else {
          logger.debug('[TxWatch] Calling watchTxState(false) - all nodes idle')
          txNotif.watchTxState(false, {})
        }
      } catch (err) {
  logger.error('[TxWatch] Error in watch callback:', err)
      }
    }
  )

  logger.info('[SourceNodeCard] TX notification watch set up successfully')
} catch (err) {
  logger.error('[SourceNodeCard] Failed to set up TX notification watch:', err)
}

// Incremental enrichment backoff for rows missing Mode/Direction/IP
// We'll schedule fetch attempts at 5s, 10s, 15s, 30s until enriched.
const enrichmentTimers = reactive({}) // nodeID -> { attempt, timeoutId }
function scheduleEnrichment(node) {
  try {
    const id = String(node.NodeID)
    if (!id) return
    const needs = !node.Mode || !node.Direction || !node.IP
    if (!needs) {
      // If previously scheduled, clear
      if (enrichmentTimers[id] && enrichmentTimers[id].timeoutId) {
        clearTimeout(enrichmentTimers[id].timeoutId)
        delete enrichmentTimers[id]
      }
      return
    }
    const prev = enrichmentTimers[id] || { attempt: 0, timeoutId: null }
    if (prev.timeoutId) return // already scheduled
    const scheduleDelays = [5000, 10000, 15000, 30000]
    const attempt = Math.min(prev.attempt + 1, scheduleDelays.length)
    const delay = scheduleDelays[attempt - 1]
    const timeoutId = setTimeout(async () => {
      // On timer: request a targeted server poll for this source node
      try {
        const auth = useAuthStore()
        const headers = auth.getAuthHeaders()
        const sid = Number(actualSourceNodeID.value || props.data?.source_node_id || props.data?.sourceNodeID || 0)
        if (sid > 0) {
          await fetch(`/api/poll-now?node=${sid}`, { method: 'POST', headers }).catch(() => {})
        }
      } catch (e) {}
      // Clear timer and, if still missing, schedule next attempt
      enrichmentTimers[id] = { attempt, timeoutId: null }
      // Re-evaluate current node data from adjacentList
      const current = adjacentList.value.find(n => String(n.NodeID) === id)
      if (current && (!current.Mode || !current.Direction || !current.IP)) {
        scheduleEnrichment(current)
      } else {
        delete enrichmentTimers[id]
      }
    }, delay)
    enrichmentTimers[id] = { attempt, timeoutId }
  } catch (e) {
    // ignore
  }
}

// Watch the adjacentList and schedule enrichment when needed
watch(() => adjacentList.value.map(n => ({ id: n.NodeID, Mode: n.Mode, Direction: n.Direction, IP: n.IP })), (rows) => {
  for (const r of rows) scheduleEnrichment(r)
}, { immediate: true, deep: true })

onUnmounted(() => {
  // Cleanup timers
  for (const k of Object.keys(enrichmentTimers)) {
    const t = enrichmentTimers[k]
    if (t && t.timeoutId) clearTimeout(t.timeoutId)
    delete enrichmentTimers[k]
  }
})

// Adjacent count derived from provided adjacentNodes (RPT_ALINKS) or links_detailed
const adjacentCount = computed(() => {
  // Prefer explicit num_alinks when provided (server-side snapshot)
  if (props.data && typeof props.data.num_alinks === 'number') return props.data.num_alinks
  if (!props.data) return 0
  if (props.data.adjacentNodes) return Object.keys(props.data.adjacentNodes).length
  // Fallback: if links_detailed provided for this source, count entries
  if (props.data.links_detailed) return props.data.links_detailed.length
  return 0
})

// Total links (per-source): prefer server-provided per-source num_links; else fall back to Adjacent
const totalLinks = computed(() => {
  const adj = (adjacentCount && adjacentCount.value) ? adjacentCount.value : 0
  if (props.data && typeof props.data.num_links === 'number') return props.data.num_links
  return adj
})

// TX active when source node reports TX; prefer several possible field names
const txActive = computed(() => {
  if (!props.data) return false
  return !!(props.data.tx_keyed || props.data.txKeyed || props.data.tx)
})

// RX active when any adjacent node reports keyed/transmitting
const rxActive = computed(() => {
  if (!props.data) return false
  const nodes = props.data.adjacentNodes || props.data.links_detailed || {}
  return Object.values(nodes).some(n => n && (n.IsKeyed || n.IsTransmitting || n.keyed))
})

function parseAnyToMs(v) {
  if (v == null) return NaN
  // Numeric epoch (seconds or ms)
  if (typeof v === 'number') {
    // Heuristic: < 1e12 => seconds, else milliseconds
    return v < 1e12 ? v * 1000 : v
  }
  if (typeof v === 'string') {
    let s = v.trim()
    // Pure digits => epoch seconds/ms
    if (/^\d+$/.test(s)) {
      const n = Number(s)
      return n < 1e12 ? n * 1000 : n
    }
    // Append Z if naive ISO-like string (treat as UTC)
    const hasTZ = /Z|[+\-]\d{2}:?\d{2}/i.test(s)
    let iso = hasTZ ? s : `${s}Z`
    // Normalize fractional seconds longer than milliseconds to 3 digits
    // Go backend returns timestamps with microsecond precision (6 digits after decimal)
    // but JavaScript Date.parse() can have issues with non-standard precision in some browsers.
    // This normalizes to standard millisecond precision (3 digits).
    // Examples:
    //   2025-10-12T22:12:14.822791-04:00 -> 2025-10-12T22:12:14.822-04:00
    //   2025-10-12T22:12:14.822791Z -> 2025-10-12T22:12:14.822Z
    iso = iso.replace(/(\.\d{3})\d+([Zz]|[+\-]\d{2}:?\d{2})$/, '$1$2')
    const ms = Date.parse(iso)
    return Number.isFinite(ms) ? ms : NaN
  }
  const d = new Date(v).getTime()
  return Number.isFinite(d) ? d : NaN
}

function formatConnectedTime(timestamp) {
  if (!timestamp) return '-'
  const nowRef = nodeStore.nowTick
  const now = (typeof nowRef === 'number') ? nowRef : ((nowRef && nowRef.value) || Date.now())
  let connectedAt = parseAnyToMs(timestamp)
  if (!Number.isFinite(connectedAt)) return '-'
  // If connectedAt is somehow in the future, clamp to now
  if (connectedAt > now) connectedAt = now
  let diffSec = Math.floor((now - connectedAt) / 1000)

  // Guard against absurd durations caused by unit mistakes; attempt a correction once
  // If diff looks like thousands of years, assume timestamp may have been provided in seconds-but-treated-as-ms or vice versa
  const HUNDRED_YEARS_SEC = 100 * 365 * 24 * 3600
  if (diffSec > HUNDRED_YEARS_SEC) {
    // Try reinterpreting original input more strictly
    const raw = typeof timestamp === 'string' ? timestamp.trim() : timestamp
    if (typeof raw === 'number') {
      // If number and >= 1e12, maybe it was milliseconds but parse went wrong; if < 1e12, maybe seconds
      connectedAt = raw < 1e12 ? raw * 1000 : raw
    } else if (typeof raw === 'string' && /^\d+$/.test(raw)) {
      const n = Number(raw)
      connectedAt = n < 1e12 ? n * 1000 : n
    }
    diffSec = Math.floor(((typeof now === 'number' ? now : Date.now()) - connectedAt) / 1000)
  }

  if (!Number.isFinite(diffSec) || diffSec < 0) return '-'
  if (diffSec < 60) return `${diffSec}s`
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m`
  const hours = Math.floor(diffSec / 3600)
  const mins = Math.floor((diffSec % 3600) / 60)
  return `${hours}h ${mins}m`
}

function formatLostTime(removedAt) {
  if (!removedAt) return '-'
  const now = nodeStore.nowTick || Date.now()
  // Coerce removedAt to numeric ms if possible
  let removedTs = null
  try {
    if (typeof removedAt === 'number') removedTs = removedAt
    else if (!isNaN(Number(removedAt))) removedTs = Number(removedAt)
    else {
      const p = new Date(removedAt).getTime()
      removedTs = isNaN(p) ? null : p
    }
  } catch (e) {
    removedTs = null
  }
  // Debugging help: show shapes when Lost stays at 0s
  try {
    const diffCheck = removedTs ? Math.floor(((typeof now === 'number' ? now : (now && now.value) || Date.now()) - removedTs) / 1000) : null
  logger.debug('[SourceNodeCard] formatLostTime', { removedAt, removedTs, now: typeof now === 'number' ? now : (now && now.value), diffCheck })
  } catch (e) {}
  if (!removedTs || removedTs === 0) return '-'
  const diffSec = Math.floor(( (typeof now === 'number' ? now : (now && now.value) || Date.now()) - removedTs) / 1000)
  if (diffSec < 60) return `${diffSec}s ago`
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`
  const hours = Math.floor(diffSec / 3600)
  const mins = Math.floor((diffSec % 3600) / 60)
  return `${hours}h ${mins}m ago`
}

function formatDuration(startTime) {
  if (!startTime) return '-'
  const nowRef = nodeStore.nowTick
  const now = (typeof nowRef === 'number') ? nowRef : ((nowRef && nowRef.value) || Date.now())
  const start = parseAnyToMs(startTime)
  if (!Number.isFinite(start)) return '-'
  const diffSec = Math.floor((now - start) / 1000)

  if (diffSec < 60) return `${diffSec}s`
  if (diffSec < 3600) {
    const mins = Math.floor(diffSec / 60)
    const secs = diffSec % 60
    return `${mins}m ${secs}s`
  }
  const hours = Math.floor(diffSec / 3600)
  const mins = Math.floor((diffSec % 3600) / 60)
  return `${hours}h ${mins}m`
}

function formatSeconds(seconds) {
  if (!seconds || seconds === 0) return '0s'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}m ${secs}s`
  }
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m`
}

function getStatusClass(node) {
  // Show keyed if transmitting, regardless of PendingUnkey state
  if (node.IsTransmitting) return 'keyed'
  return 'idle'
}

function getStatusText(node) {
  if (node.IsTransmitting) return 'Keyed'
  return 'Idle'
}

// Create a map of callsign -> grouping for quick lookups
const callsignGroupings = ref({})

// Get grouping info for a callsign from the scoreboard
function getCallsignGrouping(callsign) {
  if (!callsign) return null
  const normalized = callsign.toUpperCase()
  return callsignGroupings.value[normalized] || null
}

// Watch scoreboard and build callsign -> grouping map
watch(() => nodeStore.scoreboard, (newScoreboard) => {
  if (!newScoreboard || !Array.isArray(newScoreboard)) return
  const map = {}
  for (const entry of newScoreboard) {
    if (entry.callsign && entry.grouping) {
      map[entry.callsign.toUpperCase()] = entry.grouping
    }
  }
  callsignGroupings.value = map
}, { immediate: true, deep: true })
</script>

<style scoped>
.source-node-info {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.node-header {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding: 0.75rem;
  background: var(--bg-tertiary);
  border-radius: 6px;
}

.status-indicators {
  display: flex;
  gap: 1rem;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.link-counters {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.counter {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  background: var(--bg-secondary);
}

.counter-label {
  font-size: 0.7rem;
  color: var(--text-secondary);
}

.counter-value {
  font-weight: 700;
  color: var(--accent-primary);
  font-size: 1.05rem;
}

.indicator {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border-radius: 4px;
  background: var(--bg-secondary);
  font-size: 0.875rem;
  color: var(--text-muted);
}

.indicator.active {
  background: var(--accent-primary);
  color: white;
  animation: pulse-indicator 2s ease-in-out infinite;
}

@keyframes pulse-indicator {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.85;
    transform: scale(1.02);
  }
}

.indicator .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.no-links {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-style: italic;
}

.adjacent-links-table {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

thead {
  background: var(--bg-tertiary);
}

th {
  padding: 0.75rem;
  text-align: left;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 2px solid var(--border-color);
}

tbody tr {
  border-bottom: 1px solid var(--border-color);
  transition: background 0.15s;
}

tbody tr:hover {
  background: var(--bg-tertiary);
}

tbody tr.transmitting {
  background: rgba(var(--accent-rgb), 0.1);
}

td {
  padding: 0.75rem;
}

.node-link,
.callsign-link {
  color: var(--accent-primary);
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.node-link:hover,
.callsign-link:hover {
  color: var(--accent-hover);
  text-decoration: underline;
}

.callsign-badge {
  font-size: 1.1rem;
  line-height: 1;
  display: inline-block;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.625rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-badge.keyed {
  background: var(--accent-primary);
  color: white;
}

.status-badge.idle {
  background: var(--bg-tertiary);
  color: var(--text-muted);
}

.tx-active {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  color: var(--accent-primary);
  font-weight: 600;
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-primary);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(1.2);
  }
}

.tx-idle {
  color: var(--text-muted);
}

.stale {
  opacity: 0.55;
  background: linear-gradient(90deg, rgba(200,200,200,0.03), rgba(200,200,200,0.01));
}

.newly-connected {
  font-weight: 700;
  background: rgba(76,175,80,0.06); /* subtle green */
  transition: background 0.5s ease-in-out;
}

.time-cell {
  font-variant-numeric: tabular-nums;
  color: var(--text-secondary);
}

.description-cell {
  max-width: 200px;
}

.description-cell .description-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.description-cell .desc-line {
  color: var(--text-primary);
}

.description-cell .loc-line {
  color: var(--text-secondary);
  font-style: italic;
  font-size: 0.85em;
}

.ip-cell {
  font-family: monospace;
  font-size: 0.875rem;
}

.no-data {
  color: var(--text-muted);
}

/* Scrollbar styling */
.adjacent-links-table::-webkit-scrollbar {
  height: 8px;
}

.adjacent-links-table::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 4px;
}

.adjacent-links-table::-webkit-scrollbar-thumb {
  background: var(--border-hover);
  border-radius: 4px;
}

.adjacent-links-table::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* Settings Button */
.settings-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
  margin-left: 0.75rem;
}

.settings-btn:hover {
  background: var(--bg-secondary);
  border-color: var(--border-hover);
  color: var(--accent-primary);
}

/* Notification Settings Panel */
.notification-settings {
  background: var(--bg-tertiary);
  border-radius: 8px;
  padding: 1.25rem;
  margin-top: 1rem;
  border: 1px solid var(--border-color);
}

.setting-row {
  margin-bottom: 1rem;
}

.setting-row:last-child {
  margin-bottom: 0;
}

.setting-label {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  font-size: 0.9375rem;
  color: var(--text-primary);
  cursor: pointer;
}

.setting-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.setting-label select {
  margin-left: auto;
  padding: 0.375rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.875rem;
  cursor: pointer;
}

.setting-help {
  font-size: 0.8125rem;
  color: var(--text-muted);
  margin-top: 0.375rem;
  margin-left: 1.875rem;
  font-style: italic;
}

.volume-control {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 0.75rem;
  align-items: center;
}

.volume-slider {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 6px;
  border-radius: 3px;
  background: var(--bg-secondary);
  outline: none;
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--accent-primary);
  cursor: pointer;
  transition: transform 0.15s;
}

.volume-slider::-webkit-slider-thumb:hover {
  transform: scale(1.1);
}

.volume-value {
  min-width: 3.5rem;
  text-align: right;
  font-weight: 600;
  color: var(--accent-primary);
}

.button-row {
  display: flex;
  gap: 0.625rem;
  flex-wrap: wrap;
}

.test-notification-btn {
  flex: 1;
  min-width: 140px;
  padding: 0.625rem 1rem;
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.test-notification-btn:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
}

.test-notification-btn.secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.test-notification-btn.secondary:hover {
  background: var(--bg-tertiary);
  border-color: var(--border-hover);
}

.cooldown-status {
  padding: 0.75rem;
  background: rgba(var(--accent-rgb), 0.1);
  border-left: 3px solid var(--accent-primary);
  border-radius: 4px;
  font-size: 0.875rem;
  color: var(--text-primary);
  margin-top: 0.75rem;
}

.debug-info {
  margin-top: 0.75rem;
  padding: 0.625rem;
  background: var(--bg-secondary);
  border-radius: 4px;
  font-family: monospace;
  font-size: 0.75rem;
}

.notification-warning {
  padding: 0.75rem;
  background: rgba(255, 152, 0, 0.1);
  border-left: 3px solid #ff9800;
  border-radius: 4px;
  font-size: 0.875rem;
  color: var(--text-primary);
  margin-top: 0.75rem;
}

/* Responsive: Hide columns on mobile/smaller screens */
@media (max-width: 767px) {
  .hide-mobile {
    display: none !important;
  }
}

/* Also hide on portrait orientation even if slightly larger screen */
@media (max-width: 1024px) and (orientation: portrait) {
  .hide-mobile {
    display: none !important;
  }
}
</style>
