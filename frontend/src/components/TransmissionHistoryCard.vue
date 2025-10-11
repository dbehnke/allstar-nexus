<template>
  <div class="card">
    <div class="card-header">
      <h3>Recent Transmissions</h3>
    </div>
    <div class="card-body">
      <div v-if="pagedTransmissions.length" class="table-wrapper">
        <table class="history-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Callsign</th>
              <th>Node</th>
              <th>Duration</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(t, idx) in pagedTransmissions" :key="idx">
              <td :title="new Date(t.timestamp || t.at).toLocaleString()">{{ formatRelative(t.timestamp || t.at) }}</td>
              <td>
                <a v-if="t.callsign" class="callsign" :href="`https://www.qrz.com/db/${(t.callsign||'').toUpperCase()}`" target="_blank" rel="noopener noreferrer">{{ t.callsign }}</a>
                <span v-else class="muted">—</span>
              </td>
              <td>
                <a v-if="t.node" :href="`https://stats.allstarlink.org/stats/${t.node}`" target="_blank" rel="noopener noreferrer">{{ t.node }}</a>
                <span v-else class="muted">—</span>
              </td>
              <td>{{ formatDuration(t.duration || t.seconds || 0) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty">No transmissions yet</div>
    </div>
    <div class="card-footer" v-if="totalPages > 1">
      <div class="pagination">
        <button class="btn-secondary" :disabled="currentPage <= 1" @click="emit('page-change', currentPage - 1)">« Prev</button>
        <button
          v-for="p in totalPages"
          :key="p"
          :class="['page-btn', { active: p === currentPage }]"
          @click="emit('page-change', p)"
        >{{ p }}</button>
        <button class="btn-secondary" :disabled="currentPage >= totalPages" @click="emit('page-change', currentPage + 1)">Next »</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  transmissions: { type: Array, default: () => [] },
  currentPage: { type: Number, default: 1 },
  totalPages: { type: Number, default: 1 }
})
const emit = defineEmits(['page-change'])

const pagedTransmissions = computed(() => {
  const perPage = 10
  const start = (props.currentPage - 1) * perPage
  return (props.transmissions || []).slice(start, start + perPage)
})

function formatRelative(at) {
  try {
    const t = typeof at === 'number' ? at : new Date(at).getTime()
    const diff = Math.floor((Date.now() - t) / 1000)
    if (diff < 5) return 'just now'
    if (diff < 60) return `${diff}s ago`
    if (diff < 3600) return `${Math.floor(diff/60)}m ${diff%60}s ago`
    const h = Math.floor(diff/3600)
    return `${h}h ${Math.floor((diff%3600)/60)}m ago`
  } catch { return '' }
}

function formatDuration(secs) {
  const s = Number(secs || 0)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const ss = s % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${ss}s`
  return `${ss}s`
}
</script>

<style scoped>
.card { background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; box-shadow: 0 2px 8px var(--shadow); display: flex; flex-direction: column; }
.card-header { padding: 0.75rem 1rem; border-bottom: 1px solid var(--border-color); }
.card-body { padding: 0.75rem 1rem; }
.card-footer { padding: 0.5rem 1rem; border-top: 1px solid var(--border-color); }

.table-wrapper { max-height: 520px; overflow: auto; }
.history-table { width: 100%; border-collapse: collapse; }
.history-table th, .history-table td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--border-color); }
.muted { color: var(--text-muted); }
.callsign { color: var(--accent-primary); text-decoration: none; }
.callsign:hover { text-decoration: underline; }

.pagination { display: flex; align-items: center; gap: 0.5rem; }
.page-btn { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 0.35rem 0.65rem; border-radius: 6px; cursor: pointer; }
.page-btn.active { background: var(--accent-hover); color: white; border-color: var(--accent-hover); }
.page-btn:hover { background: var(--bg-hover); }

.empty { padding: 1rem; color: var(--text-muted); }
</style>
