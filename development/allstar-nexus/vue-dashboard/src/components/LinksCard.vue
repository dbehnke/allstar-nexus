<template>
  <Card title="Active Links">
    <div v-if="links.length" class="table-container">
      <table class="links-table">
        <thead>
          <tr>
            <th>Node</th>
            <th>Connected</th>
            <th>Last Heard</th>
            <th>TX Status</th>
            <th>Elapsed TX</th>
            <th>Total TX</th>
            <th>TX %</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in links" :key="l.node" :class="{ txing: l.current_tx }">
            <td class="node-num">{{ l.node }}</td>
            <td>{{ formatSince(l.connected_since) }}</td>
            <td>{{ l.last_heard_at ? formatSince(l.last_heard_at) : '—' }}</td>
            <td>
              <span class="status-badge" :class="{ active: l.current_tx }">
                {{ l.current_tx ? 'TX' : 'IDLE' }}
              </span>
            </td>
            <td>{{ l.current_tx ? currentElapsed(l) : '—' }}</td>
            <td>{{ l.total_tx_seconds }}s</td>
            <td>{{ txPercent(l) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="no-data">
      No active links
    </div>
  </Card>
</template>

<script setup>
import { computed } from 'vue'
import Card from './Card.vue'

const props = defineProps({
  links: Array,
  status: Object,
  nowTick: Number
})

function formatSince(ts) {
  const d = new Date(ts)
  const diff = (Date.now()-d.getTime())/1000
  if (diff < 60) return Math.floor(diff)+ 's ago'
  if (diff < 3600) return Math.floor(diff/60)+ 'm ago'
  if (diff < 86400) return Math.floor(diff/3600)+ 'h ago'
  return Math.floor(diff/86400)+'d ago'
}

function currentElapsed(l) {
  if (!l.current_tx) return '—'
  const start = new Date(l.last_tx_start || l.connected_since).getTime()
  const secs = Math.max(0, Math.floor((props.nowTick - start)/1000))
  return secs + 's'
}

function txPercent(l) {
  if (!props.status) return '—'
  const start = new Date(props.status.session_start).getTime()
  const elapsed = (Date.now() - start)/1000
  if (elapsed <= 0) return '0%'
  const pct = (l.total_tx_seconds / elapsed) * 100
  return pct.toFixed(1)+'%'
}
</script>

<style scoped>
.table-container {
  overflow-x: auto;
}

.links-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.links-table th,
.links-table td {
  padding: 0.75rem;
  text-align: left;
  border-bottom: 1px solid #2a2a2a;
}

.links-table thead th {
  background: #252525;
  color: #999;
  text-transform: uppercase;
  font-size: 0.75rem;
  font-weight: 600;
  position: sticky;
  top: 0;
}

.links-table tbody tr {
  transition: background-color 0.2s;
}

.links-table tbody tr:hover {
  background: #222;
}

.links-table tbody tr.txing {
  animation: blink 1s linear infinite;
  background: #2a2a2a;
}

@keyframes blink {
  50% { background: #333; }
}

.node-num {
  font-weight: 600;
  color: #60a5fa;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  background: #374151;
  color: #9ca3af;
}

.status-badge.active {
  background: #dc2626;
  color: #fff;
  animation: pulse-badge 1s infinite;
}

@keyframes pulse-badge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: #666;
  font-style: italic;
}
</style>
