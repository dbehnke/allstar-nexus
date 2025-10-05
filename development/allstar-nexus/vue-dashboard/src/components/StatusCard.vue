<template>
  <Card title="Node Status">
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
        <span class="label">Links:</span>
        <span class="value">{{ status.links?.join(', ') || '—' }}</span>
      </div>
      <div class="status-item">
        <span class="label">RX Keyed:</span>
        <span class="value">{{ status.rx_keyed }}</span>
      </div>
      <div class="status-item">
        <span class="label">TX Keyed:</span>
        <span class="value">{{ status.tx_keyed }}</span>
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
import Card from './Card.vue'

defineProps({
  status: Object
})

defineEmits(['refresh'])

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
  color: #999;
  text-transform: uppercase;
  font-weight: 500;
}

.value {
  font-size: 1rem;
  color: #eee;
  font-weight: 600;
}

.pulse {
  color: #4ade80;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: #666;
  font-style: italic;
}

.btn-icon {
  background: transparent;
  border: 1px solid #555;
  color: #eee;
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
  background: #333;
  border-color: #666;
}
</style>
