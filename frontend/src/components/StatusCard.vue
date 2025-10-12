<template>
  <Card :title="cardTitle">
    <template #actions>
      <button @click="$emit('refresh')" class="btn-icon">&#x21bb;</button>
    </template>

    <div v-if="status" class="status-grid">
      <div class="status-item">
        <span class="label">Updated:</span>
        <span class="value">{{ new Date(status.updated_at).toLocaleTimeString() }}</span>
      </div>
      <div class="status-item">
        <span class="label">Uptime:</span>
        <span class="value">{{ humanUptime(status) }}</span>
      </div>
      <div class="status-item">
        <span class="label">Links (Adj/Total):</span>
        <span class="value">{{ status.num_alinks || 0 }} / {{ status.num_links || 0 }}</span>
      </div>
      <div class="status-item">
        <span class="label">COS (RX):</span>
        <span class="value">
          <span class="indicator" :class="{ 'active': status.rx_keyed, 'rx': true }">
            {{ status.rx_keyed ? '●' : '○' }}
          </span>
          {{ status.rx_keyed ? 'ACTIVE' : 'Idle' }}
        </span>
      </div>
      <div class="status-item">
        <span class="label">PTT (TX):</span>
        <span class="value">
          <span class="indicator" :class="{ 'active': status.tx_keyed, 'tx': true }">
            {{ status.tx_keyed ? '●' : '○' }}
          </span>
          {{ status.tx_keyed ? 'ACTIVE' : 'Idle' }}
        </span>
      </div>
      <div class="status-item">
        <span class="label">Version:</span>
        <span class="value">{{ status.version }}</span>
      </div>
      <div class="status-item">
        <span class="label">Heartbeat:</span>
        <span class="value" :class="{ 'pulse': status.heartbeat }">{{ status.heartbeat ? '●' : '○' }}</span>
      </div>
    </div>
    <div v-else class="no-data">
      Waiting for data...
    </div>
  </Card>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import Card from './Card.vue'
import { logger } from '../utils/logger'

const props = defineProps({
  status: Object
})

defineEmits(['refresh'])

const nodeInfo = ref(null)
const cardTitle = computed(() => {
  if (!props.status) return 'Node Status'

  const nodeId = props.status.node_id
  if (!nodeId) return 'Node Status'

  let title = `Node Status - ${nodeId}`
  if (nodeInfo.value?.description) {
    title += ` - ${nodeInfo.value.description}`
  }
  return title
})

// Fetch node info when node_id changes
watch(() => props.status?.node_id, async (nodeId) => {
  if (!nodeId) {
    nodeInfo.value = null
    return
  }

  try {
    const res = await fetch(`/api/node-lookup?q=${nodeId}`)
    const data = await res.json()
    if (data.results && data.results.length > 0) {
      nodeInfo.value = data.results[0]
    }
  } catch (e) {
  logger.error('Failed to fetch node info:', e)
  }
}, { immediate: true })

function humanUptime(st) {
  if (!st) return '—'
  if (st.booted_at) {
    const boot = new Date(st.booted_at).getTime()
    const now = Date.now()
    const secs = Math.max(0, Math.floor((now - boot)/1000))
    return formatDuration(secs)
  }
  if (typeof st.uptime_sec === 'number' && st.uptime_sec > 0) return formatDuration(st.uptime_sec)
  return '—'
}

function formatDuration(total) {
  const d = Math.floor(total / 86400); total %= 86400
  const h = Math.floor(total / 3600); total %= 3600
  const m = Math.floor(total / 60); const s = total % 60
  const parts = []
  if (d) parts.push(d+ 'd')
  if (h) parts.push(h+ 'h')
  if (m) parts.push(m+ 'm')
  if (!d && !h && s) parts.push(s+ 's')
  return parts.join(' ') || '0s'
}
</script>

<style scoped>
.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.status-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.label {
  font-size: 0.75rem;
  color: var(--text-label);
  text-transform: uppercase;
  font-weight: 500;
}

.value {
  font-size: 1rem;
  color: var(--text-primary);
  font-weight: 600;
}

.pulse {
  color: var(--success);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-style: italic;
}

.btn-icon {
  background: transparent;
  border: 1px solid var(--border-hover);
  color: var(--text-primary);
  width: 32px;
  height: 32px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: var(--bg-hover);
  border-color: var(--border-hover);
}

.indicator {
  display: inline-block;
  margin-right: 0.5rem;
  font-size: 1.2rem;
  transition: all 0.2s;
}

.indicator.rx.active {
  color: var(--success);
  animation: pulse-rx 1s infinite;
}

.indicator.tx.active {
  color: var(--error);
  animation: pulse-tx 1s infinite;
}

@keyframes pulse-rx {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.7; transform: scale(1.1); }
}

@keyframes pulse-tx {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.7; transform: scale(1.1); }
}
</style>
