<template>
  <div class="node-lookup">
    <Card title="Node / Callsign Lookup">
      <div class="search-container">
        <input
          v-model="searchQuery"
          @keyup.enter="performSearch"
          type="text"
          placeholder="Enter node number or callsign (min 3 characters)..."
          class="search-input"
        />
        <button @click="performSearch" :disabled="loading || searchQuery.trim().length < 3" class="btn-primary">
          {{ loading ? 'Searching...' : 'Search' }}
        </button>
      </div>

      <div v-if="error" class="error-message">
        {{ error }}
      </div>

      <div v-if="results.length" class="results-container">
        <h4>Results ({{ results.length }})</h4>
        <div class="results-table">
          <table>
            <thead>
              <tr>
                <th>Node</th>
                <th>Callsign</th>
                <th>Description</th>
                <th>Location</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(r, idx) in results" :key="idx">
                <td class="node-num">
                  <a v-if="r.node > 0" :href="`https://stats.allstarlink.org/stats/${r.node}`" target="_blank" rel="noopener noreferrer" class="node-link">
                    {{ r.node }}
                  </a>
                  <span v-else>{{ r.node }}</span>
                </td>
                <td class="callsign">
                  <a v-if="r.callsign" :href="`https://www.qrz.com/db/${r.callsign.toUpperCase()}`" target="_blank" rel="noopener noreferrer" class="callsign-link">
                    {{ r.callsign }}
                  </a>
                  <span v-else>—</span>
                </td>
                <td>{{ r.description }}</td>
                <td>{{ r.location }}</td>
                <td>
                  <span class="status-badge" :class="r.status">{{ r.status || 'Unknown' }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else-if="searched && !loading" class="no-results">
        No results found for "{{ lastSearch }}"
      </div>

      <div class="info-box">
        <h4>Search Tips:</h4>
        <ul>
          <li><strong>Auto-search:</strong> Results appear automatically as you type (min 3 characters)</li>
          <li>Enter an AllStar node number (e.g., 12345)</li>
          <li>Enter a callsign (e.g., W1ABC)</li>
          <li>IRLP nodes: 80000-89999</li>
          <li>EchoLink nodes: 3000000+</li>
        </ul>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useAuthStore } from '../stores/auth'
import Card from '../components/Card.vue'

const authStore = useAuthStore()

const searchQuery = ref('')
const results = ref([])
const loading = ref(false)
const error = ref('')
const searched = ref(false)
const lastSearch = ref('')
let searchTimeout = null

// Watch for changes in search query and auto-search after 3+ characters
watch(searchQuery, (newValue) => {
  // Clear existing timeout
  if (searchTimeout) {
    clearTimeout(searchTimeout)
    searchTimeout = null
  }

  // If query is less than 3 characters, clear results
  if (newValue.trim().length < 3) {
    if (newValue.trim().length === 0) {
      results.value = []
      error.value = ''
      searched.value = false
    }
    return
  }

  // Debounce: wait 300ms after user stops typing
  searchTimeout = setTimeout(() => {
    performSearch()
  }, 300)
})

async function performSearch() {
  if (!searchQuery.value.trim() || searchQuery.value.trim().length < 3) return

  loading.value = true
  error.value = ''
  results.value = []
  lastSearch.value = searchQuery.value

  try {
    const headers = authStore.getAuthHeaders()
    const response = await fetch(`/api/node-lookup?q=${encodeURIComponent(searchQuery.value)}`, { headers })
    const data = await response.json()

    if (!response.ok || !data.ok) {
      error.value = data.error?.message || 'Search failed'
      return
    }

    results.value = data.data.results || []
    searched.value = true
  } catch (e) {
    error.value = 'Network error occurred'
    console.error('Search error:', e)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.node-lookup {
  padding: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

.search-container {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.search-input {
  flex: 1;
  background: #222;
  border: 1px solid #444;
  color: #eee;
  padding: 0.75rem 1rem;
  border-radius: 6px;
  font-size: 1rem;
  transition: border-color 0.2s;
}

.search-input:focus {
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

.results-container {
  margin-top: 2rem;
}

.results-container h4 {
  margin-bottom: 1rem;
  color: #eee;
}

.results-table {
  overflow-x: auto;
}

.results-table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.results-table th,
.results-table td {
  padding: 0.75rem;
  text-align: left;
  border-bottom: 1px solid #2a2a2a;
}

.results-table thead th {
  background: #252525;
  color: #999;
  text-transform: uppercase;
  font-size: 0.75rem;
  font-weight: 600;
}

.results-table tbody tr:hover {
  background: #222;
}

.node-num {
  font-weight: 600;
  color: #60a5fa;
}

.node-link {
  color: #60a5fa;
  text-decoration: none;
  border-bottom: 1px dotted #60a5fa;
  transition: all 0.2s ease;
}

.node-link:hover {
  color: #93c5fd;
  border-bottom-color: #93c5fd;
  border-bottom-style: solid;
}

.callsign {
  font-weight: 600;
  color: #4ade80;
}

.callsign-link {
  color: #4ade80;
  text-decoration: none;
  border-bottom: 1px dotted #4ade80;
  transition: all 0.2s ease;
}

.callsign-link:hover {
  color: #86efac;
  border-bottom-color: #86efac;
  border-bottom-style: solid;
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

.status-badge.online {
  background: #16a34a;
  color: #fff;
}

.status-badge.offline {
  background: #dc2626;
  color: #fff;
}

.no-results {
  text-align: center;
  padding: 3rem;
  color: #666;
  font-style: italic;
}

.info-box {
  margin-top: 2rem;
  padding: 1.5rem;
  background: #1a1a1a;
  border-left: 4px solid #3b82f6;
  border-radius: 6px;
}

.info-box h4 {
  margin-top: 0;
  margin-bottom: 0.75rem;
  color: #60a5fa;
}

.info-box ul {
  margin: 0;
  padding-left: 1.5rem;
  color: #999;
}

.info-box li {
  margin-bottom: 0.5rem;
}
</style>
