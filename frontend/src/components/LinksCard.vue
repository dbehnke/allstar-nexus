<template>
  <Card title="Active Links">
    <div v-if="links.length" class="table-container">
      <table class="links-table">
        <thead>
          <tr>
            <th>Node</th>
            <th>Node Information</th>
            <th>Status</th>
            <th>Direction</th>
            <th>Connected</th>
            <th>Mode</th>
            <th v-if="showIP">IP Address</th>
            <th>Last Heard</th>
            <th>Total TX</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in sortedLinks" :key="l.node" :class="{ txing: l.current_tx || l.is_keyed }">
            <td class="node-num">{{ l.node }}</td>
            <td class="node-info">
              <div v-if="l.node_callsign" class="callsign">{{ l.node_callsign }}</div>
              <div v-if="l.node_description || l.node_location" class="node-details">
                <span v-if="l.node_description">{{ l.node_description }}</span>
                <span v-if="l.node_location" class="location">{{ l.node_location }}</span>
              </div>
              <div v-if="!l.node_callsign" class="loading">Loading...</div>
            </td>
            <td>
              <span class="status-badge" :class="{ active: l.current_tx || l.is_keyed }">
                {{ (l.current_tx || l.is_keyed) ? '● TX' : 'IDLE' }}
              </span>
            </td>
            <td>
              <span class="direction-badge" :class="l.direction?.toLowerCase()">
                {{ l.direction || '—' }}
              </span>
            </td>
            <td>{{ l.elapsed || formatSince(l.connected_since) }}</td>
            <td>
              <span class="mode-badge" :class="modeClass(l.mode)">
                {{ formatMode(l.mode) }}
              </span>
            </td>
            <td v-if="showIP" class="ip-addr">{{ l.ip || '—' }}</td>
            <td class="last-heard">{{ l.last_heard || (l.last_heard_at ? formatSince(l.last_heard_at) : 'Never') }}</td>
            <td>{{ formatDuration(l.total_tx_seconds) }}</td>
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
import { computed, toRef } from 'vue'
import Card from './Card.vue'
import { useAuthStore } from '../stores/auth'
import { useNodeLookup } from '../composables/useNodeLookup'

const props = defineProps({
  links: Array,
  status: Object,
  nowTick: Number
})

const authStore = useAuthStore()
const showIP = computed(() => authStore.isAdmin)

// Enrich links with node information from astdb
const { enrichedLinks } = useNodeLookup(toRef(props, 'links'))

// Sort links: active talker(s) at top, then by total_tx_seconds (desc), then by connected duration (desc)
const sortedLinks = computed(() => {
  if (!enrichedLinks.value) return []
  const arr = [...enrichedLinks.value]
  const now = Date.now()
  return arr.sort((a, b) => {
    const aActive = !!(a.current_tx || a.is_keyed)
    const bActive = !!(b.current_tx || b.is_keyed)
    if (aActive && !bActive) return -1
    if (!aActive && bActive) return 1

    // Both same activity state: sort by total_tx_seconds (desc)
    const aTx = Number(a.total_tx_seconds || 0)
    const bTx = Number(b.total_tx_seconds || 0)
    if (aTx !== bTx) return bTx - aTx

    // Then by connected duration (longer connected first)
    const aConn = a.connected_since ? now - new Date(a.connected_since).getTime() : -1
    const bConn = b.connected_since ? now - new Date(b.connected_since).getTime() : -1
    if (aConn !== bConn) return bConn - aConn

    // Final fallback: most recently heard first
    const aTime = a.last_keyed_time || a.last_heard_at || a.connected_since
    const bTime = b.last_keyed_time || b.last_heard_at || b.connected_since
    if (!aTime && !bTime) return 0
    if (!aTime) return 1
    if (!bTime) return -1
    return new Date(bTime) - new Date(aTime)
  })
})

function formatSince(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  const diff = (Date.now()-d.getTime())/1000
  if (diff < 60) return Math.floor(diff)+ 's ago'
  if (diff < 3600) return Math.floor(diff/60)+ 'm ago'
  if (diff < 86400) return Math.floor(diff/3600)+ 'h ago'
  return Math.floor(diff/86400)+'d ago'
}

function formatDuration(secs) {
  if (!secs || secs === 0) return '0s'
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function formatMode(mode) {
  if (!mode) return '—'
  const modes = {
    'T': 'Transceive',
    'R': 'Receive',
    'C': 'Connecting',
    'M': 'Monitor'
  }
  return modes[mode] || mode
}

function modeClass(mode) {
  if (!mode) return ''
  return `mode-${mode.toLowerCase()}`
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
  padding: 0.5rem 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.links-table thead th {
  background: var(--bg-tertiary);
  color: var(--text-label);
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
  background: var(--bg-hover);
}

.links-table tbody tr.txing {
  animation: blink 1s linear infinite;
  background: var(--bg-hover);
}

@keyframes blink {
  50% { background: var(--bg-tertiary); }
}

.node-num {
  font-weight: 600;
  color: var(--accent-primary);
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  background: var(--bg-tertiary);
  color: var(--text-muted);
}

.status-badge.active {
  background: var(--error);
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
  color: var(--text-muted);
  font-style: italic;
}

.mode-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
}

.mode-badge.mode-t {
  background: #059669;
  color: #fff;
}

.mode-badge.mode-r {
  background: #0284c7;
  color: #fff;
}

.mode-badge.mode-c {
  background: #ca8a04;
  color: #fff;
}

.mode-badge.mode-m {
  background: #6b7280;
  color: #fff;
}

.direction-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
}

.direction-badge.in {
  background: #1e3a8a;
  color: #93c5fd;
}

.direction-badge.out {
  background: #7c2d12;
  color: #fdba74;
}

.ip-addr {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  color: #9ca3af;
}

.last-heard {
  font-weight: 500;
  color: #d1d5db;
}

.node-info {
  min-width: 200px;
}

.node-info .callsign {
  font-weight: 600;
  color: #10b981;
  font-size: 0.875rem;
  margin-bottom: 0.125rem;
}

.node-info .node-details {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  font-size: 0.75rem;
  color: #9ca3af;
}

.node-info .location {
  color: #60a5fa;
  font-style: italic;
}

.node-info .loading {
  color: #6b7280;
  font-size: 0.75rem;
  font-style: italic;
}
</style>
