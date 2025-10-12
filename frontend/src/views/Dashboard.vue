<template>
  <div class="dashboard">
    <!-- Global gauges removed: counters are shown inside each SourceNodeCard -->

    <div class="dashboard-grid">
      <!-- Render SourceNodeCard for each source node so E2E WS tests can observe cards -->
      <div class="grid-item full-width">
        <h3>Source Nodes</h3>
        <div v-if="Object.keys(nodeStore.sourceNodes || {}).length === 0" class="no-data">No source nodes</div>
        <div v-for="(entry, key) in nodeStore.sourceNodes" :key="key" class="source-node-wrapper">
          <SourceNodeCard :source-node-id="Number(key)" :data="entry" />
        </div>
      </div>

      <!-- Scoreboard + Transmission History (reordered: scoreboard left) -->
      <div class="grid-item">
        <ScoreboardCard :scoreboard="nodeStore.scoreboard" :level-config="nodeStore.levelConfig" @refresh="nodeStore.fetchScoreboard" />
      </div>

      <div class="grid-item">
        <TransmissionHistoryCard
          :transmissions="nodeStore.recentTransmissions"
          :currentPage="currentPage"
          :totalPages="totalPages"
          @page-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, watch, ref } from 'vue'
import { useNodeStore } from '../stores/node'
import { useAuthStore } from '../stores/auth'
import { connectWS } from '../env'
import { logger } from '../utils/logger'
import SourceNodeCard from '../components/SourceNodeCard.vue'
import ScoreboardCard from '../components/ScoreboardCard.vue'
import TransmissionHistoryCard from '../components/TransmissionHistoryCard.vue'

const nodeStore = useNodeStore()
const authStore = useAuthStore()

// Pagination for transmission history
const currentPage = ref(1)
const totalPages = ref(5)

function handlePageChange(page) {
  currentPage.value = page
  nodeStore.fetchRecentTransmissions(50, (page - 1) * 50)
}

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

let transmissionPollInterval = null
let txEndRefreshTimeout = null

onMounted(() => {
  initWS()
  refreshStats()
  nodeStore.startTickTimer()
  // Queue a scoreboard reload on mount so WS initial snapshot can populate first,
  // then ensure we poll the scoreboard every minute to keep it fresh.
  try {
    nodeStore.queueScoreboardReload(600, 50)
    nodeStore.startScoreboardPoll(60000, 50)
  } catch (e) {}

  // Fetch recent transmissions on mount
  nodeStore.fetchRecentTransmissions(50, 0)

  // Poll recent transmissions every 60 seconds
  transmissionPollInterval = setInterval(() => {
    nodeStore.fetchRecentTransmissions(50, (currentPage.value - 1) * 50)
  }, 60000)

  // Watch for TX events ending and trigger refresh after 5 seconds
  watch(() => nodeStore.links.some(l => l.current_tx), (anyTx, wasAnyTx) => {
    if (wasAnyTx && !anyTx) {
      // TX just ended, schedule a refresh in 5 seconds
      if (txEndRefreshTimeout) clearTimeout(txEndRefreshTimeout)
      txEndRefreshTimeout = setTimeout(() => {
        nodeStore.fetchRecentTransmissions(50, (currentPage.value - 1) * 50)
      }, 5000)
    }
  })
})

onUnmounted(() => {
  if (wsCloser) {
    wsCloser()
  }
  try { nodeStore.stopScoreboardPoll() } catch (e) {}
  if (transmissionPollInterval) clearInterval(transmissionPollInterval)
  if (txEndRefreshTimeout) clearTimeout(txEndRefreshTimeout)
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
