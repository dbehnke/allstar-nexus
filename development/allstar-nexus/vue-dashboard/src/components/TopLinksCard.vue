<template>
  <Card title="Top Links (TX Seconds)">
    <template #actions>
      <button @click="$emit('refresh')" class="btn-icon">&#x21bb;</button>
    </template>

    <div v-if="topLinks.length" class="top-links-list">
      <div v-for="(t, idx) in topLinks" :key="t.node" class="top-link-item">
        <div class="rank">{{ idx + 1 }}</div>
        <div class="details">
          <div class="node">Node {{ t.node }}</div>
          <div class="stats">
            {{ t.total_tx_seconds }}s
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
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
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

.node {
  font-weight: 600;
  color: #eee;
}

.stats {
  font-size: 0.875rem;
  color: #999;
}

.rate {
  color: #60a5fa;
  font-weight: 500;
}

.bar-container {
  background: #2a2a2a;
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
}

.bar {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  border-radius: 4px;
  transition: width 0.3s ease;
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
