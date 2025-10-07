<template>
  <div class="network-map">
    <Card title="Network Map">
      <div class="node-selector" v-if="configuredNodes.length > 1">
        <label>Select Node:</label>
        <select v-model="selectedNode" @change="onNodeChange">
          <option v-for="node in configuredNodes" :key="node" :value="node">
            {{ getNodeLabel(node) }}
          </option>
        </select>
      </div>
      
      <div class="map-container">
        <iframe
          v-if="selectedNode"
          :src="mapUrl"
          frameborder="0"
          class="map-iframe"
          title="AllStarLink Network Map"
        ></iframe>
        <div v-else class="no-data">
          No nodes configured
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useNodeStore } from '../stores/node'
import Card from '../components/Card.vue'

const nodeStore = useNodeStore()
const selectedNode = ref(null)

// Get list of configured nodes from status
const configuredNodes = computed(() => {
  if (!nodeStore.status) return []
  
  // Check for multiple nodes array format (future enhancement)
  if (nodeStore.status.nodes && Array.isArray(nodeStore.status.nodes)) {
    return nodeStore.status.nodes.map(n => n.node_id).filter(id => id > 0)
  }
  
  // Fall back to single node_id
  if (nodeStore.status.node_id && nodeStore.status.node_id > 0) {
    return [nodeStore.status.node_id]
  }
  
  return []
})

// Get node label (callsign if available, otherwise just node number)
function getNodeLabel(nodeId) {
  const link = nodeStore.links.find(l => l.node === nodeId)
  if (link && link.node_callsign) {
    return `${link.node_callsign} (${nodeId})`
  }
  return `Node ${nodeId}`
}

// Compute the map URL for the selected node
const mapUrl = computed(() => {
  if (!selectedNode.value) return ''
  return `https://stats.allstarlink.org/stats/${selectedNode.value}/networkMap`
})

// Initialize with first configured node
onMounted(() => {
  if (configuredNodes.value.length > 0) {
    selectedNode.value = configuredNodes.value[0]
  }
})

function onNodeChange() {
  // iframe will automatically reload with new URL
}
</script>

<style scoped>
.network-map {
  padding: 1rem;
  max-width: 1600px;
  margin: 0 auto;
}

.node-selector {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
  padding: 1rem;
  background: var(--bg-tertiary);
  border-radius: 6px;
}

.node-selector label {
  font-weight: 600;
  color: var(--text-primary);
}

.node-selector select {
  padding: 0.5rem 1rem;
  background: var(--bg-input);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 0.875rem;
  cursor: pointer;
  transition: border-color 0.2s;
}

.node-selector select:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.node-selector select:hover {
  border-color: var(--border-hover);
}

.map-container {
  width: 100%;
  height: calc(100vh - 300px);
  min-height: 600px;
  border-radius: 6px;
  overflow: hidden;
  background: var(--bg-secondary);
}

.map-iframe {
  width: 100%;
  height: 100%;
  border: none;
}

.no-data {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-muted);
  font-style: italic;
  font-size: 1rem;
}
</style>
