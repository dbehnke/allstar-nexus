<template>
  <div class="voter-display">
    <Card title="Voter / RTCM Display">
      <div class="node-selector">
        <label for="voter-select">Select Voter Node:</label>
        <select id="voter-select" v-model="selectedNode" @change="loadVoterData" class="node-select">
          <option value="">-- Select a voter node --</option>
          <option v-for="node in voterNodes" :key="node" :value="node">
            Node {{ node }}
          </option>
        </select>
      </div>

      <div v-if="error" class="error-message">
        {{ error }}
      </div>

      <div v-if="voterData && voterData.receivers" class="voter-display-grid">
        <div v-for="rx in voterData.receivers" :key="rx.id" class="receiver-card">
          <div class="receiver-header">
            <span class="receiver-name">{{ rx.name || `Receiver ${rx.id}` }}</span>
            <span class="receiver-status" :class="{ voted: rx.voted }">
              {{ rx.voted ? 'VOTED' : 'STANDBY' }}
            </span>
          </div>
          <div class="rssi-container">
            <div class="rssi-bar-bg">
              <div
                class="rssi-bar"
                :class="getRssiClass(rx)"
                :style="{ width: getRssiPercent(rx.rssi) }"
              ></div>
            </div>
            <div class="rssi-value">RSSI: {{ rx.rssi }}</div>
          </div>
          <div class="receiver-info">
            <div class="info-item">
              <span class="label">Type:</span>
              <span class="value">{{ rx.type || 'N/A' }}</span>
            </div>
            <div class="info-item">
              <span class="label">IP:</span>
              <span class="value">{{ rx.ip || 'N/A' }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="!loading && selectedNode" class="no-data">
        No voter data available
      </div>

      <div class="legend">
        <h4>Legend:</h4>
        <div class="legend-items">
          <div class="legend-item">
            <div class="legend-color voting"></div>
            <span>Voting Station (Blue)</span>
          </div>
          <div class="legend-item">
            <div class="legend-color voted"></div>
            <span>Voted Receiver (Green)</span>
          </div>
          <div class="legend-item">
            <div class="legend-color mix"></div>
            <span>Non-voting Mix (Cyan)</span>
          </div>
        </div>
        <p class="legend-note">
          RSSI values range from 0 to 255, representing approximately 30db. Zero indicates no signal.
        </p>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import Card from '../components/Card.vue'

const authStore = useAuthStore()

const selectedNode = ref('')
const voterData = ref(null)
const loading = ref(false)
const error = ref('')

// Mock voter nodes - in production, this would come from API
const voterNodes = ref([])

async function loadVoterData() {
  if (!selectedNode.value) return

  loading.value = true
  error.value = ''
  voterData.value = null

  try {
    const headers = authStore.getAuthHeaders()
    const response = await fetch(`/api/voter-stats?node=${selectedNode.value}`, { headers })
    const data = await response.json()

    if (!response.ok || !data.ok) {
      error.value = data.error?.message || 'Failed to load voter data'
      return
    }

    voterData.value = data.data
  } catch (e) {
    error.value = 'Network error occurred'
    console.error('Voter data error:', e)
  } finally {
    loading.value = false
  }
}

function getRssiPercent(rssi) {
  return ((rssi / 255) * 100).toFixed(1) + '%'
}

function getRssiClass(rx) {
  if (rx.voted) return 'voted'
  if (rx.type === 'mix') return 'mix'
  return 'voting'
}
</script>

<style scoped>
.voter-display {
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

.error-message {
  background: #dc2626;
  color: #fff;
  padding: 1rem;
  border-radius: 6px;
  margin-bottom: 1.5rem;
}

.voter-display-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.receiver-card {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 8px;
  padding: 1.5rem;
  transition: all 0.3s;
}

.receiver-card:hover {
  border-color: #444;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.receiver-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.receiver-name {
  font-weight: 600;
  color: #eee;
  font-size: 1.125rem;
}

.receiver-status {
  padding: 0.25rem 0.75rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  background: #374151;
  color: #9ca3af;
}

.receiver-status.voted {
  background: #16a34a;
  color: #fff;
}

.rssi-container {
  margin-bottom: 1rem;
}

.rssi-bar-bg {
  background: #2a2a2a;
  height: 32px;
  border-radius: 6px;
  overflow: hidden;
  margin-bottom: 0.5rem;
}

.rssi-bar {
  height: 100%;
  transition: width 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: 0.5rem;
  color: #fff;
  font-weight: 600;
  font-size: 0.875rem;
}

.rssi-bar.voting {
  background: linear-gradient(90deg, #0099ff, #00ccff);
}

.rssi-bar.voted {
  background: linear-gradient(90deg, #16a34a, #4ade80);
}

.rssi-bar.mix {
  background: linear-gradient(90deg, #06b6d4, #22d3ee);
}

.rssi-value {
  font-size: 0.875rem;
  color: #999;
  text-align: center;
}

.receiver-info {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 1rem;
  border-top: 1px solid #2a2a2a;
}

.info-item {
  display: flex;
  justify-content: space-between;
}

.info-item .label {
  color: #999;
  font-size: 0.875rem;
}

.info-item .value {
  color: #eee;
  font-weight: 500;
  font-size: 0.875rem;
}

.no-data {
  text-align: center;
  padding: 3rem;
  color: #666;
  font-style: italic;
}

.legend {
  margin-top: 2rem;
  padding: 1.5rem;
  background: #1a1a1a;
  border-left: 4px solid #3b82f6;
  border-radius: 6px;
}

.legend h4 {
  margin-top: 0;
  margin-bottom: 1rem;
  color: #60a5fa;
}

.legend-items {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.legend-color {
  width: 40px;
  height: 20px;
  border-radius: 4px;
}

.legend-color.voting {
  background: linear-gradient(90deg, #0099ff, #00ccff);
}

.legend-color.voted {
  background: linear-gradient(90deg, #16a34a, #4ade80);
}

.legend-color.mix {
  background: linear-gradient(90deg, #06b6d4, #22d3ee);
}

.legend-note {
  margin: 0;
  color: #999;
  font-size: 0.875rem;
  font-style: italic;
}
</style>
