<template>
  <Card title="Top Links (TX Seconds)">
    <template #actions>
      <button @click="$emit('refresh')" class="btn-icon">&#x21bb;</button>
    </template>

    <div v-if="topLinks.length" class="top-links-list">
      <div v-for="(t, idx) in topLinks" :key="t.node" class="top-link-item">
        <div class="rank">{{ idx + 1 }}</div>
        <div class="details">
          <div class="node-header">
            <span class="callsign">{{ t.callsign || 'Unknown' }}</span>
            <span class="node-num">({{ t.node }})</span>
          </div>
          <div class="node-info" v-if="t.description || t.location">
            <span v-if="t.description" class="description">{{ t.description }}</span>
            <span v-if="t.location" class="location">{{ t.location }}</span>
          </div>
          <div class="stats">
            {{ formatDuration(t.total_tx_seconds) }}
            <span v-if="t.rate" class="rate">({{ (t.rate*100).toFixed(2) }}%)</span>
          </div>
        </div>
        <div class="bar-container">
          <div class="bar" :style="{ width: getBarWidth(t, topLinks[0]) }"></div>
        </div>
      </div>
    </div>
    <div v-else class="no-data">
      No link statistics available
    </div>
  </Card>
</template>

<script setup>
import Card from './Card.vue'

defineProps({
  topLinks: Array
})

defineEmits(['refresh'])

function getBarWidth(item, max) {
  if (!max || !max.total_tx_seconds) return '0%'
  return ((item.total_tx_seconds / max.total_tx_seconds) * 100).toFixed(1) + '%'
}

function formatDuration(secs) {
  if (!secs || secs === 0) return '0s'
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  if (h > 0) return `${h}h ${m}m ${s}s`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}
</script>

<style scoped>
.top-links-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.top-link-item {
  display: grid;
  grid-template-columns: 40px 1fr 150px;
  gap: 1rem;
  align-items: center;
}

.rank {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, var(--accent-gradient-start), var(--accent-gradient-end));
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 1.125rem;
  color: #fff;
}

.details {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.callsign {
  font-weight: 700;
  color: var(--success);
  font-size: 1rem;
}

.node-num {
  font-size: 0.875rem;
  color: var(--accent-primary);
}

.node-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  font-size: 0.75rem;
}

.description {
  color: var(--text-muted);
}

.location {
  color: var(--accent-primary);
  font-style: italic;
}

.stats {
  font-size: 0.875rem;
  color: var(--text-muted);
  font-weight: 500;
}

.rate {
  color: var(--accent-primary);
  font-weight: 500;
}

.bar-container {
  background: var(--bg-tertiary);
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
}

.bar {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-gradient-start), var(--accent-primary));
  border-radius: 4px;
  transition: width 0.3s ease;
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
</style>
