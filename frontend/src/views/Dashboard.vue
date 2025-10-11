<template>
  <div class="dashboard">
    <!-- Global gauges removed: counters are shown inside each SourceNodeCard -->

    <div class="dashboard-grid">
      <!-- Source Node Cards - Full Width -->
      <div v-for="(data, sourceNodeID) in nodeStore.sourceNodes" :key="sourceNodeID" class="grid-item full-width">
        <SourceNodeCard :source-node-id="parseInt(sourceNodeID)" :data="data" />
      </div>

      <!-- Top Links -->
      <div class="grid-item full-width">
        <TopLinksCard :top-links="nodeStore.topLinks" @refresh="refreshStats" />
      </div>

      <!-- Talker Log -->
      <!-- DISABLED: Talker log feature temporarily disabled - to be revisited later -->
      <!--
      <div class="grid-item">
        <Card title="Talker Log">
          <div class="talker-log">
                <div v-if="talkerDisplay.length" class="log-entries">
                  <div v-for="(e, i) in talkerDisplay.slice(0, 20)" :key="i" class="log-entry">
                    <span class="time" :title="new Date(e.at).toLocaleString()">{{ formatRelative(e.at) }}</span>
                    <span class="kind" v-html="e.displayLabel"></span>
                  </div>
                </div>
                <div v-else class="no-data">No talker events yet</div>
              </div>
        </Card>
      </div>
      -->
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, computed, reactive, watch, ref } from 'vue'
import { useNodeStore } from '../stores/node'
import { useNodeLookup } from '../composables/useNodeLookup'
import { useAuthStore } from '../stores/auth'
import { connectWS } from '../env'
import { logger } from '../utils/logger'
import Card from '../components/Card.vue'
import TopLinksCard from '../components/TopLinksCard.vue'
import SourceNodeCard from '../components/SourceNodeCard.vue'

const nodeStore = useNodeStore()
const authStore = useAuthStore()
const { enrichedLinks } = useNodeLookup(nodeStore.links)

// local cache for node info (callsign etc) to fill in gaps when links haven't been enriched yet
const nodeInfo = reactive(new Map())

// track the last node we observed as transmitting so that global STOP events can be attributed
const lastTxNode = ref(0)

// keep lastTxNode in sync with link updates when a link is currently keyed
watch(() => nodeStore.links.map(l => l.current_tx), () => {
  const active = nodeStore.links.find(l => l.current_tx)
  lastTxNode.value = active ? active.node : lastTxNode.value
})

async function fetchNodeInfo(nodeID) {
  if (!nodeID || nodeID === 0) return null
  if (nodeInfo.has(nodeID)) return nodeInfo.get(nodeID)
  try {
    const res = await fetch(`/api/node-lookup?q=${nodeID}`)
    const data = await res.json()
    const results = (data && data.results) || (data && data.data && data.data.results) || []
    const info = results && results.length > 0 ? results[0] : null
    const cs = info ? info.callsign || '' : ''
    nodeInfo.set(nodeID, cs)
    return cs
  } catch (e) {
    logger.error('node lookup failed', e)
    nodeInfo.set(nodeID, '')
    return ''
  }
}

// watch for new talker events and ensure we attempt to resolve missing callsigns
function scanAndQueueLookups() {
  const entries = nodeStore.talker.slice(-40)
  for (const ev of entries) {
    const id = ev.node || 0
    let resolved = id
    if (resolved === 0) {
      const openLink = (enrichedLinks.value && enrichedLinks.value.length ? enrichedLinks.value : nodeStore.links).find(l => l.current_tx)
      if (openLink) resolved = openLink.node
    }
    if (resolved && resolved !== 0 && !nodeInfo.has(resolved)) {
      // fire and forget
      fetchNodeInfo(resolved)
    }
  }
}

// Trigger lookups when talker or links change
watch(() => nodeStore.talker.length, () => scanAndQueueLookups())
watch(() => nodeStore.links.length, () => scanAndQueueLookups())

let wsCloser = null

function handleMessage(ev) {
  try {
    const msg = JSON.parse(ev.data)
    nodeStore.handleWSMessage(msg)
  } catch (e) { logger.error('ws parse error:', e) }
}

function initWS() {
  if (wsCloser) {
    wsCloser()
    wsCloser = null
  }
  wsCloser = connectWS({
    onMessage: handleMessage,
  onStatus: (s) => logger.info('[ws]', s),
    tokenProvider: () => authStore.token || ''
  })
}

function refreshStats() {
  const headers = authStore.getAuthHeaders()
  Promise.all([
    fetch('/api/link-stats?sort=tx_seconds_desc&limit=10', { headers }).then(r => r.json()).catch(() => null),
    fetch('/api/link-stats/top?limit=5', { headers }).then(r => r.json()).catch(() => null),
    fetch('/api/talker-log', { headers }).then(r => r.json()).catch(() => null)
  ]).then(([all, top, talkerLog]) => {
    if (top && top.ok) {
      nodeStore.setTopLinks(top.data.results || [])
    }
    if (talkerLog && talkerLog.ok && talkerLog.events) {
      // Load existing talker events into the store
      nodeStore.loadTalkerHistory(talkerLog.events)
    }
  })
}

onMounted(() => {
  initWS()
  refreshStats()
  nodeStore.startTickTimer()
})

onUnmounted(() => {
  if (wsCloser) {
    wsCloser()
  }
})

// helper to format durations (seconds -> human friendly)
function formatDurationSecs(secs) {
  if (!secs || secs === 0) return '0s'
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

// helper to format relative times (uses nodeStore.nowTick updated every second)
function formatRelative(at) {
  const now = nodeStore.nowTick || Date.now()
  const t = new Date(at).getTime()
  const diff = Math.floor((now - t) / 1000)
  if (diff < 5) return 'just now'
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff/60)}m ${diff%60}s`
  const h = Math.floor(diff/3600)
  return `${h}h ${Math.floor((diff%3600)/60)}m`
}

// computed display list for talker: keep START and STOP events separate
// compute talk time for STOP entries by pairing with the most recent START for the same node
const talkerDisplay = computed(() => {
  const events = nodeStore.talker.slice() // oldest -> newest
  const lastStartByNode = new Map()
  const out = []
  const linkSource = enrichedLinks.value && enrichedLinks.value.length ? enrichedLinks.value : nodeStore.links

  for (const ev of events) {
    const node = ev.node || 0
    
    // Resolve node==0 by checking current keyed link or lastTxNode
    let resolvedNode = node
    let resolvedLink = null
    if (resolvedNode === 0) {
      resolvedLink = linkSource.find(l => l.current_tx)
      if (resolvedLink) resolvedNode = resolvedLink.node
      else if (lastTxNode.value) resolvedNode = lastTxNode.value
      // If still 0, skip this event
      if (resolvedNode === 0) continue
    } else {
      resolvedLink = linkSource.find(l => l.node === resolvedNode) || null
    }

    const at = new Date(ev.at)
    // Prefer enriched server-side callsign, fallback to client-side lookups
    let callsign = ev.callsign || (resolvedLink && resolvedLink.node_callsign) || ''
    if (!callsign) callsign = nodeInfo.get(resolvedNode) || ''

    // Format node number (show callsign for negative/text nodes)
    let displayNode = resolvedNode
    if (resolvedNode < 0 && callsign) {
      displayNode = callsign  // For text nodes, just show callsign
    }

    // Build clickable HTML links
    const callsignLink = callsign ? `<a href="https://www.qrz.com/db/${callsign.toUpperCase()}" target="_blank" rel="noopener noreferrer" class="callsign-link">${callsign}</a>` : ''
    const nodeLink = resolvedNode > 0 ? `<a href="https://stats.allstarlink.org/stats/${resolvedNode}" target="_blank" rel="noopener noreferrer" class="node-link">${displayNode}</a>` : `${displayNode}`
    const displayName = callsign ? `${callsignLink} (${nodeLink})` : nodeLink

    if (ev.kind === 'TX_START') {
      lastStartByNode.set(resolvedNode, at)
      out.push({ at, node: resolvedNode, displayLabel: `${displayName} — START` })
      if (resolvedNode && resolvedNode !== 0) lastTxNode.value = resolvedNode
    } else if (ev.kind === 'TX_STOP') {
      const startAt = lastStartByNode.get(resolvedNode)
      // Prefer server-calculated duration if available
      const seconds = ev.duration || (startAt ? Math.floor((at - startAt) / 1000) : null)
      if (seconds !== null) {
        out.push({ at, node: resolvedNode, displayLabel: `${displayName} — STOP — ${formatDurationSecs(seconds)}` })
        if (startAt) lastStartByNode.delete(resolvedNode)
      } else {
        out.push({ at, node: resolvedNode, displayLabel: `${displayName} — STOP` })
      }
    } else {
      out.push({ at, node: resolvedNode, displayLabel: `${displayName} — ${ev.kind}` })
    }
  }

  // Optionally include any unmatched START entries (already included when STARTs were processed)
  // Return newest-first
  return out.sort((a, b) => b.at - a.at)
})
</script>

<style scoped>
.dashboard {
  padding: 1rem;
  max-width: 1600px;
  margin: 0 auto;
}

.link-gauges {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.gauge {
  flex: 1;
  padding: 1.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  text-align: center;
  transition: all 0.2s;
}

.gauge:hover {
  border-color: var(--accent-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.gauge-label {
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin-bottom: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 500;
}

.gauge-value {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--accent-primary);
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1rem;
}

.grid-item.full-width {
  grid-column: 1 / -1;
}

.talker-log {
  max-height: 400px;
  overflow-y: auto;
}

.log-entries {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.log-entry {
  display: flex;
  gap: 1rem;
  padding: 0.5rem;
  background: var(--bg-tertiary);
  border-radius: 4px;
  font-size: 0.875rem;
}

.log-entry .time {
  color: var(--accent-primary);
  font-weight: 500;
  min-width: 100px;
}

.log-entry .kind {
  color: var(--text-secondary);
}

/* Clickable links in talker log */
.log-entry :deep(.callsign-link),
.log-entry :deep(.node-link) {
  color: var(--accent-primary);
  text-decoration: none;
  font-weight: 600;
  transition: color 0.2s;
}

.log-entry :deep(.callsign-link:hover),
.log-entry :deep(.node-link:hover) {
  color: var(--accent-hover);
  text-decoration: underline;
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-style: italic;
}

/* Scrollbar styling */
.talker-log::-webkit-scrollbar {
  width: 8px;
}

.talker-log::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 4px;
}

.talker-log::-webkit-scrollbar-thumb {
  background: var(--border-hover);
  border-radius: 4px;
}

.talker-log::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}
</style>
