<template>
  <div class="rpt-stats">
    <Card title="RPT Statistics">
      <div class="node-selector">
        <label for="node-select">Select Node:</label>
        <select id="node-select" v-model="selectedNode" @change="loadStats" class="node-select">
          <option value="">-- Select a node --</option>
          <option v-for="node in availableNodes" :key="node" :value="node">
            Node {{ node }}
          </option>
        </select>
        <button @click="loadStats" :disabled="loading || !selectedNode" class="btn-primary">
          {{ loading ? 'Loading...' : 'Load Stats' }}
        </button>
      </div>

      <div v-if="error" class="error-message">
        {{ error }}
      </div>

      <div v-if="statsData" class="stats-display">
        <pre class="stats-output">{{ statsData }}</pre>
      </div>

      <div v-else-if="!loading && selectedNode" class="no-data">
        Click "Load Stats" to view RPT statistics
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useNodeStore } from '../stores/node'
import Card from '../components/Card.vue'
import { logger } from '../utils/logger'

const authStore = useAuthStore()
const nodeStore = useNodeStore()

const selectedNode = ref('')
const statsData = ref('')
const loading = ref(false)
const error = ref('')

const availableNodes = computed(() => {
  const nodes = nodeStore.links.map(l => l.node)
  return [...new Set(nodes)].sort((a, b) => a - b)
})

async function loadStats() {
  if (!selectedNode.value) return

  loading.value = true
  error.value = ''
  statsData.value = ''

  try {
    const headers = authStore.getAuthHeaders()
    const response = await fetch(`/api/rpt-stats?node=${selectedNode.value}`, { headers })
    const data = await response.json()

    if (!response.ok || !data.ok) {
      error.value = data.error?.message || 'Failed to load stats'
      return
    }

    statsData.value = data.data.stats || 'No statistics available'
  } catch (e) {
    error.value = 'Network error occurred'
  logger.error('Stats error:', e)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.rpt-stats {
  padding: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

.node-selector {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1.5rem;
}

.node-selector label {
  font-weight: 600;
  color: #eee;
}

.node-select {
  flex: 1;
  max-width: 300px;
  background: #222;
  border: 1px solid #444;
  color: #eee;
  padding: 0.75rem 1rem;
  border-radius: 6px;
  font-size: 1rem;
  cursor: pointer;
}

.node-select:focus {
  outline: none;
  border-color: #60a5fa;
}

.btn-primary {
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  color: #fff;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.error-message {
  background: #dc2626;
  color: #fff;
  padding: 1rem;
  border-radius: 6px;
  margin-bottom: 1.5rem;
}

.stats-display {
  background: #0a0a0a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  padding: 1.5rem;
  margin-top: 1.5rem;
}

.stats-output {
  margin: 0;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  color: #4ade80;
  white-space: pre-wrap;
  word-wrap: break-word;
  line-height: 1.6;
}

.no-data {
  text-align: center;
  padding: 3rem;
  color: #666;
  font-style: italic;
}
</style>
