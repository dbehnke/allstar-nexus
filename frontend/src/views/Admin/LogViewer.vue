<template>
  <div class="log-viewer">
    <div class="header">
      <h2>📜 Log Management</h2>
      <p>Real-time log streaming and export</p>
    </div>

    <!-- Controls Section -->
    <div class="controls-panel">
      <div class="control-row">
        <div class="control-group">
          <label>Log Source</label>
          <select v-model="selectedSource" @change="stopStream" :disabled="streaming">
            <option value="">Select a source...</option>
            <option v-for="src in sources" :key="src.id" :value="src.id" :disabled="!src.available">
              {{ src.name }} {{ src.available ? '' : '(unavailable)' }}
            </option>
          </select>
        </div>

        <div class="control-group">
          <label>Log Level</label>
          <select v-model="level" @change="handleFilterChange">
            <option value="ALL">All Levels</option>
            <option value="DEBUG">Debug</option>
            <option value="INFO">Info</option>
            <option value="WARNING">Warning</option>
            <option value="ERROR">Error</option>
          </select>
        </div>

        <div class="control-group">
          <label>Tail Lines</label>
          <input type="number" v-model.number="tailLines" min="10" max="1000" @change="handleFilterChange" />
        </div>
      </div>

      <div class="control-row">
        <div class="control-group search-group">
          <label>Search Keyword</label>
          <input
            type="text"
            v-model="keyword"
            placeholder="Filter by keyword..."
            @input="handleFilterChange"
          />
        </div>

        <div class="control-group">
          <label>Actions</label>
          <div class="button-group">
            <button @click="startStream" :disabled="!selectedSource || streaming" class="btn-primary">
              ▶ Start Stream
            </button>
            <button @click="stopStream" :disabled="!streaming" class="btn-secondary">
              ⏸ Pause
            </button>
            <button @click="clearLogs" class="btn-secondary">
              🗑 Clear
            </button>
            <button @click="exportLogs" :disabled="!selectedSource" class="btn-secondary">
              💾 Export
            </button>
          </div>
        </div>
      </div>

      <!-- Status Bar -->
      <div class="status-bar" :class="{ streaming: streaming, paused: !streaming }">
        <span v-if="streaming">● STREAMING - {{ entries.length }} entries</span>
        <span v-else-if="entries.length > 0">⏸ PAUSED - {{ entries.length }} entries</span>
        <span v-else>⏹ STOPPED</span>
        
        <div class="auto-scroll-toggle">
          <label>
            <input type="checkbox" v-model="autoScroll" />
            Auto-scroll
          </label>
        </div>
      </div>
    </div>

    <!-- Log Display -->
    <div class="log-container" ref="logContainer">
      <div v-if="entries.length === 0 && !streaming" class="empty-state">
        <p>👆 Select a log source and click "Start Stream" to view logs</p>
      </div>

      <div v-else-if="entries.length === 0 && streaming" class="empty-state">
        <div class="loading-spinner"></div>
        <p>Connecting to log stream...</p>
      </div>

      <div v-else class="log-entries">
        <div
          v-for="(entry, idx) in entries"
          :key="idx"
          class="log-entry"
          :class="`level-${entry.level.toLowerCase()}`"
        >
          <span class="timestamp">{{ formatTimestamp(entry.timestamp) }}</span>
          <span class="level">{{ entry.level }}</span>
          <span class="source">{{ entry.source }}</span>
          <span class="message">{{ entry.message }}</span>
        </div>
      </div>

      <!-- Error Display -->
      <div v-if="errorMessage" class="error-banner">
        <strong>Error:</strong> {{ errorMessage }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'

interface LogSource {
  id: string
  name: string
  path: string
  description: string
  available: boolean
}

interface LogEntry {
  timestamp: string
  level: string
  source: string
  message: string
  raw: string
}

const sources = ref<LogSource[]>([])
const selectedSource = ref('')
const level = ref('ALL')
const keyword = ref('')
const tailLines = ref(100)
const autoScroll = ref(true)
const streaming = ref(false)
const entries = ref<LogEntry[]>([])
const errorMessage = ref('')
const logContainer = ref<HTMLElement | null>(null)

let eventSource: EventSource | null = null

// Fetch available log sources
const fetchSources = async () => {
  try {
    const response = await fetch('/api/admin/logs/sources', {
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
    })
    if (response.ok) {
      sources.value = await response.json()
    }
  } catch (err) {
    console.error('Failed to fetch log sources:', err)
  }
}

// Format timestamp
const formatTimestamp = (ts: string) => {
  const date = new Date(ts)
  return date.toLocaleTimeString('en-US', { 
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3
  })
}

// Start streaming logs
const startStream = () => {
  if (!selectedSource.value) return

  stopStream() // Ensure any existing stream is stopped
  errorMessage.value = ''
  streaming.value = true

  const params = new URLSearchParams({
    source: selectedSource.value,
    level: level.value,
    tail: tailLines.value.toString(),
    follow: 'true',
  })

  if (keyword.value) {
    params.append('keyword', keyword.value)
  }

  const token = localStorage.getItem('token')
  const url = `/api/admin/logs/stream?${params.toString()}`

  // SSE doesn't support custom headers, so we add token as query param
  // Note: This is a limitation of EventSource. Consider using WebSocket for production
  eventSource = new EventSource(url)

  eventSource.addEventListener('log', (e) => {
    try {
      const entry = JSON.parse(e.data) as LogEntry
      entries.value.push(entry)
      
      // Limit entries to prevent memory issues
      if (entries.value.length > 1000) {
        entries.value.shift()
      }

      if (autoScroll.value) {
        scrollToBottom()
      }
    } catch (err) {
      console.error('Failed to parse log entry:', err)
    }
  })

  eventSource.addEventListener('error', (e: any) => {
    console.error('SSE error:', e)
    errorMessage.value = e.data || 'Stream connection error'
    streaming.value = false
  })

  eventSource.addEventListener('end', () => {
    streaming.value = false
  })

  eventSource.onerror = () => {
    errorMessage.value = 'Connection to log stream lost'
    streaming.value = false
    stopStream()
  }
}

// Stop streaming
const stopStream = () => {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  streaming.value = false
}

// Clear log entries
const clearLogs = () => {
  entries.value = []
  errorMessage.value = ''
}

// Export logs
const exportLogs = async () => {
  if (!selectedSource.value) return

  const params = new URLSearchParams({
    source: selectedSource.value,
    level: level.value,
  })

  if (keyword.value) {
    params.append('keyword', keyword.value)
  }

  try {
    const response = await fetch(`/api/admin/logs/export?${params.toString()}`, {
      headers: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
    })

    if (response.ok) {
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${selectedSource.value}-${new Date().toISOString()}.log`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
    } else {
      errorMessage.value = 'Failed to export logs'
    }
  } catch (err) {
    console.error('Export error:', err)
    errorMessage.value = 'Export failed'
  }
}

// Handle filter changes
const handleFilterChange = () => {
  if (streaming.value) {
    // Restart stream with new filters
    startStream()
  }
}

// Scroll to bottom
const scrollToBottom = async () => {
  await nextTick()
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

// Watch auto-scroll
watch(autoScroll, (newVal) => {
  if (newVal) {
    scrollToBottom()
  }
})

onMounted(() => {
  fetchSources()
})

onUnmounted(() => {
  stopStream()
})
</script>

<style scoped>
.log-viewer {
  padding: 24px;
  max-width: 1600px;
  margin: 0 auto;
}

.header {
  margin-bottom: 24px;
}

.header h2 {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: #1a1a1a;
}

.header p {
  margin: 0;
  color: #666;
}

.controls-panel {
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 16px;
}

.control-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.control-row:last-child {
  margin-bottom: 0;
}

.control-group {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.search-group {
  flex: 2;
}

.control-group label {
  font-size: 12px;
  font-weight: 600;
  color: #666;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.control-group input,
.control-group select {
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
}

.control-group input:focus,
.control-group select:focus {
  outline: none;
  border-color: #2563eb;
}

.button-group {
  display: flex;
  gap: 8px;
}

button {
  padding: 10px 16px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: #2563eb;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #1d4ed8;
}

.btn-secondary {
  background: #f3f4f6;
  color: #374151;
}

.btn-secondary:hover:not(:disabled) {
  background: #e5e7eb;
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  margin-top: 16px;
}

.status-bar.streaming {
  background: #dcfce7;
  color: #166534;
}

.status-bar.paused {
  background: #fef3c7;
  color: #92400e;
}

.auto-scroll-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

.auto-scroll-toggle label {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
}

.log-container {
  background: #1a1a1a;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  height: 600px;
  overflow-y: auto;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  position: relative;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #999;
}

.empty-state p {
  margin: 8px 0 0 0;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #333;
  border-top-color: #666;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.log-entries {
  padding: 12px;
}

.log-entry {
  display: grid;
  grid-template-columns: 110px 80px 120px 1fr;
  gap: 12px;
  padding: 6px 8px;
  border-bottom: 1px solid #2a2a2a;
  transition: background 0.15s;
}

.log-entry:hover {
  background: #252525;
}

.log-entry .timestamp {
  color: #888;
}

.log-entry .level {
  font-weight: 600;
  text-transform: uppercase;
  font-size: 11px;
}

.log-entry .source {
  color: #00d4ff;
  font-size: 12px;
}

.log-entry .message {
  color: #e0e0e0;
  word-break: break-word;
}

.log-entry.level-debug .level {
  color: #a78bfa;
}

.log-entry.level-info .level {
  color: #60a5fa;
}

.log-entry.level-warning .level {
  color: #fbbf24;
}

.log-entry.level-error .level {
  color: #f87171;
}

.error-banner {
  position: sticky;
  bottom: 0;
  background: #fee2e2;
  border-top: 2px solid #ef4444;
  padding: 12px 16px;
  color: #991b1b;
}

/* Scrollbar styling */
.log-container::-webkit-scrollbar {
  width: 12px;
}

.log-container::-webkit-scrollbar-track {
  background: #2a2a2a;
}

.log-container::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 6px;
}

.log-container::-webkit-scrollbar-thumb:hover {
  background: #666;
}
</style>
