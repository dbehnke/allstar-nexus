<template>
  <div class="dashboard">
    <div class="dashboard-grid">
      <!-- Status Card -->
      <div class="grid-item full-width">
        <StatusCard :status="nodeStore.status" @refresh="refreshStats" />
      </div>

      <!-- Top Links -->
      <div class="grid-item">
        <TopLinksCard :top-links="nodeStore.topLinks" @refresh="refreshStats" />
      </div>

      <!-- Active Links -->
      <div class="grid-item full-width">
        <LinksCard
          :links="nodeStore.links"
          :status="nodeStore.status"
          :now-tick="nodeStore.nowTick"
        />
      </div>

      <!-- Talker Log -->
      <div class="grid-item">
        <Card title="Talker Log">
          <div class="talker-log">
            <div v-if="nodeStore.talker.length" class="log-entries">
              <div v-for="(e, i) in nodeStore.talker.slice().reverse().slice(0, 20)" :key="i" class="log-entry">
                <span class="time">{{ new Date(e.at).toLocaleTimeString() }}</span>
                <span class="kind">{{ e.kind }}</span>
              </div>
            </div>
            <div v-else class="no-data">No talker events yet</div>
          </div>
        </Card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { useNodeStore } from '../stores/node'
import { useAuthStore } from '../stores/auth'
import { connectWS } from '../env'
import Card from '../components/Card.vue'
import StatusCard from '../components/StatusCard.vue'
import LinksCard from '../components/LinksCard.vue'
import TopLinksCard from '../components/TopLinksCard.vue'

const nodeStore = useNodeStore()
const authStore = useAuthStore()

let wsCloser = null

function handleMessage(ev) {
  try {
    const msg = JSON.parse(ev.data)
    nodeStore.handleWSMessage(msg)
  } catch (e) {
    console.error('ws parse error:', e)
  }
}

function initWS() {
  if (wsCloser) {
    wsCloser()
    wsCloser = null
  }
  wsCloser = connectWS({
    onMessage: handleMessage,
    onStatus: (s) => console.log('[ws]', s),
    tokenProvider: () => authStore.token || ''
  })
}

function refreshStats() {
  const headers = authStore.getAuthHeaders()
  Promise.all([
    fetch('/api/link-stats?sort=tx_seconds_desc&limit=10', { headers }).then(r => r.json()).catch(() => null),
    fetch('/api/link-stats/top?limit=5', { headers }).then(r => r.json()).catch(() => null)
  ]).then(([all, top]) => {
    if (top && top.ok) {
      nodeStore.setTopLinks(top.data.results || [])
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
</script>

<style scoped>
.dashboard {
  padding: 1.5rem;
  max-width: 1400px;
  margin: 0 auto;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
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
  background: #222;
  border-radius: 4px;
  font-size: 0.875rem;
}

.log-entry .time {
  color: #60a5fa;
  font-weight: 500;
  min-width: 100px;
}

.log-entry .kind {
  color: #ddd;
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: #666;
  font-style: italic;
}

/* Scrollbar styling */
.talker-log::-webkit-scrollbar {
  width: 8px;
}

.talker-log::-webkit-scrollbar-track {
  background: #1c1c1c;
  border-radius: 4px;
}

.talker-log::-webkit-scrollbar-thumb {
  background: #444;
  border-radius: 4px;
}

.talker-log::-webkit-scrollbar-thumb:hover {
  background: #555;
}
</style>
